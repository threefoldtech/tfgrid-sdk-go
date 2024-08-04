package internal

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/pkg"
)

func assignEnvs(vm *workloads.VM, envs []string) {
	env := make(map[string]string, 0)
	for _, envVar := range envs {
		keyValuePair := strings.Split(envVar, "=")
		env[keyValuePair[0]] = keyValuePair[1]
	}

	vm.EnvVars = env
}

func assignMounts(vm *workloads.VM, volumns []string, storage map[string]pkg.Storage) ([]workloads.Disk, error) {
	var disks []workloads.Disk
	mounts := make([]workloads.Mount, 0)
	for _, volume := range volumns {
		pair := strings.Split(volume, ":")

		storage := storage[pair[0]]
		size, err := strconv.Atoi(strings.TrimSuffix(storage.Size, "GB"))

		if err != nil {
			return nil, err
		}

		disk := workloads.Disk{
			Name:   pair[0],
			SizeGB: size,
		}

		disks = append(disks, disk)

		mounts = append(mounts, workloads.Mount{
			DiskName:   disk.Name,
			MountPoint: pair[1],
		})
	}
	vm.Mounts = mounts

	return disks, nil
}

func assignNetworks(vm *workloads.VM, networks []string, networksConfig map[string]pkg.Network, network *workloads.ZNet) error {
	for _, net := range networks {
		switch networksConfig[net].Type {
		case "wg":
			network.AddWGAccess = true
		case "ip4":
			vm.PublicIP = true
		case "ip6":
			vm.PublicIP6 = true
		case "ygg":
			vm.Planetary = true
		case "myc":
			seed, err := getRandomMyceliumIPSeed()
			if err != nil {
				return fmt.Errorf("failed to get mycelium seed %w", err)
			}
			vm.MyceliumIPSeed = seed
		}
	}

	return nil
}
