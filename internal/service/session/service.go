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
	session, found, err := s.RefreshIfExists(ctx, sid)
	if err != nil {
		return UpsertResult{}, err
	}
	if found {
		return UpsertResult{
			Session: session,
			Created: false,
		}, nil
	}

	session, err = s.Create(ctx, "")
	if err != nil {
		return UpsertResult{}, err
	}

	return UpsertResult{
		Session: session,
		Created: true,
	}, nil
}

func (s *Service) RefreshIfExists(ctx context.Context, sid string) (Session, bool, error) {
	if !IsValidSID(sid) {
		return Session{}, false, nil
	}

	now := time.Now().UTC()

	found, err := s.repo.Refresh(ctx, sid, now, s.ttl)
	if err != nil {
		return Session{}, false, err
	}
	if !found {
		return Session{}, false, nil
	}

	session, exists, err := s.repo.Get(ctx, sid)
	if err != nil {
		return Session{}, false, err
	}
	if exists {
		return session, true, nil
	}

	return Session{
		ID:        sid,
		UpdatedAt: now,
	}, true, nil
}

func (s *Service) AttachUser(ctx context.Context, sid, userID string) (Session, error) {
	now := time.Now().UTC()

	if IsValidSID(sid) {
		found, err := s.repo.SetUser(ctx, sid, userID, now, s.ttl)
		if err != nil {
			return Session{}, err
		}
		if found {
			return Session{
				ID:        sid,
				UserID:    userID,
				UpdatedAt: now,
			}, nil
		}
	}

	return s.Create(ctx, userID)
}

func (s *Service) Get(ctx context.Context, sid string) (Session, bool, error) {
	if !IsValidSID(sid) {
		return Session{}, false, nil
	}

	return s.repo.Get(ctx, sid)
}

func (s *Service) Delete(ctx context.Context, sid string) error {
	if !IsValidSID(sid) {
		return nil
	}

	return s.repo.Delete(ctx, sid)
}

func (s *Service) Create(ctx context.Context, userID string) (Session, error) {
	now := time.Now().UTC()

	for i := 0; i < 5; i++ {
		newSID, err := NewSID()
		if err != nil {
			return Session{}, err
		}

		newSession := Session{
			ID:        newSID,
			UserID:    userID,
			CreatedAt: now,
			UpdatedAt: now,
		}

		created, err := s.repo.Create(ctx, newSession, s.ttl)
		if err != nil {
			return Session{}, err
		}
		if created {
			return newSession, nil
		}
	}

	return Session{}, ErrUnableToCreateSession
}
