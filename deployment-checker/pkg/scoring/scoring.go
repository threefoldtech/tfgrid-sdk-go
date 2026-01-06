package scoring

import (
	"github.com/threefoldtech/deployment-checker/pkg/models"
)

// CalculateScore calculates a node's performance score based on deployment attempt data
// Currently, the score is the same as the success rate, but this can be enhanced
// to include duration-based metrics (see TODO in original code)
func CalculateScore(data *models.NodeScoreData) *models.NodeScore {
	if data.TotalAttempts < minAttemptsThreshold {
		result := &models.NodeScore{
			NodeID:        data.NodeID,
			FarmID:        data.FarmID,
			TotalAttempts: data.TotalAttempts,
			Score:         0.0,
		}
		if data.AvgDurationMs != nil {
			result.AvgDurationMs = *data.AvgDurationMs
		}
		return result
	}

	successRate := float64(data.SuccessCount) / float64(data.TotalAttempts)

	result := &models.NodeScore{
		NodeID:        data.NodeID,
		FarmID:        data.FarmID,
		SuccessRate:   successRate,
		TotalAttempts: data.TotalAttempts,
		Score:         successRate,
	}

	if data.AvgDurationMs != nil {
		result.AvgDurationMs = *data.AvgDurationMs
	}

	return result
}
