package rmb

import (
	"context"
	"encoding/base64"
	"fmt"
)

var (
	// ErrFunctionNotFound is an err returned if the handler function is not found
	ErrFunctionNotFound = fmt.Errorf("function not found")
)

// Handler is a handler function type
type Handler func(ctx context.Context, payload []byte) (interface{}, error)

// Middleware is middleware function type
type Middleware func(ctx context.Context, payload []byte) (context.Context, error)

// Router is the router interface
type Router interface {
	WithHandler(route string, handler Handler)
	Subroute(route string) Router
	Use(Middleware)
}

// Client is an rmb abstract client interface.
type Client interface {
	Call(ctx context.Context, twin uint32, fn string, data interface{}, result interface{}) error
}

// Request is an outgoing request struct used to make rpc calls over rmb
type Request struct {
	Version    int      `json:"ver"`
	Reference  string   `json:"ref"`
	Command    string   `json:"cmd"`
	Expiration int      `json:"exp"`
	Data       string   `json:"dat"`
	TwinDest   []uint32 `json:"dst"`
	RetQueue   string   `json:"ret"`
	Schema     string   `json:"shm"`
	Epoch      int64    `json:"now"`
}

// GetPayload returns the payload for a message's data
func (m *Request) GetPayload() ([]byte, error) {
	return base64.StdEncoding.DecodeString(m.Data)
}

// Incoming request that need to be handled by servers
type Incoming struct {
	Version    int    `json:"ver"`
	Reference  string `json:"ref"`
	Command    string `json:"cmd"`
	Expiration int    `json:"exp"`
	Data       string `json:"dat"`
	TwinSrc    string `json:"src"`
	RetQueue   string `json:"ret"`
	Schema     string `json:"shm"`
	Epoch      int64  `json:"now"`
}

// GetPayload returns the payload for a message's data
func (m *Incoming) GetPayload() ([]byte, error) {
	return base64.StdEncoding.DecodeString(m.Data)
}

type Error struct {
	Code    uint32 `json:"code"`
	Message string `json:"message"`
}

type OutgoingResponse struct {
	Version   int    `json:"ver"`
	Reference string `json:"ref"`
	Data      string `json:"dat"`
	TwinDest  string `json:"dst"`
	Schema    string `json:"shm"`
	Epoch     int64  `json:"now"`
	Error     *Error `json:"err,omitempty"`
}

type IncomingResponse struct {
	Version   int    `json:"ver"`
	Reference string `json:"ref"`
	Data      string `json:"dat"`
	TwinSrc   string `json:"src"`
	Schema    string `json:"shm"`
	Epoch     int64  `json:"now"`
	Error     *Error `json:"err,omitempty"`
}
