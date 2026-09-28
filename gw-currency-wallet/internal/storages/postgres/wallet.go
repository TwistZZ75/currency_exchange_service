package postgres

import (
	"context"
	"errors"
	"fmt"
	"gw-currency-wallet/internal/domain"
	"gw-currency-wallet/internal/storages"

	"github.com/jackc/pgx/v5"
)

func (s *Storage) GetWallet(ctx context.Context, userID int64) (*domain.Wallet, error) {
	const query = `SELECT user_id, usd, rub, eur FROM wallets WHERE user_id = $1`

	var wallet domain.Wallet
	err := s.pool.QueryRow(ctx, query, userID).Scan(&wallet.UserID, &wallet.USD, &wallet.RUB, &wallet.EUR)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storages.ErrNotFound
		}
		return nil, fmt.Errorf("select wallet: %w", err)
	}
	return &wallet, nil
}

func columnForCurrency(c string) (string, error) {
	switch c {
	case "USD":
		return "usd", nil
	case "RUB":
		return "rub", nil
	case "EUR":
		return "eur", nil
	}
	return "", fmt.Errorf("unsupported currency: %s", c)
}
