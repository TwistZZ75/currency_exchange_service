package consumer

import "github.com/segmentio/kafka-go"

func NewReader(brokers []string, topic, groupID string, queueSize int) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       1,
		MaxBytes:       10e6,
		MaxWait:        500 * 1e6,
		QueueCapacity:  queueSize,
		CommitInterval: 0,
		StartOffset:    kafka.LastOffset,
	})
}
