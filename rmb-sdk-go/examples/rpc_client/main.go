package main

import (
	"context"
	"fmt"
	"log"
	"time"

	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
)

type version struct {
	ZOS   string `json:"zos"`
	ZInit string `json:"zinit"`
}

func app() error {
	mnemonics := "<mnemonics goes here>"
	subNodeURL := "wss://tfchain.dev.grid.tf/ws"
	relayURL := "wss://relay.dev.grid.tf"

	subManager := substrate.NewManager(subNodeURL)

	client, err := peer.NewRpcClient(context.Background(), peer.KeyTypeSr25519, mnemonics, relayURL, "test-client", subManager, true)
	if err != nil {
		return fmt.Errorf("failed to create direct client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	const dstTwin uint32 = 11 // <- replace this with any node i
	var ver version
	if err := client.Call(ctx, dstTwin, "zos.system.version", nil, &ver); err != nil {
		return err
	}

	fmt.Printf("output: %s\n", ver)
	return nil
}

func main() {
	if err := app(); err != nil {
		log.Fatal(err)
	}
}
