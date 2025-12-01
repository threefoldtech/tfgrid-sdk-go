package api

import (
	"github.com/threefoldtech/provision-probe/pkg/db"
)

const (
	minAttemptsThreshold = 1
)

type NodeScore struct {
	NodeID        int64   `json:"node_id"`
	FarmID        int64   `json:"farm_id"`
	SuccessRate   float64 `json:"success_rate"`
	TotalAttempts int64   `json:"total_attempts"`
	AvgDurationMs float64 `json:"avg_duration_ms,omitempty"`
	Score         float64 `json:"score"`
}

// TODO: add score to the duration ratio
func CalculateScore(data *db.NodeScoreData) *NodeScore {
	if data.TotalAttempts < minAttemptsThreshold {
		result := &NodeScore{
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

	result := &NodeScore{
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
