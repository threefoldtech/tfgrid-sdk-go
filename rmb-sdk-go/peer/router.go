package peer

import (
	"context"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer/types"
)

var (
	// ErrFunctionNotFound is an err returned if the handler function is not found
	ErrFunctionNotFound = fmt.Errorf("function is not found")
)

// twinKeyID is where the twin key is stored
type twinKeyID struct{}

// envelopeKey is where the envelope is stored
type envelopeKey struct{}

// Handler is a handler function type
type HandlerFunc func(ctx context.Context, payload []byte) (interface{}, error)

// Middleware is middleware function type
type Middleware func(ctx context.Context, payload []byte) (context.Context, error)

type Router struct {
	handlers map[string]HandlerFunc
	routes   map[string]*Router
	mw       []Middleware
}

func NewRouter() *Router {
	return &Router{
		handlers: make(map[string]HandlerFunc),
		routes:   make(map[string]*Router),
	}
}

// SubRoute add a route prefix to include more sub routes with handler from it
func (r *Router) SubRoute(prefix string) *Router {
	if strings.Contains(prefix, ".") {
		panic("invalid subrouter prefix should not have '.'")
	}

	sub, ok := r.routes[prefix]
	if ok {
		return sub
	}

	router := NewRouter()
	r.routes[prefix] = router
	return router
}

// WithHandler adds a handler function to a router sub command
func (r *Router) WithHandler(subCommand string, handler HandlerFunc) {
	if _, ok := r.handlers[subCommand]; ok {
		panic("handler function is already registered")
	}

	r.handlers[subCommand] = handler
}

// Use adds a middleware to the router
func (r *Router) Use(mw Middleware) {
	r.mw = append(r.mw, mw)
}

func (r *Router) Serve(ctx context.Context, peer Peer, env *types.Envelope, err error) {
	if err != nil {
		log.Error().Err(err).Msg("bad request")
		return
	}

	handlerCtx := context.WithValue(ctx, twinKeyID{}, env.Source.Twin)
	handlerCtx = context.WithValue(handlerCtx, envelopeKey{}, env)

	go func() {
		// parse and call request
		req := env.GetRequest()
		if req == nil {
			log.Error().Msg("received a non request envelope")
			return
		}

		if env.Schema == nil || *env.Schema != rmb.DefaultSchema {
			log.Error().Msgf("invalid schema received expected '%s'", rmb.DefaultSchema)
			return
		}

		payload, ok := env.Payload.(*types.Envelope_Plain)
		if !ok {
			// the peer makes sure at this moment, the payload is always Plain
			// but we need this just in case so the service does not panic.
			log.Warn().Msg("payload is not in plain format")
			return
		}

		cmd := env.GetRequest().Command

		response, err := r.call(handlerCtx, cmd, payload.Plain)

		// send response
		if err := peer.SendResponse(ctx, env.Uid, env.Source.Twin, env.Source.Connection, err, response); err != nil {
			log.Error().Err(err).Msgf("failed to send response to twin id '%d'", env.Destination.Twin)
		}
	}()
}

func (r *Router) call(ctx context.Context, route string, payload []byte) (result interface{}, err error) {
	for _, mw := range r.mw {
		ctx, err = mw(ctx, payload)
		if err != nil {
			return nil, err
		}
	}

	handler, ok := r.handlers[route]
	if ok {
		defer func() {
			if rec := recover(); rec != nil {
				err = fmt.Errorf("handler panicked with: %s", rec)
			}
		}()

		result, err = handler(ctx, payload)
		return
	}

	parts := strings.SplitN(route, ".", 2)

	key := parts[0]
	var subRoute string
	if len(parts) == 2 {
		subRoute = parts[1]
	}

	router, ok := r.routes[key]
	if !ok {
		return nil, ErrFunctionNotFound
	}

	return router.call(ctx, subRoute, payload)
}

// GetTwinID returns the twin id from context.
func GetTwinID(ctx context.Context) uint32 {
	twin, ok := ctx.Value(twinKeyID{}).(uint32)
	if !ok {
		panic("failed to load twin id from context")
	}

	return twin
}

// GetEnvelope gets an envelope from the context, panics if it's not there
func GetEnvelope(ctx context.Context) *types.Envelope {
	envelope, ok := ctx.Value(envelopeKey{}).(*types.Envelope)
	if !ok {
		panic("failed to load envelope from context")
	}

	return envelope
}
