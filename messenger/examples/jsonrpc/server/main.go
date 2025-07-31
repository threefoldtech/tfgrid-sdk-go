package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/messenger"
)

type Calculator struct{}

func (c *Calculator) Add(a, b float64) float64 {
	return a + b
}

func addHandler(ctx context.Context, calc *Calculator, params json.RawMessage) (interface{}, error) {
	var args []float64
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if len(args) != 2 {
		return nil, fmt.Errorf("expected 2 parameters, got %d", len(args))
	}

	twinID, ok := ctx.Value(messenger.TwinIDContextKey).(uint32)
	if !ok {
		log.Warn().Msg("can't find twin id")
		return nil, fmt.Errorf("can't find twin id")
	}

	log.Info().Uint32("twin_id", twinID).Msg("verified request from twin")
	result := calc.Add(args[0], args[1])
	return result, nil
}

const (
	chainUrl = "wss://tfchain.dev.grid.tf"
)

func main() {
	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "15:04"}).With().Timestamp().Logger()

	manager := substrate.NewManager(chainUrl)
	sub, err := manager.Substrate()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to connect to TFChain - operating without signature verification")
		sub = nil
	}

	msgr, err := messenger.NewMessenger(messenger.WithChain(sub))
	if err != nil {
		fmt.Printf("Failed to create messenger: %v\n", err)
		os.Exit(1)
	}
	defer msgr.Close()

	server := messenger.NewJSONRPCServer(msgr)
	calc := &Calculator{}

	server.RegisterHandler("calculator.add", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		return addHandler(ctx, calc, params)
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := server.Start(ctx); err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
		os.Exit(1)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	server.Stop()
	fmt.Println("Server stopped.")
}
