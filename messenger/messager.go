package messenger

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
)

const (
	// DefaultMessengerBinary is the default path to the Mycelium binary
	DefaultMessengerBinary = "mycelium"

	// DefaultAPIAddress is the default address for the Mycelium API
	DefaultAPIAddress = "http://127.0.0.1:8989"

	// DefaultTimeout is the default timeout for sending messages
	DefaultTimeout = 60 // seconds

	// DefaultRetryListenerInterval is the default interval for retrying the message listener
	DefaultRetryListenerInterval = 100 * time.Millisecond
)

// Message represents a message structure used in the Mycelium messaging system
type Message struct {
	ID      string `json:"id,omitempty"`
	Topic   string `json:"topic,omitempty"`
	SrcIP   string `json:"srcIp,omitempty"`
	SrcPK   string `json:"srcPk,omitempty"`
	DstIP   string `json:"dstIp,omitempty"`
	DstPK   string `json:"dstPk,omitempty"`
	Payload string `json:"payload,omitempty"`
}

// MessageDestination represents the destination for API messages
type MessageDestination struct {
	PK string `json:"pk,omitempty"`
	IP string `json:"ip,omitempty"`
}

// PushMessageBody represents the request body for API message sending
type PushMessageBody struct {
	Dst     MessageDestination `json:"dst"`
	Topic   string             `json:"topic,omitempty"`
	Payload string             `json:"payload"`
}

type MessageHandlerFunc func(ctx context.Context, message *Message) ([]byte, error)

type Messenger struct {
	BinaryPath string
	APIAddress string
	Timeout    int
	Mnemonic   string

	// EnableTwinIdentity will enfoce the messenger to manage twin identity on chain
	EnableTwinIdentity bool
	subCon             *substrate.Substrate
	identity           substrate.Identity
	manager            substrate.Manager

	receiveHandlers map[string]MessageHandlerFunc
	stopCh          chan struct{}
	wg              sync.WaitGroup
	mutex           sync.RWMutex
}

// MessengerOpt is a function that configures a Client
type MessengerOpt func(*Messenger)

// WithMnemonic sets the mnemonic phrase for the client,
func WithMnemonic(mnemonic string) MessengerOpt {
	return func(c *Messenger) {
		c.Mnemonic = mnemonic
	}
}

func WithIdentity(identity substrate.Identity) MessengerOpt {
	return func(c *Messenger) {
		c.identity = identity
	}
}

// WithEnableTwinIdentity enables or disables twin identity management
func WithEnableTwinIdentity(enable bool) MessengerOpt {
	return func(c *Messenger) {
		c.EnableTwinIdentity = enable
	}
}

// WithSubstrateManager sets the substrate manager for the messenger
func WithSubstrateManager(manager substrate.Manager) MessengerOpt {
	return func(c *Messenger) {
		c.manager = manager
	}
}

// WithBinaryPath sets the binary path for the messenger
func WithBinaryPath(binaryPath string) MessengerOpt {
	return func(c *Messenger) {
		c.BinaryPath = binaryPath
	}
}

// WithAPIAddress sets the API address for the messenger
func WithAPIAddress(apiAddress string) MessengerOpt {
	return func(c *Messenger) {
		c.APIAddress = apiAddress
	}
}

// NewMessenger creates a new mycelium message subsystem client with the given options
func NewMessenger(opts ...MessengerOpt) (*Messenger, error) {
	messenger := &Messenger{
		BinaryPath:         DefaultMessengerBinary,
		Timeout:            DefaultTimeout,
		APIAddress:         DefaultAPIAddress,
		EnableTwinIdentity: false,
		receiveHandlers:    make(map[string]MessageHandlerFunc),
		stopCh:             make(chan struct{}),
	}

	for _, opt := range opts {
		opt(messenger)
	}

	if !messenger.EnableTwinIdentity {
		return messenger, nil
	}

	if messenger.manager == nil {
		return nil, fmt.Errorf("substrate manager is required when EnableTwinIdentity is true")
	}

	if messenger.identity == nil && messenger.Mnemonic == "" {
		return nil, fmt.Errorf("either an identity or mnemonic phrase is required when EnableTwinIdentity is true")
	}

	if messenger.identity == nil {
		var err error
		messenger.identity, err = substrate.NewIdentityFromSr25519Phrase(messenger.Mnemonic)
		if err != nil {
			return nil, fmt.Errorf("failed to create identity from mnemonic phrase: %w", err)
		}
	}

	subCon, err := messenger.manager.Substrate()
	if err != nil {
		return nil, fmt.Errorf("failed to get substrate connection: %w", err)
	}
	messenger.subCon = subCon

	if err := messenger.UpdateMyceliumTwin(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to update twin with Mycelium public key: %w", err)
	}

	return messenger, nil
}

// the main rpc handler for the messenger for JSONRPC messages
func (c *Messenger) RegisterHandler(topic string, handler MessageHandlerFunc) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.receiveHandlers[topic] = handler
}

