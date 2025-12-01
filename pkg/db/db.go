package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, url string) (*DB, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db := &DB{pool: pool}
	if err := db.initSchema(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return db, nil
}

func (d *DB) initSchema(ctx context.Context) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS provision_attempts (
			time TIMESTAMPTZ NOT NULL,
			node_id BIGINT NOT NULL,
			farm_id BIGINT NOT NULL,
			workload_type VARCHAR(20) NOT NULL,
			status VARCHAR(20) NOT NULL,
			deploy_duration_ms INTEGER,
			start_duration_ms INTEGER,
			total_duration_ms INTEGER,
			error_code VARCHAR(100)
		)`,
		`SELECT create_hypertable('provision_attempts', 'time', if_not_exists => TRUE)`,
		`CREATE INDEX IF NOT EXISTS idx_node_time ON provision_attempts(node_id, time DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_farm_time ON provision_attempts(farm_id, time DESC)`,
	}

	for _, query := range queries {
		if _, err := d.pool.Exec(ctx, query); err != nil {
			return fmt.Errorf("failed to execute query: %w", err)
		}
	}

	return nil
}

func (d *DB) RecordAttempt(ctx context.Context, attempt Attempt) error {
	query := `INSERT INTO provision_attempts 
		(time, node_id, farm_id, workload_type, status, deploy_duration_ms, start_duration_ms, total_duration_ms, error_code)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := d.pool.Exec(ctx, query,
		attempt.Time,
		attempt.NodeID,
		attempt.FarmID,
		attempt.WorkloadType,
		attempt.Status,
		attempt.DeployDurationMs,
		attempt.StartDurationMs,
		attempt.TotalDurationMs,
		attempt.ErrorCode,
	)

	return err
}

func (d *DB) Close() {
	d.pool.Close()
}

type Attempt struct {
	Time              int64
	NodeID            int64
	FarmID            int64
	WorkloadType      string
	Status            string
	DeployDurationMs  *int
	StartDurationMs   *int
	TotalDurationMs   *int
	ErrorCode         *string
}

