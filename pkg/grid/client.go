package grid

import (
	"context"
	"fmt"
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

	networkName := fmt.Sprintf("probe-net-%d", time.Now().Unix())
	ipRange, err := zos.ParseIPNet("10.1.0.0/16")
	if err != nil {
		return &DeploymentResult{
			Success:   false,
			ErrorCode: "ip_parse_failed",
		}, fmt.Errorf("failed to parse IP range: %w", err)
	}
	network := workloads.ZNet{
		Name:        networkName,
		Description: "Probe network",
		Nodes:       []uint32{nodeID},
		IPRange:     ipRange,
	}

	deployStart := time.Now()
	err = c.tfPlugin.NetworkDeployer.Deploy(ctx, &network)
	if err != nil {
		return &DeploymentResult{
			Success:          false,
			DeployDurationMs: int(time.Since(deployStart).Milliseconds()),
			TotalDurationMs:  int(time.Since(startTime).Milliseconds()),
			ErrorCode:        "network_deploy_failed",
		}, fmt.Errorf("failed to deploy network: %w", err)
	}

	vm := workloads.VM{
		Name:         fmt.Sprintf("probe-%d", time.Now().Unix()),
		Flist:        "https://hub.grid.tf/tf-official-apps/base:latest.flist",
		CPU:          cpu,
		MemoryMB:     memoryMB,
		RootfsSizeMB: diskMB,
		Entrypoint:   "/sbin/zinit init",
		NetworkName:  networkName,
		IP:           "10.1.0.5",
	}

	dl := workloads.NewDeployment("probe", nodeID, "", nil, networkName, nil, nil, []workloads.VM{vm}, nil, nil, nil)

	vmDeployStart := time.Now()
	err = c.tfPlugin.DeploymentDeployer.Deploy(ctx, &dl)
	if err != nil {
		c.tfPlugin.NetworkDeployer.Cancel(ctx, &network)
		return &DeploymentResult{
			Success:          false,
			DeployDurationMs: int(time.Since(vmDeployStart).Milliseconds()),
			TotalDurationMs:  int(time.Since(startTime).Milliseconds()),
			ErrorCode:        "deploy_failed",
		}, fmt.Errorf("failed to deploy VM: %w", err)
	}

	vmDeployDuration := time.Since(vmDeployStart)

	startStart := time.Now()
	_, err = c.tfPlugin.State.LoadVMFromGrid(ctx, nodeID, vm.Name, dl.Name)
	if err != nil {
		c.tfPlugin.DeploymentDeployer.Cancel(ctx, &dl)
		c.tfPlugin.NetworkDeployer.Cancel(ctx, &network)
		return &DeploymentResult{
			Success:          false,
			DeployDurationMs: int(vmDeployDuration.Milliseconds()),
			StartDurationMs:  int(time.Since(startStart).Milliseconds()),
			TotalDurationMs:  int(time.Since(startTime).Milliseconds()),
			ErrorCode:        "start_failed",
		}, fmt.Errorf("failed to start VM: %w", err)
	}

	startDuration := time.Since(startStart)
	totalDuration := time.Since(startTime)

	log.Info().
		Int("deploy_ms", int(vmDeployDuration.Milliseconds())).
		Int("start_ms", int(startDuration.Milliseconds())).
		Msg("VM deployed successfully")

	c.tfPlugin.DeploymentDeployer.Cancel(ctx, &dl)
	c.tfPlugin.NetworkDeployer.Cancel(ctx, &network)

	return &DeploymentResult{
		Success:          true,
		DeployDurationMs: int(vmDeployDuration.Milliseconds()),
		StartDurationMs:  int(startDuration.Milliseconds()),
		TotalDurationMs:  int(totalDuration.Milliseconds()),
	}, nil
}
