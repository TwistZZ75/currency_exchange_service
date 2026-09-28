package service

import (
	"log/slog"

	"gw-currency-wallet/internal/auth"
	exchanger "gw-currency-wallet/internal/exchanger_client"
	"gw-currency-wallet/internal/kafka"
	"gw-currency-wallet/internal/storages"
)

type Service struct {
	Storage        storages.Storage
	JWT            *auth.JWTManager
	Rates          *exchanger.CachedRates
	Producer       *kafka.Producer
	Logger         *slog.Logger
	LargeThreshold float64
}
