// Package cmd for parsing command line arguments
package cmd

import (
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	command "github.com/threefoldtech/tfgrid-sdk-go/grid-cli/internal/cmd"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-cli/internal/config"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-cli/internal/filters"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
)

// updateKubernetesCmd represents the deploy kubernetes command
var addWorkerCmd = &cobra.Command{
	Use:   "add",
	Short: "add workders to a kubernetes cluster",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, err := cmd.Flags().GetString("name")
		if err != nil {
			return err
		}
		sshFile, err := cmd.Flags().GetString("ssh")
		if err != nil {
			return err
		}
		sshKey, err := os.ReadFile(sshFile)
		if err != nil {
			log.Fatal().Err(err).Send()
		}
		workerNumber, err := cmd.Flags().GetInt("workers-number")
		if err != nil {
			return err
		}
		workersNode, err := cmd.Flags().GetUint32("workers-node")
		if err != nil {
			return err
		}
		workersFarm, err := cmd.Flags().GetUint64("workers-farm")
		if err != nil {
			return err
		}
		workersCPU, err := cmd.Flags().GetInt("workers-cpu")
		if err != nil {
			return err
		}
		workersMemory, err := cmd.Flags().GetInt("workers-memory")
		if err != nil {
			return err
		}
		workersDisk, err := cmd.Flags().GetInt("workers-disk")
		if err != nil {
			return err
		}
		workersIPV4, err := cmd.Flags().GetBool("workers-ipv4")
		if err != nil {
			return err
		}
		workersIPV6, err := cmd.Flags().GetBool("workers-ipv6")
		if err != nil {
			return err
		}
		workersYgg, err := cmd.Flags().GetBool("workers-ygg")
		if err != nil {
			return err
		}
		workersMycelium, err := cmd.Flags().GetBool("workers-mycelium")
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

		for i := 0; i < workerNumber; i++ {
			var worker workloads.K8sNode

			if len(workers) == 0 {
				if workersNode == 0 {
					filter, disks, rootfss := filters.BuildK8sFilter(workers[0], workersFarm, uint(workerNumber))
					workersNodes, err := deployer.FilterNodes(cmd.Context(), t, filter, disks, nil, rootfss)
					if err != nil {
						log.Fatal().Err(err).Send()
					}
					workersNode = uint32(workersNodes[0].NodeID)
				}

				var seed []byte
				if workersMycelium {
					seed, err = workloads.RandomMyceliumIPSeed()
					if err != nil {
						log.Fatal().Err(err).Send()
					}
				}
				workerName := fmt.Sprintf("worker%d", i)
				worker = workloads.K8sNode{
					Name:           workerName,
					Flist:          k8sFlist,
					CPU:            workersCPU,
					Memory:         workersMemory * 1024,
					DiskSize:       workersDisk,
					PublicIP:       workersIPV4,
					PublicIP6:      workersIPV6,
					Planetary:      workersYgg,
					Node:           workersNode,
					MyceliumIPSeed: seed,
				}
			} else {
				worker = workers[0]
				worker.Name = fmt.Sprintf("worker%d", len(cluster.Workers)+1)

				var seed []byte
				if len(worker.MyceliumIP) != 0 {
					seed, err = workloads.RandomMyceliumIPSeed()
					if err != nil {
						log.Fatal().Err(err).Send()
					}
				}
				worker.MyceliumIPSeed = seed
			}

			workers = append(workers, worker)
		}

		cluster, err = command.UpdateKubernetesCluster(cmd.Context(), t, master, workers, string(sshKey))
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
		if workersIPV4 {
			for _, worker := range cluster.Workers {
				log.Info().Msgf("%s ipv4: %s", worker.Name, worker.ComputedIP)
			}
		}
		if workersIPV6 {
			for _, worker := range cluster.Workers {
				log.Info().Msgf("%s ipv6: %s", worker.Name, worker.ComputedIP6)
			}
		}
		if workersYgg {
			for _, worker := range cluster.Workers {
				log.Info().Msgf("%s planetary ip: %s", worker.Name, worker.PlanetaryIP)
			}
		}
		if workersMycelium {
			for _, worker := range cluster.Workers {
				log.Info().Msgf("%s mycelium ip: %s", worker.Name, worker.MyceliumIP)
			}
		}
		return nil
	},
}

func init() {
	updateCmd.AddCommand(addWorkerCmd)

	addWorkerCmd.Flags().StringP("name", "n", "", "name of the kubernetes cluster")
	err := addWorkerCmd.MarkFlagRequired("name")
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	addWorkerCmd.Flags().String("ssh", "", "path to public ssh key")
	err = addWorkerCmd.MarkFlagRequired("ssh")
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	addWorkerCmd.Flags().Int("workers-number", 1, "number of workers")
	addWorkerCmd.Flags().Int("workers-cpu", 1, "workers number of cpu units")
	addWorkerCmd.Flags().Int("workers-memory", 1, "workers memory size in gb")
	addWorkerCmd.Flags().Int("workers-disk", 2, "workers disk size in gb")
	addWorkerCmd.Flags().Uint32("workers-node", 0, "node id workers should be deployed on")
	addWorkerCmd.Flags().Uint64("workers-farm", 1, "farm id workers should be deployed on")
	addWorkerCmd.MarkFlagsMutuallyExclusive("workers-node", "workers-farm")
	addWorkerCmd.Flags().Bool("workers-ipv4", false, "assign public ipv4 for workers")
	addWorkerCmd.Flags().Bool("workers-ipv6", false, "assign public ipv6 for workers")
	addWorkerCmd.Flags().Bool("workers-ygg", true, "assign yggdrasil ip for workers")
	addWorkerCmd.Flags().Bool("workers-mycelium", true, "assign mycelium ip for workers")
}
