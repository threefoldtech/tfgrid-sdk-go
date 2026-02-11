package api

type GetTopScoresRequest struct {
	Window      string `form:"window" binding:"omitempty"`
	Limit       int    `form:"limit" binding:"omitempty,min=1"`
	MinAttempts int    `form:"min_attempts" binding:"omitempty,min=1"`
}

type GetNodeScoreRequest struct {
	NodeID      int64  `uri:"node_id" binding:"required,min=1"`
	Window      string `form:"window" binding:"omitempty"`
	MinAttempts int    `form:"min_attempts" binding:"omitempty,min=1"`
}
