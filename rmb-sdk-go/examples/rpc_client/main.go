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

	sub, err := substrate.NewManager(subNodeURL).Substrate()
	if err != nil {
		return fmt.Errorf("failed to connect to substrate: %w", err)
	}
	defer sub.Close()

	client, err := peer.NewRpcClient(context.Background(), peer.KeyTypeSr25519, mnemonics, relayURL, "test-client", sub, true)
	if err != nil {
		return fmt.Errorf("failed to create direct client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	const dstNode = 11 // <- replace this with any node id
	node, err := sub.GetNode(dstNode)
	if err != nil {
		return err
	}

	var ver version
	if err := client.Call(ctx, uint32(node.TwinID), "zos.system.version", nil, &ver); err != nil {
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
