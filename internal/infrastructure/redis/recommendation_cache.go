package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	eventsService "nosql/internal/service/events"

	goredis "github.com/redis/go-redis/v9"
)

const recommendationFieldEvents = "events"

type RecommendationCache struct {
	client *goredis.Client
}

func NewRecommendationCache(client *goredis.Client) *RecommendationCache {
	return &RecommendationCache{client: client}
}

func (c *RecommendationCache) GetByUserID(ctx context.Context, userID string) ([]eventsService.Event, bool, error) {
	value, err := c.client.HGet(ctx, c.key(userID), recommendationFieldEvents).Result()
	if err == goredis.Nil {
		return []eventsService.Event{}, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	var events []eventsService.Event
	if err := json.Unmarshal([]byte(value), &events); err != nil {
		return nil, false, err
	}
	return events, true, nil
}

func (c *RecommendationCache) SetByUserID(ctx context.Context, userID string, events []eventsService.Event, ttl time.Duration) error {
	data, err := json.Marshal(events)
	if err != nil {
		return err
	}

	key := c.key(userID)
	pipe := c.client.TxPipeline()
	pipe.HSet(ctx, key, recommendationFieldEvents, string(data))
	pipe.Expire(ctx, key, ttl)
	_, err = pipe.Exec(ctx)
	return err
}

func (c *RecommendationCache) key(userID string) string {
	return fmt.Sprintf("user:%s:recomms", userID)
}
