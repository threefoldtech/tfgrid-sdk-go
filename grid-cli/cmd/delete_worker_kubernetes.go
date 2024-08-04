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
	Short: "remove workder from a kubernetes cluster",
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
		master := *cluster.Master
		workers := cluster.Workers

		for i, worker := range workers {
			if worker.Name == workerName {
				workers = slices.Delete(workers, i, i+1)
			}
		}

		cluster, err = command.DeleteWorkerKubernetesCluster(cmd.Context(), t, cluster)
		if err != nil {
			log.Fatal().Err(err).Send()
		}

		log.Info().Msgf("master wireguard ip: %s", cluster.Master.IP)
		if master.PublicIP {
			log.Info().Msgf("master ipv4: %s", cluster.Master.ComputedIP)
		}
		if master.PublicIP6 {
			log.Info().Msgf("master ipv6: %s", cluster.Master.ComputedIP6)
		}
		if master.Planetary {
			log.Info().Msgf("master planetary ip: %s", cluster.Master.PlanetaryIP)
		}
		if len(master.MyceliumIP) != 0 {
			log.Info().Msgf("master mycelium ip: %s", cluster.Master.MyceliumIP)
		}

		for _, worker := range cluster.Workers {
			log.Info().Msgf("%s wireguard ip: %s", worker.Name, worker.IP)
		}
		if len(cluster.Workers) > 0 && cluster.Workers[0].PublicIP {
			for _, worker := range cluster.Workers {
				log.Info().Msgf("%s ipv4: %s", worker.Name, worker.ComputedIP)
			}
		}
		if len(cluster.Workers) > 0 && cluster.Workers[0].PublicIP6 {
			for _, worker := range cluster.Workers {
				log.Info().Msgf("%s ipv6: %s", worker.Name, worker.ComputedIP6)
			}
		}
		if len(cluster.Workers) > 0 && len(cluster.Workers[0].PlanetaryIP) > 0 {
			for _, worker := range cluster.Workers {
				log.Info().Msgf("%s planetary ip: %s", worker.Name, worker.PlanetaryIP)
			}
		}
		if len(cluster.Workers) > 0 && len(cluster.Workers[0].MyceliumIP) > 0 {
			for _, worker := range cluster.Workers {
				log.Info().Msgf("%s mycelium ip: %s", worker.Name, worker.MyceliumIP)
			}
		}
		return nil
	},
}

func init() {
	updatekubernetesCmd.AddCommand(deleteWorkerCmd)

	deleteWorkerCmd.Flags().StringP("name", "n", "", "name of the kubernetes cluster")
	err := deleteWorkerCmd.MarkFlagRequired("name")
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	deleteWorkerCmd.Flags().String("worker-name", "", "worker to delete")
}
