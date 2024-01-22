package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"log"

	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/internal"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
)

func nodesReport() error {
	mnemonics := "<mnemonics goes here>"
	subManager := substrate.NewManager("wss://tfchain.dev.grid.tf/ws")

	client, err := peer.NewRpcClient(
		context.Background(),
		peer.KeyTypeSr25519,
		mnemonics,
		"wss://relay.dev.grid.tf",
		"test-report",
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
	var nodesReport []internal.NodeReport
	if err := client.CallWithSession(ctx, farmerbotTwinID, &service, "farmerbot.farmmanager.report", nodeID, &nodesReport); err != nil {
		return err
	}

	report, err := json.MarshalIndent(nodesReport, "", "  ")
	if err != nil {
		return err
	}

	fmt.Print(string(report))

	return nil
}

func main() {
	if err := nodesReport(); err != nil {
		log.Fatal(err)
	}
}
