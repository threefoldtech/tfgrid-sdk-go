package main

import (
	"context"
	"fmt"
	"os"
	"time"

	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
)

func main() {
	// TODO: Replace these with your actual values
	mnemonics := "<mnemonics goes here>"
	nodeTwinID := uint32(21) // Replace with actual node twin ID
	subManager := substrate.NewManager("wss://tfchain.dev.grid.tf/ws")
	
	// Initialize RMB client
	client, err := peer.NewRpcClient(
		context.Background(),
		mnemonics,
		subManager,
		peer.WithKeyType(peer.KeyTypeSr25519),
		peer.WithRelay("wss://relay.dev.grid.tf"),
		peer.WithSession("test-client"),
	)
	if err != nil {
		fmt.Printf("Failed to create RMB client: %v\n", err)
		os.Exit(1)
	}

	// Define context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute) // Benchmarks might take a while
	defer cancel()

	// Perform CPU benchmark
	fmt.Printf("Starting CPU benchmark for node twin %d...\n", nodeTwinID)
	payload := struct {
		Name string
	}{
		Name: "cpu-benchmark",
	}
	const cmd = "zos.perf.get"
	var result interface{}
	
	err = client.Call(ctx, nodeTwinID, cmd, payload, &result)
	if err != nil {
		fmt.Printf("Failed to execute CPU benchmark: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("CPU Benchmark Result for node twin %d:\n%+v\n", nodeTwinID, result)
}
