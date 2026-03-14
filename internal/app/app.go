package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	stdhttp "net/http"
	"time"

	api "nosql/internal/api"
	healthHTTP "nosql/internal/api/health"
	sessionHTTP "nosql/internal/api/session"
	"nosql/internal/config"
	redisInfra "nosql/internal/infrastructure/redis"
	sessionservice "nosql/internal/service/session"

	goredis "github.com/redis/go-redis/v9"
)

type App struct {
	server *stdhttp.Server
	redis  *goredis.Client
}

func NewApp(cfg config.Config) *App {
	redisClient := redisInfra.NewClient(cfg)

	sessionRepo := redisInfra.NewSessionRepository(redisClient)
	sessionService := sessionservice.NewService(sessionRepo, cfg.UserSessionTTL)

	healthHandler := healthHTTP.NewHandler(cfg.UserSessionTTL)
	sessionHandler := sessionHTTP.NewHandler(sessionService, cfg.UserSessionTTL)

	router := api.NewRouter(healthHandler, sessionHandler)

	return &App{
		redis: redisClient,
		server: &stdhttp.Server{
			Addr:              fmt.Sprintf(":%d", cfg.Port),
			Handler:           router,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
}

func (a *App) Run() error {
	log.Println("=== Server Started ===")
	log.Printf("Listening on %s\n", a.server.Addr)

	err := a.server.ListenAndServe()
	if errors.Is(err, stdhttp.ErrServerClosed) {
		return nil
	}

	return err
}

func (a *App) Shutdown(ctx context.Context) error {
	serverErr := a.server.Shutdown(ctx)

	if a.redis != nil {
		if err := a.redis.Close(); err != nil && serverErr == nil {
			return err
		}
	}

	return serverErr
}
