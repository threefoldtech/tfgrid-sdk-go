package peer

import (
	"context"
	"fmt"
	"io"
	"net/http"
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

// InnerConnection holds the required state to create a self healing websocket connection to the rmb relay.
type InnerConnection struct {
	twinID   uint32
	session  string
	identity substrate.Identity
	url      string
}

// Writer is a channel that sends outgoing messages
type Writer chan<- []byte

func (w Writer) Write(data []byte) {
	w <- data
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
	}
}

func (c *InnerConnection) reader(ctx context.Context, cancel context.CancelFunc, con *websocket.Conn, reader chan []byte) {
	for {
		typ, data, err := con.ReadMessage()
		if err != nil {
			log.Error().Err(err).Msg("failed to read message")
			cancel()
			return
		}

		if typ != websocket.BinaryMessage {
			log.Error().Msg("invalid message type received")
			cancel()
			return
		}

		select {
		case <-ctx.Done():
			return
		case reader <- data:
		}
	}
}

func (c *InnerConnection) loop(ctx context.Context, con *websocket.Conn, output, input chan []byte) error {
	defer con.Close()

	local, cancel := context.WithCancel(ctx)
	defer cancel()

	pong := make(chan byte)
	con.SetPongHandler(func(appData string) error {
		select {
		case pong <- 1:
		default:
		}
		return nil
	})

	go c.reader(local, cancel, con, output)

	lastPong := time.Now()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-local.Done():
			// error happened with the connection
			// we return nil to try again
			return nil
		case data := <-input:
			if err := con.WriteMessage(websocket.BinaryMessage, data); err != nil {
				return err
			}
		case <-pong:
			lastPong = time.Now()
		case <-time.After(pingInterval):
			if err := con.WriteControl(websocket.PingMessage, nil, time.Now().Add(10*time.Second)); err != nil {
				return err
			}

			if time.Since(lastPong) > pongWait {
				return fmt.Errorf("connection stalling")
			}
		}
	}
}

// Start initiates the websocket connection
func (c *InnerConnection) Start(ctx context.Context) (Reader, Writer) {
	output := make(chan []byte)
	input := make(chan []byte)

	go func() {
		defer close(output)
		defer close(input)
		for {
			err := c.listenAndServe(ctx, output, input)
			if err == context.Canceled {
				break
			} else if err != nil {
				log.Error().Err(err)
			}

			<-time.After(2 * time.Second)
		}
	}()

	return output, input
}

// listenAndServe creates the websocket connection, and if successful, listens for and serves incoming and outgoing messages.
func (c *InnerConnection) listenAndServe(ctx context.Context, output chan []byte, input chan []byte) error {
	con, err := c.connect()
	if err != nil {
		return errors.Wrap(err, "failed to reconnect")
	}

	return c.loop(ctx, con, output, input)
}

func (c *InnerConnection) connect() (*websocket.Conn, error) {
	token, err := NewJWT(c.identity, c.twinID, c.session, 60)
	if err != nil {
		return nil, errors.Wrap(err, "could not create new jwt")
	}

	relayURL := fmt.Sprintf("%s?%s", c.url, token)

	con, resp, err := websocket.DefaultDialer.Dial(relayURL, nil)
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

	return con, nil
}
