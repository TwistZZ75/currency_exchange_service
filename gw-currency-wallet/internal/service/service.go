package service

import (
	"context"
	"log/slog"

	"gw-currency-wallet/internal/auth"
	"gw-currency-wallet/internal/domain"
	"gw-currency-wallet/internal/storages"
)

type RatesProvider interface {
	Get(ctx context.Context) (map[string]float64, error)
	Invalidate()
}

type EventPublisher interface {
	SendIfLarge(ctx context.Context, ev domain.TransferEvent) error
}

type Service struct {
	Storage        storages.Storage
	JWT            *auth.JWTManager
	Rates          RatesProvider
	Producer       EventPublisher
	Logger         *slog.Logger
	LargeThreshold float64
}
