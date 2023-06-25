package common

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/cosmos/go-bip39"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/common/types"
)

const (
	KeyTypeEd25519 = "ed25519"
	KeyTypeSr25519 = "sr25519"
)

type BaseClient struct {
	Source  *types.Address
	signer  substrate.Identity
	twinDB  TwinDB
	privKey *secp256k1.PrivateKey
	Reader  Reader
	Writer  Writer
}

func getIdentity(keytype string, mnemonics string) (substrate.Identity, error) {
	var identity substrate.Identity
	var err error

	switch keytype {
	case KeyTypeEd25519:
		identity, err = substrate.NewIdentityFromEd25519Phrase(mnemonics)
	case KeyTypeSr25519:
		identity, err = substrate.NewIdentityFromSr25519Phrase(mnemonics)
	default:
		return nil, fmt.Errorf("invalid key type %s, should be one of %s or %s ", keytype, KeyTypeEd25519, KeyTypeSr25519)
	}

	if err != nil {
		return nil, errors.Wrap(err, "failed to create identity")
	}
	return identity, nil
}

func generateSecureKey(mnemonics string) (*secp256k1.PrivateKey, error) {
	seed, err := bip39.NewSeedWithErrorChecking(mnemonics, "")
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode mnemonics")
	}
	priv := secp256k1.PrivKeyFromBytes(seed[:32])
	return priv, nil

}

func newAEAD(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func generateNonce(size int) ([]byte, error) {
	nonce := make([]byte, size)
	_, err := rand.Read(nonce)
	if err != nil {
		return nil, err
	}

	return nonce, nil
}

func NewBaseClient(ctx context.Context, keytype string, mnemonics string, relayURL string, session string, sub *substrate.Substrate, enableEncryption bool) (*BaseClient, error) {
	identity, err := getIdentity(keytype, mnemonics)
	if err != nil {
		return nil, err
	}

	twinDB := NewTwinDB(sub)
	id, err := twinDB.GetByPk(identity.PublicKey())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get twin by public key")
	}

	twin, err := twinDB.Get(id)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get twin id: %d", id)
	}

	url, err := url.Parse(relayURL)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse url: %s", relayURL)
	}

	var publicKey []byte
	var privKey *secp256k1.PrivateKey
	if enableEncryption {
		privKey, err = generateSecureKey(mnemonics)
		if err != nil {
			return nil, errors.Wrapf(err, "could not generate secure key")

		}
		publicKey = privKey.PubKey().SerializeCompressed()
	}

	if !bytes.Equal(twin.E2EKey, publicKey) || twin.Relay == nil || url.Hostname() != *twin.Relay {
		log.Info().Msg("twin relay/public key didn't match, updating on chain ...")
		if _, err = sub.UpdateTwin(identity, url.Hostname(), publicKey); err != nil {
			return nil, errors.Wrap(err, "could not update twin relay information")
		}
	}

	var sessionP *string
	if session != "" {
		sessionP = &session
	}
	source := types.Address{
		Twin:       id,
		Connection: sessionP,
	}
	conn := NewConnection(identity, relayURL, session, twin.ID)

	reader, writer := conn.Start(ctx)

	cl := &BaseClient{
		Source:  &source,
		signer:  identity,
		twinDB:  twinDB,
		privKey: privKey,
		Reader:  reader,
		Writer:  writer,
	}

	return cl, nil
}

func (b *BaseClient) decrypt(data []byte, pubKey []byte) ([]byte, error) {
	secPubKey, err := secp256k1.ParsePubKey(pubKey)
	if err != nil {
		return nil, errors.Wrapf(err, "could not parse dest public key")
	}
	sharedSecret := b.generateSharedSect(secPubKey)
	aead, err := newAEAD(sharedSecret[:])
	if err != nil {
		return nil, errors.Wrap(err, "failed to create AEAD")
	}
	if len(data) < aead.NonceSize() {
		return nil, errors.Errorf("Invalid cipher")
	}
	nonce := data[:aead.NonceSize()]

	decrypted, err := aead.Open(nil, nonce, data[aead.NonceSize():], nil)
	if err != nil {
		return nil, errors.Wrap(err, "could not decrypt message")
	}
	return decrypted, nil
}

