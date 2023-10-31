package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/direct"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/direct/types"
)

type resultsChan struct {
	resutls chan bool
}

func app() error {
	mnemonics := "<mnemonics goes here>"
	subManager := substrate.NewManager("wss://tfchain.dev.grid.tf/ws")
	sub, err := subManager.Substrate()
	ctx := context.Background()
	if err != nil {
		return fmt.Errorf("failed to connect to substrate: %w", err)
	}

	defer sub.Close()

	res := resultsChan{resutls: make(chan bool)}

	client, err := direct.NewClient(
		ctx,
		direct.KeyTypeSr25519,
		mnemonics,
		"wss://relay.dev.grid.tf",
		"test-client",
		sub,
		false,
		res.relayCallback,
	)

	if err != nil {
		return fmt.Errorf("failed to create direct client: %w", err)
	}

	for i := 0; i < 20; i++ {
		data := []float64{rand.Float64(), rand.Float64()}
		if err := client.Call(ctx, uuid.NewString(), 4973, "calculator.add", data); err != nil {
			return err
		}
	}
	for i := 0; i < 20; i++ {
		<-res.resutls
	}

	return nil
}

func (res *resultsChan) relayCallback(response *types.Envelope, callBackErr error) {

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
	res.resutls <- true
}

func main() {
	if err := app(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
