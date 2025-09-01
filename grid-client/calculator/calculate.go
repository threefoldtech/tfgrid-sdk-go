package calculator

import (
	"fmt"
	"math"
	"math/big"
	"sync"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/hashicorp/go-multierror"
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

var ErrContractDeleted = errors.New("contract is deleted")

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
func (c *Calculator) CalculateCost(cru, mru, hru, sru uint64, publicIP, certified bool) (float64, error) {

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
	// cost per month in unit-USD
	costPerMonth := (cu*float64(pricingPolicy.CU.Value) + su*float64(pricingPolicy.SU.Value) + ipv4*float64(pricingPolicy.IPU.Value)) * certifiedFactor * 24 * 30
	// convert to USD
	return unitToUSD(costPerMonth), nil
}

// CalculatePricesAfterDiscount calculates the prices after discount
// it takes the cost in USD and returns the prices after discount
func (c *Calculator) CalculatePricesAfterDiscount(cost float64) (dedicatedPrice, sharedPrice float64, err error) {
	pricingPolicy, err := c.substrateConn.GetPricingPolicy(defaultPricingPolicyID)
	if err != nil {
		return
	}

	sharedPrice = cost
	discount := float64(pricingPolicy.DedicatedNodesDiscount)
	dedicatedPrice = cost - cost*(discount/100)

	if c.identity == nil {
		return
	}
	accountBalance, err := c.substrateConn.GetBalance(c.identity)
	if err != nil {
		return
	}

	balanceTFT := unitToTFT(accountBalance.Free.Int)

	balanceUSD, err := c.TFTtoUSD(balanceTFT)
	if err != nil {
		return
	}

	sharedDiscount, dedicatedDiscount := getApplicableDiscount(balanceUSD, dedicatedPrice, sharedPrice)

	dedicatedPrice = dedicatedPrice - dedicatedPrice*dedicatedDiscount
	sharedPrice = sharedPrice - sharedPrice*sharedDiscount

	return
}

func getApplicableDiscount(balance float64, dedicatedPrice float64, sharedPrice float64) (bestSharedDiscount, bestDedicatedDiscount float64) {
	packages := []struct {
		name     string
		duration float64
		discount float64
	}{
		{name: "none", duration: 0, discount: 0},
		{name: "default", duration: 1.5, discount: 20},
		{name: "bronze", duration: 3, discount: 30},
		{name: "silver", duration: 6, discount: 40},
		{name: "gold", duration: 18, discount: 60},
	}

	var bestSharedDiscountValue, bestDedicatedDiscountValue float64 = 0, 0

	for _, pkg := range packages {
		sharedThreshold := sharedPrice * pkg.duration
		dedicatedThreshold := dedicatedPrice * pkg.duration

		if balance > sharedThreshold {
			bestSharedDiscountValue = pkg.discount
		}

		if balance > dedicatedThreshold {
			bestDedicatedDiscountValue = pkg.discount
		}
	}

	return bestSharedDiscountValue / 100, bestDedicatedDiscountValue / 100
}

func calculateSU(hru, sru uint64) float64 {
	return float64(hru)/1200 + float64(sru)/200
}

func calculateCU(cru, mru uint64) float64 {

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

// Calculates the cost of a public IP per month in USD.
func (c *Calculator) calculateIPV4CostPerMonth() (float64, error) {
	// cost in unit-USD per month
	pricingPolicy, err := c.substrateConn.GetPricingPolicy(defaultPricingPolicyID)
	if err != nil {
		return 0, err
	}
	// cost in unit-USD per month
	monthlyCost := float64(pricingPolicy.IPU.Value) * 24 * 30

	return unitToUSD(monthlyCost), nil
}

// Calculates the overdue amount in TFT.
//
// The overdue amount basically is the sum of three parts:
//
//	1- Total over draft: is the sum of additional overdraft and standard overdraft.
//	2- Unbilled NU: is the unbilled amount of network usage.
//	3- The estimated cost of the contract for the total period: this part is dependant on the contract type and if the contract is on rented node or not.
//
// If the contract is rent contract, will add both of ipv4 cost and the total overdue of all associated contracts.
// The total period is the time since the last billing added to Allowance period.
// The resulting overdue amount represents the amount that needs to be addressed.
func (c Calculator) CalculateContractOverdue(id uint64, allowance time.Duration) (int64, error) {
	contract, err := c.substrateConn.GetContract(id)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to get contract ID %d", id)
	}

	if contract.IsDeleted() {
		return 0, ErrContractDeleted
	}
	contractInfo := contract.Contract

	contractPaymentState, err := c.substrateConn.GetContractPaymentState(id)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to get payment state for contract ID %d", id)
	}

	lastBillingAt := time.Unix(int64(contractPaymentState.LastUpdatedSeconds), 0)

	// totalOverDraft represents the sum of standard and additional overdraft amounts for the contract TFT
	totalOverDraftTFT := calculateTotalOverdraftTFT(&contractPaymentState)

	node, err := c.getNode(contractInfo.ContractType)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get node")
	}

	var isCertifiedNode bool
	if node != nil {
		isCertifiedNode = node.Certification.IsCertified
	}

	unbilledNuTFT, err := c.getUnbilledAmountInTFT(contractInfo, isCertifiedNode)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get unbilled amount")
	}
	var nodeInfo substrate.Node
	if node != nil {
		nodeInfo = *node
	}
	periodCostTFT, err := c.calculatePeriodCostTFT(lastBillingAt, *contractInfo, nodeInfo, allowance)
	if err != nil {
		return 0, errors.Wrap(err, "failed to calculate period cost")
	}
	totalOverDraftTFT += periodCostTFT + unbilledNuTFT

	if contract.ContractType.IsRentContract {
		// add all contracts overdue on a node
		totalContractsCost, err := c.calculateTotalContractsOverdueOnNode(uint32(contract.ContractType.RentContract.Node), allowance)
		if err != nil {
			return 0, errors.Wrap(err, "failed to calculate total contracts overdue on node")
		}
		totalOverDraftTFT += float64(totalContractsCost)
	}

	return int64(math.Ceil(totalOverDraftTFT)), nil

}

