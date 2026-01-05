package app

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/deployment-checker/pkg/api"
	"github.com/threefoldtech/deployment-checker/pkg/config"
	"github.com/threefoldtech/deployment-checker/pkg/db"
	"github.com/threefoldtech/deployment-checker/pkg/grid"
	"github.com/threefoldtech/deployment-checker/pkg/models"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"golang.org/x/sync/semaphore"
)

type App struct {
	cfg          *config.Config
	database     *db.DB
	gridClient   *grid.Client
	apiServer    *api.Server
	cycleWg      sync.WaitGroup
	shuttingDown atomic.Bool
}

func New(cfg *config.Config) (*App, error) {
	database, err := db.New(context.Background(), cfg.TimescaleDB.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	gridClient, err := grid.NewClient(cfg)
	if err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to create grid client: %w", err)
	}

	apiServer := api.NewServer(database, cfg)

	return &App{
		cfg:        cfg,
		database:   database,
		gridClient: gridClient,
		apiServer:  apiServer,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	ticker := time.NewTicker(a.cfg.Interval())
	defer ticker.Stop()

	log.Info().
		Str("interval", a.cfg.Interval().String()).
		Str("network", a.cfg.Grid.Network).
		Msg("Starting deployment checker service")

	apiErrChan := make(chan error, 1)
	go func() {
		if err := a.apiServer.Start(); err != nil && err != http.ErrServerClosed {
			apiErrChan <- err
		}
	}()

	if err := a.runCycle(ctx); err != nil {
		log.Error().Err(err).Msg("Initial cycle failed")
	}

	for {
		select {
		case <-ctx.Done():
			return a.shutdown()
		case err := <-apiErrChan:
			return fmt.Errorf("API server error: %w", err)
		case <-ticker.C:
			if a.shuttingDown.Load() {
				continue // do not start new cycle if shutting down
			}

			if err := a.runCycle(ctx); err != nil {
				log.Error().Err(err).Msg("Cycle failed")
			}
		}
	}
}

func (a *App) shutdown() error {
	log.Info().Msg("Initiating graceful shutdown")

	a.shuttingDown.Store(true) // do not start new cycle/routine if shutting down

	log.Info().Msg("Waiting for current deployments to finish...")
	done := make(chan struct{})
	go func() {
		a.cycleWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Info().Msg("All deployments completed") // shutdown went well
	case <-time.After(a.cfg.ShutdownTimeout()):
		log.Warn().
			Dur("timeout", a.cfg.ShutdownTimeout()).
			Msg("Shutdown timeout reached, some deployments may not have completed")
		// continue shutdown anyway
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := a.apiServer.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("Error shutting down API server")
	} else {
		log.Info().Msg("API server shut down successfully")
	}

	return nil
}

func (a *App) Close() {
	a.database.Close()
	// a.gridClient.Close()
}

func (a *App) runCycle(ctx context.Context) error {
	log.Info().Msg("Starting deployment cycle")

	proxyClient := a.gridClient.GetProxyClient()
	nodes, err := grid.GetNodes(ctx, proxyClient, a.cfg.Nodes)
	if err != nil {
		return fmt.Errorf("failed to get nodes: %w", err)
	}

	if len(nodes) == 0 {
		log.Warn().Msg("No eligible nodes found")
		return nil
	}

	log.Info().
		Int("nodes", len(nodes)).
		Int("max_concurrent", a.cfg.Probe.ConcurrencyLimit).
		Str("workload", a.cfg.Probe.WorkloadSize).
		Msg("Deployment cycle started")

	sem := semaphore.NewWeighted(int64(a.cfg.Probe.ConcurrencyLimit))
	var mu sync.Mutex
	var cycleErrors []error

	cpu, memoryMB, diskMB := a.cfg.GetWorkload()

	for i, node := range nodes {

		a.cycleWg.Add(1)
		go func(idx int, n types.Node) {
			defer a.cycleWg.Done()

			if err := sem.Acquire(ctx, 1); err != nil { // blocks if at capacity
				mu.Lock()
				cycleErrors = append(cycleErrors, fmt.Errorf("failed to acquire semaphore for node %d: %w", n.NodeID, err))
				mu.Unlock()
				return
			}
			defer sem.Release(1)

			if a.shuttingDown.Load() {
				log.Debug().
					Int("node_id", n.NodeID).
					Msg("Shutdown requested, skipping deployment")
				return // do not start new job if shutting down
			}

			log.Debug().
				Int("node_index", idx+1).
				Int("total_nodes", len(nodes)).
				Int("node_id", n.NodeID).
				Int("farm_id", n.FarmID).
				Str("workload", a.cfg.Probe.WorkloadSize).
				Msg("Deploying VM to node")

			timeoutCtx, cancel := context.WithTimeout(ctx, a.cfg.Timeout())
			result, err := a.gridClient.DeployVM(timeoutCtx, uint32(int(n.NodeID)), cpu, memoryMB, diskMB)
			cancel()

			attempt := models.Attempt{
				Time:   time.Now().Unix(),
				NodeID: int64(int(n.NodeID)),
				FarmID: int64(int(n.FarmID)),
				Status: "failed",
			}

			if err != nil {
				errorCode := "unknown_error"
				if result != nil && result.ErrorCode != "" {
					errorCode = result.ErrorCode
					attempt.TotalDurationMs = &result.TotalDurationMs
				}
				attempt.ErrorCode = &errorCode
				log.Debug().
					Err(err).
					Int("node_id", n.NodeID).
					Str("error_code", errorCode).
					Msg("Deployment failed")
			} else if result != nil {
				attempt.Status = "success"
				attempt.TotalDurationMs = &result.TotalDurationMs
				log.Debug().
					Int("node_id", n.NodeID).
					Msg("Deployment succeeded")
			}

			if err := a.database.RecordAttempt(ctx, attempt); err != nil {
				mu.Lock()
				cycleErrors = append(cycleErrors, fmt.Errorf("failed to record attempt for node %d: %w", n.NodeID, err))
				mu.Unlock()
				log.Error().
					Err(err).
					Int("node_id", n.NodeID).
					Msg("Failed to record attempt")
			}
		}(i, node)
	}

	a.cycleWg.Wait()

	log.Info().
		Int("total_nodes", len(nodes)).
		Int("errors", len(cycleErrors)).
		Msg("Deployment cycle completed")
	return nil
}
