package internal

import (
	"fmt"
	"slices"
	"sort"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/pkg"
)

// FindNode finds an available node in the farm
func (f *FarmerBot) findNode(sub Substrate, nodeOptions NodeFilterOption) (uint32, error) {
	log.Info().Msg("Finding a node")

	nodeOptionsCapacity := capacity{
		hru: convertGBToBytes(nodeOptions.HRU),
		sru: convertGBToBytes(nodeOptions.SRU),
		cru: nodeOptions.CRU,
		mru: convertGBToBytes(nodeOptions.MRU),
	}

	gpuVram := convertGBToBytes(nodeOptions.GPUVram)

	if (len(nodeOptions.GPUVendors) > 0 || len(nodeOptions.GPUDevices) > 0 || gpuVram > 0) && nodeOptions.NumGPU == 0 {
		// at least one gpu in case the user didn't provide the amount
		nodeOptions.NumGPU = 1
	}

	log.Debug().Interface("required filter options", nodeOptions)

	if nodeOptions.PublicIPs > 0 {
		var publicIpsUsedByNodes uint64
		for _, node := range f.nodes {
			publicIpsUsedByNodes += node.publicIPsUsed
		}

		if publicIpsUsedByNodes+nodeOptions.PublicIPs > uint64(len(f.farm.PublicIPs)) {
			return 0, fmt.Errorf("not enough public ips available for farm %d", f.farm.ID)
		}
	}

	var possibleNodes []node
	for _, node := range f.nodes {
		gpus := node.gpus
		if nodeOptions.NumGPU > 0 {
			gpus = filterGPUs(gpus, nodeOptions.GPUVendors, nodeOptions.GPUDevices, gpuVram)

			if len(gpus) < int(nodeOptions.NumGPU) {
				continue
			}
		}

		if nodeOptions.Certified && !node.Certification.IsCertified {
			continue
		}

		if nodeOptions.PublicConfig && !node.PublicConfig.HasValue {
			continue
		}

		if node.hasActiveRentContract {
			continue
		}

		if nodeOptions.Dedicated {
			if !node.dedicated || !node.isUnused() {
				continue
			}
		} else {
			if node.dedicated && nodeOptionsCapacity != node.resources.total {
				continue
			}
		}

		if slices.Contains(nodeOptions.NodesExcluded, uint32(node.ID)) {
			continue
		}

		if !node.canClaimResources(nodeOptionsCapacity, f.config.Power.OverProvisionCPU) {
			continue
		}

		possibleNodes = append(possibleNodes, node)
	}

	if len(possibleNodes) == 0 {
		return 0, fmt.Errorf("could not find a suitable node with the given options: %+v", nodeOptions)
	}

	// Sort the nodes on power state (the ones that are ON first then waking up, off, shutting down)
	sort.Slice(possibleNodes, func(i, j int) bool {
		return possibleNodes[i].powerState < possibleNodes[j].powerState
	})

	nodeFound := possibleNodes[0]
	log.Debug().Uint32("nodeID", uint32(nodeFound.ID)).Msg("Found a node")

	// claim the resources until next update of the data
	// add a timeout (after 30 minutes we update the resources)
	nodeFound.timeoutClaimedResources = time.Now().Add(timeoutPowerStateChange)

	if nodeOptions.Dedicated || nodeOptions.NumGPU > 0 {
		// claim all capacity
		nodeFound.claimResources(nodeFound.resources.total)
	} else {
		nodeFound.claimResources(nodeOptionsCapacity)
	}

	// claim public ips until next update of the data
	if nodeOptions.PublicIPs > 0 {
		nodeFound.publicIPsUsed += nodeOptions.PublicIPs
	}

	// power on the node if it is down or if it is shutting down
	if nodeFound.powerState == off || nodeFound.powerState == shuttingDown {
		if err := f.powerOn(sub, uint32(nodeFound.ID)); err != nil {
			return 0, fmt.Errorf("failed to power on found node %d", nodeFound.ID)
		}
	}

	// update claimed resources
	err := f.updateNode(nodeFound)
	if err != nil {
		return 0, fmt.Errorf("failed to power on found node %d", nodeFound.ID)
	}

	return uint32(nodeFound.ID), nil
}

func filterGPUs(gpus []pkg.GPU, vendors []string, devices []string, vram uint64) (filtered []pkg.GPU) {
	for _, gpu := range gpus {
		vendorMatch := len(vendors) == 0
		if len(vendors) > 0 {
			vendorMatch = slices.Contains(vendors, gpu.Vendor)
		}

		deviceMatch := len(devices) == 0
		if len(devices) > 0 {
			deviceMatch = slices.Contains(devices, gpu.Device)
		}

		if vendorMatch && deviceMatch && (vram == 0 || gpu.Vram >= vram) {
			filtered = append(filtered, gpu)
		}
	}

	return
}

func convertGBToBytes(gb uint64) uint64 {
	return gb * 1024 * 1024 * 1024
}
