package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	EventsProcessed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "gw_analytics_events_processed_total",
		Help: "Обработано событий по статусу (saved|failed).",
	}, []string{"status"})

	DLQEvents = promauto.NewCounter(prometheus.CounterOpts{
		Name: "gw_analytics_dlq_events_total",
		Help: "Сколько сообщений ушло в DLQ.",
	})

	ClickHouseErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "gw_analytics_clickhouse_errors_total",
		Help: "Сколько раз batch insert исчерпал ретраи.",
	})

	RetryAttempts = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "gw_analytics_retry_attempts",
		Help:    "Число retry до успеха.",
		Buckets: []float64{0, 1, 2, 3, 4, 5},
	})

	BatchSize = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "gw_analytics_batch_size",
		Help:    "Размер батча при вставке в ClickHouse.",
		Buckets: []float64{1, 10, 50, 100, 250, 500, 1000, 5000},
	})

	EndToEndLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "gw_analytics_end_to_end_latency_seconds",
		Help:    "Latency от event_time (создание в wallet) до записи в ClickHouse.",
		Buckets: []float64{.01, .05, .1, .25, .5, 1, 2.5, 5, 10, 30, 60},
	})

	ClickHouseWriteDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "gw_analytics_clickhouse_write_duration_seconds",
		Help:    "Длительность batch insert в ClickHouse (включая retry).",
		Buckets: prometheus.DefBuckets,
	})

	HTTPRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "gw_analytics_http_requests_total",
		Help: "HTTP-запросы по методу, пути и статусу.",
	}, []string{"method", "path", "status"})
)
