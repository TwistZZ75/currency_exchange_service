package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gw-currency-wallet/internal/domain"
	"gw-currency-wallet/internal/storages"
	"gw-currency-wallet/internal/storages/mocks"
)

func TestGetWallet_OK(t *testing.T) {
	st := &mocks.Storage{}
	st.On("GetWallet", mock.Anything, int64(1)).
		Return(&domain.Wallet{UserID: 1, USD: 100, RUB: 5000}, nil).Once()

	svc := &Service{Storage: st, Logger: newLogger()}
	w, err := svc.GetWallet(context.Background(), 1)
	require.NoError(t, err)
	require.EqualValues(t, 100, w.USD)
}

func TestGetWallet_NotFound(t *testing.T) {
	st := &mocks.Storage{}
	st.On("GetWallet", mock.Anything, int64(1)).Return(nil, storages.ErrNotFound).Once()

	svc := &Service{Storage: st, Logger: newLogger()}
	_, err := svc.GetWallet(context.Background(), 1)
	require.ErrorIs(t, err, ErrWalletNotFound)
}

func TestDeposit_InvalidAmount(t *testing.T) {
	svc := &Service{Storage: &mocks.Storage{}, Logger: newLogger()}

	_, err := svc.Deposit(context.Background(), 1, "USD", 0, "")
	require.ErrorIs(t, err, ErrInvalidAmount)

	_, err = svc.Deposit(context.Background(), 1, "USD", -5, "")
	require.ErrorIs(t, err, ErrInvalidAmount)
}

func TestDeposit_InvalidCurrency(t *testing.T) {
	svc := &Service{Storage: &mocks.Storage{}, Logger: newLogger()}
	_, err := svc.Deposit(context.Background(), 1, "BTC", 100, "")
	require.ErrorIs(t, err, ErrInvalidCurrency)
}

func TestDeposit_OK(t *testing.T) {
	st := &mocks.Storage{}
	st.On("Deposit", mock.Anything, mock.AnythingOfType("string"), int64(1), "USD", float64(100)).
		Return(&domain.Wallet{UserID: 1, USD: 100}, nil).Once()

	producer := &fakeProducer{}
	svc := &Service{Storage: st, Logger: newLogger(), Producer: producer, LargeThreshold: 30000}

	w, err := svc.Deposit(context.Background(), 1, "USD", 100, "")
	require.NoError(t, err)
	require.EqualValues(t, 100, w.USD)
	require.Len(t, producer.events, 1, "сервис всегда зовёт SendIfLarge, порог решает Producer")
}

func TestDeposit_IdempotencyKey_PassedThrough(t *testing.T) {
	st := &mocks.Storage{}
	key := "550e8400-e29b-41d4-a716-446655440000"
	st.On("Deposit", mock.Anything, key, int64(1), "USD", float64(10)).
		Return(&domain.Wallet{UserID: 1, USD: 10}, nil).Once()

	svc := &Service{Storage: st, Logger: newLogger()}
	_, err := svc.Deposit(context.Background(), 1, "USD", 10, key)
	require.NoError(t, err)
	st.AssertExpectations(t)
}

func TestDeposit_IdempotencyKey_NonUUID_Hashed(t *testing.T) {
	st := &mocks.Storage{}
	st.On("Deposit", mock.Anything, mock.MatchedBy(func(s string) bool {
		return len(s) == 36 // UUID-формат
	}), int64(1), "USD", float64(10)).
		Return(&domain.Wallet{UserID: 1, USD: 10}, nil).Once()

	svc := &Service{Storage: st, Logger: newLogger()}
	_, err := svc.Deposit(context.Background(), 1, "USD", 10, "abc")
	require.NoError(t, err)
	st.AssertExpectations(t)
}

func TestWithdraw_InsufficientFunds(t *testing.T) {
	st := &mocks.Storage{}
	st.On("Withdraw", mock.Anything, mock.Anything, int64(1), "USD", float64(100)).
		Return(nil, storages.ErrInsufficientFunds).Once()

	svc := &Service{Storage: st, Logger: newLogger()}
	_, err := svc.Withdraw(context.Background(), 1, "USD", 100, "")
	require.ErrorIs(t, err, ErrInsufficientFunds)
}

func TestWithdraw_OK(t *testing.T) {
	st := &mocks.Storage{}
	st.On("Withdraw", mock.Anything, mock.Anything, int64(1), "USD", float64(50)).
		Return(&domain.Wallet{UserID: 1, USD: 50}, nil).Once()

	svc := &Service{Storage: st, Logger: newLogger()}
	w, err := svc.Withdraw(context.Background(), 1, "USD", 50, "")
	require.NoError(t, err)
	require.EqualValues(t, 50, w.USD)
}

func TestWithdraw_InvalidAmount(t *testing.T) {
	svc := &Service{Storage: &mocks.Storage{}, Logger: newLogger()}
	_, err := svc.Withdraw(context.Background(), 1, "USD", 0, "")
	require.ErrorIs(t, err, ErrInvalidAmount)
}

func TestWithdraw_InvalidCurrency(t *testing.T) {
	svc := &Service{Storage: &mocks.Storage{}, Logger: newLogger()}
	_, err := svc.Withdraw(context.Background(), 1, "BTC", 100, "")
	require.ErrorIs(t, err, ErrInvalidCurrency)
}
