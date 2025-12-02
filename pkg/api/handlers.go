// Package api provides HTTP handlers for the provision-probe API
// @title Provision Probe API
// @version 1.0
// @description API for querying node provision scores and health status
// @termsOfService http://swagger.io/terms/
// @contact.name API Support
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @host localhost:8080
// @BasePath /api/v1
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

// Response types

// HealthCheck represents a single health check result
type HealthCheck struct {
	Status  string `json:"status" example:"healthy"` // healthy or unhealthy
	Message string `json:"message,omitempty" example:"connection failed"`
}

// HealthResponse represents the health endpoint response
// @Description Health check response
type HealthResponse struct {
	Status string                 `json:"status" example:"healthy"` // Overall health status: healthy or unhealthy
	Checks map[string]HealthCheck `json:"checks"`                   // Individual health checks
}

// TopScoresResponse represents the response for GET /api/v1/scores
// @Description Top node scores response
type TopScoresResponse struct {
	Window      string      `json:"window" example:"90d"`     // Time window used for scoring
	Limit       int         `json:"limit" example:"10"`       // Maximum number of results returned
	MinAttempts int         `json:"min_attempts" example:"1"` // Minimum attempts required
	Scores      []NodeScore `json:"scores"`                   // List of node scores
}

// NodeScoreResponse represents the response for GET /api/v1/scores/node/:node_id
// @Description Single node score response
type NodeScoreResponse struct {
	Window      string    `json:"window" example:"90d"`     // Time window used for scoring
	MinAttempts int       `json:"min_attempts" example:"1"` // Minimum attempts required
	Score       NodeScore `json:"score"`                    // Node score details
}

// ErrorResponse represents an error response
// @Description Error response
type ErrorResponse struct {
	Error string `json:"error" example:"invalid node_id parameter"`
}

func NewHandlers(database *db.DB, cfg *config.Config) *Handlers {
	return &Handlers{
		database:      database,
		cfg:           cfg,
		defaultWindow: cfg.ScoreWindow(),
	}
}

// GetTopScores handles GET /api/v1/scores
// @Summary Get top node scores
// @Description Returns the top performing nodes based on success rate within a time window
// @Tags scores
// @Accept json
// @Produce json
// @Param window query string false "Time window for scoring (e.g., '90d', '30d', '7d')" default to the config value
// @Param limit query int false "Maximum number of results to return" default(10) minimum(1)
// @Param min_attempts query int false "Minimum number of attempts required" default(1) minimum(1)
// @Success 200 {object} TopScoresResponse "Successfully retrieved top scores"
// @Failure 400 {object} ErrorResponse "Invalid request parameters"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/scores [get]
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

	respondJSON(w, http.StatusOK, TopScoresResponse{
		Window:      window.String(),
		Limit:       limit,
		MinAttempts: minAttempts,
		Scores:      scores,
	})
}

// GetNodeScore handles GET /api/v1/scores/node/:node_id
// @Summary Get score for a specific node
// @Description Returns the score and performance metrics for a specific node within a time window
// @Tags scores
// @Accept json
// @Produce json
// @Param node_id path int true "Node ID" example(123)
// @Param window query string false "Time window for scoring (e.g., '90d', '30d', '7d')" default("90d")
// @Param min_attempts query int false "Minimum number of attempts required" default(1) minimum(1)
// @Success 200 {object} NodeScoreResponse "Successfully retrieved node score"
// @Failure 400 {object} ErrorResponse "Invalid request parameters"
// @Failure 404 {object} ErrorResponse "Node not found or insufficient attempts"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/scores/node/{node_id} [get]
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

	respondJSON(w, http.StatusOK, NodeScoreResponse{
		Window:      window.String(),
		MinAttempts: minAttempts,
		Score:       *score,
	})
}

// GetHealth handles GET /api/v1/health
// @Summary Health check endpoint
// @Description Returns the health status of the service and its dependencies
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} HealthResponse "Service is healthy"
// @Success 503 {object} HealthResponse "Service is unhealthy"
// @Router /api/v1/health [get]
func (h *Handlers) GetHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	health := HealthResponse{
		Status: "healthy",
		Checks: make(map[string]HealthCheck),
	}

	if err := h.database.Ping(ctx); err != nil {
		health.Checks["database"] = HealthCheck{
			Status:  "unhealthy",
			Message: err.Error(),
		}
		health.Status = "unhealthy"
	} else {
		health.Checks["database"] = HealthCheck{
			Status: "healthy",
		}
	}

	statusCode := http.StatusOK
	if health.Status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}

	respondJSON(w, statusCode, health)
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
	respondJSON(w, status, ErrorResponse{
		Error: message,
	})
}
