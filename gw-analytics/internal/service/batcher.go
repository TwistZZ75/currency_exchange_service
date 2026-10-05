package service

import (
	"context"
	"gw-analytics/internal/domain"
	"gw-analytics/internal/metrics"
	"gw-analytics/pkg/retry"
	"time"
)

func (p *Processor) batchLoop(ctx context.Context, incoming <-chan batchItem) error {
	var batch []batchItem
	ticker := time.NewTicker(p.cfg.BatchTimeout)
	defer ticker.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}
		p.flushBatch(ctx, batch)
		batch = batch[:0]
	}

	for {
		select {
		case it, ok := <-incoming:
			if !ok {
				flush()
				return nil
			}
			batch = append(batch, it)
			if len(batch) >= p.cfg.BatchSize {
				flush()
			}

		case <-ticker.C:
			flush()

		case <-ctx.Done():
			for it := range incoming {
				batch = append(batch, it)
				if len(batch) >= p.cfg.BatchSize {
					flush()
				}
			}
			flush()
			return nil
		}
	}
}

func (p *Processor) flushBatch(ctx context.Context, batch []batchItem) {
	start := time.Now()
	rows := make([]domain.ClickHouseRow, len(batch))
	for i, it := range batch {
		rows[i] = it.row
	}

	metrics.BatchSize.Observe(float64(len(batch)))

	attempts := 0
	err := retry.Retry(ctx, p.retry, func(ctx context.Context) error {
		attempts++
		insertCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		return p.storage.InsertBatch(insertCtx, rows)
	})
	metrics.RetryAttempts.Observe(float64(attempts - 1))
	metrics.ClickHouseWriteDuration.Observe(time.Since(start).Seconds())

	if err != nil {
		metrics.ClickHouseErrors.Inc()
		metrics.EventsProcessed.WithLabelValues("failed").Add(float64(len(batch)))
		p.log.Error("batch insert failed after retries, sending to dlq",
			"error", err, "count", len(batch), "attempts", attempts)

		for _, it := range batch {
			dlqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			dlqErr := p.dlq.Write(dlqCtx, it.msg, err)
			cancel()
			if dlqErr != nil {
				p.log.Error("dlq write failed; message will be redelivered",
					"error", dlqErr, "transaction_id", it.row.TransactionID)
				continue
			}
			metrics.DLQEvents.Inc()
			p.addToCommit(it.msg)
		}
		return
	}

	for _, it := range batch {
		metrics.EndToEndLatency.Observe(time.Since(it.row.EventTime).Seconds())
	}

	metrics.EventsProcessed.WithLabelValues("saved").Add(float64(len(batch)))
	p.log.Info("batch saved", "count", len(batch), "attempts", attempts)

	for _, it := range batch {
		p.addToCommit(it.msg)
	}
}
