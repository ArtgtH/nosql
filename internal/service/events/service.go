package events

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const DefaultCategory = "other"

var validCategories = map[string]struct{}{
	"meetup":     {},
	"concert":    {},
	"exhibition": {},
	"party":      {},
	"other":      {},
}

var (
	ErrInvalidTitle                = errors.New(`invalid "title" field`)
	ErrInvalidAddress              = errors.New(`invalid "address" field`)
	ErrInvalidStartedAt            = errors.New(`invalid "started_at" field`)
	ErrInvalidFinishedAt           = errors.New(`invalid "finished_at" field`)
	ErrInvalidCategory             = errors.New(`invalid "category" field`)
	ErrInvalidPrice                = errors.New(`invalid "price" field`)
	ErrInvalidCity                 = errors.New(`invalid "city" field`)
	ErrEventExists                 = errors.New("event already exists")
	ErrEventNotFoundOrNotOrganizer = errors.New("Not found. Be sure that event exists and you are the organizer")
)

type Location struct {
	Address string `bson:"address" json:"address"`
	City    string `bson:"city,omitempty" json:"city,omitempty"`
}

type Event struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Title       string             `bson:"title" json:"title"`
	Category    string             `bson:"category,omitempty" json:"category"`
	Price       uint64             `bson:"price,omitempty" json:"price"`
	Description string             `bson:"description" json:"description"`
	Location    Location           `bson:"location" json:"location"`
	CreatedAt   string             `bson:"created_at" json:"created_at"`
	CreatedBy   string             `bson:"created_by" json:"created_by"`
	StartedAt   string             `bson:"started_at" json:"started_at"`
	FinishedAt  string             `bson:"finished_at" json:"finished_at"`
}

type ListFilter struct {
	ID              string
	Title           string
	Category        string
	City            string
	CreatedBy       string
	DateFrom        string
	DateToExclusive string
	PriceFrom       *uint64
	PriceTo         *uint64
	Limit           uint64
	Offset          uint64
}

type EventPatch struct {
	Category *string
	Price    *uint64
	City     *string
}

type Repository interface {
	Create(ctx context.Context, event Event) (string, error)
	ExistsByTitle(ctx context.Context, title string) (bool, error)
	GetByID(ctx context.Context, id string) (Event, bool, error)
	UpdateByOrganizer(ctx context.Context, eventID, organizerID string, patch EventPatch) (bool, error)
	List(ctx context.Context, filter ListFilter) ([]Event, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(
	ctx context.Context,
	createdBy string,
	title string,
	address string,
	startedAt string,
	finishedAt string,
	description string,
) (string, error) {
	title = strings.TrimSpace(title)
	address = strings.TrimSpace(address)
	description = strings.TrimSpace(description)

	if title == "" {
		return "", ErrInvalidTitle
	}
	if address == "" {
		return "", ErrInvalidAddress
	}

	parsedStartedAt, err := time.Parse(time.RFC3339, startedAt)
	if err != nil {
		return "", ErrInvalidStartedAt
	}

	parsedFinishedAt, err := time.Parse(time.RFC3339, finishedAt)
	if err != nil {
		return "", ErrInvalidFinishedAt
	}
	if !parsedFinishedAt.After(parsedStartedAt) {
		return "", ErrInvalidFinishedAt
	}

	exists, err := s.repo.ExistsByTitle(ctx, title)
	if err != nil {
		return "", err
	}
	if exists {
		return "", ErrEventExists
	}

	event := Event{
		Title:       title,
		Category:    DefaultCategory,
		Price:       0,
		Description: description,
		Location: Location{
			Address: address,
		},
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
		CreatedBy:  createdBy,
		StartedAt:  parsedStartedAt.Format(time.RFC3339),
		FinishedAt: parsedFinishedAt.Format(time.RFC3339),
	}

	return s.repo.Create(ctx, event)
}

func (s *Service) Patch(ctx context.Context, eventID, organizerID string, patch EventPatch) error {
	if patch.Category != nil {
		value := strings.TrimSpace(*patch.Category)
		if _, ok := validCategories[value]; !ok {
			return ErrInvalidCategory
		}
		patch.Category = &value
	}

	if patch.City != nil {
		value := strings.TrimSpace(*patch.City)
		patch.City = &value
	}

	if !patch.hasChanges() {
		event, found, err := s.repo.GetByID(ctx, eventID)
		if err != nil {
			return err
		}
		if !found || event.CreatedBy != organizerID {
			return ErrEventNotFoundOrNotOrganizer
		}
		return nil
	}

	updated, err := s.repo.UpdateByOrganizer(ctx, eventID, organizerID, patch)
	if err != nil {
		return err
	}
	if !updated {
		return ErrEventNotFoundOrNotOrganizer
	}

	return nil
}

func (s *Service) GetByID(ctx context.Context, id string) (Event, bool, error) {
	event, found, err := s.repo.GetByID(ctx, strings.TrimSpace(id))
	if err != nil {
		return Event{}, false, err
	}
	if !found {
		return Event{}, false, nil
	}

	event.Category = NormalizeCategory(event.Category)
	return event, true, nil
}

func (s *Service) List(ctx context.Context, filter ListFilter) ([]Event, error) {
	filter.ID = strings.TrimSpace(filter.ID)
	filter.Title = strings.TrimSpace(filter.Title)
	filter.Category = strings.TrimSpace(filter.Category)
	filter.City = strings.TrimSpace(filter.City)
	filter.CreatedBy = strings.TrimSpace(filter.CreatedBy)

	events, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	for i := range events {
		events[i].Category = NormalizeCategory(events[i].Category)
	}

	return events, nil
}

func NormalizeCategory(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return DefaultCategory
	}
	return value
}

func (p EventPatch) hasChanges() bool {
	return p.Category != nil || p.Price != nil || p.City != nil
}
