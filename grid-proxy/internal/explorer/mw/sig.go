package mw

import (
	"github.com/pkg/errors"
	"github.com/vedhavyas/go-subkey/v2"
	"github.com/vedhavyas/go-subkey/v2/ed25519"
	"github.com/vedhavyas/go-subkey/v2/sr25519"
)

const (
	PubKeySize = 32
)

var (
	ErrInvalidInput = errors.New("invalid input")
	ErrVerifyFailed = errors.New("signature verification failed")
)

// verifyWithScheme verifies a signature using the given scheme (ed25519 or sr25519)
func verifyWithScheme(scheme subkey.Scheme, publicKey, message, signature []byte) bool {
	key, err := scheme.FromPublicKey(publicKey)
	if err != nil {
		return false
	}
	return key.Verify(message, signature)
}

func verifySignature(publicKey, message, signature []byte) error {
	// Validate public key length
	if len(publicKey) != PubKeySize {
		return errors.Wrapf(ErrInvalidInput, "invalid public key size: expected %d, got %d", PubKeySize, len(publicKey))
	}

	if len(message) == 0 {
		return errors.Wrap(ErrInvalidInput, "invalid message size, not expected to be zero")
	}

	if len(signature) == 0 {
		return errors.Wrap(ErrInvalidInput, "invalid signature size, not expected to be zero")
	}

	// Try ED25519 verification first (most common)
	if verifyWithScheme(ed25519.Scheme{}, publicKey, message, signature) {
		return nil
	}

	// Fallback to SR25519 verification
	if verifyWithScheme(sr25519.Scheme{}, publicKey, message, signature) {
		return nil
	}

	return ErrVerifyFailed
}
