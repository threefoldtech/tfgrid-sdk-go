// package main

// import (
// 	"fmt"
// 	"math"
// )

// // Resource pricing in mUSD
// type Pricing struct {
// 	CUPrice float64 // price per compute unit per hour
// 	SUPrice float64 // price per storage unit per hour
// }

// // calculateCU calculates compute units based on the formula from the calculator package
// func calculateCU(cru, mru int64) float64 {
// 	MruUsed1 := float64(mru) / 4
// 	CruUsed1 := float64(cru) / 2
// 	cu1 := math.Max(MruUsed1, CruUsed1)

// 	MruUsed2 := float64(mru) / 8
// 	CruUsed2 := float64(cru)
// 	cu2 := math.Max(MruUsed2, CruUsed2)

// 	MruUsed3 := float64(mru) / 2
// 	CruUsed3 := float64(cru) / 4
// 	cu3 := math.Max(MruUsed3, CruUsed3)

// 	cu := math.Min(cu1, cu2)
// 	cu = math.Min(cu, cu3)

// 	return cu
// }

// // calculateSU calculates storage units based on SRU
// func calculateSU(sru, hru int64) float64 {
// 	return float64(sru)/200 + float64(hru)/1200 // SU is calculated as SRU/1024 (GB → TB)
// }

// // calculateResourceCost calculates the total cost based on resources and pricing
// func calculateResourceCost(cru, mru, sru int64, pricing Pricing) {
// 	cu := calculateCU(cru, mru)
// 	fmt.Println(cu * 2)
// 	// 	su := calculateSU(sru)

// 	// 	cuCost := cu * pricing.CUPrice
// 	// 	suCost := su * pricing.SUPrice
// 	// 	totalCost := cuCost + suCost

// 	// 	return cuCost, suCost, totalCost
// 	// }

// }
// func main() {
// 	// Define command-line flags
// 	cru := 8
// 	mru := 32
// 	sru := 50
// 	cuPrice := 2
// 	suPrice := 2
// 	ipPrice := 2
// 	ipv4 := 1
// 	cu := calculateCU(int64(cru), int64(mru))
// 	su := calculateSU(int64(sru), int64(0))

// 	totalCost := cu*float64(cuPrice) + su*float64(suPrice) + (float64(ipv4) * float64(ipPrice))
// 	fmt.Printf("ip cost %f\n", float64(ipv4)*float64(ipPrice))
// 	fmt.Printf("total cost in mUSD/hour: %f\n", totalCost)
// 	totalCost = totalCost * 24 * 30
// 	fmt.Printf("total cost in mUSD/month: %f\n", totalCost*1.25)
// 	// fmt.Printf("total cost in TFT/month: %f\n", (totalCost/1000)*1.25)

// }
