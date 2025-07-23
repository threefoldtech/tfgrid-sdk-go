package peer

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer/types"
)

func TestPeer_HandleIncoming_ExpiredEnvelope(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	twinDB := NewMockTwinDB(ctrl)

	p := Peer{twinDB: twinDB}

	envelope := &types.Envelope{
		Timestamp:  uint64(time.Now().Add(-2 * time.Minute).Unix()), // 2 minutes ago
		Expiration: 60,                                              // 60 seconds TTL
	}

	err := p.handleIncoming(envelope)
	require.Error(t, err)
	require.Equal(t, "received an expired envelope", err.Error())
}

func TestPeer_HandleIncoming_NotExpiredEnvelope(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	twinDB := NewMockTwinDB(ctrl)
	// Expect a call to Get during signature verification
	twinDB.EXPECT().Get(uint32(1)).Return(Twin{}, nil).AnyTimes()

	p := Peer{twinDB: twinDB}

	envelope := &types.Envelope{
		Source:     &types.Address{Twin: 1}, // a source is needed to pass validation
		Timestamp:  uint64(time.Now().Unix()),
		Expiration: 60, // not expired
	}

	// We don't care about the error here because it will fail on signature verification,
	// which happens AFTER the expiration check. So if we get that error, it means
	// the expiration check has passed.
	err := p.handleIncoming(envelope)
	if err != nil {
		require.NotEqual(t, "received an expired envelope", err.Error())
	}
}
