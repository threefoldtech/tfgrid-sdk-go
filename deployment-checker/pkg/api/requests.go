package api

// GetTopScoresRequest represents the request parameters for GET /api/v1/scores
type GetTopScoresRequest struct {
	Window      string `form:"window" binding:"omitempty"`             // Time window for scoring (e.g., '90d', '30d', '7d')
	Limit       int    `form:"limit" binding:"omitempty,min=1"`        // Maximum number of results to return (default: 10)
	MinAttempts int    `form:"min_attempts" binding:"omitempty,min=1"` // Minimum number of attempts required (default: 1)
}

// GetNodeScoreRequest represents the request parameters for GET /api/v1/scores/node/:node_id
type GetNodeScoreRequest struct {
	NodeID      int64  `uri:"node_id" binding:"required,min=1"`        // Node ID (path parameter)
	Window      string `form:"window" binding:"omitempty"`             // Time window for scoring (e.g., '90d', '30d', '7d')
	MinAttempts int    `form:"min_attempts" binding:"omitempty,min=1"` // Minimum number of attempts required (default: 1)
}
