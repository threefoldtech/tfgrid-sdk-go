package grid

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/deployment-checker/pkg/config"
	"github.com/threefoldtech/deployment-checker/pkg/retry"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	proxy "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/client"
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

// CleanupOrphanedContracts cancels all contracts matching the probe project name pattern
func (c *Client) CleanupOrphanedContracts(ctx context.Context) error {
	log.Info().Msg("Starting cleanup of orphaned probe contracts")

	// Get all contracts for this twin
	contracts, err := c.tfPlugin.ContractsGetter.ListContractsByTwinID([]string{"Created", "GracePeriod"})
	if err != nil {
		return fmt.Errorf("failed to list contracts: %w", err)
	}

	// Pattern: probe_*_project (e.g., probe_1234567890_project)
	probeProjectPattern := regexp.MustCompile(`^probe_\d+_project$`)

	var contractIDs []uint64

	// Filter node contracts by project name pattern
	for _, contract := range contracts.NodeContracts {
		deploymentData, err := workloads.ParseDeploymentData(contract.DeploymentData)
		if err != nil {
			log.Warn().
				Err(err).
				Str("contract_id", contract.ContractID).
				Msg("Skipping contract with invalid deployment data")
			continue
		}

		if probeProjectPattern.MatchString(deploymentData.ProjectName) {
			contractID, err := strconv.ParseUint(contract.ContractID, 0, 64)
			if err != nil {
				log.Warn().
					Err(err).
					Str("contract_id", contract.ContractID).
					Msg("Failed to parse contract ID")
				continue
			}
			contractIDs = append(contractIDs, contractID)
		}
	}

	// For name contracts, we can use ListContractsOfProjectName for each matching project
	// But since we're cleaning up all probe contracts, we can iterate through all node contracts
	// and use the project name to get associated name contracts
	probeProjectNames := make(map[string]bool)
	for _, contract := range contracts.NodeContracts {
		deploymentData, err := workloads.ParseDeploymentData(contract.DeploymentData)
		if err != nil {
			continue
		}
		if probeProjectPattern.MatchString(deploymentData.ProjectName) {
			probeProjectNames[deploymentData.ProjectName] = true
		}
	}

	// Get name contracts for each probe project
	for projectName := range probeProjectNames {
		projectContracts, err := c.tfPlugin.ContractsGetter.ListContractsOfProjectName(projectName, false)
		if err != nil {
			log.Warn().
				Err(err).
				Str("project_name", projectName).
				Msg("Failed to get contracts for project")
			continue
		}

		// Add name contract IDs
		for _, contract := range projectContracts.NameContracts {
			contractID, err := strconv.ParseUint(contract.ContractID, 0, 64)
			if err != nil {
				log.Warn().
					Err(err).
					Str("contract_id", contract.ContractID).
					Msg("Failed to parse name contract ID")
				continue
			}
			// Avoid duplicates
			found := false
			for _, id := range contractIDs {
				if id == contractID {
					found = true
					break
				}
			}
			if !found {
				contractIDs = append(contractIDs, contractID)
			}
		}
	}

	if len(contractIDs) == 0 {
		log.Info().Msg("No orphaned probe contracts found")
		return nil
	}

	log.Info().
		Int("count", len(contractIDs)).
		Msg("Found orphaned probe contracts, canceling")

	// Cancel all contracts in a single call
	if err := c.tfPlugin.BatchCancelContract(contractIDs); err != nil {
		return fmt.Errorf("failed to cancel contracts: %w", err)
	}

	log.Info().
		Int("count", len(contractIDs)).
		Msg("Successfully canceled orphaned probe contracts")

	return nil
}

type DeploymentResult struct {
	Success   bool
	ErrorCode string
	// TotalDurationMs is the total end-to-end time from deployment start to completion (in milliseconds)
	// This includes network build, network deployment, VM deployment, and VM verification
	TotalDurationMs int
}

