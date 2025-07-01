package calculator

import (
	"errors"
	"math/big"
	"testing"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/mocks"
)

func TestCalculator(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sub := mocks.NewMockSubstrateExt(ctrl)
	identity, err := substrate.NewIdentityFromSr25519Phrase("//Alice")
	assert.NoError(t, err)

	calculator := NewCalculator(sub, identity)

	sub.EXPECT().GetTFTPrice().Return(types.U32(5), nil).AnyTimes()
	sub.EXPECT().GetPricingPolicy(uint32(1)).Return(substrate.PricingPolicy{
		ID: 1,
		SU: substrate.Policy{
			Value: 2,
		},
		CU: substrate.Policy{
			Value: 2,
		},
		IPU: substrate.Policy{
			Value: 2,
		},
	}, nil).AnyTimes()

	cost, err := calculator.CalculateCost(8, 32, 0, 50, true, true)
	assert.NoError(t, err)
	assert.Equal(t, cost, 16.65)

	sub.EXPECT().GetBalance(identity).Return(substrate.Balance{
		Free: types.U128{
			Int: big.NewInt(50000000),
		},
	}, nil)

	dedicatedPrice, sharedPrice, err := calculator.CalculateDiscount(cost)
	assert.NoError(t, err)
	assert.Equal(t, dedicatedPrice, sharedPrice)
}

func TestSubstrateErrors(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sub := mocks.NewMockSubstrateExt(ctrl)
	identity, err := substrate.NewIdentityFromSr25519Phrase("//Alice")
	assert.NoError(t, err)

	calculator := NewCalculator(sub, identity)

	t.Run("test tft price error", func(t *testing.T) {
		sub.EXPECT().GetTFTPrice().Return(types.U32(1), errors.New("error")).AnyTimes()
		_, _, err = calculator.CalculateDiscount(200)
		assert.Error(t, err)
	})

	t.Run("test tft pricing policy error", func(t *testing.T) {
		sub.EXPECT().GetTFTPrice().Return(types.U32(1), nil).AnyTimes()
		sub.EXPECT().GetPricingPolicy(uint32(1)).Return(substrate.PricingPolicy{}, errors.New("error")).AnyTimes()

		_, err := calculator.CalculateCost(0, 0, 0, 0, false, false)
		assert.Error(t, err)

		_, _, err = calculator.CalculateDiscount(200)
		assert.Error(t, err)
	})

	t.Run("test tft balance error", func(t *testing.T) {
		sub.EXPECT().GetTFTPrice().Return(types.U32(1), nil).AnyTimes()
		sub.EXPECT().GetPricingPolicy(uint32(1)).Return(substrate.PricingPolicy{}, nil).AnyTimes()
		sub.EXPECT().GetBalance(identity).Return(substrate.Balance{}, errors.New("error")).AnyTimes()

		_, _, err = calculator.CalculateDiscount(0)
		assert.Error(t, err)
	})
}

func TestCalculateUniqueNameCost(t *testing.T) {
	t.Run("success case", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		sub := mocks.NewMockSubstrateExt(ctrl)
		sub.EXPECT().GetPricingPolicy(uint32(1)).Return(substrate.PricingPolicy{
			ID: 1,
			UniqueName: substrate.Policy{
				Value: 2500,
			},
		}, nil)

		identity, err := substrate.NewIdentityFromSr25519Phrase("//Alice")
		assert.NoError(t, err)

		calculator := NewCalculator(sub, identity)

		// Calculate expected result: Value * 24 * 30  and converted from Unit-USD to USD
		expectedCost := 0.18

		cost, err := calculator.CalculateUniqueNameCost()

		assert.NoError(t, err)
		assert.Equal(t, expectedCost, cost)
	})

	t.Run("error pricing policy case", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		sub := mocks.NewMockSubstrateExt(ctrl)
		sub.EXPECT().GetPricingPolicy(uint32(1)).Return(substrate.PricingPolicy{}, errors.New("error"))

		identity, err := substrate.NewIdentityFromSr25519Phrase("//Alice")
		assert.NoError(t, err)

		calculator := NewCalculator(sub, identity)

		_, err = calculator.CalculateUniqueNameCost()

		assert.Error(t, err)
	})

}
