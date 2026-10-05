package storage

import (
	"context"

	"gw-notifier/internal/domain"
)

type Storage interface {
	// SaveEvent одиночный upsert для тестов
	SaveEvent(ctx context.Context, ev domain.TransferEvent) (bool, error)

	SaveBatch(ctx context.Context, events []domain.TransferEvent) (int, error)

	Ping(ctx context.Context) error
	Close(ctx context.Context) error
}
