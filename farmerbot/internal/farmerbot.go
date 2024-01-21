package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
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
	*state
	substrateManager substrate.Manager
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

	rmb, err := peer.NewRpcClient(ctx, peer.KeyTypeSr25519, farmerbot.mnemonicOrSeed, relayURLs[network], fmt.Sprintf("farmerbot-rpc-%d", config.FarmID), farmerbot.substrateManager, true)
	if err != nil {
		return FarmerBot{}, fmt.Errorf("could not create rmb client with error %w", err)
	}

	farmerbot.rmbNodeClient = NewRmbNodeClient(rmb)

	subConn, err := farmerbot.substrateManager.Substrate()
	if err != nil {
		return FarmerBot{}, err
	}
	defer subConn.Close()

	state, err := newState(ctx, subConn, farmerbot.rmbNodeClient, config)
	if err != nil {
		return FarmerBot{}, err
	}
	farmerbot.state = state

	return farmerbot, nil
}

// Run runs farmerbot to update nodes and power management
func (f *FarmerBot) Run(ctx context.Context) error {
	if err := f.serve(ctx); err != nil {
		return err
	}

	log.Info().Msg("up and running...")

	for {
		subConn, err := f.substrateManager.Substrate()
		if err != nil {
			log.Error().Err(err).Msg("failed to open substrate connection")
		}

		err = f.iterateOnNodes(ctx, subConn)
		if err != nil {
			log.Error().Err(err).Msg("failed to iterate on nodes")
		}
		subConn.Close()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(timeoutUpdate):
		}
	}
}

func (f *FarmerBot) serve(ctx context.Context) error {
	router := peer.NewRouter()
	farmerbot := router.SubRoute("farmerbot")

	farmRouter := farmerbot.SubRoute("farmmanager")
	nodeRouter := farmerbot.SubRoute("nodemanager")
	powerRouter := farmerbot.SubRoute("powermanager")

	// TODO: didn't work
	// powerRouter.Use(f.authorize)

	subConn, err := f.substrateManager.Substrate()
	if err != nil {
		return err
	}
	// defer subConn.Close()

	balance, err := f.getAccountBalanceInTFT(subConn)
	if err != nil {
		return err
	}

	if balance < minBalanceToRun {
		return fmt.Errorf("account contains %v tft, you need to have at least %v tft", balance, minBalanceToRun)
	}

	if balance < recommendedBalanceToRun {
		log.Warn().Float64("current balance", balance).Msgf("Recommended balance to run farmerbot is %v tft", recommendedBalanceToRun)
	}

	farmerTwinID, err := subConn.GetTwinByPubKey(f.identity.PublicKey())
	if err != nil {
		return err
	}

	farmRouter.WithHandler("version", func(ctx context.Context, payload []byte) (interface{}, error) {
		return version.Version, nil
	})

	nodeRouter.WithHandler("findnode", func(ctx context.Context, payload []byte) (interface{}, error) {
		var options NodeFilterOption

		if err := json.Unmarshal(payload, &options); err != nil {
			return nil, fmt.Errorf("failed to load request payload: %w", err)
		}

		nodeID, err := f.findNode(subConn, options)
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

		_, ok := f.nodes[nodeID]
		if ok {
			return nil, fmt.Errorf("node %d already exists", nodeID)
		}

		if slices.Contains(f.config.ExcludedNodes, nodeID) ||
			len(f.config.ExcludedNodes) == 0 && !slices.Contains(f.config.IncludedNodes, nodeID) {
			return nil, fmt.Errorf("node %d is excluded, cannot add it", nodeID)
		}

		neverShutDown := slices.Contains(f.config.NeverShutDownNodes, nodeID)
		node, err := getNode(ctx, subConn, f.rmbNodeClient, nodeID, neverShutDown, false, f.farm.DedicatedFarm, on)
		if err != nil {
			return nil, fmt.Errorf("failed to include node with id %d with error: %w", nodeID, err)
		}

		f.state.addNode(node)
		return nil, nil
	})

	powerRouter.WithHandler("poweroff", func(ctx context.Context, payload []byte) (interface{}, error) {
		err := authorize(ctx, farmerTwinID)
		if err != nil {
			return nil, err
		}

		if err := f.validateAccountEnoughBalance(subConn); err != nil {
			return nil, fmt.Errorf("failed to validate account balance: %w", err)
		}

		var nodeID uint32
		if err := json.Unmarshal(payload, &nodeID); err != nil {
			return nil, fmt.Errorf("failed to load request payload: %w", err)
		}

		if err := f.powerOff(subConn, nodeID); err != nil {
			return nil, fmt.Errorf("failed to power off node %d: %w", nodeID, err)
		}

		// Exclude node from farmerbot management
		// (It is not allowed if we tried to power on a node the farmer decided to power off)
		// the farmer should include it again if he wants to the bot to manage it
		f.state.deleteNode(nodeID)
		return nil, nil
	})

	powerRouter.WithHandler("poweron", func(ctx context.Context, payload []byte) (interface{}, error) {
		err := authorize(ctx, farmerTwinID)
		if err != nil {
			return nil, err
		}

		if err := f.validateAccountEnoughBalance(subConn); err != nil {
			return nil, fmt.Errorf("failed to validate account balance: %w", err)
		}

		var nodeID uint32
		if err := json.Unmarshal(payload, &nodeID); err != nil {
			return nil, fmt.Errorf("failed to load request payload: %w", err)
		}

		if err := f.powerOn(subConn, nodeID); err != nil {
			return nil, fmt.Errorf("failed to power on node %d: %w", nodeID, err)
		}

		// Exclude node from farmerbot management
		// (It is not allowed if we tried to power off a node the farmer decided to power on)
		// the farmer should include it again if he wants to the bot to manage it
		f.state.deleteNode(nodeID)
		return nil, nil
	})

	_, err = peer.NewPeer(
		ctx,
		f.mnemonicOrSeed,
		f.substrateManager,
		router.Serve,
		peer.WithRelay(relayURLs[f.network]),
		peer.WithSession(fmt.Sprintf("farmerbot-%d", f.farm.ID)),
	)

	if err != nil {
		return fmt.Errorf("failed to create farmerbot direct peer with error: %w", err)
	}

	return nil
}

