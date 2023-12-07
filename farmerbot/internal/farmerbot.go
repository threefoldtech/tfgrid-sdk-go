package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"slices"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/version"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
)

// FarmerBot for managing farms
type FarmerBot struct {
	state            *state
	substrateManager substrate.Manager
	powerManager     *PowerManager
	rmbNodeClient    RMB
	network          string
	mnemonicOrSeed   string
	identity         substrate.Identity
}

// NewFarmerBot generates a new farmer bot
func NewFarmerBot(ctx context.Context, config Config, network, mnemonicOrSeed string) (FarmerBot, error) {
	identity, err := substrate.NewIdentityFromSr25519Phrase(mnemonicOrSeed)
	if err != nil {
		return FarmerBot{}, err
	}

	farmerbot := FarmerBot{
		substrateManager: substrate.NewManager(SubstrateURLs[network]...),
		network:          network,
		mnemonicOrSeed:   mnemonicOrSeed,
		identity:         identity,
	}

	subConn, err := farmerbot.substrateManager.Substrate()
	if err != nil {
		return FarmerBot{}, err
	}

	rmb, err := peer.NewRpcClient(ctx, peer.KeyTypeSr25519, farmerbot.mnemonicOrSeed, relayURLS[network], fmt.Sprintf("farmerbot-rpc-%d", config.FarmID), subConn, true)
	if err != nil {
		return FarmerBot{}, errors.Wrap(err, "could not create rmb client")
	}

	farmerbot.rmbNodeClient = NewRmbNodeClient(rmb)

	state, err := newState(ctx, subConn, farmerbot.rmbNodeClient, config)
	if err != nil {
		return FarmerBot{}, err
	}
	farmerbot.state = state

	powerManager := NewPowerManager(identity, farmerbot.state)
	farmerbot.powerManager = &powerManager

	return farmerbot, nil
}

// Run runs farmerbot to update nodes and power management
func (f *FarmerBot) Run(ctx context.Context) error {
	subConn, err := f.substrateManager.Substrate()
	if err != nil {
		return err
	}

	defer subConn.Close()

	if err := f.serve(ctx, subConn); err != nil {
		return err
	}

	log.Info().Msg("up and running...")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			startTime := time.Now()

			log.Debug().Msg("check nodes latest updates")
			for _, node := range f.state.nodes {
				neverShutDown := slices.Contains(f.state.config.NeverShutDownNodes, uint32(node.ID))
				hasClaimedResources := node.timeoutClaimedResources.After(time.Now())
				dedicatedFarm := f.state.farm.DedicatedFarm
				overProvisionCPU := f.state.config.Power.OverProvisionCPU

				ltsNode, err := getNodeWithLatestChanges(ctx, subConn, f.rmbNodeClient, uint32(node.ID), neverShutDown, hasClaimedResources, dedicatedFarm, overProvisionCPU)
				if err != nil {
					log.Error().Err(err).Uint32("node ID", uint32(ltsNode.ID)).Msg("Get latest updates for node failed")
					if ltsNode.powerState == on {
						log.Error().Uint32("node ID", uint32(ltsNode.ID)).Msg("Node is not responding while we expect it to.")
					}
					continue
				}

				if ltsNode.powerState == wakingUP && time.Since(ltsNode.lastTimePowerStateChanged) > timeoutPowerStateChange {
					log.Warn().Uint32("node ID", uint32(ltsNode.ID)).Msg("Wakeup was unsuccessful. Putting its state back to off.")
					ltsNode.powerState = off
					ltsNode.lastTimePowerStateChanged = time.Now()
				}

				if ltsNode.powerState == shuttingDown && time.Since(ltsNode.lastTimePowerStateChanged) > timeoutPowerStateChange {
					log.Warn().Uint32("node ID", uint32(ltsNode.ID)).Msg("Shutdown was unsuccessful. Putting its state back to on.")
					ltsNode.powerState = on
					ltsNode.lastTimeAwake = time.Now()
					ltsNode.lastTimePowerStateChanged = time.Now()
				}

				err = f.state.updateNode(node)
				if err != nil {
					log.Error().Err(err).Uint32("node ID", uint32(ltsNode.ID)).Msg("failed to update node")
					continue
				}

				log.Debug().Uint32("node ID", uint32(ltsNode.ID)).Msg("Node is updated with latest changes successfully")
			}

			log.Debug().Msg("periodic wake up")
			err = f.powerManager.PeriodicWakeUp(subConn)
			if err != nil {
				log.Error().Err(err).Msg("failed to perform periodic wake up")
			}

			log.Debug().Msg("power management")
			err = f.powerManager.PowerManagement(subConn)
			if err != nil {
				log.Error().Err(err).Msg("failed to power management nodes")
			}

			delta := time.Since(startTime)
			log.Debug().Float64("Elapsed time for updates in minutes", delta.Minutes())

			// sleep if finished before the update timeout
			var timeToSleep float64
			if delta.Minutes() >= timeoutUpdate.Minutes() {
				timeToSleep = 0
			} else {
				timeToSleep = timeoutUpdate.Minutes() - delta.Minutes()
			}
			time.Sleep(time.Duration(timeToSleep) * time.Minute)
		}
	}
}

