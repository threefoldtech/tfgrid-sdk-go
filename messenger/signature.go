package messenger

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	sr25519 "github.com/ChainSafe/go-schnorrkel"
	"github.com/gtank/merlin"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
)

// SignedMessage represents a cryptographically signed message with twin verification
// This structure ensures that messages are authenticated against the TFChain using SR25519
type SignedMessage struct {
	TwinID    uint32 `json:"twin_id"`   // Twin ID from TFChain
	Message   string `json:"message"`   // Original message content
	Signature string `json:"signature"` // Hex-encoded SR25519 signature
	Timestamp int64  `json:"timestamp"` // Unix timestamp for replay protection
}

// TwinKeyProvider defines the interface for retrieving twin public keys from blockchain
type TwinKeyProvider interface {
	// GetTwinPublicKey retrieves the SR25519 public key for a given twin ID
	GetTwinPublicKey(twinID uint32) ([]byte, error)
}

// signingContext creates the signing context used by SR25519
func signingContext(msg []byte) *merlin.Transcript {
	return sr25519.NewSigningContext([]byte("substrate"), msg)
}

// TFChainKeyProvider implements TwinKeyProvider using TFChain substrate connection
type TFChainKeyProvider struct {
	substrate *substrate.Substrate
}

// NewTFChainKeyProvider creates a new TFChain-based twin key provider
func NewTFChainKeyProvider(sub *substrate.Substrate) *TFChainKeyProvider {
	return &TFChainKeyProvider{substrate: sub}
}

// GetTwinPublicKey retrieves the SR25519 public key for a twin from TFChain
func (p *TFChainKeyProvider) GetTwinPublicKey(twinID uint32) ([]byte, error) {
	twin, err := p.substrate.GetTwin(twinID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve twin %d from TFChain: %w", twinID, err)
	}

	//TODO: is this right?
	accountBytes := twin.Account[:]
	if len(accountBytes) < 32 { // SR25519 public key size
		return nil, fmt.Errorf("invalid account ID length for twin %d: expected at least %d bytes, got %d",
			twinID, 32, len(accountBytes))
	}

	return accountBytes[:32], nil
}

// VerifyMessageSignature verifies a signed message against the twin's public key from TFChain
func VerifyMessageSignature(signedMsg *SignedMessage, keyProvider TwinKeyProvider) error {
	publicKeyBytes, err := keyProvider.GetTwinPublicKey(signedMsg.TwinID)
	if err != nil {
		return fmt.Errorf("failed to retrieve twin public key: %w", err)
	}

	if len(publicKeyBytes) != 32 { // SR25519 public key size
		return fmt.Errorf("invalid public key length for twin %d: expected %d bytes, got %d",
			signedMsg.TwinID, 32, len(publicKeyBytes))
	}

	signatureBytes, err := hex.DecodeString(signedMsg.Signature)
	if err != nil {
		return fmt.Errorf("invalid signature encoding: %w", err)
	}

	if len(signatureBytes) != 64 { // SR25519 signature size
		return fmt.Errorf("invalid signature length: expected %d bytes, got %d",
			64, len(signatureBytes))
	}

	// Convert public key bytes to SR25519 public key
	var pubKeyArray [32]byte
	copy(pubKeyArray[:], publicKeyBytes)
	publicKey := new(sr25519.PublicKey)
	if err := publicKey.Decode(pubKeyArray); err != nil {
		return fmt.Errorf("failed to decode SR25519 public key: %w", err)
	}

	// Convert signature bytes to SR25519 signature
	var sigArray [64]byte
	copy(sigArray[:], signatureBytes)
	signature := new(sr25519.Signature)
	if err := signature.Decode(sigArray); err != nil {
		return fmt.Errorf("failed to decode SR25519 signature: %w", err)
	}

	messageBytes := []byte(signedMsg.Message)

	// Verify the signature using SR25519
	valid, err := publicKey.Verify(signature, signingContext(messageBytes))
	if err != nil {
		return fmt.Errorf("SR25519 signature verification error: %w", err)
	}

	if !valid {
		return fmt.Errorf("cryptographic signature verification failed for twin %d", signedMsg.TwinID)
	}

	return nil
}

// ParseSignedMessage parses a JSON payload into a SignedMessage struct
func ParseSignedMessage(payload string) (*SignedMessage, error) {
	var signedMsg SignedMessage
	if err := json.Unmarshal([]byte(payload), &signedMsg); err != nil {
		return nil, fmt.Errorf("failed to parse signed message: %w", err)
	}

	if signedMsg.TwinID == 0 {
		return nil, fmt.Errorf("twin_id is required and must be greater than 0")
	}
	if signedMsg.Message == "" {
		return nil, fmt.Errorf("message content is required")
	}
	if signedMsg.Signature == "" {
		return nil, fmt.Errorf("signature is required")
	}

	return &signedMsg, nil
}

// ValidateAndExtractMessage validates a signed message against TFChain and extracts the content
// Returns: twinID, originalMessage, error
func ValidateAndExtractMessage(payload string, keyProvider TwinKeyProvider) (uint32, string, error) {
	signedMsg, err := ParseSignedMessage(payload)
	if err != nil {
		return 0, "", fmt.Errorf("message parsing failed: %w", err)
	}

	if err := VerifyMessageSignature(signedMsg, keyProvider); err != nil {
		return 0, "", fmt.Errorf("signature verification failed: %w", err)
	}

	return signedMsg.TwinID, signedMsg.Message, nil
}

// CreateSignedMessage creates a cryptographically signed message using twin's private key
func CreateSignedMessage(twinID uint32, message string, identity substrate.Identity) (*SignedMessage, error) {
	signature, err := identity.Sign([]byte(message))
	if err != nil {
		return nil, fmt.Errorf("failed to sign message with twin identity: %w", err)
	}

	signedMsg := &SignedMessage{
		TwinID:    twinID,
		Message:   message,
		Signature: hex.EncodeToString(signature),
		Timestamp: time.Now().Unix(),
	}

	return signedMsg, nil
}
