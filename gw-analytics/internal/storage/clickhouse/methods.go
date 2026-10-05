package clickhouse

import (
	"context"
	"fmt"

	"gw-analytics/internal/domain"
)

func (s *Storage) InsertBatch(ctx context.Context, rows []domain.ClickHouseRow) error {
	if len(rows) == 0 {
		return nil
	}

	batch, err := s.conn.PrepareBatch(ctx, "INSERT INTO events")
	if err != nil {
		return fmt.Errorf("prepare batch: %w", err)
	}

	for _, r := range rows {
		if err := batch.Append(
			r.TransactionID,
			r.UserID,
			r.Type,
			r.Status,
			r.FromCurrency,
			r.ToCurrency,
			r.Amount,
			r.EventTime,
			r.ReceivedAt,
			r.LatencyMs,
			r.Topic,
			r.Partition,
			r.Offset,
			r.Version,
		); err != nil {
			return fmt.Errorf("batch append: %w", err)
		}
	}

	if err := batch.Send(); err != nil {
		return fmt.Errorf("batch send: %w", err)
	}
	return nil
}
