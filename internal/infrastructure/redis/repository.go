package redis

import (
	"context"
	"errors"
	"time"

	goredis "github.com/redis/go-redis/v9"

	sessionService "nosql/internal/service/session"
)

const maxRetries = 5

type SessionRepository struct {
	client *goredis.Client
}

func NewSessionRepository(client *goredis.Client) *SessionRepository {
	return &SessionRepository{client: client}
}

func (r *SessionRepository) key(sid string) string {
	return sessionService.KeyPrefix + sid
}

func (r *SessionRepository) Create(
	ctx context.Context,
	s sessionService.Session,
	ttl time.Duration,
) (bool, error) {
	key := r.key(s.ID)

	for i := 0; i < maxRetries; i++ {
		var created bool

		err := r.client.Watch(ctx, func(tx *goredis.Tx) error {
			exists, err := tx.Exists(ctx, key).Result()
			if err != nil {
				return err
			}
			if exists == 1 {
				created = false
				return nil
			}

			_, err = tx.TxPipelined(ctx, func(pipe goredis.Pipeliner) error {
				pipe.HSet(ctx, key,
					"created_at", s.CreatedAt.Format(time.RFC3339),
					"updated_at", s.UpdatedAt.Format(time.RFC3339),
				)
				pipe.Expire(ctx, key, ttl)
				return nil
			})
			if err != nil {
				return err
			}

			created = true
			return nil
		}, key)

		if err == nil {
			return created, nil
		}
		if errors.Is(err, goredis.TxFailedErr) {
			continue
		}
		return false, err
	}

	return false, goredis.TxFailedErr
}

func (r *SessionRepository) Refresh(
	ctx context.Context,
	sid string,
	updatedAt time.Time,
	ttl time.Duration,
) (bool, error) {
	key := r.key(sid)

	for i := 0; i < maxRetries; i++ {
		var found bool

		err := r.client.Watch(ctx, func(tx *goredis.Tx) error {
			exists, err := tx.Exists(ctx, key).Result()
			if err != nil {
				return err
			}
			if exists == 0 {
				found = false
				return nil
			}

			_, err = tx.TxPipelined(ctx, func(pipe goredis.Pipeliner) error {
				pipe.HSet(ctx, key, "updated_at", updatedAt.Format(time.RFC3339))
				pipe.Expire(ctx, key, ttl)
				return nil
			})
			if err != nil {
				return err
			}

			found = true
			return nil
		}, key)

		if err == nil {
			return found, nil
		}
		if errors.Is(err, goredis.TxFailedErr) {
			continue
		}
		return false, err
	}

	return false, goredis.TxFailedErr
}
