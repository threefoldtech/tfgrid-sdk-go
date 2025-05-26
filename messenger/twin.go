package messenger

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

type twinIdCtx struct{}

var TwinIdContextKey = twinIdCtx{}

// MyceliumNodeInfo represents the output of the Mycelium API admin endpoint
type MyceliumNodeInfo struct {
	NodeSubnet string `json:"nodeSubnet"`
	NodePubkey string `json:"nodePubkey"`
}

// GetMyceliumInfo retrieves the current Mycelium node information from the API
func (c *Messenger) GetMyceliumInfo() (*MyceliumNodeInfo, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(c.APIAddress + "/api/v1/admin")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Mycelium API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read API response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned non-OK status: %d - %s", resp.StatusCode, string(body))
	}

	var nodeInfo MyceliumNodeInfo
	if err := json.Unmarshal(body, &nodeInfo); err != nil {
		return nil, fmt.Errorf("failed to parse API response: %w", err)
	}

	log.Debug().Str("nodePubkey", nodeInfo.NodePubkey).
		Msg("Retrieved Mycelium node information")

	return &nodeInfo, nil
}

// UpdateTwinWithMyceliumPubkey updates mycelium public key in the twin map on chain
func (c *Messenger) UpdateTwinWithMyceliumPubkey(ctx context.Context) error {
	nodeInfo, err := c.GetMyceliumInfo()
	if err != nil {
		return fmt.Errorf("failed to get Mycelium node information: %w", err)
	}
	myceliumPk := []byte(nodeInfo.NodePubkey)

	twinid, err := c.subCon.GetTwinByPubKey([]byte(c.identity.PublicKey()))
	if err != nil {
		return fmt.Errorf("error getting twin public key: %w", err)
	}

	// TODO: better to map to pubkey instead of using twinid?
	return c.subCon.SetTwinMyceliumPK(c.identity, twinid, myceliumPk)
}
