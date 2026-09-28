package service

import (
	"context"
	"errors"
	"fmt"

	"gw-currency-wallet/internal/auth"
	"gw-currency-wallet/internal/domain"
	"gw-currency-wallet/internal/storages"
)

var (
	ErrUserExists         = errors.New("username or email already exists")
	ErrInvalidCredentials = errors.New("invalid username or password")
)

func (s *Service) Register(ctx context.Context, username, password, email string) error {
	hash, err := auth.HashPassword(password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	u := &domain.User{Username: username, Password: hash, Email: email}
	if _, err := s.Storage.CreateUser(ctx, u); err != nil {
		if errors.Is(err, storages.ErrUserExists) {
			return ErrUserExists
		}
		return err
	}
	s.Logger.InfoContext(ctx, "user registered", "username", username)
	return nil
}

func (s *Service) Login(ctx context.Context, username, password string) (string, error) {
	u, err := s.Storage.GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, storages.ErrNotFound) {
			return "", ErrInvalidCredentials
		}
		return "", err
	}
	if !auth.CheckPassword(u.Password, password) {
		return "", ErrInvalidCredentials
	}
	token, err := s.JWT.Generate(u.ID, u.Username)
	if err != nil {
		return "", fmt.Errorf("generate jwt: %w", err)
	}
	s.Logger.InfoContext(ctx, "user logged in", "user_id", u.ID)
	return token, nil
}
