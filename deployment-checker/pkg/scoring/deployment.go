package scoring

import (
	"context"
	"fmt"
	"time"

	"github.com/threefoldtech/deployment-checker/pkg/db"
)

const (
	minAttemptsThreshold = 1
)

// DeploymentScorer calculates scores based on deployment success rate
type DeploymentScorer struct {
	database    *db.DB
	weight      float64
	minAttempts int64
}

// NewDeploymentScorer creates a new deployment scorer
func NewDeploymentScorer(database *db.DB, weight float64, minAttempts int64) *DeploymentScorer {
	return &DeploymentScorer{
		database:    database,
		weight:      weight,
		minAttempts: minAttempts,
	}
}

// Name returns the name of this scorer
func (d *DeploymentScorer) Name() string {
	return "deployment"
}

// Weight returns the weight of this scorer
func (d *DeploymentScorer) Weight() float64 {
	return d.weight
}

// Calculate computes the deployment success rate score for a node
func (d *DeploymentScorer) Calculate(ctx context.Context, nodeID int64, window time.Duration) (*ScoreResult, error) {
	scoreData, err := d.database.GetNodeScoreData(ctx, nodeID, window)
	if err != nil {
		return nil, fmt.Errorf("failed to get node score data: %w", err)
	}

	// Check minimum attempts threshold
	if scoreData.TotalAttempts < d.minAttempts {
		return &ScoreResult{
			Score:  0.0,
			Metric: "deployment_success",
			Value: map[string]interface{}{
				"total_attempts": scoreData.TotalAttempts,
				"min_attempts":   d.minAttempts,
			},
		}, nil
	}

	// Calculate success rate
	successRate := float64(scoreData.SuccessCount) / float64(scoreData.TotalAttempts)

	return &ScoreResult{
		Score:  successRate,
		Metric: "deployment_success",
		Value: map[string]interface{}{
			"success_rate":   successRate,
			"total_attempts": scoreData.TotalAttempts,
			"success_count":  scoreData.SuccessCount,
		},
	}, nil
}
