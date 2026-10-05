package domain

import (
	"encoding/json"
	"errors"
	"time"
)

var ErrInvalidEvent = errors.New("invalid transfer event")

type TransferEvent struct {
	TransactionID string    `json:"transaction_id"`
	UserID        int64     `json:"user_id"`
	Type          string    `json:"type"`
	Status        string    `json:"status,omitempty"`
	FromCurrency  string    `json:"from_currency,omitempty"`
	ToCurrency    string    `json:"to_currency,omitempty"`
	Amount        float64   `json:"amount"`
	Timestamp     time.Time `json:"timestamp"`

	ReceivedAt time.Time `json:"-"`
	Topic      string    `json:"-"`
	Partition  int       `json:"-"`
	Offset     int64     `json:"-"`
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

type ClickHouseRow struct {
	TransactionID string
	UserID        int64
	Type          string
	Status        string
	FromCurrency  string
	ToCurrency    string
	Amount        float64
	EventTime     time.Time
	ReceivedAt    time.Time
	LatencyMs     int64
	Topic         string
	Partition     int32
	Offset        int64
	Version       uint64
}

type DLQEvent struct {
	Original  json.RawMessage `json:"original"`
	Error     string          `json:"error"`
	Topic     string          `json:"topic"`
	Partition int             `json:"partition"`
	Offset    int64           `json:"offset"`
	FailedAt  time.Time       `json:"failed_at"`
}
