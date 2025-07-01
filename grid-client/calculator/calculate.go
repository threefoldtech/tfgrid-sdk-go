package calculator

import (
	"fmt"
	"math"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/subi"
)

const defaultPricingPolicyID = uint32(1)

// The price of TFT stored on the TFChain is expressed in mUSD per 1 TFT.
// To convert this to USD, the formula is:
// tft_price_units_usd = tft_price / 1000
const mUSDToUSD = 1000

// UnitFactor represents the smallest unit conversion factor for both USD and TFT
// 1 USD = 10,000,000 unit-USD
// 1 TFT = 10,000,000 unit-TFT (TFT's Planck)
const UnitFactor = 1e7

// Calculator struct for calculating the cost of resources
type Calculator struct {
	substrateConn subi.SubstrateExt
	identity      substrate.Identity
}

// NewCalculator creates a new Calculator
func NewCalculator(substrateConn subi.SubstrateExt, identity substrate.Identity) Calculator {
	return Calculator{substrateConn: substrateConn, identity: identity}
}

// CalculateCost calculates the cost in $ per month of the given resources without a discount
func (c *Calculator) CalculateCost(cru, mru, hru, sru int64, publicIP, certified bool) (float64, error) {

	pricingPolicy, err := c.substrateConn.GetPricingPolicy(defaultPricingPolicyID)
	if err != nil {
		return 0, err
	}

	cu := calculateCU(cru, mru)
	su := calculateSU(hru, sru)

	var ipv4 float64
	if publicIP {
		ipv4 = 1
	}

	certifiedFactor := 1.0
	if certified {
		certifiedFactor = 1.25
	}
	// cost per month in mUSD
	costPerMonth := (cu*float64(pricingPolicy.CU.Value) + su*float64(pricingPolicy.SU.Value) + ipv4*float64(pricingPolicy.IPU.Value)) * certifiedFactor * 24 * 30
	// convert to USD
	return costPerMonth / mUSDToUSD, nil
}

// CalculateDiscount calculates the discount of a given cost
func (c *Calculator) CalculateDiscount(cost float64) (dedicatedPrice, sharedPrice float64, err error) {
	tftPrice, err := c.substrateConn.GetTFTPrice()
	if err != nil {
		return
	}

	pricingPolicy, err := c.substrateConn.GetPricingPolicy(defaultPricingPolicyID)
	if err != nil {
		return
	}

	// discount for shared Nodes
	sharedPrice = cost

	// discount for Dedicated Nodes
	discount := float64(pricingPolicy.DedicatedNodesDiscount)
	dedicatedPrice = cost - cost*(discount/100)

	// discount for Twin Balance in TFT
	accountBalance, err := c.substrateConn.GetBalance(c.identity)
	if err != nil {
		return
	}
	balance := float64(tftPrice) / 1000 * float64(accountBalance.Free.Int64()) * 10000000

	discountPackages := map[string]map[string]float64{
		"none": {
			"duration": 0,
			"discount": 0,
		},
		"default": {
			"duration": 1.5,
			"discount": 20,
		},
		"bronze": {
			"duration": 3,
			"discount": 30,
		},
		"silver": {
			"duration": 6,
			"discount": 40,
		},
		"gold": {
			"duration": 18,
			"discount": 60,
		},
	}

	// check which package will be used according to the balance
	dedicatedPackage := "none"
	sharedPackage := "none"
	for pkg := range discountPackages {
		if balance > dedicatedPrice*discountPackages[pkg]["duration"] {
			dedicatedPackage = pkg
		}
		if balance > sharedPrice*discountPackages[pkg]["duration"] {
			sharedPackage = pkg
		}
	}

	dedicatedPrice = (dedicatedPrice - dedicatedPrice*(discountPackages[dedicatedPackage]["discount"]/100)) / 1e7
	sharedPrice = (sharedPrice - sharedPrice*(discountPackages[sharedPackage]["discount"]/100)) / 1e7

	return
}

func calculateSU(hru, sru int64) float64 {
	return float64(hru)/1200 + float64(sru)/200
}

func calculateCU(cru, mru int64) float64 {

	MruUsed1 := float64(mru) / 4
	CruUsed1 := float64(cru) / 2
	cu1 := math.Max(MruUsed1, CruUsed1)

	MruUsed2 := float64(mru) / 8
	CruUsed2 := float64(cru)
	cu2 := math.Max(MruUsed2, CruUsed2)

	MruUsed3 := float64(mru) / 2
	CruUsed3 := float64(cru) / 4
	cu3 := math.Max(MruUsed3, CruUsed3)

	cu := math.Min(cu1, cu2)
	cu = math.Min(cu, cu3)

	return cu
}