func (f *FarmerBot) iterateOnNodes(ctx context.Context, subConn Substrate) error {
	roundStart := time.Now()
	var wakeUpCalls uint8

	log.Debug().Msg("Fetch nodes")
	farmNodes, err := subConn.GetNodes(uint32(f.state.farm.ID))
	if err != nil {
		return err
	}

	// remove nodes that don't exist anymore in the farm
	for nodeID := range f.state.nodes {
		if !slices.Contains(farmNodes, nodeID) {
			f.state.deleteNode(nodeID)
		}
	}

	for _, nodeID := range farmNodes {
		if slices.Contains(f.state.config.ExcludedNodes, nodeID) {
			continue
		}

		// if the user specified included nodes or
		// no nodes are specified so all nodes will be added (except excluded)
		if !slices.Contains(f.state.config.IncludedNodes, nodeID) && len(f.state.config.IncludedNodes) > 0 {
			continue
		}

		log.Debug().Uint32("nodeID", nodeID).Msg("Add/update node")
		err = f.addOrUpdateNode(ctx, subConn, nodeID)
		if err != nil {
			log.Error().Err(err).Send()
		}

		node := f.state.nodes[nodeID]

		if node.powerState == off && (node.neverShutDown || node.hasActiveRentContract) {
			log.Debug().Uint32("nodeID", nodeID).Msg("Power on node because it is set to never shutdown")
			err := f.powerOn(subConn, nodeID)
			if err != nil {
				log.Error().Err(err).Send()
			}
		}

		if roundStart.Day() == 1 && roundStart.Hour() == 1 && roundStart.Minute() < int(timeoutUpdate.Minutes()) {
			log.Debug().Uint32("nodeID", nodeID).Msg("Reset random wake-up times the first day of the month")
			node.timesRandomWakeUps = 0
			err = f.state.updateNode(node)
			if err != nil {
				log.Error().Err(err).Send()
			}
		}

		if f.shouldWakeUp(subConn, node, roundStart) && wakeUpCalls < f.config.Power.PeriodicWakeUpLimit {
			err := f.powerOn(subConn, nodeID)
			if err != nil {
				log.Error().Err(err).Uint32("nodeID", nodeID).Msg("failed to power on node")
				continue
			}

			wakeUpCalls++
		}

	}

	err = f.manageNodesPower(subConn)
	if err != nil {
		return fmt.Errorf("failed to manage nodes power with error: %w", err)
	}

	return nil
}

