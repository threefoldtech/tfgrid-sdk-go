package grid

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/deployment-checker/pkg/config"
	"github.com/threefoldtech/deployment-checker/pkg/retry"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/zos"
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

	network, err := buildNetwork(networkName, projectName, nodeID)
	if err != nil {
		return &DeploymentResult{
			Success:   false,
			ErrorCode: "network_build_failed",
		}, fmt.Errorf("failed to build network: %w", err)
	}

	vm, err := buildVM(vmName, nodeID, networkName, cpu, memoryMB, diskMB)
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

	log.Debug().Str("vm", vmName).Uint32("node_id", nodeID).Msg("Deploying VM")
	err = retry.DoWithBackoff(ctx, c.retryCfg, func() error {
		return c.tfPlugin.DeploymentDeployer.Deploy(ctx, &dl)
	})
	if err != nil {
		revertDeployment(ctx, c.tfPlugin, &dl, &network, false)
		return &DeploymentResult{
			Success:         false,
			TotalDurationMs: int(time.Since(startTime).Milliseconds()),
			ErrorCode:       "deploy_failed",
		}, fmt.Errorf("failed to deploy VM on node %d: %w", nodeID, err)
	}

	err = retry.DoWithBackoff(ctx, c.retryCfg, func() error {
		_, err := c.tfPlugin.State.LoadVMFromGrid(ctx, nodeID, vm.Name, dl.Name)
		return err
	})
	if err != nil {
		revertDeployment(ctx, c.tfPlugin, &dl, &network, true)
		return &DeploymentResult{
			Success:         false,
			TotalDurationMs: int(time.Since(startTime).Milliseconds()),
			ErrorCode:       "start_failed",
		}, fmt.Errorf("failed to load VM from node %d: %w", nodeID, err)
	}

	totalDuration := time.Since(startTime)

	log.Debug().
		Int("total_ms", int(totalDuration.Milliseconds())).
		Uint32("node_id", nodeID).
		Msg("VM deployed successfully")

	revertDeployment(ctx, c.tfPlugin, &dl, &network, true)

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

	network, err := buildNetworkLight(networkName, projectName, nodeID)
	if err != nil {
		return &DeploymentResult{
			Success:   false,
			ErrorCode: "network_build_failed",
		}, fmt.Errorf("failed to build network: %w", err)
	}

	vm, err := buildVMLight(vmName, nodeID, networkName, cpu, memoryMB, diskMB)
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

	log.Debug().Str("vm", vmName).Uint32("node_id", nodeID).Msg("Deploying VM light")
	err = retry.DoWithBackoff(ctx, c.retryCfg, func() error {
		return c.tfPlugin.DeploymentDeployer.Deploy(ctx, &dl)
	})
	if err != nil {
		revertDeploymentLight(ctx, c.tfPlugin, &dl, &network, false)
		return &DeploymentResult{
			Success:         false,
			TotalDurationMs: int(time.Since(startTime).Milliseconds()),
			ErrorCode:       "deploy_failed",
		}, fmt.Errorf("failed to deploy VM on node %d: %w", nodeID, err)
	}

	err = retry.DoWithBackoff(ctx, c.retryCfg, func() error {
		_, err := c.tfPlugin.State.LoadVMLightFromGrid(ctx, nodeID, vm.Name, dl.Name)
		return err
	})
	if err != nil {
		revertDeploymentLight(ctx, c.tfPlugin, &dl, &network, true)
		return &DeploymentResult{
			Success:         false,
			TotalDurationMs: int(time.Since(startTime).Milliseconds()),
			ErrorCode:       "start_failed",
		}, fmt.Errorf("failed to load VM from node %d: %w", nodeID, err)
	}

	totalDuration := time.Since(startTime)

	log.Debug().
		Int("total_ms", int(totalDuration.Milliseconds())).
		Uint32("node_id", nodeID).
		Msg("VM light deployed successfully")

	revertDeploymentLight(ctx, c.tfPlugin, &dl, &network, true)

	return &DeploymentResult{
		Success:         true,
		TotalDurationMs: int(totalDuration.Milliseconds()),
	}, nil
}

