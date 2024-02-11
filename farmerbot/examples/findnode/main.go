package main

import (
	"context"
	"fmt"
	"time"

	"log"

	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/internal"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
)

func findNode() (uint32, error) {
	mnemonics := "<mnemonics goes here>"
	subManager := substrate.NewManager("wss://tfchain.dev.grid.tf/ws")

	client, err := peer.NewRpcClient(
		context.Background(),
		mnemonics,
		subManager,
		peer.WithRelay("wss://relay.dev.grid.tf"),
		peer.WithSession("test-find-node"),
	)

	if err != nil {
		return 0, fmt.Errorf("failed to create rpc client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	farmID := 53 // <- replace this with the farm id of the farmerbot
	service := fmt.Sprintf("farmerbot-%d", farmID)
	const farmerbotTwinID = 164 // <- replace this with the twin id of where the farmerbot is running

	options := internal.NodeFilterOption{
		NodesExcluded: []uint32{},
		NumGPU:        0,
		GPUVendors:    []string{},
		GPUDevices:    []string{},
		Certified:     false,
		Dedicated:     false,
		PublicConfig:  false,
		PublicIPs:     0,
		HRU:           0,
		SRU:           0,
		CRU:           0,
		MRU:           0,
	}
	var output uint32
	if err := client.CallWithSession(ctx, farmerbotTwinID, &service, "farmerbot.nodemanager.findnode", options, &output); err != nil {
		return 0, err
	}

	return output, nil
}

func main() {
	nodeID, err := findNode()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("nodeID: %v\n", nodeID)
}
