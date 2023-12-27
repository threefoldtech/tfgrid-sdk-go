package main

import (
	"context"
	"fmt"
	"time"

	"log"

	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
)

func version() error {
	mnemonics := "<Enter MNEMONIC here>"
	subManager := substrate.NewManager("wss://tfchain.dev.grid.tf/ws")

	client, err := peer.NewRpcClient(
		context.Background(),
		peer.KeyTypeSr25519,
		mnemonics,
		"wss://relay.dev.grid.tf",
		"test-version",
		subManager,
		true,
	)
	if err != nil {
		return fmt.Errorf("failed to create rpc client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	farmID := 53 // <- replace this with the farm id of the farmerbot
	service := fmt.Sprintf("farmerbot-%d", farmID)
	const farmerbotTwinID = 164 // <- replace this with the twin id of where the farmerbot is running

	var nodeID uint32
	var version string
	if err := client.CallWithSession(ctx, farmerbotTwinID, &service, "farmerbot.farmmanager.version", nodeID, &version); err != nil {
		return err
	}

	fmt.Printf("version: %v\n", version)

	return nil
}

func main() {
	if err := version(); err != nil {
		log.Fatal(err)
	}
}
