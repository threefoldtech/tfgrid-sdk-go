// Package cmd for parsing command line arguments
package cmd

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	command "github.com/threefoldtech/tfgrid-sdk-go/grid-cli/internal/cmd"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-cli/internal/config"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-cli/internal/filters"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
)

// addWorkerCmd represents the deploy kubernetes command
var addWorkerCmd = &cobra.Command{
	Use:   "add",
	Short: "add  workders to a kubernetes cluster",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, err := cmd.Flags().GetString("name")
		if err != nil {
			return err
		}
		workersNumber, err := cmd.Flags().GetInt("workers-number")
		if err != nil {
			return err
		}
		workersNodes, err := cmd.Flags().GetUintSlice("workers-nodes")
		if err != nil {
			return err
		}
		workersFarm, err := cmd.Flags().GetUint64("workers-farm")
		if err != nil {
			return err
		}
		workersCPU, err := cmd.Flags().GetUint8("workers-cpu")
		if err != nil {
			return err
		}
		workersMemory, err := cmd.Flags().GetUint64("workers-memory")
		if err != nil {
			return err
		}
		workersDisk, err := cmd.Flags().GetUint64("workers-disk")
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
		noColor, err := cmd.Flags().GetBool("no-color")
		if err != nil {
			return err
		}

		cfg, err := config.GetUserConfig()
		if err != nil {
			log.Fatal().Err(err).Send()
		}

		opts := []deployer.PluginOpt{
			deployer.WithNetwork(cfg.Network),
			deployer.WithRMBTimeout(100),
		}

		if noColor {
			opts = append(opts, deployer.WithNoColorLogs())
		}

		t, err := deployer.NewTFPluginClient(cfg.Mnemonics, opts...)
		if err != nil {
			log.Fatal().Err(err).Send()
		}

		cluster, err := command.GetK8sCluster(cmd.Context(), t, name)
		if err != nil {
			log.Fatal().Err(err).Send()
		}

		workers := cluster.Workers

		worker := workloads.K8sNode{
			VM: &workloads.VM{
				CPU:       workersCPU,
				MemoryMB:  workersMemory * 1024,
				PublicIP:  workersIPV4,
				PublicIP6: workersIPV6,
				Planetary: workersYgg,
			},
			DiskSizeGB: workersDisk,
		}

		if workersNumber > len(workersNodes) && workersNumber > 0 {
			filter, disks, rootfss := filters.BuildK8sNodeFilter(worker, workersFarm)
			nodes, err := deployer.FilterNodes(
				cmd.Context(),
				t,
				filter,
				disks,
				nil,
				rootfss,
				uint64(workersNumber-len(workersNodes)))
			if err != nil {
				log.Fatal().Err(err).Send()
			}

			for _, node := range nodes {
				workersNodes = append(workersNodes, uint(node.NodeID))
			}
		}

		var addMycelium bool
		for i := 0; i < workersNumber; i++ {
			var seed []byte
			if len(worker.MyceliumIP) != 0 || workersMycelium {
				addMycelium = true
				seed, err = workloads.RandomMyceliumIPSeed()
				if err != nil {
					log.Fatal().Err(err).Send()
				}
			}

			vm := *worker.VM
			worker.VM = &vm
			worker.Name = fmt.Sprintf("worker%d", len(workers))
			worker.NodeID = uint32(workersNodes[i])
			worker.MyceliumIPSeed = seed

			workers = append(workers, worker)
		}
		cluster.Workers = workers

		cluster, err = command.AddWorkersKubernetesCluster(cmd.Context(), t, cluster, addMycelium)
		if err != nil {
			log.Fatal().Err(err).Send()
		}

		log.Info().Msgf("master wireguard ip: %s", cluster.Master.IP)
		if cluster.Master.PublicIP {
			log.Info().Msgf("master ipv4: %s", cluster.Master.ComputedIP)
		}
		if cluster.Master.PublicIP6 {
			log.Info().Msgf("master ipv6: %s", cluster.Master.ComputedIP6)
		}
		if cluster.Master.Planetary {
			log.Info().Msgf("master planetary ip: %s", cluster.Master.PlanetaryIP)
		}

		if len(cluster.Master.MyceliumIP) != 0 {
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
	updateKubernetesCmd.AddCommand(addWorkerCmd)

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

	addWorkerCmd.Flags().Int("workers-number", 1, "number of workers to add")
	addWorkerCmd.Flags().Uint8("workers-cpu", 1, "workers number of cpu units")
	addWorkerCmd.Flags().Uint64("workers-memory", 1, "workers memory size in gb")
	addWorkerCmd.Flags().Uint64("workers-disk", 2, "workers disk size in gb")
	addWorkerCmd.Flags().UintSlice("workers-nodes", []uint{}, "node ids workers should be deployed on")
	addWorkerCmd.Flags().Uint64("workers-farm", 1, "farm id workers should be deployed on")
	addWorkerCmd.MarkFlagsMutuallyExclusive("workers-nodes", "workers-farm")
	addWorkerCmd.Flags().Bool("workers-ipv4", false, "assign public ipv4 for workers")
	addWorkerCmd.Flags().Bool("workers-ipv6", false, "assign public ipv6 for workers")
	addWorkerCmd.Flags().Bool("workers-ygg", true, "assign yggdrasil ip for workers")
	addWorkerCmd.Flags().Bool("workers-mycelium", true, "assign mycelium ip for workers")
	addWorkerCmd.Flags().BoolP("no-color", "n", false, "disable output styling")
}
