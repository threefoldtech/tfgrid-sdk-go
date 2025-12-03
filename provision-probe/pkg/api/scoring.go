package api

import (
	"github.com/threefoldtech/provision-probe/pkg/db"
)

const (
	minAttemptsThreshold = 1
)

// NodeScore represents a node's performance score
// @Description Node performance score and metrics
type NodeScore struct {
	NodeID        int64   `json:"node_id" example:"123"`                    // Node ID
	FarmID        int64   `json:"farm_id" example:"1"`                      // Farm ID
	SuccessRate   float64 `json:"success_rate" example:"0.95"`              // Success rate (0.0 to 1.0)
	TotalAttempts int64   `json:"total_attempts" example:"100"`             // Total number of deployment attempts
	AvgDurationMs float64 `json:"avg_duration_ms,omitempty" example:"2500"` // Average deployment duration in milliseconds
	Score         float64 `json:"score" example:"0.95"`                     // Calculated score (currently same as success rate)
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
