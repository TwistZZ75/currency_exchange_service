package domain

import (
	"encoding/json"
	"time"
)

// TransferEvent — формат события, приходящего из Kafka от gw-currency-wallet.
type TransferEvent struct {
	TransactionID string    `json:"transaction_id" bson:"transaction_id"`
	UserID        int64     `json:"user_id" bson:"user_id"`
	Type          string    `json:"type" bson:"type"`
	Status        string    `json:"status,omitempty" bson:"status,omitempty"`
	FromCurrency  string    `json:"from_currency,omitempty" bson:"from_currency,omitempty"`
	ToCurrency    string    `json:"to_currency,omitempty" bson:"to_currency,omitempty"`
	Amount        float64   `json:"amount" bson:"amount"`
	Timestamp     time.Time `json:"timestamp" bson:"timestamp"`

	// Служебные поля — заполняются consumer'ом при записи в Mongo.
	ReceivedAt time.Time `json:"-" bson:"received_at"`
	Topic      string    `json:"-" bson:"topic"`
	Partition  int       `json:"-" bson:"partition"`
	Offset     int64     `json:"-" bson:"offset"`
}

func (e TransferEvent) Validate() error {
	if e.TransactionID == "" {
		return ErrInvalidEvent
	}
	if e.UserID <= 0 {
		return ErrInvalidEvent
	}
	if e.Amount <= 0 {
		return ErrInvalidEvent
	}
	return nil
}

// DLQEvent это формат сообщения для dead-letter топика.
type DLQEvent struct {
	Original  json.RawMessage `json:"original"`
	Error     string          `json:"error"`
	Topic     string          `json:"topic"`
	Partition int             `json:"partition"`
	Offset    int64           `json:"offset"`
	FailedAt  time.Time       `json:"failed_at"`
}
