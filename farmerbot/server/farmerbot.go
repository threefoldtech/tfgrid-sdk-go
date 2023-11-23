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
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/parser"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/version"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
)

// TODO: get farms and nodes from substrate and rmb

// FarmerBot for managing farms
type FarmerBot struct {
	config       *models.Config
	sub          substrate.Substrate
	rmbClient    rmb.Client
	powerManager manager.PowerManager
	nodeManager  manager.NodeManager
	dataManager  manager.DataManager
	twinID       uint32
	network      string
	mnemonic     string
	identity     substrate.Identity
}

// TODO: seed
// NewFarmerBot generates a new farmer bot
func NewFarmerBot(ctx context.Context, configPath, network, mnemonic, redisAddr string) (FarmerBot, error) {
	jsonContent, format, err := parser.ReadFile(configPath)
	if err != nil {
		return FarmerBot{}, err
	}

	config, err := parser.ParseIntoConfig(jsonContent, format)
	if err != nil {
		return FarmerBot{}, err
	}

	substrateManager := substrate.NewManager(constants.SubstrateURLs[network]...)
	sub, err := substrateManager.Substrate()
	if err != nil {
		return FarmerBot{}, fmt.Errorf("error: %w, getting substrate connection using %s", err, constants.SubstrateURLs[network])
	}

	// TODO:
	// defer sub.Close()

	identity, err := substrate.NewIdentityFromSr25519Phrase(mnemonic)
	if err != nil {
		return FarmerBot{}, err
	}

	twinID, err := sub.GetTwinByPubKey(identity.PublicKey())
	if err != nil {
		return FarmerBot{}, err
	}

	powerManager := manager.NewPowerManager(identity, sub, config)
	if err != nil {
		return FarmerBot{}, err
	}

	nodeManager := manager.NewNodeManager(identity, sub, config)
	if err != nil {
		return FarmerBot{}, err
	}

	rmbClient, err := peer.NewRpcClient(ctx, peer.KeyTypeSr25519, mnemonic, constants.RelayURLS[network], fmt.Sprintf("farmerbot-rpc-%d", config.Farm.ID), sub, true)
	if err != nil {
		return FarmerBot{}, errors.Wrap(err, "could not create rmb client")
	}

	dataManager := manager.NewDataManager(identity, sub, config, rmbClient)
	if err != nil {
		return FarmerBot{}, err
	}

	return FarmerBot{
		config:       config,
		sub:          *sub,
		rmbClient:    rmbClient,
		powerManager: powerManager,
		nodeManager:  nodeManager,
		dataManager:  dataManager,
		twinID:       twinID,
		mnemonic:     mnemonic,
		network:      network,
		identity:     identity,
	}, nil
}

func (f *FarmerBot) start(ctx context.Context) error {
	return f.powerManager.PowerOnAllNodes()
}

func (f *FarmerBot) stop(ctx context.Context) error {
	return f.powerManager.PowerOnAllNodes()
}

func (f *FarmerBot) update(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	default:
		startTime := time.Now()

		// data update
		log.Debug().Msg("[FARMERBOT] data update")
		err := f.dataManager.Update(ctx)
		if err != nil {
			log.Error().Err(err).Msgf("[FARMERBOT] failed to update")
		}

		// periodic wakeup
		log.Debug().Msg("[FARMERBOT] periodic wake up")
		err = f.powerManager.PeriodicWakeUp()
		if err != nil {
			log.Error().Err(err).Msgf("[FARMERBOT] failed to perform periodic wake up")
		}

		// power management
		log.Debug().Msg("[FARMERBOT] power management")
		err = f.powerManager.PowerManagement()
		if err != nil {
			log.Error().Err(err).Msgf("[FARMERBOT] failed to power management nodes")
		}

		delta := time.Since(startTime)
		log.Debug().Msgf("[FARMERBOT] Elapsed time for update: %v minutes", delta.Minutes())

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

// Run runs farmerbot to update nodes and power management
func (f *FarmerBot) Run(ctx context.Context) {
	go f.update(ctx)

	// run the server
	go f.serve(ctx)

	log.Info().Msg("[FARMERBOT] up and running...")
	f.start(ctx)

	// graceful shutdown
	log.Info().Msg("[FARMERBOT] stopped serving new requests")
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownRelease()

	if err := f.stop(shutdownCtx); err != nil {
		log.Fatal().Err(err).Msg("[FARMERBOT] shutdown error")
	}
	log.Info().Msg("[FARMERBOT] graceful shutdown successful")
}

func (f *FarmerBot) serve(ctx context.Context) error {
	router := peer.NewRouter()
	farmerbot := router.SubRoute("farmerbot")

	farmRouter := farmerbot.SubRoute("farmmanager")
	nodeRouter := farmerbot.SubRoute("nodemanager")
	powerRouter := farmerbot.SubRoute("powermanager")

	powerRouter.Use(f.authorize)

	farmRouter.WithHandler("version", func(ctx context.Context, payload []byte) (interface{}, error) {
		return version.Version, nil
	})

	nodeRouter.WithHandler("findnode", func(ctx context.Context, payload []byte) (interface{}, error) {
		var options models.NodeOptions

		if err := json.Unmarshal(payload, &options); err != nil {
			return nil, fmt.Errorf("failed to load request payload: %w", err)
		}

		nodeID, err := f.nodeManager.FindNode(options)
		return nodeID, err
	})

	powerRouter.WithHandler("poweroff", func(ctx context.Context, payload []byte) (interface{}, error) {
		var nodeID uint32

		if err := validateAccountEnoughBalance(f.identity, f.sub); err != nil {
			return nil, fmt.Errorf("failed to validate account balance: %w", err)
		}

		if err := json.Unmarshal(payload, &nodeID); err != nil {
			return nil, fmt.Errorf("failed to load request payload: %w", err)
		}

		err := f.powerManager.PowerOff(nodeID)
		return nil, err
	})

	powerRouter.WithHandler("poweron", func(ctx context.Context, payload []byte) (interface{}, error) {
		var nodeID uint32

		if err := validateAccountEnoughBalance(f.identity, f.sub); err != nil {
			return nil, fmt.Errorf("failed to validate account balance: %w", err)
		}

		if err := json.Unmarshal(payload, &nodeID); err != nil {
			return nil, fmt.Errorf("failed to load request payload: %w", err)
		}

		err := f.powerManager.PowerOn(nodeID)
		return nil, err
	})

	_, err := peer.NewPeer(
		ctx,
		peer.KeyTypeSr25519,
		f.mnemonic,
		constants.RelayURLS[f.network],
		fmt.Sprintf("farmerbot-%d", f.config.Farm.ID),
		&f.sub,
		false,
		router.Serve,
	)

	if err != nil {
		return fmt.Errorf("failed to create farmerbot direct peer: %w", err)
	}

	select {}
}

func (f *FarmerBot) authorize(ctx context.Context, payload []byte) (context.Context, error) {
	twinID := peer.GetTwinID(ctx)
	if twinID != f.twinID {
		return ctx, fmt.Errorf("you are not authorized for this action")
	}
	return ctx, nil
}

func validateAccountEnoughBalance(identity substrate.Identity, sub substrate.Substrate) error {
	accountAddress, err := substrate.FromAddress(identity.Address())
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