func (f *FarmerBot) addOrUpdateNode(ctx context.Context, subConn Substrate, nodeID uint32) error {
	neverShutDown := slices.Contains(f.state.config.NeverShutDownNodes, nodeID)

	oldNode, nodeExists := f.state.nodes[nodeID]
	if nodeExists {
		updateErr := oldNode.update(ctx, subConn, f.rmbNodeClient, neverShutDown, f.state.farm.DedicatedFarm)

		// update old node state even if it failed
		if err := f.state.updateNode(oldNode); err != nil {
			return fmt.Errorf("failed to update node state %d with error: %w", uint32(oldNode.ID), err)
		}

		if updateErr != nil {
			return fmt.Errorf("failed to update node %d with error: %w", uint32(oldNode.ID), updateErr)
		}

		log.Debug().Uint32("nodeID", nodeID).Msg("Node is updated with latest changes successfully")
		return nil
	}

	// if node doesn't exist, we should add it
	nodeObj, err := getNode(ctx, subConn, f.rmbNodeClient, nodeID, neverShutDown, false, f.state.farm.DedicatedFarm, on)
	if err != nil {
		return fmt.Errorf("failed to get node %d: %w", nodeID, err)
	}

	f.state.addNode(nodeObj)
	log.Debug().Uint32("nodeID", nodeID).Msg("Node is added with latest changes successfully")
	return nil
}

func (f *FarmerBot) shouldWakeUp(sub Substrate, node node, roundStart time.Time) bool {
	if node.powerState != off {
		return false
	}

	periodicWakeUpStart := f.config.Power.PeriodicWakeUpStart.PeriodicWakeUpTime()
	if periodicWakeUpStart.Before(roundStart) && node.lastTimeAwake.Before(periodicWakeUpStart) {
		// we wake up the node if the periodic wake up start time has started and only if the last time the node was awake
		// was before the periodic wake up start of that day
		log.Info().Uint32("nodeID", uint32(node.ID)).Msg("Periodic wake up")
		return true
	}

	nodesLen := len(f.nodes)

	// TODO:
	if node.timesRandomWakeUps < defaultRandomWakeUpsAMonth &&
		int(rand.Int31())%((8460-(defaultRandomWakeUpsAMonth*6)-
			(defaultRandomWakeUpsAMonth*(nodesLen-1))/int(math.Min(float64(f.config.Power.PeriodicWakeUpLimit), float64(nodesLen))))/
			defaultRandomWakeUpsAMonth) == 0 {
		// Random periodic wake up (10 times a month on average if the node is almost always down)
		// we execute this code every 5 minutes => 288 times a day => 8640 times a month on average (30 days)
		// but we have 30 minutes of periodic wake up every day (6 times we do not go through this code) => so 282 times a day => 8460 times a month on average (30 days)
		// as we do a random wake up 10 times a month we know the node will be on for 30 minutes 10 times a month so we can subtract 6 times the amount of random wake ups a month
		// we also do not go through the code if we have woken up too many nodes at once => subtract (10 * (n-1))/min(periodic_wake up_limit, amount_of_nodes) from 8460
		// now we can divide that by 10 and randomly generate a number in that range, if it's 0 we do the random wake up
		log.Info().Uint32("nodeID", uint32(node.ID)).Msg("Random wake up")
		return true
	}

	return false
}

func (f *FarmerBot) getAccountBalanceInTFT(sub *substrate.Substrate) (float64, error) {
	accountAddress, err := substrate.FromAddress(f.identity.Address())
	if err != nil {
		return 0, err
	}

	balance, err := sub.GetBalance(accountAddress)
	if err != nil && !errors.Is(err, substrate.ErrAccountNotFound) {
		return 0, fmt.Errorf("failed to get a valid account with error: %w", err)
	}

	return float64(balance.Free.Int64()) / math.Pow(10, 7), nil
}

func (f *FarmerBot) validateAccountEnoughBalance(sub *substrate.Substrate) error {
	required := 0.002

	balance, err := f.getAccountBalanceInTFT(sub)
	if err != nil {
		return err
	}

	if balance < required {
		return fmt.Errorf("account contains %v tft, you need to have at least %v tft", balance, required)
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
