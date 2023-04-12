package rmb

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

const (
	numWorkers = 5
)

// twinKeyID is where the twin key is stored
type twinKeyID struct{}

// messageKey is where the original message is stored
type messageKey struct{}

type messageBusSubrouter struct {
	handlers map[string]Handler
	sub      map[string]*messageBusSubrouter
	mw       []Middleware
}

func newSubRouter() messageBusSubrouter {
	return messageBusSubrouter{
		handlers: make(map[string]Handler),
		sub:      make(map[string]*messageBusSubrouter),
	}
}

func (m *messageBusSubrouter) call(ctx context.Context, route string, payload []byte) (result interface{}, err error) {
	for _, mw := range m.mw {
		ctx, err = mw(ctx, payload)
		if err != nil {
			return nil, err
		}
	}

	handler, ok := m.handlers[route]
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
	var subroute string
	if len(parts) == 2 {
		subroute = parts[1]
	}

	router, ok := m.sub[key]
	if !ok {
		return nil, ErrFunctionNotFound
	}

	return router.call(ctx, subroute, payload)
}

func (m *messageBusSubrouter) Use(mw Middleware) {
	m.mw = append(m.mw, mw)
}

func (m *messageBusSubrouter) Subroute(prefix string) Router {
	//r.handle('abc.def.fun', handler)
	// sub = r.handle(xyz)
	// sub.use(middle)
	// sub.handle('func', handler) // xyz.func
	if strings.Contains(prefix, ".") {
		panic("invalid subrouter prefix should not have '.'")
	}

	sub, ok := m.sub[prefix]
	if ok {
		return sub
	}

	r := newSubRouter()
	m.sub[prefix] = &r
	return &r
}

// WithHandler adds a topic handler to the messagebus
func (m *messageBusSubrouter) WithHandler(topic string, handler Handler) {
	if _, ok := m.handlers[topic]; ok {
		panic("handler already registered")
	}

	m.handlers[topic] = handler
}

func (m *messageBusSubrouter) getTopics(prefix string, l *[]string) {
	for r := range m.handlers {
		if len(prefix) != 0 {
			r = fmt.Sprintf("%s.%s", prefix, r)
		}
		*l = append(*l, r)
	}

	for r, sub := range m.sub {
		if len(prefix) != 0 {
			r = fmt.Sprintf("%s.%s", prefix, r)
		}
		sub.getTopics(r, l)
	}

}

// DefaultRouter implements Router interface. It then can be used to register handlers
// to quickly implement servers that are callable over RMB.
type DefaultRouter struct {
	messageBusSubrouter
	pool *redis.Pool
}

// NewRouter creates a new default router. with the local redis address.
// Normally you want to do NewRouter(DefaultAddress)
func NewRouter(address string) (*DefaultRouter, error) {
	pool, err := newRedisPool(address)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to connect to %s", address)
	}

	return &DefaultRouter{
		pool:                pool,
		messageBusSubrouter: newSubRouter(),
	}, nil
}

// Handlers return full name of all registered handlers
func (m *DefaultRouter) Handlers() []string {
	topics := make([]string, 0)
	m.getTopics("", &topics)

	return topics
}

func (m *DefaultRouter) getOne(args redis.Args) ([][]byte, error) {
	con := m.pool.Get()
	defer con.Close()

	data, err := redis.ByteSlices(con.Do("BRPOP", args...))
	if err != nil && err != redis.ErrNil {
		return nil, err
	}

	if err == redis.ErrNil || data == nil {
		//timeout, just try again immediately
		return nil, redis.ErrNil
	}

	return data, nil
}

// Run runs listeners to the configured handlers
// and will trigger the handlers in the case an event comes in
func (m *DefaultRouter) Run(ctx context.Context) error {
	con := m.pool.Get()
	defer con.Close()

	topics := m.Handlers()
	for i, topic := range topics {
		topics[i] = "msgbus." + topic
	}

	jobs := make(chan Incoming, numWorkers)
	for i := 1; i <= numWorkers; i++ {
		go m.worker(ctx, jobs)
	}

	args := redis.Args{}.AddFlat(topics).Add(3)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		data, err := m.getOne(args)

		if err == redis.ErrNil {
			continue
		} else if err != nil {
			log.Err(err).Msg("failed to read from system local messagebus, retry in 2 seconds")
			<-time.After(2 * time.Second)
			continue
		}

		var message Incoming
		err = json.Unmarshal(data[1], &message)
		if err != nil {
			log.Error().Err(err).Msg("failed to unmarshal message")
			continue
		}

		select {
		case jobs <- message:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (m *DefaultRouter) worker(ctx context.Context, jobs chan Incoming) {
	for {
		select {
		case <-ctx.Done():
			return
		case message := <-jobs:
			bytes, err := message.GetPayload()
			if err != nil {
				log.Err(err).Msg("err while parsing payload reply")
			}

			var twinID uint32
			if _, err := fmt.Sscanf(message.TwinSrc, "%d", &twinID); err != nil {
				// while this should not happen, we still log and continue with the processing
				// the twin id hence will be 0
				log.Error().Err(err).Msg("failed to extract twin source from message!")
			}

			requestCtx := context.WithValue(ctx, twinKeyID{}, twinID)
			requestCtx = context.WithValue(requestCtx, messageKey{}, message)

			data, err := m.call(requestCtx, message.Command, bytes)

			response := OutgoingResponse{
				Version:   message.Version,
				Reference: message.Reference,
				TwinDest:  message.TwinSrc,
				Schema:    message.Schema,
				Epoch:     time.Now().Unix(),
			}

			if err != nil {
				log.Debug().
					Err(err).
					Str("twin", message.TwinSrc).
					Str("handler", message.Command).
					Msg("error while handling job")
				// TODO: create an error object
				response.Error = &Error{
					Code:    255, //client error
					Message: err.Error(),
				}
			}

			err = m.sendReply(message.RetQueue, response, data)
			if err != nil {
				log.Err(err).Msg("err while sending reply")
			}
		}
	}
}

// GetTwinID returns the twin id from context.
func GetTwinID(ctx context.Context) uint32 {
	twin, ok := ctx.Value(twinKeyID{}).(uint32)
	if !ok {
		panic("failed to load twind from context")
	}

	return twin
}

// GetRequest gets a message from the context, panics if it's not there
func GetRequest(ctx context.Context) Incoming {
	message, ok := ctx.Value(messageKey{}).(Incoming)
	if !ok {
		panic("failed to load message from context")
	}

	return message
}

// sendReply send a reply to the message bus with some data
func (m *DefaultRouter) sendReply(retQueue string, message OutgoingResponse, data interface{}) error {
	con := m.pool.Get()
	defer con.Close()

	bytes, err := json.Marshal(data)
	if err != nil {
		return errors.Wrap(err, "failed to serialize response data")
	}
	// base 64 encode the response data
	message.Data = base64.StdEncoding.EncodeToString(bytes)

	bytes, err = json.Marshal(message)
	if err != nil {
		return errors.Wrap(err, "failed to serialize response message")
	}

	log.Debug().
		Str("id", message.Reference).
		Str("to", message.TwinDest).
		Msg("pushing response")

	_, err = con.Do("LPUSH", retQueue, string(bytes))
	if err != nil {
		return errors.Wrap(err, "failed to push response to message bus")
	}

	return nil
}
