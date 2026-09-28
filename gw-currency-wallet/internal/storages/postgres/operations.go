package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"gw-currency-wallet/internal/domain"
	"gw-currency-wallet/internal/storages"
)

func (s *Storage) Deposit(
	ctx context.Context,
	txID string,
	userID int64,
	currency string,
	amount float64,
) (*domain.Wallet, error) {

	col, err := columnForCurrency(currency)
	if err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// Сначала пытаемся зарегистрировать операцию
	// Если INSERT произошёл — это новая операция
	// Если RETURNING ничего не вернул — transaction_id уже существует
	var operationID int64

	err = tx.QueryRow(ctx, `
		INSERT INTO operations (
			transaction_id,
			user_id,
			type,
			to_currency,
			amount
		)
		VALUES ($1, $2, 'deposit', $3, $4)
		ON CONFLICT (transaction_id) DO NOTHING
		RETURNING id
	`,
		txID,
		userID,
		currency,
		amount,
	).Scan(&operationID)

	if errors.Is(err, pgx.ErrNoRows) {
		operation, err := getOperationTx(ctx, tx, txID)
		if err != nil {
			return nil, err
		}

		// Проверяем, что повторный запрос действительно
		// является тем же самым запросом
		if !sameDepositOperation(
			operation,
			userID,
			currency,
			amount,
		) {
			return nil, storages.ErrIdempotencyConflict
		}

		wallet, err := getWalletTx(ctx, tx, userID)
		if err != nil {
			return nil, err
		}

		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit idempotent deposit: %w", err)
		}

		return wallet, nil
	}

	if err != nil {
		return nil, fmt.Errorf("insert deposit operation: %w", err)
	}

	query := fmt.Sprintf(`
		UPDATE wallets
		SET %s = %s + $1,
			updated_at = NOW()
		WHERE user_id = $2
	`, col, col)

	tag, err := tx.Exec(ctx, query, amount, userID)
	if err != nil {
		return nil, fmt.Errorf("update deposit balance: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return nil, storages.ErrNotFound
	}

	wallet, err := getWalletTx(ctx, tx, userID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit deposit: %w", err)
	}

	return wallet, nil
}

func (s *Storage) Withdraw(
	ctx context.Context,
	txID string,
	userID int64,
	currency string,
	amount float64,
) (*domain.Wallet, error) {

	col, err := columnForCurrency(currency)
	if err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var operationID int64

	err = tx.QueryRow(ctx, `
		INSERT INTO operations (
			transaction_id,
			user_id,
			type,
			from_currency,
			amount
		)
		VALUES ($1, $2, 'withdraw', $3, $4)
		ON CONFLICT (transaction_id) DO NOTHING
		RETURNING id
	`,
		txID,
		userID,
		currency,
		amount,
	).Scan(&operationID)

	if errors.Is(err, pgx.ErrNoRows) {
		// если transaction_id уже существует
		operation, err := getOperationTx(ctx, tx, txID)
		if err != nil {
			return nil, err
		}

		// Повторный запрос должен полностью совпадать с исходной операцией
		if !sameWithdrawOperation(
			operation,
			userID,
			currency,
			amount,
		) {
			return nil, storages.ErrIdempotencyConflict
		}

		wallet, err := getWalletTx(ctx, tx, userID)
		if err != nil {
			return nil, err
		}

		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit idempotent withdraw: %w", err)
		}

		return wallet, nil
	}

	if err != nil {
		return nil, fmt.Errorf("insert withdraw operation: %w", err)
	}

	// Блокируем кошелёк
	var balance float64

	lockQuery := fmt.Sprintf(`
		SELECT %s
		FROM wallets
		WHERE user_id = $1
		FOR UPDATE
	`, col)

	err = tx.QueryRow(
		ctx,
		lockQuery,
		userID,
	).Scan(&balance)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storages.ErrNotFound
		}

		return nil, fmt.Errorf("lock wallet balance: %w", err)
	}

	if balance < amount {
		return nil, storages.ErrInsufficientFunds
	}

	updateQuery := fmt.Sprintf(`
		UPDATE wallets
		SET %s = %s - $1,
			updated_at = NOW()
		WHERE user_id = $2
	`, col, col)

	_, err = tx.Exec(
		ctx,
		updateQuery,
		amount,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("update withdraw balance: %w", err)
	}

	wallet, err := getWalletTx(ctx, tx, userID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit withdraw: %w", err)
	}

	return wallet, nil
}

