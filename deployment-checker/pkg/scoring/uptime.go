package scoring

import (
	"context"
	"fmt"
	"time"

	"github.com/threefoldtech/deployment-checker/pkg/db"
)

// UptimeScorer calculates scores based on node uptime
// This is a placeholder for future implementation
// Requires additional data collection (uptime tracking)
type UptimeScorer struct {
	database    *db.DB
	weight      float64
	minAttempts int64
	// Future: can add uptime thresholds, minimum uptime percentage, etc.
}

// NewUptimeScorer creates a new uptime scorer
func NewUptimeScorer(database *db.DB, weight float64, minAttempts int64) *UptimeScorer {
	return &UptimeScorer{
		database:    database,
		weight:      weight,
		minAttempts: minAttempts,
	}
}

// Name returns the name of this scorer
func (u *UptimeScorer) Name() string {
	return "uptime"
}

// Weight returns the weight of this scorer
func (u *UptimeScorer) Weight() float64 {
	return u.weight
}

// Calculate computes the uptime-based score for a node
// Currently returns a placeholder score
// Future implementation: calculate uptime percentage from deployment attempts
// or from dedicated uptime tracking data
func (u *UptimeScorer) Calculate(ctx context.Context, nodeID int64, window time.Duration) (*ScoreResult, error) {
	scoreData, err := u.database.GetNodeScoreData(ctx, nodeID, window)
	if err != nil {
		return nil, fmt.Errorf("failed to get node score data: %w", err)
	}

	// Check minimum attempts threshold
	if scoreData.TotalAttempts < u.minAttempts {
		return &ScoreResult{
			Score:  0.0,
			Metric: "uptime",
			Value: map[string]interface{}{
				"total_attempts": scoreData.TotalAttempts,
				"min_attempts":   u.minAttempts,
			},
		}, nil
	}

	// Placeholder: return neutral score until implemented
	// Future: calculate uptime from deployment success rate or dedicated uptime data
	// For now, we can use success rate as a proxy for uptime
	uptimePercentage := float64(scoreData.SuccessCount) / float64(scoreData.TotalAttempts)

	return &ScoreResult{
		Score:  uptimePercentage,
		Metric: "uptime",
		Value: map[string]interface{}{
			"uptime_percentage": uptimePercentage,
			"total_attempts":    scoreData.TotalAttempts,
			"note":              "placeholder implementation - uses success rate as proxy",
		},
	}, nil
}
