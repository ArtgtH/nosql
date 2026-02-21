package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port int
}

func Load() (Config, error) {
	_ = godotenv.Load(".env.local")
	portStr := os.Getenv("APP_PORT")

	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 1000 || port > 65535 {
		return Config{}, fmt.Errorf("invalid APP_PORT=%q", portStr)
	}

	return Config{Port: port}, nil
}
