package peer

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"runtime/debug"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
)

const (
	pongWait     = 40 * time.Second
	pingInterval = 20 * time.Second
)

var errTimeout = fmt.Errorf("connection timeout")

// Explicit errors for common exit reasons to avoid hardcoded strings.
var (
	ErrConnectionStalling = errors.New("connection stalling")
	ErrLocalCanceled      = errors.New("local canceled (reader/transport error)")
	ErrContextCanceled    = errors.New("context canceled")
	ErrInvalidMessageType = errors.New("invalid message type")
)

// InnerConnection holds the required state to create a self healing websocket connection to the rmb relay.
type InnerConnection struct {
	twinID    uint32
	session   string
	identity  substrate.Identity
	url       string
	writer    chan send
	connected int32 // 1 when loop is active with an open websocket
}

type send struct {
	data []byte
	err  chan error
}

func (s *send) reply(ctx context.Context, err error) error {
	defer func() {
		if r := recover(); r != nil {
			log.Debug().Msgf("recovered from panic: %v", r)
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case s.err <- err:
		return err
	}
}

// Reader is a channel that receives incoming messages
type Reader <-chan []byte

func (r Reader) Read() []byte {
	return <-r
}

// NewConnection creates a new InnerConnection instance
func NewConnection(identity substrate.Identity, url string, session string, twinID uint32) InnerConnection {
	return InnerConnection{
		twinID:   twinID,
		identity: identity,
		url:      url,
		session:  session,
		writer:   make(chan send), // TODO: it should be buffered
	}
}

func (c *InnerConnection) reader(ctx context.Context, cancel context.CancelFunc, con *websocket.Conn, reader chan []byte) {
	var exitReason string
	var exitErr error
	defer func() {
		if r := recover(); r != nil {
			log.Error().Str("url", c.url).Interface("panic", r).Bytes("stack", debug.Stack()).Msg("relay reader panic recovered")
		} else {
			log.Debug().Str("url", c.url).Str("reason", exitReason).Err(exitErr).Msg("relay reader exited")
		}
	}()

	for {
		typ, data, err := con.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err) || websocket.IsUnexpectedCloseError(err) || err == io.EOF {
				exitReason = "close"
				exitErr = err
			} else {
				exitReason = "read error"
				exitErr = err
			}
			cancel()
			return
		}

		if typ != websocket.BinaryMessage {
			exitReason = "invalid message type"
			exitErr = ErrInvalidMessageType
			// signal the supervisor loop() to tear down
			cancel()
			return
		}

		select {
		case <-ctx.Done():
			exitReason = ErrContextCanceled.Error()
			exitErr = ctx.Err()
			return
		case reader <- data:
		}
	}
}