func (c *Messenger) SendMessage(destination, payload string, topic string, waitForReply bool, timeout int) (*Message, error) {
	log.Debug().
		Str("destination", destination).
		Msg("sending message")
	args := []string{"message", "send", destination, payload}

	if waitForReply {
		args = append(args, "--wait")

		if timeout > 0 {
			args = append(args, "--timeout", fmt.Sprintf("%d", timeout))
		} else if c.Timeout > 0 {
			args = append(args, "--timeout", fmt.Sprintf("%d", c.Timeout))
		}
	}
	if topic != "" {
		args = append(args, "--topic", topic)
	}

	cmd := exec.Command(c.BinaryPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %s: %w", string(output), err)
	}

	if !waitForReply {
		return nil, nil
	}

	responseStr := string(output)
	responseStr = strings.TrimSpace(responseStr)

	if responseStr == "" {
		return nil, nil
	}

	var msg Message
	if err := json.Unmarshal([]byte(responseStr), &msg); err != nil {
		return nil, fmt.Errorf("failed to parse message response: %w", err)
	}

	log.Debug().
		Str("source", msg.SrcPK).
		Str("msg_id", msg.ID).
		Msg("received reply")
	return &msg, nil
}

func (c *Messenger) SendReply(originalMessageID, destination, payload string) error {
	log.Debug().
		Str("destination", destination).
		Str("msg_id", originalMessageID).
		Msg("replying to message")

	encodedPayload := base64.StdEncoding.EncodeToString([]byte(payload))
	// TODO: better error handling
	requestBody := PushMessageBody{
		Dst: MessageDestination{
			PK: destination,
		},
		Payload: encodedPayload,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/messages/reply/%s", c.APIAddress, originalMessageID)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{
		Timeout: time.Duration(c.Timeout) * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("API returned non-success status: %d - %s", resp.StatusCode, string(body))
	}

	return nil
}

func (c *Messenger) ReceiveMessage() (*Message, error) {
	args := []string{"message", "receive"}

	cmd := exec.Command(c.BinaryPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to receive message: %s: %w", string(output), err)
	}

	responseStr := string(output)
	responseStr = strings.TrimSpace(responseStr)

	if responseStr == "" {
		return nil, nil
	}

	var msg Message
	if err := json.Unmarshal([]byte(responseStr), &msg); err != nil {
		return nil, fmt.Errorf("failed to parse message: %w", err)
	}

	log.Debug().
		Str("source", msg.SrcPK).
		Str("msg_id", msg.ID).
		Msg("received message")
	return &msg, nil
}

func (c *Messenger) StartReceiver(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.receiveLoop(ctx)
	}()

	return nil
}

func (c *Messenger) StopReceiver() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	close(c.stopCh)
	c.wg.Wait()
	c.stopCh = make(chan struct{})
}

func (c *Messenger) receiveLoop(ctx context.Context) {
	for {
		select {
		case <-c.stopCh:
			return
		case <-ctx.Done():
			return
		default:
			log.Debug().Msg("listening for message...")
			msg, err := c.ReceiveMessage()
			if err != nil {
				// Check if this is a graceful shutdown scenario
				if isGracefulShutdownError(err) || ctx.Err() != nil {
					log.Info().Msg("shutting down message listener gracefully")
					return
				}
				log.Error().Err(err).Msg("failed receiving message")
				time.Sleep(DefaultRetryListenerInterval)
				continue
			}

			if msg == nil {
				time.Sleep(DefaultRetryListenerInterval)
				continue
			}

			go c.processMessage(ctx, msg)
		}
	}
}

// TODO: each reply with error, should follow the same pattern
func (c *Messenger) processMessage(ctx context.Context, message *Message) {
	sendErrorReply := func(errorMsg string) {
		if err := c.SendReply(message.ID, message.SrcPK, errorMsg); err != nil {
			log.Error().Err(err).Str("msg_id", message.ID).
				Msg("failed to send error reply")
		}
	}

	// decide which handler group to use based on the topic
	c.mutex.RLock()
	handler, exists := c.receiveHandlers[message.Topic]
	c.mutex.RUnlock()

	if !exists {
		log.Error().Str("topic", message.Topic).
			Msg("no handler registered for topic")
		sendErrorReply(fmt.Sprintf("no handler registered for topic: %s", message.Topic))
		return
	}

	// add twin id to the context for later use
	if c.EnableTwinIdentity {
		twin, err := c.subCon.GetMyceliumTwin(message.SrcPK)
		if err != nil {
			log.Error().Err(err).Str("key", message.SrcPK).Msg("failed to get twin ID from Mycelium public key")
			sendErrorReply(fmt.Sprintf("failed to get twin ID: %v", err))
			return
		}
		ctx = context.WithValue(ctx, TwinIdContextKey, twin)
	}

	response, err := handler(ctx, message)
	if err != nil {
		log.Error().Err(err).Str("msg_id", message.ID).
			Msg("failed to process message")
		sendErrorReply(fmt.Sprintf("failed to process message: %v", err))
		return
	}

	if response != nil {
		if err := c.SendReply(message.ID, message.SrcPK, string(response)); err != nil {
			log.Error().Err(err).Str("msg_id", message.ID).
				Msg("failed to send reply")
		}
	}
}

func (c *Messenger) Close() {
	c.StopReceiver()

	if c.subCon != nil {
		c.subCon.Close()
		c.subCon = nil
	}
}

// isGracefulShutdownError checks if an error is related to graceful shutdown
func isGracefulShutdownError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "context canceled") ||
		strings.Contains(errStr, "context deadline exceeded") ||
		strings.Contains(errStr, "signal: interrupt") ||
		strings.Contains(errStr, "receive cancelled") ||
		strings.Contains(errStr, "receive interrupted")
}
