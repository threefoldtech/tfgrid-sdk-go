package grid

import (
	"fmt"
	"hash/fnv"
	"net"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/zos"
)

// BuildNetwork builds a network configuration for deployment
func BuildNetwork(name, projectName string, nodeID uint32) (workloads.ZNet, error) {
	key, err := workloads.RandomMyceliumKey()
	if err != nil {
		return workloads.ZNet{}, fmt.Errorf("failed to generate mycelium key for node %d: %w", nodeID, err)
	}

	return workloads.ZNet{
		Name:  name,
		Nodes: []uint32{nodeID},
		IPRange: zos.IPNet{IPNet: net.IPNet{
			IP:   net.IPv4(10, 20, 0, 0),
			Mask: net.CIDRMask(16, 32),
		}},
		MyceliumKeys: map[uint32][]byte{nodeID: key},
		SolutionType: projectName,
		Description:  "Probe network",
	}, nil
}

// BuildVM builds a VM configuration for deployment
func BuildVM(name string, nodeID uint32, networkName string, cpu uint8, memoryMB uint64, diskMB uint64) (workloads.VM, error) {
	ipSeed, err := workloads.RandomMyceliumIPSeed()
	if err != nil {
		return workloads.VM{}, fmt.Errorf("failed to generate mycelium IP seed: %w", err)
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

// BuildNetworkLight builds a light network configuration for zoslight nodes
func BuildNetworkLight(name, projectName string, nodeID uint32) (workloads.ZNetLight, error) {
	key, err := workloads.RandomMyceliumKey()
	if err != nil {
		return workloads.ZNetLight{}, fmt.Errorf("failed to generate mycelium key for node %d: %w", nodeID, err)
	}

	return workloads.ZNetLight{
		Name:  name,
		Nodes: []uint32{nodeID},
		IPRange: zos.IPNet{IPNet: net.IPNet{
			IP:   net.IPv4(10, 20, 0, 0),
			Mask: net.CIDRMask(16, 32),
		}},
		MyceliumKeys: map[uint32][]byte{nodeID: key},
		SolutionType: projectName,
		Description:  "Probe network light",
	}, nil
}

// BuildVMLight builds a light VM configuration for zoslight nodes
func BuildVMLight(name string, nodeID uint32, networkName string, cpu uint8, memoryMB uint64, diskMB uint64) (workloads.VMLight, error) {
	ipSeed, err := workloads.RandomMyceliumIPSeed()
	if err != nil {
		return workloads.VMLight{}, fmt.Errorf("failed to generate mycelium IP seed: %w", err)
	}

	// Generate unique IP based on VM name hash to avoid conflicts
	ip := generateUniqueIPFromName(name)

	return workloads.VMLight{
		Name:           name,
		NodeID:         nodeID,
		Flist:          "https://hub.threefold.me/tf-official-apps/threefoldtech-ubuntu-22.04.flist",
		CPU:            cpu,
		MemoryMB:       memoryMB,
		RootfsSizeMB:   diskMB,
		Entrypoint:     "/sbin/zinit init",
		NetworkName:    networkName,
		IP:             ip,
		MyceliumIPSeed: ipSeed,
	}, nil
}

// generateUniqueIPFromName generates a unique IP address in range 10.20.2.2-10.20.2.254
// based on the hash of the VM name to avoid conflicts with concurrent deployments
func generateUniqueIPFromName(vmName string) string {
	h := fnv.New32a()
	h.Write([]byte(vmName))
	hash := h.Sum32()
	lastOctet := 2 + (hash % 253) // Range: 2-254
	return fmt.Sprintf("10.20.2.%d", lastOctet)
}
