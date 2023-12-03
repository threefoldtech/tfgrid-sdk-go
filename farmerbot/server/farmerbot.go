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
	"golang.org/x/sync/errgroup"
)

// FarmerBot for managing farms
type FarmerBot struct {
	config           *models.Config
	substrateManager substrate.Manager
	powerManager     *manager.PowerManager
	dataManager      manager.DataManager
	network          string
	mnemonic         string
	identity         substrate.Identity
}

// NewFarmerBot generates a new farmer bot
func NewFarmerBot(ctx context.Context, inputs models.InputConfig, network, mnemonic string) (FarmerBot, error) {
	farmerbot := FarmerBot{mnemonic: mnemonic, network: network}

	substrateManager := substrate.NewManager(constants.SubstrateURLs[network]...)
	farmerbot.substrateManager = substrateManager

	err := farmerbot.SetConfig(inputs)
	if err != nil {
		return FarmerBot{}, err
	}

	// TODO: check farm hex seed if available
	identity, err := substrate.NewIdentityFromSr25519Phrase(mnemonic)
	if err != nil {
		return FarmerBot{}, err
	}

	farmerbot.identity = identity

	powerManager := manager.NewPowerManager(identity, substrateManager, farmerbot.config)
	farmerbot.powerManager = &powerManager

	con, err := substrateManager.Substrate()
	if err != nil {
		return FarmerBot{}, err
	}
	// TODO:
	// defer con.Close()

	rmb, err := peer.NewRpcClient(ctx, peer.KeyTypeSr25519, mnemonic, constants.RelayURLS[network], fmt.Sprintf("farmerbot-rpc-%d", farmerbot.config.Farm.ID), con, true)
	if err != nil {
		return FarmerBot{}, errors.Wrap(err, "could not create rmb client")
	}

	rmbNodeClient := manager.NewRmbNodeClient(rmb)
	farmerbot.dataManager = manager.NewDataManager(substrateManager, farmerbot.config, &rmbNodeClient)

	return farmerbot, nil
}

// Run runs farmerbot to update nodes and power management
func (f *FarmerBot) Run(ctx context.Context) error {
	if err := f.start(ctx); err != nil {
		return err
	}

	go f.update(ctx)

	errs, ctx := errgroup.WithContext(ctx)
	errs.Go(func() error {
		return f.serve(ctx)
	})

	if err := errs.Wait(); err != nil {
		return err
	}

	log.Info().Msg("up and running...")

	// graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownRelease()

	if err := f.stop(shutdownCtx); err != nil {
		return err
	}

	log.Info().Msg("graceful shutdown successful")
	return nil
}

func (f *FarmerBot) SetConfig(inputs models.InputConfig) error {
	if f.config == nil {
		f.config = &models.Config{}
	}

	return f.config.Set(f.substrateManager, inputs)
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

		log.Debug().Msg("data update")
		err := f.dataManager.Update(ctx)
		if err != nil {
			log.Error().Err(err).Msg("failed to update")
		}

		log.Debug().Msg("periodic wake up")
		err = f.powerManager.PeriodicWakeUp()
		if err != nil {
			log.Error().Err(err).Msg("failed to perform periodic wake up")
		}

		log.Debug().Msg("power management")
		err = f.powerManager.PowerManagement()
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

func (f *FarmerBot) serve(ctx context.Context) error {
	con, err := f.substrateManager.Substrate()
	if err != nil {
		return err
	}
	defer con.Close()

	router := peer.NewRouter()
	farmerbot := router.SubRoute("farmerbot")

	farmRouter := farmerbot.SubRoute("farmmanager")
	nodeRouter := farmerbot.SubRoute("nodemanager")
	powerRouter := farmerbot.SubRoute("powermanager")

	// TODO: didn't work
	// powerRouter.Use(f.authorize)

	farmerTwinID, err := con.GetTwinByPubKey(f.identity.PublicKey())
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

		nodeID, err := f.powerManager.FindNode(options)
		return nodeID, err
	})

	powerRouter.WithHandler("poweroff", func(ctx context.Context, payload []byte) (interface{}, error) {
		err := authorize(ctx, farmerTwinID)
		if err != nil {
			return nil, err
		}

		var nodeID uint32
		if err := f.validateAccountEnoughBalance(); err != nil {
			return nil, fmt.Errorf("failed to validate account balance: %w", err)
		}

		if err := json.Unmarshal(payload, &nodeID); err != nil {
			return nil, fmt.Errorf("failed to load request payload: %w", err)
		}

		return nil, f.powerManager.PowerOff(nodeID)
	})

	powerRouter.WithHandler("poweron", func(ctx context.Context, payload []byte) (interface{}, error) {
		err := authorize(ctx, farmerTwinID)
		if err != nil {
			return nil, err
		}

		var nodeID uint32
		if err := f.validateAccountEnoughBalance(); err != nil {
			return nil, fmt.Errorf("failed to validate account balance: %w", err)
		}

		if err := json.Unmarshal(payload, &nodeID); err != nil {
			return nil, fmt.Errorf("failed to load request payload: %w", err)
		}

		err = f.powerManager.PowerOn(nodeID)
		return nil, err
	})

	_, err = peer.NewPeer(
		ctx,
		peer.KeyTypeSr25519,
		f.mnemonic,
		constants.RelayURLS[f.network],
		fmt.Sprintf("farmerbot-%d", f.config.Farm.ID),
		con,
		false,
		router.Serve,
	)

	if err != nil {
		return errors.New("failed to create farmerbot direct peer")
	}

	select {}
}

func (f *FarmerBot) validateAccountEnoughBalance() error {
	con, err := f.substrateManager.Substrate()
	if err != nil {
		return err
	}
	defer con.Close()

	accountAddress, err := substrate.FromAddress(f.identity.Address())
	if err != nil {
		return err
	}

	balance, err := con.GetBalance(accountAddress)
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
