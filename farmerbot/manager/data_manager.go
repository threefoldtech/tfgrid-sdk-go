package manager

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/constants"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/models"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go"
	"github.com/threefoldtech/zos/pkg"
)

// DataManager manages data
type DataManager struct {
	config     *models.Config
	identity   substrate.Identity
	subConn    models.Sub
	rmbClient  rmb.Client
	rmbTimeout time.Duration
}

// NewDataManager creates a new DataManager
func NewDataManager(identity substrate.Identity, subConn models.Sub, config *models.Config, rmb rmb.Client) DataManager {
	return DataManager{config, identity, subConn, rmb, constants.TimeoutRMBResponse}
}

func (m *DataManager) Update(ctx context.Context) error {
	// TODO: loop on nodes without batch
	// update resources for nodes that have no claimed resources
	noClaimedResourcesNodes := FilterNodesNoClaimedResources(m.config.Nodes)

	// we do not update the resources for the nodes that have claimed resources because those resources should not be overwritten until the timeout
	nodesToUpdate, err := m.BatchStatistics(ctx, noClaimedResourcesNodes)
	if err != nil {
		return err
	}
	nodesToUpdate, err = m.BatchStoragePools(ctx, nodesToUpdate)
	if err != nil {
		return err
	}
	nodesToUpdate, err = m.BatchPublicConfigGet(ctx, nodesToUpdate)
	if err != nil {
		return err
	}
	nodesToUpdate, err = m.BatchListGPUs(ctx, nodesToUpdate)
	if err != nil {
		return err
	}
	nodesToUpdate, err = m.BatchUpdateHasRentContract(ctx, nodesToUpdate)
	if err != nil {
		return err
	}

	// update state: if we didn't get any response => node is offline
	for _, node := range m.config.Nodes {
		if containsNode(nodesToUpdate, node) {
			m.updatePowerState(node, true)
		} else {
			m.updatePowerState(node, false)
		}
	}

	return nil
}

func (m *DataManager) BatchStatistics(ctx context.Context, nodes []models.Node) ([]models.Node, error) {
	ctx, cancel := context.WithTimeout(ctx, m.rmbTimeout)
	defer cancel()

	const cmd = "zos.statistics.get"

	var successNodes []models.Node
	for _, node := range nodes {
		var result models.ZosResourcesStatistics

		err := m.rmbClient.Call(ctx, uint32(node.TwinID), cmd, nil, &result)
		if err != nil {
			log.Error().Err(err).Uint32("node ID", uint32(node.ID)).Msg("[DATA MANAGER] Failed to get statistics")
			continue
		}

		node.UpdateResources(result)
		successNodes = append(successNodes, node)
	}

	return successNodes, nil
}

func (m *DataManager) BatchStoragePools(ctx context.Context, nodes []models.Node) ([]models.Node, error) {
	ctx, cancel := context.WithTimeout(ctx, m.rmbTimeout)
	defer cancel()

	const cmd = "zos.storage.pools"

	var successNodes []models.Node
	for _, node := range nodes {
		var pools []pkg.PoolMetrics
		err := m.rmbClient.Call(ctx, uint32(node.TwinID), cmd, nil, &pools)
		if err != nil {
			log.Error().Err(err).Uint32("node ID", uint32(node.ID)).Msg("[DATA MANAGER] Failed to get storage pools")
			continue
		}

		node.Pools = pools
		successNodes = append(successNodes, node)
	}

	return successNodes, nil
}

func (m *DataManager) BatchPublicConfigGet(ctx context.Context, nodes []models.Node) ([]models.Node, error) {
	var successNodes []models.Node
	for _, node := range nodes {
		subNode, err := m.subConn.GetNode(uint32(node.ID))
		if err != nil {
			log.Error().Err(err).Uint32("node ID", uint32(node.ID)).Msg("[DATA MANAGER] Failed to get node from substrate")
			continue
		}

		node.PublicConfig = subNode.PublicConfig
		successNodes = append(successNodes, node)
	}
	return successNodes, nil
}

