package server

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/constants"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/manager"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/models"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/version"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
)

// FarmerBot for managing farms
type FarmerBot struct {
	config           *models.Config
	substrateManager substrate.Manager
	powerManager     *manager.PowerManager
	dataManager      manager.DataManager
	network          string
	mnemonicOrSeed   string
	identity         substrate.Identity
}

// NewFarmerBot generates a new farmer bot
func NewFarmerBot(ctx context.Context, inputs models.InputConfig, network, mnemonicOrSeed string) (FarmerBot, error) {
	substrateManager := substrate.NewManager(constants.SubstrateURLs[network]...)

	identity, err := substrate.NewIdentityFromSr25519Phrase(mnemonicOrSeed)
	if err != nil {
		return FarmerBot{}, err
	}

	farmerbot := FarmerBot{
		substrateManager: substrateManager,
		network:          network,
		mnemonicOrSeed:   mnemonicOrSeed,
		identity:         identity,
	}

	subConn, err := substrateManager.Substrate()
	if err != nil {
		return FarmerBot{}, err
	}

	// TODO:
	// defer subConn.Close()

	err = farmerbot.SetConfig(inputs)
	if err != nil {
		return FarmerBot{}, err
	}

	powerManager := manager.NewPowerManager(identity, farmerbot.config)
	farmerbot.powerManager = &powerManager

	rmb, err := peer.NewRpcClient(ctx, peer.KeyTypeSr25519, farmerbot.mnemonicOrSeed, constants.RelayURLS[network], fmt.Sprintf("farmerbot-rpc-%d", farmerbot.config.Farm.ID), subConn, true)
	if err != nil {
		return FarmerBot{}, errors.Wrap(err, "could not create rmb client")
	}

	rmbNodeClient := manager.NewRmbNodeClient(rmb)
	farmerbot.dataManager = manager.NewDataManager(farmerbot.config, rmbNodeClient)

	return farmerbot, nil
}

// Run runs farmerbot to update nodes and power management
func (f *FarmerBot) Run(ctx context.Context) error {
	subConn, err := f.substrateManager.Substrate()
	if err != nil {
		return err
	}

	defer subConn.Close()

	if err := f.start(ctx, subConn); err != nil {
		return err
	}

	if err := f.serve(ctx, subConn); err != nil {
		return err
	}

	go f.update(ctx, subConn)
	log.Info().Msg("up and running...")

	// graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownRelease()

	if err := f.stop(shutdownCtx, subConn); err != nil {
		return err
	}

	log.Info().Msg("graceful shutdown successful")
	return nil
}

func (f *FarmerBot) SetConfig(inputs models.InputConfig) error {
	subConn, err := f.substrateManager.Substrate()
	if err != nil {
		return err
	}

	defer subConn.Close()

	if f.config == nil {
		f.config = &models.Config{}
	}

	return f.config.Set(subConn, inputs)
}

func (f *FarmerBot) start(ctx context.Context, sub models.Sub) error {
	return f.powerManager.PowerOnAllNodes(sub)
}

func (f *FarmerBot) stop(ctx context.Context, sub models.Sub) error {
	return f.powerManager.PowerOnAllNodes(sub)
}

func (f *FarmerBot) update(ctx context.Context, sub models.Sub) {
	for {
		startTime := time.Now()

		log.Debug().Msg("data update")
		err := f.dataManager.Update(ctx, sub)
		if err != nil {
			log.Error().Err(err).Msg("failed to update")
		}

		log.Debug().Msg("periodic wake up")
		err = f.powerManager.PeriodicWakeUp(sub)
		if err != nil {
			log.Error().Err(err).Msg("failed to perform periodic wake up")
		}

		log.Debug().Msg("power management")
		err = f.powerManager.PowerManagement(sub)
		if err != nil {
			log.Error().Err(err).Msg("failed to power management nodes")
		}

		delta := time.Since(startTime)
		log.Debug().Float64("Elapsed time for updates in minutes", delta.Minutes())

		// sleep if finished before the update timeout
		var timeToSleep float64
		if delta.Minutes() >= constants.TimeoutUpdate.Minutes() {
			timeToSleep = 0
		} else {
			timeToSleep = constants.TimeoutUpdate.Minutes() - delta.Minutes()
		}
		time.Sleep(time.Duration(timeToSleep) * time.Minute)
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
		var options models.NodeOptions

		if err := json.Unmarshal(payload, &options); err != nil {
			return nil, fmt.Errorf("failed to load request payload: %w", err)
		}

		nodeID, err := f.powerManager.FindNode(sub, options)
		return nodeID, err
	})

	powerRouter.WithHandler("poweroff", func(ctx context.Context, payload []byte) (interface{}, error) {
		err := authorize(ctx, farmerTwinID)
		if err != nil {
			return nil, err
		}

		var nodeID uint32
		if err := f.validateAccountEnoughBalance(sub); err != nil {
			return nil, fmt.Errorf("failed to validate account balance: %w", err)
		}

		if err := json.Unmarshal(payload, &nodeID); err != nil {
			return nil, fmt.Errorf("failed to load request payload: %w", err)
		}

		return nil, f.powerManager.PowerOff(sub, nodeID)
	})

	powerRouter.WithHandler("poweron", func(ctx context.Context, payload []byte) (interface{}, error) {
		err := authorize(ctx, farmerTwinID)
		if err != nil {
			return nil, err
		}

		var nodeID uint32
		if err := f.validateAccountEnoughBalance(sub); err != nil {
			return nil, fmt.Errorf("failed to validate account balance: %w", err)
		}

		if err := json.Unmarshal(payload, &nodeID); err != nil {
			return nil, fmt.Errorf("failed to load request payload: %w", err)
		}

		err = f.powerManager.PowerOn(sub, nodeID)
		return nil, err
	})

	_, err = peer.NewPeer(
		ctx,
		peer.KeyTypeSr25519,
		f.mnemonicOrSeed,
		constants.RelayURLS[f.network],
		fmt.Sprintf("farmerbot-%d", f.config.Farm.ID),
		sub,
		true,
		router.Serve,
	)

	if err != nil {
		return errors.New("failed to create farmerbot direct peer")
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
