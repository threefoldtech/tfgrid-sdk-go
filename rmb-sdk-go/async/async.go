package async

import (
	"context"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/common"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/common/types"
	"google.golang.org/protobuf/proto"
)

type AsyncClient struct {
	baseClient       *common.BaseClient
	responseListener RMBResponseListener
}

type RMBResponseListener func(result []byte) error

func NewAsyncClient(ctx context.Context, responseListener RMBResponseListener, keytype string, mnemonics string, relayURL string, session string, sub *substrate.Substrate, enableEncryption bool) (*AsyncClient, error) {
	baseClient, err := common.NewBaseClient(ctx, keytype, mnemonics, relayURL, session, sub, enableEncryption)
	if err != nil {
		return nil, err
	}

	cl := &AsyncClient{
		baseClient:       baseClient,
		responseListener: responseListener,
	}

	go cl.process(ctx)

	return cl, nil
}

func (d *AsyncClient) Send(ctx context.Context, twin uint32, fn string, data interface{}) error {
	request, err := d.baseClient.MakeRequest(ctx, twin, fn, data)
	if err != nil {
		return errors.Wrap(err, "failed to build request")
	}
	bytes, err := proto.Marshal(request)
	if err != nil {
		return err
	}

	select {
	case d.baseClient.Writer <- bytes:
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

func (d *AsyncClient) process(ctx context.Context) {
	for {
		select {
		case incoming := <-d.baseClient.Reader:
			var env types.Envelope
			if err := proto.Unmarshal(incoming, &env); err != nil {
				log.Error().Err(err).Msg("invalid message payload")
				return
			}
			output, err := d.baseClient.HandleResponse(&env)
			if err != nil {
				log.Error().Err(err).Msg("error while reading response")
			}
			if err := d.responseListener(output); err != nil {
				log.Error().Err(err).Msg("error while performing listener action")
			}
		case <-ctx.Done():
			return
		}
	}
}
