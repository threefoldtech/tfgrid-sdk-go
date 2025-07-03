// package main

// import (
// 	"fmt"
// 	"log"
// 	"os"
// 	"strconv"
// 	"time"

// 	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
// 	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/calculator"
// 	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/subi"
// )

// func main() {
// 	if len(os.Args) != 2 {
// 		log.Fatalf("Usage: %s <contract_id>\n", os.Args[0])
// 	}

// 	// Parse contract ID from command line arguments
// 	contractID, err := strconv.ParseUint(os.Args[1], 10, 64)
// 	if err != nil {
// 		log.Fatalf("Failed to parse contract ID: %v", err)
// 	}

// 	// Connect to substrate
// 	substrateURL := "wss://tfchain.dev.grid.tf" // Dev net URL - replace as needed
// 	manager := subi.NewManager(substrateURL)

// 	// Get substrate client
// 	subImpl, err := manager.SubstrateExt()
// 	if err != nil {
// 		log.Fatalf("Failed to get substrate client: %v", err)
// 	}
// 	defer subImpl.Close()

// 	// Get contract information
// 	// contractResult, err := subImpl.GetContract(contractID)
// 	// if err != nil {
// 	// 	log.Fatalf("Failed to get contract %d: %v", contractID, err)
// 	// }

// 	// // Extract node ID from the contract
// 	// nodeID, err := getNodeIDFromContract(contractResult.Contract)
// 	// if err != nil {
// 	// 	log.Fatalf("Failed to get node ID from contract: %v", err)
// 	// }

// 	// fmt.Printf("Contract %d is for node %d\n", contractID, nodeID)

// 	// // Get node information
// 	// node, err := subImpl.GetNode(nodeID)
// 	// if err != nil {
// 	// 	log.Fatalf("Failed to get node %d: %v", nodeID, err)
// 	// }

// 	// Create calculator
// 	calc := calculator.NewCalculator(subImpl, nil) // No identity needed for cost calculation

// 	// Calculate rent cost
// 	cost, err := calc.CalculateContractOverdue(contractID, time.Hour)
// 	if err != nil {
// 		log.Fatalf("Failed to calculate rent cost: %v", err)
// 	}

// 	// Convert cost to TFT
// 	// tftAmount, err := calc.USDtoTFT(cost)
// 	// if err != nil {
// 	// 	log.Fatalf("Failed to convert USD to TFT: %v", err)
// 	// }

// 	// Format TFT amount for display

// 	fmt.Printf("\nRent Cost Results:\n")
// 	fmt.Printf(" contractOverDue TFT %v \n", cost)
// 	// fmt.Printf("  Monthly cost in TFT: %f TFT\n", tftAmount)

// 	// Calculate the breakdown
// 	fmt.Printf("\nCost Breakdown:\n")

// }

// // getNodeIDFromContract returns the node ID of a contract
// func getNodeIDFromContract(contract *substrate.Contract) (uint32, error) {
// 	if contract.ContractType.IsNodeContract {
// 		return uint32(contract.ContractType.NodeContract.Node), nil
// 	}
// 	if contract.ContractType.IsRentContract {
// 		return uint32(contract.ContractType.RentContract.Node), nil
// 	}
// 	return 0, fmt.Errorf("contract id %d is not a node contract nor rent contract", contract.ContractID)
// }
