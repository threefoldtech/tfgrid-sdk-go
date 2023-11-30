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
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
)

// FarmerBot for managing farms
type FarmerBot struct {
	configFile   string
	config       *models.Config
	sub          substrate.Substrate
	powerManager *manager.PowerManager
	dataManager  manager.DataManager
	twinID       uint32
	network      string
	mnemonic     string
	identity     substrate.Identity
}

// NewFarmerBot generates a new farmer bot
func NewFarmerBot(ctx context.Context, configFile, network, mnemonic string) (FarmerBot, error) {
	farmerbot := FarmerBot{mnemonic: mnemonic, network: network, configFile: configFile}

	substrateManager := substrate.NewManager(constants.SubstrateURLs[network]...)
	sub, err := substrateManager.Substrate()
	if err != nil {
		return FarmerBot{}, err
	}

	farmerbot.sub = *sub

	err = farmerbot.setConfig()
	if err != nil {
		return FarmerBot{}, err
	}

	// TODO: check farm hex seed if available
	identity, err := substrate.NewIdentityFromSr25519Phrase(mnemonic)
	if err != nil {
		return FarmerBot{}, err
	}

	farmerbot.identity = identity

	twinID, err := sub.GetTwinByPubKey(identity.PublicKey())
	if err != nil {
		return FarmerBot{}, err
	}

	farmerbot.twinID = twinID

	powerManager := manager.NewPowerManager(identity, sub, farmerbot.config)
	if err != nil {
		return FarmerBot{}, err
	}

	farmerbot.powerManager = &powerManager

	rmb, err := peer.NewRpcClient(ctx, peer.KeyTypeSr25519, mnemonic, constants.RelayURLS[network], fmt.Sprintf("farmerbot-rpc-%d", farmerbot.config.Farm.ID), sub, true)
	if err != nil {
		return FarmerBot{}, errors.Wrap(err, "could not create rmb client")
	}

	rmbNodeClient := manager.NewRmbNodeClient(rmb)
	dataManager := manager.NewDataManager(sub, farmerbot.config, &rmbNodeClient)
	if err != nil {
		return FarmerBot{}, err
	}

	farmerbot.dataManager = dataManager

	return farmerbot, nil
}

// Run runs farmerbot to update nodes and power management
func (f *FarmerBot) Run(ctx context.Context) {
	if err := f.start(ctx); err != nil {
		log.Fatal().Err(err).Msg("[FARMERBOT] error starting")
	}

	go f.update(ctx)
	go f.serve(ctx)

	log.Info().Msg("[FARMERBOT] up and running...")

	// graceful shutdown
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

func (f *FarmerBot) setConfig() error {
	content, format, err := parser.ReadFile(f.configFile)
	if err != nil {
		return err
	}

	inputs, err := parser.ParseIntoInputConfig(content, format)
	if err != nil {
		return err
	}

	if f.config == nil {
		f.config = &models.Config{}
	}

	err = f.config.Set(&f.sub, inputs)
	if err != nil {
		return err
	}

	return nil
}

func (f *FarmerBot) start(ctx context.Context) error {
	return f.powerManager.PowerOnAllNodes()
}

func (f *FarmerBot) stop(ctx context.Context) error {
	return f.powerManager.PowerOnAllNodes()
}

func (f *FarmerBot) update(ctx context.Context) {
	for {
		startTime := time.Now()

		log.Debug().Msg("[FARMERBOT] update configurations")
		err := f.setConfig()
		if err != nil {
			log.Error().Err(err).Str("config file", f.configFile).Msg("[FARMERBOT] failed to update configurations")
		}

		log.Debug().Msg("[FARMERBOT] data update")
		err = f.dataManager.Update(ctx)
		if err != nil {
			log.Error().Err(err).Msg("[FARMERBOT] failed to update")
		}

		log.Debug().Msg("[FARMERBOT] periodic wake up")
		err = f.powerManager.PeriodicWakeUp()
		if err != nil {
			log.Error().Err(err).Msg("[FARMERBOT] failed to perform periodic wake up")
		}

		log.Debug().Msg("[FARMERBOT] power management")
		err = f.powerManager.PowerManagement()
		if err != nil {
			log.Error().Err(err).Msg("[FARMERBOT] failed to power management nodes")
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

func (f *FarmerBot) serve(ctx context.Context) {
	router := peer.NewRouter()
	farmerbot := router.SubRoute("farmerbot")

	farmRouter := farmerbot.SubRoute("farmmanager")
	nodeRouter := farmerbot.SubRoute("nodemanager")
	powerRouter := farmerbot.SubRoute("powermanager")

	// TODO: didn't work
	// powerRouter.Use(f.authorize)

	farmRouter.WithHandler("version", func(ctx context.Context, payload []byte) (interface{}, error) {
		return version.Version, nil
	})

	nodeRouter.WithHandler("findnode", func(ctx context.Context, payload []byte) (interface{}, error) {
		var options models.NodeOptions

		if err := json.Unmarshal(payload, &options); err != nil {
			return nil, fmt.Errorf("failed to load request payload: %w", err)
		}

		nodeID, err := f.powerManager.FindNode(options)
		return nodeID, err
	})

	powerRouter.WithHandler("poweroff", func(ctx context.Context, payload []byte) (interface{}, error) {
		err := f.authorize(ctx)
		if err != nil {
			return nil, err
		}

		var nodeID uint32
		if err := validateAccountEnoughBalance(f.identity, f.sub); err != nil {
			return nil, fmt.Errorf("failed to validate account balance: %w", err)
		}

		if err := json.Unmarshal(payload, &nodeID); err != nil {
			return nil, fmt.Errorf("failed to load request payload: %w", err)
		}

		return nil, f.powerManager.PowerOff(nodeID)
	})

	powerRouter.WithHandler("poweron", func(ctx context.Context, payload []byte) (interface{}, error) {
		err := f.authorize(ctx)
		if err != nil {
			return nil, err
		}

		var nodeID uint32
		if err := validateAccountEnoughBalance(f.identity, f.sub); err != nil {
			return nil, fmt.Errorf("failed to validate account balance: %w", err)
		}

		if err := json.Unmarshal(payload, &nodeID); err != nil {
			return nil, fmt.Errorf("failed to load request payload: %w", err)
		}

		err = f.powerManager.PowerOn(nodeID)
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
		log.Fatal().Err(err).Msg("[FARMERBOT] failed to create farmerbot direct peer")
	}

	select {}
}

func (f *FarmerBot) authorize(ctx context.Context) error {
	twinID := peer.GetTwinID(ctx)
	if twinID != f.twinID {
		return fmt.Errorf("you are not authorized for this action. your twin id is `%d`, only the farm owner with twin id `%d` is authorized", twinID, f.twinID)
	}
	return nil
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
