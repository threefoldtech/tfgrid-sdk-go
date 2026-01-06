package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/threefoldtech/deployment-checker/pkg/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type DB struct {
	db *gorm.DB
}

func New(ctx context.Context, url string) (*DB, error) {
	gormDB, err := gorm.Open(postgres.Open(url), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying sql.DB to set connection pool settings
	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Configure connection pool
	// MaxOpenConns: based on concurrency_limit (10) + API requests (estimate 15) = 25
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	sqlDB.SetConnMaxIdleTime(1 * time.Minute)

	// Test connection
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db := &DB{db: gormDB}
	if err := db.initSchema(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return db, nil
}

func (d *DB) Close() {
	sqlDB, err := d.db.DB()
	if err != nil {
		return
	}
	_ = sqlDB.Close()
}

func (d *DB) Ping(ctx context.Context) error {
	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

func (d *DB) initSchema(ctx context.Context) error {
	// Use GORM AutoMigrate to create the table
	if err := d.db.WithContext(ctx).AutoMigrate(&models.DeploymentAttempt{}); err != nil {
		return fmt.Errorf("failed to auto migrate: %w", err)
	}

	// Create hypertable if it doesn't exist
	// TimescaleDB's create_hypertable function is idempotent with if_not_exists
	result := d.db.WithContext(ctx).Exec(`SELECT create_hypertable('deployment_attempts', 'time', if_not_exists => TRUE)`)
	if result.Error != nil {
		// Check if error is because hypertable already exists
		// Common error messages: "relation ... is already a hypertable" or "hypertable already exists"
		errMsg := strings.ToLower(result.Error.Error())
		if !strings.Contains(errMsg, "already") && !strings.Contains(errMsg, "hypertable") {
			return fmt.Errorf("failed to create hypertable: %w", result.Error)
		}
		// If it's an "already exists" error, we can safely ignore it
	}

	// Ensure indexes exist (GORM should create them from tags, but we'll verify)
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_time ON deployment_attempts(time DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_node_time ON deployment_attempts(node_id, time DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_farm_time ON deployment_attempts(farm_id, time DESC)`,
	}

	for _, indexQuery := range indexes {
		if err := d.db.WithContext(ctx).Exec(indexQuery).Error; err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

func (d *DB) RecordAttempt(ctx context.Context, attempt models.Attempt) error {
	gormAttempt := models.DeploymentAttempt{
		Time:            attempt.Time,
		NodeID:          attempt.NodeID,
		FarmID:          attempt.FarmID,
		Status:          attempt.Status,
		TotalDurationMs: attempt.TotalDurationMs,
		ErrorCode:       attempt.ErrorCode,
	}

	result := d.db.WithContext(ctx).Create(&gormAttempt)
	return result.Error
}

// RecordAttemptsBatch inserts multiple attempts in batches
func (d *DB) RecordAttemptsBatch(ctx context.Context, attempts []models.Attempt, batchSize int) error {
	if len(attempts) == 0 {
		return nil
	}

	// Convert models.Attempt to models.DeploymentAttempt
	gormAttempts := make([]models.DeploymentAttempt, len(attempts))
	for i, attempt := range attempts {
		gormAttempts[i] = models.DeploymentAttempt{
			Time:            attempt.Time,
			NodeID:          attempt.NodeID,
			FarmID:          attempt.FarmID,
			Status:          attempt.Status,
			TotalDurationMs: attempt.TotalDurationMs,
			ErrorCode:       attempt.ErrorCode,
		}
	}

	result := d.db.WithContext(ctx).CreateInBatches(gormAttempts, batchSize)
	return result.Error
}

func (d *DB) GetNodeScoreData(ctx context.Context, nodeID int64, window time.Duration) (*models.NodeScoreData, error) {
	windowStart := time.Now().Add(-window).Unix()

	var data models.NodeScoreData

	result := d.db.WithContext(ctx).
		Model(&models.DeploymentAttempt{}).
		Select(`
			node_id,
			farm_id,
			COUNT(*) as total_attempts,
			COUNT(*) FILTER (WHERE status = 'success') as success_count,
			AVG(total_duration_ms) FILTER (WHERE total_duration_ms IS NOT NULL) as avg_duration_ms
		`).
		Where("node_id = ? AND time >= ?", nodeID, windowStart).
		Group("node_id, farm_id").
		Scan(&data)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to get node score data: %w", result.Error)
	}

	// Check if any rows were returned
	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("failed to get node score data: no rows in result set")
	}

	return &data, nil
}

func (d *DB) GetTopNodeScores(ctx context.Context, window time.Duration, limit int, minAttempts int) ([]models.NodeScoreData, error) {
	windowStart := time.Now().Add(-window).Unix()

	var results []models.NodeScoreData

	err := d.db.WithContext(ctx).
		Model(&models.DeploymentAttempt{}).
		Select(`
			node_id,
			farm_id,
			COUNT(*) as total_attempts,
			COUNT(*) FILTER (WHERE status = 'success') as success_count,
			AVG(total_duration_ms) FILTER (WHERE total_duration_ms IS NOT NULL) as avg_duration_ms
		`).
		Where("time >= ?", windowStart).
		Group("node_id, farm_id").
		Having("COUNT(*) >= ?", minAttempts).
		Order("(COUNT(*) FILTER (WHERE status = 'success')::float / COUNT(*)) DESC").
		Limit(limit).
		Scan(&results).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get top node scores: %w", err)
	}

	return results, nil
}

// SetupRetentionPolicy sets up TimescaleDB retention policy for the deployment_attempts table
// This is the preferred method as it's automatic and efficient
func (d *DB) SetupRetentionPolicy(ctx context.Context, retentionDays int) error {
	// Check if retention policy already exists
	var exists bool
	checkQuery := `
		SELECT EXISTS (
			SELECT 1 
			FROM timescaledb_information.jobs j
			JOIN timescaledb_information.job_stats js ON j.job_id = js.job_id
			WHERE j.proc_name = 'policy_retention'
			AND j.hypertable_name = 'deployment_attempts'
		)
	`
	if err := d.db.WithContext(ctx).Raw(checkQuery).Scan(&exists).Error; err != nil {
		// If the query fails, it might be because TimescaleDB is not available
		// or the timescaledb_information schema doesn't exist
		// We'll try to add the policy anyway
		exists = false
	}

	if exists {
		// Policy already exists, try to update it
		// First, drop the existing policy and recreate it
		dropQuery := `SELECT remove_retention_policy('deployment_attempts', if_exists => TRUE)`
		if err := d.db.WithContext(ctx).Exec(dropQuery).Error; err != nil {
			// Log but don't fail - the policy might not exist or might be in a different format
		}
	}

	// Add retention policy
	// TimescaleDB retention policies use INTERVAL format
	interval := fmt.Sprintf("%d days", retentionDays)
	addQuery := fmt.Sprintf(
		`SELECT add_retention_policy('deployment_attempts', INTERVAL '%s', if_not_exists => TRUE)`,
		interval,
	)

	result := d.db.WithContext(ctx).Exec(addQuery)
	if result.Error != nil {
		// Check if error is because policy already exists or table is not a hypertable
		errMsg := strings.ToLower(result.Error.Error())
		if strings.Contains(errMsg, "already") || strings.Contains(errMsg, "exists") {
			// Policy already exists, which is fine
			return nil
		}
		if strings.Contains(errMsg, "not a hypertable") {
			// Table is not a hypertable, retention policies won't work
			return fmt.Errorf("table is not a hypertable, cannot use TimescaleDB retention policy: %w", result.Error)
		}
		return fmt.Errorf("failed to add retention policy: %w", result.Error)
	}

	return nil
}

// CleanupOldData manually deletes old data from the deployment_attempts table
// This is a fallback method for non-TimescaleDB setups or when retention policies are disabled
func (d *DB) CleanupOldData(ctx context.Context, retentionDays int) error {
	cutoffTime := time.Now().AddDate(0, 0, -retentionDays).Unix()

	result := d.db.WithContext(ctx).
		Where("time < ?", cutoffTime).
		Delete(&models.DeploymentAttempt{})

	if result.Error != nil {
		return fmt.Errorf("failed to cleanup old data: %w", result.Error)
	}

	return nil
}
