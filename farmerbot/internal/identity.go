package internal

import (
	"fmt"

	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
)

// GetIdentityWithKeyType returns chain identity given a key type. key type can be "ed25519" or "sr25519"
func GetIdentityWithKeyType(mnemonicOrSeed, keyType string) (identity substrate.Identity, err error) {
	switch keyType {
	case peer.KeyTypeEd25519:
		identity, err = substrate.NewIdentityFromEd25519Phrase(mnemonicOrSeed)
	case peer.KeyTypeSr25519:
		identity, err = substrate.NewIdentityFromSr25519Phrase(mnemonicOrSeed)
	default:
		err = fmt.Errorf("invalid key type %q", keyType)
	}
	return
}
