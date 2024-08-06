package internal

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	types "github.com/threefoldtech/tfgrid-sdk-go/grid-compose/pkg"
)

func assignEnvs(vm *workloads.VM, envs []string) {
	env := make(map[string]string, 0)
	for _, envVar := range envs {
		keyValuePair := strings.Split(envVar, "=")
		env[keyValuePair[0]] = keyValuePair[1]
	}

	vm.EnvVars = env
}

func assignMounts(vm *workloads.VM, volumns []string, storage map[string]types.Storage) ([]workloads.Disk, error) {
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

func assignNetworksTypes(vm *workloads.VM, networksTypes []string) error {
	for _, networkType := range networksTypes {
		switch networkType {
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

func generateIPNet(ip types.IP, mask types.IPMask) net.IPNet {
	var ipNet net.IPNet

	switch ip.Type {
	case "ip4":
		ipSplit := strings.Split(ip.IP, ".")
		byte1, _ := strconv.ParseUint(ipSplit[0], 10, 8)
		byte2, _ := strconv.ParseUint(ipSplit[1], 10, 8)
		byte3, _ := strconv.ParseUint(ipSplit[2], 10, 8)
		byte4, _ := strconv.ParseUint(ipSplit[3], 10, 8)

		ipNet.IP = net.IPv4(byte(byte1), byte(byte2), byte(byte3), byte(byte4))
	default:
		return ipNet
	}

	var maskIP net.IPMask

	switch mask.Type {
	case "cidr":
		maskSplit := strings.Split(mask.Mask, "/")
		maskOnes, _ := strconv.ParseInt(maskSplit[0], 10, 8)
		maskBits, _ := strconv.ParseInt(maskSplit[1], 10, 8)

		maskIP = net.CIDRMask(int(maskOnes), int(maskBits))
		ipNet.Mask = maskIP
	default:
		return ipNet
	}

	return ipNet
}
