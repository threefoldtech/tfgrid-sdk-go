package manager

import (
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
)

// Sub is substrate client interface
type Sub interface {
	SetNodePowerState(identity substrate.Identity, up bool) (hash types.Hash, err error)
	GetNodeRentContract(node uint32) (uint64, error)
	// NewIdentityFromSr25519Phrase(mnemonics string) (substrate.Identity, error)
}
