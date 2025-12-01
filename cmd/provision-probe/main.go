package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/provision-probe/pkg/app"
	"github.com/threefoldtech/provision-probe/pkg/config"
	"github.com/threefoldtech/provision-probe/pkg/logger"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger.Init(cfg.LogLevel)

	application, err := app.New(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize app")
	}
	defer application.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Info().Msg("Shutting down...")
		cancel()
	}()

	if err := application.Run(ctx); err != nil {
		log.Fatal().Err(err).Msg("Service failed")
	}
}
