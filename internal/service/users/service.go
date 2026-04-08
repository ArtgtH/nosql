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
	ID           primitive.ObjectID `bson:"_id,omitempty"`
	FullName     string             `bson:"full_name"`
	Username     string             `bson:"username"`
	PasswordHash string             `bson:"password_hash"`
}

type Repository interface {
	Create(ctx context.Context, user User) (string, error)
	FindByUsername(ctx context.Context, username string) (User, bool, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
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

	return s.repo.Create(ctx, User{
		FullName:     fullName,
		Username:     username,
		PasswordHash: string(hash),
	})
}
