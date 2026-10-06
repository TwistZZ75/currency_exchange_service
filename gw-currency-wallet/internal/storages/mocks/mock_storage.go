package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"gw-currency-wallet/internal/domain"
)

type Storage struct{ mock.Mock }

func (m *Storage) CreateUser(ctx context.Context, u *domain.User) (int64, error) {
	args := m.Called(ctx, u)
	return args.Get(0).(int64), args.Error(1)
}

func (m *Storage) GetUserByUsername(ctx context.Context, username string) (*domain.User, error) {
	args := m.Called(ctx, username)
	u, _ := args.Get(0).(*domain.User)
	return u, args.Error(1)
}

func (m *Storage) GetUserByID(ctx context.Context, id int64) (*domain.User, error) {
	args := m.Called(ctx, id)
	u, _ := args.Get(0).(*domain.User)
	return u, args.Error(1)
}

func (m *Storage) GetWallet(ctx context.Context, userID int64) (*domain.Wallet, error) {
	args := m.Called(ctx, userID)
	w, _ := args.Get(0).(*domain.Wallet)
	return w, args.Error(1)
}

func (m *Storage) Deposit(ctx context.Context, txID string, userID int64, currency string, amount float64) (*domain.Wallet, error) {
	args := m.Called(ctx, txID, userID, currency, amount)
	w, _ := args.Get(0).(*domain.Wallet)
	return w, args.Error(1)
}

func (m *Storage) Withdraw(ctx context.Context, txID string, userID int64, currency string, amount float64) (*domain.Wallet, error) {
	args := m.Called(ctx, txID, userID, currency, amount)
	w, _ := args.Get(0).(*domain.Wallet)
	return w, args.Error(1)
}

func (m *Storage) Exchange(
	ctx context.Context,
	txID string,
	userID int64,
	from, to string,
	amount, rate, received float64,
) (*domain.Wallet, *domain.Operation, error) {
	args := m.Called(ctx, txID, userID, from, to, amount, rate, received)
	w, _ := args.Get(0).(*domain.Wallet)
	op, _ := args.Get(1).(*domain.Operation)
	return w, op, args.Error(2)
}

func (m *Storage) GetOperationByTxID(ctx context.Context, txID string) (*domain.Operation, error) {
	args := m.Called(ctx, txID)
	o, _ := args.Get(0).(*domain.Operation)
	return o, args.Error(1)
}

func (m *Storage) Ping(ctx context.Context) error { return m.Called(ctx).Error(0) }
func (m *Storage) Close()                         { m.Called() }