func buildNetwork(name, projectName string, nodeID uint32) (workloads.ZNet, error) {
	key, err := workloads.RandomMyceliumKey()
	if err != nil {
		return workloads.ZNet{}, fmt.Errorf("failed to generate mycelium key for node %d: %w", nodeID, err)
	}

	return workloads.ZNet{
		Name:  name,
		Nodes: []uint32{nodeID},
		IPRange: zos.IPNet{IPNet: net.IPNet{
			IP:   net.IPv4(10, 20, 0, 0),
			Mask: net.CIDRMask(16, 32),
		}},
		MyceliumKeys: map[uint32][]byte{nodeID: key},
		SolutionType: projectName,
		Description:  "Probe network",
	}, nil
}

func buildVM(name string, nodeID uint32, networkName string, cpu uint8, memoryMB uint64, diskMB uint64) (workloads.VM, error) {
	ipSeed, err := workloads.RandomMyceliumIPSeed()
	if err != nil {
		return workloads.VM{}, fmt.Errorf("failed to generate mycelium IP seed: %w", err)
	}
	return workloads.VM{
		Name:           name,
		NodeID:         nodeID,
		Flist:          "https://hub.threefold.me/tf-official-apps/threefoldtech-ubuntu-22.04.flist",
		CPU:            cpu,
		MemoryMB:       memoryMB,
		RootfsSizeMB:   diskMB,
		Entrypoint:     "/sbin/zinit init",
		NetworkName:    networkName,
		MyceliumIPSeed: ipSeed,
	}, nil
}

func revertDeployment(ctx context.Context, tfPlugin deployer.TFPluginClient, dl *workloads.Deployment, network *workloads.ZNet, deleteVM bool) {
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

func buildNetworkLight(name, projectName string, nodeID uint32) (workloads.ZNetLight, error) {
	key, err := workloads.RandomMyceliumKey()
	if err != nil {
		return workloads.ZNetLight{}, fmt.Errorf("failed to generate mycelium key for node %d: %w", nodeID, err)
	}

	return workloads.ZNetLight{
		Name:  name,
		Nodes: []uint32{nodeID},
		IPRange: zos.IPNet{IPNet: net.IPNet{
			IP:   net.IPv4(10, 20, 0, 0),
			Mask: net.CIDRMask(16, 32),
		}},
		MyceliumKeys: map[uint32][]byte{nodeID: key},
		SolutionType: projectName,
		Description:  "Probe network light",
	}, nil
}

func buildVMLight(name string, nodeID uint32, networkName string, cpu uint8, memoryMB uint64, diskMB uint64) (workloads.VMLight, error) {
	ipSeed, err := workloads.RandomMyceliumIPSeed()
	if err != nil {
		return workloads.VMLight{}, fmt.Errorf("failed to generate mycelium IP seed: %w", err)
	}
	return workloads.VMLight{
		Name:           name,
		NodeID:         nodeID,
		Flist:          "https://hub.threefold.me/tf-official-apps/threefoldtech-ubuntu-22.04.flist",
		CPU:            cpu,
		MemoryMB:       memoryMB,
		RootfsSizeMB:   diskMB,
		Entrypoint:     "/sbin/zinit init",
		NetworkName:    networkName,
		IP:             "10.20.2.5", // Fixed IP for light VMs
		MyceliumIPSeed: ipSeed,
	}, nil
}

func revertDeploymentLight(ctx context.Context, tfPlugin deployer.TFPluginClient, dl *workloads.Deployment, network *workloads.ZNetLight, deleteVM bool) {
	if deleteVM {
		log.Debug().Msg("Cleaning up deployment light")
		if err := tfPlugin.DeploymentDeployer.Cancel(ctx, dl); err != nil {
			log.Error().Err(err).Msg("Failed to cancel deployment")
		}
	}
	if err := tfPlugin.NetworkDeployer.Cancel(ctx, network); err != nil {
		log.Error().Err(err).Msg("Failed to cancel network")
	}
}
