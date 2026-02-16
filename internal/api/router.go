package api

import (
	"net/http"
	"nosql/internal/api/health"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(10 * time.Second))

	{
		h := health.HealthHandler{}
		r.Get("/health", h.Health)
		r.Get("/unhealth", h.Unhealth)
	}

	return r
}
