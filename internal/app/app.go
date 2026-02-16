package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"nosql/internal/api"
	"time"

	"nosql/internal/config"
)

type App struct {
	server *http.Server
}

func NewApp(cfg config.Config) *App {
	return &App{
		server: &http.Server{
			Addr:              fmt.Sprintf(":%d", cfg.Port),
			Handler:           api.NewRouter(),
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
}

func (a *App) Run() error {
	fmt.Println("===Server Started===")
	fmt.Printf("Listening on port %s", a.server.Addr)
	err := a.server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (a *App) Shutdown(ctx context.Context) error {
	return a.server.Shutdown(ctx)
}
