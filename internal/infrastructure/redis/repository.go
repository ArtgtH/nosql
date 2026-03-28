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
				fields := map[string]any{
					"created_at": s.CreatedAt.Format(time.RFC3339),
					"updated_at": s.UpdatedAt.Format(time.RFC3339),
				}
				if s.UserID != "" {
					fields["user_id"] = s.UserID
				}

				pipe.HSet(ctx, key, fields)
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

func (r *SessionRepository) Get(
	ctx context.Context,
	sid string,
) (sessionService.Session, bool, error) {
	key := r.key(sid)

	values, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		return sessionService.Session{}, false, err
	}
	if len(values) == 0 {
		return sessionService.Session{}, false, nil
	}

	session := sessionService.Session{
		ID:     sid,
		UserID: values["user_id"],
	}

	if createdAt := values["created_at"]; createdAt != "" {
		t, err := time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return sessionService.Session{}, false, err
		}
		session.CreatedAt = t
	}

	if updatedAt := values["updated_at"]; updatedAt != "" {
		t, err := time.Parse(time.RFC3339, updatedAt)
		if err != nil {
			return sessionService.Session{}, false, err
		}
		session.UpdatedAt = t
	}

	return session, true, nil
}

func (r *SessionRepository) SetUser(
	ctx context.Context,
	sid string,
	userID string,
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
				pipe.HSet(ctx, key,
					"user_id", userID,
					"updated_at", updatedAt.Format(time.RFC3339),
				)
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

func (r *SessionRepository) Delete(ctx context.Context, sid string) error {
	return r.client.Del(ctx, r.key(sid)).Err()
}