func (c *Calculator) calculateIPV4() (float64, error) {
	pricingPolicy, err := c.substrateConn.GetPricingPolicy(defaultPricingPolicyID)
	if err != nil {
		return 0, err
	}
	// cost in unit-USD
	monthlyCost := pricingPolicy.IPU.Value * 24 * 30

	return float64(monthlyCost) / UnitFactor, nil
}
func (c *Calculator) CalculateUniqueNameCost() (float64, error) {
	pricingPolicy, err := c.substrateConn.GetPricingPolicy(defaultPricingPolicyID)
	if err != nil {
		return 0, err
	}
	// cost in unit-USD
	monthlyCost := float64(pricingPolicy.UniqueName.Value) * 24 * 30
	return float64(monthlyCost) / UnitFactor, nil
}

func (c Calculator) CalculateContractTotalOverdraft(id uint64) (types.U128, error) {
	contract, err := c.substrateConn.GetContract(id)
	if err != nil {
		return types.U128{}, errors.Wrap(err, "failed to get contract")
	}

	if contract.IsDeleted() {
		return types.U128{}, errors.New("contract is deleted")
	}
	// contractCost, err := c.CalculateContractCost(contract.Contract)
	// if err != nil {
	// 	return types.U128{}, err
	// }

	contractPaymentState, err := c.substrateConn.GetContractPaymentState(id)
	if err != nil {
		return types.U128{}, errors.Wrap(err, "failed to get contract payment state")
	}

	if err != nil {
		return types.U128{}, errors.Wrap(err, "failed to get contract billing info")
	}
	// Convert float64 contractCost to big.Int, considering UnitFactor (1e7)
	return types.U128(contractPaymentState.StandardOverdraft), nil
}

func (c *Calculator) CalculateContractCost(contract *substrate.Contract) (float64, error) {
	if contract.ContractType.IsNameContract {
		return c.CalculateUniqueNameCost()
	}

	nodeID, err := getNodeID(contract)
	if err != nil {
		return 0, err
	}

	node, err := c.substrateConn.GetNode(nodeID)
	if err != nil {
		return 0, err
	}

	nodeRentContract, err := c.substrateConn.GetNodeRentContract(nodeID)
	if err != nil && err != substrate.ErrAccountNotFound {
		return 0, errors.Wrap(err, "failed to get node rent contract")
	}

	if contract.ContractType.IsNodeContract {
		return c.CalculateNodeContractCost(contract, node, nodeRentContract > 0)
	}

	if contract.ContractType.IsRentContract {
		return c.CalculateRentCost(contract, *node)
	}
	return 0, nil
}

func (c *Calculator) CalculateNodeContractCost(contract *substrate.Contract, node *substrate.Node, isOnRentedNode bool) (float64, error) {
	if !contract.ContractType.IsNodeContract {
		return 0, fmt.Errorf("contract id %d is not a node contract", contract.ContractID)
	}
	publicIPsCount := contract.ContractType.NodeContract.PublicIPsCount

	isCertified := node.Certification.IsCertified

	/** Node contract on rented node
	 * If the node contract has IPV4 will return the price of the ipv4 per month
	 * If not there is no cost, will return zero
	 */
	if isOnRentedNode {
		if publicIPsCount > 0 {
			cost, err := c.calculateIPV4()
			if err != nil {
				return 0, err
			}
			totalCost := cost * float64(publicIPsCount)
			if isCertified {
				totalCost *= 1.25
			}
			return totalCost, nil
		}
		return 0, nil
	}

	// Get the node resources
	resources, err := c.substrateConn.GetNodeContractResources(uint64(contract.ContractID))
	if err != nil {
		return 0, err
	}

	return c.CalculateCost(int64(resources.Used.CRU), int64(resources.Used.MRU), int64(resources.Used.HRU), int64(resources.Used.SRU), publicIPsCount > 0, isCertified)
}

func (c *Calculator) CalculateRentCost(contract *substrate.Contract, node substrate.Node) (float64, error) {

	CRU := node.Resources.CRU
	MRU := convertBytesToGB(uint64(node.Resources.MRU))
	HRU := convertBytesToGB(uint64(node.Resources.HRU))
	SRU := convertBytesToGB(uint64(node.Resources.SRU))

	isCertified := node.Certification.IsCertified

	cost, err := c.CalculateCost(int64(CRU), int64(MRU), int64(HRU), int64(SRU), false, isCertified)
	if err != nil {
		return 0, err
	}
	// GetNodeExtraFee, this will be in Milli USD
	extraFee, err := c.substrateConn.GetDedicatedNodePrice(uint32(contract.ContractID))
	if err != nil {
		return 0, errors.Wrap(err, "failed to get dedicated node extra fee")
	}
	cost += (float64(extraFee) / 1000)
	return cost, nil
}

func getNodeID(contract *substrate.Contract) (uint32, error) {
	if contract.ContractType.IsNodeContract {
		return uint32(contract.ContractType.NodeContract.Node), nil
	}
	if contract.ContractType.IsRentContract {
		return uint32(contract.ContractType.RentContract.Node), nil
	}
	return 0, fmt.Errorf("contract id %d is not a node contract nor rent contract", contract.ContractID)
}

func convertBytesToGB(bytes uint64) int64 {
	return int64(bytes / 1024 / 1024 / 1024)
}
