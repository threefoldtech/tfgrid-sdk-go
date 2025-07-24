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

// API
type Calculator struct{}

func (c *Calculator) Add(a, b float64) float64 {
	return a + b
}

// HANDLERS
func addHandler(ctx context.Context, calc *Calculator, params json.RawMessage) (interface{}, error) {
	var args []float64
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if len(args) != 2 {
		return nil, fmt.Errorf("expected 2 parameters, got %d", len(args))
	}

	result := calc.Add(args[0], args[1])
	return result, nil
}

const (
	chainUrl = "ws://192.168.1.10:9944"
)

func main() {
	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "15:04"}).With().Timestamp().Logger()
	mnemonic := os.Getenv("MNEMONIC")

	manager := substrate.NewManager(chainUrl)

	msgr, err := messenger.NewMessenger(
		messenger.WithSubstrateManager(manager),
		messenger.WithMnemonic(mnemonic),
		messenger.WithEnableTwinIdentity(true),
	)

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

	fmt.Println("Server started. Press Ctrl+C to stop.")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	server.Stop()
	fmt.Println("Server stopped.")
}