// unitToUSD converts unit-USD to USD as float64
func unitToUSD(units float64) float64 {
	return units / UnitFactor
}

// unitToTFT converts unit-TFT (big.Int) to TFT
func unitToTFT(units *big.Int) float64 {
	result := new(big.Float).SetInt(units)
	val, _ := result.Quo(result, big.NewFloat(UnitFactor)).Float64()
	return val
}

// GetUnbilledAmountInTFT returns the amount unbilled for a given contract in TFT
//
// The amount unbilled is the amount that is not billed yet for a node contract
func (c *Calculator) getUnbilledAmountInTFT(contract *substrate.Contract, isCertifiedNode bool) (float64, error) {
	if contract.ContractType.IsNameContract || contract.ContractType.IsRentContract ||
		(contract.ContractType.IsNodeContract && contract.ContractType.NodeContract.PublicIPsCount == 0) {
		return 0, nil
	}
	billingInfo, err := c.substrateConn.GetContractBillingInfo(uint64(contract.ContractID))
	if err != nil && !errors.Is(err, substrate.ErrNotFound) {
		return 0, err
	}

	var unbilled float64 = 0
	if billingInfo.AmountUnbilled != types.U64(0) {
		unbilled = float64(billingInfo.AmountUnbilled)
	}
	// amount unbilled is in unit-USD
	unbilledUSD := unitToUSD(unbilled)
	if isCertifiedNode {
		unbilledUSD *= 1.25
	}
	return c.USDtoTFT(unbilledUSD)
}

// Calculates the total overdue of contracts on a node in TFT
func (c *Calculator) calculateTotalContractsOverdueOnNode(nodeID uint32, allowance time.Duration) (int64, error) {
	contracts, err := c.substrateConn.GetNodeContracts(nodeID)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to get contracts for node ID %d", nodeID)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	wg.Add(len(contracts))

	var totalCost int64 = 0
	var result *multierror.Error

	for _, contract := range contracts {
		go func(contract uint64) {
			defer wg.Done()
			cost, err := c.CalculateContractOverdue(contract, allowance)
			if err != nil {
				if errors.Is(err, ErrContractDeleted) {
					return
				}
				mu.Lock()
				result = multierror.Append(result, fmt.Errorf("error with contract %d: %w", contract, err))
				mu.Unlock()
				return
			}
			mu.Lock()
			totalCost += cost
			mu.Unlock()
		}(uint64(contract))
	}

	wg.Wait()

	if result != nil {
		return 0, result.ErrorOrNil()
	}
	return totalCost, nil
}

