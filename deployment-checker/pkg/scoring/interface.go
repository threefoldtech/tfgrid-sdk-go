package scoring

import (
	"context"
	"time"
)

// Scorer is the interface for scorers
// It defines the methods that a scorer must implement
// to be used in the scoring service
type Scorer interface {
	Name() string
	Calculate(ctx context.Context, nodeID int64, window time.Duration) (*ScoreResult, error)
	Weight() float64
}

type ScoreResult struct {
	Score  float64
	Metric string
	Value  interface{}
}
