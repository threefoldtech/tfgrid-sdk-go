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
			Value: 50000,
		},
		CU: substrate.Policy{
			Value: 100000,
		},
		IPU: substrate.Policy{
			Value: 40000,
		},
	}, nil).AnyTimes()

	cost, err := calculator.CalculateCost(8, 32, 0, 50, true, true)
	assert.NoError(t, err)
	assert.Equal(t, 76.725, cost)

	sub.EXPECT().GetBalance(identity).Return(substrate.Balance{
		Free: types.U128{
			Int: big.NewInt(50000000),
		},
	}, nil)

	dedicatedPrice, sharedPrice, err := calculator.CalculatePricesAfterDiscount(cost)
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

	t.Run("test tft pricing policy error", func(t *testing.T) {
		sub.EXPECT().GetPricingPolicy(uint32(1)).Return(substrate.PricingPolicy{}, errors.New("error")).AnyTimes()

		_, err := calculator.CalculateCost(0, 0, 0, 0, false, false)
		assert.Error(t, err)

		_, _, err = calculator.CalculatePricesAfterDiscount(200)
		assert.Error(t, err)
	})

	t.Run("test tft balance error", func(t *testing.T) {
		sub.EXPECT().GetPricingPolicy(uint32(1)).Return(substrate.PricingPolicy{}, nil).AnyTimes()
		sub.EXPECT().GetBalance(identity).Return(substrate.Balance{}, errors.New("error")).AnyTimes()

		_, _, err = calculator.CalculatePricesAfterDiscount(0)
		assert.Error(t, err)
	})
}

func TestGetApplicableDiscount(t *testing.T) {
	testCases := []struct {
		name                      string
		balance                   float64
		dedicatedPrice            float64
		sharedPrice               float64
		expectedDedicatedDiscount float64
		expectedSharedDiscount    float64
	}{
		{
			name:                      "No balance",
			balance:                   0,
			dedicatedPrice:            100,
			sharedPrice:               80,
			expectedDedicatedDiscount: 0,
			expectedSharedDiscount:    0,
		},
		{
			name:                      "Insufficient balance for any package",
			balance:                   50,
			dedicatedPrice:            100,
			sharedPrice:               80,
			expectedDedicatedDiscount: 0,
			expectedSharedDiscount:    0,
		},
		{
			name:                      "Balance enough for default package only for shared",
			balance:                   130, // > 80 * 1.5 but < 100 * 1.5
			dedicatedPrice:            100,
			sharedPrice:               80,
			expectedDedicatedDiscount: 0,
			expectedSharedDiscount:    0.2, // Default package discount 20%
		},
		{
			name:                      "Balance enough for default package for both",
			balance:                   160, // > 100 * 1.5 and > 80 * 1.5
			dedicatedPrice:            100,
			sharedPrice:               80,
			expectedDedicatedDiscount: 0.2, // Default package discount 20%
			expectedSharedDiscount:    0.2, // Default package discount 20%
		},
		{
			name:                      "Balance enough for bronze package for shared, default for dedicated",
			balance:                   250, // > 80 * 3 and < 100 * 1.5
			dedicatedPrice:            100,
			sharedPrice:               80,
			expectedDedicatedDiscount: 0.2, // Default Package discount 20%
			expectedSharedDiscount:    0.3, // Bronze package discount 30%
		},
		{
			name:                      "Balance enough for bronze package for both",
			balance:                   350, // > 100 * 3 and > 80 * 3
			dedicatedPrice:            100,
			sharedPrice:               80,
			expectedDedicatedDiscount: 0.3, // Bronze package discount 30%
			expectedSharedDiscount:    0.3, // Bronze package discount 30%
		},
		{
			name:                      "Balance enough for silver package for shared, and bronze for dedicated",
			balance:                   500, // > 80 * 6 but < 100 * 6
			dedicatedPrice:            100,
			sharedPrice:               80,
			expectedDedicatedDiscount: 0.3, // Bronze package discount 30%
			expectedSharedDiscount:    0.4, // Silver package discount 40%
		},
		{
			name:                      "Balance enough for silver package for both",
			balance:                   650, // > 100 * 6 and > 80 * 6
			dedicatedPrice:            100,
			sharedPrice:               80,
			expectedDedicatedDiscount: 0.4,
			expectedSharedDiscount:    0.4,
		},
		{
			name:                      "Balance enough for gold package for shared, and Silver for dedicated",
			balance:                   1500, // > 80 * 18 but < 100 * 18
			dedicatedPrice:            100,
			sharedPrice:               80,
			expectedDedicatedDiscount: 0.4, // Silver package discount 40%
			expectedSharedDiscount:    0.6, // Gold package discount 60%
		},
		{
			name:                      "Balance enough for gold package for both",
			balance:                   2000, // > 100 * 18 and > 80 * 18
			dedicatedPrice:            100,
			sharedPrice:               80,
			expectedDedicatedDiscount: 0.6,
			expectedSharedDiscount:    0.6,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sharedDiscount, dedicatedDiscount := getApplicableDiscount(tc.balance, tc.dedicatedPrice, tc.sharedPrice)

			assert.Equal(t, tc.expectedDedicatedDiscount, dedicatedDiscount, "Dedicated discount percentage mismatch")
			assert.Equal(t, tc.expectedSharedDiscount, sharedDiscount, "Shared discount percentage mismatch")
		})
	}
}

