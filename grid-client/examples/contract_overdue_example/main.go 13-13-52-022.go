package main

import (
	"fmt"
	"log"

	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/calculator"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/subi"
)

func main() {
	// Create a substrate connection
	// Note: Replace "wss://your-substrate-endpoint" with the actual endpoint URL
	subManager := subi.NewManager("wss://tfchain.dev.grid.tf")

	// Get the SubstrateExt implementation
	subConn, err := subManager.SubstrateExt()
	if err != nil {
		log.Fatalf("Failed to create substrate connection: %v", err)
	}

	// Create an identity (replace with your own mnemonic)
	// WARNING: Never expose your mnemonic phrase in production code
	identity, err := substrate.NewIdentityFromSr25519Phrase("energy garage female trend guard pipe skill dumb drill defy crush genuine")
	if err != nil {
		log.Fatalf("Failed to create identity: %v", err)
	}

	// Initialize the calculator
	calc := calculator.NewCalculator(subConn, identity)
	fmt.Println(calc.CalculateDiscount(200))
	// cost, err := calc.CalculateCost(8, 32, 0, 50, true, true)
	// if err != nil {
	// 	log.Fatalf("Failed to calculate cost: %v", err)
	// }
	// fmt.Println(cost)
	// fmt.Println(calc.CalculateDiscount(cost))
	// Example contract ID
	// contractID := uint64(12345)

	// Call ContractOverDue function
	// overdueAmount, err := calc.CalculateContractOverDue(contractID)
	// if err != nil {
	// 	log.Fatalf("Error calculating overdue amount: %v", err)
	// }

	// Print the result - the Int field of U128 is a *big.Int
	// fmt.Printf("Overdue amount for contract %d: %s\n", contractID, overdueAmount.Int.String())

	// Example of working with the returned U128 value
	// For example, checking if the overdue amount is greater than zero
	// if overdueAmount.Int.Cmp(big.NewInt(0)) > 0 {
	// 	fmt.Println("Contract has an overdue amount that needs to be paid")
	// } else {
	// 	fmt.Println("Contract is up to date with payments")
	// }
}
