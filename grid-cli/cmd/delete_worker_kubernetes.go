// Package cmd for parsing command line arguments
package cmd

import (
	"slices"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	command "github.com/threefoldtech/tfgrid-sdk-go/grid-cli/internal/cmd"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-cli/internal/config"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
)

// deleteWorkerCmd represents the update kubernetes command
var deleteWorkerCmd = &cobra.Command{
	Use:   "delete",
	Short: "remove worker from a kubernetes cluster",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, err := cmd.Flags().GetString("name")
		if err != nil {
			return err
		}
		workerName, err := cmd.Flags().GetString("worker-name")
		if err != nil {
			return err
		}
		cfg, err := config.GetUserConfig()
		if err != nil {
			log.Fatal().Err(err).Send()
		}

		t, err := deployer.NewTFPluginClient(cfg.Mnemonics, deployer.WithNetwork(cfg.Network), deployer.WithRMBTimeout(100))
		if err != nil {
			log.Fatal().Err(err).Send()
		}

		cluster, err := command.GetK8sCluster(cmd.Context(), t, name)
		if err != nil {
			log.Fatal().Err(err).Send()
		}

		for i, worker := range cluster.Workers {
			if worker.Name == workerName {
				cluster.Workers = slices.Delete(cluster.Workers, i, i+1)
			}
		}

		err = command.DeleteWorkerKubernetesCluster(cmd.Context(), t, cluster)
		if err != nil {
			log.Fatal().Err(err).Send()
		}

		log.Info().Msgf("Done deleting worker: %s", workerName)
		return nil
	},
}

func init() {
	updateKubernetesCmd.AddCommand(deleteWorkerCmd)

	deleteWorkerCmd.Flags().StringP("name", "n", "", "name of the kubernetes cluster")
	err := deleteWorkerCmd.MarkFlagRequired("name")
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	deleteWorkerCmd.Flags().String("worker-name", "", "worker to delete")
	err = deleteWorkerCmd.MarkFlagRequired("worker-name")
	if err != nil {
		log.Fatal().Err(err).Send()
	}
}
