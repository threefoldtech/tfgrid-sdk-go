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

func TestCalculateIPV4(t *testing.T) {
	t.Run("success case", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		sub := mocks.NewMockSubstrateExt(ctrl)
		sub.EXPECT().GetPricingPolicy(uint32(1)).Return(substrate.PricingPolicy{
			ID: 1,
			IPU: substrate.Policy{
				Value: 2000,
			},
		}, nil)

		identity, err := substrate.NewIdentityFromSr25519Phrase("//Alice")
		assert.NoError(t, err)

		calculator := NewCalculator(sub, identity)

		// Calculate expected result: Value * 24 * 30 and converted from Unit-USD to USD
		expectedCost := 0.144 // 2000 * 24 * 30 / 10_000_000

		cost, err := calculator.calculateIPV4()

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

		_, err = calculator.calculateIPV4()

		assert.Error(t, err)
	})
}

func TestConvertBytesToGB(t *testing.T) {
	testCases := []struct {
		name           string
		inputBytes     uint64
		expectedGigaBytes int64
	}{
		{
			name:           "zero bytes",
			inputBytes:     0,
			expectedGigaBytes: 0,
		},
		{
			name:           "less than 1GB",
			inputBytes:     500 * 1024 * 1024, // 500 MB
			expectedGigaBytes: 0,              // Should be 0 since integer division
		},
		{
			name:           "exactly 1GB",
			inputBytes:     1024 * 1024 * 1024,
			expectedGigaBytes: 1,
		},
		{
			name:           "multiple GBs",
			inputBytes:     5 * 1024 * 1024 * 1024,
			expectedGigaBytes: 5,
		},
		{
			name:           "large number",
			inputBytes:     1000 * 1024 * 1024 * 1024,
			expectedGigaBytes: 1000,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := convertBytesToGB(tc.inputBytes)
			assert.Equal(t, tc.expectedGigaBytes, result, "Conversion from bytes to GB failed")
		})
	}
}

func TestCalculateRentCost(t *testing.T) {
	t.Run("success case", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// Create mock substrate extension
		sub := mocks.NewMockSubstrateExt(ctrl)
		
		// Setup mock expectations
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
		
		// Setup mock for GetDedicatedNodePrice
		contractID := uint32(123)
		extraFee := uint64(5000000) // 0.5 USD in unit factor
		sub.EXPECT().GetDedicatedNodePrice(contractID).Return(extraFee, nil)

		// Create calculator instance
		identity, err := substrate.NewIdentityFromSr25519Phrase("//Alice")
		assert.NoError(t, err)
		calculator := NewCalculator(sub, identity)

		// Create mock contract and node
		contract := &substrate.Contract{
			ContractID: types.U64(contractID),
			State:      substrate.ContractState{IsCreated: true},
		}
		
		node := substrate.Node{
			Resources: substrate.Resources{
				CRU: 8,
				MRU: 16 * 1024 * 1024 * 1024, // 16GB
				HRU: 1000 * 1024 * 1024 * 1024, // 1000GB
				SRU: 500 * 1024 * 1024 * 1024, // 500GB
			},
			Certification: substrate.NodeCertification{
				IsCertified: true,
			},
		}

		// Call the function being tested
		cost, err := calculator.CalculateRentCost(contract, node)

		// Assert the results
		assert.NoError(t, err)
		// Expected cost includes the base compute and storage costs plus the extra fee
		// Base cost from resources calculation + 0.5 USD from dedicated node extra fee
		expectedCost := 16.65 + 0.5
		assert.Equal(t, expectedCost, cost)
	})

	t.Run("error from CalculateCost", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// Create mock substrate extension
		sub := mocks.NewMockSubstrateExt(ctrl)

		// Setup mock expectations to return error for pricing policy
		sub.EXPECT().GetTFTPrice().Return(types.U32(0), nil).AnyTimes()
		sub.EXPECT().GetPricingPolicy(uint32(1)).Return(substrate.PricingPolicy{}, errors.New("pricing policy error"))

		// Create calculator instance
		identity, err := substrate.NewIdentityFromSr25519Phrase("//Alice")
		assert.NoError(t, err)
		calculator := NewCalculator(sub, identity)

		// Create mock contract and node
		contract := &substrate.Contract{
			ContractID: types.U64(123),
			State:      substrate.ContractState{IsCreated: true},
		}

		node := substrate.Node{
			Resources: substrate.Resources{
				CRU: 4,
				MRU: 8 * 1024 * 1024 * 1024,
				SRU: 250 * 1024 * 1024 * 1024,
			},
		}

		// Call the function being tested
		_, err = calculator.CalculateRentCost(contract, node)

		// Assert an error was returned
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "pricing policy error")
	})

	t.Run("error from GetDedicatedNodePrice", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// Create mock substrate extension
		sub := mocks.NewMockSubstrateExt(ctrl)

		// Setup mock expectations for successful policy retrieval
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
		}, nil)

		// Setup mock for GetDedicatedNodePrice to return an error
		contractID := uint32(123)
		sub.EXPECT().GetDedicatedNodePrice(contractID).Return(uint64(0), errors.New("dedicated node price error"))

		// Create calculator instance
		identity, err := substrate.NewIdentityFromSr25519Phrase("//Alice")
		assert.NoError(t, err)
		calculator := NewCalculator(sub, identity)

		// Create mock contract and node
		contract := &substrate.Contract{
			ContractID: types.U64(contractID),
			State:      substrate.ContractState{IsCreated: true},
		}

		node := substrate.Node{
			Resources: substrate.Resources{
				CRU: 8,
				MRU: 16 * 1024 * 1024 * 1024,
				SRU: 500 * 1024 * 1024 * 1024,
			},
			Certification: substrate.NodeCertification{
				IsCertified: true,
			},
		}

		// Call the function being tested
		_, err = calculator.CalculateRentCost(contract, node)

		// Assert an error was returned
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get dedicated node extra fee")
	})
}
