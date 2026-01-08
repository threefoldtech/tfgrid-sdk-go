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

type ScoringService struct {
	database       *db.DB
	scorerRegistry *scoring.Registry
	DefaultWindow  time.Duration
}

func NewScoringService(database *db.DB, scorerRegistry *scoring.Registry, defaultWindow time.Duration) *ScoringService {
	return &ScoringService{
		database:       database,
		scorerRegistry: scorerRegistry,
		DefaultWindow:  defaultWindow,
	}
}

func (s *ScoringService) GetTopScores(ctx context.Context, window time.Duration, limit int, minAttempts int) ([]models.NodeScore, error) {
	scoreData, err := s.database.GetTopNodeScores(ctx, window, limit, minAttempts)
	if err != nil {
		return nil, fmt.Errorf("failed to get top node scores: %w", err)
	}

	scores := make([]models.NodeScore, 0, len(scoreData))
	for _, data := range scoreData {
		compositeResult, err := s.scorerRegistry.CalculateComposite(ctx, data.NodeID, window)
		if err != nil {
			log.Warn().
				Err(err).
				Int64("node_id", data.NodeID).
				Msg("Failed to calculate composite score, skipping node")
			continue
		}

		nodeScore := s.convertCompositeToNodeScore(compositeResult, &data)
		scores = append(scores, nodeScore)
	}

	return scores, nil
}

func (s *ScoringService) GetNodeScore(ctx context.Context, nodeID int64, window time.Duration, minAttempts int) (*models.NodeScore, error) {
	scoreData, err := s.database.GetNodeScoreData(ctx, nodeID, window)
	if err != nil {
		return nil, fmt.Errorf("failed to get node score data: %w", err)
	}

	if scoreData.TotalAttempts < int64(minAttempts) {
		return nil, fmt.Errorf("node does not meet minimum attempts requirement: %d attempts (required: %d)", scoreData.TotalAttempts, minAttempts)
	}

	compositeResult, err := s.scorerRegistry.CalculateComposite(ctx, nodeID, window)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate composite score: %w", err)
	}

	nodeScore := s.convertCompositeToNodeScore(compositeResult, scoreData)
	return &nodeScore, nil
}

func (s *ScoringService) convertCompositeToNodeScore(composite *scoring.CompositeResult, data *models.NodeScoreData) models.NodeScore {
	var successRate float64
	for _, result := range composite.ScorerResults {
		if result.Metric == "deployment_success" {
			successRate = result.Score
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
		TotalAttempts: data.TotalAttempts,
		AvgDurationMs: avgDurationMs,
		Score:         composite.CompositeScore,
	}
}
