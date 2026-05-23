package users

import (
	"context"
	"errors"
	"strings"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidFullName = errors.New(`invalid "full_name" field`)
	ErrInvalidUsername = errors.New(`invalid "username" field`)
	ErrInvalidPassword = errors.New(`invalid "password" field`)
	ErrUserExists      = errors.New("user already exists")
)

type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	FullName     string             `bson:"full_name" json:"full_name"`
	Username     string             `bson:"username" json:"username"`
	PasswordHash string             `bson:"password_hash" json:"-"`
}

type ListFilter struct {
	ID     string
	Name   string
	Limit  uint64
	Offset uint64
}

type Repository interface {
	Create(ctx context.Context, user User) (string, error)
	FindByUsername(ctx context.Context, username string) (User, bool, error)
	FindByID(ctx context.Context, id string) (User, bool, error)
	List(ctx context.Context, filter ListFilter) ([]User, error)
}

type Graph interface {
	CreateUser(ctx context.Context, id string) error
}

type Service struct {
	repo  Repository
	graph Graph
}

func NewService(repo Repository, graph ...Graph) *Service {
	service := &Service{repo: repo}
	if len(graph) > 0 {
		service.graph = graph[0]
	}
	return service
}

func (s *Service) Create(ctx context.Context, fullName, username, password string) (string, error) {
	fullName = strings.TrimSpace(fullName)
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)

	if fullName == "" {
		return "", ErrInvalidFullName
	}
	if username == "" {
		return "", ErrInvalidUsername
	}
	if password == "" {
		return "", ErrInvalidPassword
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	id, err := s.repo.Create(ctx, User{
		FullName:     fullName,
		Username:     username,
		PasswordHash: string(hash),
	})
	if err != nil {
		return "", err
	}
	if s.graph != nil {
		if err := s.graph.CreateUser(ctx, id); err != nil {
			return "", err
		}
	}
	return id, nil
}

func (s *Service) FindByUsername(ctx context.Context, username string) (User, bool, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return User{}, false, nil
	}
	return s.repo.FindByUsername(ctx, username)
}

func (s *Service) GetByID(ctx context.Context, id string) (User, bool, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return User{}, false, nil
	}
	return s.repo.FindByID(ctx, id)
}

func (s *Service) List(ctx context.Context, filter ListFilter) ([]User, error) {
	filter.ID = strings.TrimSpace(filter.ID)
	filter.Name = strings.TrimSpace(filter.Name)
	return s.repo.List(ctx, filter)
}
