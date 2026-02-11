package grid

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/deployment-checker/pkg/config"
	"github.com/threefoldtech/deployment-checker/pkg/retry"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/client"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

const (
	proxyRetryMaxRetries     = 3
	proxyRetryInitialBackoff = 1 * time.Second
	proxyRetryMaxBackoff     = 10 * time.Second
	proxyRetryMultiplier     = 2.0
)

func GetNodes(ctx context.Context, proxyClient client.Client, filters config.NodesConfig) ([]types.Node, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	limit := types.Limit{
		Size: 100,
		Page: 1,
	}
	filter := buildFilters(filters)

	retryCfg := retry.BackoffConfig{
		MaxRetries:     proxyRetryMaxRetries,
		InitialBackoff: proxyRetryInitialBackoff,
		MaxBackoff:     proxyRetryMaxBackoff,
		Multiplier:     proxyRetryMultiplier,
	}

	var allNodes []types.Node
	var total int
	for {
		var nodes []types.Node

		operation := func() error {
			var pageTotal int
			var err error
			nodes, pageTotal, err = proxyClient.Nodes(timeoutCtx, filter, limit)
			if err != nil {
				return fmt.Errorf("failed to query nodes page %d: %w", limit.Page, err)
			}
			if total == 0 {
				total = pageTotal
			}
			return nil
		}

		if err := retry.DoWithBackoff(timeoutCtx, retryCfg, operation); err != nil {
			return nil, fmt.Errorf("failed to get nodes after retries: %w", err)
		}

		allNodes = append(allNodes, nodes...)

		if len(allNodes) >= total || len(nodes) == 0 {
			break
		}

		limit.Page++
	}

	log.Info().Int("count", len(allNodes)).Msg("Found eligible nodes")

	return allNodes, nil
}

func buildFilters(filters config.NodesConfig) types.NodeFilter {
	filter := types.NodeFilter{
		FarmIDs:  filters.Farms,
		NodeIDs:  filters.Nodes,
		Excluded: filters.Exclude,
	}

	switch filters.Status {
	case "up":
		filter.Status = []string{"up"}
	case "healthy":
		filter.Healthy = &[]bool{true}[0]
	}

	rented := false
	filter.Rented = &rented

	inDedicatedFarm := false
	filter.InDedicatedFarm = &inDedicatedFarm

	return filter
}
