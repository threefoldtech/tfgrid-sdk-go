package calculator

import (
	"math"

	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/subi"
)

const defaultPricingPolicyID = uint32(1)

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
	tftPrice, err := c.substrateConn.GetTFTPrice()
	if err != nil {
		return 0, err
	}

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

	costPerMonth := (cu*float64(pricingPolicy.CU.Value) + su*float64(pricingPolicy.SU.Value) + ipv4*float64(pricingPolicy.IPU.Value)) * certifiedFactor * 24 * 30
	return costPerMonth / float64(tftPrice) / 1000, nil
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
	return float64(hru/1200 + sru/200)
}

func calculateCU(cru, mru int64) float64 {
	MruUsed1 := float64(mru / 4)
	CruUsed1 := float64(cru / 2)
	cu1 := math.Max(MruUsed1, CruUsed1)

	MruUsed2 := float64(mru / 8)
	CruUsed2 := float64(cru)
	cu2 := math.Max(MruUsed2, CruUsed2)

	MruUsed3 := float64(mru / 2)
	CruUsed3 := float64(cru / 4)
	cu3 := math.Max(MruUsed3, CruUsed3)

	cu := math.Min(cu1, cu2)
	cu = math.Min(cu, cu3)

	return cu
}