// Calculates the cost with a period in TFT.
//
// The period is the time since last updated in seconds with the provided allowance time.
func (c *Calculator) calculatePeriodCostTFT(lastUpdatedSeconds time.Time, contract substrate.Contract, node substrate.Node, allowance time.Duration) (float64, error) {

	// Calculate the elapsed seconds since last billing
	elapsedSeconds := math.Ceil(time.Since(lastUpdatedSeconds).Seconds())
	totalPeriodSeconds := elapsedSeconds + allowance.Seconds()

	contractMonthlyCostUSD, err := c.calculateContractCost(contract, node)
	if err != nil {
		return 0, errors.Wrap(err, "failed to calculate contract cost")
	}

	contractMonthlyCostTFT, err := c.USDtoTFT(contractMonthlyCostUSD)
	if err != nil {
		return 0, errors.Wrap(err, "failed to convert contract cost to TFT")
	}
	secondsPerMonth := 30 * 24 * 60 * 60 // 30 days * 24 hours * 60 minutes * 60 seconds
	contractCostPerSecond := contractMonthlyCostTFT / float64(secondsPerMonth)
	return contractCostPerSecond * totalPeriodSeconds, nil

}

// Calculates the cost of a contract per month in USD.
func (c *Calculator) calculateContractCost(contract substrate.Contract, node substrate.Node) (float64, error) {
	if contract.ContractType.IsNameContract {
		return c.calculateUniqueNameCost()
	}

	if node.ID == 0 {
		return 0, errors.New("Invalid node")
	}

	if contract.ContractType.IsNodeContract {

		nodeRentContract, err := c.substrateConn.GetNodeRentContract(uint32(node.ID))
		if err != nil {
			if errors.Is(err, substrate.ErrNotFound) {
				return 0, nil
			}
			return 0, err
		}

		return c.calculateNodeContractCost(contract, node.Certification.IsCertified, nodeRentContract > 0)
	}

	if contract.ContractType.IsRentContract {

		return c.calculateRentCost(contract, node)
	}
	return 0, nil
}

// Calculates the cost of a unique name per month in USD.
func (c *Calculator) calculateUniqueNameCost() (float64, error) {

	pricingPolicy, err := c.substrateConn.GetPricingPolicy(defaultPricingPolicyID)
	if err != nil {
		return 0, err
	}
	// cost in unit-USD
	monthlyCost := float64(pricingPolicy.UniqueName.Value) * 24 * 30

	costUSD := unitToUSD(monthlyCost)

	_, afterDiscount, err := c.CalculatePricesAfterDiscount(costUSD)
	if err != nil {
		return costUSD, nil
	}
	return afterDiscount, nil
}

// Calculates the cost of a node contract per month in USD.
//
// There are two cases for node contract cost:
//  1. Node contract on shared node: the cost of the used resources of (shared)
//  2. Node contract on rented node: the cost of the IPV4 only if the contact includes ipv4, else it will return zero.
func (c *Calculator) calculateNodeContractCost(contract substrate.Contract, onCertifiedNode, isOnRentedNode bool) (float64, error) {
	if !contract.ContractType.IsNodeContract {
		return 0, fmt.Errorf("contract ID %d is not a node contract", contract.ContractID)
	}
	publicIPsCount := contract.ContractType.NodeContract.PublicIPsCount

	// Node contract on rented node
	if isOnRentedNode {
		if publicIPsCount == 0 {
			return 0, nil
		}
		cost, err := c.calculateIPV4CostPerMonth()
		if err != nil {
			return 0, err
		}
		totalCost := cost * float64(publicIPsCount)
		if onCertifiedNode {
			totalCost *= 1.25
		}

		_, sharedPrice, err := c.CalculatePricesAfterDiscount(totalCost)
		if err != nil {
			return totalCost, err
		}
		return sharedPrice, nil
	}

	// Normal node contract on sharedNode

	resources, err := c.substrateConn.GetNodeContractResources(uint64(contract.ContractID))
	if err != nil {
		return 0, err
	}
	CRU := uint64(resources.Used.CRU)
	MRU := convertBytesToGB(resources.Used.MRU)
	HRU := convertBytesToGB(resources.Used.HRU)
	SRU := convertBytesToGB(resources.Used.SRU)

	cost, err := c.CalculateCost(CRU, MRU, HRU, SRU, publicIPsCount > 0, onCertifiedNode)
	if err != nil {
		return 0, err
	}
	_, sharedPrice, err := c.CalculatePricesAfterDiscount(cost)
	if err != nil {
		return 0, err
	}
	return sharedPrice, nil
}

