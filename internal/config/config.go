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

	port, err := getPortEnv("APP_PORT")
	if err != nil {
		return Config{}, err
	}

	ttlSeconds, err := getPositiveIntEnv("APP_USER_SESSION_TTL")
	if err != nil {
		return Config{}, err
	}

	redisHost, err := getRequiredEnv("REDIS_HOST")
	if err != nil {
		return Config{}, err
	}
	if redisHost == "" {
		return Config{}, fmt.Errorf("REDIS_HOST is required")
	}

	redisPort, err := getPortEnv("REDIS_PORT")
	if err != nil {
		return Config{}, err
	}

	redisDBStr, err := getRequiredEnv("REDIS_DB")
	if err != nil {
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

func getRequiredEnv(envName string) (string, error) {
	value, ok := os.LookupEnv(envName)
	if !ok || value == "" {
		return "", fmt.Errorf("%s is required", envName)
	}
	return value, nil
}

func getPortEnv(envName string) (int, error) {
	value, err := getRequiredEnv(envName)
	if err != nil {
		return 0, err
	}

	port, err := strconv.Atoi(value)
	if err != nil || port <= 1000 || port > 65535 {
		return 0, fmt.Errorf("invalid %s=%q", envName, value)
	}
	return port, nil
}

func getPositiveIntEnv(envName string) (int, error) {
	value, err := getRequiredEnv(envName)
	if err != nil {
		return 0, err
	}

	number, err := strconv.Atoi(value)
	if err != nil || number < 0 {
		return 0, fmt.Errorf("invalid %s=%q", envName, value)
	}
	return number, nil
}
