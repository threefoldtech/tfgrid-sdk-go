package scoring

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/threefoldtech/deployment-checker/pkg/db"
)

type DurationScorer struct {
	database    *db.DB
	weight      float64
	minAttempts int64
}

func NewDurationScorer(database *db.DB, weight float64, minAttempts int64) *DurationScorer {
	return &DurationScorer{
		database:    database,
		weight:      weight,
		minAttempts: minAttempts,
	}
}

func (d *DurationScorer) Name() string {
	return "duration"
}

func (d *DurationScorer) Weight() float64 {
	return d.weight
}

func (d *DurationScorer) Calculate(ctx context.Context, nodeID int64, window time.Duration) (*ScoreResult, error) {
	scoreData, err := d.database.GetNodeScoreData(ctx, nodeID, window)
	if err != nil {
		return nil, fmt.Errorf("failed to get node score data: %w", err)
	}

	if scoreData.TotalAttempts < d.minAttempts {
		return &ScoreResult{
			Score:  0.0,
			Metric: "duration",
			Value: map[string]interface{}{
				"total_attempts": scoreData.TotalAttempts,
				"min_attempts":   d.minAttempts,
			},
		}, nil
	}

	var avgDuration float64
	if scoreData.AvgDurationMs == nil || *scoreData.AvgDurationMs <= 0 {
		return &ScoreResult{
			Score:  0.5,
			Metric: "duration",
			Value: map[string]interface{}{
				"avg_duration_ms": 0,
				"total_attempts":  scoreData.TotalAttempts,
			},
		}, nil
	}

	avgDuration = *scoreData.AvgDurationMs

	// Normalize duration to 0-1 scale where shorter durations score higher
	// Using exponential decay: score = e^(-avgDuration / threshold)
	// Threshold of 20000ms (20s) gives a smooth curve where:
	// - Very fast deployments (< 5s) score close to 1.0
	// - Medium deployments (~20s) score around 0.37
	// - Slow deployments (> 60s) score close to 0.0
	const thresholdMs = 20000.0 // 20 seconds threshold for exponential decay
	score := math.Exp(-avgDuration / thresholdMs)

	// Clamp to [0, 1] range
	score = math.Max(0.0, math.Min(1.0, score))

	return &ScoreResult{
		Score:  score,
		Metric: "duration",
		Value: map[string]interface{}{
			"avg_duration_ms": avgDuration,
			"total_attempts":  scoreData.TotalAttempts,
		},
	}, nil
}
