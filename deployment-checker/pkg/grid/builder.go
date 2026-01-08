package grid

import (
	"fmt"
	"net"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/zos"
)

func buildNetworkConfig(nodeID uint32) ([]byte, zos.IPNet, error) {
	key, err := workloads.RandomMyceliumKey()
	if err != nil {
		return nil, zos.IPNet{}, fmt.Errorf("failed to generate mycelium key for node %d: %w", nodeID, err)
	}

	ipRange := zos.IPNet{IPNet: net.IPNet{
		IP:   net.IPv4(10, 20, 0, 0),
		Mask: net.CIDRMask(16, 32),
	}}

	return key, ipRange, nil
}

func BuildNetwork(name, projectName string, nodeID uint32) (workloads.ZNet, error) {
	key, ipRange, err := buildNetworkConfig(nodeID)
	if err != nil {
		return workloads.ZNet{}, err
	}

	return workloads.ZNet{
		Name:         name,
		Nodes:        []uint32{nodeID},
		IPRange:      ipRange,
		MyceliumKeys: map[uint32][]byte{nodeID: key},
		SolutionType: projectName,
		Description:  "Probe network",
	}, nil
}

func BuildNetworkLight(name, projectName string, nodeID uint32) (workloads.ZNetLight, error) {
	key, ipRange, err := buildNetworkConfig(nodeID)
	if err != nil {
		return workloads.ZNetLight{}, err
	}

	return workloads.ZNetLight{
		Name:         name,
		Nodes:        []uint32{nodeID},
		IPRange:      ipRange,
		MyceliumKeys: map[uint32][]byte{nodeID: key},
		SolutionType: projectName,
		Description:  "Probe network light",
	}, nil
}

func buildVMConfig() ([]byte, error) {
	ipSeed, err := workloads.RandomMyceliumIPSeed()
	if err != nil {
		return nil, fmt.Errorf("failed to generate mycelium IP seed: %w", err)
	}
	return ipSeed, nil
}

func BuildVM(name string, nodeID uint32, networkName string, cpu uint8, memoryMB uint64, diskMB uint64) (workloads.VM, error) {
	ipSeed, err := buildVMConfig()
	if err != nil {
		return workloads.VM{}, err
	}

	return workloads.VM{
		Name:           name,
		NodeID:         nodeID,
		Flist:          "https://hub.threefold.me/tf-official-apps/threefoldtech-ubuntu-22.04.flist",
		CPU:            cpu,
		MemoryMB:       memoryMB,
		RootfsSizeMB:   diskMB,
		Entrypoint:     "/sbin/zinit init",
		NetworkName:    networkName,
		MyceliumIPSeed: ipSeed,
	}, nil
}

func BuildVMLight(name string, nodeID uint32, networkName string, cpu uint8, memoryMB uint64, diskMB uint64) (workloads.VMLight, error) {
	ipSeed, err := buildVMConfig()
	if err != nil {
		return workloads.VMLight{}, err
	}

	return workloads.VMLight{
		Name:           name,
		NodeID:         nodeID,
		Flist:          "https://hub.threefold.me/tf-official-apps/threefoldtech-ubuntu-22.04.flist",
		CPU:            cpu,
		MemoryMB:       memoryMB,
		RootfsSizeMB:   diskMB,
		Entrypoint:     "/sbin/zinit init",
		NetworkName:    networkName,
		MyceliumIPSeed: ipSeed,
	}, nil
}
