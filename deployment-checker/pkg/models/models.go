package models

// Attempt represents a single deployment attempt record
type Attempt struct {
	Time            int64
	NodeID          int64
	FarmID          int64
	Status          string
	TotalDurationMs *int
	ErrorCode       *string
}

// NodeScoreData represents aggregated score data for a node
type NodeScoreData struct {
	NodeID        int64
	FarmID        int64
	TotalAttempts int64
	SuccessCount  int64
	AvgDurationMs *float64
}

// NodeScore represents a node's performance score and metrics
type NodeScore struct {
	NodeID        int64   `json:"node_id" example:"123"`                    // Node ID
	FarmID        int64   `json:"farm_id" example:"1"`                      // Farm ID
	SuccessRate   float64 `json:"success_rate" example:"0.95"`              // Success rate (0.0 to 1.0)
	TotalAttempts int64   `json:"total_attempts" example:"100"`             // Total number of deployment attempts
	AvgDurationMs float64 `json:"avg_duration_ms,omitempty" example:"2500"` // Average deployment duration in milliseconds
	Score         float64 `json:"score" example:"0.95"`                     // Calculated score (currently same as success rate)
}

