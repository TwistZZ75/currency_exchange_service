package storages

import "context"

type Storage interface {
	// GetRate - получить курс передаваемой валюты к рублю
	GetRate(ctx context.Context, from, to string) (float64, error)

	// GetMap - получить курс всех поддерживаемых валют к рублю
	GetMap(ctx context.Context) (map[string]float64, error)

	// Ping - проверить доступность хранилища
	Ping(ctx context.Context) error

	// Close - закрыть соединиение с хранилищем
	Close()
}
