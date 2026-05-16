package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
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

type CassandraConfig struct {
	Enabled     bool
	Hosts       []string
	Port        int
	Username    string
	Password    string
	Keyspace    string
	Consistency string
}

type Config struct {
	Port           int
	UserSessionTTL time.Duration
	LikeTTL        time.Duration
	Redis          RedisConfig
	Mongo          MongoConfig
	Cassandra      CassandraConfig
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

	likeTTLSeconds, err := getPositiveIntEnvOrDefault("APP_LIKE_TTL", 60)
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

	cfg := Config{
		Port:           port,
		UserSessionTTL: time.Duration(ttlSeconds) * time.Second,
		LikeTTL:        time.Duration(likeTTLSeconds) * time.Second,
		Redis: RedisConfig{
			Host:     redisHost,
			Port:     redisPort,
			Password: os.Getenv("REDIS_PASSWORD"),
			DB:       redisDB,
		},
	}

	mongoDatabase := firstNonEmpty(os.Getenv("MONGODB_DATABASE"), os.Getenv("MONGODB_DATABSE"))
	mongoUser := os.Getenv("MONGODB_USER")
	mongoPassword := os.Getenv("MONGODB_PASSWORD")
	mongoHost := os.Getenv("MONGODB_HOST")
	mongoPortStr := os.Getenv("MONGODB_PORT")

	mongoAnySet := mongoDatabase != "" || mongoUser != "" || mongoPassword != "" || mongoHost != "" || mongoPortStr != ""
	if mongoAnySet {
		if mongoDatabase == "" {
			return Config{}, fmt.Errorf("MONGODB_DATABASE is required")
		}
		if mongoHost == "" {
			return Config{}, fmt.Errorf("MONGODB_HOST is required")
		}
		if mongoPortStr == "" {
			return Config{}, fmt.Errorf("MONGODB_PORT is required")
		}
		if (mongoUser == "") != (mongoPassword == "") {
			return Config{}, fmt.Errorf("MONGODB_USER and MONGODB_PASSWORD must be set together")
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
	}

	cassandraHostsRaw := os.Getenv("CASSANDRA_HOSTS")
	cassandraPortStr := os.Getenv("CASSANDRA_PORT")
	cassandraUsername := os.Getenv("CASSANDRA_USERNAME")
	cassandraPassword := os.Getenv("CASSANDRA_PASSWORD")
	cassandraKeyspace := strings.TrimSpace(os.Getenv("CASSANDRA_KEYSPACE"))
	cassandraConsistency := strings.TrimSpace(os.Getenv("CASSANDRA_CONSISTENCY"))

	cassandraAnySet := cassandraHostsRaw != "" || cassandraPortStr != "" || cassandraUsername != "" || cassandraPassword != "" || cassandraKeyspace != "" || cassandraConsistency != ""
	if cassandraAnySet {
		if cassandraHostsRaw == "" {
			return Config{}, fmt.Errorf("CASSANDRA_HOSTS is required")
		}
		if cassandraPortStr == "" {
			return Config{}, fmt.Errorf("CASSANDRA_PORT is required")
		}
		if cassandraKeyspace == "" {
			return Config{}, fmt.Errorf("CASSANDRA_KEYSPACE is required")
		}
		if cassandraConsistency == "" {
			cassandraConsistency = "ONE"
		}
		if (cassandraUsername == "") != (cassandraPassword == "") {
			return Config{}, fmt.Errorf("CASSANDRA_USERNAME and CASSANDRA_PASSWORD must be set together")
		}

		cassandraPort, err := strconv.Atoi(cassandraPortStr)
		if err != nil || cassandraPort <= 1000 || cassandraPort > 65535 {
			return Config{}, fmt.Errorf("invalid CASSANDRA_PORT=%q", cassandraPortStr)
		}

		hosts := splitWithTrim(cassandraHostsRaw, ",")
		if len(hosts) == 0 {
			return Config{}, fmt.Errorf("CASSANDRA_HOSTS is required")
		}
		if !isValidIdentifier(cassandraKeyspace) {
			return Config{}, fmt.Errorf("invalid CASSANDRA_KEYSPACE=%q", cassandraKeyspace)
		}

		cfg.Cassandra = CassandraConfig{
			Enabled:     true,
			Hosts:       hosts,
			Port:        cassandraPort,
			Username:    cassandraUsername,
			Password:    cassandraPassword,
			Keyspace:    cassandraKeyspace,
			Consistency: strings.ToUpper(cassandraConsistency),
		}
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

func splitWithTrim(value, separator string) []string {
	parts := strings.Split(value, separator)
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func isValidIdentifier(value string) bool {
	if value == "" {
		return false
	}
	for i, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9' && i > 0:
		case r == '_' && i > 0:
		default:
			return false
		}
	}
	return true
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

func getPositiveIntEnvOrDefault(envName string, defaultValue int) (int, error) {
	value := strings.TrimSpace(os.Getenv(envName))
	if value == "" {
		return defaultValue, nil
	}

	number, err := strconv.Atoi(value)
	if err != nil || number < 0 {
		return 0, fmt.Errorf("invalid %s=%q", envName, value)
	}

	return number, nil
}