func (s *Storage) Exchange(
	ctx context.Context,
	txID string,
	userID int64,
	from string,
	to string,
	amount float64,
	rate float64,
	received float64,
) (*domain.Wallet, *domain.Operation, error) {

	fromCol, err := columnForCurrency(from)
	if err != nil {
		return nil, nil, err
	}

	toCol, err := columnForCurrency(to)
	if err != nil {
		return nil, nil, err
	}

	if fromCol == toCol {
		return nil, nil, fmt.Errorf(
			"source and destination currencies must differ",
		)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var operationID int64

	err = tx.QueryRow(ctx, `
		INSERT INTO operations (
			transaction_id,
			user_id,
			type,
			from_currency,
			to_currency,
			amount,
			exchanged_amount,
			rate
		)
		VALUES (
			$1,
			$2,
			'exchange',
			$3,
			$4,
			$5,
			$6,
			$7
		)
		ON CONFLICT (transaction_id) DO NOTHING
		RETURNING id
	`,
		txID,
		userID,
		from,
		to,
		amount,
		received,
		rate,
	).Scan(&operationID)

	if errors.Is(err, pgx.ErrNoRows) {
		// transaction_id уже существует.
		operation, err := getOperationTx(ctx, tx, txID)
		if err != nil {
			return nil, nil, err
		}

		// Проверяем абсолютно все параметры исходной операции
		if !sameExchangeOperation(
			operation,
			userID,
			from,
			to,
			amount,
			rate,
			received,
		) {
			return nil, nil, storages.ErrIdempotencyConflict
		}

		wallet, err := getWalletTx(ctx, tx, userID)
		if err != nil {
			return nil, nil, err
		}

		if err := tx.Commit(ctx); err != nil {
			return nil, nil, fmt.Errorf(
				"commit idempotent exchange: %w",
				err,
			)
		}

		return wallet, operation, nil
	}

	if err != nil {
		return nil, nil, fmt.Errorf(
			"insert exchange operation: %w",
			err,
		)
	}

	var balance float64

	lockQuery := fmt.Sprintf(`
		SELECT %s
		FROM wallets
		WHERE user_id = $1
		FOR UPDATE
	`, fromCol)

	err = tx.QueryRow(
		ctx,
		lockQuery,
		userID,
	).Scan(&balance)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, storages.ErrNotFound
		}

		return nil, nil, fmt.Errorf(
			"lock exchange balance: %w",
			err,
		)
	}

	if balance < amount {
		return nil, nil, storages.ErrInsufficientFunds
	}

	// Списываем одну валюту и зачисляем другую
	// в рамках одной транзакции
	updateQuery := fmt.Sprintf(`
		UPDATE wallets
		SET
			%s = %s - $1,
			%s = %s + $2,
			updated_at = NOW()
		WHERE user_id = $3
	`,
		fromCol,
		fromCol,
		toCol,
		toCol,
	)

	_, err = tx.Exec(
		ctx,
		updateQuery,
		amount,
		received,
		userID,
	)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"update exchange balances: %w",
			err,
		)
	}

	wallet, err := getWalletTx(ctx, tx, userID)
	if err != nil {
		return nil, nil, err
	}

	operation := &domain.Operation{
		ID:              operationID,
		TransactionID:   txID,
		UserID:          userID,
		Type:            domain.OpExchange,
		FromCurrency:    from,
		ToCurrency:      to,
		Amount:          amount,
		ExchangedAmount: received,
		Rate:            rate,
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf(
			"commit exchange: %w",
			err,
		)
	}

	return wallet, operation, nil
}

func sameDepositOperation(
	op *domain.Operation,
	userID int64,
	currency string,
	amount float64,
) bool {
	return op.UserID == userID &&
		op.Type == domain.OpDeposit &&
		op.ToCurrency == currency &&
		op.Amount == amount
}

func sameWithdrawOperation(
	op *domain.Operation,
	userID int64,
	currency string,
	amount float64,
) bool {
	return op.UserID == userID &&
		op.Type == domain.OpWithdraw &&
		op.FromCurrency == currency &&
		op.Amount == amount
}

func sameExchangeOperation(
	op *domain.Operation,
	userID int64,
	from string,
	to string,
	amount float64,
	rate float64,
	received float64,
) bool {
	return op.UserID == userID &&
		op.Type == domain.OpExchange &&
		op.FromCurrency == from &&
		op.ToCurrency == to &&
		op.Amount == amount &&
		op.Rate == rate &&
		op.ExchangedAmount == received
}

// чтение операции по transaction_id внутри транзакции
func getOperationTx(ctx context.Context, tx pgx.Tx, txID string) (*domain.Operation, error) {
	const query = `
        SELECT id, transaction_id, user_id, type,
               COALESCE(from_currency, ''), COALESCE(to_currency, ''),
               amount, COALESCE(exchanged_amount, 0), COALESCE(rate, 0), created_at
        FROM operations WHERE transaction_id = $1
    `
	var operation domain.Operation
	err := tx.QueryRow(ctx, query, txID).Scan(
		&operation.ID,
		&operation.TransactionID,
		&operation.UserID,
		&operation.Type,
		&operation.FromCurrency,
		&operation.ToCurrency,
		&operation.Amount,
		&operation.ExchangedAmount,
		&operation.Rate,
		&operation.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storages.ErrNotFound
		}
		return nil, fmt.Errorf("select operation in tx: %w", err)
	}
	return &operation, nil
}

// чтение кошелька внутри транзакции
func getWalletTx(ctx context.Context, tx pgx.Tx, userID int64) (*domain.Wallet, error) {
	const query = `SELECT user_id, usd, rub, eur FROM wallets WHERE user_id = $1`
	var wallet domain.Wallet
	if err := tx.QueryRow(ctx, query, userID).Scan(
		&wallet.UserID,
		&wallet.USD,
		&wallet.RUB,
		&wallet.EUR,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storages.ErrNotFound
		}
		return nil, fmt.Errorf("select wallet in tx: %w", err)
	}
	return &wallet, nil
}
