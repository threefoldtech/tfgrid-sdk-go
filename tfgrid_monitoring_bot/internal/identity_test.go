// Package internal contains all logic for monitoring service
package internal

import (
	"testing"
)

func TestIdentity(t *testing.T) {
	mnemonics := "//Alice"
	address := "5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY"

	t.Run("test_right_identity", func(t *testing.T) {
		identity, err := NewIdentityFromSr25519Phrase(mnemonics)

		if err != nil {
			t.Error(err)
		}

		if identity.Address() != address {
			t.Errorf("wrong identity is generated")
		}
	})

	t.Run("test_wrong_identity", func(t *testing.T) {
		_, err := NewIdentityFromSr25519Phrase("mnemonics")

		if err == nil {
			t.Errorf("expected failure, got no error")
		}
	})
}
