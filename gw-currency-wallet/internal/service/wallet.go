package service

import (
	"context"
	"errors"
	"gw-exchanger/pkg"
	"time"

	"github.com/google/uuid"

	"gw-currency-wallet/internal/domain"
	"gw-currency-wallet/internal/storages"
)

var (
	ErrWalletNotFound      = errors.New("wallet not found")
	ErrInvalidAmount       = errors.New("invalid amount")
	ErrInvalidCurrency     = errors.New("invalid currency")
	ErrInsufficientFunds   = errors.New("insufficient funds")
	ErrInvalidCurrencyPair = errors.New("invalid currency pair")
)

func (s *Service) GetWallet(ctx context.Context, userID int64) (*domain.Wallet, error) {
	w, err := s.Storage.GetWallet(ctx, userID)
	if err != nil {
		if errors.Is(err, storages.ErrNotFound) {
			return nil, ErrWalletNotFound
		}
		return nil, err
	}
	return w, nil
}

func (s *Service) Deposit(ctx context.Context, userID int64, currency string, amount float64, idempotencyKey string) (*domain.Wallet, error) {
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}
	currency, err := pkg.NormalizeCurrency(currency)
	if err != nil {
		return nil, ErrInvalidCurrency
	}

	txID := pickTxID(idempotencyKey)

	w, err := s.Storage.Deposit(ctx, txID, userID, currency, amount)
	if err != nil {
		return nil, err
	}

	s.publishIfLarge(ctx, domain.TransferEvent{
		TransactionID: txID,
		UserID:        userID,
		Type:          string(domain.OpDeposit),
		ToCurrency:    currency,
		Amount:        amount,
		Timestamp:     time.Now(),
	})

	s.Logger.InfoContext(ctx, "deposit", "user_id", userID, "amount", amount, "currency", currency)
	return w, nil
}

func (s *Service) Withdraw(ctx context.Context, userID int64, currency string, amount float64, idempotencyKey string) (*domain.Wallet, error) {
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}
	currency, err := pkg.NormalizeCurrency(currency)
	if err != nil {
		return nil, ErrInvalidCurrency
	}

	txID := pickTxID(idempotencyKey)

	w, err := s.Storage.Withdraw(ctx, txID, userID, currency, amount)
	if err != nil {
		if errors.Is(err, storages.ErrInsufficientFunds) {
			return nil, ErrInsufficientFunds
		}
		return nil, err
	}

	s.publishIfLarge(ctx, domain.TransferEvent{
		TransactionID: txID,
		UserID:        userID,
		Type:          string(domain.OpWithdraw),
		FromCurrency:  currency,
		Amount:        amount,
		Timestamp:     time.Now(),
	})

	s.Logger.InfoContext(ctx, "withdraw", "user_id", userID, "amount", amount, "currency", currency)
	return w, nil
}

func (s *Service) publishIfLarge(ctx context.Context, ev domain.TransferEvent) {
	if s.Producer == nil {
		return
	}
	if err := s.Producer.SendIfLarge(ctx, ev); err != nil {
		s.Logger.ErrorContext(ctx, "kafka send failed", "error", err, "transaction_id", ev.TransactionID)
	}
}

func pickTxID(key string) string {
	if key == "" {
		return uuid.NewString()
	}
	if _, err := uuid.Parse(key); err == nil {
		return key
	}
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(key)).String()
}
