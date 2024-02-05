package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"

	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
	// "rmbClient/peer"
)

const (
	chainUrl = "wss://tfchain.grid.tf/"
	relayUrl = "wss://relay.grid.tf"
	mnemonic = "<MNEMONIC>"
)

type Node struct {
	TwinId uint32 `json:"twinId"`
}

func main() {
	subMan := substrate.NewManager(chainUrl)
	bus, err := peer.NewRpcClient(context.Background(), peer.KeyTypeSr25519, mnemonic, relayUrl, "rmb-playground999", subMan, false)
	if err != nil {
		fmt.Println("failed to create peer client: %w", err)
		os.Exit(1)
	}

	res, err := http.Get("https://gridproxy.bknd1.ninja.tf/nodes?healthy=true&size=100")
	if err != nil {
		fmt.Println("failed getting nodes")
	}

	var nodes []Node
	err = json.NewDecoder(res.Body).Decode(&nodes)
	if err != nil {
		fmt.Println("failed to decode res")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	for _, node := range nodes {
		wg.Add(1)
		go func(twinId uint32) {
			defer wg.Done()
			err := rmbCall(ctx, bus, twinId)
			if err != nil {
				log.Error().Err(err).Uint32("twinId", twinId).Msg("failed")
			}

		}(node.TwinId)
	}

	wg.Wait()
}

func rmbCall(ctx context.Context, bus *peer.RpcClient, twinId uint32) error {

	var res interface{}
	err := bus.Call(ctx, twinId, "zos.system.version", nil, &res)
	if err != nil {
		return err
	}

	log.Info().Uint32("twinId", twinId).Msgf("%+v", res)

	return nil
}
