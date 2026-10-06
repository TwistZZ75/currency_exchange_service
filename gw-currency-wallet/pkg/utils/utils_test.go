package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeCurrency_OK(t *testing.T) {
	for _, in := range []string{"USD", "usd", " Usd ", "  eur  "} {
		out, err := NormalizeCurrency(in)
		require.NoError(t, err)
		require.Contains(t, []string{"USD", "EUR", "RUB"}, out)
	}
}

func TestNormalizeCurrency_Unsupported(t *testing.T) {
	for _, in := range []string{"BTC", "ETH", "", "usd1", "юсд"} {
		_, err := NormalizeCurrency(in)
		require.Error(t, err, "expected error for %q", in)
	}
}

func TestRound2(t *testing.T) {
	cases := []struct {
		in   float64
		want float64
	}{
		{0.0, 0.0},
		{90.0, 90.0},
		{90.456, 90.46},
		{90.454, 90.45},
		{90.455, 90.46},
		{89.999, 90.0},
		{1.0 / 3.0, 0.33},
	}
	for _, c := range cases {
		got := Round2(c.in)
		require.InDelta(t, c.want, got, 0.00001, "in=%v", c.in)
	}
}
