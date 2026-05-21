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

	reactionsService "nosql/internal/service/reactions"
)

const (
	reactionFieldLikes    = "likes"
	reactionFieldDislikes = "dislikes"
)

type ReactionCache struct {
	client *goredis.Client
}

func NewReactionCache(client *goredis.Client) *ReactionCache {
	return &ReactionCache{client: client}
}

func (c *ReactionCache) GetByTitle(ctx context.Context, title string) (reactionsService.Counts, bool, error) {
	values, err := c.client.HMGet(ctx, c.key(title), reactionFieldLikes, reactionFieldDislikes).Result()
	if err != nil {
		return reactionsService.Counts{}, false, err
	}

	if len(values) != 2 || values[0] == nil || values[1] == nil {
		return reactionsService.Counts{}, false, nil
	}

	likes, err := strconv.Atoi(fmt.Sprint(values[0]))
	if err != nil {
		return reactionsService.Counts{}, false, err
	}

	dislikes, err := strconv.Atoi(fmt.Sprint(values[1]))
	if err != nil {
		return reactionsService.Counts{}, false, err
	}

	return reactionsService.Counts{
		Likes:    likes,
		Dislikes: dislikes,
	}, true, nil
}

func (c *ReactionCache) SetByTitle(ctx context.Context, title string, counts reactionsService.Counts, ttl time.Duration) error {
	key := c.key(title)

	pipe := c.client.TxPipeline()
	pipe.HSet(ctx, key,
		reactionFieldLikes, counts.Likes,
		reactionFieldDislikes, counts.Dislikes,
	)
	pipe.Expire(ctx, key, ttl)

	_, err := pipe.Exec(ctx)
	return err
}

func (c *ReactionCache) DeleteByTitle(ctx context.Context, title string) error {
	return c.client.Del(ctx, c.key(title)).Err()
}

func (c *ReactionCache) key(title string) string {
	sum := md5.Sum([]byte(strings.TrimSpace(title)))
	return fmt.Sprintf("event:%s:reactions", hex.EncodeToString(sum[:]))
}
