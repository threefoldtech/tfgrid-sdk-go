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
	BinaryPath     string
	APIAddress     string
	Timeout        int
	MnemonicPhrase string

	AutoUpdateTwin bool // TODO: manage twin identity
	subCon         *substrate.Substrate
	identity       substrate.Identity

	receiveHandlers map[string]MessageHandlerFunc
	stopCh          chan struct{}
	wg              sync.WaitGroup
	mutex           sync.RWMutex
}

// MessengerOpt is a function that configures a Client
type MessengerOpt func(*Messenger)

// WithMnemonicPhrase sets the mnemonic phrase for the client,
func WithMnemonicPhrase(mnemonicPhrase string) MessengerOpt {
	return func(c *Messenger) {
		c.MnemonicPhrase = mnemonicPhrase
	}
}

func WithIdentity(identity substrate.Identity) MessengerOpt {
	return func(c *Messenger) {
		c.identity = identity
	}
}

// WithAutoUpdateTwin enables or disables automatic twin IP update
func WithAutoUpdateTwin(autoUpdate bool) MessengerOpt {
	return func(c *Messenger) {
		c.AutoUpdateTwin = autoUpdate
	}
}

func WithTimeout(timeout int) MessengerOpt {
	return func(c *Messenger) {
		c.Timeout = timeout
	}
}

// NewMessenger creates a new mycelium client with the given options
func NewMessenger(binaryPath string, defaultTimeout int, man substrate.Manager, opts ...MessengerOpt) (*Messenger, error) {
	if binaryPath == "" {
		binaryPath = DefaultMessengerBinary
	}

	if defaultTimeout <= 0 {
		defaultTimeout = DefaultTimeout
	}

	var err error
	messenger := &Messenger{
		BinaryPath:      binaryPath,
		Timeout:         defaultTimeout,
		APIAddress:      DefaultAPIAddress,
		AutoUpdateTwin:  true,
		receiveHandlers: make(map[string]MessageHandlerFunc),
		stopCh:          make(chan struct{}),
	}

	for _, opt := range opts {
		opt(messenger)
	}

	if !messenger.AutoUpdateTwin {
		return messenger, nil
	}

	// TODO: all args can be optional and add validation for related args

	if messenger.identity == nil {
		if messenger.MnemonicPhrase == "" {
			return nil, fmt.Errorf("either an identity or mnemonic phrase is required to create a new messenger")
		}

		messenger.identity, err = substrate.NewIdentityFromSr25519Phrase(messenger.MnemonicPhrase)
		if err != nil {
			return nil, fmt.Errorf("failed to create identity from mnemonic phrase: %w", err)
		}
	}

	subCon, err := man.Substrate()
	if err != nil {
		return nil, fmt.Errorf("failed to get substrate connection: %w", err)

	}
	messenger.subCon = subCon

	if err := messenger.UpdateTwinWithMyceliumPubkey(context.Background()); err != nil {
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

// TODO: should add topic rpc so it can get the right handler
func (c *Messenger) SendMessage(destination, payload string, topic string, waitForReply bool, timeout int) (*Message, error) {
	// TODO: run on network namespace? WithNetNs
	log.Debug().Str("destination", destination).
		Str("payload", payload).
		Msg("sending message")
	args := []string{"message", "send", destination, payload}

	if waitForReply {
		// TODO: is there a wait option on the http server, yes
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

	log.Debug().Str("source", msg.SrcPK).
		Str("payload", msg.Payload).
		Msg("received reply")
	return &msg, nil
}

func (c *Messenger) SendReply(originalMessageID, destination, payload string) error {
	log.Debug().Str("destination", destination).
		Str("originalMessageID", originalMessageID).
		Str("payload", payload).
		Msg("sending reply via API")

	encodedPayload := base64.StdEncoding.EncodeToString([]byte(payload))
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

	log.Debug().Str("source", msg.SrcPK).
		Str("payload", msg.Payload).
		Msg("received message")
	return &msg, nil
}

func (c *Messenger) StartReceiver(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// TODO: multiple workers?
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
	// Helper function to send error reply
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
	if c.AutoUpdateTwin {
		twin, err := c.subCon.GetTwinByMyceliumPK(message.SrcPK)
		if err != nil {
			log.Error().Err(err).Str("key", message.SrcPK).Msg("failed to get twin ID from Mycelium public key")
			sendErrorReply(fmt.Sprintf("failed to get twin ID: %v", err))
			return
		}
		ctx = context.WithValue(ctx, TwinIdContextKey, twin.ID)
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
