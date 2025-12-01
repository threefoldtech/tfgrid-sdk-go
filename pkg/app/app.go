package app

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/provision-probe/pkg/config"
	"github.com/threefoldtech/provision-probe/pkg/db"
	"github.com/threefoldtech/provision-probe/pkg/grid"
)

type App struct {
	cfg        *config.Config
	database   *db.DB
	gridClient *grid.Client
}

func New(cfg *config.Config) (*App, error) {
	database, err := db.New(context.Background(), cfg.TimescaleDB.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	gridClient, err := grid.NewClient(cfg.Grid.Network, cfg.Grid.Mnemonic, cfg.LogLevel)
	if err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to create grid client: %w", err)
	}

	return &App{
		cfg:        cfg,
		database:   database,
		gridClient: gridClient,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	ticker := time.NewTicker(a.cfg.Interval())
	defer ticker.Stop()

	log.Info().
		Str("interval", a.cfg.Interval().String()).
		Str("network", a.cfg.Grid.Network).
		Msg("Starting provision probe service")

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := a.runCycle(ctx); err != nil {
				log.Error().Err(err).Msg("Cycle failed")
			}
		}
	}
}

func (a *App) Close() {
	a.database.Close()
}

func (a *App) runCycle(ctx context.Context) error {
	log.Info().Msg("Starting deployment cycle")

	filters := a.buildFilters()
	proxyClient := a.gridClient.GetProxyClient()
	nodes, err := grid.GetNodes(ctx, proxyClient, filters)
	if err != nil {
		return fmt.Errorf("failed to get nodes: %w", err)
	}

	if len(nodes) == 0 {
		log.Warn().Msg("No eligible nodes found")
		return nil
	}

	node := nodes[0]
	cpu, memoryMB, diskMB := a.cfg.GetWorkload()

	log.Info().
		Int("node_id", node.NodeID).
		Int("farm_id", node.FarmID).
		Str("workload", a.cfg.Workload).
		Msg("Deploying VM to node")

	timeoutCtx, cancel := context.WithTimeout(ctx, a.cfg.Timeout())
	defer cancel()

	result, err := a.gridClient.DeployVM(timeoutCtx, uint32(int(node.NodeID)), cpu, memoryMB, diskMB)

	attempt := db.Attempt{
		Time:         time.Now().Unix(),
		NodeID:       int64(int(node.NodeID)),
		FarmID:       int64(int(node.FarmID)),
		WorkloadType: a.cfg.Workload,
		Status:       "failed",
	}

	if err != nil {
		errorCode := "unknown_error"
		if result != nil {
			errorCode = result.ErrorCode
			attempt.DeployDurationMs = &result.DeployDurationMs
			attempt.TotalDurationMs = &result.TotalDurationMs
		}
		attempt.ErrorCode = &errorCode
		log.Error().Err(err).Str("error_code", errorCode).Msg("Deployment failed")
	} else if result != nil {
		attempt.Status = "success"
		attempt.DeployDurationMs = &result.DeployDurationMs
		attempt.StartDurationMs = &result.StartDurationMs
		attempt.TotalDurationMs = &result.TotalDurationMs
		log.Info().Msg("Deployment succeeded")
	}

	if err := a.database.RecordAttempt(ctx, attempt); err != nil {
		return fmt.Errorf("failed to record attempt: %w", err)
	}

	return nil
}

func (a *App) buildFilters() grid.NodeFilters {
	var status *string
	if len(a.cfg.Nodes.Status) > 0 {
		s := a.cfg.Nodes.Status[0]
		status = &s
	}

	var farmIDs []uint64
	for _, id := range a.cfg.Nodes.Farms {
		farmIDs = append(farmIDs, uint64(id))
	}

	var nodeIDs []uint64
	for _, id := range a.cfg.Nodes.Nodes {
		nodeIDs = append(nodeIDs, uint64(id))
	}

	return grid.NodeFilters{
		Status:  status,
		FarmIDs: farmIDs,
		NodeIDs: nodeIDs,
		Exclude: a.cfg.Nodes.Exclude,
	}
}
