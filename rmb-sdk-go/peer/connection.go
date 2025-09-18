package peer

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"runtime/debug"
	"sync"
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
	twinID           uint32
	session          string
	identity         substrate.Identity
	url              string
	writer           chan send
	connected        int32 // 1 when loop is active with an open websocket
	connectionsTotal int64 // total successful connections (initial + reconnects)
	observer         ConnObserver
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
		twinID:           twinID,
		identity:         identity,
		url:              url,
		session:          session,
		writer:           make(chan send, 64), // buffered to smooth short spikes from concurrent senders (bounded for backpressure)
		observer:         NewLogObserver(url),
		connectionsTotal: 0,
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
			// If we're shutting down, prefer a clean context-canceled reason over transport error noise
			if ctx.Err() != nil {
				exitReason = ErrContextCanceled.Error()
				exitErr = ctx.Err()
			} else if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				// Map read deadline timeouts to stalling for clearer diagnostics
				exitReason = ErrConnectionStalling.Error()
				exitErr = ErrConnectionStalling
			} else if websocket.IsCloseError(err) || websocket.IsUnexpectedCloseError(err) || err == io.EOF {
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

		// Refresh the read deadline on every successfully read frame to keep the connection alive.
		_ = con.SetReadDeadline(time.Now().Add(pongWait))

		// Backpressure-aware send into reader channel: never drop.
		{
			delivered := false
			for !delivered {
				select {
				case <-ctx.Done():
					exitReason = ErrContextCanceled.Error()
					exitErr = ctx.Err()
					return
				case reader <- data:
					// delivered
					delivered = true
				case <-time.After(100 * time.Millisecond):
					c.observer.ReaderBackpressure(c.url, len(reader), cap(reader))
				}
			}
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
	var stats ConnStats

	// Attempt a graceful close handshake on exit
	defer func() {
		if r := recover(); r != nil {
			log.Error().Str("url", c.url).Interface("panic", r).Bytes("stack", debug.Stack()).Msg("relay loop panic recovered")
		} else {
			log.Debug().Str("url", c.url).Str("reason", exitReason).Err(exitErr).Msg("relay loop exited")
		}
		recons := c.reconnections()
		stats.Exit(c.observer, c.url, exitReason, exitErr, recons)
		// Gracefully close the websocket: send a normal close frame, then close the connection.
		deadline := time.Now().Add(1 * time.Second)
		_ = con.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			deadline,
		)
		// Set a read deadline to unblock the reader goroutine and avoid hangs.
		_ = con.SetReadDeadline(deadline)
		// Do NOT read here to drain frames; the dedicated reader goroutine owns all reads.
		// Concurrent reads on the same *websocket.Conn cause data races.
		_ = con.Close()
	}()
	local, cancel := context.WithCancel(ctx)
	defer cancel()
	atomic.StoreInt32(&c.connected, 1)
	defer atomic.StoreInt32(&c.connected, 0)

	// Set initial read deadline and extend it on every pong to detect stalled connections.
	_ = con.SetReadDeadline(time.Now().Add(pongWait))
	con.SetPongHandler(func(appData string) error {
		// Extend read deadline on pong; the reader goroutine will time out if no frames arrive.
		_ = con.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	outputCh := make(chan []byte, 1024)

	atomic.AddInt64(&c.connectionsTotal, 1)

	go c.reader(local, cancel, con, outputCh)

	summaryTicker := time.NewTicker(time.Minute)
	defer summaryTicker.Stop()
	// Use a ticker for ping cadence (avoids repeated allocations and jitter of time.After)
	pingTicker := time.NewTicker(pingInterval)
	defer pingTicker.Stop()
	logSummary := func() {
		stats.Summary(c.observer, c.url, c.reconnections())
	}
	for {
		select {
		case <-summaryTicker.C:
			logSummary()
		case <-ctx.Done():
			exitReason = ErrContextCanceled.Error()
			exitErr = ctx.Err()
			return ctx.Err()
		case <-local.Done():
			exitReason = ErrLocalCanceled.Error()
			exitErr = ErrLocalCanceled
			return nil
		case data, ok := <-outputCh:
			if !ok {
				exitReason = ErrLocalCanceled.Error()
				exitErr = ErrLocalCanceled
				return nil
			}
			delivered := false
			for !delivered {
				select {
				case output <- data:
					stats.OnDelivered()
					delivered = true
				case <-time.After(100 * time.Millisecond):
					c.observer.OutputBackpressure(c.url, len(output), cap(output), len(outputCh), cap(outputCh), len(c.writer), cap(c.writer))
				case <-ctx.Done():
					exitReason = ErrContextCanceled.Error()
					exitErr = ctx.Err()
					return ctx.Err()
				}
			}
		case sent := <-c.writer:
			writeStart := time.Now()
			err := con.WriteMessage(websocket.BinaryMessage, sent.data)
			writeDur := time.Since(writeStart)
			stats.OnWriteResult(writeDur, err)
			// Spike diagnostics: warn (sampled) if write is slow or writer queue is saturated
			c.observer.MaybeWriteSpike(c.url, writeDur, len(c.writer), cap(c.writer), len(output), cap(output), len(outputCh), cap(outputCh))
			// Try to notify the sender about the write result. If notification fails,
			if replyErr := sent.reply(ctx, err); replyErr != nil {
				log.Warn().Err(replyErr).Msg("failed to deliver write result to sender")
			}
			// On actual write error, tear down to trigger reconnect.
			if err != nil {
				exitReason = "write error"
				exitErr = err
				return err
			}
		case <-pingTicker.C:
			if err := con.WriteControl(websocket.PingMessage, nil, time.Now().Add(10*time.Second)); err != nil {
				exitReason = "ping write error"
				exitErr = err
				return err
			}
		}
	}
}

// reconnections returns the number of reconnections that happened after the initial successful connection.
// It is computed as max(connectionsTotal-1, 0).
func (c *InnerConnection) reconnections() int64 {
	total := atomic.LoadInt64(&c.connectionsTotal)
	if total <= 1 {
		return 0
	}
	return total - 1
}

// Start initiates the websocket connection
func (c *InnerConnection) Start(ctx context.Context, output chan []byte, wg *sync.WaitGroup) {
	if wg != nil {
		wg.Add(1)
	}

	// Start initiates the websocket connection.
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error().Str("url", c.url).Interface("panic", r).Bytes("stack", debug.Stack()).Msg("relay start worker panic recovered")
			}
			if wg != nil {
				wg.Done()
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
		HandshakeTimeout: 10 * time.Second,
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
