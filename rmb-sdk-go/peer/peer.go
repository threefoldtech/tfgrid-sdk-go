// Package direct package provides the functionality to create a direct websocket connection to rmb relays without the need to rmb peers.
package peer

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer/encoder"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer/types"
	"google.golang.org/protobuf/proto"
)

const (
	KeyTypeEd25519 = "ed25519"
	KeyTypeSr25519 = "sr25519"
)

// Handler is a call back that is called with verified and decrypted incoming
// messages. An error can be non-nil error if verification or decryption failed
type Handler func(ctx context.Context, peer *Peer, env *types.Envelope, err error)

type cacheFactory = func(inner TwinDB, chainURL string) (TwinDB, error)

var (
	// ErrNoValidRelayURLs is returned when no valid relay URLs are provided.
	ErrNoValidRelayURLs = errors.New("no valid relay URLs provided")
	// ErrNoFunctionalRelayAtStartup is returned when no relay connections are functional at startup.
	ErrNoFunctionalRelayAtStartup = errors.New("no relay connections are functional at startup")
)

type peerCfg struct {
	// Require at least one working relay at startup (default: false, for backward compatibility)
	RequireFunctionalRelayOnStartup bool
	relayURLs                       []string
	keyType                         string
	session                         string
	enableEncryption                bool
	encoder                         encoder.Encoder
	cacheFactory                    cacheFactory
}

type PeerOpt func(*peerCfg)

// WithSession set a custom session name, default is the nil session
func WithSession(session string) PeerOpt {
	return func(p *peerCfg) {
		p.session = session
	}
}

// enable or disable encryption, default is enabled
func WithEncryption(enable bool) PeerOpt {
	return func(p *peerCfg) {
		p.enableEncryption = enable
	}
}

// WithRequireFunctionalRelayOnStartup configures whether the peer should require at least one working relay at startup.
// If not set, the peer will always self-heal (default, backward compatible).
func WithRequireFunctionalRelayOnStartup(required bool) PeerOpt {
	return func(p *peerCfg) {
		p.RequireFunctionalRelayOnStartup = required
	}
}

// WithRelay set up the relay url, default is mainnet relay
func WithRelay(urls ...string) PeerOpt {
	return func(p *peerCfg) {
		p.relayURLs = urls
	}
}

// WithKeyType set up the mnemonic key type, default is Sr25519
func WithKeyType(keyType string) PeerOpt {
	return func(p *peerCfg) {
		// to ensure only ed25519 and sr25519 are used
		if keyType != KeyTypeEd25519 {
			keyType = KeyTypeSr25519
		}
		p.keyType = keyType
	}
}

// WithEncoder sets encoding of the payload default is application/json
func WithEncoder(encoder encoder.Encoder) PeerOpt {
	return func(p *peerCfg) {
		p.encoder = encoder
	}
}

// WithTwinCache cache twin information for this ttl number of seconds
// if ttl == 0, twins are cached forever
func WithTmpCacheExpiration(ttl uint64) PeerOpt {
	return func(pc *peerCfg) {
		pc.cacheFactory = func(inner TwinDB, chainURL string) (TwinDB, error) {
			return newTmpCache(ttl, inner, chainURL)
		}
	}
}

// if ttl == 0 twins are cached forever
func WithInMemoryExpiration(ttl uint64) PeerOpt {
	return func(pc *peerCfg) {
		pc.cacheFactory = func(inner TwinDB, chainURL string) (TwinDB, error) {
			return newInMemoryCache(inner, ttl), nil
		}
	}
}

// Peer exposes the functionality to talk directly to an rmb relay
type Peer struct {
	source  *types.Address
	signer  substrate.Identity
	twinDB  TwinDB
	privKey *secp256k1.PrivateKey
	reader  Reader
	cons    *WeightSlice[InnerConnection]
	handler Handler
	encoder encoder.Encoder
	relays  []string
}

