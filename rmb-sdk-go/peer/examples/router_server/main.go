package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
)

func app() error {
	router := peer.NewRouter()
	app := router.SubRoute("calculator")

	// this function then becomes calculator.add
	app.WithHandler("add", func(ctx context.Context, payload []byte) (interface{}, error) {
		env := peer.GetEnvelope(ctx)
		envErr := env.GetError()
		if envErr != nil {
			return []byte{}, errors.New(envErr.Message)
		}

		if env.Schema == nil || *env.Schema != rmb.DefaultSchema {
			return []byte{}, fmt.Errorf("invalid schema received expected '%s'", rmb.DefaultSchema)
		}

		var numbers []float64

		if err := json.Unmarshal(payload, &numbers); err != nil {
			return nil, fmt.Errorf("failed to load request payload was expecting list of float: %w", err)
		}

		var result float64
		for _, v := range numbers {
			result += v
		}

		return result, nil
	})

	// this function then becomes calculator.sub
	app.WithHandler("sub", func(ctx context.Context, payload []byte) (interface{}, error) {
		var numbers []float64

		if err := json.Unmarshal(payload, &numbers); err != nil {
			return nil, fmt.Errorf("failed to load request payload was expecting list of float: %w", err)
		}

		var result float64
		for _, v := range numbers {
			result -= v
		}

		return result, nil
	})

	// adding a peer for the router
	mnemonics := "<mnemonics goes here>"
	subManager := substrate.NewManager("wss://tfchain.dev.grid.tf/ws")
	sub, err := subManager.Substrate()
	ctx := context.Background()
	if err != nil {
		return fmt.Errorf("failed to connect to substrate: %w", err)
	}

	defer sub.Close()

	// this peer will be a 'calculator' session.
	// means other peers on the network need to know that
	// session id to use when they are making calls
	_, err = peer.NewPeer(
		ctx,
		mnemonics,
		sub,
		router.Serve,
		peer.WithRelay("wss://relay.dev.grid.tf"),
		peer.WithSession("calculator"),
		peer.WithEncryption(true),
	)

	if err != nil {
		return fmt.Errorf("failed to create direct client: %w", err)
	}

	select {}
}

func main() {
	if err := app(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
