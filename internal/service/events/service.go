package events

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	ErrInvalidTitle      = errors.New(`invalid "title" field`)
	ErrInvalidAddress    = errors.New(`invalid "address" field`)
	ErrInvalidStartedAt  = errors.New(`invalid "started_at" field`)
	ErrInvalidFinishedAt = errors.New(`invalid "finished_at" field`)
	ErrEventExists       = errors.New("event already exists")
)

type Location struct {
	Address string `bson:"address" json:"address"`
}

type Event struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	Title       string             `bson:"title"`
	Description string             `bson:"description"`
	Location    Location           `bson:"location"`
	CreatedAt   string             `bson:"created_at"`
	CreatedBy   string             `bson:"created_by"`
	StartedAt   string             `bson:"started_at"`
	FinishedAt  string             `bson:"finished_at"`
}

type ListFilter struct {
	Title  string
	Limit  uint64
	Offset uint64
}

type Repository interface {
	Create(ctx context.Context, event Event) (string, error)
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

	event := Event{
		Title:       title,
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

func (s *Service) List(ctx context.Context, title string, limit uint64, offset uint64) ([]Event, error) {
	return s.repo.List(ctx, ListFilter{
		Title:  strings.TrimSpace(title),
		Limit:  limit,
		Offset: offset,
	})
}
