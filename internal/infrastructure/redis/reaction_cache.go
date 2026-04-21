package redis

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"

	reactionsService "nosql/internal/service/reactions"
)

type ReactionCache struct {
	client *goredis.Client
}

func NewReactionCache(client *goredis.Client) *ReactionCache {
	return &ReactionCache{client: client}
}

func (c *ReactionCache) GetByTitle(ctx context.Context, title string) (reactionsService.Counts, bool, error) {
	value, err := c.client.Get(ctx, c.key(title)).Result()
	if err != nil {
		if err == goredis.Nil {
			return reactionsService.Counts{}, false, nil
		}
		return reactionsService.Counts{}, false, err
	}

	var counts reactionsService.Counts
	if err := json.Unmarshal([]byte(value), &counts); err != nil {
		return reactionsService.Counts{}, false, err
	}

	return counts, true, nil
}

func (c *ReactionCache) SetByTitle(ctx context.Context, title string, counts reactionsService.Counts, ttl time.Duration) error {
	payload, err := json.Marshal(counts)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, c.key(title), payload, ttl).Err()
}

func (c *ReactionCache) DeleteByTitle(ctx context.Context, title string) error {
	return c.client.Del(ctx, c.key(title)).Err()
}

func (c *ReactionCache) key(title string) string {
	sum := md5.Sum([]byte(strings.TrimSpace(title)))
	return fmt.Sprintf("events:%s:reactions", hex.EncodeToString(sum[:]))
}
