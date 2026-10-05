package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"

	"gw-analytics/internal/domain"
)

type DLQWriter struct {
	writer *kafka.Writer
}

func NewDLQWriter(brokers []string, topic string) *DLQWriter {
	return &DLQWriter{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafka.Hash{},
			RequiredAcks: kafka.RequireOne,
			Async:        false,
		},
	}
}

// Write отправляет битое сообщение в DLQ, оборачивая его причиной ошибки.
func (d *DLQWriter) Write(ctx context.Context, msg kafka.Message, reason error) error {
	ev := domain.DLQEvent{
		Original:  json.RawMessage(msg.Value),
		Error:     reason.Error(),
		Topic:     msg.Topic,
		Partition: msg.Partition,
		Offset:    msg.Offset,
		FailedAt:  time.Now().UTC(),
	}
	data, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("marshal dlq event: %w", err)
	}
	return d.writer.WriteMessages(ctx, kafka.Message{
		Key:   msg.Key,
		Value: data,
		Headers: []kafka.Header{
			{Key: "x-error", Value: []byte(reason.Error())},
			{Key: "x-origin-topic", Value: []byte(msg.Topic)},
		},
	})
}

func (d *DLQWriter) Close() error {
	return d.writer.Close()
}
