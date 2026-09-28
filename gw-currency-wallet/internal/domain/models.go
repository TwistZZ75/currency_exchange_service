package domain

import "time"

type User struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Password  string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
}

type Wallet struct {
	UserID int64   `json:"-"`
	USD    float64 `json:"USD"`
	RUB    float64 `json:"RUB"`
	EUR    float64 `json:"EUR"`
}

type OperationType string

const (
	OpDeposit  OperationType = "deposit"
	OpWithdraw OperationType = "withdraw"
	OpExchange OperationType = "exchange"
)

type Operation struct {
	ID              int64
	TransactionID   string
	UserID          int64
	Type            OperationType
	FromCurrency    string
	ToCurrency      string
	Amount          float64
	ExchangedAmount float64
	Rate            float64
	CreatedAt       time.Time
}

type TransferEvent struct {
	TransactionID string    `json:"transaction_id"`
	UserID        int64     `json:"user_id"`
	Type          string    `json:"type"`
	FromCurrency  string    `json:"from_currency,omitempty"`
	ToCurrency    string    `json:"to_currency,omitempty"`
	Amount        float64   `json:"amount"`
	Timestamp     time.Time `json:"timestamp"`
}
