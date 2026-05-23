package recommendations

import (
	"context"
	"sort"
	"strings"
	"time"

	eventsService "nosql/internal/service/events"
)

type Candidate struct {
	EventID string
	Score   int
}

type Graph interface {
	Recommendations(ctx context.Context, userID string) ([]Candidate, error)
}

type Cache interface {
	GetByUserID(ctx context.Context, userID string) ([]eventsService.Event, bool, error)
	SetByUserID(ctx context.Context, userID string, events []eventsService.Event, ttl time.Duration) error
}

type EventRepository interface {
	ListByIDs(ctx context.Context, ids []string) ([]eventsService.Event, error)
}

type Service struct {
	graph  Graph
	cache  Cache
	events EventRepository
	ttl    time.Duration
}

func NewService(graph Graph, cache Cache, events EventRepository, ttl time.Duration) *Service {
	return &Service{
		graph:  graph,
		cache:  cache,
		events: events,
		ttl:    ttl,
	}
}

func (s *Service) List(ctx context.Context, userID string) ([]eventsService.Event, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return []eventsService.Event{}, nil
	}

	if s.cache != nil {
		events, found, err := s.cache.GetByUserID(ctx, userID)
		if err != nil {
			return nil, err
		}
		if found {
			return events, nil
		}
	}

	candidates, err := s.graph.Recommendations(ctx, userID)
	if err != nil {
		return nil, err
	}

	events, err := s.resolveEvents(ctx, candidates)
	if err != nil {
		return nil, err
	}

	if s.cache != nil {
		if err := s.cache.SetByUserID(ctx, userID, events, s.ttl); err != nil {
			return nil, err
		}
	}

	return events, nil
}

func (s *Service) resolveEvents(ctx context.Context, candidates []Candidate) ([]eventsService.Event, error) {
	if len(candidates) == 0 {
		return []eventsService.Event{}, nil
	}

	ids := make([]string, 0, len(candidates))
	scoreByID := make(map[string]int, len(candidates))
	for _, candidate := range candidates {
		id := strings.TrimSpace(candidate.EventID)
		if id == "" {
			continue
		}
		ids = append(ids, id)
		scoreByID[id] = candidate.Score
	}

	events, err := s.events.ListByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	byTitle := make(map[string]rankedEvent, len(events))
	for _, event := range events {
		title := strings.TrimSpace(event.Title)
		if title == "" || event.ID.IsZero() {
			continue
		}

		current := byTitle[title]
		current.score += scoreByID[event.ID.Hex()]
		if current.event.ID.IsZero() || startsBefore(event, current.event) {
			current.event = event
		}
		byTitle[title] = current
	}

	ranked := make([]rankedEvent, 0, len(byTitle))
	for _, item := range byTitle {
		ranked = append(ranked, item)
	}

	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].score != ranked[j].score {
			return ranked[i].score > ranked[j].score
		}
		return startsBefore(ranked[i].event, ranked[j].event)
	})

	result := make([]eventsService.Event, 0, len(ranked))
	for _, item := range ranked {
		result = append(result, item.event)
	}
	return result, nil
}

type rankedEvent struct {
	event eventsService.Event
	score int
}

func startsBefore(a, b eventsService.Event) bool {
	aTime, aErr := time.Parse(time.RFC3339, a.StartedAt)
	bTime, bErr := time.Parse(time.RFC3339, b.StartedAt)
	if aErr == nil && bErr == nil {
		return aTime.Before(bTime)
	}
	return a.StartedAt < b.StartedAt
}
