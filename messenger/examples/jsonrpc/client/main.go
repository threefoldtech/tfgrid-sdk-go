package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/messenger"
)

const (
	chainUrl = "wss://tfchain.dev.grid.tf"
)

func main() {
	mnemonic := os.Getenv("MNEMONIC")

	var dest string
	flag.StringVar(&dest, "dest", "", "destination public key or IP address")
	flag.Parse()

	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "15:04"}).With().Timestamp().Logger()

	manager := substrate.NewManager(chainUrl)
	sub, err := manager.Substrate()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to connect to TFChain - will send unsigned messages")
		sub = nil
	}

	msgr, err := messenger.NewMessenger(messenger.WithChain(sub))
	if err != nil {
		fmt.Printf("Failed to create messenger client: %v\n", err)
		os.Exit(1)
	}
	defer msgr.Close()

	rpcClient := messenger.NewJSONRPCClient(msgr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// TODO: should send identity to call and it will sign the message
	id, err := substrate.NewIdentityFromEd25519Phrase(mnemonic)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create identity from mnemonic")
		os.Exit(1)
	}

	var result float64
	err = rpcClient.Call(ctx, dest, "calculator.add", []float64{10, 20}, &result)
	if err != nil {
		log.Error().Err(err).Msg("Failed to call calculator.add")
		os.Exit(1)
	}

	fmt.Println(result)
}
