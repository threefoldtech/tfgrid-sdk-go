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
	"github.com/threefoldtech/deployment-checker/pkg/probe"
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

	// Setup data retention
	ctx := context.Background()
	retentionDays := cfg.RetentionDays()
	if retentionDays > 0 {
		if cfg.UseTimescaleDBRetention() {
			// Use TimescaleDB retention policies (preferred)
			if err := database.SetupRetentionPolicy(ctx, retentionDays); err != nil {
				log.Warn().
					Err(err).
					Int("retention_days", retentionDays).
					Msg("Failed to setup TimescaleDB retention policy, will use manual cleanup")
				// Fallback to manual cleanup
				if err := database.CleanupOldData(ctx, retentionDays); err != nil {
					log.Warn().
						Err(err).
						Msg("Failed to perform initial manual cleanup")
				} else {
					log.Info().
						Int("retention_days", retentionDays).
						Msg("Performed initial manual cleanup of old data")
				}
			} else {
				log.Info().
					Int("retention_days", retentionDays).
					Msg("TimescaleDB retention policy configured successfully")
			}
		} else {
			// Use manual cleanup
			if err := database.CleanupOldData(ctx, retentionDays); err != nil {
				log.Warn().
					Err(err).
					Msg("Failed to perform initial manual cleanup")
			} else {
				log.Info().
					Int("retention_days", retentionDays).
					Msg("Performed initial manual cleanup of old data")
			}
		}
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

	// Cleanup orphaned contracts on startup if enabled
	if a.cfg.CleanupOnStartup() {
		log.Info().Msg("Running startup cleanup of orphaned contracts")
		if err := a.gridClient.CleanupOrphanedContracts(ctx); err != nil {
			log.Warn().Err(err).Msg("Failed to cleanup orphaned contracts on startup")
		}
	}

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

	// Create probe executor and cycle
	executor := probe.NewExecutor(a.gridClient, a.cfg)
	cycle := probe.NewCycle(executor, a.database, a.cfg)

	// Run the cycle
	if err := cycle.Run(ctx, proxyClient, nodes, func() bool {
		return a.shuttingDown.Load()
	}); err != nil {
		return fmt.Errorf("cycle execution failed: %w", err)
	}

	return nil
}
