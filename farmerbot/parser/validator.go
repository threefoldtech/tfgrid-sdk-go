package parser

import (
	"fmt"
	"slices"

	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/internal"
)

// wrapper for validateInput
func ValidateConfig(input internal.Config, network string) error {
	manager := substrate.NewManager(internal.SubstrateURLs[network]...)
	subConn, err := manager.Substrate()
	if err != nil {
		return err
	}
	defer subConn.Close()
	return validateInput(input, subConn)
}

// validateInput validates that included, excluded and priority nodes are in the farm
func validateInput(input internal.Config, sub internal.Substrate) error {
	nodes, err := sub.GetNodes(input.FarmID)
	if err != nil {
		return fmt.Errorf("couldn't retrieve node for %d : %v", input.FarmID, err)
	}
	nodesMap := make(map[uint32]bool)
	for _, node := range nodes {
		nodesMap[node] = true
	}
	includedNodes := make(map[uint32]bool)
	if len(input.IncludedNodes) != 0 {
		if err := validateIncludedNodes(input.IncludedNodes, input.ExcludedNodes, nodesMap); err != nil {
			return err
		}
		for _, includedNode := range input.IncludedNodes {
			includedNodes[includedNode] = true
		}
	} else {
		for key, value := range nodesMap {
			if slices.Contains(input.ExcludedNodes, key) {
				continue
			}
			includedNodes[key] = value
		}
	}
	if err := validateExcludedNodes(input.ExcludedNodes, nodesMap); err != nil {
		return err
	}
	if err := validatePriorityOrNeverShutdown("priority", input.PriorityNodes, includedNodes); err != nil {
		return err
	}
	if err := validatePriorityOrNeverShutdown("never shutdown", input.NeverShutDownNodes, includedNodes); err != nil {
		return err
	}
	return nil
}

func validateIncludedNodes(included, excluded []uint32, farmNodes map[uint32]bool) error {
	for _, node := range included {
		if _, ok := farmNodes[node]; !ok {
			return fmt.Errorf("included node with id %d doesn't exist in the farm", node)
		}
		if slices.Contains(excluded, node) {
			return fmt.Errorf("cannot include and exclude the same node %d", node)
		}
	}
	return nil
}

func validatePriorityOrNeverShutdown(typeOfValidation string, toBeValidated []uint32, includedNodes map[uint32]bool) error {
	for _, node := range toBeValidated {
		if _, ok := includedNodes[node]; !ok {
			return fmt.Errorf("%s node with id %d doesn't exist in the included nodes ", typeOfValidation, node)
		}
	}
	return nil
}

func validateExcludedNodes(excluded []uint32, farmNodes map[uint32]bool) error {
	for _, node := range excluded {
		if _, ok := farmNodes[node]; !ok {
			return fmt.Errorf("excluded node with id %d doesn't exist in the farm", node)
		}
	}
	return nil
}
