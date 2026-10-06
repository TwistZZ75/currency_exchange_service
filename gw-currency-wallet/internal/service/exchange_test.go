package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gw-currency-wallet/internal/domain"
	"gw-currency-wallet/internal/storages"
	"gw-currency-wallet/internal/storages/mocks"
)

func TestGetRates_OK(t *testing.T) {
	rates := &fakeRates{rates: map[string]float64{"USD": 90, "EUR": 100, "RUB": 1}}
	svc := &Service{Rates: rates, Logger: newLogger()}

	got, err := svc.GetRates(context.Background())
	require.NoError(t, err)
	require.Equal(t, rates.rates, got)
}

func TestGetRates_Error(t *testing.T) {
	rates := &fakeRates{err: errors.New("boom")}
	svc := &Service{Rates: rates, Logger: newLogger()}

	_, err := svc.GetRates(context.Background())
	require.Error(t, err)
}

func TestExchange_OK(t *testing.T) {
	st := &mocks.Storage{}
	st.On("Exchange",
		mock.Anything, mock.Anything,
		int64(1), "USD", "EUR",
		float64(100), float64(0.9), float64(90),
	).Return(
		&domain.Wallet{UserID: 1, USD: 0, EUR: 90},
		&domain.Operation{
			TransactionID:   "t",
			Type:            domain.OpExchange,
			FromCurrency:    "USD",
			ToCurrency:      "EUR",
			Amount:          100,
			ExchangedAmount: 90,
			Rate:            0.9,
		},
		nil,
	).Once()

	rates := &fakeRates{rates: map[string]float64{"USD": 90, "EUR": 100, "RUB": 1}}
	svc := &Service{Storage: st, Rates: rates, Logger: newLogger()}

	w, received, rate, err := svc.Exchange(context.Background(), 1, "USD", "EUR", 100, "")
	require.NoError(t, err)
	require.InDelta(t, 0.9, rate, 0.0001)
	require.InDelta(t, 90, received, 0.01)
	require.EqualValues(t, 90, w.EUR)
	require.True(t, rates.invalidated, "кэш должен сброситься после обмена")
}

func TestExchange_SameCurrency(t *testing.T) {
	svc := &Service{
		Storage: &mocks.Storage{},
		Rates:   &fakeRates{rates: map[string]float64{"USD": 90}},
		Logger:  newLogger(),
	}
	_, _, _, err := svc.Exchange(context.Background(), 1, "USD", "USD", 100, "")
	require.ErrorIs(t, err, ErrInvalidCurrencyPair)
}

func TestExchange_InvalidCurrency(t *testing.T) {
	svc := &Service{
		Storage: &mocks.Storage{},
		Rates:   &fakeRates{rates: map[string]float64{"USD": 90}},
		Logger:  newLogger(),
	}
	_, _, _, err := svc.Exchange(context.Background(), 1, "BTC", "USD", 100, "")
	require.ErrorIs(t, err, ErrInvalidCurrency)
}

func TestExchange_InvalidAmount(t *testing.T) {
	svc := &Service{
		Storage: &mocks.Storage{},
		Rates:   &fakeRates{rates: map[string]float64{"USD": 90}},
		Logger:  newLogger(),
	}
	_, _, _, err := svc.Exchange(context.Background(), 1, "USD", "EUR", 0, "")
	require.ErrorIs(t, err, ErrInvalidAmount)
}

func TestExchange_InsufficientFunds(t *testing.T) {
	st := &mocks.Storage{}
	st.On("Exchange", mock.Anything, mock.Anything, int64(1), "USD", "EUR",
		mock.Anything, mock.Anything, mock.Anything).
		Return(nil, nil, storages.ErrInsufficientFunds).Once()

	rates := &fakeRates{rates: map[string]float64{"USD": 90, "EUR": 100, "RUB": 1}}
	svc := &Service{Storage: st, Rates: rates, Logger: newLogger()}

	_, _, _, err := svc.Exchange(context.Background(), 1, "USD", "EUR", 100, "")
	require.ErrorIs(t, err, ErrInsufficientFunds)
}

func TestExchange_Retry_ReturnsOriginalRate(t *testing.T) {
	// имитируем повторный вызов: БД возвращает сохранённую операцию
	// с rate=0.9, даже если сервис посчитал бы другой (0.8).
	st := &mocks.Storage{}
	st.On("Exchange", mock.Anything, mock.Anything, int64(1), "USD", "EUR",
		float64(100), mock.Anything, mock.Anything,
	).Return(
		&domain.Wallet{UserID: 1, USD: 0, EUR: 90},
		&domain.Operation{
			TransactionID:   "t",
			ExchangedAmount: 90,
			Rate:            0.9,
		},
		nil,
	).Once()

	rates := &fakeRates{rates: map[string]float64{"USD": 80, "EUR": 100, "RUB": 1}}
	svc := &Service{Storage: st, Rates: rates, Logger: newLogger()}

	_, received, rate, err := svc.Exchange(context.Background(), 1, "USD", "EUR", 100, "")
	require.NoError(t, err)
	require.InDelta(t, 0.9, rate, 0.0001)
	require.InDelta(t, 90, received, 0.01)
}
