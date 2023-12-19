package main

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer/types"
)

var resultsChan = make(chan bool)

func app() error {
	mnemonics := "<mnemonics goes here>"
	subManager := substrate.NewManager("wss://tfchain.dev.grid.tf/ws")
	sub, err := subManager.Substrate()
	ctx := context.Background()
	if err != nil {
		return fmt.Errorf("failed to connect to substrate: %w", err)
	}

	defer sub.Close()

	peer, err := peer.NewPeer(
		ctx,
		mnemonics,
		sub,
		relayCallback,
		peer.WithRelay("wss://relay.dev.grid.tf"),
		peer.WithSession("test-client"),
	)

	if err != nil {
		return fmt.Errorf("failed to create direct client: %w", err)
	}

	const dst = 7 // <- replace this with the twin id of where the service is running

	for i := 0; i < 20; i++ {
		data := []float64{rand.Float64(), rand.Float64()}
		var session *string
		// uncomment if you are using peer router example
		// routerSession := "test-router"
		// session = &routerSession

		if err := peer.SendRequest(ctx, uuid.NewString(), dst, session, "calculator.add", data); err != nil {
			return err
		}
	}
	for i := 0; i < 20; i++ {
		<-resultsChan
	}

	return nil
}

func relayCallback(ctx context.Context, p peer.Peer, response *types.Envelope, callBackErr error) {
	output, err := peer.Json(response, callBackErr)
	if err != nil {
		log.Error().Err(err)
		return
	}

	fmt.Printf("output: %s\n", string(output))
	resultsChan <- true
}

func main() {
	if err := app(); err != nil {
		log.Fatal().Msg(err.Error())
	}
}
