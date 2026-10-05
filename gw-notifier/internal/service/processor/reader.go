package processor

import (
	"context"
	"encoding/json"
	"errors"
	"gw-notifier/internal/domain"
	"gw-notifier/internal/metrics"
	"time"

	"github.com/segmentio/kafka-go"
)

func (p *Processor) readLoop(ctx context.Context, out chan<- batchItem) error {
	for {
		msg, err := p.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil
			}
			return err
		}

		var ev domain.TransferEvent
		if err := json.Unmarshal(msg.Value, &ev); err != nil {
			p.handleBadMessage(ctx, msg, err, "unmarshal failed")
			continue
		}

		ev.ReceivedAt = time.Now().UTC()
		ev.Topic = msg.Topic
		ev.Partition = msg.Partition
		ev.Offset = msg.Offset

		if err := ev.Validate(); err != nil {
			p.handleBadMessage(ctx, msg, err, "invalid event")
			continue
		}

		select {
		case out <- batchItem{msg: msg, ev: ev}:
		case <-ctx.Done():
			return nil
		}
	}
}

// handleBadMessage шлёт сообщение в DLQ. При успехе — коммитит.
func (p *Processor) handleBadMessage(ctx context.Context, msg kafka.Message, reason error, kind string) {
	log := p.log.With(
		"topic", msg.Topic,
		"partition", msg.Partition,
		"offset", msg.Offset,
		"reason", kind,
		"error", reason.Error(),
	)

	dlqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := p.dlq.Write(dlqCtx, msg, reason); err != nil {
		log.Error("dlq write failed; message will be redelivered", "dlq_error", err)
		return
	}

	metrics.DLQEvents.Inc()
	log.Warn("message sent to dlq")
	p.addToCommit(msg)
}
