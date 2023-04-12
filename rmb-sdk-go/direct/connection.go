package direct

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/substrate-client"
)

const (
	PongWait     = 40 * time.Second
	PingInterval = 20 * time.Second
)

type InnerConnection struct {
	twinID   uint32
	session  string
	identity substrate.Identity
	url      string
}

type Writer chan<- []byte

func (w Writer) Write(data []byte) {
	w <- data
}

type Reader <-chan []byte

func (r Reader) Read() []byte {
	return <-r
}

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
		case <-local.Done():
			return local.Err()
		case data := <-input:
			if err := con.WriteMessage(websocket.BinaryMessage, data); err != nil {
				return err
			}
		case <-pong:
			lastPong = time.Now()
		case <-time.After(PingInterval):
			if err := con.WriteControl(websocket.PingMessage, nil, time.Now().Add(10*time.Second)); err != nil {
				return err
			}

			if time.Since(lastPong) > PongWait {
				return fmt.Errorf("connection stalling")
			}
		}
	}
}

func (c *InnerConnection) Start(ctx context.Context) (Reader, Writer) {
	output := make(chan []byte)
	input := make(chan []byte)

	go func() {
		defer close(output)
		defer close(input)
		for {
			con, err := c.connect()
			if err != nil {
				log.Error().Err(err).Msg("failed to reconnect")
				continue
			}

			err = c.loop(ctx, con, output, input)
			if err == context.Canceled {
				break
			} else if err != nil {
				log.Error().Err(err)
			}
		}
	}()

	return output, input
}

func (c *InnerConnection) connect() (*websocket.Conn, error) {
	token, err := NewJWT(c.identity, c.twinID, c.session, 60)
	if err != nil {
		return nil, errors.Wrap(err, "could not create new jwt")
	}

	relayUrl := fmt.Sprintf("%s?%s", c.url, token)

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

	return con, nil
}
