// Package cmd for handling commands
package cmd

import (
	"context"
	"fmt"
	"net"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid3-go/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid3-go/workloads"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// DeployVM deploys a vm with mounts
func DeployVM(t deployer.TFPluginClient, vm workloads.VM, mount workloads.Disk, node uint32) (workloads.VM, error) {
	networkName := fmt.Sprintf("%snetwork", vm.Name)
	network := buildNetwork(networkName, vm.Name, []uint32{node})

	mounts := []workloads.Disk{}
	if mount.SizeGB != 0 {
		mounts = append(mounts, mount)
	}
	vm.NetworkName = networkName
	dl := workloads.NewDeployment(vm.Name, node, vm.Name, nil, networkName, mounts, nil, []workloads.VM{vm}, nil)

	log.Info().Msg("deploying network")
	err := t.NetworkDeployer.Deploy(context.Background(), &network)
	if err != nil {
		return workloads.VM{}, errors.Wrapf(err, "failed to deploy network on node %d", node)
	}
	log.Info().Msg("deploying vm")
	err = t.DeploymentDeployer.Deploy(context.Background(), &dl)
	if err != nil {
		return workloads.VM{}, errors.Wrapf(err, "failed to deploy vm on node %d", node)
	}
	resVM, err := t.State.LoadVMFromGrid(node, vm.Name, dl.Name)
	if err != nil {
		return workloads.VM{}, errors.Wrapf(err, "failed to load vm from node %d", node)
	}
	return resVM, nil
}

// DeployKubernetesCluster deploys a kubernetes cluster
func DeployKubernetesCluster(t deployer.TFPluginClient, master workloads.K8sNode, workers []workloads.K8sNode, sshKey string) (workloads.K8sCluster, error) {

	networkName := fmt.Sprintf("%snetwork", master.Name)
	networkNodes := []uint32{master.Node}
	if len(workers) > 0 && workers[0].Node != master.Node {
		networkNodes = append(networkNodes, workers[0].Node)
	}
	network := buildNetwork(networkName, master.Name, networkNodes)

	cluster := workloads.K8sCluster{
		Master:  &master,
		Workers: workers,
		// TODO: should be randomized
		Token:        "securetoken",
		SolutionType: master.Name,
		SSHKey:       sshKey,
		NetworkName:  networkName,
	}
	log.Info().Msg("deploying network")
	err := t.NetworkDeployer.Deploy(context.Background(), &network)
	if err != nil {
		return workloads.K8sCluster{}, errors.Wrapf(err, "failed to deploy network on nodes %v", network.Nodes)
	}
	log.Info().Msg("deploying cluster")
	err = t.K8sDeployer.Deploy(context.Background(), &cluster)
	if err != nil {
		return workloads.K8sCluster{}, errors.Wrap(err, "failed to deploy kubernetes cluster")
	}
	nodeIDs := []uint32{master.Node}
	for _, worker := range workers {
		nodeIDs = append(nodeIDs, worker.Node)
	}
	return t.State.LoadK8sFromGrid(
		nodeIDs,
		master.Name,
	)
}

// DeployGatewayName deploys a gateway name
func DeployGatewayName(t deployer.TFPluginClient, gateway workloads.GatewayNameProxy) (workloads.GatewayNameProxy, error) {
	log.Info().Msg("deploying gateway name")
	err := t.GatewayNameDeployer.Deploy(context.Background(), &gateway)
	if err != nil {
		return workloads.GatewayNameProxy{}, errors.Wrapf(err, "failed to deploy gateway on node %d", gateway.NodeID)
	}
	return t.State.LoadGatewayNameFromGrid(gateway.NodeID, gateway.Name, gateway.Name)
}

// DeployGatewayFQDN deploys a gateway fqdn
func DeployGatewayFQDN(t deployer.TFPluginClient, gateway workloads.GatewayFQDNProxy) error {

	log.Info().Msg("deploying gateway fqdn")
	err := t.GatewayFQDNDeployer.Deploy(context.Background(), &gateway)
	if err != nil {
		return errors.Wrapf(err, "failed to deploy gateway on node %d", gateway.NodeID)
	}
	return nil
}

func buildNetwork(name, projectName string, nodes []uint32) workloads.ZNet {
	return workloads.ZNet{
		Name:  name,
		Nodes: nodes,
		IPRange: gridtypes.NewIPNet(net.IPNet{
			IP:   net.IPv4(10, 20, 0, 0),
			Mask: net.CIDRMask(16, 32),
		}),
		SolutionType: projectName,
	}
}
