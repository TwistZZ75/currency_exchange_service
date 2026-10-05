package processor

import (
	"context"
	"gw-notifier/internal/domain"
	"gw-notifier/internal/metrics"
	"gw-notifier/pkg/retry"
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
	events := make([]domain.TransferEvent, len(batch))
	for i, it := range batch {
		events[i] = it.ev
	}

	metrics.BatchSize.Observe(float64(len(batch)))

	var (
		inserted int
		attempts int
	)
	err := retry.Retry(ctx, p.retry, func(ctx context.Context) error {
		attempts++
		saveCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		n, err := p.storage.SaveBatch(saveCtx, events)
		if err != nil {
			return err
		}
		inserted = n
		return nil
	})
	metrics.RetryAttempts.Observe(float64(attempts - 1))
	metrics.MongoWriteDuration.Observe(time.Since(start).Seconds())
	metrics.ProcessingDuration.Observe(time.Since(start).Seconds())

	if err != nil {
		metrics.MongoErrors.Inc()
		metrics.EventsProcessed.WithLabelValues("failed").Add(float64(len(batch)))
		p.log.Error("batch write failed after retries, sending to dlq",
			"error", err, "count", len(batch), "attempts", attempts)

		for _, it := range batch {
			dlqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			dlqErr := p.dlq.Write(dlqCtx, it.msg, err)
			cancel()

			if dlqErr != nil {
				p.log.Error("dlq write failed; message will be redelivered",
					"error", dlqErr,
					"transaction_id", it.ev.TransactionID,
					"offset", it.msg.Offset)
				continue
			}
			metrics.DLQEvents.Inc()
			p.addToCommit(it.msg)
		}
		return
	}

	metrics.EventsProcessed.WithLabelValues("saved").Add(float64(inserted))
	metrics.EventsProcessed.WithLabelValues("duplicate").Add(float64(len(batch) - inserted))
	p.log.Info("batch saved", "count", len(batch), "inserted", inserted, "attempts", attempts)

	for _, it := range batch {
		p.addToCommit(it.msg)
	}
}
