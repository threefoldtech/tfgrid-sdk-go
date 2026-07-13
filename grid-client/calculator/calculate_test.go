package calculator

import (
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/zos_sdk_go/grid-client/mocks"
	"github.com/threefoldtech/zos_sdk_go/grid-client/subi"
)

func TestCalculator(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sub := mocks.NewMockSubstrateExt(ctrl)
	identity, err := substrate.NewIdentityFromSr25519Phrase("//Alice")
	assert.NoError(t, err)

	calculator := NewCalculator(sub, identity)

	sub.EXPECT().GetTFTBillingRate().Return(types.U32(5), nil).AnyTimes()
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
		sub.EXPECT().GetTFTBillingRate().Return(types.U32(5), nil)

		result, err := calculator.TFTtoUSD(10)
		assert.NoError(t, err)
		assert.Equal(t, 0.05, result)
	})

	t.Run("error case", func(t *testing.T) {
		sub.EXPECT().GetTFTBillingRate().Return(types.U32(0), errors.New("failed to get TFT price"))

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
		sub.EXPECT().GetTFTBillingRate().Return(types.U32(0), errors.New("failed to get TFT price"))

		_, err := calculator.USDtoTFT(10)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "failed to get TFT price")
	})
	t.Run("success case", func(t *testing.T) {
		// 5 mUSD = 0.005 USD per TFT
		sub.EXPECT().GetTFTBillingRate().Return(types.U32(5), nil).AnyTimes()

		// 10 USD / 0.005 USD/TFT = 2000 TFT
		result, err := calculator.USDtoTFT(10)
		assert.NoError(t, err)

		expected := float64(2000)
		assert.Equal(t, expected, result, "Expected %v but got %v", expected, result)
	})

	t.Run("large amount case 1 million USD", func(t *testing.T) {

		result, err := calculator.USDtoTFT(1000000.)
		assert.NoError(t, err)

		expected := float64(200000000)
		assert.Equal(t, expected, result, "Expected %v but got %v", expected, result)
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
	contract := substrate.Contract{
		ContractID: types.U64(contractID),
		ContractType: substrate.ContractType{
			IsNodeContract: true,
			NodeContract: substrate.NodeContract{
				Node:           1,
				PublicIPsCount: 1, // Add PublicIPsCount to ensure billing calculations are performed
			},
		},
	}

	// Test name contract case
	t.Run("Name contract returns 0", func(t *testing.T) {
		nameContract := substrate.Contract{
			ContractID: types.U64(contractID),
			ContractType: substrate.ContractType{
				IsNameContract: true,
			},
		}

		result, err := calculator.getUnbilledAmountInTFT(&nameContract, true)
		assert.NoError(t, err)
		assert.Equal(t, 0.0, result)
	})

	// Test rent contract case
	t.Run("Rent contract returns 0", func(t *testing.T) {
		rentContract := substrate.Contract{
			ContractID: types.U64(contractID),
			ContractType: substrate.ContractType{
				IsRentContract: true,
			},
		}

		result, err := calculator.getUnbilledAmountInTFT(&rentContract, true)
		assert.NoError(t, err)
		assert.Equal(t, 0.0, result)
	})

	// Test no public IPs
	t.Run("Node contract with no public IPs returns 0", func(t *testing.T) {
		noIPContract := substrate.Contract{
			ContractID: types.U64(contractID),
			ContractType: substrate.ContractType{
				IsNodeContract: true,
				NodeContract: substrate.NodeContract{
					Node:           1,
					PublicIPsCount: 0,
				},
			},
		}

		result, err := calculator.getUnbilledAmountInTFT(&noIPContract, true)
		assert.NoError(t, err)
		assert.Equal(t, 0.0, result)
	})

	t.Run("Amount is 5 USD in Unit-USD, should return 1000 TFT", func(t *testing.T) {
		sub.EXPECT().GetContractBillingInfo(contractID).Return(substrate.ContractBillingInfo{
			AmountUnbilled: types.U64(1e7 * 5),
		}, nil)

		sub.EXPECT().GetTFTBillingRate().Return(types.U32(5), nil)

		expected := 1000.0 * 1.25

		result, err := calculator.getUnbilledAmountInTFT(&contract, true)
		assert.NoError(t, err)
		assert.Equal(t, expected, result, "Expected %v but got %v", expected, result)
	})

	t.Run("Amount is 5 USD in Unit-USD, should return 1000 TFT, non certified node", func(t *testing.T) {
		sub.EXPECT().GetContractBillingInfo(contractID).Return(substrate.ContractBillingInfo{
			AmountUnbilled: types.U64(1e7 * 5),
		}, nil)

		sub.EXPECT().GetTFTBillingRate().Return(types.U32(5), nil)

		expected := 1000.0

		result, err := calculator.getUnbilledAmountInTFT(&contract, false)
		assert.NoError(t, err)
		assert.Equal(t, expected, result, "Expected %v but got %v", expected, result)
	})

	t.Run("Amount is 0 USD, should return 0 TFT", func(t *testing.T) {
		sub.EXPECT().GetContractBillingInfo(contractID).Return(substrate.ContractBillingInfo{
			AmountUnbilled: types.U64(0),
		}, nil)

		sub.EXPECT().GetTFTBillingRate().Return(types.U32(5), nil)

		expected := 0.0

		result, err := calculator.getUnbilledAmountInTFT(&contract, false)
		assert.NoError(t, err)
		assert.Equal(t, expected, result, "Expected %v but got %v", expected, result)
	})

	t.Run("ErrNotFound in GetContractBillingInfo should return 0 TFT", func(t *testing.T) {
		sub.EXPECT().GetContractBillingInfo(contractID).Return(substrate.ContractBillingInfo{}, substrate.ErrNotFound)
		sub.EXPECT().GetTFTBillingRate().Return(types.U32(5), nil)

		expected := 0.0

		result, err := calculator.getUnbilledAmountInTFT(&contract, false)
		assert.NoError(t, err)
		assert.Equal(t, expected, result, "Expected %v but got %v", expected, result)
	})

	t.Run("Small amount test (5000 Unit-USD)", func(t *testing.T) {
		// 5000 Unit-USD = 0.0005 USD
		sub.EXPECT().GetContractBillingInfo(contractID).Return(substrate.ContractBillingInfo{
			AmountUnbilled: types.U64(5000),
		}, nil)

		sub.EXPECT().GetTFTBillingRate().Return(types.U32(5), nil)

		// 0.0005 USD / 0.005 USD per TFT = 0.1 TFT
		expected := 0.1

		result, err := calculator.getUnbilledAmountInTFT(&contract, false)
		assert.NoError(t, err)
		assert.Equal(t, expected, result, "Expected %v but got %v", expected, result)
	})

	t.Run("Large amount test (1 million Unit-USD), certified node", func(t *testing.T) {
		// 1 million Unit-USD = 1 USD
		sub.EXPECT().GetContractBillingInfo(contractID).Return(substrate.ContractBillingInfo{
			AmountUnbilled: types.U64(1e7),
		}, nil)

		sub.EXPECT().GetTFTBillingRate().Return(types.U32(5), nil)

		// 1 USD / 0.005 USD per TFT = 200 TFT
		expected := 200.0 * 1.25

		result, err := calculator.getUnbilledAmountInTFT(&contract, true)
		assert.NoError(t, err)
		assert.Equal(t, expected, result, "Expected %v but got %v", expected, result)
	})

	t.Run("Very large amount test (10 billion Unit-USD)", func(t *testing.T) {
		// 10 billion Unit-USD = 1000 USD
		sub.EXPECT().GetContractBillingInfo(contractID).Return(substrate.ContractBillingInfo{
			AmountUnbilled: types.U64(1e10),
		}, nil)

		sub.EXPECT().GetTFTBillingRate().Return(types.U32(5), nil)

		// 1000 USD / 0.005 USD per TFT = 200,000 TFT
		expected := 200000.0

		result, err := calculator.getUnbilledAmountInTFT(&contract, false)
		assert.NoError(t, err)
		assert.Equal(t, expected, result, "Expected %v but got %v", expected, result)
	})

	t.Run("error in GetContractBillingInfoByID", func(t *testing.T) {
		sub.EXPECT().GetContractBillingInfo(contractID).Return(substrate.ContractBillingInfo{}, errors.New("failed to get billing info"))

		_, err := calculator.getUnbilledAmountInTFT(&contract, false)
		assert.Error(t, err)
	})

	t.Run("error in USDtoTFT", func(t *testing.T) {
		sub.EXPECT().GetContractBillingInfo(contractID).Return(substrate.ContractBillingInfo{
			AmountUnbilled: types.U64(1000),
		}, nil)

		sub.EXPECT().GetTFTBillingRate().Return(types.U32(0), errors.New("failed to get TFT price"))

		_, err := calculator.getUnbilledAmountInTFT(&contract, false)
		assert.Error(t, err)
	})

}

func TestCalculateNodeContractCost(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sub := mocks.NewMockSubstrateExt(ctrl)
	identity, err := substrate.NewIdentityFromSr25519Phrase("//Alice")
	assert.NoError(t, err)

	calculator := NewCalculator(sub, identity)

	sub.EXPECT().GetPricingPolicy(defaultPricingPolicyID).Return(substrate.PricingPolicy{
		ID: 1,
		IPU: substrate.Policy{
			Value: 15000, // 15000 unit-USD
		},
		DedicatedNodesDiscount: 20,
	}, nil).MaxTimes(10)
	sub.EXPECT().GetTFTBillingRate().Return(types.U32(5), nil).AnyTimes()

	t.Run("node contract on rented node with public IPs", func(t *testing.T) {
		// Create a node contract with 2 public IPs
		nodeContract := &substrate.Contract{
			ContractID: 123,
			ContractType: substrate.ContractType{
				IsNodeContract: true,
				NodeContract: substrate.NodeContract{
					PublicIPsCount: 2,
				},
			},
		}

		// Expect GetBalance and GetTFTBillingRate to be called
		sub.EXPECT().GetBalance(identity).Return(substrate.Balance{
			Free: types.U128{
				Int: big.NewInt(100000), // Small balance, no discount
			},
		}, nil)

		// Calculate the expected cost:
		// IPV4 cost = 15000 * 24 * 30 / 10^7 = 0.045 USD per month
		expected := ((15000 * 24 * 30 / 1e7) * 2)
		result, err := calculator.calculateNodeContractCost(*nodeContract, false, true) // not certified, is rented
		assert.NoError(t, err)
		assert.Equal(t, expected, result)
	})
	t.Run("node contract on rented node with public IPs with gold discount", func(t *testing.T) {
		// Create a node contract with 2 public IPs
		nodeContract := &substrate.Contract{
			ContractID: 123,
			ContractType: substrate.ContractType{
				IsNodeContract: true,
				NodeContract: substrate.NodeContract{
					PublicIPsCount: 2,
				},
			},
		}

		// Expect GetBalance and GetTFTBillingRate to be called
		sub.EXPECT().GetBalance(identity).Return(substrate.Balance{
			Free: types.U128{
				Int: big.NewInt(100000000000), // Big balance, gold discount
			},
		}, nil)

		// Calculate the expected cost:
		// IPV4 cost = 15000 * 24 * 30 / 10^7 = 0.045 USD per month
		// For 2 IPs without certified
		expected := ((15000 * 24 * 30 / 1e7) * 2) * 0.4
		result, err := calculator.calculateNodeContractCost(*nodeContract, false, true) // not certified, is rented
		assert.NoError(t, err)
		assert.InDelta(t, expected, result, 0.0001)
	})
	t.Run("node contract on rented node with public IPs with gold discount, certified node", func(t *testing.T) {
		// Create a node contract with 2 public IPs
		nodeContract := &substrate.Contract{
			ContractID: 123,
			ContractType: substrate.ContractType{
				IsNodeContract: true,
				NodeContract: substrate.NodeContract{
					PublicIPsCount: 2,
				},
			},
		}

		// Expect GetBalance and GetTFTBillingRate to be called
		sub.EXPECT().GetBalance(identity).Return(substrate.Balance{
			Free: types.U128{
				Int: big.NewInt(100000000000), // Big balance, gold discount
			},
		}, nil)

		expected := ((15000 * 24 * 30 / 1e7) * 2) * 0.4 * 1.25
		result, err := calculator.calculateNodeContractCost(*nodeContract, true, true) // certified, is rented
		assert.NoError(t, err)
		assert.InDelta(t, expected, result, 0.0001)
	})

	t.Run("node contract on certified rented node with public IPs", func(t *testing.T) {
		// Create a node contract with 2 public IPs
		nodeContract := &substrate.Contract{
			ContractID: 124,
			ContractType: substrate.ContractType{
				IsNodeContract: true,
				NodeContract: substrate.NodeContract{
					PublicIPsCount: 2,
				},
			},
		}

		// Expect GetBalance and GetTFTBillingRate to be called
		sub.EXPECT().GetBalance(identity).Return(substrate.Balance{
			Free: types.U128{
				Int: big.NewInt(0), // Small balance, no discount
			},
		}, nil).MaxTimes(1)

		expected := 2.7

		// Execute test
		result, err := calculator.calculateNodeContractCost(*nodeContract, true, true) // certified, is rented
		assert.NoError(t, err)
		assert.InDelta(t, expected, result, 0.0001, "Expected %v but got %v", expected, result)
	})

	t.Run("node contract on rented node with public IPs and gold discount", func(t *testing.T) {

		// Create a node contract with 2 public IPs
		nodeContract := &substrate.Contract{
			ContractID: 125,
			ContractType: substrate.ContractType{
				IsNodeContract: true,
				NodeContract: substrate.NodeContract{
					PublicIPsCount: 2,
				},
			},
		}

		sub.EXPECT().GetBalance(identity).Return(substrate.Balance{
			Free: types.U128{
				Int: big.NewInt(200000000000), // More than enough for gold discount
			},
		}, nil).MaxTimes(1)

		expected := ((15000 * 24 * 30 / 1e7) * 2) * 0.4

		// Execute test
		result, err := calculator.calculateNodeContractCost(*nodeContract, false, true) // not certified, is rented
		assert.NoError(t, err)
		assert.InDelta(t, expected, result, 0.0001, "Expected %v but got %v", expected, result)
	})

	t.Run("node contract on shared node", func(t *testing.T) {
		// Create a node contract without public IPs
		nodeContract := &substrate.Contract{
			ContractID: 126,
			ContractType: substrate.ContractType{
				IsNodeContract: true,
				NodeContract: substrate.NodeContract{
					PublicIPsCount: 0,
				},
			},
		}

		// Mock GetNodeContractResources
		sub.EXPECT().GetNodeContractResources(uint64(126)).Return(substrate.NodeContractResources{
			Used: substrate.Resources{
				CRU: 4,                        // 4 vCPU
				MRU: 8 * 1024 * 1024 * 1024,   // 8 GB RAM in bytes
				SRU: 100 * 1024 * 1024 * 1024, // 100 GB SSD in bytes
				HRU: 0,                        // No HDD
			},
		}, nil)

		// Mock pricing policy for CU/SU calculation
		sub.EXPECT().GetPricingPolicy(defaultPricingPolicyID).Return(substrate.PricingPolicy{
			ID: 1,
			CU: substrate.Policy{
				Value: 100000,
			},

			SU: substrate.Policy{
				Value: 50000,
			},
			DedicatedNodesDiscount: 20,
		}, nil).MaxTimes(2)

		// Expect GetBalance and GetTFTBillingRate to be called for discount calculation
		sub.EXPECT().GetBalance(identity).Return(substrate.Balance{
			Free: types.U128{
				Int: big.NewInt(100000), // Small balance, no discount
			},
		}, nil)

		// Execute test - we don't need to check the exact result as CalculateCost is tested separately
		result, err := calculator.calculateNodeContractCost(*nodeContract, false, false) // not certified, not rented
		assert.NoError(t, err)
		assert.Equal(t, 16.2, result)
	})
}

func TestCalculateUniqueNameCost(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sub := mocks.NewMockSubstrateExt(ctrl)
	identity, err := substrate.NewIdentityFromSr25519Phrase("//Alice")
	assert.NoError(t, err)

	calculator := NewCalculator(sub, identity)

	t.Run("success case with no discount", func(t *testing.T) {
		// Setup: 1000 units as the unique name cost in pricing policy
		sub.EXPECT().GetPricingPolicy(defaultPricingPolicyID).Return(substrate.PricingPolicy{
			ID: 1,
			UniqueName: substrate.Policy{
				Value: 1000,
			},
		}, nil)

		// Expect CalculatePricesAfterDiscount to be called
		sub.EXPECT().GetPricingPolicy(defaultPricingPolicyID).Return(substrate.PricingPolicy{
			ID:                     1,
			DedicatedNodesDiscount: 20,
		}, nil)

		// Expect GetBalance to be called
		sub.EXPECT().GetBalance(identity).Return(substrate.Balance{
			Free: types.U128{
				Int: big.NewInt(0), // No balance, so no discount
			},
		}, nil)

		// Expect GetTFTBillingRate to be called for the conversion in USDtoTFT
		sub.EXPECT().GetTFTBillingRate().Return(types.U32(5), nil)

		// Calculate expected result: 1000 * 24 * 30 / 10^7 = 0.072 USD per month
		expected := 0.072

		result, err := calculator.calculateUniqueNameCost()
		assert.NoError(t, err)
		assert.Equal(t, expected, result, "Expected %v but got %v", expected, result)
	})

	t.Run("verify discount is applied", func(t *testing.T) {
		// Set up a separate test controller and mock for this test
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		sub := mocks.NewMockSubstrateExt(ctrl)

		// Create two calculators - one with identity (which allows discount)
		// and one without identity (which won't apply discount)
		calcWithIdentity := NewCalculator(sub, identity)

		// We'll call both and verify the with-identity version costs less

		// Setup pricing policy mock - will be called twice
		sub.EXPECT().GetPricingPolicy(defaultPricingPolicyID).Return(substrate.PricingPolicy{
			ID: 1,
			UniqueName: substrate.Policy{
				Value: 1000,
			},
			DedicatedNodesDiscount: 20,
		}, nil).Times(2)

		// For calculator with identity, also expect balance and TFT price calls
		// The exact values don't matter as long as there's a large enough balance
		// for some discount to apply
		sub.EXPECT().GetPricingPolicy(defaultPricingPolicyID).Return(substrate.PricingPolicy{
			ID:                     1,
			DedicatedNodesDiscount: 20,
		}, nil).AnyTimes()

		sub.EXPECT().GetBalance(identity).Return(substrate.Balance{
			Free: types.U128{
				Int: big.NewInt(10000000000), // Large enough for some discount
			},
		}, nil).AnyTimes()

		sub.EXPECT().GetTFTBillingRate().Return(types.U32(5), nil).AnyTimes()

		// Run both calculations
		priceWithDiscount, err := calcWithIdentity.calculateUniqueNameCost()
		assert.NoError(t, err)

		// Verify discount was applied by checking that price with discount is less
		assert.Equal(t, 0.0288, priceWithDiscount)
	})

}

func TestCalculateContractCost(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sub := mocks.NewMockSubstrateExt(ctrl)
	identity, err := substrate.NewIdentityFromSr25519Phrase("//Alice")
	assert.NoError(t, err)

	calculator := NewCalculator(sub, identity)

	t.Run("test node with nil node", func(t *testing.T) {
		contract := substrate.Contract{
			ContractID: types.U64(42),
			ContractType: substrate.ContractType{
				IsNodeContract: true,
				NodeContract: substrate.NodeContract{
					Node:           1,
					PublicIPsCount: 1,
				},
			},
		}

		node := substrate.Node{}

		res, err := calculator.calculateContractCost(contract, node)

		assert.Error(t, err)
		assert.Equal(t, 0.0, res)
	})
}
func TestCalculateTotalContractsOverdueOnNode(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sub := mocks.NewMockSubstrateExt(ctrl)

	identity, err := substrate.NewIdentityFromSr25519Phrase("//Alice")
	assert.NoError(t, err)

	calculator := NewCalculator(sub, identity)

	nodeID := uint32(123)

	// all contracts on this node are on rented node
	sub.EXPECT().GetNodeRentContract(nodeID).Return(uint64(1), nil).AnyTimes()
	allowance := time.Hour

	sub.EXPECT().GetBalance(identity).Return(substrate.Balance{
		Free: types.U128{Int: big.NewInt(0)},
	}, nil).AnyTimes()
	sub.EXPECT().GetTFTBillingRate().Return(types.U32(10), nil).AnyTimes()

	node := &substrate.Node{
		ID: types.U32(nodeID),
	}
	sub.EXPECT().GetContractPaymentState(uint64(1)).Return(substrate.ContractPaymentState{
		LastUpdatedSeconds:  types.U64(time.Now().Add(-30 * time.Minute).Unix()),
		StandardOverdraft:   types.U128{Int: big.NewInt(1e7)},
		AdditionalOverdraft: types.U128{},
	}, nil)
	sub.EXPECT().GetContractPaymentState(uint64(2)).Return(substrate.ContractPaymentState{
		LastUpdatedSeconds:  types.U64(time.Now().Add(-30 * time.Minute).Unix()),
		StandardOverdraft:   types.U128{},
		AdditionalOverdraft: types.U128{},
	}, nil)
	sub.EXPECT().GetContractPaymentState(uint64(3)).Return(substrate.ContractPaymentState{
		LastUpdatedSeconds:  types.U64(time.Now().Add(-30 * time.Minute).Unix()),
		StandardOverdraft:   types.U128{Int: big.NewInt(1e7)},
		AdditionalOverdraft: types.U128{},
	}, nil)
	sub.EXPECT().GetContractPaymentState(uint64(4)).Return(substrate.ContractPaymentState{
		LastUpdatedSeconds:  types.U64(time.Now().Add(-30 * time.Minute).Unix()),
		StandardOverdraft:   types.U128{},
		AdditionalOverdraft: types.U128{},
	}, nil)
	sub.EXPECT().GetContractPaymentState(uint64(5)).Return(substrate.ContractPaymentState{
		LastUpdatedSeconds:  types.U64(time.Now().Add(-30 * time.Minute).Unix()),
		StandardOverdraft:   types.U128{Int: big.NewInt(1e7)},
		AdditionalOverdraft: types.U128{},
	}, nil)
	sub.EXPECT().GetNode(nodeID).Return(node, nil).AnyTimes()
	billingInfoWithUnbilled := substrate.ContractBillingInfo{
		AmountUnbilled: types.U64(1e7),
	}
	billingInfoWithoutUnbilled := substrate.ContractBillingInfo{
		AmountUnbilled: types.U64(0),
	}

	sub.EXPECT().GetPricingPolicy(uint32(1)).Return(substrate.PricingPolicy{
		ID:         1,
		CU:         substrate.Policy{Value: 100000}, // 0.1 USD per CU
		SU:         substrate.Policy{Value: 50000},  // 0.2 USD per SU
		NU:         substrate.Policy{Value: 15000},  // 0.03 USD per NU
		IPU:        substrate.Policy{Value: 40000},  // 1.5 USD per IPU
		UniqueName: substrate.Policy{Value: 2500},
	}, nil).AnyTimes()

	nodeContractWithPublicIP := subi.Contract{
		Contract: &substrate.Contract{ContractID: 1,
			State: substrate.ContractState{},
			ContractType: substrate.ContractType{
				IsNodeContract: true,
				NodeContract: substrate.NodeContract{
					Node:           types.U32(nodeID),
					PublicIPsCount: 1,
				},
			},
		},
	}
	sub.EXPECT().GetNodeContractResources(uint64(nodeContractWithPublicIP.ContractID)).Return(substrate.NodeContractResources{
		Used: substrate.Resources{
			CRU: 2,
			MRU: 4 * 1024 * 1024 * 1024, // 4GB
			HRU: 0,
			SRU: 50 * 1024 * 1024 * 1024, // 50GB
		},
	}, nil).AnyTimes()
	nodeContractWithoutPublicIP := subi.Contract{
		Contract: &substrate.Contract{ContractID: 1,
			State: substrate.ContractState{},
			ContractType: substrate.ContractType{
				IsNodeContract: true,
				NodeContract: substrate.NodeContract{
					Node: types.U32(nodeID),
				},
			},
		},
	}
	sub.EXPECT().GetNodeContractResources(uint64(nodeContractWithoutPublicIP.ContractID)).Return(substrate.NodeContractResources{
		Used: substrate.Resources{
			CRU: 2,
			MRU: 4 * 1024 * 1024 * 1024, // 4GB
			HRU: 0,
			SRU: 50 * 1024 * 1024 * 1024, // 50GB
		},
	}, nil).AnyTimes()
	t.Run("success case with multiple contracts", func(t *testing.T) {

		contracts := []types.U64{types.U64(1), types.U64(2), types.U64(3), types.U64(4), types.U64(5)}

		sub.EXPECT().GetContract(uint64(1)).Return(nodeContractWithPublicIP, nil)
		sub.EXPECT().GetContract(uint64(2)).Return(nodeContractWithoutPublicIP, nil)
		sub.EXPECT().GetContract(uint64(3)).Return(nodeContractWithPublicIP, nil)
		sub.EXPECT().GetContract(uint64(4)).Return(nodeContractWithoutPublicIP, nil)
		sub.EXPECT().GetContract(uint64(5)).Return(nodeContractWithPublicIP, nil)

		sub.EXPECT().GetNodeContracts(nodeID).Return(contracts, nil)

		sub.EXPECT().GetContractBillingInfo(uint64(1)).Return(billingInfoWithUnbilled, nil).AnyTimes()
		sub.EXPECT().GetContractBillingInfo(uint64(2)).Return(billingInfoWithoutUnbilled, nil).AnyTimes()
		sub.EXPECT().GetContractBillingInfo(uint64(3)).Return(billingInfoWithUnbilled, nil).AnyTimes()
		sub.EXPECT().GetContractBillingInfo(uint64(4)).Return(billingInfoWithoutUnbilled, nil).AnyTimes()
		sub.EXPECT().GetContractBillingInfo(uint64(5)).Return(billingInfoWithUnbilled, nil).AnyTimes()

		expectedTotal := int64(306)

		// Call the function being tested
		totalCost, err := calculator.calculateTotalContractsOverdueOnNode(nodeID, allowance)

		// Verify results
		assert.NoError(t, err)
		assert.Equal(t, expectedTotal, totalCost, "Expected total cost to be %v, got %v", expectedTotal, totalCost)
	})

	// Test case: multiple errors from different contracts
	t.Run("multiple errors from different contracts", func(t *testing.T) {
		contracts := []types.U64{types.U64(1), types.U64(2), types.U64(3)}
		sub.EXPECT().GetNodeContracts(nodeID).Return(contracts, nil)

		// Setup first and third contracts to cause errors
		sub.EXPECT().GetContract(uint64(1)).Return(subi.Contract{}, assert.AnError)
		sub.EXPECT().GetContract(uint64(3)).Return(subi.Contract{}, assert.AnError)

		sub.EXPECT().GetContract(uint64(2)).Return(nodeContractWithoutPublicIP, nil)
		sub.EXPECT().GetContractBillingInfo(uint64(2)).Return(billingInfoWithoutUnbilled, nil).AnyTimes()
		sub.EXPECT().GetContractPaymentState(uint64(2)).Return(substrate.ContractPaymentState{
			LastUpdatedSeconds: types.U64(time.Now().Add(-30 * time.Minute).Unix()),
		}, nil)

		_, err := calculator.calculateTotalContractsOverdueOnNode(nodeID, allowance)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error with contract 1")
		assert.Contains(t, err.Error(), "error with contract 3")
	})

}