func (b *BaseClient) encrypt(data []byte, pubKey []byte) ([]byte, error) {
	secPubKey, err := secp256k1.ParsePubKey(pubKey)
	if err != nil {
		return nil, errors.Wrapf(err, "could not parse dest public key")
	}
	sharedSecret := b.generateSharedSect(secPubKey)
	// Using ECDHE, derive a shared symmetric key for encryption of the plaintext.
	aead, err := newAEAD(sharedSecret[:])
	if err != nil {
		return nil, errors.Wrap(err, "failed to create AEAD {}")
	}

	nonce, err := generateNonce(aead.NonceSize())
	if err != nil {
		return nil, errors.Wrap(err, "could not generate nonce")
	}
	cipherText := make([]byte, len(nonce))
	copy(cipherText, nonce)
	cipherText = aead.Seal(cipherText, nonce, data, nil)
	return cipherText, nil
}

func (b *BaseClient) generateSharedSect(pubkey *secp256k1.PublicKey) [32]byte {
	point := secp256k1.GenerateSharedSecret(b.privKey, pubkey)
	return sha256.Sum256(point)
}

func (b *BaseClient) MakeRequest(ctx context.Context, twin uint32, fn string, data interface{}) (*types.Envelope, error) {
	payload, err := json.Marshal(data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to serialize request body")
	}

	var ttl uint64 = 5 * 60
	deadline, ok := ctx.Deadline()
	if ok {
		ttl = uint64(time.Until(deadline).Seconds())
	}

	schema := rmb.DefaultSchema

	env := types.Envelope{
		Uid:         uuid.NewString(),
		Timestamp:   uint64(time.Now().Unix()),
		Expiration:  ttl,
		Source:      b.Source,
		Destination: &types.Address{Twin: twin},
		Schema:      &schema,
	}

	env.Message = &types.Envelope_Request{
		Request: &types.Request{
			Command: fn,
		},
	}

	destTwin, err := b.twinDB.Get(twin)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get twin for %d", twin)
	}

	if len(destTwin.E2EKey) > 0 && b.privKey != nil {
		// destination public key is set, use e2e
		cipher, err := b.encrypt(payload, destTwin.E2EKey)
		if err != nil {
			return nil, errors.Wrapf(err, "could not encrypt data")
		}
		env.Payload = &types.Envelope_Cipher{
			Cipher: cipher,
		}

	} else {
		env.Payload = &types.Envelope_Plain{
			Plain: payload,
		}
	}

	env.Federation = destTwin.Relay

	toSign, err := Challenge(&env)
	if err != nil {
		return nil, err
	}

	env.Signature, err = Sign(b.signer, toSign)
	if err != nil {
		return nil, err
	}

	return &env, nil
}

func (b *BaseClient) HandleResponse(response *types.Envelope) ([]byte, error) {
	errResp := response.GetError()
	if response.Source == nil {
		// an envelope received that has NO source twin
		// this is possible only if the relay returned an error
		// hence
		if errResp != nil {
			return nil, fmt.Errorf(errResp.Message)
		}

		// otherwise that's a malformed message
		return nil, fmt.Errorf("received an invalid envelope")
	}

	err := VerifySignature(b.twinDB, response)
	if err != nil {
		return nil, errors.Wrap(err, "message signature verification failed")
	}

	if errResp != nil {
		// todo: include code also
		return nil, fmt.Errorf(errResp.Message)
	}

	resp := response.GetResponse()
	if resp == nil {
		return nil, fmt.Errorf("received a non response envelope")
	}

	if response.Schema == nil || *response.Schema != rmb.DefaultSchema {
		return nil, fmt.Errorf("invalid schema received expected '%s'", rmb.DefaultSchema)
	}

	var output []byte
	switch payload := response.Payload.(type) {
	case *types.Envelope_Cipher:
		twin, err := b.twinDB.Get(response.Source.Twin)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get twin object for %d", response.Source.Twin)
		}
		if len(twin.E2EKey) == 0 {
			return nil, errors.Wrap(err, "bad twin pk")
		}
		output, err = b.decrypt(payload.Cipher, twin.E2EKey)
		if err != nil {
			return nil, errors.Wrap(err, "could not decrypt payload")
		}
	case *types.Envelope_Plain:
		output = payload.Plain
	}

	return output, nil
}
