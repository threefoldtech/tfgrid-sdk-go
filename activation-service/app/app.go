// Package app for c4s backend app
package app

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/activation-service/config"
	"github.com/threefoldtech/tfgrid-sdk-go/activation-service/middlewares"
)

// App for all dependencies of backend server
type App struct {
	config        config.Configuration
	substrateConn *substrate.Substrate
	identity      substrate.Identity
}

// NewApp creates new server app all configurations
func NewApp(ctx context.Context, configFile string) (app *App, err error) {
	config, err := config.ReadConfFile(configFile)
	if err != nil {
		return
	}

	manager := substrate.NewManager(config.SubstrateURL)
	if err != nil {
		return
	}

	sub, err := manager.Substrate()
	if err != nil {
		return
	}

	identity, err := substrate.NewIdentityFromSr25519Phrase(config.Mnemonic)
	if err != nil {
		return
	}

	return &App{
		config:        config,
		substrateConn: sub,
		identity:      identity,
	}, nil
}

// Start starts the app
func (a *App) Start(ctx context.Context) (err error) {
	a.registerHandlers()
	log.Info().Msg("Server is listening on port 3000")

	srv := &http.Server{
		Addr: ":3000",
	}

	go func() {
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("HTTP server error")
		}
		log.Info().Msg("Stopped serving new connections")
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownRelease()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal().Err(err).Msg("HTTP shutdown error")
	}
	log.Info().Msg("Graceful shutdown complete")

	return nil
}

func (a *App) registerHandlers() {
	r := mux.NewRouter()

	r.HandleFunc("/activation/activate", WrapFunc(a.activateHandler)).Methods("POST", "OPTIONS")

	// middlewares
	r.Use(middlewares.EnableCors)
	http.Handle("/", r)
}
