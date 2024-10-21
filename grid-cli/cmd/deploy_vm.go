// Package cmd for parsing command line arguments
package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	command "github.com/threefoldtech/tfgrid-sdk-go/grid-cli/internal/cmd"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-cli/internal/config"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-cli/internal/filters"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/zos"
)

var (
	ubuntuFlist           = "https://hub.grid.tf/tf-official-apps/threefoldtech-ubuntu-22.04.flist"
	ubuntuFlistEntrypoint = "/sbin/zinit init"
)

func convertGPUsToZosGPUs(gpus []string) (zosGPUs []zos.GPU) {
	for _, g := range gpus {
		zosGPUs = append(zosGPUs, zos.GPU(g))
	}
	return
}

// deployVMCmd represents the deploy vm command
var deployVMCmd = &cobra.Command{
	Use:   "vm",
	Short: "Deploy a vm",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, err := cmd.Flags().GetString("name")
		if err != nil {
			return err
		}
		env, err := cmd.Flags().GetStringToString("env")
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
		env["SSH_KEY"] = string(sshKey)
		node, err := cmd.Flags().GetUint32("node")
		if err != nil {
			return err
		}
		farm, err := cmd.Flags().GetUint64("farm")
		if err != nil {
			return err
		}
		cpu, err := cmd.Flags().GetUint8("cpu")
		if err != nil {
			return err
		}
		memory, err := cmd.Flags().GetUint64("memory")
		if err != nil {
			return err
		}
		rootfs, err := cmd.Flags().GetUint64("rootfs")
		if err != nil {
			return err
		}
		disk, err := cmd.Flags().GetUint64("disk")
		if err != nil {
			return err
		}
		volume, err := cmd.Flags().GetUint64("volume")
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
		gpus, err := cmd.Flags().GetStringSlice("gpus")
		if err != nil {
			return err
		}
		if len(gpus) > 0 && node == 0 {
			log.Fatal().Msg("must specify node ID when using GPUs")
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
		var seed []byte
		if mycelium {
			seed, err = workloads.RandomMyceliumIPSeed()
			if err != nil {
				log.Fatal().Err(err).Send()
			}
		}

		cfg, err := config.GetUserConfig()
		if err != nil {
			log.Fatal().Err(err).Send()
		}

		t, err := deployer.NewTFPluginClient(cfg.Mnemonics, deployer.WithNetwork(cfg.Network))
		if err != nil {
			log.Fatal().Err(err).Send()
		}

		// if no public ips or yggdrasil then we should go for the light deployment
		if !ipv4 && !ipv6 && !ygg {
			vm := workloads.VMLight{
				Name:           name,
				EnvVars:        env,
				CPU:            cpu,
				MemoryMB:       memory * 1024,
				GPUs:           convertGPUsToZosGPUs(gpus),
				RootfsSizeMB:   rootfs * 1024,
				Flist:          flist,
				Entrypoint:     entrypoint,
				MyceliumIPSeed: seed,
			}
			err = executeVMLight(cmd.Context(), t, vm, node, farm, disk, volume)
			if err == nil {
				return nil
			}

			if !errors.Is(err, deployer.ErrNoNodesMatchesResources) {
				log.Fatal().Err(err).Send()
			}
		}

		vm := workloads.VM{
			Name:           name,
			EnvVars:        env,
			CPU:            cpu,
			MemoryMB:       memory * 1024,
			GPUs:           convertGPUsToZosGPUs(gpus),
			RootfsSizeMB:   rootfs * 1024,
			Flist:          flist,
			Entrypoint:     entrypoint,
			PublicIP:       ipv4,
			PublicIP6:      ipv6,
			MyceliumIPSeed: seed,
			Planetary:      ygg,
		}
		err = executeVM(cmd.Context(), t, vm, node, farm, disk, volume)
		if err != nil {
			log.Fatal().Err(err).Send()
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

	deployVMCmd.Flags().Uint8("cpu", 1, "number of cpu units")
	deployVMCmd.Flags().Uint64("memory", 1, "memory size in gb")
	deployVMCmd.Flags().Uint64("rootfs", 2, "root filesystem size in gb")
	deployVMCmd.Flags().Uint64("disk", 0, "disk size in gb mounted on /data")
	deployVMCmd.Flags().String("flist", ubuntuFlist, "flist for vm")
	deployVMCmd.Flags().StringSlice("gpus", []string{}, "gpus for vm")
	deployVMCmd.Flags().Uint64("volume", 0, "volume size in gb mounted on /volume")

	deployVMCmd.Flags().String("entrypoint", ubuntuFlistEntrypoint, "entrypoint for vm")
	// to ensure entrypoint is provided for custom flist
	deployVMCmd.MarkFlagsRequiredTogether("flist", "entrypoint")

	deployVMCmd.Flags().Bool("ipv4", false, "assign public ipv4 for vm")
	deployVMCmd.Flags().Bool("ipv6", false, "assign public ipv6 for vm")
	deployVMCmd.Flags().Bool("ygg", false, "assign yggdrasil ip for vm")
	deployVMCmd.Flags().Bool("mycelium", true, "assign mycelium ip for vm")
	deployVMCmd.Flags().StringToStringP("env", "e", make(map[string]string), "environment variables for the vm")
}

func executeVM(
	ctx context.Context, t deployer.TFPluginClient,
	vm workloads.VM,
	node uint32,
	farm, disk, volume uint64,
) error {
	var diskMount workloads.Disk
	if disk != 0 {
		diskName := fmt.Sprintf("%sdisk", vm.Name)
		diskMount = workloads.Disk{Name: diskName, SizeGB: disk}
		vm.Mounts = []workloads.Mount{{Name: diskName, MountPoint: "/data"}}
	}

	var volumeMount workloads.Volume
	if volume != 0 {
		volumeName := fmt.Sprintf("%svolume", vm.Name)
		volumeMount = workloads.Volume{Name: volumeName, SizeGB: volume}
		vm.Mounts = append(vm.Mounts, workloads.Mount{Name: volumeName, MountPoint: "/volume"})
	}

	if node == 0 {
		filter, ssd, rootfss := filters.BuildVMFilter(diskMount, volumeMount, farm, vm.MemoryMB, vm.RootfsSizeMB, vm.PublicIP, false)
		nodes, err := deployer.FilterNodes(
			ctx,
			t,
			filter,
			ssd,
			nil,
			rootfss,
		)
		if err != nil {
			return err
		}

		node = uint32(nodes[0].NodeID)
	}

	vm.NodeID = node
	resVM, err := command.DeployVM(ctx, t, vm, diskMount, volumeMount)
	if err != nil {
		return err
	}

	if vm.PublicIP {
		log.Info().Msgf("vm ipv4: %s", resVM.ComputedIP)
	}
	if vm.PublicIP6 {
		log.Info().Msgf("vm ipv6: %s", resVM.ComputedIP6)
	}
	if vm.Planetary {
		log.Info().Msgf("vm planetary ip: %s", resVM.PlanetaryIP)
	}
	if len(resVM.MyceliumIP) != 0 {
		log.Info().Msgf("vm mycelium ip: %s", resVM.MyceliumIP)
	}

	return nil
}

func executeVMLight(
	ctx context.Context, t deployer.TFPluginClient,
	vm workloads.VMLight,
	node uint32,
	farm, disk, volume uint64,
) error {
	var diskMount workloads.Disk
	if disk != 0 {
		diskName := fmt.Sprintf("%sdisk", vm.Name)
		diskMount = workloads.Disk{Name: diskName, SizeGB: disk}
		vm.Mounts = []workloads.Mount{{Name: diskName, MountPoint: "/data"}}
	}

	var volumeMount workloads.Volume
	if volume != 0 {
		volumeName := fmt.Sprintf("%svolume", vm.Name)
		volumeMount = workloads.Volume{Name: volumeName, SizeGB: volume}
		vm.Mounts = append(vm.Mounts, workloads.Mount{Name: volumeName, MountPoint: "/volume"})
	}

	if node == 0 {
		filter, ssd, rootfss := filters.BuildVMFilter(diskMount, volumeMount, farm, vm.MemoryMB, vm.RootfsSizeMB, false, true)
		nodes, err := deployer.FilterNodes(
			ctx,
			t,
			filter,
			ssd,
			nil,
			rootfss,
		)
		if err != nil {
			return err
		}

		node = uint32(nodes[0].NodeID)
	}

	vm.NodeID = node
	resVM, err := command.DeployVMLight(ctx, t, vm, diskMount, volumeMount)
	if err != nil {
		return err
	}

	if len(resVM.MyceliumIP) != 0 {
		log.Info().Msgf("vm mycelium ip: %s", resVM.MyceliumIP)
	}

	return nil
}