// Calculates the cost of a rent contract per month in USD.
//
// Rent contract cost is the cost of the node (dedicated discount applied) + the node extra fee
func (c *Calculator) calculateRentCost(contract substrate.Contract, node substrate.Node) (float64, error) {
	if !contract.ContractType.IsRentContract {
		return 0, fmt.Errorf("contract ID %d is not a rent contract", contract.ContractID)
	}

	CRU := uint64(node.Resources.CRU)
	MRU := convertBytesToGB(node.Resources.MRU)
	HRU := convertBytesToGB(node.Resources.HRU)
	SRU := convertBytesToGB(node.Resources.SRU)

	isCertified := node.Certification.IsCertified

	cost, err := c.CalculateCost(CRU, MRU, HRU, SRU, false, isCertified)
	if err != nil {
		return 0, err
	}

	//apply stacking discount
	dedicatedPrice, _, err := c.CalculatePricesAfterDiscount(cost)
	if err != nil {
		return 0, err
	}

	// GetNodeExtraFee, this will be in mUSD
	extraFee, err := c.substrateConn.GetDedicatedNodePrice(uint32(contract.ContractID))
	if err != nil {
		return 0, errors.Wrap(err, "failed to get dedicated node extra fee")
	}
	dedicatedPrice += (float64(extraFee) / mUSDToUSD)
	return dedicatedPrice, nil
}

// convertBytesToGB converts bytes to gigabytes by dividing by 1024^3
func convertBytesToGB(bytes types.U64) uint64 {
	return uint64(bytes) / 1024 / 1024 / 1024
}

// TFTtoUSD converts TFT amount to USD based on the current price
func (c *Calculator) TFTtoUSD(tft float64) (float64, error) {
	tftPrice, err := c.substrateConn.GetTFTBillingRate()
	if err != nil {
		return 0, errors.Wrap(err, "failed to get TFT price")
	}
	return tft * (float64(tftPrice) / mUSDToUSD), nil
}

// USDtoTFT converts USD amount to TFT based on the current price
func (c *Calculator) USDtoTFT(usd float64) (float64, error) {
	tftPrice, err := c.substrateConn.GetTFTBillingRate()
	if err != nil {
		return 0, errors.Wrap(err, "failed to get TFT price")
	}
	// convert from unit-USD to TFT
	tftPriceUSD := float64(tftPrice) / mUSDToUSD
	return usd / tftPriceUSD, nil
}

// calculateTotalOverdraftTFT calculates the total overdraft from payment state in TFT
func calculateTotalOverdraftTFT(paymentState *substrate.ContractPaymentState) float64 {
	totalOverDraft := types.U128{Int: big.NewInt(0)}

	if paymentState.StandardOverdraft.Int != nil {
		totalOverDraft.Add(paymentState.StandardOverdraft.Int, totalOverDraft.Int)
	}

	if paymentState.AdditionalOverdraft.Int != nil {
		totalOverDraft.Add(paymentState.AdditionalOverdraft.Int, totalOverDraft.Int)
	}
	return unitToTFT(totalOverDraft.Int)
}

func (c Calculator) getNode(contractType substrate.ContractType) (node *substrate.Node, err error) {
	if contractType.IsNodeContract {
		return c.substrateConn.GetNode(uint32(contractType.NodeContract.Node))
	}
	if contractType.IsRentContract {
		return c.substrateConn.GetNode(uint32(contractType.RentContract.Node))
	}

	// contract type is not a node contract nor rent contract, and that is fine as the node will not be used in name contract
	return nil, nil
}
