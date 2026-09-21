package storages

import "errors"

var (
	// ErrNotFound возвращается, если запись отсутствует.
	ErrNotFound = errors.New("storage: not found")

	// ErrUnsupportedPair возвращается для неподдерживаемой пары валют.
	ErrUnsupportedPair = errors.New("storage: unsupported currency pair")

	// ErrStorageUnavailable — общая ошибка недоступности хранилища.
	ErrStorageUnavailable = errors.New("storage: unavailable")
)
