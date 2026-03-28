package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	stdhttp "net/http"
	"time"

	api "nosql/internal/api"
	authHTTP "nosql/internal/api/auth"
	eventsHTTP "nosql/internal/api/events"
	healthHTTP "nosql/internal/api/health"
	sessionHTTP "nosql/internal/api/session"
	usersHTTP "nosql/internal/api/users"
	"nosql/internal/config"
	mongoInfra "nosql/internal/infrastructure/mongo"
	redisInfra "nosql/internal/infrastructure/redis"
	authService "nosql/internal/service/auth"
	eventsService "nosql/internal/service/events"
	sessionService "nosql/internal/service/session"
	usersService "nosql/internal/service/users"

	goredis "github.com/redis/go-redis/v9"
	gomongo "go.mongodb.org/mongo-driver/mongo"
)

type App struct {
	server *stdhttp.Server
	redis  *goredis.Client
	mongo  *gomongo.Client
}

func NewApp(cfg config.Config) (*App, error) {
	redisClient := redisInfra.NewClient(cfg)

	sessionRepo := redisInfra.NewSessionRepository(redisClient)
	sessionSvc := sessionService.NewService(sessionRepo, cfg.UserSessionTTL)

	healthHandler := healthHTTP.NewHandler(cfg.UserSessionTTL)
	sessionHandler := sessionHTTP.NewHandler(sessionSvc, cfg.UserSessionTTL)

	var mongoClient *gomongo.Client
	var userHandler *usersHTTP.Handler
	var authHandler *authHTTP.Handler
	var eventHandler *eventsHTTP.Handler

	if cfg.Mongo.Enabled {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var err error
		mongoClient, err = mongoInfra.NewClient(ctx, cfg)
		if err != nil {
			_ = redisClient.Close()
			return nil, err
		}

		db := mongoClient.Database(cfg.Mongo.Database)

		if err := mongoInfra.EnsureIndexes(ctx, db); err != nil {
			_ = mongoClient.Disconnect(context.Background())
			_ = redisClient.Close()
			return nil, err
		}

		userRepo := mongoInfra.NewUserRepository(db)
		eventRepo := mongoInfra.NewEventRepository(db)

		userSvc := usersService.NewService(userRepo)
		authSvc := authService.NewService(userRepo)
		eventSvc := eventsService.NewService(eventRepo)

		userHandler = usersHTTP.NewHandler(userSvc, sessionSvc, cfg.UserSessionTTL)
		authHandler = authHTTP.NewHandler(authSvc, sessionSvc, cfg.UserSessionTTL)
		eventHandler = eventsHTTP.NewHandler(eventSvc, sessionSvc, cfg.UserSessionTTL)
	}

	router := api.NewRouter(
		healthHandler,
		sessionHandler,
		userHandler,
		authHandler,
		eventHandler,
	)

	return &App{
		redis: redisClient,
		mongo: mongoClient,
		server: &stdhttp.Server{
			Addr:              fmt.Sprintf(":%d", cfg.Port),
			Handler:           router,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}, nil
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

	var mongoErr error
	if a.mongo != nil {
		mongoErr = a.mongo.Disconnect(ctx)
	}

	var redisErr error
	if a.redis != nil {
		redisErr = a.redis.Close()
	}

	if serverErr != nil {
		return serverErr
	}
	if mongoErr != nil {
		return mongoErr
	}
	if redisErr != nil {
		return redisErr
	}

	return nil
}