func (c *Client) DeployVM(ctx context.Context, nodeID uint32, cpu uint8, memoryMB uint64, diskMB uint64) (*DeploymentResult, error) {
	startTime := time.Now()

	vmName := fmt.Sprintf("probe_%d", startTime.Unix())
	networkName := fmt.Sprintf("%s_net", vmName)
	projectName := fmt.Sprintf("%s_project", vmName)

	var network workloads.ZNet
	var networkDeployed bool
	var deploymentDeployed bool

	// Defer cleanup function to ensure network is cleaned up on early failure
	defer func() {
		if networkDeployed && !deploymentDeployed {
			// Network was deployed but deployment failed, clean it up
			log.Debug().Msg("Cleaning up network after deployment failure")
			if err := c.tfPlugin.NetworkDeployer.Cancel(ctx, &network); err != nil {
				log.Error().Err(err).Msg("Failed to cleanup network in defer")
			}
		}
	}()

	network, err := BuildNetwork(networkName, projectName, nodeID)
	if err != nil {
		return &DeploymentResult{
			Success:   false,
			ErrorCode: "network_build_failed",
		}, fmt.Errorf("failed to build network: %w", err)
	}

	vm, err := BuildVM(vmName, nodeID, networkName, cpu, memoryMB, diskMB)
	if err != nil {
		return &DeploymentResult{
			Success:   false,
			ErrorCode: "vm_build_failed",
		}, fmt.Errorf("failed to build VM: %w", err)
	}

	dl := workloads.NewDeployment(vmName, nodeID, projectName, nil, networkName, nil, nil, []workloads.VM{vm}, nil, nil, nil)

	log.Debug().Str("network", networkName).Uint32("node_id", nodeID).Msg("Deploying network")
	err = retry.DoWithBackoff(ctx, c.retryCfg, func() error {
		return c.tfPlugin.NetworkDeployer.Deploy(ctx, &network)
	})
	if err != nil {
		return &DeploymentResult{
			Success:         false,
			TotalDurationMs: int(time.Since(startTime).Milliseconds()),
			ErrorCode:       "network_deploy_failed",
		}, fmt.Errorf("failed to deploy network on node %d: %w", nodeID, err)
	}
	networkDeployed = true // Mark network as deployed

	log.Debug().Str("vm", vmName).Uint32("node_id", nodeID).Msg("Deploying VM")
	err = retry.DoWithBackoff(ctx, c.retryCfg, func() error {
		return c.tfPlugin.DeploymentDeployer.Deploy(ctx, &dl)
	})
	if err != nil {
		deploymentDeployed = false
		RevertDeployment(ctx, c.tfPlugin, &dl, &network, false)
		return &DeploymentResult{
			Success:         false,
			TotalDurationMs: int(time.Since(startTime).Milliseconds()),
			ErrorCode:       "deploy_failed",
		}, fmt.Errorf("failed to deploy VM on node %d: %w", nodeID, err)
	}
	deploymentDeployed = true // Mark deployment as successful

	err = retry.DoWithBackoff(ctx, c.retryCfg, func() error {
		_, err := c.tfPlugin.State.LoadVMFromGrid(ctx, nodeID, vm.Name, dl.Name)
		return err
	})
	if err != nil {
		deploymentDeployed = false
		RevertDeployment(ctx, c.tfPlugin, &dl, &network, true)
		return &DeploymentResult{
			Success:         false,
			TotalDurationMs: int(time.Since(startTime).Milliseconds()),
			ErrorCode:       "start_failed",
		}, fmt.Errorf("failed to load VM from node %d: %w", nodeID, err)
	}
	deploymentDeployed = true // Mark verification as successful

	totalDuration := time.Since(startTime)

	log.Debug().
		Int("total_ms", int(totalDuration.Milliseconds())).
		Uint32("node_id", nodeID).
		Msg("VM deployed successfully")

	RevertDeployment(ctx, c.tfPlugin, &dl, &network, true)

	return &DeploymentResult{
		Success:         true,
		TotalDurationMs: int(totalDuration.Milliseconds()),
	}, nil
}