func (c *InnerConnection) send(ctx context.Context, data []byte) error {
	resp := make(chan error)
	defer close(resp)

	s := send{
		data: data,
		err:  resp,
	}

	select {
	case c.writer <- s:
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(5 * time.Second):
		return errTimeout
	}

	select {
	case err := <-resp:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *InnerConnection) loop(ctx context.Context, con *websocket.Conn, output chan []byte) error {
	var exitReason string
	var exitErr error

	// Attempt a graceful close handshake on exit; log either panic or normal exit, not both.
	defer func() {
		if r := recover(); r != nil {
			log.Error().Str("url", c.url).Interface("panic", r).Bytes("stack", debug.Stack()).Msg("relay loop panic recovered")
		} else {
			log.Debug().Str("url", c.url).Str("reason", exitReason).Err(exitErr).Msg("relay loop exited")
		}

		deadline := time.Now().Add(1 * time.Second)
		_ = con.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), deadline)
		// give the server a moment to respond and drain
		_ = con.SetReadDeadline(deadline)
		for {
			if _, _, err := con.ReadMessage(); err != nil {
				break
			}
		}
		_ = con.Close()
	}()

	local, cancel := context.WithCancel(ctx)
	defer cancel()
	atomic.StoreInt32(&c.connected, 1)
	defer atomic.StoreInt32(&c.connected, 0)

	pong := make(chan byte)
	con.SetPongHandler(func(appData string) error {
		select {
		case pong <- 1:
		default:
		}
		return nil
	})

	outputCh := make(chan []byte) // TODO: it should be buffered
	defer close(outputCh)

	go c.reader(local, cancel, con, outputCh)

	lastPong := time.Now()
	for {
		select {
		case <-ctx.Done():
			exitReason = ErrContextCanceled.Error()
			exitErr = ctx.Err()
			return ctx.Err()
		case <-local.Done():
			exitReason = ErrLocalCanceled.Error()
			exitErr = ErrLocalCanceled
			return nil // error happened with the connection, return nil to try again
		case data := <-outputCh:
			// TODO: can we protect the loop from stalling by using a short timeout with logging
			output <- data
			lastPong = time.Now()
		case sent := <-c.writer:
			// Write the message to the websocket transport.
			err := con.WriteMessage(websocket.BinaryMessage, sent.data)
			// Try to notify the sender about the write result. If notification fails,
			// log and continue; it's not a transport failure.
			if replyErr := sent.reply(ctx, err); replyErr != nil {
				log.Warn().Err(replyErr).Msg("failed to deliver write result to sender")
			}
			// On actual write error, tear down to trigger reconnect.
			if err != nil {
				exitReason = "write error"
				exitErr = err
				return err
			}
		case <-pong:
			lastPong = time.Now()
		case <-time.After(pingInterval):
			if err := con.WriteControl(websocket.PingMessage, nil, time.Now().Add(10*time.Second)); err != nil {
				exitReason = "ping write error"
				exitErr = err
				return err
			}

			if time.Since(lastPong) > pongWait {
				exitReason = ErrConnectionStalling.Error()
				exitErr = ErrConnectionStalling
				return ErrConnectionStalling
			}
		}
	}
}

// Start initiates the websocket connection
func (c *InnerConnection) Start(ctx context.Context, output chan []byte) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error().Str("url", c.url).Interface("panic", r).Bytes("stack", debug.Stack()).Msg("relay start worker panic recovered")
			}
		}()
		// Do not close output or c.writer here:
		// - output is owned by the Peer and may be shared among connections.
		// - c.writer may still be used by senders racing with shutdown.
		for {
			if ctx.Err() != nil {
				log.Debug().Str("url", c.url).Err(ctx.Err()).Msg("relay worker stopping: context canceled")
				break
			}
			err := c.listenAndServe(ctx, output)
			if err == context.Canceled {
				log.Debug().Str("url", c.url).Msg("relay worker stopping: context canceled")
				break
			} else if err != nil {
				log.Error().Err(err).Str("url", c.url).Msg("relay connection error")
			}

			// Backoff or exit promptly on cancellation
			select {
			case <-ctx.Done():
				log.Debug().Str("url", c.url).Err(ctx.Err()).Msg("relay worker stopping: context canceled")
				return
			case <-time.After(2 * time.Second):
			}
		}
	}()
}

// listenAndServe creates the websocket connection, and if successful, listens for and serves incoming and outgoing messages.
func (c *InnerConnection) listenAndServe(ctx context.Context, output chan []byte) error {
	con, err := c.connect()
	if err != nil {
		return errors.Wrap(err, "failed to reconnect")
	}

	return c.loop(ctx, con, output)
}

// IsConnected reports whether the websocket loop is currently active.
func (c *InnerConnection) IsConnected() bool {
	return atomic.LoadInt32(&c.connected) == 1
}

// TryConnect attempts to establish a connection and returns true if successful, false otherwise
func (c *InnerConnection) TryConnect() bool {
	con, err := c.connect()
	if err != nil {
		log.Debug().Err(err).Str("url", c.url).Msg("failed to connect to relay")
		return false
	}

	con.Close()
	return true
}

func (c *InnerConnection) connect() (*websocket.Conn, error) {
	token, err := NewJWT(c.identity, c.twinID, c.session, 60)
	if err != nil {
		return nil, errors.Wrap(err, "could not create new jwt")
	}

	relayURL := fmt.Sprintf("%s?%s", c.url, token)
	log.Debug().Str("url", c.url).Msg("connecting")

	dialer := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
		Proxy:            http.ProxyFromEnvironment,
	}

	con, resp, err := dialer.Dial(relayURL, nil)
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
	log.Debug().Str("url", c.url).Msg("connected")

	return con, nil
}
