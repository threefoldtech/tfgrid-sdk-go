// Package direct package provides the functionality to create a direct websocket connection to rmb relays without the need to rmb peers.
package direct

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
	"sync"
	"time"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/direct/types"
	"github.com/tyler-smith/go-bip39"
	"google.golang.org/protobuf/proto"
)

const (
	KeyTypeEd25519 = "ed25519"
	KeyTypeSr25519 = "sr25519"
)

var (
	_ rmb.Client = (*DirectClient)(nil)
)

// DirectClient exposes the functionality to talk directly to an rmb relay
type DirectClient struct {
	source    *types.Address
	signer    substrate.Identity
	responses map[string]chan *types.Envelope
	respM     sync.Mutex
	twinDB    TwinDB
	privKey   *secp256k1.PrivateKey
	reader    Reader
	writer    Writer
}

func generateSecureKey(mnemonics string) (*secp256k1.PrivateKey, error) {
	seed, err := bip39.NewSeedWithErrorChecking(mnemonics, "")
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode mnemonics")
	}
	priv := secp256k1.PrivKeyFromBytes(seed[:32])
	return priv, nil

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

// NewClient creates a new RMB direct client. It connects directly to the RMB-Relay, and peridically tries to reconnect if the connection broke.
//
// You can close the connection by canceling the passed context.
//
// Make sure the context passed to Call() does not outlive the directClient's context.
// Call() will panic if called while the directClient's context is canceled.
func NewClient(ctx context.Context, keytype string, mnemonics string, relayURL string, session string, sub *substrate.Substrate, enableEncryption bool) (*DirectClient, error) {
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
	conn := NewConnection(identity, relayURL, session, twin.ID)

	reader, writer := conn.Start(ctx)
	source := types.Address{
		Twin:       id,
		Connection: &session,
	}

	cl := &DirectClient{
		source:    &source,
		signer:    identity,
		responses: make(map[string]chan *types.Envelope),
		twinDB:    twinDB,
		privKey:   privKey,
		reader:    reader,
		writer:    writer,
	}
	go cl.process()

	return cl, nil
}

func (d *DirectClient) process() {
	for incoming := range d.reader {
		var env types.Envelope
		if err := proto.Unmarshal(incoming, &env); err != nil {
			log.Error().Err(err).Msg("invalid message payload")
			return
		}
		d.router(&env)
	}
}

