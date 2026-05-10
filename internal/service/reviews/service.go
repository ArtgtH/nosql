package reviews

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	eventsService "nosql/internal/service/events"
)

var (
	ErrInvalidComment = errors.New(`invalid "comment" field`)
	ErrInvalidRating  = errors.New(`invalid "rating" field`)
	ErrAlreadyExists  = errors.New("Already exists")
	ErrEventNotFound  = errors.New("Event not found")
	ErrReviewNotFound = errors.New("Review not found")
)

type Review struct {
	ID        string
	EventID   string
	Rating    int8
	Comment   string
	CreatedAt time.Time
	CreatedBy string
	UpdatedAt time.Time
}

type Counts struct {
	Count  int     `json:"count"`
	Rating float64 `json:"rating"`
}

type Patch struct {
	Comment *string
	Rating  *int8
}

type Repository interface {
	Create(ctx context.Context, review Review) error
	GetByEventAndUser(ctx context.Context, eventID, userID string) (Review, bool, error)
	GetByEventAndID(ctx context.Context, eventID, reviewID string) (Review, bool, error)
	UpdateByEventAndUser(ctx context.Context, eventID, userID string, patch Patch, updatedAt time.Time) error
	ListByEventID(ctx context.Context, eventID string, limit, offset uint64) ([]Review, error)
	ListByEventIDs(ctx context.Context, eventIDs []string) (map[string][]Review, error)
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
	return &Service{repo: repo, cache: cache, events: events, ttl: ttl}
}

func (s *Service) Create(ctx context.Context, eventID, userID, comment string, rating int8) (string, error) {
	event, found, err := s.events.GetByID(ctx, strings.TrimSpace(eventID))
	if err != nil {
		return "", err
	}
	if !found {
		return "", ErrEventNotFound
	}

	comment = strings.TrimSpace(comment)
	if comment == "" || len([]rune(comment)) > 300 {
		return "", ErrInvalidComment
	}
	if rating < 1 || rating > 5 {
		return "", ErrInvalidRating
	}

	_, exists, err := s.repo.GetByEventAndUser(ctx, event.ID.Hex(), strings.TrimSpace(userID))
	if err != nil {
		return "", err
	}
	if exists {
		return "", ErrAlreadyExists
	}

	now := time.Now().UTC()
	reviewID, err := newUUID()
	if err != nil {
		return "", err
	}
	review := Review{
		ID:        reviewID,
		EventID:   event.ID.Hex(),
		Rating:    rating,
		Comment:   comment,
		CreatedAt: now,
		CreatedBy: strings.TrimSpace(userID),
		UpdatedAt: now,
	}
	if err := s.repo.Create(ctx, review); err != nil {
		return "", err
	}

	if err := s.refreshTitleCache(ctx, event.Title); err != nil {
		return "", err
	}

	return review.ID, nil
}

func newUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

func (s *Service) ListByEventID(ctx context.Context, eventID string, limit, offset uint64) ([]Review, error) {
	event, found, err := s.events.GetByID(ctx, strings.TrimSpace(eventID))
	if err != nil {
		return nil, err
	}
	if !found {
		return []Review{}, nil
	}
	return s.repo.ListByEventID(ctx, event.ID.Hex(), limit, offset)
}

func (s *Service) Update(ctx context.Context, eventID, reviewID, userID string, patch Patch) error {
	event, found, err := s.events.GetByID(ctx, strings.TrimSpace(eventID))
	if err != nil {
		return err
	}
	if !found {
		return ErrEventNotFound
	}

	if patch.Comment != nil {
		comment := strings.TrimSpace(*patch.Comment)
		if comment == "" || len([]rune(comment)) > 300 {
			return ErrInvalidComment
		}
		patch.Comment = &comment
	}
	if patch.Rating != nil && (*patch.Rating < 1 || *patch.Rating > 5) {
		return ErrInvalidRating
	}

	review, found, err := s.repo.GetByEventAndID(ctx, event.ID.Hex(), strings.TrimSpace(reviewID))
	if err != nil {
		return err
	}
	if !found || review.CreatedBy != strings.TrimSpace(userID) {
		return ErrReviewNotFound
	}

	if err := s.repo.UpdateByEventAndUser(ctx, event.ID.Hex(), review.CreatedBy, patch, time.Now().UTC()); err != nil {
		return err
	}

	return s.refreshTitleCache(ctx, event.Title)
}

func (s *Service) GetByTitle(ctx context.Context, title string) (Counts, error) {
	countsByTitle, err := s.GetByTitles(ctx, []string{title})
	if err != nil {
		return Counts{}, err
	}
	return countsByTitle[strings.TrimSpace(title)], nil
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
	missing := make([]string, 0, len(cleaned))
	for _, title := range cleaned {
		if s.cache == nil {
			missing = append(missing, title)
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
		missing = append(missing, title)
	}

	if len(missing) == 0 {
		return result, nil
	}

	fresh, err := s.countByTitles(ctx, missing)
	if err != nil {
		return nil, err
	}
	for _, title := range missing {
		counts := fresh[title]
		result[title] = counts
		if s.cache != nil && counts.Count > 0 {
			if err := s.cache.SetByTitle(ctx, title, counts, s.ttl); err != nil {
				return nil, err
			}
		}
	}

	return result, nil
}

func (s *Service) refreshTitleCache(ctx context.Context, title string) error {
	if s.cache == nil {
		return nil
	}
	_ = s.cache.DeleteByTitle(ctx, title)
	counts, err := s.countByTitle(ctx, title)
	if err != nil {
		return err
	}
	if counts.Count == 0 {
		return nil
	}
	return s.cache.SetByTitle(ctx, title, counts, s.ttl)
}

func (s *Service) countByTitle(ctx context.Context, title string) (Counts, error) {
	countsByTitle, err := s.countByTitles(ctx, []string{title})
	if err != nil {
		return Counts{}, err
	}
	return countsByTitle[strings.TrimSpace(title)], nil
}

func (s *Service) countByTitles(ctx context.Context, titles []string) (map[string]Counts, error) {
	result := make(map[string]Counts, len(titles))
	events, err := s.events.ListByTitles(ctx, titles)
	if err != nil {
		return nil, err
	}

	eventIDsByTitle := make(map[string][]string, len(titles))
	allEventIDs := make([]string, 0, len(events))
	for _, event := range events {
		title := strings.TrimSpace(event.Title)
		if title == "" || event.ID.IsZero() {
			continue
		}
		id := event.ID.Hex()
		eventIDsByTitle[title] = append(eventIDsByTitle[title], id)
		allEventIDs = append(allEventIDs, id)
	}

	reviewsByEventID := map[string][]Review{}
	if len(allEventIDs) > 0 {
		reviewsByEventID, err = s.repo.ListByEventIDs(ctx, allEventIDs)
		if err != nil {
			return nil, err
		}
	}

	for _, title := range titles {
		title = strings.TrimSpace(title)
		count := 0
		sum := 0
		for _, eventID := range eventIDsByTitle[title] {
			for _, review := range reviewsByEventID[eventID] {
				count++
				sum += int(review.Rating)
			}
		}
		result[title] = Counts{Count: count, Rating: average(sum, count)}
	}

	return result, nil
}

func average(sum, count int) float64 {
	if count == 0 {
		return 0
	}
	return math.Round((float64(sum)/float64(count))*10) / 10
}
