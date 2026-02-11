package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/threefoldtech/deployment-checker/docs/swagger"
	"github.com/threefoldtech/deployment-checker/pkg/config"
	"github.com/threefoldtech/deployment-checker/pkg/db"
	"github.com/threefoldtech/deployment-checker/pkg/services"
)

type Server struct {
	server   *http.Server
	handlers *Handlers
	engine   *gin.Engine
}

func NewServer(database *db.DB, cfg *config.Config, scoringService *services.ScoringService) *Server {
	handlers := NewHandlers(database, cfg, scoringService)

	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()

	engine.Use(ginLogger())
	engine.Use(gin.Recovery())
	engine.Use(ginTimeout(60 * time.Second))

	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1 := engine.Group("/api/v1")
	{
		v1.GET("/scores", handlers.GetTopScores)
		v1.GET("/scores/node/:node_id", handlers.GetNodeScore)
		v1.GET("/health", handlers.GetHealth)
	}

	swagger.SwaggerInfo.Host = fmt.Sprintf("%s:%d", cfg.API.Host, cfg.API.Port)
	swagger.SwaggerInfo.BasePath = "/api/v1"
	swagger.SwaggerInfo.Schemes = []string{"http", "https"}

	addr := fmt.Sprintf("%s:%d", cfg.API.Host, cfg.API.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: engine,
	}

	return &Server{
		server:   server,
		handlers: handlers,
		engine:   engine,
	}
}

func (s *Server) Start() error {
	log.Info().
		Str("address", s.server.Addr).
		Msg("Starting API server")
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	log.Info().Msg("Shutting down API server")
	return s.server.Shutdown(ctx)
}