func (f *FarmerBot) serve(ctx context.Context, sub *substrate.Substrate) error {
	router := peer.NewRouter()
	farmerbot := router.SubRoute("farmerbot")

	farmRouter := farmerbot.SubRoute("farmmanager")
	nodeRouter := farmerbot.SubRoute("nodemanager")
	powerRouter := farmerbot.SubRoute("powermanager")

	// TODO: didn't work
	// powerRouter.Use(f.authorize)

	farmerTwinID, err := sub.GetTwinByPubKey(f.identity.PublicKey())
	if err != nil {
		return err
	}

	farmRouter.WithHandler("version", func(ctx context.Context, payload []byte) (interface{}, error) {
		return version.Version, nil
	})

	nodeRouter.WithHandler("findnode", func(ctx context.Context, payload []byte) (interface{}, error) {
		var options NodeOptions

		if err := json.Unmarshal(payload, &options); err != nil {
			return nil, fmt.Errorf("failed to load request payload: %w", err)
		}

		nodeID, err := f.powerManager.FindNode(sub, options)
		return nodeID, err
	})

	powerRouter.WithHandler("includenode", func(ctx context.Context, payload []byte) (interface{}, error) {
		err := authorize(ctx, farmerTwinID)
		if err != nil {
			return nil, err
		}

		var nodeID uint32
		if err := json.Unmarshal(payload, &nodeID); err != nil {
			return nil, fmt.Errorf("failed to load request payload: %w", err)
		}

		_, ok := f.state.nodes[nodeID]
		if ok {
			return nil, fmt.Errorf("node %d already exists", nodeID)
		}

		if slices.Contains(f.state.config.ExcludedNodes, nodeID) ||
			len(f.state.config.ExcludedNodes) == 0 && !slices.Contains(f.state.config.IncludedNodes, nodeID) {
			return nil, fmt.Errorf("node %d is excluded, cannot add it", nodeID)
		}

		neverShutDown := slices.Contains(f.state.config.NeverShutDownNodes, nodeID)
		node, err := getNodeWithLatestChanges(ctx, sub, f.rmbNodeClient, nodeID, neverShutDown, false, f.state.farm.DedicatedFarm, f.state.config.Power.OverProvisionCPU)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to include node with id %d", nodeID)
		}

		f.state.m.Lock()
		f.state.nodes[nodeID] = node
		f.state.m.Unlock()

		return nil, nil
	})

	powerRouter.WithHandler("poweroff", func(ctx context.Context, payload []byte) (interface{}, error) {
		err := authorize(ctx, farmerTwinID)
		if err != nil {
			return nil, err
		}

		if err := f.validateAccountEnoughBalance(sub); err != nil {
			return nil, fmt.Errorf("failed to validate account balance: %w", err)
		}

		var nodeID uint32
		if err := json.Unmarshal(payload, &nodeID); err != nil {
			return nil, fmt.Errorf("failed to load request payload: %w", err)
		}

		if err := f.powerManager.PowerOff(sub, nodeID); err != nil {
			return nil, fmt.Errorf("failed to power off node %d: %w", nodeID, err)
		}

		// Exclude node from farmerbot management
		// (It is not allowed if we tried to power on a node the farmer decided to power off)
		// the farmer should include it again if he wants to the bot to manage it
		f.state.m.Lock()
		delete(f.state.nodes, nodeID)
		f.state.m.Unlock()

		return nil, nil
	})

	powerRouter.WithHandler("poweron", func(ctx context.Context, payload []byte) (interface{}, error) {
		err := authorize(ctx, farmerTwinID)
		if err != nil {
			return nil, err
		}

		if err := f.validateAccountEnoughBalance(sub); err != nil {
			return nil, fmt.Errorf("failed to validate account balance: %w", err)
		}

		var nodeID uint32
		if err := json.Unmarshal(payload, &nodeID); err != nil {
			return nil, fmt.Errorf("failed to load request payload: %w", err)
		}

		if err := f.powerManager.PowerOn(sub, nodeID); err != nil {
			return nil, fmt.Errorf("failed to power on node %d: %w", nodeID, err)
		}

		// Exclude node from farmerbot management
		// (It is not allowed if we tried to power off a node the farmer decided to power on)
		// the farmer should include it again if he wants to the bot to manage it
		f.state.m.Lock()
		delete(f.state.nodes, nodeID)
		f.state.m.Unlock()

		return nil, nil
	})

	_, err = peer.NewPeer(
		ctx,
		peer.KeyTypeSr25519,
		f.mnemonicOrSeed,
		relayURLS[f.network],
		fmt.Sprintf("farmerbot-%d", f.state.farm.ID),
		sub,
		true,
		router.Serve,
	)

	if err != nil {
		return errors.Wrap(err, "failed to create farmerbot direct peer")
	}

	return nil
}

func (f *FarmerBot) validateAccountEnoughBalance(sub *substrate.Substrate) error {
	accountAddress, err := substrate.FromAddress(f.identity.Address())
	if err != nil {
		return err
	}

	balance, err := sub.GetBalance(accountAddress)
	if err != nil && !errors.Is(err, substrate.ErrAccountNotFound) {
		return errors.Wrap(err, "failed to get a valid account")
	}

	if balance.Free.Cmp(big.NewInt(20000)) == -1 {
		return errors.Errorf("account contains %f tft, min fee is 0.002 tft", float64(balance.Free.Int64())/math.Pow(10, 7))
	}

	return nil
}

func authorize(ctx context.Context, farmerTwinID uint32) error {
	twinID := peer.GetTwinID(ctx)
	if twinID != farmerTwinID {
		return fmt.Errorf("you are not authorized for this action. your twin id is `%d`, only the farm owner with twin id `%d` is authorized", twinID, farmerTwinID)
	}
	return nil
}
