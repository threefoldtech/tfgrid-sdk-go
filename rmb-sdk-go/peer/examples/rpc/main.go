package main

import (
	"context"
	"fmt"
	"time"

	"log"

	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
)

func app() error {
	mnemonics := "<mnemonics goes here>"
	subManager := substrate.NewManager("wss://tfchain.dev.grid.tf/ws")
	sub, err := subManager.Substrate()
	if err != nil {
		return fmt.Errorf("failed to connect to substrate: %w", err)
	}

	defer sub.Close()
	client, err := peer.NewRpcClient(
		context.Background(),
		peer.KeyTypeSr25519,
		mnemonics,
		"wss://relay.dev.grid.tf",
		"test-client",
		sub,
		true,
	)
	if err != nil {
		return fmt.Errorf("failed to create direct client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	const dst = 7 // <- replace this with the twin id of where the service is running
	// it's okay to run both the server and the client behind the same rmb-peer
	var output float64
	for i := 0; i < 20; i++ {
		var session *string
		// uncomment it  you are using peer router example
		// routerSession := "test-router"
		// session = &routerSession

		if err := client.Call(ctx, dst, session, "calculator.add", []float64{output, float64(i)}, &output); err != nil {
			return err
		}
	}

	fmt.Printf("output: %f\n", output)

	return nil
}

func main() {
	if err := app(); err != nil {
		log.Fatal(err)
	}
}
