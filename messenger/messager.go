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

// Context key for twin ID
type twinIDCtx struct{}

var TwinIDContextKey = twinIDCtx{}

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

	// Substrate connection for twin verification
	substrateConn   *substrate.Substrate
	twinKeyProvider TwinKeyProvider

	receiveHandlers map[string]MessageHandlerFunc
	stopCh          chan struct{}
	wg              sync.WaitGroup
	mutex           sync.RWMutex
}

// MessengerOpt is a function that configures a Client
type MessengerOpt func(*Messenger)

// WithChain sets the substrate connection for blockchain verification
func WithChain(sub *substrate.Substrate) MessengerOpt {
	return func(c *Messenger) {
		c.substrateConn = sub
		c.twinKeyProvider = NewTFChainKeyProvider(sub)
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
		BinaryPath:      DefaultMessengerBinary,
		Timeout:         DefaultTimeout,
		APIAddress:      DefaultAPIAddress,
		receiveHandlers: make(map[string]MessageHandlerFunc),
		stopCh:          make(chan struct{}),
	}

	for _, opt := range opts {
		opt(messenger)
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

// SendSignedMessage sends a message with cryptographic signature verification
func (c *Messenger) SendSignedMessage(destination, payload string, topic string, twinID uint32, identity substrate.Identity, waitForReply bool, timeout int) (*Message, error) {
	signedMsg, err := CreateSignedMessage(twinID, payload, identity)
	if err != nil {
		return nil, fmt.Errorf("failed to create signed message: %w", err)
	}

	signedPayload, err := json.Marshal(signedMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal signed message: %w", err)
	}

	return c.SendMessage(destination, string(signedPayload), topic, waitForReply, timeout)
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

	// TODO: both client/server should use full signed msgs, client sign before send, server verify, server sign before reply, client verify
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
		// TODO: should be part of SEND
		if err := c.SendReply(message.ID, message.SrcPK, errorMsg); err != nil {
			log.Error().Err(err).Str("msg_id", message.ID).
				Msg("failed to send error reply")
		}
	}

	// Verify signature against blockchain if key provider is available
	var twinID uint32
	var originalMessage string
	var err error

	if c.twinKeyProvider != nil {
		// Use secure blockchain-based verification
		twinID, originalMessage, err = ValidateAndExtractMessage(message.Payload, c.twinKeyProvider)
		if err != nil {
			log.Error().Err(err).Str("msg_id", message.ID).
				Msg("TFChain signature verification failed")
			sendErrorReply(fmt.Sprintf("signature verification failed: %v", err))
			return
		}
	} else {
		log.Warn().Str("msg_id", message.ID).
			Msg("no TFChain connection available - message not verified against blockchain")
		// For backward compatibility, allow unverified messages with warning
		originalMessage = message.Payload
		twinID = 0 // Unknown twin

		return
	}

	log.Debug().
		Uint32("twin_id", twinID).
		Str("msg_id", message.ID).
		Msg("signature verification successful")

	verifiedMessage := &Message{
		ID:      message.ID,
		Topic:   message.Topic,
		SrcIP:   message.SrcIP,
		SrcPK:   message.SrcPK,
		DstIP:   message.DstIP,
		DstPK:   message.DstPK,
		Payload: originalMessage,
	}

	ctx = context.WithValue(ctx, TwinIDContextKey, twinID)

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

	response, err := handler(ctx, verifiedMessage)
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

	// Close substrate connection if available
	if c.substrateConn != nil {
		c.substrateConn.Close()
		c.substrateConn = nil
		c.twinKeyProvider = nil
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
