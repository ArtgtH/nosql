package session

import (
	"context"
	"time"
)

const (
	CookieName = "X-Session-ID"
	KeyPrefix  = "sid:"
)

type Session struct {
	ID        string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type SessionRepository interface {
	Create(ctx context.Context, s Session, ttl time.Duration) (bool, error)
	Refresh(ctx context.Context, sid string, updatedAt time.Time, ttl time.Duration) (bool, error)
}
