package redis

import (
	"fmt"

	goredis "github.com/redis/go-redis/v9"

	"nosql/internal/config"
)

func NewClient(cfg config.Config) *goredis.Client {
	return goredis.NewClient(&goredis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
}
