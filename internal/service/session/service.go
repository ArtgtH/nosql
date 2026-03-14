package session

import (
	"context"
	"errors"
	"time"
)

var ErrUnableToCreateSession = errors.New("unable to create session")

type Service struct {
	repo SessionRepository
	ttl  time.Duration
}

func NewService(repo SessionRepository, ttl time.Duration) *Service {
	return &Service{repo: repo, ttl: ttl}
}

type UpsertResult struct {
	Session Session
	Created bool
}

func (s *Service) Upsert(ctx context.Context, sid string) (UpsertResult, error) {
	now := time.Now().UTC()

	if IsValidSID(sid) {
		found, err := s.repo.Refresh(ctx, sid, now, s.ttl)
		if err != nil {
			return UpsertResult{}, err
		}
		if found {
			return UpsertResult{
				Session: Session{
					ID:        sid,
					UpdatedAt: now,
				},
				Created: false,
			}, nil
		}
	}

	for i := 0; i < 5; i++ {
		newSID, err := NewSID()
		if err != nil {
			return UpsertResult{}, err
		}

		newSession := Session{
			ID:        newSID,
			CreatedAt: now,
			UpdatedAt: now,
		}

		created, err := s.repo.Create(ctx, newSession, s.ttl)
		if err != nil {
			return UpsertResult{}, err
		}
		if created {
			return UpsertResult{
				Session: newSession,
				Created: true,
			}, nil
		}
	}

	return UpsertResult{}, ErrUnableToCreateSession
}
