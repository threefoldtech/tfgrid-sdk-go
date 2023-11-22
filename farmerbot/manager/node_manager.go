// Package manager provides how to manage nodes, farms and power
package manager

import (
	"fmt"
	"sort"
	"time"

	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/constants"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/models"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/slice"
)

// NodeManager manages nodes
type NodeManager struct {
	db       models.RedisManager
	identity substrate.Identity
	subConn  Sub
}

// NewNodeManager creates a new NodeManager
func NewNodeManager(identity substrate.Identity, subConn Sub, db models.RedisManager) NodeManager {
	return NodeManager{db, identity, subConn}
}

// Define defines a node
func (n *NodeManager) Define(node models.Node) error {
	log.Debug().Msgf("[NODE MANAGER] define node %d", node.ID)

	cpuProv, err := n.db.GetCPUOverProvision()
	if err != nil {
		return err
	}

	if node.Resources.OverProvisionCPU == 0 {
		node.Resources.OverProvisionCPU = float32(cpuProv)
	}

	if node.Resources.OverProvisionCPU < 1 || node.Resources.OverProvisionCPU > 4 {
		return fmt.Errorf("cpu over provision should be a value between 1 and 4 not %v", node.Resources.OverProvisionCPU)
	}

	return n.db.UpdatesNode(node)
}

// FindNode finds an available node in the farm
func (n *NodeManager) FindNode(nodeOptions models.NodeOptions) (uint32, error) {
	log.Info().Msg("[NODE MANAGER] Finding a node")

	if (len(nodeOptions.GPUVendors) > 0 || len(nodeOptions.GPUDevices) > 0) && nodeOptions.HasGPUs == 0 {
		// at least one gpu in case the user didn't provide the amount
		nodeOptions.HasGPUs = 1
	}

	log.Debug().Msgf("[NODE MANAGER] Requirements:\n%+v", nodeOptions)

	nodes, err := n.db.GetNodes()
	if err != nil {
		return 0, err
	}

	farm, err := n.db.GetFarm()
	if err != nil {
		return 0, err
	}

	if nodeOptions.PublicIPs > 0 {
		var publicIpsUsedByNodes uint64
		for _, node := range nodes {
			publicIpsUsedByNodes += node.PublicIPsUsed
		}

		if publicIpsUsedByNodes+nodeOptions.PublicIPs > farm.PublicIPs {
			return 0, fmt.Errorf("not enough public ips available for farm %d", farm.ID)
		}
	}

	var possibleNodes []models.Node
	for _, node := range nodes {
		gpus := node.GPUs
		if nodeOptions.HasGPUs > 1 {
			if len(nodeOptions.GPUVendors) > 0 {
				gpus = filterGPUs(gpus, nodeOptions.GPUVendors, false)
			}

			if len(nodeOptions.GPUDevices) > 0 {
				gpus = filterGPUs(gpus, nodeOptions.GPUDevices, true)
			}

			if len(gpus) < int(nodeOptions.HasGPUs) {
				continue
			}
		}

		if nodeOptions.Certified && !node.Certified {
			continue
		}

		if nodeOptions.PublicConfig && !node.PublicConfig {
			continue
		}

		if node.HasActiveRentContract {
			continue
		}

		if nodeOptions.Dedicated && (!node.Dedicated || !node.IsUnused()) {
			continue
		}

		if nodeOptions.Dedicated {
			if !node.Dedicated || !node.IsUnused() {
				continue
			}
		} else {
			if node.Dedicated && nodeOptions.Capacity != node.Resources.Total {
				continue
			}
		}

		if slice.Contains(nodeOptions.NodeExclude, node.ID) {
			continue
		}

		if !node.CanClaimResources(nodeOptions.Capacity) {
			continue
		}

		possibleNodes = append(possibleNodes, node)
	}

	if len(possibleNodes) == 0 {
		return 0, fmt.Errorf("could not find a suitable node with the given options: %+v", possibleNodes)
	}

	// Sort the nodes on power state (the ones that are ON first then waking up, off, shutting down)
	sort.Slice(possibleNodes, func(i, j int) bool {
		return possibleNodes[i].PowerState < possibleNodes[j].PowerState
	})

	nodeFound := possibleNodes[0]
	log.Debug().Msgf("[NODE MANAGER] Found a node: %d", nodeFound.ID)

	// claim the resources until next update of the data
	// add a timeout (after 30 minutes we update the resources)
	nodeFound.TimeoutClaimedResources = time.Now().Add(constants.TimeoutPowerStateChange)
	if nodeOptions.Dedicated || nodeOptions.HasGPUs > 0 {
		// claim all capacity
		nodeFound.ClaimResources(nodeFound.Resources.Total)
	} else {
		nodeFound.ClaimResources(nodeOptions.Capacity)
	}

	// claim public ips until next update of the data
	if nodeOptions.PublicIPs > 0 {
		nodeFound.PublicIPsUsed += nodeOptions.PublicIPs
	}

	// power on the node if it is down or if it is shutting down
	if nodeFound.PowerState == models.OFF || nodeFound.PowerState == models.ShuttingDown {
		powerManager := NewPowerManager(n.identity, n.subConn, n.db)
		if err := powerManager.PowerOn(nodeFound.ID); err != nil {
			return 0, fmt.Errorf("failed to power on found node %d", nodeFound.ID)
		}
	}

	return nodeFound.ID, nil
}

// FilterIncludesSubStr filters a string slice according to if elements include a sub string
func filterGPUs(gpus []models.GPU, filters []string, device bool) (filtered []models.GPU) {
	for _, gpu := range gpus {
		for _, filter := range filters {
			if gpu.Device == filter && device {
				filtered = append(filtered, gpu)
			}

			if gpu.Vendor == filter && !device {
				filtered = append(filtered, gpu)
			}
		}
	}
	return
}
