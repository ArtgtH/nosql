package api

import (
	"net/http"
	"time"

	authHTTP "nosql/internal/api/auth"
	eventsHTTP "nosql/internal/api/events"
	healthHTTP "nosql/internal/api/health"
	sessionHTTP "nosql/internal/api/session"
	usersHTTP "nosql/internal/api/users"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(
	healthHandler *healthHTTP.Handler,
	sessionHandler *sessionHTTP.Handler,
	userHandler *usersHTTP.Handler,
	authHandler *authHTTP.Handler,
	eventHandler *eventsHTTP.Handler,
) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(10 * time.Second))

	r.Get("/health", healthHandler.Health)
	r.Post("/session", sessionHandler.CreateOrRefresh)

	if userHandler != nil {
		r.Post("/users", userHandler.Create)
		r.Get("/users", userHandler.List)
		r.Get("/users/{id}", userHandler.GetByID)
		r.Get("/users/{id}/events", userHandler.ListEvents)
	}

	if authHandler != nil {
		r.Post("/auth/login", authHandler.Login)
		r.Post("/auth/logout", authHandler.Logout)
	}

	if eventHandler != nil {
		r.Post("/events", eventHandler.Create)
		r.Get("/events", eventHandler.List)
		r.Get("/events/{id}", eventHandler.GetByID)
		r.Patch("/events/{id}", eventHandler.Patch)
		if eventHandler.HasReactions() {
			r.Post("/events/{id}/like", eventHandler.Like)
			r.Post("/events/{id}/dislike", eventHandler.Dislike)
		}
	}

	return r
}
