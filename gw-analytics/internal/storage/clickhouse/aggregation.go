package clickhouse

import (
	"context"
	"fmt"
	"time"

	"gw-analytics/internal/storage"
)

func periodToInterval(period string) (string, error) {
	switch period {
	case "1m":
		return "1 MINUTE", nil
	case "5m":
		return "5 MINUTE", nil
	case "1h":
		return "1 HOUR", nil
	case "1d":
		return "1 DAY", nil
	case "1w":
		return "1 WEEK", nil
	}
	return "", fmt.Errorf("unsupported period: %s", period)
}

func (s *Storage) CountByTypeStatus(
	ctx context.Context,
	from, to time.Time,
	period string,
) ([]storage.TypeStatusCount, error) {
	interval, err := periodToInterval(period)
	if err != nil {
		return nil, err
	}

	q := fmt.Sprintf(`
        SELECT
            toStartOfInterval(event_time, INTERVAL %s) AS bucket,
            type,
            status,
            count() AS cnt
        FROM events FINAL
        WHERE event_time >= ? AND event_time < ?
        GROUP BY bucket, type, status
        ORDER BY bucket DESC, type, status
    `, interval)

	rows, err := s.conn.Query(ctx, q, from, to)
	if err != nil {
		return nil, fmt.Errorf("query count: %w", err)
	}
	defer rows.Close()

	out := []storage.TypeStatusCount{}
	for rows.Next() {
		var r storage.TypeStatusCount
		if err := rows.Scan(&r.Bucket, &r.Type, &r.Status, &r.Count); err != nil {
			return nil, fmt.Errorf("scan count: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Storage) LatencyStats(
	ctx context.Context,
	from, to time.Time,
	period string,
) ([]storage.LatencyBucket, error) {
	interval, err := periodToInterval(period)
	if err != nil {
		return nil, err
	}

	q := fmt.Sprintf(`
        SELECT
            toStartOfInterval(event_time, INTERVAL %s) AS bucket,
            type,
            avg(latency_ms)              AS avg_ms,
            quantile(0.95)(latency_ms)   AS p95_ms,
            quantile(0.99)(latency_ms)   AS p99_ms,
            count()                      AS cnt
        FROM events FINAL
        WHERE event_time >= ? AND event_time < ?
        GROUP BY bucket, type
        ORDER BY bucket DESC, type
    `, interval)

	rows, err := s.conn.Query(ctx, q, from, to)
	if err != nil {
		return nil, fmt.Errorf("query latency: %w", err)
	}
	defer rows.Close()

	out := []storage.LatencyBucket{}
	for rows.Next() {
		var r storage.LatencyBucket
		if err := rows.Scan(&r.Bucket, &r.Type, &r.AvgMs, &r.P95Ms, &r.P99Ms, &r.Count); err != nil {
			return nil, fmt.Errorf("scan latency: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Storage) ErrorStats(
	ctx context.Context,
	from, to time.Time,
	period string,
) ([]storage.ErrorBucket, error) {
	interval, err := periodToInterval(period)
	if err != nil {
		return nil, err
	}

	q := fmt.Sprintf(`
        SELECT
            toStartOfInterval(event_time, INTERVAL %s) AS bucket,
            type,
            countIf(status = 'error') AS error_count,
            count()                   AS total_count,
            if(count() = 0, 0, countIf(status = 'error') / count()) AS error_rate
        FROM events FINAL
        WHERE event_time >= ? AND event_time < ?
        GROUP BY bucket, type
        ORDER BY bucket DESC, type
    `, interval)

	rows, err := s.conn.Query(ctx, q, from, to)
	if err != nil {
		return nil, fmt.Errorf("query errors: %w", err)
	}
	defer rows.Close()

	out := []storage.ErrorBucket{}
	for rows.Next() {
		var r storage.ErrorBucket
		if err := rows.Scan(&r.Bucket, &r.Type, &r.ErrorCount, &r.TotalCount, &r.ErrorRate); err != nil {
			return nil, fmt.Errorf("scan errors: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
