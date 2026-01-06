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
)

type Server struct {
	server   *http.Server
	handlers *Handlers
	engine   *gin.Engine
}

func NewServer(database *db.DB, cfg *config.Config) *Server {
	handlers := NewHandlers(database, cfg)

	// Set Gin to release mode for production
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()

	// Middleware
	engine.Use(ginLogger())
	engine.Use(gin.Recovery())
	engine.Use(ginTimeout(60 * time.Second))

	// Swagger documentation routes
	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API routes
	v1 := engine.Group("/api/v1")
	{
		v1.GET("/scores", handlers.GetTopScores)
		v1.GET("/scores/node/:node_id", handlers.GetNodeScore)
		v1.GET("/health", handlers.GetHealth)
	}

	// Initialize swagger info
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

// ginLogger is a custom logger middleware for Gin that uses zerolog
func ginLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Log request
		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()

		if raw != "" {
			path = path + "?" + raw
		}

		log.Info().
			Str("method", method).
			Str("path", path).
			Int("status", statusCode).
			Dur("latency", latency).
			Str("ip", clientIP).
			Str("error", errorMessage).
			Msg("HTTP request")
	}
}

// ginTimeout creates a timeout middleware for Gin
func ginTimeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
