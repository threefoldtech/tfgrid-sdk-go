package scoring

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/rs/zerolog/log"
)

type Registry struct {
	scorers []Scorer
}

func NewRegistry(scorers ...Scorer) *Registry {
	return &Registry{
		scorers: scorers,
	}
}

func (r *Registry) Register(scorer Scorer) {
	r.scorers = append(r.scorers, scorer)
}

func (r *Registry) GetScorer(name string) (Scorer, error) {
	for _, scorer := range r.scorers {
		if scorer.Name() == name {
			return scorer, nil
		}
	}
	return nil, fmt.Errorf("scorer not found: %s", name)
}

func (r *Registry) GetAllScorers() []Scorer {
	return r.scorers
}

func (r *Registry) CalculateComposite(ctx context.Context, nodeID int64, window time.Duration) (*CompositeResult, error) {
	var results []ScoreResult
	var totalWeight float64
	var weightedSum float64

	for _, scorer := range r.scorers {
		if scorer.Weight() <= 0 {
			continue
		}

		result, err := scorer.Calculate(ctx, nodeID, window)
		if err != nil {
			log.Error().
				Err(err).
				Str("scorer", scorer.Name()).
				Int64("node_id", nodeID).
				Msg("Scorer calculation failed")

			if scorer.Weight() > 0.5 {
				return nil, fmt.Errorf("critical scorer %s failed: %w", scorer.Name(), err)
			}

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

	if math.IsNaN(compositeScore) || math.IsInf(compositeScore, 0) {
		return nil, fmt.Errorf("invalid composite score calculated: %f", compositeScore)
	}

	return &CompositeResult{
		CompositeScore: compositeScore,
		ScorerResults:  results,
		TotalWeight:    totalWeight,
	}, nil
}

type CompositeResult struct {
	CompositeScore float64
	ScorerResults  []ScoreResult
	TotalWeight    float64
}
