package scoring

import (
	"context"
	"fmt"
	"time"

	"github.com/threefoldtech/deployment-checker/pkg/db"
)

// DurationScorer calculates scores based on deployment duration
// This is a placeholder for future implementation
type DurationScorer struct {
	database    *db.DB
	weight      float64
	minAttempts int64
	// Future: can add thresholds like maxDurationMs, idealDurationMs
}

// NewDurationScorer creates a new duration scorer
func NewDurationScorer(database *db.DB, weight float64, minAttempts int64) *DurationScorer {
	return &DurationScorer{
		database:    database,
		weight:      weight,
		minAttempts: minAttempts,
	}
}

// Name returns the name of this scorer
func (d *DurationScorer) Name() string {
	return "duration"
}

// Weight returns the weight of this scorer
func (d *DurationScorer) Weight() float64 {
	return d.weight
}

// Calculate computes the duration-based score for a node
// Currently returns a placeholder score
// Future implementation: normalize duration (shorter = better, score 0-1)
func (d *DurationScorer) Calculate(ctx context.Context, nodeID int64, window time.Duration) (*ScoreResult, error) {
	scoreData, err := d.database.GetNodeScoreData(ctx, nodeID, window)
	if err != nil {
		return nil, fmt.Errorf("failed to get node score data: %w", err)
	}

	// Check minimum attempts threshold
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

	// Placeholder: return neutral score until implemented
	// Future: normalize average duration to 0-1 scale
	// Shorter durations should score higher
	var avgDuration float64
	if scoreData.AvgDurationMs != nil {
		avgDuration = *scoreData.AvgDurationMs
	}

	// Placeholder score (always returns 0.5 until properly implemented)
	score := 0.5

	return &ScoreResult{
		Score:  score,
		Metric: "duration",
		Value: map[string]interface{}{
			"avg_duration_ms": avgDuration,
			"total_attempts":  scoreData.TotalAttempts,
		},
	}, nil
}
