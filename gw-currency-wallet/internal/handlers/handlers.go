package handlers

import (
	"log/slog"

	"gw-currency-wallet/internal/service"
)

type Handlers struct {
	Svc    *service.Service
	Logger *slog.Logger
}

func NewHandler(svc *service.Service, log *slog.Logger) *Handlers {
	return &Handlers{Svc: svc, Logger: log}
}
