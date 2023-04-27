package main

import (
	"context"
	"fmt"
	"os"
	"time"

	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/direct"
)

func app() error {
	mnemonics := "<mnemonics goes here>"
	subManager := substrate.NewManager("wss://tfchain.dev.grid.tf/ws")
	sub, err := subManager.Substrate()
	if err != nil {
		return fmt.Errorf("failed to connect to substrate: %w", err)
	}

	defer sub.Close()
	client, err := direct.NewClient(context.Background(), direct.KeyTypeSr25519, mnemonics, "wss://relay.dev.grid.tf", "test-client", sub, true)
	if err != nil {
		return fmt.Errorf("failed to create direct client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := client.Ping(ctx); err != nil {
		return fmt.Errorf("failed to do high level ping: %s", err)
	}
	const dst = 7 // <- replace this with the twin id of where the service is running
	// it's okay to run both the server and the client behind the same rmb-peer
	var output float64
	for i := 0; i < 20; i++ {
		if err := client.Call(ctx, dst, "calculator.add", []float64{output, float64(i)}, &output); err != nil {
			return err
		}
	}

	fmt.Printf("output: %f\n", output)

	return nil
}

func main() {
	if err := app(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