func generateSecureKey(identity substrate.Identity) (*secp256k1.PrivateKey, error) {
	keyPair, err := identity.KeyPair()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate identity key pair")
	}

	priv := secp256k1.PrivKeyFromBytes(keyPair.Seed())
	return priv, nil
}
func validateRelayURLs(relayURLs []string) ([]*url.URL, error) {
	var validRelayURLs []*url.URL

	for _, relayURL := range relayURLs {
		parsedURL, err := url.Parse(strings.ToLower(relayURL))
		if err != nil {
			log.Warn().Err(err).Str("url", relayURL).Msg("failed to parse relay URL, skipping")
			continue
		}
		// make sure it is ws or wss
		if parsedURL.Scheme != "ws" && parsedURL.Scheme != "wss" {
			log.Warn().Str("url", relayURL).Msg("relay URL must be ws or wss, skipping")
			continue
		}
		// make sure Hostname is not empty
		if parsedURL.Hostname() == "" {
			log.Warn().Str("url", relayURL).Msg("relay URL must have a hostname, skipping")
			continue
		}
		validRelayURLs = append(validRelayURLs, parsedURL)
	}

	if len(validRelayURLs) == 0 {
		return nil, ErrNoValidRelayURLs
	}

	validRelayURLs = slices.CompactFunc(validRelayURLs, func(a, b *url.URL) bool {
		return a.Hostname() == b.Hostname()
	})

	slices.SortFunc(validRelayURLs, func(a, b *url.URL) int {
		return strings.Compare(a.Hostname(), b.Hostname())
	})
	return validRelayURLs, nil
}

// getRelayConnections tries to connect to all relays and returns only the successful ones
// getRelayConnections returns InnerConnections for all valid relay URLs
func getRelayConnections(relayURLs []string, identity substrate.Identity, session string, twinID uint32) ([]string, []InnerConnection, error) {
	validRelayURLs, err := validateRelayURLs(relayURLs)
	if err != nil {
		return nil, nil, err
	}
	connections := make([]InnerConnection, 0, len(validRelayURLs))
	hosts := make([]string, 0, len(validRelayURLs))

	for _, relayURL := range validRelayURLs {
		conn := NewConnection(identity, relayURL.String(), session, twinID)
		connections = append(connections, conn)
		host := relayURL.Hostname()
		hosts = append(hosts, host)
	}

	return hosts, connections, nil
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

// NewPeer creates a new RMB peer client. It connects directly to the RMB-Relay, and tries to reconnect if the connection broke.
//
// You can close the connection by canceling the passed context.
//
// Make sure the context passed to Call() does not outlive the directClient's context.
// Call() will panic if called while the directClient's context is canceled.
func NewPeer(
	ctx context.Context,
	mnemonics string,
	subManager substrate.Manager,
	handler Handler,
	opts ...PeerOpt,
) (*Peer, error) {
	cfg := &peerCfg{
		relayURLs:        []string{"wss://relay.grid.tf"},
		session:          "",
		enableEncryption: true,
		keyType:          KeyTypeSr25519,
		cacheFactory: func(inner TwinDB, _ string) (TwinDB, error) {
			return newInMemoryCache(inner, 0), nil
		},
	}

	for _, o := range opts {
		o(cfg)
	}

	if cfg.encoder == nil {
		cfg.encoder = encoder.NewJSONEncoder()
	}
	identity, err := getIdentity(cfg.keyType, mnemonics)
	if err != nil {
		return nil, err
	}

	subConn, err := subManager.Substrate()
	if err != nil {
		return nil, err
	}

	api, _, err := subConn.GetClient()
	if err != nil {
		return nil, err
	}

	twinDB, err := cfg.cacheFactory(NewTwinDB(subConn), api.Client.URL())
	if err != nil {
		return nil, err
	}

	id, err := twinDB.GetByPk(identity.PublicKey())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get twin by public key")
	}

	log.Info().Uint32("twin", id).Str("session", cfg.session).Msg("starting peer")

	twin, err := twinDB.Get(id)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get twin id: %d", id)
	}

	var publicKey []byte
	var privKey *secp256k1.PrivateKey
	if cfg.enableEncryption {
		privKey, err = generateSecureKey(identity)
		if err != nil {
			return nil, errors.Wrapf(err, "could not generate secure key")
		}
		publicKey = privKey.PubKey().SerializeCompressed()
	}

	hosts, conns, err := getRelayConnections(cfg.relayURLs, identity, cfg.session, twin.ID)
	if err != nil {
		return nil, err
	}

	// Hybrid approach: if RequireFunctionalRelayOnStartup is true, require at least one working relay at startup
	if cfg.RequireFunctionalRelayOnStartup {
		firstWorking := slices.IndexFunc(conns, func(c InnerConnection) bool {
			return c.TryConnect()
		})
		if firstWorking == -1 {
			return nil, ErrNoFunctionalRelayAtStartup
		}
	}

	joinURLs := strings.Join(hosts, "_")
	if !bytes.Equal(twin.E2EKey, publicKey) || twin.Relay == nil || joinURLs != *twin.Relay {
		log.Info().Str("Relay url/s", joinURLs).Msg("twin relay/public key didn't match, updating on chain ...")
		if _, err = subConn.UpdateTwin(identity, joinURLs, publicKey); err != nil {
			return nil, errors.Wrap(err, "could not update twin relay information")
		}
	}

	reader := make(chan []byte)
	weightCons := make([]WeightItem[InnerConnection], 0, len(conns))
	for _, conn := range conns {
		// Always start reconnection goroutine for all valid relays
		conn.Start(ctx, reader)
		weightCons = append(weightCons, WeightItem[InnerConnection]{Item: conn, Weight: 1})
	}

	cons, err := NewWeightSlice(weightCons)
	if err != nil {
		return nil, err
	}

	var sessionP *string
	if cfg.session != "" {
		sessionP = &cfg.session
	}
	source := types.Address{
		Twin:       id,
		Connection: sessionP,
	}

	cl := &Peer{
		source:  &source,
		signer:  identity,
		twinDB:  twinDB,
		privKey: privKey,
		reader:  reader,
		cons:    cons,
		handler: handler,
		encoder: cfg.encoder,
		relays:  hosts,
	}

	go cl.process(ctx)

	return cl, nil
}

