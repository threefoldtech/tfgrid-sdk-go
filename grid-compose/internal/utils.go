package internal

import (
	"fmt"
	mrand "math/rand"
	"net"
	"strconv"
	"strings"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	types "github.com/threefoldtech/tfgrid-sdk-go/grid-compose/pkg"
	"github.com/threefoldtech/zos/pkg/gridtypes"
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
	case "ipv4":
		ipSplit := strings.Split(ip.IP, ".")
		byte1, _ := strconv.Atoi(ipSplit[0])
		byte2, _ := strconv.Atoi(ipSplit[1])
		byte3, _ := strconv.Atoi(ipSplit[2])
		byte4, _ := strconv.Atoi(ipSplit[3])

		ipNet.IP = net.IPv4(byte(byte1), byte(byte2), byte(byte3), byte(byte4))
	default:
		return ipNet
	}

	var maskIP net.IPMask

	switch mask.Type {
	case "cidr":
		maskSplit := strings.Split(mask.Mask, "/")
		maskOnes, _ := strconv.Atoi(maskSplit[0])
		maskBits, _ := strconv.Atoi(maskSplit[1])
		maskIP = net.CIDRMask(maskOnes, maskBits)
	default:
		return ipNet
	}

	ipNet.Mask = maskIP

	return ipNet
}

func generateNetworks(networks map[string]types.Network) map[string]*workloads.ZNet {
	zNets := make(map[string]*workloads.ZNet, 0)

	for key, network := range networks {
		zNet := workloads.ZNet{
			Name:        network.Name,
			Description: network.Description,
			Nodes:       network.Nodes,
			IPRange: gridtypes.NewIPNet(net.IPNet{
				IP:   net.IPv4(10, 20, 0, 0),
				Mask: net.CIDRMask(16, 32),
			}),
			AddWGAccess:  network.AddWGAccess,
			MyceliumKeys: network.MyceliumKeys,
		}

		zNets[key] = &zNet
	}

	return zNets
}

func generateRandString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[mrand.Intn(len(letters))]
	}
	return string(b)
}
