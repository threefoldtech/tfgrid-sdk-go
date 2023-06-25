// Package direct package provides the functionality to create a direct websocket connection to rmb relays without the need to rmb peers.
package direct

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/types"
	"google.golang.org/protobuf/proto"
)

var (
	_ rmb.Client = (*DirectClient)(nil)
)

// DirectClient exposes the functionality to talk directly to an rmb relay
type DirectClient struct {
	baseClient *rmb.BaseClient
	responses  map[string]chan *types.Envelope
	respM      sync.Mutex
}

// NewClient creates a new RMB direct client. It connects directly to the RMB-Relay, and peridically tries to reconnect if the connection broke.
//
// You can close the connection by canceling the passed context.
//
// Make sure the context passed to Call() does not outlive the directClient's context.
// Call() will panic if called while the directClient's context is canceled.
func NewClient(ctx context.Context, keytype string, mnemonics string, relayURL string, session string, sub *substrate.Substrate, enableEncryption bool) (*DirectClient, error) {
	baseClient, err := rmb.NewBaseClient(ctx, keytype, mnemonics, relayURL, session, sub, enableEncryption)
	if err != nil {
		return nil, err
	}

	cl := &DirectClient{
		baseClient: baseClient,
		responses:  make(map[string]chan *types.Envelope),
	}
	go cl.process(ctx)

	return cl, nil
}

func (d *DirectClient) process(ctx context.Context) {
	for {
		select {
		case incoming := <-d.baseClient.Reader:
			var env types.Envelope
			if err := proto.Unmarshal(incoming, &env); err != nil {
				log.Error().Err(err).Msg("invalid message payload")
				return
			}
			d.router(&env)
		case <-ctx.Done():
			return
		}
	}
}

func (d *DirectClient) router(env *types.Envelope) {
	d.respM.Lock()
	defer d.respM.Unlock()

	ch, ok := d.responses[env.Uid]
	if !ok {
		return
	}

	select {
	case ch <- env:
	default:
		// client is not waiting anymore! just return then
	}
}

func (d *DirectClient) request(ctx context.Context, request *types.Envelope) (*types.Envelope, error) {

	ch := make(chan *types.Envelope)
	d.respM.Lock()
	d.responses[request.Uid] = ch
	d.respM.Unlock()

	bytes, err := proto.Marshal(request)
	if err != nil {
		return nil, err
	}

	select {
	case d.baseClient.Writer <- bytes:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	var response *types.Envelope
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case response = <-ch:
	}
	if response == nil {
		// shouldn't happen but just in case
		return nil, fmt.Errorf("no response received")
	}

	return response, nil
}

// Call sends an rmb call to the relay
func (d *DirectClient) Call(ctx context.Context, twin uint32, fn string, data interface{}, result interface{}) error {
	request, err := d.baseClient.MakeRequest(ctx, twin, fn, data)
	if err != nil {
		return errors.Wrap(err, "failed to build request")
	}

	response, err := d.request(ctx, request)
	if err != nil {
		return err
	}

	if result == nil {
		return nil
	}

	output, err := d.baseClient.HandleResponse(response)
	if err != nil {
		return err
	}
	return json.Unmarshal(output, &result)
}

// Ping sends an application level ping. You normally do not ever need to call this
// yourself because this rmb client takes care of automatic pinging of the server
// and reconnecting if needed. But in case you want to test if a connection is active
// and established you can call this Ping method yourself.
// If no error is returned then ping has succeeded.
// Make sure to always provide a ctx with a timeout or a deadline otherwise the call
// will block forever waiting for a response.
func (d *DirectClient) Ping(ctx context.Context) error {
	uid := uuid.NewString()
	request := types.Envelope{
		Uid:     uid,
		Source:  d.baseClient.Source,
		Message: &types.Envelope_Ping{},
	}

	response, err := d.request(ctx, &request)
	if err != nil {
		return err
	}
	_, ok := response.Message.(*types.Envelope_Pong)
	if !ok {
		return fmt.Errorf("expected a pong response got %T", response.Message)
	}

	return nil
}
