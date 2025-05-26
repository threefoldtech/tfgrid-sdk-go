package main

import (
	"context"
	"fmt"
	"os"
	"time"

	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/messenger"
)

const (
	chainUrl = "ws://192.168.1.125:9944"

	// destination is mycelium pk or ip
	destination = "22b45ca2c6c40650fa4c739942a7c863deeb4a88a6a2cb38b8c9b273f4ad7b0c"
)

func main() {
	mnemonic := os.Getenv("MNEMONIC")

	man := substrate.NewManager(chainUrl)

	msgr, err := messenger.NewMessenger(
		"",
		60,
		man,
		messenger.WithMnemonicPhrase(mnemonic),
	)
	if err != nil {
		fmt.Printf("Failed to create Mycelium client: %v\n", err)
		os.Exit(1)
	}
	defer msgr.Close()

	rpcClient := messenger.NewJSONRPCClient(msgr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var addResult float64
	// should timeout if no response
	err = rpcClient.Call(ctx, destination, "calculator.add", []float64{10, 20}, &addResult)
	if err != nil {
		fmt.Printf("Failed to call calculator.add: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("10 + 20 = %f\n", addResult)
}
