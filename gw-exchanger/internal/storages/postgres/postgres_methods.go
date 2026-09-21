package postgres

import (
	"context"
	"errors"
	"fmt"
	"gw-exchanger/internal/storages"
	"gw-exchanger/pkg"
)

// GetRates возвращает курсы всех валют к рублю.
func (s *Storage) GetMap(ctx context.Context) (map[string]float64, error) {
	const q = `SELECT currency, rate_to_rub FROM rates`

	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query rates: %w", err)
	}
	defer rows.Close()

	rates := make(map[string]float64, 4)
	for rows.Next() {
		var (
			currency string
			rate     float64
		)
		if err := rows.Scan(&currency, &rate); err != nil {
			return nil, fmt.Errorf("scan rate: %w", err)
		}
		rates[currency] = rate
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows err: %w", err)
	}

	if len(rates) == 0 {
		return nil, storages.ErrNotFound
	}
	return rates, nil
}

// GetRate возвращает курс пары from→to
func (s *Storage) GetRate(ctx context.Context, from, to string) (float64, error) {
	from, err := pkg.NormalizeCurrency(from)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", storages.ErrUnsupportedPair, err)
	}
	to, err = pkg.NormalizeCurrency(to)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", storages.ErrUnsupportedPair, err)
	}

	if from == to {
		return 1.0, nil
	}

	const q = `
        SELECT currency, rate_to_rub
        FROM rates
        WHERE currency = ANY($1)
    `

	rows, err := s.pool.Query(ctx, q, []string{from, to})
	if err != nil {
		return 0, fmt.Errorf("query rate: %w", err)
	}
	defer rows.Close()

	rates := make(map[string]float64, 2)
	for rows.Next() {
		var (
			currency string
			rate     float64
		)
		if err := rows.Scan(&currency, &rate); err != nil {
			return 0, fmt.Errorf("scan rate: %w", err)
		}
		rates[currency] = rate
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("rows err: %w", err)
	}

	fromRate, ok := rates[from]
	if !ok {
		return 0, fmt.Errorf("%w: %s", storages.ErrNotFound, from)
	}
	toRate, ok := rates[to]
	if !ok {
		return 0, fmt.Errorf("%w: %s", storages.ErrNotFound, to)
	}
	if toRate == 0 {
		return 0, errors.New("rate_to_rub for target currency is zero")
	}

	return pkg.RoundRate(fromRate / toRate), nil
}
