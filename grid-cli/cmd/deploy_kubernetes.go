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

// deployKubernetesCmd represents the deploy kubernetes command
var deployKubernetesCmd = &cobra.Command{
	Use:   "kubernetes",
	Short: "Deploy a kubernetes cluster",
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
		masterNode, err := cmd.Flags().GetUint32("master-node")
		if err != nil {
			return err
		}
		masterFarm, err := cmd.Flags().GetUint64("master-farm")
		if err != nil {
			return err
		}
		masterCPU, err := cmd.Flags().GetUint8("master-cpu")
		if err != nil {
			return err
		}
		masterMemory, err := cmd.Flags().GetUint64("master-memory")
		if err != nil {
			return err
		}
		masterDisk, err := cmd.Flags().GetUint64("master-disk")
		if err != nil {
			return err
		}
		ipv4, err := cmd.Flags().GetBool("ipv4")
		if err != nil {
			return err
		}
		ipv6, err := cmd.Flags().GetBool("ipv6")
		if err != nil {
			return err
		}
		ygg, err := cmd.Flags().GetBool("ygg")
		if err != nil {
			return err
		}
		mycelium, err := cmd.Flags().GetBool("mycelium")
		if err != nil {
			return err
		}
		noColor, err := cmd.Flags().GetBool("no-color")
		if err != nil {
			return err
		}

		var seed []byte
		if mycelium {
			seed, err = workloads.RandomMyceliumIPSeed()
			if err != nil {
				log.Fatal().Err(err).Send()
			}
		}
		master := workloads.K8sNode{
			VM: &workloads.VM{
				Name:           name,
				CPU:            masterCPU,
				MemoryMB:       masterMemory * 1024,
				PublicIP:       ipv4,
				PublicIP6:      ipv6,
				Planetary:      ygg,
				MyceliumIPSeed: seed,
			},
			DiskSizeGB: masterDisk,
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
		var workers []workloads.K8sNode
		for i := 0; i < workersNumber; i++ {
			var seed []byte
			if workersMycelium {
				seed, err = workloads.RandomMyceliumIPSeed()
				if err != nil {
					log.Fatal().Err(err).Send()
				}
			}
			workerName := fmt.Sprintf("worker%d", i)
			worker := workloads.K8sNode{
				VM: &workloads.VM{
					Name:           workerName,
					CPU:            workersCPU,
					MemoryMB:       workersMemory * 1024,
					PublicIP:       workersIPV4,
					PublicIP6:      workersIPV6,
					Planetary:      workersYgg,
					MyceliumIPSeed: seed,
				},
				DiskSizeGB: workersDisk,
			}
			workers = append(workers, worker)
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

		if masterNode == 0 {

			filter, disks, rootfss := filters.BuildK8sNodeFilter(
				master,
				masterFarm,
			)
			nodes, err := deployer.FilterNodes(
				cmd.Context(),
				t,
				filter,
				disks,
				nil,
				rootfss,
			)
			if err != nil {
				log.Fatal().Err(err).Send()
			}

			masterNode = uint32(nodes[0].NodeID)
		}
		master.NodeID = masterNode
		if len(workersNodes) < workersNumber {
			filter, disks, rootfss := filters.BuildK8sNodeFilter(
				workers[0],
				workersFarm,
			)
			nodes, err := deployer.FilterNodes(
				cmd.Context(),
				t,
				filter,
				disks,
				nil,
				rootfss,
				uint64(workersNumber-len(workersNodes)),
			)
			if err != nil {
				log.Fatal().Err(err).Send()
			}
			for _, node := range nodes {
				workersNodes = append(workersNodes, uint(node.NodeID))
			}
		}
		for i := range workers {
			workers[i].NodeID = uint32(workersNodes[i])
		}
		cluster, err := command.DeployKubernetesCluster(cmd.Context(), t, master, workers, string(sshKey), workloads.K8sFlist)
		if err != nil {
			log.Fatal().Err(err).Send()
		}

		log.Info().Msgf("master wireguard ip: %s", cluster.Master.IP)
		if ipv4 {
			log.Info().Msgf("master ipv4: %s", cluster.Master.ComputedIP)
		}
		if ipv6 {
			log.Info().Msgf("master ipv6: %s", cluster.Master.ComputedIP6)
		}
		if ygg {
			log.Info().Msgf("master planetary ip: %s", cluster.Master.PlanetaryIP)
		}
		if mycelium {
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
	deployCmd.AddCommand(deployKubernetesCmd)

	deployKubernetesCmd.Flags().StringP("name", "n", "", "name of the kubernetes cluster")
	err := deployKubernetesCmd.MarkFlagRequired("name")
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	deployKubernetesCmd.Flags().String("ssh", "", "path to public ssh key")
	err = deployKubernetesCmd.MarkFlagRequired("ssh")
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	deployKubernetesCmd.Flags().Uint8("master-cpu", 1, "master number of cpu units")
	deployKubernetesCmd.Flags().Uint64("master-memory", 1, "master memory size in gb")
	deployKubernetesCmd.Flags().Uint64("master-disk", 2, "master disk size in gb")
	deployKubernetesCmd.Flags().Uint32("master-node", 0, "node id master should be deployed on")
	deployKubernetesCmd.Flags().Uint64("master-farm", 1, "farm id master should be deployed on")
	deployKubernetesCmd.MarkFlagsMutuallyExclusive("master-node", "master-farm")

	deployKubernetesCmd.Flags().Int("workers-number", 0, "number of workers")
	deployKubernetesCmd.Flags().Uint8("workers-cpu", 1, "workers number of cpu units")
	deployKubernetesCmd.Flags().Uint64("workers-memory", 1, "workers memory size in gb")
	deployKubernetesCmd.Flags().Uint64("workers-disk", 2, "workers disk size in gb")
	deployKubernetesCmd.Flags().UintSlice("workers-nodes", []uint{}, "node id workers should be deployed on")
	deployKubernetesCmd.Flags().Uint64("workers-farm", 1, "farm id workers should be deployed on")
	deployKubernetesCmd.MarkFlagsMutuallyExclusive("workers-nodes", "workers-farm")
	deployKubernetesCmd.Flags().Bool("workers-ipv4", false, "assign public ipv4 for workers")
	deployKubernetesCmd.Flags().Bool("workers-ipv6", false, "assign public ipv6 for workers")
	deployKubernetesCmd.Flags().Bool("workers-ygg", true, "assign yggdrasil ip for workers")
	deployKubernetesCmd.Flags().Bool("workers-mycelium", true, "assign mycelium ip for workers")

	deployKubernetesCmd.Flags().Bool("ipv4", false, "assign public ipv4 for master")
	deployKubernetesCmd.Flags().Bool("ipv6", false, "assign public ipv6 for master")
	deployKubernetesCmd.Flags().Bool("ygg", true, "assign yggdrasil ip for master")
	deployKubernetesCmd.Flags().Bool("mycelium", true, "assign mycelium ip for master")
	deployKubernetesCmd.Flags().BoolP("no-color", "n", false, "disable output styling")
}
