package probe

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/deployment-checker/pkg/config"
	"github.com/threefoldtech/deployment-checker/pkg/db"
	"github.com/threefoldtech/deployment-checker/pkg/grid"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/client"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"golang.org/x/sync/semaphore"
)

// Cycle manages a deployment cycle
type Cycle struct {
	gridClient *grid.Client
	database   *db.DB
	cfg        *config.Config
	errors     []error
	errorsMu   sync.Mutex
	wg         sync.WaitGroup
}

// NewCycle creates a new deployment cycle
func NewCycle(gridClient *grid.Client, database *db.DB, cfg *config.Config) *Cycle {
	return &Cycle{
		gridClient: gridClient,
		database:   database,
		cfg:        cfg,
		errors:     make([]error, 0),
	}
}

// Run executes a deployment cycle for all eligible nodes
func (c *Cycle) Run(ctx context.Context, proxyClient client.Client, nodes []types.Node, shuttingDown func() bool) error {
	if len(nodes) == 0 {
		log.Warn().Msg("No eligible nodes found")
		return nil
	}

	log.Info().
		Int("nodes", len(nodes)).
		Int("max_concurrent", c.cfg.Probe.ConcurrencyLimit).
		Str("workload", c.cfg.Probe.WorkloadSize).
		Msg("Deployment cycle started")

	sem := semaphore.NewWeighted(int64(c.cfg.Probe.ConcurrencyLimit))

	jitterMin := c.cfg.JitterMinSeconds()
	jitterMax := c.cfg.JitterMaxSeconds()

	for _, node := range nodes {
		c.wg.Add(1)
		go func(n types.Node) {
			defer c.wg.Done()

			if err := sem.Acquire(ctx, 1); err != nil {
				c.addError(fmt.Errorf("failed to acquire semaphore for node %d: %w", n.NodeID, err))
				return
			}
			defer sem.Release(1)

			if shuttingDown != nil && shuttingDown() {
				log.Debug().
					Int("node_id", n.NodeID).
					Msg("Shutdown requested, skipping deployment")
				return
			}

			if jitter := calculateJitter(jitterMin, jitterMax, n.NodeID); jitter > 0 {
				time.Sleep(jitter)
			}

			log.Info().
				Int("node_id", n.NodeID).
				Int("farm_id", n.FarmID).
				Str("workload", c.cfg.Probe.WorkloadSize).
				Msg("Deploying VM to node")

			cpu, memoryMB, diskMB := c.cfg.GetWorkload()
			timeoutCtx, cancel := context.WithTimeout(ctx, c.cfg.Timeout())
			result, err := c.gridClient.MakeDeployment(timeoutCtx, n, cpu, memoryMB, diskMB)
			cancel()

			attempt := mapResultToAttempt(result, n, err)

			if err := c.database.RecordAttempt(ctx, *attempt); err != nil {
				log.Error().
					Err(err).
					Int("node_id", n.NodeID).
					Msg("Failed to record attempt")
				c.addError(fmt.Errorf("failed to record attempt for node %d: %w", n.NodeID, err))
			}
		}(node)
	}

	c.wg.Wait()

	log.Info().
		Int("total_nodes", len(nodes)).
		Int("errors", len(c.errors)).
		Msg("Deployment cycle completed")

	return nil
}

// GetErrors returns all errors collected during the cycle
func (c *Cycle) GetErrors() []error {
	c.errorsMu.Lock()
	defer c.errorsMu.Unlock()
	return c.errors
}

func (c *Cycle) addError(err error) {
	c.errorsMu.Lock()
	defer c.errorsMu.Unlock()
	c.errors = append(c.errors, err)
}
