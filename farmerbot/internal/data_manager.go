package internal

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/constants"
)

// DataManager manages data
type DataManager struct {
	state         *state
	rmbNodeClient RMB
}

// NewDataManager creates a new Data updates Manager
func NewDataManager(state *state, rmbNodeClient RMB) DataManager {
	return DataManager{state, rmbNodeClient}
}

func (m *DataManager) Update(ctx context.Context, sub Sub) error {
	var failedNodes []node
	for _, node := range m.state.nodes {
		// we ping nodes (the ones with claimed resources)
		if node.TimeoutClaimedResources.Before(time.Now()) {
			err := m.rmbNodeClient.SystemVersion(ctx, uint32(node.TwinID))
			if err != nil {
				log.Error().Err(err).Uint32("node ID", uint32(node.ID)).Str("manager", "data").Msg("Failed to get system version of node")
				continue
			}
			continue
		}

		// update resources for nodes that have no claimed resources
		// we do not update the resources for the nodes that have claimed resources because those resources should not be overwritten until the timeout

		stats, err := m.rmbNodeClient.Statistics(ctx, uint32(node.TwinID))
		if err != nil {
			log.Error().Err(err).Uint32("node ID", uint32(node.ID)).Str("manager", "data").Msg("Failed to get statistics")
			failedNodes = append(failedNodes, node)
			continue
		}

		node.updateResources(stats)

		pools, err := m.rmbNodeClient.GetStoragePools(ctx, uint32(node.TwinID))
		if err != nil {
			log.Error().Err(err).Uint32("node ID", uint32(node.ID)).Str("manager", "data").Msg("Failed to get pools")
			failedNodes = append(failedNodes, node)
			continue
		}

		node.Pools = pools

		subNode, err := sub.GetNode(uint32(node.ID))
		if err != nil {
			log.Error().Err(err).Uint32("node ID", uint32(node.ID)).Str("manager", "data").Msg("Failed to get node from substrate")
			failedNodes = append(failedNodes, node)
			continue
		}

		node.PublicConfig = subNode.PublicConfig

		gpus, err := m.rmbNodeClient.ListGPUs(ctx, uint32(node.TwinID))
		if err != nil {
			log.Error().Err(err).Uint32("node ID", uint32(node.ID)).Str("manager", "data").Msg("Failed to get gpus")
			failedNodes = append(failedNodes, node)
			continue
		}

		node.GPUs = gpus

		rentContract, err := sub.GetNodeRentContract(uint32(node.ID))
		if err != nil {
			log.Error().Err(err).Uint32("node ID", uint32(node.ID)).Str("manager", "data").Msg("Failed to get node rent contract")
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

func (m *DataManager) updatePowerState(nodeObj node, updated bool) {
	if !updated {
		// No response from ZOS node: if the state is waking up we wait for either the node to come up or the
		// timeout to hit. If the time out hits we change the state to off (AKA unsuccessful wakeup)
		// If the state was not waking up the node is considered off
		switch nodeObj.PowerState {
		case WakingUP:
			if time.Since(nodeObj.LastTimePowerStateChanged) < constants.TimeoutPowerStateChange {
				log.Info().Uint32("node ID", uint32(nodeObj.ID)).Str("manager", "data").Msg("Node is waking up")
				return
			}
			log.Error().Uint32("node ID", uint32(nodeObj.ID)).Str("manager", "data").Msg("Wakeup was unsuccessful. Putting its state back to off.")
		case ShuttingDown:
			log.Info().Uint32("node ID", uint32(nodeObj.ID)).Str("manager", "data").Msg("Shutdown was successful.")
		case ON:
			log.Error().Uint32("node ID", uint32(nodeObj.ID)).Str("manager", "data").Msg("Node is not responding while we expect it to.")
		case OFF:
			log.Info().Uint32("node ID", uint32(nodeObj.ID)).Str("manager", "data").Msg("Node is OFF.")
		}

		if nodeObj.PowerState != OFF {
			nodeObj.LastTimePowerStateChanged = time.Now()
		}

		nodeObj.PowerState = OFF
		return
	}

	// We got a response from ZOS: it is still online. If the power state is shutting down
	// we check if the timeout has not exceeded yet. If it has we consider the attempt to shutting
	// down the down a failure and set the power state back to on
	if nodeObj.PowerState == ShuttingDown {
		if time.Since(nodeObj.LastTimePowerStateChanged) < constants.TimeoutPowerStateChange {
			log.Info().Uint32("node ID", uint32(nodeObj.ID)).Str("manager", "data").Msg("Node is shutting down.")
			return
		}
		log.Error().Uint32("node ID", uint32(nodeObj.ID)).Str("manager", "data").Msg("Shutdown was unsuccessful. Putting its state back to on.")
	} else {
		log.Info().Uint32("node ID", uint32(nodeObj.ID)).Str("manager", "data").Msg("Node is ON.")
	}

	log.Debug().
		Uint32("node ID", uint32(nodeObj.ID)).
		Interface("resources", nodeObj.Resources).
		Interface("pools", nodeObj.Pools).
		Bool("has active rent contract", nodeObj.HasActiveRentContract).
		Msg("[DATA MANAGER] Capacity updated for node")

	if nodeObj.PowerState != ON {
		nodeObj.LastTimePowerStateChanged = time.Now()
	}

	nodeObj.PowerState = ON
	nodeObj.LastTimeAwake = time.Now()
}