func (m *DataManager) BatchListGPUs(ctx context.Context, nodes []models.Node) ([]models.Node, error) {
	ctx, cancel := context.WithTimeout(ctx, m.rmbTimeout)
	defer cancel()

	const cmd = "zos.gpu.list"

	var successNodes []models.Node
	for _, node := range nodes {
		var gpus []models.GPU
		err := m.rmbClient.Call(ctx, uint32(node.TwinID), cmd, nil, &gpus)
		if err != nil {
			log.Error().Err(err).Uint32("node ID", uint32(node.ID)).Msg("[DATA MANAGER] Failed to get gpus")
			continue
		}

		node.GPUs = gpus
		successNodes = append(successNodes, node)
	}

	return successNodes, nil
}

// BatchUpdateHasRentContract updates if they have rent contract (done through tfchain)
func (m *DataManager) BatchUpdateHasRentContract(ctx context.Context, nodes []models.Node) ([]models.Node, error) {
	var successNodes []models.Node
	for _, node := range nodes {
		rentContract, err := m.subConn.GetNodeRentContract(uint32(node.ID))
		if err != nil {
			log.Error().Err(err).Uint32("node ID", uint32(node.ID)).Msg("[DATA MANAGER] Failed to get node rent contract")
			continue
		}

		node.HasActiveRentContract = rentContract != 0
		successNodes = append(successNodes, node)
	}
	return successNodes, nil
}

func (m *DataManager) updatePowerState(node models.Node, updated bool) {
	if !updated {
		// No response from ZOS node: if the state is waking up we wait for either the node to come up or the
		// timeout to hit. If the time out hits we change the state to off (AKA unsuccessful wakeup)
		// If the state was not waking up the node is considered off
		switch node.PowerState {
		case models.WakingUP:
			if time.Since(node.LastTimePowerStateChanged) < constants.TimeoutPowerStateChange {
				log.Info().Uint32("node ID", uint32(node.ID)).Msg("[DATA MANAGER] Node is waking up")
				return
			}
			log.Error().Uint32("node ID", uint32(node.ID)).Msg("[DATA MANAGER] Wakeup was unsuccessful. Putting its state back to off.")
		case models.ShuttingDown:
			log.Info().Uint32("node ID", uint32(node.ID)).Msg("[DATA MANAGER] Shutdown was successful.")
		case models.ON:
			log.Error().Uint32("node ID", uint32(node.ID)).Msg("[DATA MANAGER] Node is not responding while we expect it to.")
		case models.OFF:
			log.Info().Uint32("node ID", uint32(node.ID)).Msg("[DATA MANAGER] Node is OFF.")
		}

		if node.PowerState != models.OFF {
			node.LastTimePowerStateChanged = time.Now()
		}

		node.PowerState = models.OFF
		return
	}

	// We got a response from ZOS: it is still online. If the power state is shutting down
	// we check if the timeout has not exceeded yet. If it has we consider the attempt to shutting
	// down the down a failure and set the power state back to on
	if node.PowerState == models.ShuttingDown {
		if time.Since(node.LastTimePowerStateChanged) < constants.TimeoutPowerStateChange {
			log.Info().Uint32("node ID", uint32(node.ID)).Msg("[DATA MANAGER] Node is shutting down.")
			return
		}
		log.Error().Uint32("node ID", uint32(node.ID)).Msg("[DATA MANAGER] Shutdown was unsuccessful. Putting its state back to on.")
	} else {
		log.Info().Uint32("node ID", uint32(node.ID)).Msg("[DATA MANAGER] Node is ON.")
	}

	log.Debug().
		Uint32("node ID", uint32(node.ID)).
		Interface("resources", node.Resources).
		Interface("pools", node.Pools).
		Bool("has active rent contract", node.HasActiveRentContract).
		Msg("[DATA MANAGER] Capacity updated for node")

	if node.PowerState != models.ON {
		node.LastTimePowerStateChanged = time.Now()
	}

	node.PowerState = models.ON
	node.LastTimeAwake = time.Now()
}

// ContainsNode check if a slice of nodes contains an node
func containsNode(nodes []models.Node, node models.Node) bool {
	for _, n := range nodes {
		if node.ID == n.ID {
			return true
		}
	}
	return false
}

// FilterNodesClaimedResources filters nodes that have no claimed resources
func FilterNodesNoClaimedResources(nodes []models.Node) []models.Node {
	filtered := make([]models.Node, 0)
	for _, node := range nodes {
		if node.TimeoutClaimedResources.Before(time.Now()) {
			filtered = append(filtered, node)
		}
	}
	return filtered
}
