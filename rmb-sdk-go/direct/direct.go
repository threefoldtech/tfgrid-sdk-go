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
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/direct/types"
	"github.com/tyler-smith/go-bip39"
	"google.golang.org/protobuf/proto"
)

const (
	KeyTypeEd25519 = "ed25519"
	KeyTypeSr25519 = "sr25519"
)

type directClient struct {
	source    *types.Address
	signer    substrate.Identity
	con       *websocket.Conn
	responses map[string]chan *types.Envelope
	m         sync.Mutex
	twinDB    TwinDB
	privKey   *secp256k1.PrivateKey
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

// id is the twin id that is associated with the given identity.
func NewClient(keytype string, mnemonics string, relayUrl string, session string, sub *substrate.Substrate) (rmb.Client, error) {

	identity, err := getIdentity(keytype, mnemonics)
	if err != nil {
		return nil, err
	}

	privKey, err := generateSecureKey(mnemonics)
	if err != nil {
		return nil, errors.Wrapf(err, "could not generate secure key")
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

	url, err := url.Parse(relayUrl)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse url: %s", relayUrl)
	}

	if !bytes.Equal(twin.E2EKey, privKey.PubKey().SerializeCompressed()) || twin.Relay == nil || url.Hostname() != *twin.Relay {
		log.Info().Msg("twin relay/public key didn't match, updating on chain ...")
		_, err := sub.UpdateTwin(identity, url.Hostname(), privKey.PubKey().SerializeCompressed())
		if err != nil {
			log.Error().Err(err)
		}
	}

	token, err := NewJWT(identity, id, session, 60) // use 1 min token ttl
	if err != nil {
		return nil, errors.Wrap(err, "failed to build authentication token")
	}
	// wss://relay.dev.grid.tf/?<JWT>
	relayUrl = fmt.Sprintf("%s?%s", relayUrl, token)
	source := types.Address{
		Twin:       id,
		Connection: &session,
	}

	con, resp, err := websocket.DefaultDialer.Dial(relayUrl, nil)
	if err != nil {
		var body []byte
		var status string
		if resp != nil {
			status = resp.Status
			body, _ = io.ReadAll(resp.Body)
		}
		return nil, errors.Wrapf(err, "failed to connect (%s): %s", status, string(body))
	}

	if resp.StatusCode != http.StatusSwitchingProtocols {
		return nil, fmt.Errorf("invalid response %s", resp.Status)
	}

	cl := &directClient{
		source:    &source,
		signer:    identity,
		con:       con,
		responses: make(map[string]chan *types.Envelope),
		twinDB:    twinDB,
		privKey:   privKey,
	}

	go cl.process()
	return cl, nil
}

func (d *directClient) process() {
	defer d.con.Close()
	// todo: set error on connection here
	for {
		typ, msg, err := d.con.ReadMessage()
		if err != nil {
			log.Error().Err(err).Msg("websocket error connection closed")
			return
		}
		if typ != websocket.BinaryMessage {
			continue
		}

		var env types.Envelope
		if err := proto.Unmarshal(msg, &env); err != nil {
			log.Error().Err(err).Msg("invalid message payload")
			return
		}

		d.router(&env)
	}
}

func (d *directClient) router(env *types.Envelope) {
	d.m.Lock()
	defer d.m.Unlock()

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

func generateNonce(size int) []byte {
	nonce := make([]byte, size)
	_, err := rand.Read(nonce)
	if err != nil {
		log.Error().Err(err)
	}
	return nonce
}

func (d *directClient) generateSharedSect(pubkey *secp256k1.PublicKey) [32]byte {
	point := secp256k1.GenerateSharedSecret(d.privKey, pubkey)
	return sha256.Sum256(point)
}
func (d *directClient) encrypt(data []byte, pubKey []byte) ([]byte, error) {
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

	nonce := generateNonce(aead.NonceSize())
	cipherText := make([]byte, len(nonce))
	copy(cipherText, nonce)
	cipherText = aead.Seal(cipherText, nonce, data, nil)
	return cipherText, nil
}

func (d *directClient) decrypt(data []byte, pubKey []byte) ([]byte, error) {
	secPubKey, err := secp256k1.ParsePubKey(pubKey)
	if err != nil {
		return nil, errors.Wrapf(err, "could not parse dest public key")
	}
	sharedSecret := d.generateSharedSect(secPubKey)
	aead, err := newAEAD(sharedSecret[:])
	if err != nil {
		return nil, errors.Wrap(err, "failed to create AEAD {}")
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

func (d *directClient) makeRequest(dest uint32, cmd string, data []byte, ttl uint64) (*types.Envelope, error) {
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

	if len(destTwin.E2EKey) > 0 {
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

func (d *directClient) Call(ctx context.Context, twin uint32, fn string, data interface{}, result interface{}) error {

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

	ch := make(chan *types.Envelope)
	d.m.Lock()
	d.responses[request.Uid] = ch
	d.m.Unlock()

	bytes, err := proto.Marshal(request)
	if err != nil {
		return err
	}

	if err := d.con.WriteMessage(websocket.BinaryMessage, bytes); err != nil {
		return err
	}

	var response *types.Envelope
	select {
	case <-ctx.Done():
		return ctx.Err()
	case response = <-ch:
	}
	if response == nil {
		// shouldn't happen but just in case
		return fmt.Errorf("no response received")
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
			return errors.Wrapf(err, "failed to get twin object for {%d}", response.Source.Twin)
		}
		if len(twin.E2EKey) == 0 {
			return errors.Wrap(err, "bad twin pk")
		}
		fmt.Println(twin.E2EKey)
		output, err = d.decrypt(payload.Cipher, twin.E2EKey)
		if err != nil {
			return errors.Wrap(err, "could not decrypt payload")
		}
	case *types.Envelope_Plain:
		output = payload.Plain
	}

	return json.Unmarshal(output, &result)
}
