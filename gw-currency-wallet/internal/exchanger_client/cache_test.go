package exchanger_test

import (
	"context"
	"errors"
	exchanger "gw-currency-wallet/internal/exchanger_client"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type fakeClient struct {
	calls int32
	rates map[string]float64
	err   error
	delay time.Duration
}

func (f *fakeClient) GetMap(ctx context.Context) (map[string]float64, error) {
	atomic.AddInt32(&f.calls, 1)
	if f.delay > 0 {
		select {
		case <-time.After(f.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if f.err != nil {
		return nil, f.err
	}

	out := make(map[string]float64, len(f.rates))
	for k, v := range f.rates {
		out[k] = v
	}
	return out, nil
}

func TestCache_FreshHit(t *testing.T) {
	fc := &fakeClient{rates: map[string]float64{"USD": 90}}
	c := exchanger.NewCachedRatesWithFetcher(fc, time.Second)

	for i := 0; i < 5; i++ {
		_, err := c.Get(context.Background())
		require.NoError(t, err)
	}
	require.EqualValues(t, 1, atomic.LoadInt32(&fc.calls), "за 5 вызовов fetcher должен дёрнуться 1 раз")
}

func TestCache_Expires(t *testing.T) {
	fc := &fakeClient{rates: map[string]float64{"USD": 90}}
	c := exchanger.NewCachedRatesWithFetcher(fc, 30*time.Millisecond)

	_, err := c.Get(context.Background())
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	_, err = c.Get(context.Background())
	require.NoError(t, err)

	require.EqualValues(t, 2, atomic.LoadInt32(&fc.calls))
}

func TestCache_Invalidate(t *testing.T) {
	fc := &fakeClient{rates: map[string]float64{"USD": 90}}
	c := exchanger.NewCachedRatesWithFetcher(fc, time.Hour)

	_, _ = c.Get(context.Background())
	c.Invalidate()
	_, _ = c.Get(context.Background())

	require.EqualValues(t, 2, atomic.LoadInt32(&fc.calls))
}

func TestCache_Error_NotCached(t *testing.T) {
	fc := &fakeClient{err: errors.New("boom")}
	c := exchanger.NewCachedRatesWithFetcher(fc, time.Hour)

	_, err := c.Get(context.Background())
	require.Error(t, err)
	_, err = c.Get(context.Background())
	require.Error(t, err)

	require.EqualValues(t, 2, atomic.LoadInt32(&fc.calls), "ошибки не должны кэшироваться")
}

func TestCache_Concurrent_Singleflight(t *testing.T) {
	fc := &fakeClient{
		rates: map[string]float64{"USD": 90},
		delay: 50 * time.Millisecond,
	}
	c := exchanger.NewCachedRatesWithFetcher(fc, time.Second)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = c.Get(context.Background())
		}()
	}
	wg.Wait()

	require.LessOrEqual(t, atomic.LoadInt32(&fc.calls), int32(2),
		"одновременные запросы при холодном кэше не должны давать много вызовов")
}
