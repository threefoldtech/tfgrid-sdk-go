package messenger

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
)

// SignedMessage represents a cryptographically signed message with twin verification
// This structure ensures that messages are authenticated against the TFChain
type SignedMessage struct {
	TwinID    uint32 `json:"twin_id"`   // Twin ID from TFChain
	Message   string `json:"message"`   // Original message content
	Signature string `json:"signature"` // Hex-encoded Ed25519 signature
	Timestamp int64  `json:"timestamp"` // Unix timestamp for replay protection
}

// TwinKeyProvider defines the interface for retrieving twin public keys from blockchain
type TwinKeyProvider interface {
	// GetTwinPublicKey retrieves the Ed25519 public key for a given twin ID
	GetTwinPublicKey(twinID uint32) ([]byte, error)
}

// TFChainKeyProvider implements TwinKeyProvider using TFChain substrate connection
type TFChainKeyProvider struct {
	substrate *substrate.Substrate
}

// NewTFChainKeyProvider creates a new TFChain-based twin key provider
func NewTFChainKeyProvider(sub *substrate.Substrate) *TFChainKeyProvider {
	return &TFChainKeyProvider{substrate: sub}
}

// GetTwinPublicKey retrieves the Ed25519 public key for a twin from TFChain
func (p *TFChainKeyProvider) GetTwinPublicKey(twinID uint32) ([]byte, error) {
	twin, err := p.substrate.GetTwin(twinID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve twin %d from TFChain: %w", twinID, err)
	}

	//TODO: is this right?
	accountBytes := twin.Account[:]
	if len(accountBytes) < ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid account ID length for twin %d: expected at least %d bytes, got %d",
			twinID, ed25519.PublicKeySize, len(accountBytes))
	}

	return accountBytes[:ed25519.PublicKeySize], nil
}

// VerifyMessageSignature verifies a signed message against the twin's public key from TFChain
func VerifyMessageSignature(signedMsg *SignedMessage, keyProvider TwinKeyProvider) error {
	publicKeyBytes, err := keyProvider.GetTwinPublicKey(signedMsg.TwinID)
	if err != nil {
		return fmt.Errorf("failed to retrieve twin public key: %w", err)
	}

	if len(publicKeyBytes) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid public key length for twin %d: expected %d bytes, got %d",
			signedMsg.TwinID, ed25519.PublicKeySize, len(publicKeyBytes))
	}

	signatureBytes, err := hex.DecodeString(signedMsg.Signature)
	if err != nil {
		return fmt.Errorf("invalid signature encoding: %w", err)
	}

	if len(signatureBytes) != ed25519.SignatureSize {
		return fmt.Errorf("invalid signature length: expected %d bytes, got %d",
			ed25519.SignatureSize, len(signatureBytes))
	}

	publicKey := ed25519.PublicKey(publicKeyBytes)
	messageBytes := []byte(signedMsg.Message)

	if !ed25519.Verify(publicKey, messageBytes, signatureBytes) {
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
