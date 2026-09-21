package server

import (
	"context"
	"errors"
	"gw-exchanger/internal/storages"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "proto_exchange/exchange"
)

type Server struct {
	pb.UnimplementedExchangeServiceServer

	storage storages.Storage
	logger  *slog.Logger
}

func NewServer(storage storages.Storage, logger *slog.Logger) *Server {
	return &Server{
		storage: storage,
		logger:  logger,
	}
}

func (s *Server) GetExchangeRates(ctx context.Context, _ *pb.Empty) (*pb.ExchangeMapResponse, error) {
	rates, err := s.storage.GetMap(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "get exchange rates failed", "error", err)
		return nil, mapError(err)
	}

	out := make(map[string]float32, len(rates))
	for k, v := range rates {
		out[k] = float32(v)
	}

	s.logger.InfoContext(ctx, "get exchange rates ok", "count", len(out))
	return &pb.ExchangeMapResponse{Rates: out}, nil
}

// mapError маппит ошибки хранилища в gRPC-статусы
func mapError(err error) error {
	switch {
	case errors.Is(err, storages.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, storages.ErrUnsupportedPair):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, storages.ErrStorageUnavailable):
		return status.Error(codes.Unavailable, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}

func (s *Server) GetExchangeRateForCurrency(
	ctx context.Context,
	req *pb.CurrencyRequest,
) (*pb.ExchangeRateResponse, error) {
	if req.GetFromCurrency() == "" || req.GetToCurrency() == "" {
		return nil, status.Error(codes.InvalidArgument, "from_currency and to_currency are required")
	}

	rate, err := s.storage.GetRate(ctx, req.GetFromCurrency(), req.GetToCurrency())
	if err != nil {
		s.logger.ErrorContext(ctx, "get exchange rate failed",
			"from", req.GetFromCurrency(),
			"to", req.GetToCurrency(),
			"error", err,
		)
		return nil, mapError(err)
	}

	s.logger.InfoContext(ctx, "get exchange rate ok",
		"from", req.GetFromCurrency(),
		"to", req.GetToCurrency(),
		"rate", rate,
	)

	return &pb.ExchangeRateResponse{
		FromCurrency: req.GetFromCurrency(),
		ToCurrency:   req.GetToCurrency(),
		Rate:         float32(rate),
	}, nil
}
