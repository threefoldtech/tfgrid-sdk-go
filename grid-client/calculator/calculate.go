package calculator

import (
	"math"

	"github.com/pkg/errors"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/subi"
)

const defaultPricingPolicyID = uint32(1)

// The price of TFT stored on the TFChain is expressed in mUSD per 1 TFT.
// To convert this to USD, the formula is:
// tft_price_units_usd = tft_price / 1000
const mUSDToUSD = 1000

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

// CalculatePricesAfterDiscount calculates the prices after discount
func (c *Calculator) CalculatePricesAfterDiscount(cost float64) (dedicatedPrice, sharedPrice float64, err error) {
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

	BalanceTFT := float64(accountBalance.Free.Int64()) / 1e7

	balance, err := c.TFTtoUSD(BalanceTFT)
	if err != nil {
		return
	}

	sharedDiscount, dedicatedDiscount := getApplicableDiscount(balance, dedicatedPrice, sharedPrice)

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

// TFTtoUSD converts TFT amount to USD based on the current price
func (c *Calculator) TFTtoUSD(tft float64) (float64, error) {
	tftPrice, err := c.substrateConn.GetTFTPrice()
	if err != nil {
		return 0, errors.Wrap(err, "failed to get TFT price")
	}
	return tft * (float64(tftPrice) / mUSDToUSD), nil
}
