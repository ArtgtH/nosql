package redis

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"

	reviewsService "nosql/internal/service/reviews"
)

const (
	reviewFieldCount  = "count"
	reviewFieldRating = "rating"
)

type ReviewCache struct {
	client *goredis.Client
}

func NewReviewCache(client *goredis.Client) *ReviewCache {
	return &ReviewCache{client: client}
}

func (c *ReviewCache) GetByTitle(ctx context.Context, title string) (reviewsService.Counts, bool, error) {
	values, err := c.client.HMGet(ctx, c.key(title), reviewFieldCount, reviewFieldRating).Result()
	if err != nil {
		return reviewsService.Counts{}, false, err
	}
	if len(values) != 2 || values[0] == nil || values[1] == nil {
		return reviewsService.Counts{}, false, nil
	}

	count, err := strconv.Atoi(fmt.Sprint(values[0]))
	if err != nil {
		return reviewsService.Counts{}, false, err
	}
	rating, err := strconv.ParseFloat(fmt.Sprint(values[1]), 64)
	if err != nil {
		return reviewsService.Counts{}, false, err
	}

	return reviewsService.Counts{Count: count, Rating: rating}, true, nil
}

func (c *ReviewCache) SetByTitle(ctx context.Context, title string, counts reviewsService.Counts, ttl time.Duration) error {
	key := c.key(title)
	pipe := c.client.TxPipeline()
	pipe.HSet(ctx, key,
		reviewFieldCount, counts.Count,
		reviewFieldRating, strconv.FormatFloat(counts.Rating, 'f', 1, 64),
	)
	pipe.Expire(ctx, key, ttl)
	_, err := pipe.Exec(ctx)
	return err
}

func (c *ReviewCache) DeleteByTitle(ctx context.Context, title string) error {
	return c.client.Del(ctx, c.key(title)).Err()
}

func (c *ReviewCache) key(title string) string {
	sum := md5.Sum([]byte(strings.TrimSpace(title)))
	return fmt.Sprintf("event:%s:reviews", hex.EncodeToString(sum[:]))
}
