package parser

import (
	"fmt"
	"slices"

	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/internal"
)

// wrapper for validateInput
func ValidateInput(input internal.Config, network string) error {
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
	if len(input.IncludedNodes) != 0 {
		//validate included nodes
		for _, includedNode := range input.IncludedNodes {
			if _, ok := nodesMap[includedNode]; !ok {
				return fmt.Errorf("included node with id %d doesn't exist in the farm", includedNode)
			}
			if slices.Contains(input.ExcludedNodes, includedNode) {
				return fmt.Errorf("cannot include and exclude the same node %d", includedNode)
			}
		}
		//validate priority nodes in case of included nodes
		for _, priorityNode := range input.PriorityNodes {
			if !slices.Contains(input.IncludedNodes, priorityNode) {
				return fmt.Errorf("priority node with id %d doesn't exist included nodes", priorityNode)
			}
		}
		//validate never shutdown nodes in case of included nodes
		for _, neverShutdownNode := range input.NeverShutDownNodes {
			if !slices.Contains(input.IncludedNodes, neverShutdownNode) {
				return fmt.Errorf("never shutdown node with id %d doesn't exist included nodes", neverShutdownNode)
			}
		}
	} else {
		//validate priority nodes in case of all nodes are included
		for _, priorityNode := range input.PriorityNodes {
			if _, ok := nodesMap[priorityNode]; !ok {
				return fmt.Errorf("priority node with id %d doesn't exist in the farm", priorityNode)
			}
			if slices.Contains(input.ExcludedNodes, priorityNode) {
				return fmt.Errorf("cannot priortize and exclude the same node %d", priorityNode)
			}
		}
		//validate never shutdown nodes in case of all nodes are included
		for _, neverShutdownNode := range input.NeverShutDownNodes {
			if _, ok := nodesMap[neverShutdownNode]; !ok {
				return fmt.Errorf("never shutdown node with id %d doesn't exist in the farm", neverShutdownNode)
			}
			if slices.Contains(input.ExcludedNodes, neverShutdownNode) {
				return fmt.Errorf("cannot never shutdown and exclude the same node %d", neverShutdownNode)
			}
		}
	}
	//validate excluded nodes
	for _, excludedNode := range input.ExcludedNodes {
		if _, ok := nodesMap[excludedNode]; !ok {
			return fmt.Errorf("excluded node with id %d doesn't exist in the farm", excludedNode)
		}
	}

	return nil
}
