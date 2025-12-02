package peer

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer/types"
)

var (
	_ rmb.Client = (*RpcClient)(nil)
)

type incomingEnv struct {
	env *types.Envelope
	err error
}

// Wait blocks until the underlying peer has fully shut down.
// Caller should cancel the context passed to NewRpcClient/NewPeer() to initiate shutdown.
func (d *RpcClient) Wait() {
	if d == nil || d.base == nil {
		return
	}
	d.base.Wait()
}

// RpcClient is a peer connection that makes it easy to make rpc calls
type RpcClient struct {
	base      *Peer
	responses map[string]chan incomingEnv
	m         sync.RWMutex
}

// NewRpcClient create a new rpc client
// the rpc client is a full peer, but provide a custom handler to make
// it easy to make rpc calls
func NewRpcClient(
	ctx context.Context,
	mnemonics string,
	subManager substrate.Manager,
	opts ...PeerOpt) (*RpcClient, error) {

	rpc := RpcClient{
		responses: make(map[string]chan incomingEnv),
	}

	base, err := NewPeer(
		ctx,
		mnemonics,
		subManager,
		rpc.router,
		opts...,
	)

	if err != nil {
		return nil, err
	}

	rpc.base = base
	return &rpc, nil
}

func (d *RpcClient) router(ctx context.Context, peer *Peer, env *types.Envelope, err error) {
	d.m.RLock()
	defer d.m.RUnlock()

	ch, ok := d.responses[env.Uid]
	if !ok {
		return
	}

	select {
	case ch <- incomingEnv{env: env, err: err}:
	default:
		// client is not waiting anymore! just return then
	}
}

func (d *RpcClient) Call(ctx context.Context, twin uint32, fn string, data interface{}, result interface{}) error {
	return d.CallWithSession(ctx, twin, nil, fn, data, result)
}

// CallWithTags is like Call but allows specifying request tags (e.g., fail-fast)
func (d *RpcClient) CallWithTags(ctx context.Context, twin uint32, fn string, data interface{}, result interface{}, tags []Tag) error {
	return d.CallWithSessionWithTags(ctx, twin, nil, fn, data, result, tags)
}

func (d *RpcClient) CallWithSession(ctx context.Context, twin uint32, session *string, fn string, data interface{}, result interface{}) error {
	return d.CallWithSessionWithTags(ctx, twin, session, fn, data, result, nil)
}

// CallWithSessionWithTags is like CallWithSession but allows specifying request tags (e.g., fail-fast)
// Tags are normalized and serialized by the underlying peer before being set on the envelope.
func (d *RpcClient) CallWithSessionWithTags(ctx context.Context, twin uint32, session *string, fn string, data interface{}, result interface{}, tags []Tag) error {
	id := uuid.NewString()

	ch := make(chan incomingEnv, 1)
	defer func() {
		d.m.Lock()
		delete(d.responses, id)
		d.m.Unlock()
	}()

	d.m.Lock()
	d.responses[id] = ch
	d.m.Unlock()

	if err := d.base.SendRequestWithTags(ctx, id, twin, session, fn, data, tags); err != nil {
		return fmt.Errorf("failed to send request (twin=%d, fn=%s): %w", twin, fn, err)
	}

	var incoming incomingEnv
	select {
	case <-ctx.Done():
		return fmt.Errorf("rpc call canceled/timeout (twin=%d, fn=%s): %w", twin, fn, ctx.Err())
	case incoming = <-ch:
	}

	if incoming.err != nil {
		// Error envelope from a destination twin
		if incoming.env != nil && incoming.env.Source != nil {
			return fmt.Errorf(
				"remote error from twin %d (fn=%s, uid=%s): %w",
				incoming.env.Source.Twin, fn, id, incoming.err,
			)
		}
		// Relay-origin or transport error; envelope may be nil
		return fmt.Errorf(
			"relay/transport error (twin=%d, fn=%s, uid=%s): %w",
			twin, fn, id, incoming.err,
		)
	}

	response := incoming.env

	errResp := response.GetError()

	if errResp != nil {
		// todo: include code also
		return errors.New(errResp.Message)
	}

	resp := response.GetResponse()
	if resp == nil {
		return fmt.Errorf("received a non response envelope")
	}

	if result == nil {
		return nil
	}

	if response.Schema == nil || *response.Schema != rmb.DefaultSchema {
		return fmt.Errorf("invalid schema received expected '%s'", rmb.DefaultSchema)
	}

	// this is safe to do because the underlying client
	// always decrypt any encrypted messages so this
	// can only be plain
	output := response.Payload.(*types.Envelope_Plain).Plain

	return json.Unmarshal(output, &result)
}

// // Ping sends an application level ping. You normally do not ever need to call this
// // yourself because this rmb client takes care of automatic pinging of the server
// // and reconnecting if needed. But in case you want to test if a connection is active
// // and established you can call this Ping method yourself.
// // If no error is returned then ping has succeeded.
// // Make sure to always provide a ctx with a timeout or a deadline otherwise the call
// // will block forever waiting for a response.
// func (d *RpcCLient) Ping(ctx context.Context) error {
// 	uid := uuid.NewString()
// 	request := types.Envelope{
// 		Uid:     uid,
// 		Source:  d.source,
// 		Message: &types.Envelope_Ping{},
// 	}

// 	response, err := d.request(ctx, &request)
// 	if err != nil {
// 		return err
// 	}
// 	_, ok := response.Message.(*types.Envelope_Pong)
// 	if !ok {
// 		return fmt.Errorf("expected a pong response got %T", response.Message)
// 	}

// 	return nil
// }
