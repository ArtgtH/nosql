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

type MongoConfig struct {
	Enabled  bool
	Database string
	User     string
	Password string
	Host     string
	Port     int
}

type Config struct {
	Port           int
	UserSessionTTL time.Duration
	Redis          RedisConfig
	Mongo          MongoConfig
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

	mongoDatabase := firstNonEmpty(os.Getenv("MONGODB_DATABASE"), os.Getenv("MONGODB_DATABSE"))
	mongoUser := os.Getenv("MONGODB_USER")
	mongoPassword := os.Getenv("MONGODB_PASSWORD")
	mongoHost := os.Getenv("MONGODB_HOST")
	mongoPortStr := os.Getenv("MONGODB_PORT")

	mongoAnySet := mongoDatabase != "" || mongoUser != "" || mongoPassword != "" || mongoHost != "" || mongoPortStr != ""

	cfg := Config{
		Port:           port,
		UserSessionTTL: time.Duration(ttlSeconds) * time.Second,
		Redis: RedisConfig{
			Host:     redisHost,
			Port:     redisPort,
			Password: os.Getenv("REDIS_PASSWORD"),
			DB:       redisDB,
		},
	}

	if !mongoAnySet {
		return cfg, nil
	}

	if mongoDatabase == "" {
		return Config{}, fmt.Errorf("MONGODB_DATABASE is required")
	}
	if mongoUser == "" {
		return Config{}, fmt.Errorf("MONGODB_USER is required")
	}
	if mongoPassword == "" {
		return Config{}, fmt.Errorf("MONGODB_PASSWORD is required")
	}
	if mongoHost == "" {
		return Config{}, fmt.Errorf("MONGODB_HOST is required")
	}

	mongoPort, err := strconv.Atoi(mongoPortStr)
	if err != nil || mongoPort <= 1000 || mongoPort > 65535 {
		return Config{}, fmt.Errorf("invalid MONGODB_PORT=%q", mongoPortStr)
	}

	cfg.Mongo = MongoConfig{
		Enabled:  true,
		Database: mongoDatabase,
		User:     mongoUser,
		Password: mongoPassword,
		Host:     mongoHost,
		Port:     mongoPort,
	}

	return cfg, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
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
