package probe

import (
	"context"
	"fmt"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/deployment-checker/pkg/config"
	"github.com/threefoldtech/deployment-checker/pkg/db"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/client"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"golang.org/x/sync/semaphore"
)

// Cycle manages a deployment cycle
type Cycle struct {
	executor  *Executor
	database  *db.DB
	cfg       *config.Config
	collector *Collector
	errors    []error
	errorsMu  sync.Mutex
	wg        sync.WaitGroup
}

// NewCycle creates a new deployment cycle
func NewCycle(executor *Executor, database *db.DB, cfg *config.Config) *Cycle {
	return &Cycle{
		executor:  executor,
		database:  database,
		cfg:       cfg,
		collector: NewCollector(),
		errors:    make([]error, 0),
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

	for i, node := range nodes {
		c.wg.Add(1)
		go func(idx int, n types.Node) {
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

			attempt, err := c.executor.ExecuteDeployment(ctx, n, jitterMin, jitterMax)
			if err != nil {
				c.addError(fmt.Errorf("deployment execution failed for node %d: %w", n.NodeID, err))
				return
			}

			c.collector.Add(*attempt)
		}(i, node)
	}

	c.wg.Wait()

	// Batch insert all attempts
	attempts := c.collector.GetAll()
	if len(attempts) > 0 {
		batchSize := c.cfg.BatchSize()
		if err := c.database.RecordAttemptsBatch(ctx, attempts, batchSize); err != nil {
			log.Error().
				Err(err).
				Int("attempts", len(attempts)).
				Int("batch_size", batchSize).
				Msg("Failed to record attempts batch")
			c.addError(fmt.Errorf("failed to record attempts batch: %w", err))
		} else {
			log.Debug().
				Int("attempts", len(attempts)).
				Int("batch_size", batchSize).
				Msg("Successfully recorded attempts batch")
		}
	}

	log.Info().
		Int("total_nodes", len(nodes)).
		Int("attempts_recorded", len(attempts)).
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
