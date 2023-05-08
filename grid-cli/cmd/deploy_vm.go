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

var ubuntuFlist = "https://hub.grid.tf/tf-official-apps/threefoldtech-ubuntu-22.04.flist"
var ubuntuFlistEntrypoint = "/sbin/zinit init"

// deployVMCmd represents the deploy vm command
var deployVMCmd = &cobra.Command{
	Use:   "vm",
	Short: "Deploy a vm",
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
		node, err := cmd.Flags().GetUint32("node")
		if err != nil {
			return err
		}
		farm, err := cmd.Flags().GetUint64("farm")
		if err != nil {
			return err
		}
		cpu, err := cmd.Flags().GetInt("cpu")
		if err != nil {
			return err
		}
		memory, err := cmd.Flags().GetInt("memory")
		if err != nil {
			return err
		}
		rootfs, err := cmd.Flags().GetInt("rootfs")
		if err != nil {
			return err
		}
		disk, err := cmd.Flags().GetInt("disk")
		if err != nil {
			return err
		}
		flist, err := cmd.Flags().GetString("flist")
		if err != nil {
			return err
		}
		entrypoint, err := cmd.Flags().GetString("entrypoint")
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
		vm := workloads.VM{
			Name:       name,
			EnvVars:    map[string]string{"SSH_KEY": string(sshKey)},
			CPU:        cpu,
			Memory:     memory * 1024,
			RootfsSize: rootfs * 1024,
			Flist:      flist,
			Entrypoint: entrypoint,
			PublicIP:   ipv4,
			PublicIP6:  ipv6,
			Planetary:  ygg,
		}
		var mount workloads.Disk
		if disk != 0 {
			diskName := fmt.Sprintf("%sdisk", name)
			mount = workloads.Disk{Name: diskName, SizeGB: disk}
			vm.Mounts = []workloads.Mount{{DiskName: diskName, MountPoint: "/data"}}
		}
		cfg, err := config.GetUserConfig()
		if err != nil {
			log.Fatal().Err(err).Send()
		}
		t, err := deployer.NewTFPluginClient(cfg.Mnemonics, "sr25519", cfg.Network, "", "", "", 100, true)
		if err != nil {
			log.Fatal().Err(err).Send()
		}
		if node == 0 {
			nodes, err := deployer.FilterNodes(
				cmd.Context(),
				t,
				filters.BuildVMFilter(vm, mount, farm),
			)
			if err != nil {
				log.Fatal().Err(err).Send()
			}

			node = uint32(nodes[0].NodeID)
		}
		resVM, err := command.DeployVM(cmd.Context(), t, vm, mount, node)
		if err != nil {
			log.Fatal().Err(err).Send()
		}

		if ipv4 {
			log.Info().Msgf("vm ipv4: %s", resVM.ComputedIP)
		}
		if ipv6 {
			log.Info().Msgf("vm ipv6: %s", resVM.ComputedIP6)
		}
		if ygg {
			log.Info().Msgf("vm yggdrasil ip: %s", resVM.YggIP)
		}
		return nil
	},
}

func init() {
	deployCmd.AddCommand(deployVMCmd)

	deployVMCmd.Flags().StringP("name", "n", "", "name of the virtual machine")
	err := deployVMCmd.MarkFlagRequired("name")
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	deployVMCmd.Flags().String("ssh", "", "path to public ssh key")
	// should it be required?
	err = deployVMCmd.MarkFlagRequired("ssh")
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	deployVMCmd.Flags().Uint32("node", 0, "node id vm should be deployed on")
	deployVMCmd.Flags().Uint64("farm", 1, "farm id vm should be deployed on")
	deployVMCmd.MarkFlagsMutuallyExclusive("node", "farm")

	deployVMCmd.Flags().Int("cpu", 1, "number of cpu units")
	deployVMCmd.Flags().Int("memory", 1, "memory size in gb")
	deployVMCmd.Flags().Int("rootfs", 2, "root filesystem size in gb")
	deployVMCmd.Flags().Int("disk", 0, "disk size in gb mounted on /data")
	deployVMCmd.Flags().String("flist", ubuntuFlist, "flist for vm")
	deployVMCmd.Flags().String("entrypoint", ubuntuFlistEntrypoint, "entrypoint for vm")
	// to ensure entrypoint is provided for custom flist
	deployVMCmd.MarkFlagsRequiredTogether("flist", "entrypoint")

	deployVMCmd.Flags().Bool("ipv4", false, "assign public ipv4 for vm")
	deployVMCmd.Flags().Bool("ipv6", false, "assign public ipv6 for vm")
	deployVMCmd.Flags().Bool("ygg", true, "assign yggdrasil ip for vm")
}