// Encoder returns the peer's encoder.
func (p *Peer) Encoder() encoder.Encoder {
	return p.encoder
}

func (d *Peer) handleIncoming(incoming *types.Envelope) error {
	errResp := incoming.GetError()
	if incoming.Source == nil {
		// an envelope received that has NO source twin
		// this is possible only if the relay returned an error
		// hence
		if errResp != nil {
			return errors.New(errResp.Message)
		}

		// otherwise that's a malformed message
		return fmt.Errorf("received an invalid envelope")
	}

	if err := VerifySignature(d.twinDB, incoming); err != nil {
		return errors.Wrap(err, "message signature verification failed")
	}

	if errResp != nil {
		// todo: include code also
		return errors.New(errResp.Message)
	}

	var output []byte
	switch payload := incoming.Payload.(type) {
	case *types.Envelope_Cipher:
		if d.privKey == nil {
			// we received an encrypted message while
			// we have no encryption enabled on that peer
			return fmt.Errorf("received an encrypted message while encryption is not enabled")
		}
		twin, err := d.twinDB.Get(incoming.Source.Twin)
		if err != nil {
			return errors.Wrapf(err, "failed to get twin object for %d", incoming.Source.Twin)
		}
		if len(twin.E2EKey) == 0 {
			return errors.Wrap(err, "bad twin pk")
		}
		output, err = d.decrypt(payload.Cipher, twin.E2EKey)
		if err != nil {
			return errors.Wrap(err, "could not decrypt payload")
		}

		incoming.Payload = &types.Envelope_Plain{Plain: output}
	}

	return nil
}

