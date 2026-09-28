package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/segmentio/kafka-go"

	"gw-currency-wallet/internal/domain"
)

type Producer struct {
	writer     *kafka.Writer
	topic      string
	threshhold float64
}

func NewProducer(brokers []string, topic string, threshold float64) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafka.Hash{},
			RequiredAcks: kafka.RequireOne,
			Async:        false,
		},
		topic:      topic,
		threshhold: threshold,
	}
}

func (p *Producer) Close() error { return p.writer.Close() }

// SendIfLarge отправляет событие, если сумма больше или равна порогу
// Возвращает nil, если меньше
func (p *Producer) SendIfLarge(ctx context.Context, ev domain.TransferEvent) error {
	if ev.Amount < p.threshhold {
		return nil
	}
	data, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("marshal transfer event: %w", err)
	}
	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(ev.TransactionID),
		Value: data,
	})
}
