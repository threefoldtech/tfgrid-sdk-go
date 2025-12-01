package grid

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/zos"
	proxy "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/client"
)

type Client struct {
	tfPlugin deployer.TFPluginClient
}

func NewClient(network, mnemonic string, logLevel string) (*Client, error) {
	opts := []deployer.PluginOpt{
		deployer.WithNetwork(network),
		deployer.WithDisableSentry(),
	}

	if logLevel == "debug" {
		opts = append(opts, deployer.WithLogs())
	}

	tfPlugin, err := deployer.NewTFPluginClient(mnemonic, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create grid client: %w", err)
	}

	return &Client{tfPlugin: tfPlugin}, nil
}

func (c *Client) GetProxyClient() proxy.Client {
	return c.tfPlugin.GridProxyClient
}

type DeploymentResult struct {
	Success          bool
	DeployDurationMs int
	StartDurationMs  int
	TotalDurationMs  int
	ErrorCode        string
}

func (c *Client) DeployVM(ctx context.Context, nodeID uint32, cpu uint8, memoryMB uint64, diskMB uint64) (*DeploymentResult, error) {
	startTime := time.Now()
	timestamp := time.Now().Unix()

	vmName := fmt.Sprintf("probe_%d", timestamp)
	networkName := fmt.Sprintf("%s_net", vmName)
	projectName := fmt.Sprintf("%s_project", vmName)

	network, err := buildNetwork(networkName, projectName, []uint32{nodeID})
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

	deployStart := time.Now()
	log.Debug().Str("network", networkName).Uint32("node_id", nodeID).Msg("Deploying network")
	err = c.tfPlugin.NetworkDeployer.Deploy(ctx, &network)
	if err != nil {
		return &DeploymentResult{
			Success:          false,
			DeployDurationMs: int(time.Since(deployStart).Milliseconds()),
			TotalDurationMs:  int(time.Since(startTime).Milliseconds()),
			ErrorCode:        "network_deploy_failed",
		}, fmt.Errorf("failed to deploy network on node %d: %w", nodeID, err)
	}

	vmDeployStart := time.Now()
	log.Debug().Str("vm", vmName).Uint32("node_id", nodeID).Msg("Deploying VM")
	err = c.tfPlugin.DeploymentDeployer.Deploy(ctx, &dl)
	if err != nil {
		revertDeployment(ctx, c.tfPlugin, &dl, &network, false)
		return &DeploymentResult{
			Success:          false,
			DeployDurationMs: int(time.Since(vmDeployStart).Milliseconds()),
			TotalDurationMs:  int(time.Since(startTime).Milliseconds()),
			ErrorCode:        "deploy_failed",
		}, fmt.Errorf("failed to deploy VM on node %d: %w", nodeID, err)
	}

	vmDeployDuration := time.Since(vmDeployStart)

	startStart := time.Now()
	_, err = c.tfPlugin.State.LoadVMFromGrid(ctx, nodeID, vm.Name, dl.Name)
	if err != nil {
		revertDeployment(ctx, c.tfPlugin, &dl, &network, true)
		return &DeploymentResult{
			Success:          false,
			DeployDurationMs: int(vmDeployDuration.Milliseconds()),
			StartDurationMs:  int(time.Since(startStart).Milliseconds()),
			TotalDurationMs:  int(time.Since(startTime).Milliseconds()),
			ErrorCode:        "start_failed",
		}, fmt.Errorf("failed to load VM from node %d: %w", nodeID, err)
	}

	startDuration := time.Since(startStart)
	totalDuration := time.Since(startTime)

	log.Debug().
		Int("deploy_ms", int(vmDeployDuration.Milliseconds())).
		Int("start_ms", int(startDuration.Milliseconds())).
		Uint32("node_id", nodeID).
		Msg("VM deployed successfully")

	revertDeployment(ctx, c.tfPlugin, &dl, &network, true)

	return &DeploymentResult{
		Success:          true,
		DeployDurationMs: int(vmDeployDuration.Milliseconds()),
		StartDurationMs:  int(startDuration.Milliseconds()),
		TotalDurationMs:  int(totalDuration.Milliseconds()),
	}, nil
}

func buildNetwork(name, projectName string, nodes []uint32) (workloads.ZNet, error) {
	keys := make(map[uint32][]byte)
	for _, node := range nodes {
		key, err := workloads.RandomMyceliumKey()
		if err != nil {
			return workloads.ZNet{}, fmt.Errorf("failed to generate mycelium key for node %d: %w", node, err)
		}
		keys[node] = key
	}

	return workloads.ZNet{
		Name:  name,
		Nodes: nodes,
		IPRange: zos.IPNet{IPNet: net.IPNet{
			IP:   net.IPv4(10, 20, 0, 0),
			Mask: net.CIDRMask(16, 32),
		}},
		MyceliumKeys: keys,
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
