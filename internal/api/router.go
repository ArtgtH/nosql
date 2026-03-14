package api

import (
	"net/http"
	healthHTTP "nosql/internal/api/health"
	sessionHTTP "nosql/internal/api/session"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(
	healthHandler *healthHTTP.Handler,
	sessionHandler *sessionHTTP.Handler,
) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(10 * time.Second))

	r.Get("/health", healthHandler.Health)
	r.Post("/session", sessionHandler.CreateOrRefresh)

	return r
}
