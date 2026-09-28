package storages

import (
	"context"
	"gw-currency-wallet/internal/domain"
)

type Storage interface {
	// Users
	CreateUser(ctx context.Context, user *domain.User) (int64, error)
	GetUserByUsername(ctx context.Context, username string) (*domain.User, error)
	GetUserByID(ctx context.Context, userID int64) (*domain.User, error)

	// Wallet
	GetWallet(ctx context.Context, userID int64) (*domain.Wallet, error)

	// Operations
	Deposit(ctx context.Context, txID string, userID int64, currency string, amount float64) (*domain.Wallet, error)
	Withdraw(ctx context.Context, txID string, userID int64, currency string, amount float64) (*domain.Wallet, error)
	Exchange(ctx context.Context, txID string, userID int64,
		from_currency, to_currency string, amount, rate, received float64) (*domain.Wallet, *domain.Operation, error)

	// DB interactions
	Ping(ctx context.Context) error
	Close()
}
