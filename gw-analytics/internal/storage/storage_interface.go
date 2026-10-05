package storage

import (
	"context"
	"time"

	"gw-analytics/internal/domain"
)

type Storage interface {
	InsertBatch(ctx context.Context, rows []domain.ClickHouseRow) error

	CountByTypeStatus(ctx context.Context, from, to time.Time, period string) ([]TypeStatusCount, error)
	LatencyStats(ctx context.Context, from, to time.Time, period string) ([]LatencyBucket, error)
	ErrorStats(ctx context.Context, from, to time.Time, period string) ([]ErrorBucket, error)

	Ping(ctx context.Context) error
	Close() error
}

type TypeStatusCount struct {
	Bucket time.Time `json:"bucket"`
	Type   string    `json:"type"`
	Status string    `json:"status"`
	Count  uint64    `json:"count"`
}

type LatencyBucket struct {
	Bucket time.Time `json:"bucket"`
	Type   string    `json:"type"`
	AvgMs  float64   `json:"avg_ms"`
	P95Ms  float64   `json:"p95_ms"`
	P99Ms  float64   `json:"p99_ms"`
	Count  uint64    `json:"count"`
}

type ErrorBucket struct {
	Bucket     time.Time `json:"bucket"`
	Type       string    `json:"type"`
	ErrorCount uint64    `json:"error_count"`
	TotalCount uint64    `json:"total_count"`
	ErrorRate  float64   `json:"error_rate"`
}
