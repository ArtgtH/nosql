package api

import (
	"net/http"
	authHTTP "nosql/internal/api/auth"
	eventsHTTP "nosql/internal/api/events"
	healthHTTP "nosql/internal/api/health"
	sessionHTTP "nosql/internal/api/session"
	usersHTTP "nosql/internal/api/users"
	"time"

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
	}
	if authHandler != nil {
		r.Post("/auth/login", authHandler.Login)
		r.Post("/auth/logout", authHandler.Logout)
	}
	if eventHandler != nil {
		r.Post("/events", eventHandler.Create)
		r.Get("/events", eventHandler.List)
	}

	return r
}
