package parser

import (
	"fmt"
	"slices"

	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/internal"
)

// ValidateInput validates that included, excluded and priority nodes are in the farm
func ValidateInput(input *internal.Config, sub internal.Substrate) error {
	nodes, err := sub.GetNodes(input.FarmID)
	if err != nil {
		return fmt.Errorf("couldn't retrieve node for %d : %v", input.FarmID, err)
	}
	nodesMap := make(map[uint32]bool)
	for _, node := range nodes {
		nodesMap[node] = true
	}

	//validate included nodes
	for _, includedNode := range input.IncludedNodes {
		if _, ok := nodesMap[includedNode]; !ok {
			return fmt.Errorf("included node with id %d doesn't exist in the farm", includedNode)
		}
	}
	//validate excluded nodes
	for _, excludedNode := range input.ExcludedNodes {
		if _, ok := nodesMap[excludedNode]; !ok {
			return fmt.Errorf("excluded node with id %d doesn't exist in the farm", excludedNode)
		}
		if slices.Contains(input.IncludedNodes, excludedNode) {
			return fmt.Errorf("cannot include and exclude the same node %d", excludedNode)
		}
	}

	//validate priority nodes
	for _, priorityNode := range input.PriorityNodes {
		if !slices.Contains(input.IncludedNodes, priorityNode) {
			return fmt.Errorf("priority node with id %d doesn't exist included nodes", priorityNode)
		}
	}
	return nil
}
