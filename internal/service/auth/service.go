package auth

import (
	"context"
	"errors"
	"strings"

	usersService "nosql/internal/service/users"

	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

type Service struct {
	users usersService.Repository
}

func NewService(users usersService.Repository) *Service {
	return &Service{users: users}
}

func (s *Service) Login(ctx context.Context, username, password string) (string, error) {
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)

	if username == "" {
		return "", usersService.ErrInvalidUsername
	}
	if password == "" {
		return "", usersService.ErrInvalidPassword
	}

	user, found, err := s.users.FindByUsername(ctx, username)
	if err != nil {
		return "", err
	}
	if !found {
		return "", ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", ErrInvalidCredentials
	}

	return user.ID.Hex(), nil
}
