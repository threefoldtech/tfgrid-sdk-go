package deploy

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/internal/types"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// TODO: needs to be refactored
func BuildNetworks(networkNodeMap map[string]*types.NetworkData, networks map[string]types.Network, defaultNetName string, getProjectName func(string) string) map[string]*workloads.ZNet {
	zNets := make(map[string]*workloads.ZNet, 0)
	if _, ok := networkNodeMap[defaultNetName]; ok {
		zNets[defaultNetName] = &workloads.ZNet{
			Name: defaultNetName,
			IPRange: gridtypes.NewIPNet(net.IPNet{
				IP:   net.IPv4(10, 20, 0, 0),
				Mask: net.CIDRMask(16, 32),
			}),
			AddWGAccess:  false,
			Nodes:        []uint32{networkNodeMap[defaultNetName].NodeID},
			SolutionType: getProjectName(defaultNetName),
		}
	}

	for networkName, network := range networks {
		zNets[networkName] = &workloads.ZNet{
			Name:         network.Name,
			Description:  network.Description,
			IPRange:      gridtypes.NewIPNet(GenerateIPNet(network.IPRange.IP, network.IPRange.Mask)),
			AddWGAccess:  network.AddWGAccess,
			MyceliumKeys: network.MyceliumKeys,
			Nodes:        []uint32{networkNodeMap[networkName].NodeID},
			SolutionType: getProjectName(networkName),
		}
	}

	return zNets
}

func GenerateDefaultNetworkName(services map[string]types.Service) string {
	var defaultNetName string

	for serviceName := range services {
		defaultNetName += serviceName[:2]
	}

	return fmt.Sprintf("net_%s", defaultNetName)
}

func GenerateIPNet(ip types.IP, mask types.IPMask) net.IPNet {
	var ipNet net.IPNet

	switch ip.Type {
	case "ipv4":
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