func (d *DirectClient) router(env *types.Envelope) {
	d.respM.Lock()
	defer d.respM.Unlock()

	ch, ok := d.responses[env.Uid]
	if !ok {
		return
	}

	select {
	case ch <- env:
	default:
		// client is not waiting anymore! just return then
	}
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

func (d *DirectClient) generateSharedSect(pubkey *secp256k1.PublicKey) [32]byte {
	point := secp256k1.GenerateSharedSecret(d.privKey, pubkey)
	return sha256.Sum256(point)
}

func (d *DirectClient) encrypt(data []byte, pubKey []byte) ([]byte, error) {
	secPubKey, err := secp256k1.ParsePubKey(pubKey)
	if err != nil {
		return nil, errors.Wrapf(err, "could not parse dest public key")
	}
	sharedSecret := d.generateSharedSect(secPubKey)
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

func (d *DirectClient) decrypt(data []byte, pubKey []byte) ([]byte, error) {
	secPubKey, err := secp256k1.ParsePubKey(pubKey)
	if err != nil {
		return nil, errors.Wrapf(err, "could not parse dest public key")
	}
	sharedSecret := d.generateSharedSect(secPubKey)
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

func (d *DirectClient) makeRequest(dest uint32, cmd string, data []byte, ttl uint64) (*types.Envelope, error) {
	schema := rmb.DefaultSchema

	env := types.Envelope{
		Uid:         uuid.NewString(),
		Timestamp:   uint64(time.Now().Unix()),
		Expiration:  ttl,
		Source:      d.source,
		Destination: &types.Address{Twin: dest},
		Schema:      &schema,
	}

	env.Message = &types.Envelope_Request{
		Request: &types.Request{
			Command: cmd,
		},
	}

	destTwin, err := d.twinDB.Get(dest)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get twin for %d", dest)
	}

	if len(destTwin.E2EKey) > 0 && d.privKey != nil {
		// destination public key is set, use e2e
		cipher, err := d.encrypt(data, destTwin.E2EKey)
		if err != nil {
			return nil, errors.Wrapf(err, "could not encrypt data")
		}
		env.Payload = &types.Envelope_Cipher{
			Cipher: cipher,
		}

	} else {
		env.Payload = &types.Envelope_Plain{
			Plain: data,
		}
	}

	env.Federation = destTwin.Relay

	toSign, err := Challenge(&env)
	if err != nil {
		return nil, err
	}

	env.Signature, err = Sign(d.signer, toSign)
	if err != nil {
		return nil, err
	}

	return &env, nil

}

func (d *DirectClient) request(ctx context.Context, request *types.Envelope) (*types.Envelope, error) {

	ch := make(chan *types.Envelope)
	d.respM.Lock()
	d.responses[request.Uid] = ch
	d.respM.Unlock()

	bytes, err := proto.Marshal(request)
	if err != nil {
		return nil, err
	}

	select {
	case d.writer <- bytes:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	var response *types.Envelope
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case response = <-ch:
	}
	if response == nil {
		// shouldn't happen but just in case
		return nil, fmt.Errorf("no response received")
	}

	return response, nil
}

// Call sends an rmb call to the relay
func (d *DirectClient) Call(ctx context.Context, twin uint32, fn string, data interface{}, result interface{}) error {

	payload, err := json.Marshal(data)
	if err != nil {
		return errors.Wrap(err, "failed to serialize request body")
	}

	var ttl uint64 = 5 * 60
	deadline, ok := ctx.Deadline()
	if ok {
		ttl = uint64(time.Until(deadline).Seconds())
	}

	request, err := d.makeRequest(twin, fn, payload, ttl)
	if err != nil {
		return errors.Wrap(err, "failed to build request")
	}

	response, err := d.request(ctx, request)
	if err != nil {
		return err
	}

	errResp := response.GetError()
	if response.Source == nil {
		// an envelope received that has NO source twin
		// this is possible only if the relay returned an error
		// hence
		if errResp != nil {
			return fmt.Errorf(errResp.Message)
		}

		// otherwise that's a malformed message
		return fmt.Errorf("received an invalid envelope")
	}

	err = VerifySignature(d.twinDB, response)
	if err != nil {
		return errors.Wrap(err, "message signature verification failed")
	}

	if errResp != nil {
		// todo: include code also
		return fmt.Errorf(errResp.Message)
	}

	resp := response.GetResponse()
	if resp == nil {
		return fmt.Errorf("received a non response envelope")
	}

	if result == nil {
		return nil
	}

	if response.Schema == nil || *response.Schema != rmb.DefaultSchema {
		return fmt.Errorf("invalid schema received expected '%s'", rmb.DefaultSchema)
	}

	var output []byte
	switch payload := response.Payload.(type) {
	case *types.Envelope_Cipher:
		twin, err := d.twinDB.Get(response.Source.Twin)
		if err != nil {
			return errors.Wrapf(err, "failed to get twin object for %d", response.Source.Twin)
		}
		if len(twin.E2EKey) == 0 {
			return errors.Wrap(err, "bad twin pk")
		}
		output, err = d.decrypt(payload.Cipher, twin.E2EKey)
		if err != nil {
			return errors.Wrap(err, "could not decrypt payload")
		}
	case *types.Envelope_Plain:
		output = payload.Plain
	}

	return json.Unmarshal(output, &result)
}

// Ping sends an application level ping. You normally do not ever need to call this
// yourself because this rmb client takes care of automatic pinging of the server
// and reconnecting if needed. But in case you want to test if a connection is active
// and established you can call this Ping method yourself.
// If no error is returned then ping has succeeded.
// Make sure to always provide a ctx with a timeout or a deadline otherwise the call
// will block forever waiting for a response.
func (d *DirectClient) Ping(ctx context.Context) error {
	uid := uuid.NewString()
	request := types.Envelope{
		Uid:     uid,
		Source:  d.source,
		Message: &types.Envelope_Ping{},
	}

	response, err := d.request(ctx, &request)
	if err != nil {
		return err
	}
	_, ok := response.Message.(*types.Envelope_Pong)
	if !ok {
		return fmt.Errorf("expected a pong response got %T", response.Message)
	}

	return nil
}
