package manager

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/constants"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/models"
)

// DataManager manages data
type DataManager struct {
	config        *models.Config
	subConn       models.Sub
	rmbNodeClient RMB
}

// NewDataManager creates a new Data updates Manager
func NewDataManager(subConn models.Sub, config *models.Config, rmbNodeClient RMB) DataManager {
	return DataManager{config, subConn, rmbNodeClient}
}

func (m *DataManager) Update(ctx context.Context) error {
	var failedNodes []models.Node
	for _, node := range m.config.Nodes {
		// we ping nodes (the ones with claimed resources)
		if node.TimeoutClaimedResources.Before(time.Now()) {
			err := m.rmbNodeClient.SystemVersion(ctx, uint32(node.TwinID))
			if err != nil {
				log.Error().Err(err).Msgf("[DATA MANAGER] Failed to get system version of node %d", node.ID)
				continue
			}
			continue
		}

		// update resources for nodes that have no claimed resources
		// we do not update the resources for the nodes that have claimed resources because those resources should not be overwritten until the timeout

		stats, err := m.rmbNodeClient.Statistics(ctx, uint32(node.TwinID))
		if err != nil {
			log.Error().Err(err).Uint32("node ID", uint32(node.ID)).Msg("[DATA MANAGER] Failed to get statistics")
			failedNodes = append(failedNodes, node)
			continue
		}

		node.UpdateResources(stats)

		pools, err := m.rmbNodeClient.GetStoragePools(ctx, uint32(node.TwinID))
		if err != nil {
			log.Error().Err(err).Uint32("node ID", uint32(node.ID)).Msg("[DATA MANAGER] Failed to get pools")
			failedNodes = append(failedNodes, node)
			continue
		}

		node.Pools = pools

		subNode, err := m.subConn.GetNode(uint32(node.ID))
		if err != nil {
			log.Error().Err(err).Uint32("node ID", uint32(node.ID)).Msg("[DATA MANAGER] Failed to get node from substrate")
			failedNodes = append(failedNodes, node)
			continue
		}

		node.PublicConfig = subNode.PublicConfig

		gpus, err := m.rmbNodeClient.ListGPUs(ctx, uint32(node.TwinID))
		if err != nil {
			log.Error().Err(err).Uint32("node ID", uint32(node.ID)).Msg("[DATA MANAGER] Failed to get gpus")
			failedNodes = append(failedNodes, node)
			continue
		}

		node.GPUs = gpus

		rentContract, err := m.subConn.GetNodeRentContract(uint32(node.ID))
		if err != nil {
			log.Error().Err(err).Uint32("node ID", uint32(node.ID)).Msg("[DATA MANAGER] Failed to get node rent contract")
			failedNodes = append(failedNodes, node)
			continue
		}

		node.HasActiveRentContract = rentContract != 0

		m.updatePowerState(node, true)
	}

	// update state: if we didn't get any response => node is offline
	for _, node := range failedNodes {
		m.updatePowerState(node, false)
	}

	return nil
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
