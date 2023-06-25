package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/async"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/common"
)

// Change result to the expected response
type Result struct {
	ID string `json:"id"`
}

func listener(res []byte) error {
	var output Result
	json.Unmarshal(res, &output)
	fmt.Printf("%+v", output)
	return nil
}

func app() error {
	mnemonics := "<mnemonics goes here>"
	subManager := substrate.NewManager("wss://tfchain.dev.grid.tf/ws")
	sub, err := subManager.Substrate()
	if err != nil {
		return fmt.Errorf("failed to connect to substrate: %w", err)
	}

	defer sub.Close()
	client, err := async.NewAsyncClient(context.Background(), listener, common.KeyTypeSr25519, mnemonics, "wss://relay.dev.grid.tf", "test-client", sub, true)
	if err != nil {
		return fmt.Errorf("failed to create direct client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// if err := client.Ping(ctx); err != nil {
	// 	return fmt.Errorf("failed to do high level ping: %s", err)
	// }
	const dst = 7 // <- replace this with the twin id of where the service is running
	// it's okay to run both the server and the client behind the same rmb-peer
	if err := client.Send(ctx, dst, "calculator.add", nil); err != nil {
		return err
	}

	time.Sleep(3 * time.Second)

	return nil
}

func main() {
	if err := app(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