// DeployVMLight deploys a light VM on a zoslight node
func (c *Client) DeployVMLight(ctx context.Context, nodeID uint32, cpu uint8, memoryMB uint64, diskMB uint64) (*DeploymentResult, error) {
	startTime := time.Now()

	vmName := fmt.Sprintf("probe_%d", startTime.Unix())
	networkName := fmt.Sprintf("%s_net", vmName)
	projectName := fmt.Sprintf("%s_project", vmName)

	var network workloads.ZNetLight
	var networkDeployed bool
	var deploymentDeployed bool

	// Defer cleanup function to ensure network is cleaned up on early failure
	defer func() {
		if networkDeployed && !deploymentDeployed {
			// Network was deployed but deployment failed, clean it up
			log.Debug().Msg("Cleaning up network light after deployment failure")
			if err := c.tfPlugin.NetworkDeployer.Cancel(ctx, &network); err != nil {
				log.Error().Err(err).Msg("Failed to cleanup network in defer")
			}
		}
	}()

	network, err := BuildNetworkLight(networkName, projectName, nodeID)
	if err != nil {
		return &DeploymentResult{
			Success:   false,
			ErrorCode: "network_build_failed",
		}, fmt.Errorf("failed to build network: %w", err)
	}

	vm, err := BuildVMLight(vmName, nodeID, networkName, cpu, memoryMB, diskMB)
	if err != nil {
		return &DeploymentResult{
			Success:   false,
			ErrorCode: "vm_build_failed",
		}, fmt.Errorf("failed to build VM: %w", err)
	}

	dl := workloads.NewDeployment(vmName, nodeID, projectName, nil, networkName, nil, nil, nil, []workloads.VMLight{vm}, nil, nil)

	log.Debug().Str("network", networkName).Uint32("node_id", nodeID).Msg("Deploying network light")
	err = retry.DoWithBackoff(ctx, c.retryCfg, func() error {
		return c.tfPlugin.NetworkDeployer.Deploy(ctx, &network)
	})
	if err != nil {
		return &DeploymentResult{
			Success:         false,
			TotalDurationMs: int(time.Since(startTime).Milliseconds()),
			ErrorCode:       "network_deploy_failed",
		}, fmt.Errorf("failed to deploy network on node %d: %w", nodeID, err)
	}
	networkDeployed = true // Mark network as deployed

	log.Debug().Str("vm", vmName).Uint32("node_id", nodeID).Msg("Deploying VM light")
	err = retry.DoWithBackoff(ctx, c.retryCfg, func() error {
		return c.tfPlugin.DeploymentDeployer.Deploy(ctx, &dl)
	})
	if err != nil {
		deploymentDeployed = false
		RevertDeploymentLight(ctx, c.tfPlugin, &dl, &network, false)
		return &DeploymentResult{
			Success:         false,
			TotalDurationMs: int(time.Since(startTime).Milliseconds()),
			ErrorCode:       "deploy_failed",
		}, fmt.Errorf("failed to deploy VM on node %d: %w", nodeID, err)
	}
	deploymentDeployed = true // Mark deployment as successful

	err = retry.DoWithBackoff(ctx, c.retryCfg, func() error {
		_, err := c.tfPlugin.State.LoadVMLightFromGrid(ctx, nodeID, vm.Name, dl.Name)
		return err
	})
	if err != nil {
		deploymentDeployed = false
		RevertDeploymentLight(ctx, c.tfPlugin, &dl, &network, true)
		return &DeploymentResult{
			Success:         false,
			TotalDurationMs: int(time.Since(startTime).Milliseconds()),
			ErrorCode:       "start_failed",
		}, fmt.Errorf("failed to load VM from node %d: %w", nodeID, err)
	}
	deploymentDeployed = true // Mark verification as successful

	totalDuration := time.Since(startTime)

	log.Debug().
		Int("total_ms", int(totalDuration.Milliseconds())).
		Uint32("node_id", nodeID).
		Msg("VM light deployed successfully")

	RevertDeploymentLight(ctx, c.tfPlugin, &dl, &network, true)

	return &DeploymentResult{
		Success:         true,
		TotalDurationMs: int(totalDuration.Milliseconds()),
	}, nil
}
