package postgres

import (
	"context"
	"errors"
	"fmt"
	"gw-currency-wallet/internal/domain"
	"gw-currency-wallet/internal/storages"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (s *Storage) CreateUser(ctx context.Context, user *domain.User) (int64, error) {
	const query = `INSERT INTO users (username, email, password) VALUES ($1, $2, $3) RETURNING id`

	var id int64
	err := s.pool.QueryRow(ctx, query, user.Username, user.Email, user.Password).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return 0, storages.ErrUserExists
		}
		return 0, fmt.Errorf("insert user: %w", err)
	}

	// сразу создаём пустой кошелёк
	if _, err := s.pool.Exec(ctx, `INSERT INTO wallets (user_id) VALUES ($1)`, id); err != nil {
		return 0, fmt.Errorf("insert wallet: %w", err)
	}

	return id, nil
}

func (s *Storage) GetUserByUsername(ctx context.Context, username string) (*domain.User, error) {
	const query = `SELECT id, username, email, password, created_at FROM users WHERE username = $1`

	var user domain.User
	err := s.pool.QueryRow(ctx, query, username).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storages.ErrNotFound
		}
		return nil, fmt.Errorf("select user: %w", err)
	}
	return &user, nil
}

func (s *Storage) GetUserByID(ctx context.Context, id int64) (*domain.User, error) {
	const query = `SELECT id, username, email, password, created_at FROM users WHERE id = $1`

	var user domain.User
	err := s.pool.QueryRow(ctx, query, id).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storages.ErrNotFound
		}
		return nil, fmt.Errorf("select user by id: %w", err)
	}
	return &user, nil
}
