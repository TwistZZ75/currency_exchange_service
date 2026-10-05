package service

import (
	"context"
	"encoding/json"
	"errors"
	"gw-analytics/internal/domain"
	"gw-analytics/internal/metrics"
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

		if err := ev.Validate(); err != nil {
			p.handleBadMessage(ctx, msg, err, "invalid event")
			continue
		}

		receivedAt := time.Now().UTC()
		ev.ReceivedAt = receivedAt
		ev.Topic = msg.Topic
		ev.Partition = msg.Partition
		ev.Offset = msg.Offset

		// status опционален: если не пришёл — считаем success.
		status := ev.Status
		if status == "" {
			status = "success"
		}

		latency := receivedAt.Sub(ev.Timestamp).Milliseconds()
		if latency < 0 {
			latency = 0
		}

		row := domain.ClickHouseRow{
			TransactionID: ev.TransactionID,
			UserID:        ev.UserID,
			Type:          ev.Type,
			Status:        status,
			FromCurrency:  ev.FromCurrency,
			ToCurrency:    ev.ToCurrency,
			Amount:        ev.Amount,
			EventTime:     ev.Timestamp,
			ReceivedAt:    receivedAt,
			LatencyMs:     latency,
			Topic:         msg.Topic,
			Partition:     int32(msg.Partition),
			Offset:        msg.Offset,
			// Version = unix ms; ReplacingMergeTree оставит последнюю запись.
			Version: uint64(receivedAt.UnixMilli()),
		}

		select {
		case out <- batchItem{msg: msg, row: row}:
		case <-ctx.Done():
			return nil
		}
	}
}

func (p *Processor) handleBadMessage(ctx context.Context, msg kafka.Message, reason error, kind string) {
	log := p.log.With(
		"topic", msg.Topic, "partition", msg.Partition, "offset", msg.Offset,
		"reason", kind, "error", reason.Error(),
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
