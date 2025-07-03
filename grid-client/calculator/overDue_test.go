package calculator

import (
	"math/big"
	"testing"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/mocks"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/subi"
)

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
	sub.EXPECT().GetTFTPrice().Return(types.U32(10), nil).AnyTimes()

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

		contracts := []types.U64{types.U64(1), types.U64(2)}

		sub.EXPECT().GetContract(uint64(1)).Return(nodeContractWithPublicIP, nil)
		sub.EXPECT().GetContract(uint64(2)).Return(nodeContractWithoutPublicIP, nil)

		sub.EXPECT().GetNodeContracts(nodeID).Return(contracts, nil)

		sub.EXPECT().GetContractBillingInfo(uint64(1)).Return(billingInfoWithUnbilled, nil).AnyTimes()
		sub.EXPECT().GetContractBillingInfo(uint64(2)).Return(billingInfoWithoutUnbilled, nil).AnyTimes()

		expectedTotal := int64(102)

		// Call the function being tested
		totalCost, err := calculator.calculateTotalContractsOverdueOnNode(nodeID, allowance)

		// Verify results
		assert.NoError(t, err)
		assert.Equal(t, expectedTotal, totalCost, "Expected total cost to be %v, got %v", expectedTotal, totalCost)
	})
}
