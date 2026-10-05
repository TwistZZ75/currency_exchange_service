package clickhouse

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
)

type Storage struct {
	conn clickhouse.Conn
}

func NewConn(ctx context.Context, addr, db, user, password string, timeout time.Duration) (*Storage, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Database: db,
			Username: user,
			Password: password,
		},
		DialTimeout:     timeout,
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
	})
	if err != nil {
		return nil, fmt.Errorf("clickhouse open: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if err := conn.Ping(pingCtx); err != nil {
		return nil, fmt.Errorf("clickhouse ping: %w", err)
	}
	return &Storage{conn: conn}, nil
}

func (s *Storage) Ping(ctx context.Context) error {
	return s.conn.Ping(ctx)
}

func (s *Storage) Close() error {
	return s.conn.Close()
}
