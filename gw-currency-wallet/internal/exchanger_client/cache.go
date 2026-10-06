package exchanger

import (
	"context"
	"sync"
	"time"
)

type RatesFetcher interface {
	GetMap(ctx context.Context) (map[string]float64, error)
}

// NewCachedRatesWithFetcher -конструктор для тестов
func NewCachedRatesWithFetcher(f RatesFetcher, ttl time.Duration) *CachedRates {
	return &CachedRates{fetcher: f, ttl: ttl}
}

type CachedRates struct {
	mu      sync.RWMutex
	rates   map[string]float64
	at      time.Time
	ttl     time.Duration
	fetcher RatesFetcher
}

func NewCachedRates(cli *Client, ttl time.Duration) *CachedRates {
	return &CachedRates{fetcher: cli, ttl: ttl}
}

// Get возвращает курсы: если кэш свежий из памяти, иначе идёт в gRPC
func (c *CachedRates) Get(ctx context.Context) (map[string]float64, error) {
	c.mu.RLock()
	if c.rates != nil && time.Since(c.at) < c.ttl {
		r := c.rates
		c.mu.RUnlock()
		return r, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.rates != nil && time.Since(c.at) < c.ttl {
		return c.rates, nil
	}

	rates, err := c.fetcher.GetMap(ctx)
	if err != nil {
		return nil, err
	}
	c.rates = rates
	c.at = time.Now()
	return rates, nil
}

// Invalidate — сбросить кэш (например, после обмена, чтобы следующий
// запрос шёл за свежим курсом)
func (c *CachedRates) Invalidate() {
	c.mu.Lock()
	c.rates = nil
	c.mu.Unlock()
}
