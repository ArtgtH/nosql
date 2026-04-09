package main

import (
	"context"
	"log"
	"nosql/internal/config"
	"os"
	"os/signal"
	"syscall"
	"time"

	"nosql/internal/app"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	a, err := app.NewApp(cfg)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		if err := a.Run(); err != nil {
			log.Fatal(err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := a.Shutdown(ctx); err != nil {
		log.Fatal(err)
	}
}
