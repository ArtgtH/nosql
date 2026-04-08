package session

import (
	"context"
	"time"
)

const (
	CookieName = "X-Session-Id"
	KeyPrefix  = "sid:"
)

type Session struct {
	ID        string
	UserID    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type SessionRepository interface {
	Create(ctx context.Context, s Session, ttl time.Duration) (bool, error)
	Refresh(ctx context.Context, sid string, updatedAt time.Time, ttl time.Duration) (bool, error)
	Get(ctx context.Context, sid string) (Session, bool, error)
	SetUser(ctx context.Context, sid string, userID string, updatedAt time.Time, ttl time.Duration) (bool, error)
	Delete(ctx context.Context, sid string) error
}
