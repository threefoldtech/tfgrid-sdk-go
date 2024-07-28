package cmd

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
)

// UpdateKubernetesCluster deploys a kubernetes cluster
func UpdateKubernetesCluster(ctx context.Context, t deployer.TFPluginClient, master workloads.K8sNode, workers []workloads.K8sNode, sshKey string) (workloads.K8sCluster, error) {
	networkName := master.NetworkName
	projectName := fmt.Sprintf("kubernetes/%s", master.Name)
	fmt.Println(workers)

	cluster := workloads.K8sCluster{
		Master:       &master,
		Workers:      workers,
		Token:        "securetoken",
		SolutionType: projectName,
		SSHKey:       sshKey,
		NetworkName:  networkName,
	}

	log.Info().Msg("updating cluster")
	err := t.K8sDeployer.Deploy(ctx, &cluster)
	if err != nil {
		log.Warn().Msg("error happened while updating")
		return workloads.K8sCluster{}, errors.Wrap(err, "failed to update kubernetes cluster")
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