func (d *Peer) process(ctx context.Context) {
	for {
		select {
		case incoming := <-d.reader:
			var env types.Envelope
			if err := proto.Unmarshal(incoming, &env); err != nil {
				log.Error().Err(err).Msg("invalid message payload")
				return
			}
			// verify and decoding!
			err := d.handleIncoming(&env)
			d.handler(ctx, d, &env, err)
		case <-ctx.Done():
			return
		}
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

func (d *Peer) generateSharedSect(pubkey *secp256k1.PublicKey) [32]byte {
	point := secp256k1.GenerateSharedSecret(d.privKey, pubkey)
	return sha256.Sum256(point)
}

func (d *Peer) encrypt(data []byte, pubKey []byte) ([]byte, error) {
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

func (d *Peer) decrypt(data []byte, pubKey []byte) ([]byte, error) {
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

func (d *Peer) makeEnvelope(id string, dest uint32, session *string, cmd *string, err error, data []byte, ttl uint64) (*types.Envelope, error) {
	schema := d.encoder.Schema()

	env := types.Envelope{
		Uid:        id,
		Timestamp:  uint64(time.Now().Unix()),
		Expiration: ttl,
		Source:     d.source,
		Destination: &types.Address{
			Twin:       dest,
			Connection: session,
		},
		Schema: &schema,
		Relays: d.relays,
	}

	if err != nil {
		env.Message = &types.Envelope_Error{
			Error: &types.Error{
				Message: err.Error(),
			},
		}
	} else if cmd == nil {
		env.Message = &types.Envelope_Response{
			Response: &types.Response{},
		}
	} else {
		env.Message = &types.Envelope_Request{
			Request: &types.Request{
				Command: *cmd,
			},
		}
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

func (d *Peer) send(ctx context.Context, request *types.Envelope) error {
	bytes, err := proto.Marshal(request)
	if err != nil {
		return err
	}

	var errs error

	for i := 0; i < len(d.cons.data); i++ {
		index, con := d.cons.Choose()
		err := con.send(ctx, bytes)
		if err != nil {
			errs = multierror.Append(errs, err)
			if errors.Is(err, errTimeout) && d.cons.data[index].Weight > 0 {
				d.cons.data[index].Weight--
			}
			continue
		}

		if d.cons.data[index].Weight < 100 {
			d.cons.data[index].Weight++
		}
		return nil
	}

	return errs
}

// SendRequest sends an rmb message to the relay
func (d *Peer) SendRequest(ctx context.Context, id string, twin uint32, session *string, fn string, data interface{}) error {
	payload, err := d.encoder.Encode(data)
	if err != nil {
		return errors.Wrap(err, "failed to serialize request body")
	}

	var ttl uint64 = 5 * 60
	deadline, ok := ctx.Deadline()
	if ok {
		ttl = uint64(time.Until(deadline).Seconds())
	}

	request, err := d.makeEnvelope(id, twin, session, &fn, nil, payload, ttl)
	if err != nil {
		return errors.Wrap(err, "failed to build request")
	}

	if err := d.send(ctx, request); err != nil {
		return err
	}

	return nil
}

// SendResponse sends an rmb message to the relay
func (d *Peer) SendResponse(ctx context.Context, id string, twin uint32, session *string, responseError error, data interface{}) error {
	payload, err := d.encoder.Encode(data)
	if err != nil {
		return errors.Wrap(err, "failed to serialize request body")
	}

	var ttl uint64 = 5 * 60
	deadline, ok := ctx.Deadline()
	if ok {
		ttl = uint64(time.Until(deadline).Seconds())
	}

	request, err := d.makeEnvelope(id, twin, session, nil, responseError, payload, ttl)
	if err != nil {
		return errors.Wrap(err, "failed to build request")
	}

	if err := d.send(ctx, request); err != nil {
		return err
	}

	return nil
}

// Json extracts the json payload envelope and validate the schema
func Json(response *types.Envelope, callBackErr error) ([]byte, error) {
	if callBackErr != nil {
		return []byte{}, callBackErr
	}

	errResp := response.GetError()
	if errResp != nil {
		return []byte{}, errors.New(errResp.Message)
	}

	resp := response.GetResponse()
	if resp == nil {
		return []byte{}, errors.New("received a non response envelope")
	}

	if response.Schema == nil || *response.Schema != rmb.DefaultSchema {
		return []byte{}, fmt.Errorf("invalid schema received expected '%s'", rmb.DefaultSchema)
	}

	output := response.Payload.(*types.Envelope_Plain).Plain
	return output, nil
}
