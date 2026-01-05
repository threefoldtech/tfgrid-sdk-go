package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/threefoldtech/deployment-checker/pkg/models"
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

func (d *DB) Close() {
	d.pool.Close()
}

func (d *DB) Ping(ctx context.Context) error {
	return d.pool.Ping(ctx)
}

func (d *DB) initSchema(ctx context.Context) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS deployment_attempts (
			time BIGINT NOT NULL,
			node_id BIGINT NOT NULL,
			farm_id BIGINT NOT NULL,
			status VARCHAR(20) NOT NULL,
			total_duration_ms INTEGER,
			error_code VARCHAR(100)
		)`,
		`SELECT create_hypertable('deployment_attempts', 'time', if_not_exists => TRUE)`,
		`CREATE INDEX IF NOT EXISTS idx_node_time ON deployment_attempts(node_id, time DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_farm_time ON deployment_attempts(farm_id, time DESC)`,
	}

	for _, query := range queries {
		if _, err := d.pool.Exec(ctx, query); err != nil {
			return fmt.Errorf("failed to execute query: %w", err)
		}
	}

	return nil
}

func (d *DB) RecordAttempt(ctx context.Context, attempt models.Attempt) error {
	query := `INSERT INTO deployment_attempts 
		(time, node_id, farm_id, status, total_duration_ms, error_code)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := d.pool.Exec(ctx, query,
		attempt.Time,
		attempt.NodeID,
		attempt.FarmID,
		attempt.Status,
		attempt.TotalDurationMs,
		attempt.ErrorCode,
	)

	return err
}

func (d *DB) GetNodeScoreData(ctx context.Context, nodeID int64, window time.Duration) (*models.NodeScoreData, error) {
	windowStart := time.Now().Add(-window).Unix()

	query := `
		SELECT 
			node_id,
			farm_id,
			COUNT(*) as total_attempts,
			COUNT(*) FILTER (WHERE status = 'success') as success_count,
			AVG(total_duration_ms) FILTER (WHERE total_duration_ms IS NOT NULL) as avg_duration_ms
		FROM deployment_attempts
		WHERE node_id = $1 AND time >= $2
		GROUP BY node_id, farm_id
	`

	var data models.NodeScoreData
	err := d.pool.QueryRow(ctx, query, nodeID, windowStart).Scan(
		&data.NodeID,
		&data.FarmID,
		&data.TotalAttempts,
		&data.SuccessCount,
		&data.AvgDurationMs,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get node score data: %w", err)
	}

	return &data, nil
}

func (d *DB) GetTopNodeScores(ctx context.Context, window time.Duration, limit int, minAttempts int) ([]models.NodeScoreData, error) {
	windowStart := time.Now().Add(-window).Unix()

	query := `
		SELECT 
			node_id,
			farm_id,
			COUNT(*) as total_attempts,
			COUNT(*) FILTER (WHERE status = 'success') as success_count,
			AVG(total_duration_ms) FILTER (WHERE total_duration_ms IS NOT NULL) as avg_duration_ms
		FROM deployment_attempts
		WHERE time >= $1
		GROUP BY node_id, farm_id
		HAVING COUNT(*) >= $3
		ORDER BY 
			(COUNT(*) FILTER (WHERE status = 'success')::float / COUNT(*)) DESC
		LIMIT $2
	`

	rows, err := d.pool.Query(ctx, query, windowStart, limit, minAttempts)
	if err != nil {
		return nil, fmt.Errorf("failed to get top node scores: %w", err)
	}
	defer rows.Close()

	var results []models.NodeScoreData
	for rows.Next() {
		var data models.NodeScoreData
		err := rows.Scan(
			&data.NodeID,
			&data.FarmID,
			&data.TotalAttempts,
			&data.SuccessCount,
			&data.AvgDurationMs,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan node score data: %w", err)
		}
		results = append(results, data)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return results, nil
}
