package scoring

import (
	"context"
	"time"
)

// Scorer defines the interface for calculating node scores
type Scorer interface {
	// Name returns the name of the scorer
	Name() string

	// Calculate computes a score for a node within a given time window
	// Returns a normalized score (0.0 to 1.0) and associated metrics
	Calculate(ctx context.Context, nodeID int64, window time.Duration) (*ScoreResult, error)

	// Weight returns the weight of this scorer for composite scoring
	// The weight determines how much this scorer contributes to the final composite score
	Weight() float64
}

// ScoreResult represents the result of a scoring calculation
type ScoreResult struct {
	// Score is the normalized score (0.0 to 1.0)
	Score float64

	// Metric is the name of the metric being scored (e.g., "deployment_success", "duration", "uptime")
	Metric string

	// Value is the raw value of the metric (can be any type)
	// For example: success rate (float64), average duration (int), uptime percentage (float64)
	Value interface{}
}
