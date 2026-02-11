package grid

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/deployment-checker/pkg/config"
	"github.com/threefoldtech/deployment-checker/pkg/retry"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/zos"
	proxy "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/client"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

type Client struct {
	tfPlugin deployer.TFPluginClient
	retryCfg retry.BackoffConfig
}

func NewClient(cfg *config.Config) (*Client, error) {
	opts := []deployer.PluginOpt{
		deployer.WithNetwork(cfg.Grid.Network),
		deployer.WithDisableSentry(),
	}

	if cfg.LogLevel == "debug" {
		opts = append(opts, deployer.WithLogs())
	}

	tfPlugin, err := deployer.NewTFPluginClient(cfg.Grid.Mnemonic, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create grid client: %w", err)
	}

	retryCfg := retry.BackoffConfig{
		MaxRetries:     cfg.RetryConfig().MaxRetries,
		InitialBackoff: cfg.InitialBackoff(),
		MaxBackoff:     cfg.MaxBackoff(),
		Multiplier:     cfg.RetryConfig().Multiplier,
	}

	return &Client{
		tfPlugin: tfPlugin,
		retryCfg: retryCfg,
	}, nil
}

func (c *Client) Close() {
	c.tfPlugin.Close()
}

func (c *Client) GetProxyClient() proxy.Client {
	return c.tfPlugin.GridProxyClient
}

func (c *Client) CleanupOrphanedContracts(ctx context.Context) error {
	log.Info().Msg("Starting cleanup of orphaned probe contracts")

	contracts, err := c.tfPlugin.ContractsGetter.ListContractsByTwinID([]string{"Created", "GracePeriod"})
	if err != nil {
		return fmt.Errorf("failed to list contracts: %w", err)
	}

	var contractIDs []uint64
	for _, contract := range contracts.NodeContracts {
		contractID, err := strconv.ParseUint(contract.ContractID, 10, 64)
		if err != nil {
			return fmt.Errorf("failed to parse contract ID: %w", err)
		}
		contractIDs = append(contractIDs, contractID)
	}

	if len(contractIDs) == 0 {
		log.Info().Msg("No orphaned probe contracts found")
		return nil
	}

	log.Info().
		Int("count", len(contractIDs)).
		Msg("Found orphaned probe contracts, canceling")

	if err := c.tfPlugin.BatchCancelContract(contractIDs); err != nil {
		return fmt.Errorf("failed to cancel contracts: %w", err)
	}

	log.Info().
		Int("count", len(contractIDs)).
		Msg("Successfully canceled orphaned probe contracts")

	return nil
}

type DeploymentResult struct {
	Success         bool
	ErrorCode       string
	TotalDurationMs int
}

