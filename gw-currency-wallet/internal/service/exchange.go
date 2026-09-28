package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gw-currency-wallet/internal/domain"
	"gw-currency-wallet/internal/storages"
	"gw-currency-wallet/pkg/utils"
)

func (s *Service) GetRates(ctx context.Context) (map[string]float64, error) {
	rates, err := s.Rates.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("get rates: %w", err)
	}
	return rates, nil
}

func (s *Service) Exchange(ctx context.Context, userID int64, from, to string, amount float64, idempotencyKey string) (*domain.Wallet, float64, float64, error) {
	if amount <= 0 {
		return nil, 0, 0, ErrInvalidAmount
	}
	var err error
	if from, err = utils.NormalizeCurrency(from); err != nil {
		return nil, 0, 0, ErrInvalidCurrency
	}
	if to, err = utils.NormalizeCurrency(to); err != nil {
		return nil, 0, 0, ErrInvalidCurrency
	}
	if from == to {
		return nil, 0, 0, ErrInvalidCurrencyPair
	}

	// Получаем курсы валют через кэш (TTL 30 сек)
	rates, err := s.Rates.Get(ctx)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("get rates: %w", err)
	}
	fromRate, ok := rates[from]
	if !ok {
		return nil, 0, 0, ErrInvalidCurrency
	}
	toRate, ok := rates[to]
	if !ok {
		return nil, 0, 0, ErrInvalidCurrency
	}
	if toRate == 0 {
		return nil, 0, 0, fmt.Errorf("zero rate for %s", to)
	}

	rate := fromRate / toRate
	received := utils.Round2(amount * rate)

	txID := pickTxID(idempotencyKey)

	w, op, err := s.Storage.Exchange(ctx, txID, userID, from, to, amount, rate, received)
	if err != nil {
		if errors.Is(err, storages.ErrInsufficientFunds) {
			return nil, 0, 0, ErrInsufficientFunds
		}
		return nil, 0, 0, err
	}

	// после обмена сбрасываем кэш курсов — следующий запрос пойдёт за свежим
	s.Rates.Invalidate()

	s.publishIfLarge(ctx, domain.TransferEvent{
		TransactionID: txID,
		UserID:        userID,
		Type:          string(domain.OpExchange),
		FromCurrency:  from,
		ToCurrency:    to,
		Amount:        amount,
		Timestamp:     time.Now(),
	})

	// возвращаем то, что сохранено в БД
	// При повторе это будут те же числа, что и в первый раз
	s.Logger.InfoContext(ctx, "exchange",
		"user_id", userID, "from", from, "to", to,
		"amount", amount, "received", op.ExchangedAmount, "rate", op.Rate,
	)
	return w, op.ExchangedAmount, op.Rate, nil
}
