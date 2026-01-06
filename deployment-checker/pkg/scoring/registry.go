package scoring

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/rs/zerolog/log"
)

// Registry manages multiple scorers and provides composite scoring
type Registry struct {
	scorers []Scorer
}

// NewRegistry creates a new scorer registry
func NewRegistry(scorers ...Scorer) *Registry {
	return &Registry{
		scorers: scorers,
	}
}

// Register adds a scorer to the registry
func (r *Registry) Register(scorer Scorer) {
	r.scorers = append(r.scorers, scorer)
}

// GetScorer returns a scorer by name
func (r *Registry) GetScorer(name string) (Scorer, error) {
	for _, scorer := range r.scorers {
		if scorer.Name() == name {
			return scorer, nil
		}
	}
	return nil, fmt.Errorf("scorer not found: %s", name)
}

// GetAllScorers returns all registered scorers
func (r *Registry) GetAllScorers() []Scorer {
	return r.scorers
}

// CalculateComposite computes a composite score by combining all enabled scorers
// Each scorer's contribution is weighted by its Weight() value
// Returns the composite score (0.0 to 1.0) and individual scorer results
func (r *Registry) CalculateComposite(ctx context.Context, nodeID int64, window time.Duration) (*CompositeResult, error) {
	var results []ScoreResult
	var totalWeight float64
	var weightedSum float64

	for _, scorer := range r.scorers {
		// Skip scorers with zero weight
		if scorer.Weight() <= 0 {
			continue
		}

		result, err := scorer.Calculate(ctx, nodeID, window)
		if err != nil {
			// Log error with context
			log.Error().
				Err(err).
				Str("scorer", scorer.Name()).
				Int64("node_id", nodeID).
				Msg("Scorer calculation failed")
			
			// Fail fast if critical scorer (weight > 0.5) fails
			if scorer.Weight() > 0.5 {
				return nil, fmt.Errorf("critical scorer %s failed: %w", scorer.Name(), err)
			}
			
			// For non-critical scorers, continue but track failures
			continue
		}

		if result == nil {
			continue
		}

		results = append(results, *result)
		weight := scorer.Weight()
		totalWeight += weight
		weightedSum += result.Score * weight
	}

	if totalWeight == 0 {
		return nil, fmt.Errorf("no enabled scorers with positive weight")
	}

	compositeScore := weightedSum / totalWeight
	
	// Validate result is not NaN or Inf
	if math.IsNaN(compositeScore) || math.IsInf(compositeScore, 0) {
		return nil, fmt.Errorf("invalid composite score calculated: %f", compositeScore)
	}

	return &CompositeResult{
		CompositeScore: compositeScore,
		ScorerResults:  results,
		TotalWeight:    totalWeight,
	}, nil
}

// CompositeResult represents the result of composite scoring
type CompositeResult struct {
	// CompositeScore is the final weighted average score (0.0 to 1.0)
	CompositeScore float64

	// ScorerResults contains individual scorer results
	ScorerResults []ScoreResult

	// TotalWeight is the sum of all scorer weights used in calculation
	TotalWeight float64
}
