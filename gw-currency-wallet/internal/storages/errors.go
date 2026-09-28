package storages

import "errors"

var (
	ErrNotFound            = errors.New("storage: not found")
	ErrUserExists          = errors.New("storage: user already exists")
	ErrInsufficientFunds   = errors.New("storage: insufficient funds")
	ErrDuplicate           = errors.New("storage: duplicate")
	ErrUnavailable         = errors.New("storage: unavailable")
	ErrIdempotencyConflict = errors.New("transaction_id is already used with different parameters")
)
