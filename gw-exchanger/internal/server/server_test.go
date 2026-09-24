package server_test

import (
	"context"
	"errors"
	"gw-exchanger/internal/storages"
	"io"
	"log/slog"
	pb "proto_exchange/exchange"
	"testing"

	"gw-exchanger/internal/server"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockStorage struct {
	rates    map[string]float64
	rateErr  error
	ratesErr error
	pingErr  error
}

func (m *mockStorage) GetMap(_ context.Context) (map[string]float64, error) {
	return m.rates, m.ratesErr
}

func (m *mockStorage) GetRate(_ context.Context, from, to string) (float64, error) {
	return 0, m.rateErr
}

func (m *mockStorage) Ping(_ context.Context) error { return m.pingErr }
func (m *mockStorage) Close()                       {}

func newLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// rateMock отдельный мок только для GetRate.
type rateMock struct{ rate float64 }

func (m *rateMock) GetMap(context.Context) (map[string]float64, error) { return nil, nil }
func (m *rateMock) GetRate(context.Context, string, string) (float64, error) {
	return m.rate, nil
}
func (m *rateMock) Ping(context.Context) error { return nil }
func (m *rateMock) Close()                     {}

func TestServer_GetExchangeRates_OK(t *testing.T) {
	st := &mockStorage{rates: map[string]float64{"USD": 90, "EUR": 100, "RUB": 1}}
	srv := server.NewServer(st, newLogger())

	resp, err := srv.GetExchangeMap(context.Background(), &pb.Empty{})
	require.NoError(t, err)
	require.Len(t, resp.Rates, 3)
	require.Equal(t, float32(90), resp.Rates["USD"])
}

func TestServer_GetExchangeRates_NotFound(t *testing.T) {
	st := &mockStorage{ratesErr: storages.ErrNotFound}
	srv := server.NewServer(st, newLogger())

	_, err := srv.GetExchangeMap(context.Background(), &pb.Empty{})
	require.Error(t, err)
	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestServer_GetExchangeRateForCurrency_OK(t *testing.T) {
	st := &mockStorage{}
	srv := server.NewServer(st, newLogger())

	st.rateErr = nil
	m := &rateMock{rate: 1.11}
	srv = server.NewServer(m, newLogger())

	resp, err := srv.GetExchangeRateForCurrency(context.Background(), &pb.CurrencyRequest{
		FromCurrency: "USD",
		ToCurrency:   "EUR",
	})
	require.NoError(t, err)
	require.Equal(t, "USD", resp.FromCurrency)
	require.Equal(t, "EUR", resp.ToCurrency)
	require.InDelta(t, 1.11, float64(resp.Rate), 0.0001)
}

func TestServer_GetExchangeRateForCurrency_InvalidArg(t *testing.T) {
	srv := server.NewServer(&mockStorage{}, newLogger())

	_, err := srv.GetExchangeRateForCurrency(context.Background(), &pb.CurrencyRequest{})
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}
func TestServer_GetExchangeRateForCurrency_NotFound(t *testing.T) {
	srv := server.NewServer(&mockStorage{rateErr: storages.ErrNotFound}, newLogger())

	_, err := srv.GetExchangeRateForCurrency(context.Background(), &pb.CurrencyRequest{
		FromCurrency: "USD",
		ToCurrency:   "EUR",
	})
	require.Error(t, err)
	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestMapError_Internal(t *testing.T) {
	srv := server.NewServer(&mockStorage{ratesErr: errors.New("boom")}, newLogger())
	_, err := srv.GetExchangeMap(context.Background(), &pb.Empty{})
	require.Error(t, err)
	require.Equal(t, codes.Internal, status.Code(err))
}