func (c *Client) MakeDeployment(ctx context.Context, node types.Node, cpu uint8, memoryMB uint64, diskMB uint64) (*DeploymentResult, error) {
	var (
		isLight   = isZoslightNode(node)
		nodeID    = uint32(int(node.NodeID))
		startTime = time.Now()

		vmName      = fmt.Sprintf("probe_%d", startTime.Unix())
		networkName = fmt.Sprintf("%s_net", vmName)
		projectName = fmt.Sprintf("%s_project", vmName)

		networkDeployed    bool
		deploymentDeployed bool
		network            workloads.Network
		dl                 workloads.Deployment
		vmNameForLoad      string
		err                error
	)

	if isLight {
		networkLight, buildErr := BuildNetworkLight(networkName, projectName, nodeID)
		if buildErr != nil {
			return &DeploymentResult{
				Success:   false,
				ErrorCode: "network_build_failed",
			}, fmt.Errorf("failed to build network: %w", buildErr)
		}
		network = &networkLight

		vm, buildErr := BuildVMLight(vmName, nodeID, networkName, cpu, memoryMB, diskMB)
		if buildErr != nil {
			return &DeploymentResult{
				Success:   false,
				ErrorCode: "vm_build_failed",
			}, fmt.Errorf("failed to build VM: %w", buildErr)
		}
		vmNameForLoad = vm.Name
		dl = workloads.NewDeployment(vmName, nodeID, projectName, nil, networkName, nil, nil, nil, []workloads.VMLight{vm}, nil, nil)
	} else {
		networkNormal, buildErr := BuildNetwork(networkName, projectName, nodeID)
		if buildErr != nil {
			return &DeploymentResult{
				Success:   false,
				ErrorCode: "network_build_failed",
			}, fmt.Errorf("failed to build network: %w", buildErr)
		}
		network = &networkNormal

		vm, buildErr := BuildVM(vmName, nodeID, networkName, cpu, memoryMB, diskMB)
		if buildErr != nil {
			return &DeploymentResult{
				Success:   false,
				ErrorCode: "vm_build_failed",
			}, fmt.Errorf("failed to build VM: %w", buildErr)
		}
		vmNameForLoad = vm.Name
		dl = workloads.NewDeployment(vmName, nodeID, projectName, nil, networkName, nil, nil, []workloads.VM{vm}, nil, nil, nil)
	}

	defer func() {
		if networkDeployed && !deploymentDeployed {
			log.Debug().Msgf("Cleaning up network after deployment failure")
			if err := c.tfPlugin.NetworkDeployer.Cancel(ctx, network); err != nil {
				log.Error().Err(err).Msg("Failed to cleanup network in defer")
			}
		}
	}()

	log.Debug().Str("network", networkName).Uint32("node_id", nodeID).Msgf("Deploying network")
	err = retry.DoWithBackoff(ctx, c.retryCfg, func() error {
		return c.tfPlugin.NetworkDeployer.Deploy(ctx, network)
	})
	if err != nil {
		return &DeploymentResult{
			Success:         false,
			TotalDurationMs: int(time.Since(startTime).Milliseconds()),
			ErrorCode:       "network_deploy_failed",
		}, fmt.Errorf("failed to deploy network on node %d: %w", nodeID, err)
	}
	networkDeployed = true

	log.Debug().Str("vm", vmName).Uint32("node_id", nodeID).Msgf("Deploying VM")
	err = retry.DoWithBackoff(ctx, c.retryCfg, func() error {
		return c.tfPlugin.DeploymentDeployer.Deploy(ctx, &dl)
	})
	if err != nil {
		deploymentDeployed = false
		RevertDeployment(ctx, c.tfPlugin, &dl, network, false)
		return &DeploymentResult{
			Success:         false,
			TotalDurationMs: int(time.Since(startTime).Milliseconds()),
			ErrorCode:       "deploy_failed",
		}, fmt.Errorf("failed to deploy VM on node %d: %w", nodeID, err)
	}
	deploymentDeployed = true

	err = retry.DoWithBackoff(ctx, c.retryCfg, func() error {
		var loadErr error
		if isLight {
			_, loadErr = c.tfPlugin.State.LoadVMLightFromGrid(ctx, nodeID, vmNameForLoad, dl.Name)
		} else {
			_, loadErr = c.tfPlugin.State.LoadVMFromGrid(ctx, nodeID, vmNameForLoad, dl.Name)
		}
		return loadErr
	})
	if err != nil {
		deploymentDeployed = false
		RevertDeployment(ctx, c.tfPlugin, &dl, network, true)
		return &DeploymentResult{
			Success:         false,
			TotalDurationMs: int(time.Since(startTime).Milliseconds()),
			ErrorCode:       "start_failed",
		}, fmt.Errorf("failed to load VM from node %d: %w", nodeID, err)
	}
	deploymentDeployed = true

	totalDuration := time.Since(startTime)

	log.Debug().
		Int("total_ms", int(totalDuration.Milliseconds())).
		Uint32("node_id", nodeID).
		Bool("light", isLight).
		Msg("VM deployed successfully")

	RevertDeployment(ctx, c.tfPlugin, &dl, network, true)

	return &DeploymentResult{
		Success:         true,
		TotalDurationMs: int(totalDuration.Milliseconds()),
	}, nil
}

func isZoslightNode(node types.Node) bool {
	hasNetworkLight := slices.Contains(node.Features, zos.NetworkLightType)
	hasZMachineLight := slices.Contains(node.Features, zos.ZMachineLightType)

	return hasNetworkLight && hasZMachineLight
}

func RevertDeployment(ctx context.Context, tfPlugin deployer.TFPluginClient, dl *workloads.Deployment, network workloads.Network, deleteVM bool) {
	if deleteVM {
		log.Debug().Msg("Cleaning up deployment")
		if err := tfPlugin.DeploymentDeployer.Cancel(ctx, dl); err != nil {
			log.Error().Err(err).Msg("Failed to cancel deployment")
		}
	}
	if err := tfPlugin.NetworkDeployer.Cancel(ctx, network); err != nil {
		log.Error().Err(err).Msg("Failed to cancel network")
	}
}
