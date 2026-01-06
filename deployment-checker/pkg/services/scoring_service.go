package services

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/deployment-checker/pkg/db"
	"github.com/threefoldtech/deployment-checker/pkg/models"
	"github.com/threefoldtech/deployment-checker/pkg/scoring"
)

// ScoringService provides scoring business logic
type ScoringService struct {
	database       *db.DB
	scorerRegistry *scoring.Registry
	DefaultWindow  time.Duration
}

// NewScoringService creates a new scoring service
func NewScoringService(database *db.DB, scorerRegistry *scoring.Registry, defaultWindow time.Duration) *ScoringService {
	return &ScoringService{
		database:       database,
		scorerRegistry: scorerRegistry,
		DefaultWindow:  defaultWindow,
	}
}

// GetTopScores retrieves top node scores
func (s *ScoringService) GetTopScores(ctx context.Context, window time.Duration, limit int, minAttempts int) ([]models.NodeScore, error) {
	scoreData, err := s.database.GetTopNodeScores(ctx, window, limit, minAttempts)
	if err != nil {
		return nil, fmt.Errorf("failed to get top node scores: %w", err)
	}

	scores := make([]models.NodeScore, 0, len(scoreData))
	for _, data := range scoreData {
		// Use composite scoring for consistency
		compositeResult, err := s.scorerRegistry.CalculateComposite(ctx, data.NodeID, window)
		if err != nil {
			log.Warn().
				Err(err).
				Int64("node_id", data.NodeID).
				Msg("Failed to calculate composite score, using fallback")
			// Fallback to simple scoring
			score := scoring.CalculateScore(&data)
			scores = append(scores, *score)
			continue
		}

		// Convert composite result to NodeScore format
		nodeScore := s.convertCompositeToNodeScore(compositeResult, &data)
		scores = append(scores, nodeScore)
	}

	return scores, nil
}

// GetNodeScore retrieves score for a specific node
func (s *ScoringService) GetNodeScore(ctx context.Context, nodeID int64, window time.Duration, minAttempts int) (*models.NodeScore, error) {
	scoreData, err := s.database.GetNodeScoreData(ctx, nodeID, window)
	if err != nil {
		return nil, fmt.Errorf("failed to get node score data: %w", err)
	}

	if scoreData.TotalAttempts < int64(minAttempts) {
		return nil, fmt.Errorf("node does not meet minimum attempts requirement: %d attempts (required: %d)", scoreData.TotalAttempts, minAttempts)
	}

	// Use composite scoring from registry
	compositeResult, err := s.scorerRegistry.CalculateComposite(ctx, nodeID, window)
	if err != nil {
		log.Error().Err(err).Int64("node_id", nodeID).Msg("Failed to calculate composite score")
		// Fallback to old scoring method for backward compatibility
		score := scoring.CalculateScore(scoreData)
		return score, nil
	}

	// Convert composite result to NodeScore format
	nodeScore := s.convertCompositeToNodeScore(compositeResult, scoreData)
	return &nodeScore, nil
}

// convertCompositeToNodeScore converts composite scoring result to NodeScore model
func (s *ScoringService) convertCompositeToNodeScore(composite *scoring.CompositeResult, data *models.NodeScoreData) models.NodeScore {
	var successRate float64
	var totalAttempts int64 = data.TotalAttempts

	// Extract deployment scorer result for success rate
	for _, result := range composite.ScorerResults {
		if result.Metric == "deployment_success" {
			successRate = result.Score
			if valueMap, ok := result.Value.(map[string]interface{}); ok {
				if ta, ok := valueMap["total_attempts"].(int64); ok {
					totalAttempts = ta
				}
			}
			break
		}
	}

	var avgDurationMs float64
	if data.AvgDurationMs != nil {
		avgDurationMs = *data.AvgDurationMs
	}

	return models.NodeScore{
		NodeID:        data.NodeID,
		FarmID:        data.FarmID,
		SuccessRate:   successRate,
		TotalAttempts: totalAttempts,
		AvgDurationMs: avgDurationMs,
		Score:         composite.CompositeScore,
	}
}
