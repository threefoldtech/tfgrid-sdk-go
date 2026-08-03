// Package cmd for parsing command line arguments
package cmd

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"slices"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	command "github.com/threefoldtech/zos_sdk_go/grid-cli/internal/cmd"
	"github.com/threefoldtech/zos_sdk_go/grid-cli/internal/config"
	"github.com/threefoldtech/zos_sdk_go/grid-cli/internal/filters"
	"github.com/threefoldtech/zos_sdk_go/grid-client/deployer"
	client "github.com/threefoldtech/zos_sdk_go/grid-client/node"
	"github.com/threefoldtech/zos_sdk_go/grid-client/subi"
	"github.com/threefoldtech/zos_sdk_go/grid-client/workloads"
	"github.com/threefoldtech/zos_sdk_go/grid-client/zos"
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
		noColor, err := cmd.Parent().Flags().GetBool("no-color")
		if err != nil {
			return err
		}
		disableSentry, err := cmd.Parent().Flags().GetBool("disable-sentry")
		if err != nil {
			return err
		}

		myceliumKeyHex, err := cmd.Flags().GetString("mycelium-key")
		if err != nil {
			return err
		}
		myceliumKeyEnv, err := cmd.Flags().GetString("mycelium-key-env")
		if err != nil {
			return err
		}
		myceliumSeedHex, err := cmd.Flags().GetString("mycelium-seed")
		if err != nil {
			return err
		}

		// The key may be named rather than written out. Resolve it before anything
		// else touches it, and return the error rather than logging it: this is the
		// one path in this command that handles a credential, and an error sink that
		// ships elsewhere is not somewhere a caller can predict.
		myceliumKeyHex, err = resolveMyceliumKeyHex(myceliumKeyHex, myceliumKeyEnv)
		if err != nil {
			return err
		}

		// The key and the seed together determine the mycelium address, so supplying
		// one without the other still yields a different address on redeployment.
		// Refuse that rather than silently returning a machine the caller cannot
		// reach where it expects.
		myceliumKey, seed, err := parseMyceliumIdentity(myceliumKeyHex, myceliumSeedHex, mycelium)
		if err != nil {
			log.Fatal().Err(err).Send()
		}

		if mycelium && len(seed) == 0 {
			seed, err = workloads.RandomMyceliumIPSeed()
			if err != nil {
				log.Fatal().Err(err).Send()
			}
		}

		cfg, err := config.GetUserConfig()
		if err != nil {
			log.Fatal().Err(err).Send()
		}

		opts := []deployer.PluginOpt{
			deployer.WithNetwork(cfg.Network),
		}

		if noColor {
			opts = append(opts, deployer.WithNoColorLogs())
		}

		if disableSentry {
			opts = append(opts, deployer.WithDisableSentry())
		}
		t, err := deployer.NewTFPluginClient(cfg.Mnemonics, opts...)
		if err != nil {
			log.Fatal().Err(err).Send()
		}

		// if no public ips or yggdrasil then we should go for the light deployment
		isLight := !ipv4 && !ipv6 && !ygg

		if node != 0 {
			isLight, err = isZos4Node(cmd.Context(), t.NcPool, t.SubstrateConn, node)
			if err != nil {
				log.Fatal().Err(err).Send()
			}
		}

		if isLight {
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
			err = executeVMLight(cmd.Context(), t, vm, node, farm, disk, volume, myceliumKey)
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
		err = executeVM(cmd.Context(), t, vm, node, farm, disk, volume, myceliumKey)
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

	// A mycelium address is determined by the network's key and the machine's ip
	// seed. Both are generated when not supplied, which is what every existing
	// caller gets. Supplying them lets a deployment that is destroyed and rebuilt
	// later — on this node or another one — come back on the same address.
	//
	// The key is a credential — whoever holds it can answer at the address it
	// defines — so it can also be NAMED instead of written out, with
	// `--mycelium-key-env`, and read from that environment variable. A value on the
	// command line is readable by anything that can list processes, for as long as
	// this one runs. The ip seed carries no such weight and stays a plain flag: it
	// selects an address within the network the key defines, and knowing it grants
	// nothing.
	deployVMCmd.Flags().String("mycelium-key", "", "hex encoded 32 byte mycelium key for the vm's network, generated when omitted")
	deployVMCmd.Flags().String("mycelium-key-env", "", "name of an environment variable holding the hex encoded mycelium key, read instead of passing the key on the command line")
	deployVMCmd.Flags().String("mycelium-seed", "", "hex encoded 6 byte mycelium ip seed for the vm, generated when omitted")
	deployVMCmd.MarkFlagsMutuallyExclusive("mycelium-key", "mycelium-key-env")

	deployVMCmd.Flags().StringToStringP("env", "e", make(map[string]string), "environment variables for the vm")
}

