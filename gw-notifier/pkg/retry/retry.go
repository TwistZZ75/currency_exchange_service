package retry

import (
	"context"
	"fmt"
	"time"
)

type Config struct {
	MaxAttempts int
	Initial     time.Duration
	Max         time.Duration
}

func Retry(ctx context.Context, cfg Config, fn func(ctx context.Context) error) error {
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 1
	}
	if cfg.Initial <= 0 {
		cfg.Initial = 100 * time.Millisecond
	}
	if cfg.Max <= 0 {
		cfg.Max = 5 * time.Second
	}

	delay := cfg.Initial
	var lastErr error

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		lastErr = fn(ctx)
		if lastErr == nil {
			return nil
		}
		if attempt == cfg.MaxAttempts {
			break
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}

		delay *= 2
		if delay > cfg.Max {
			delay = cfg.Max
		}
	}
	return fmt.Errorf("retry exhausted after %d attempts: %w", cfg.MaxAttempts, lastErr)
}
