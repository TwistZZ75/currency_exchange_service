package postgres_test

import (
	"context"
	"os"
	"testing"

	"gw-exchanger/internal/storages/postgres"

	"github.com/stretchr/testify/require"
)

func TestStorage_GetRates(t *testing.T) {
	dsn := getDSN(t)

	st, err := postgres.NewPool(context.Background(), dsn)
	require.NoError(t, err)
	t.Cleanup(st.Close)

	rates, err := st.GetMap(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, rates)
	require.Contains(t, rates, "USD")
	require.Contains(t, rates, "RUB")
	require.Contains(t, rates, "EUR")
}

func TestStorage_GetRate(t *testing.T) {
	dsn := getDSN(t)

	st, err := postgres.NewPool(context.Background(), dsn)
	require.NoError(t, err)
	t.Cleanup(st.Close)

	ctx := context.Background()

	// 1 USD → USD = 1
	r, err := st.GetRate(ctx, "USD", "USD")
	require.NoError(t, err)
	require.Equal(t, 1.0, r)

	// 1 USD → RUB = rate_USD
	usdToRub, err := st.GetRate(ctx, "USD", "RUB")
	require.NoError(t, err)
	require.Greater(t, usdToRub, 0.0)

	// 1 RUB → USD = 1 / rate_USD
	rubToUsd, err := st.GetRate(ctx, "RUB", "USD")
	require.NoError(t, err)
	require.InDelta(t, 1.0/usdToRub, rubToUsd, 0.001)

	// Неподдерживаемая пара
	_, err = st.GetRate(ctx, "BTC", "USD")
	require.Error(t, err)
}
func getDSN(t *testing.T) string {
	t.Helper()
	dsn := env("TEST_DSN", "")
	if dsn == "" {
		t.Skip("TEST_DSN is not set, skipping integration test")
	}
	return dsn
}

func env(key, def string) string {
	if v, ok := lookupEnv(key); ok {
		return v
	}
	return def
}

var lookupEnv = func(key string) (string, bool) {
	return os.LookupEnv(key)
}
