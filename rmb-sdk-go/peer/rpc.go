package peer

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/google/uuid"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer/types"
)

var (
	_ rmb.Client = (*RpcCLient)(nil)
)

type incomingEnv struct {
	env *types.Envelope
	err error
}

// RpcClient is a peer connection that makes it easy to make rpc calls
type RpcCLient struct {
	base      *Peer
	responses map[string]chan incomingEnv
	m         sync.Mutex
}

// NewRpcClient create a new rpc client
// the rpc client is a full peer, but provide a custom handler to make
// it easy to make rpc calls
func NewRpcClient(
	ctx context.Context,
	keytype string,
	mnemonics string,
	relayURL string,
	session string,
	sub *substrate.Substrate,
	enableEncryption bool) (*RpcCLient, error) {

	rpc := RpcCLient{
		responses: make(map[string]chan incomingEnv),
	}

	base, err := NewPeer(ctx, keytype, mnemonics, relayURL, session, sub, enableEncryption, rpc.router)

	if err != nil {
		return nil, err
	}

	rpc.base = base
	return &rpc, nil
}

func (d *RpcCLient) router(env *types.Envelope, err error) {
	d.m.Lock()
	defer d.m.Unlock()

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

func (d *RpcCLient) Call(ctx context.Context, twin uint32, fn string, data interface{}, result interface{}) error {
	id := uuid.NewString()

	ch := make(chan incomingEnv)
	defer func() {
		close(ch)

		d.m.Lock()
		delete(d.responses, id)
		d.m.Unlock()
	}()

	d.m.Lock()
	d.responses[id] = ch
	d.m.Unlock()

	if err := d.base.Send(ctx, id, twin, fn, data); err != nil {
		return err
	}

	var incoming incomingEnv
	select {
	case <-ctx.Done():
		return ctx.Err()
	case incoming = <-ch:
	}

	if incoming.err != nil {
		return incoming.err
	}

	response := incoming.env

	errResp := response.GetError()

	if errResp != nil {
		// todo: include code also
		return fmt.Errorf(errResp.Message)
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