// resolveMyceliumKeyHex returns the mycelium key as hex, from whichever source the
// caller chose: `--mycelium-key` carries the value itself, `--mycelium-key-env`
// carries the NAME of an environment variable holding it. Neither one supplied
// means "generate", exactly as before.
//
// Errors name the variable and never its contents. A value that was rejected is
// still a key, and the reason a caller reaches for this flag at all is to keep that
// value out of places it can be read from.
func resolveMyceliumKeyHex(keyHex, keyEnvName string) (string, error) {
	if keyEnvName == "" {
		return keyHex, nil
	}

	// Cobra rejects both flags together, but the rule is stated here as well so it
	// does not depend on where this is called from.
	if keyHex != "" {
		return "", errors.New("supply the mycelium key either directly or by naming an environment variable, not both")
	}

	value, ok := os.LookupEnv(keyEnvName)
	if !ok {
		return "", fmt.Errorf("environment variable %q was named as the source of the mycelium key, but it is not set", keyEnvName)
	}
	if value == "" {
		return "", fmt.Errorf("environment variable %q was named as the source of the mycelium key, but it is empty", keyEnvName)
	}

	return value, nil
}

// parseMyceliumIdentity decodes and validates a caller-supplied mycelium key and ip
// seed. Empty values mean "generate", which is the behaviour every caller had before
// these flags existed. It returns the decoded key and seed, either of which may be
// nil when not supplied.
func parseMyceliumIdentity(keyHex, seedHex string, mycelium bool) (key, seed []byte, err error) {
	if keyHex == "" && seedHex == "" {
		return nil, nil, nil
	}

	if !mycelium {
		return nil, nil, errors.New("a mycelium key and ip seed were supplied but mycelium is disabled; drop them or pass --mycelium")
	}

	// Cobra's MarkFlagsRequiredTogether already rejects one without the other, but
	// this function is also the single place the rule is stated, so it does not
	// depend on where it is called from.
	if keyHex == "" || seedHex == "" {
		return nil, nil, errors.New("a mycelium key and ip seed must be supplied together; either alone still changes the address")
	}

	key, err = hex.DecodeString(keyHex)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to decode mycelium key as hex")
	}
	if len(key) != zos.MyceliumKeyLen {
		return nil, nil, fmt.Errorf("invalid mycelium key length %d, must be %d bytes", len(key), zos.MyceliumKeyLen)
	}

	seed, err = hex.DecodeString(seedHex)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to decode mycelium ip seed as hex")
	}
	if len(seed) != zos.MyceliumIPSeedLen {
		return nil, nil, fmt.Errorf("invalid mycelium ip seed length %d, must be %d bytes", len(seed), zos.MyceliumIPSeedLen)
	}

	return key, seed, nil
}

func executeVM(
	ctx context.Context, t deployer.TFPluginClient,
	vm workloads.VM,
	node uint32,
	farm, disk, volume uint64,
	myceliumKey []byte,
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
	resVM, err := command.DeployVM(ctx, t, vm, diskMount, volumeMount, myceliumKey)
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
	myceliumKey []byte,
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
	resVM, err := command.DeployVMLight(ctx, t, vm, diskMount, volumeMount, myceliumKey)
	if err != nil {
		return err
	}

	if len(resVM.MyceliumIP) != 0 {
		log.Info().Msgf("vm mycelium ip: %s", resVM.MyceliumIP)
	}

	return nil
}

func isZos4Node(ctx context.Context, client client.NodeClientGetter, sub subi.SubstrateExt, node uint32) (isLight bool, err error) {
	cli, err := client.GetNodeClient(sub, node)
	if err != nil {
		return
	}
	feat, err := cli.SystemGetNodeFeatures(ctx)
	if err != nil {
		return
	}

	return slices.Contains(feat, zos.NetworkLightType), nil
}
