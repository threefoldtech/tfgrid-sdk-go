// Package cmd for handling commands
package cmd

import (
	"context"
	"fmt"
	"net"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// DeployVM deploys a vm with mounts
func DeployVM(ctx context.Context, t deployer.TFPluginClient, vm workloads.VM, mount workloads.Disk, node uint32) (workloads.VM, error) {
	networkName := fmt.Sprintf("%snetwork", vm.Name)
	projectName := fmt.Sprintf("vm/%s", vm.Name)
	network, err := buildNetwork(networkName, projectName, []uint32{node}, len(vm.MyceliumIPSeed) != 0)
	if err != nil {
		return workloads.VM{}, err
	}

	mounts := []workloads.Disk{}
	if mount.SizeGB != 0 {
		mounts = append(mounts, mount)
	}
	vm.NetworkName = networkName
	dl := workloads.NewDeployment(vm.Name, node, projectName, nil, networkName, mounts, nil, []workloads.VM{vm}, nil)

	log.Info().Msg("deploying network")
	err = t.NetworkDeployer.Deploy(ctx, &network)
	if err != nil {
		return workloads.VM{}, errors.Wrapf(err, "failed to deploy network on node %d", node)
	}

	log.Info().Msg("deploying vm")
	err = t.DeploymentDeployer.Deploy(ctx, &dl)
	if err != nil {
		log.Warn().Msg("error happened while deploying. removing network")
		revertErr := t.NetworkDeployer.Cancel(ctx, &network)
		if revertErr != nil {
			log.Error().Err(revertErr).Msg("failed to remove network")
		}
		return workloads.VM{}, errors.Wrapf(err, "failed to deploy vm on node %d", node)
	}
	resVM, err := t.State.LoadVMFromGrid(ctx, node, vm.Name, dl.Name)
	if err != nil {
		return workloads.VM{}, errors.Wrapf(err, "failed to load vm from node %d", node)
	}
	return resVM, nil
}

// DeployKubernetesCluster deploys a kubernetes cluster
func DeployKubernetesCluster(ctx context.Context, t deployer.TFPluginClient, master workloads.K8sNode, workers []workloads.K8sNode, sshKey string) (workloads.K8sCluster, error) {
	networkName := fmt.Sprintf("%snetwork", master.Name)
	projectName := fmt.Sprintf("kubernetes/%s", master.Name)
	networkNodes := []uint32{master.Node}
	for _, worker := range workers {
		if worker.Node != master.Node {
			networkNodes = append(networkNodes, worker.Node)
		}
	}
	network, err := buildNetwork(networkName, projectName, networkNodes, len(master.MyceliumIPSeed) != 0)
	if err != nil {
		return workloads.K8sCluster{}, err
	}

	cluster := workloads.K8sCluster{
		Master:  &master,
		Workers: workers,
		// TODO: should be randomized
		Token:        "securetoken",
		SolutionType: projectName,
		SSHKey:       sshKey,
		NetworkName:  networkName,
	}
	log.Info().Msg("deploying network")
	err = t.NetworkDeployer.Deploy(ctx, &network)
	if err != nil {
		return workloads.K8sCluster{}, errors.Wrapf(err, "failed to deploy network on nodes %v", network.Nodes)
	}

	log.Info().Msg("deploying cluster")
	err = t.K8sDeployer.Deploy(ctx, &cluster)
	if err != nil {
		log.Warn().Msg("error happened while deploying. removing network")
		revertErr := t.NetworkDeployer.Cancel(ctx, &network)
		if revertErr != nil {
			log.Error().Err(revertErr).Msg("failed to remove network")
		}
		return workloads.K8sCluster{}, errors.Wrap(err, "failed to deploy kubernetes cluster")
	}
	nodeIDs := []uint32{master.Node}
	for _, worker := range workers {
		nodeIDs = append(nodeIDs, worker.Node)
	}
	return t.State.LoadK8sFromGrid(
		ctx,
		nodeIDs,
		master.Name,
	)
}

// DeployGatewayName deploys a gateway name
func DeployGatewayName(ctx context.Context, t deployer.TFPluginClient, gateway workloads.GatewayNameProxy) (workloads.GatewayNameProxy, error) {
	log.Info().Msg("deploying gateway name")
	err := t.GatewayNameDeployer.Deploy(ctx, &gateway)
	if err != nil {
		return workloads.GatewayNameProxy{}, errors.Wrapf(err, "failed to deploy gateway on node %d", gateway.NodeID)
	}

	return t.State.LoadGatewayNameFromGrid(ctx, gateway.NodeID, gateway.Name, gateway.Name)
}

// DeployGatewayFQDN deploys a gateway fqdn
func DeployGatewayFQDN(ctx context.Context, t deployer.TFPluginClient, gateway workloads.GatewayFQDNProxy) error {
	log.Info().Msg("deploying gateway fqdn")
	err := t.GatewayFQDNDeployer.Deploy(ctx, &gateway)
	if err != nil {
		return errors.Wrapf(err, "failed to deploy gateway on node %d", gateway.NodeID)
	}
	return nil
}

// DeployZDBs deploys multiple zdbs
func DeployZDBs(ctx context.Context, t deployer.TFPluginClient, projectName string, zdbs []workloads.ZDB, n int, node uint32) ([]workloads.ZDB, error) {
	dl := workloads.NewDeployment(projectName, node, projectName, nil, "", nil, zdbs, nil, nil)
	log.Info().Msgf("deploying zdbs")
	err := t.DeploymentDeployer.Deploy(ctx, &dl)
	if err != nil {
		return []workloads.ZDB{}, errors.Wrapf(err, "failed to deploy zdbs on node %d", node)
	}

	var resZDBs []workloads.ZDB
	for _, zdb := range zdbs {
		resZDB, err := t.State.LoadZdbFromGrid(ctx, node, zdb.Name, dl.Name)
		if err != nil {
			return []workloads.ZDB{}, errors.Wrapf(err, "failed to load zdb '%s' from node %d", zdb.Name, node)
		}

		resZDBs = append(resZDBs, resZDB)
	}

	return resZDBs, nil
}

func buildNetwork(name, projectName string, nodes []uint32, addMycelium bool) (workloads.ZNet, error) {
	keys := make(map[uint32][]byte)
	if addMycelium {
		for _, node := range nodes {
			key, err := workloads.RandomMyceliumKey()
			if err != nil {
				return workloads.ZNet{}, err
			}
			keys[node] = key
		}
	}
	return workloads.ZNet{
		Name:  name,
		Nodes: nodes,
		IPRange: gridtypes.NewIPNet(net.IPNet{
			IP:   net.IPv4(10, 20, 0, 0),
			Mask: net.CIDRMask(16, 32),
		}),
		MyceliumKeys: keys,
		SolutionType: projectName,
	}, nil
}
