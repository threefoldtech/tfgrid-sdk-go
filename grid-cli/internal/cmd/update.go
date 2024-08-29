package cmd

import (
	"context"
	"slices"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
)

// AddWorkersKubernetesCluster deploys a kubernetes cluster
func AddWorkersKubernetesCluster(ctx context.Context, t deployer.TFPluginClient, cluster workloads.K8sCluster, addMycelium bool) (workloads.K8sCluster, error) {
	master := *cluster.Master
	workers := cluster.Workers

	log.Info().Msg("updating network")
	network, err := t.State.LoadNetworkFromGrid(ctx, master.NetworkName)
	if err != nil {
		return workloads.K8sCluster{}, err
	}

	for _, worker := range workers {
		if !slices.Contains(network.Nodes, worker.NodeID) {
			network.Nodes = append(network.Nodes, worker.NodeID)
		}
	}

	if addMycelium {
		for _, node := range network.Nodes {
			key, err := workloads.RandomMyceliumKey()
			if err != nil {
				return workloads.K8sCluster{}, err
			}
			network.MyceliumKeys[node] = key
		}
	}

	err = t.NetworkDeployer.Deploy(ctx, &network)
	if err != nil {
		return workloads.K8sCluster{}, errors.Wrapf(err, "failed to update network on nodes %v", network.Nodes)
	}

	log.Info().Msg("updating cluster")
	err = t.K8sDeployer.Deploy(ctx, &cluster)
	if err != nil {
		log.Warn().Msg("error happened while update.")
		return workloads.K8sCluster{}, errors.Wrap(err, "failed to deploy kubernetes cluster")
	}

	return t.State.LoadK8sFromGrid(
		ctx,
		network.Nodes,
		master.Name,
	)
}

func DeleteWorkerKubernetesCluster(ctx context.Context, t deployer.TFPluginClient, cluster workloads.K8sCluster) error {
	usedNodes := []uint32{cluster.Master.NodeID}
	for _, worker := range cluster.Workers {
		usedNodes = append(usedNodes, worker.NodeID)
	}

	log.Info().Msg("updating network")
	network, err := t.State.LoadNetworkFromGrid(ctx, cluster.Master.NetworkName)
	if err != nil {
		return err
	}

	var removedNodes []uint32
	for _, node := range network.Nodes {
		if !slices.Contains(usedNodes, node) {
			removedNodes = append(removedNodes, node)
			delete(network.MyceliumKeys, node)
		}
	}
	network.Nodes = usedNodes

	err = t.NetworkDeployer.Deploy(ctx, &network)
	if err != nil {
		return errors.Wrapf(err, "failed to update network on nodes %v", network.Nodes)
	}

	for _, node := range removedNodes {
		delete(t.State.CurrentNodeDeployments, node)
	}

	log.Info().Msg("updating cluster")
	err = t.K8sDeployer.Deploy(ctx, &cluster)
	if err != nil {
		log.Warn().Msg("error happened while update.")
		return errors.Wrap(err, "failed to update kubernetes cluster")
	}

	return nil
}
