package rmb

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

const (
	DefaultSchema = "application/json"

	systemLocalBus = "msgbus.system.local"

	// DefaultAddress default redis address when no address is passed
	DefaultAddress = "tcp://127.0.0.1:6379"
)

type redisClient struct {
	pool *redis.Pool
}

// Default return instance of to default (local) rmb
// shortcut for NewClient(DefaultAddress)
func Default() (Client, error) {
	return NewRMBClient(DefaultAddress)
}

// NewRMBClient creates a new rmb client that runs behind an rmb-peer. This
// client does not talk to the rmb relay directly, instead talk to an rmb-peer
// instance (like a gateway) that itself maintains a connection to the relay.
// the rmb-peer does all the heavy lifting, including signing, encryption,
// validation of the response, etc...
//
// hence the address in this case, is an address to the local redis that must
// be the same one used with the rmb-peer process.
//
// for more details about rmb-peer please check https://github.com/threefoldtech/rmb-rs
// Since the rmb protocol does not specify a "payload" format this Client and the DefaultRouter
// both uses json to encode and decode the rpc body. Hence this client should be always
// 100% compatible with services built with the DefaultRouter.
func NewRMBClient(address string, poolSize ...uint32) (Client, error) {

	if len(address) == 0 {
		address = DefaultAddress
	}

	pool, err := newRedisPool(address, poolSize...)
	if err != nil {
		return nil, err
	}

	return &redisClient{
		pool: pool,
	}, nil
}

// Close closes the rmb client
func (c *redisClient) Close() error {
	return c.pool.Close()
}

// Call calls the twin with given function and message. Can return a RemoteError if error originated by remote peer
// in that case it should also include extra Code
func (c *redisClient) Call(ctx context.Context, twin uint32, fn string, data interface{}, result interface{}) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return errors.Wrap(err, "failed to serialize request data")
	}

	var ttl uint64 = 5 * 60
	deadline, ok := ctx.Deadline()
	if ok {
		ttl = uint64(time.Until(deadline).Seconds())
	}

	queue := uuid.NewString()
	msg := Request{
		Version:    1,
		Expiration: int(ttl),
		Command:    fn,
		TwinDest:   []uint32{twin},
		Data:       base64.StdEncoding.EncodeToString(bytes),
		Schema:     DefaultSchema,
		RetQueue:   queue,
	}

	bytes, err = json.Marshal(msg)
	if err != nil {
		return errors.Wrap(err, "failed to serialize message")
	}
	con := c.pool.Get()
	defer con.Close()

	_, err = con.Do("RPUSH", systemLocalBus, bytes)
	if err != nil {
		return errors.Wrap(err, "failed to push message to local twin")
	}

	// now wait for response.
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		slice, err := redis.ByteSlices(con.Do("BLPOP", queue, 5))
		if err != nil && err != redis.ErrNil {
			return errors.Wrap(err, "unexpected error during waiting for the response")
		}

		if err == redis.ErrNil || slice == nil {
			//timeout, just try again immediately
			continue
		}

		// found a response
		bytes = slice[1]
		break
	}

	var ret IncomingResponse

	// we have a response, so load or fail
	if err := json.Unmarshal(bytes, &ret); err != nil {
		return errors.Wrap(err, "failed to load response message")
	}
	// errorred ?
	if ret.Error != nil {
		return RemoteError{
			Code:    ret.Error.Code,
			Message: ret.Error.Message,
		}
	}

	// not expecting a result
	if result == nil {
		return nil
	}

	if ret.Schema != DefaultSchema {
		return fmt.Errorf("received invalid schema '%s' was expecting %s", ret.Schema, DefaultSchema)
	}

	if len(ret.Data) == 0 {
		return fmt.Errorf("no response body was returned")
	}

	bytes, err = base64.StdEncoding.DecodeString(ret.Data)
	if err != nil {
		return errors.Wrap(err, "invalid data body encoding")
	}

	if err := json.Unmarshal(bytes, result); err != nil {
		return errors.Wrap(err, "failed to decode response body")
	}

	return nil
}

type RemoteError struct {
	Code    uint32
	Message string
}

func (e RemoteError) Error() string {
	return e.Message
}
