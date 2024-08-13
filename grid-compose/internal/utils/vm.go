package utils

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/internal/types"
)

func AssignEnvs(vm *workloads.VM, envs []string) {
	env := make(map[string]string, 0)
	for _, envVar := range envs {
		keyValuePair := strings.Split(envVar, "=")
		env[keyValuePair[0]] = keyValuePair[1]
	}

	vm.EnvVars = env
}

// TODO: Create a parser to parse the size given to each field in service
func AssignMounts(vm *workloads.VM, serviceVolumes []string, volumes map[string]types.Volume) ([]workloads.Disk, error) {
	var disks []workloads.Disk
	mounts := make([]workloads.Mount, 0)
	for _, volumeName := range serviceVolumes {
		volume := volumes[volumeName]

		size, err := strconv.Atoi(strings.TrimSuffix(volume.Size, "GB"))

		if err != nil {
			return nil, err
		}

		disk := workloads.Disk{
			Name:   volumeName,
			SizeGB: size,
		}

		disks = append(disks, disk)

		mounts = append(mounts, workloads.Mount{
			DiskName:   disk.Name,
			MountPoint: volume.MountPoint,
		})
	}
	vm.Mounts = mounts

	return disks, nil
}

func AssignNetworksTypes(vm *workloads.VM, ipTypes []string) error {
	for _, ipType := range ipTypes {
		switch ipType {
		case "ipv4":
			vm.PublicIP = true
		case "ipv6":
			vm.PublicIP6 = true
		case "ygg":
			vm.Planetary = true
		case "myc":
			seed, err := GetRandomMyceliumIPSeed()
			if err != nil {
				return fmt.Errorf("failed to get mycelium seed %w", err)
			}
			vm.MyceliumIPSeed = seed
		}
	}

	return nil
}

func GetVmAddresses(vm workloads.VM) string {
	var addresses strings.Builder

	if vm.IP != "" {
		addresses.WriteString(fmt.Sprintf("wireguard: %v, ", vm.IP))
	}
	if vm.Planetary {
		addresses.WriteString(fmt.Sprintf("yggdrasil: %v, ", vm.PlanetaryIP))
	}
	if vm.PublicIP {
		addresses.WriteString(fmt.Sprintf("publicIp4: %v, ", vm.ComputedIP))
	}
	if vm.PublicIP6 {
		addresses.WriteString(fmt.Sprintf("publicIp6: %v, ", vm.ComputedIP6))
	}
	if len(vm.MyceliumIPSeed) != 0 {
		addresses.WriteString(fmt.Sprintf("mycelium: %v, ", vm.MyceliumIP))
	}

	return strings.TrimSuffix(addresses.String(), ", ")
}