func TestTFTtoUSD(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sub := mocks.NewMockSubstrateExt(ctrl)
	identity, err := substrate.NewIdentityFromSr25519Phrase("//Alice")
	assert.NoError(t, err)

	calculator := NewCalculator(sub, identity)

	t.Run("success case", func(t *testing.T) {
		sub.EXPECT().GetTFTPrice().Return(types.U32(5), nil)

		result, err := calculator.TFTtoUSD(10)
		assert.NoError(t, err)
		assert.Equal(t, 0.05, result)
	})

	t.Run("error case", func(t *testing.T) {
		sub.EXPECT().GetTFTPrice().Return(types.U32(0), errors.New("failed to get TFT price"))

		_, err := calculator.TFTtoUSD(100)
		assert.Error(t, err)
	})
}
func TestUSDtoTFT(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sub := mocks.NewMockSubstrateExt(ctrl)
	// Add mock implementation for GetNodeContracts to fix compiler error
	sub.EXPECT().GetNodeContracts(gomock.Any()).Return([]types.U64{}, nil).AnyTimes()

	identity, err := substrate.NewIdentityFromSr25519Phrase("//Alice")
	assert.NoError(t, err)

	calculator := NewCalculator(sub, identity)

	t.Run("error case", func(t *testing.T) {
		sub.EXPECT().GetTFTPrice().Return(types.U32(0), errors.New("failed to get TFT price"))

		_, err := calculator.USDtoTFT(10)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get TFT price")
	})
	t.Run("success case", func(t *testing.T) {
		// 5 mUSD = 0.005 USD per TFT
		sub.EXPECT().GetTFTPrice().Return(types.U32(5), nil).AnyTimes()

		// 10 USD / 0.005 USD/TFT = 2000 TFT
		result, err := calculator.USDtoTFT(10)
		assert.NoError(t, err)

		expected := big.NewFloat(2000)
		assert.Equal(t, 0, result.Cmp(expected), "Expected %v but got %v", expected, result)
	})

	t.Run("large amount case 1 million USD", func(t *testing.T) {

		result, err := calculator.USDtoTFT(1000000.)
		assert.NoError(t, err)

		expected := big.NewFloat(200000000)
		assert.Equal(t, 0, result.Cmp(expected), "Expected %v but got %v", expected, result)
	})

	t.Run("large floating point number", func(t *testing.T) {

		result, err := calculator.USDtoTFT(9876543.21)
		assert.NoError(t, err)

		expected := new(big.Float).Quo(big.NewFloat(9876543.21), big.NewFloat(0.005))
		assert.Equal(t, 0, result.Cmp(expected), "Expected %v but got %v", expected, result)
	})

	t.Run("high precision floating point number", func(t *testing.T) {
		result, err := calculator.USDtoTFT(9876543.21453)
		assert.NoError(t, err)

		expected := new(big.Float).Quo(big.NewFloat(9876543.21453), big.NewFloat(0.005))
		assert.Equal(t, 0, result.Cmp(expected), "Expected %v but got %v", expected, result)
	})
}

