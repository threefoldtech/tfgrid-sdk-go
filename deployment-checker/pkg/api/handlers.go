// Package api provides HTTP handlers for the deployment-checker API
// @title Deployment Checker API
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
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/deployment-checker/pkg/config"
	"github.com/threefoldtech/deployment-checker/pkg/db"
	"github.com/threefoldtech/deployment-checker/pkg/models"
	"github.com/threefoldtech/deployment-checker/pkg/services"
)

type Handlers struct {
	scoringService *services.ScoringService
	cfg            *config.Config
	database       *db.DB
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
	Window      string             `json:"window" example:"90d"`     // Time window used for scoring
	Limit       int                `json:"limit" example:"10"`       // Maximum number of results returned
	MinAttempts int                `json:"min_attempts" example:"1"` // Minimum attempts required
	Scores      []models.NodeScore `json:"scores"`                   // List of node scores
}

// NodeScoreResponse represents the response for GET /api/v1/scores/node/:node_id
// @Description Single node score response
type NodeScoreResponse struct {
	Window      string           `json:"window" example:"90d"`     // Time window used for scoring
	MinAttempts int              `json:"min_attempts" example:"1"` // Minimum attempts required
	Score       models.NodeScore `json:"score"`                    // Node score details
}

// ErrorResponse represents an error response
// @Description Error response
type ErrorResponse struct {
	Error string `json:"error" example:"invalid node_id parameter"`
}

func NewHandlers(database *db.DB, cfg *config.Config, scoringService *services.ScoringService) *Handlers {
	return &Handlers{
		scoringService: scoringService,
		cfg:            cfg,
		database:       database,
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
func (h *Handlers) GetTopScores(c *gin.Context) {
	var req GetTopScoresRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: fmt.Sprintf("invalid request parameters: %v", err),
		})
		return
	}

	// Set defaults
	window := h.scoringService.DefaultWindow
	if req.Window != "" {
		parsedWindow, err := config.ParseDuration(req.Window)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error: fmt.Sprintf("invalid window parameter: %v", err),
			})
			return
		}
		window = parsedWindow
	}

	limit := req.Limit
	if limit == 0 {
		limit = 10
	}

	minAttempts := req.MinAttempts
	if minAttempts == 0 {
		minAttempts = 1
	}

	scores, err := h.scoringService.GetTopScores(c.Request.Context(), window, limit, minAttempts)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get top node scores")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "failed to retrieve scores",
		})
		return
	}

	c.JSON(http.StatusOK, TopScoresResponse{
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
func (h *Handlers) GetNodeScore(c *gin.Context) {
	var req GetNodeScoreRequest
	if err := c.ShouldBindUri(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: fmt.Sprintf("invalid node_id parameter: %v", err),
		})
		return
	}

	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: fmt.Sprintf("invalid query parameters: %v", err),
		})
		return
	}

	window := h.scoringService.DefaultWindow
	if req.Window != "" {
		parsedWindow, err := config.ParseDuration(req.Window)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error: fmt.Sprintf("invalid window parameter: %v", err),
			})
			return
		}
		window = parsedWindow
	}

	minAttempts := req.MinAttempts
	if minAttempts == 0 {
		minAttempts = 1
	}

	nodeScore, err := h.scoringService.GetNodeScore(c.Request.Context(), req.NodeID, window, minAttempts)
	if err != nil {
		errMsg := err.Error()
		if errMsg == "failed to get node score data: no rows in result set" ||
			errMsg == "no rows in result set" ||
			errMsg == "node not found or no data available" {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error: "node not found or no data available",
			})
			return
		}
		if errMsg == "node does not meet minimum attempts requirement" {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error: errMsg,
			})
			return
		}
		log.Error().Err(err).Int64("node_id", req.NodeID).Msg("Failed to get node score")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "failed to retrieve node score",
		})
		return
	}

	c.JSON(http.StatusOK, NodeScoreResponse{
		Window:      window.String(),
		MinAttempts: minAttempts,
		Score:       *nodeScore,
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
func (h *Handlers) GetHealth(c *gin.Context) {
	health := HealthResponse{
		Status: "healthy",
		Checks: make(map[string]HealthCheck),
	}

	if err := h.database.Ping(c.Request.Context()); err != nil {
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

	c.JSON(statusCode, health)
}
