package logger

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// LoggingInterceptor логирует каждый gRPC-вызов (метод, длительность, статус)
func LoggingInterceptor(log *slog.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		dur := time.Since(start)

		code := status.Code(err).String()
		attrs := []any{
			"grpc.method", info.FullMethod,
			"grpc.duration_ms", dur.Milliseconds(),
			"grpc.code", code,
		}
		if err != nil {
			attrs = append(attrs, "error", err.Error())
			log.ErrorContext(ctx, "grpc call failed", attrs...)
		} else {
			log.InfoContext(ctx, "grpc call ok", attrs...)
		}
		return resp, err
	}
}
