package reactions

import (
	"context"
	"errors"
	"strings"
	"time"

	eventsService "nosql/internal/service/events"
)

var ErrEventNotFound = errors.New("Event not found")

type Counts struct {
	Likes    int `json:"likes"`
	Dislikes int `json:"dislikes"`
}

type Repository interface {
	Upsert(ctx context.Context, eventID, userID string, likeValue bool, createdAt time.Time) error
	CountByEventIDs(ctx context.Context, eventIDs []string) (map[string]Counts, error)
}

type Cache interface {
	GetByTitle(ctx context.Context, title string) (Counts, bool, error)
	SetByTitle(ctx context.Context, title string, counts Counts, ttl time.Duration) error
	DeleteByTitle(ctx context.Context, title string) error
}

type EventRepository interface {
	GetByID(ctx context.Context, id string) (eventsService.Event, bool, error)
	ListByTitles(ctx context.Context, titles []string) ([]eventsService.Event, error)
}

type Service struct {
	repo   Repository
	cache  Cache
	events EventRepository
	ttl    time.Duration
}

func NewService(repo Repository, cache Cache, events EventRepository, ttl time.Duration) *Service {
	return &Service{
		repo:   repo,
		cache:  cache,
		events: events,
		ttl:    ttl,
	}
}

func (s *Service) Like(ctx context.Context, eventID, userID string) error {
	return s.setReaction(ctx, eventID, userID, true)
}

func (s *Service) Dislike(ctx context.Context, eventID, userID string) error {
	return s.setReaction(ctx, eventID, userID, false)
}

func (s *Service) GetByTitle(ctx context.Context, title string) (Counts, error) {
	countsByTitle, err := s.GetByTitles(ctx, []string{title})
	if err != nil {
		return Counts{}, err
	}

	title = strings.TrimSpace(title)
	return countsByTitle[title], nil
}

func (s *Service) GetByTitles(ctx context.Context, titles []string) (map[string]Counts, error) {
	cleaned := make([]string, 0, len(titles))
	seen := make(map[string]struct{}, len(titles))

	for _, title := range titles {
		title = strings.TrimSpace(title)
		if title == "" {
			continue
		}
		if _, ok := seen[title]; ok {
			continue
		}

		seen[title] = struct{}{}
		cleaned = append(cleaned, title)
	}

	result := make(map[string]Counts, len(cleaned))
	if len(cleaned) == 0 {
		return result, nil
	}

	missingTitles := make([]string, 0, len(cleaned))

	for _, title := range cleaned {
		if s.cache == nil {
			missingTitles = append(missingTitles, title)
			continue
		}

		counts, found, err := s.cache.GetByTitle(ctx, title)
		if err != nil {
			return nil, err
		}
		if found {
			result[title] = counts
			continue
		}

		missingTitles = append(missingTitles, title)
	}

	if len(missingTitles) == 0 {
		return result, nil
	}

	events, err := s.events.ListByTitles(ctx, missingTitles)
	if err != nil {
		return nil, err
	}

	eventIDsByTitle := make(map[string][]string, len(missingTitles))
	allEventIDs := make([]string, 0, len(events))

	for _, event := range events {
		title := strings.TrimSpace(event.Title)
		if title == "" || event.ID.IsZero() {
			continue
		}

		eventID := event.ID.Hex()
		eventIDsByTitle[title] = append(eventIDsByTitle[title], eventID)
		allEventIDs = append(allEventIDs, eventID)
	}

	countsByEventID := map[string]Counts{}
	if len(allEventIDs) > 0 {
		countsByEventID, err = s.repo.CountByEventIDs(ctx, allEventIDs)
		if err != nil {
			return nil, err
		}
	}

	for _, title := range missingTitles {
		counts := Counts{}
		eventIDs := eventIDsByTitle[title]

		for _, eventID := range eventIDs {
			eventCounts := countsByEventID[eventID]
			counts.Likes += eventCounts.Likes
			counts.Dislikes += eventCounts.Dislikes
		}

		result[title] = counts

		if s.cache != nil && len(eventIDs) > 0 {
			if err := s.cache.SetByTitle(ctx, title, counts, s.ttl); err != nil {
				return nil, err
			}
		}
	}

	return result, nil
}

func (s *Service) setReaction(ctx context.Context, eventID, userID string, likeValue bool) error {
	event, found, err := s.events.GetByID(ctx, strings.TrimSpace(eventID))
	if err != nil {
		return err
	}
	if !found {
		return ErrEventNotFound
	}

	if err := s.repo.Upsert(ctx, event.ID.Hex(), strings.TrimSpace(userID), likeValue, time.Now().UTC()); err != nil {
		return err
	}

	if s.cache != nil {
		_ = s.cache.DeleteByTitle(ctx, event.Title)

		if _, err := s.GetByTitle(ctx, event.Title); err != nil {
			return err
		}
	}

	return nil
}
