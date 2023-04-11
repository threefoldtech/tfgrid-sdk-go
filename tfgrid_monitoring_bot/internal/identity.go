// Package internal contains all logic for monitoring service
package internal

import (
	substrate "github.com/threefoldtech/substrate-client"
)

// Identity is the user identity to be used in substrate
type Identity substrate.Identity

// NewIdentityFromSr25519Phrase generates a new Sr25519 identity from mnemonics
func NewIdentityFromSr25519Phrase(phrase string) (Identity, error) {
	id, err := substrate.NewIdentityFromSr25519Phrase(phrase)
	return Identity(id), err
}
