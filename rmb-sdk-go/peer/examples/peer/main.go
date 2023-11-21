package main

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go"
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
		peer.KeyTypeSr25519,
		mnemonics,
		"wss://relay.dev.grid.tf",
		"test-client",
		sub,
		false,
		relayCallback,
	)

	if err != nil {
		return fmt.Errorf("failed to create direct client: %w", err)
	}

	const dist = 7 // <- replace this with the twin id of where the service is running

	for i := 0; i < 20; i++ {
		data := []float64{rand.Float64(), rand.Float64()}
		if err := peer.SendRequest(ctx, uuid.NewString(), dist, nil, "calculator.add", data); err != nil {
			return err
		}
	}
	for i := 0; i < 20; i++ {
		<-resultsChan
	}

	return nil
}

func relayCallback(response *types.Envelope, callBackErr error) {

	errResp := response.GetError()

	if errResp != nil {
		log.Error().Msg(errResp.Message)
		return
	}

	resp := response.GetResponse()
	if resp == nil {
		log.Error().Msg("received a non response envelope")
		return
	}

	if response.Schema == nil || *response.Schema != rmb.DefaultSchema {
		log.Error().Msgf("invalid schema received expected '%s'", rmb.DefaultSchema)
		return
	}

	output := response.Payload.(*types.Envelope_Plain).Plain

	fmt.Printf("output: %s\n", string(output))
	resultsChan <- true
}

func main() {
	if err := app(); err != nil {
		log.Fatal().Msg(err.Error())
	}
}
