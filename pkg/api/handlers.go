package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/provision-probe/pkg/config"
	"github.com/threefoldtech/provision-probe/pkg/db"
)

type Handlers struct {
	database      *db.DB
	cfg           *config.Config
	defaultWindow time.Duration
}

func NewHandlers(database *db.DB, cfg *config.Config) *Handlers {
	return &Handlers{
		database:      database,
		cfg:           cfg,
		defaultWindow: cfg.ScoreWindow(),
	}
}

// GetTopScores handles GET /api/v1/scores
// Query params: window (duration, defaults to config value), limit (int, default 10), min_attempts (int, default 1)
func (h *Handlers) GetTopScores(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	windowStr := r.URL.Query().Get("window")
	window := h.defaultWindow
	if windowStr != "" {
		parsedWindow, err := config.ParseDuration(windowStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid window parameter: "+err.Error())
			return
		}
		window = parsedWindow
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err != nil || parsedLimit <= 0 {
			respondError(w, http.StatusBadRequest, "invalid limit parameter: must be a positive integer")
			return
		}
		limit = parsedLimit
	}

	minAttemptsStr := r.URL.Query().Get("min_attempts")
	minAttempts := 1
	if minAttemptsStr != "" {
		parsedMinAttempts, err := strconv.Atoi(minAttemptsStr)
		if err != nil || parsedMinAttempts < 1 {
			respondError(w, http.StatusBadRequest, "invalid min_attempts parameter: must be a positive integer")
			return
		}
		minAttempts = parsedMinAttempts
	}

	scoreData, err := h.database.GetTopNodeScores(ctx, window, limit, minAttempts)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get top node scores")
		respondError(w, http.StatusInternalServerError, "failed to retrieve scores")
		return
	}

	scores := make([]NodeScore, 0, len(scoreData))
	for _, data := range scoreData {
		score := CalculateScore(&data)
		scores = append(scores, *score)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"window":       window.String(),
		"limit":        limit,
		"min_attempts": minAttempts,
		"scores":       scores,
	})
}

// GetNodeScore handles GET /api/v1/scores/node/:node_id
// Query params: window (duration string, defaults to config value), min_attempts (int, default 1)
func (h *Handlers) GetNodeScore(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	nodeIDStr := chi.URLParam(r, "node_id")
	nodeID, err := strconv.ParseInt(nodeIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid node_id parameter")
		return
	}

	windowStr := r.URL.Query().Get("window")
	window := h.defaultWindow
	if windowStr != "" {
		parsedWindow, err := config.ParseDuration(windowStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid window parameter: "+err.Error())
			return
		}
		window = parsedWindow
	}

	minAttemptsStr := r.URL.Query().Get("min_attempts")
	minAttempts := 1
	if minAttemptsStr != "" {
		parsedMinAttempts, err := strconv.Atoi(minAttemptsStr)
		if err != nil || parsedMinAttempts < 1 {
			respondError(w, http.StatusBadRequest, "invalid min_attempts parameter: must be a positive integer")
			return
		}
		minAttempts = parsedMinAttempts
	}

	scoreData, err := h.database.GetNodeScoreData(ctx, nodeID, window)
	if err != nil {
		if err.Error() == "failed to get node score data: no rows in result set" ||
			err.Error() == "no rows in result set" {
			respondError(w, http.StatusNotFound, "node not found or no data available")
			return
		}
		log.Error().Err(err).Int64("node_id", nodeID).Msg("Failed to get node score")
		respondError(w, http.StatusInternalServerError, "failed to retrieve node score")
		return
	}

	if scoreData.TotalAttempts < int64(minAttempts) {
		respondError(w, http.StatusNotFound, fmt.Sprintf("node does not meet minimum attempts requirement: %d attempts (required: %d)", scoreData.TotalAttempts, minAttempts))
		return
	}

	score := CalculateScore(scoreData)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"window":       window.String(),
		"min_attempts": minAttempts,
		"score":        score,
	})
}

// GetHealth handles GET /api/v1/health
func (h *Handlers) GetHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := h.database.Ping(ctx); err != nil {
		respondJSON(w, http.StatusServiceUnavailable, map[string]interface{}{
			"status":  "unhealthy",
			"message": "database connection failed",
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status": "healthy",
	})
}

// Helper functions

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Error().Err(err).Msg("Failed to encode JSON response")
	}
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]interface{}{
		"error": message,
	})
}
