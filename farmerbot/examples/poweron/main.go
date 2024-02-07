package main

import (
	"context"
	"fmt"
	"time"

	"log"

	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
)

func powerOn() error {
	mnemonics := "<mnemonics goes here>"
	subManager := substrate.NewManager("wss://tfchain.dev.grid.tf/ws")

	client, err := peer.NewRpcClient(
		context.Background(),
		mnemonics,
		subManager,
		peer.WithRelay("wss://relay.dev.grid.tf"),
		peer.WithSession("test-power-on"),
	)

	if err != nil {
		return fmt.Errorf("failed to create rpc client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	farmID := 53 // <- replace this with the farm id of the farmerbot
	service := fmt.Sprintf("farmerbot-%d", farmID)
	const farmerbotTwinID = 164 // <- replace this with the twin id of where the farmerbot is running

	nodeID := uint32(83)
	if err := client.CallWithSession(ctx, farmerbotTwinID, &service, "farmerbot.powermanager.poweron", nodeID, nil); err != nil {
		return err
	}

	return nil
}

func main() {
	if err := powerOn(); err != nil {
		log.Fatal(err)
	}
}
