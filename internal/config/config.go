package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

type Config struct {
	Port           int
	UserSessionTTL time.Duration
	Redis          RedisConfig
}

func Load() (Config, error) {
	_ = godotenv.Load(".env.local")

	port, err := mustPort("APP_PORT")
	if err != nil {
		return Config{}, err
	}

	ttlSeconds, err := mustPositiveInt("APP_USER_SESSION_TTL")
	if err != nil {
		return Config{}, err
	}

	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		return Config{}, fmt.Errorf("REDIS_HOST is required")
	}

	redisPort, err := mustPort("REDIS_PORT")
	if err != nil {
		return Config{}, err
	}

	redisDBStr := os.Getenv("REDIS_DB")
	if redisDBStr == "" {
		return Config{}, fmt.Errorf("REDIS_DB is required")
	}
	redisDB, err := strconv.Atoi(redisDBStr)
	if err != nil || redisDB < 0 {
		return Config{}, fmt.Errorf("invalid REDIS_DB=%q", redisDBStr)
	}

	return Config{
		Port:           port,
		UserSessionTTL: time.Duration(ttlSeconds) * time.Second,
		Redis: RedisConfig{
			Host:     redisHost,
			Port:     redisPort,
			Password: os.Getenv("REDIS_PASSWORD"),
			DB:       redisDB,
		},
	}, nil
}

func mustPort(envName string) (int, error) {
	value := os.Getenv(envName)
	port, err := strconv.Atoi(value)
	if err != nil || port <= 1000 || port > 65535 {
		return 0, fmt.Errorf("invalid %s=%q", envName, value)
	}
	return port, nil
}

func mustPositiveInt(envName string) (int, error) {
	value := os.Getenv(envName)
	number, err := strconv.Atoi(value)
	if err != nil || number < 0 {
		return 0, fmt.Errorf("invalid %s=%q", envName, value)
	}
	return number, nil
}
