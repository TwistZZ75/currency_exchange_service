package service

import (
	"context"

	"gw-currency-wallet/internal/domain"
)

type fakeRates struct {
	rates       map[string]float64
	err         error
	invalidated bool
}

func (f *fakeRates) Get(_ context.Context) (map[string]float64, error) {
	return f.rates, f.err
}
func (f *fakeRates) Invalidate() { f.invalidated = true }

type fakeProducer struct {
	events []domain.TransferEvent
	err    error
}

func (f *fakeProducer) SendIfLarge(_ context.Context, ev domain.TransferEvent) error {
	f.events = append(f.events, ev)
	return f.err
}
