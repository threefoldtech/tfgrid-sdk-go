package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/log"
	httpSwagger "github.com/swaggo/http-swagger"
	"github.com/threefoldtech/provision-probe/docs/swagger"
	"github.com/threefoldtech/provision-probe/pkg/config"
	"github.com/threefoldtech/provision-probe/pkg/db"
)

type Server struct {
	server   *http.Server
	handlers *Handlers
}

func NewServer(database *db.DB, cfg *config.Config) *Server {
	handlers := NewHandlers(database, cfg)

	router := chi.NewRouter()
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(60 * time.Second))

	// Swagger documentation routes
	router.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"), // Relative URL works better
	))

	router.Route("/api/v1", func(r chi.Router) {
		r.Get("/scores", handlers.GetTopScores)
		r.Route("/scores/node", func(r chi.Router) {
			r.Get("/{node_id}", handlers.GetNodeScore)
		})
		r.Get("/health", handlers.GetHealth)
	})

	// Initialize swagger info
	swagger.SwaggerInfo.Host = fmt.Sprintf("%s:%d", cfg.API.Host, cfg.API.Port)
	swagger.SwaggerInfo.BasePath = "/api/v1"
	swagger.SwaggerInfo.Schemes = []string{"http", "https"}

	addr := fmt.Sprintf("%s:%d", cfg.API.Host, cfg.API.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	return &Server{
		server:   server,
		handlers: handlers,
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