func TestGetUnbilledAmountInTFT(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	sub := mocks.NewMockSubstrateExt(ctrl)

	identity, err := substrate.NewIdentityFromSr25519Phrase("//Alice")
	assert.NoError(t, err)

	calculator := NewCalculator(sub, identity)

	contractID := uint64(42)

	t.Run("Amount is 5 USD in Unit-USD, should return 1000 TFT", func(t *testing.T) {
		sub.EXPECT().GetContractBillingInfo(contractID).Return(substrate.ContractBillingInfo{
			AmountUnbilled: types.U64(1e7 * 5),
		}, nil)

		sub.EXPECT().GetTFTPrice().Return(types.U32(5), nil)

		expected := big.NewFloat(1000)

		result, err := calculator.getUnbilledAmountInTFT(contractID)
		assert.NoError(t, err)
		assert.Equal(t, 0, result.Cmp(expected), "Expected %v but got %v", expected, result)
	})

	t.Run("Amount is 0 USD, should return 0 TFT", func(t *testing.T) {
		sub.EXPECT().GetContractBillingInfo(contractID).Return(substrate.ContractBillingInfo{
			AmountUnbilled: types.U64(0),
		}, nil)

		sub.EXPECT().GetTFTPrice().Return(types.U32(5), nil)

		expected := big.NewFloat(0)

		result, err := calculator.getUnbilledAmountInTFT(contractID)
		assert.NoError(t, err)
		assert.Equal(t, 0, result.Cmp(expected), "Expected %v but got %v", expected, result)
	})

	t.Run("ErrNotFound in GetContractBillingInfo should return 0 TFT", func(t *testing.T) {
		sub.EXPECT().GetContractBillingInfo(contractID).Return(substrate.ContractBillingInfo{}, substrate.ErrNotFound)
		sub.EXPECT().GetTFTPrice().Return(types.U32(5), nil)

		expected := big.NewFloat(0)

		result, err := calculator.getUnbilledAmountInTFT(contractID)
		assert.NoError(t, err)
		assert.Equal(t, 0, result.Cmp(expected), "Expected %v but got %v", expected, result)
	})

	t.Run("Small amount test (5000 Unit-USD)", func(t *testing.T) {
		// 5000 Unit-USD = 0.0005 USD
		sub.EXPECT().GetContractBillingInfo(contractID).Return(substrate.ContractBillingInfo{
			AmountUnbilled: types.U64(5000),
		}, nil)

		sub.EXPECT().GetTFTPrice().Return(types.U32(5), nil)

		// 0.0005 USD / 0.005 USD per TFT = 0.1 TFT
		expected := big.NewFloat(0.1)

		result, err := calculator.getUnbilledAmountInTFT(contractID)
		assert.NoError(t, err)
		assert.Equal(t, 0, result.Cmp(expected), "Expected %v but got %v", expected, result)
	})

	t.Run("Large amount test (1 million Unit-USD)", func(t *testing.T) {
		// 1 million Unit-USD = 1 USD
		sub.EXPECT().GetContractBillingInfo(contractID).Return(substrate.ContractBillingInfo{
			AmountUnbilled: types.U64(1e7),
		}, nil)

		sub.EXPECT().GetTFTPrice().Return(types.U32(5), nil)

		// 1 USD / 0.005 USD per TFT = 200 TFT
		expected := big.NewFloat(200)

		result, err := calculator.getUnbilledAmountInTFT(contractID)
		assert.NoError(t, err)
		assert.Equal(t, 0, result.Cmp(expected), "Expected %v but got %v", expected, result)
	})

	t.Run("Very large amount test (10 billion Unit-USD)", func(t *testing.T) {
		// 10 billion Unit-USD = 1000 USD
		sub.EXPECT().GetContractBillingInfo(contractID).Return(substrate.ContractBillingInfo{
			AmountUnbilled: types.U64(1e10),
		}, nil)

		sub.EXPECT().GetTFTPrice().Return(types.U32(5), nil)

		// 1000 USD / 0.005 USD per TFT = 200,000 TFT
		expected := big.NewFloat(200000)

		result, err := calculator.getUnbilledAmountInTFT(contractID)
		assert.NoError(t, err)
		assert.Equal(t, 0, result.Cmp(expected), "Expected %v but got %v", expected, result)
	})

	t.Run("error in GetContractBillingInfoByID", func(t *testing.T) {
		sub.EXPECT().GetContractBillingInfo(contractID).Return(substrate.ContractBillingInfo{}, errors.New("failed to get billing info"))

		_, err := calculator.getUnbilledAmountInTFT(contractID)
		assert.Error(t, err)
	})

	t.Run("error in USDtoTFT", func(t *testing.T) {
		sub.EXPECT().GetContractBillingInfo(contractID).Return(substrate.ContractBillingInfo{
			AmountUnbilled: types.U64(1000),
		}, nil)

		sub.EXPECT().GetTFTPrice().Return(types.U32(0), errors.New("failed to get TFT price"))

		_, err := calculator.getUnbilledAmountInTFT(contractID)
		assert.Error(t, err)
	})
}
