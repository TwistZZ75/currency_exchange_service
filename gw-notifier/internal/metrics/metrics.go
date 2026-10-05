package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	EventsProcessed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "gw_notification_events_processed_total",
		Help: "Сколько событий обработано, по статусу (saved|duplicate|failed).",
	}, []string{"status"})

	DLQEvents = promauto.NewCounter(prometheus.CounterOpts{
		Name: "gw_notification_dlq_events_total",
		Help: "Сколько сообщений ушло в DLQ.",
	})

	MongoErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "gw_notification_mongo_errors_total",
		Help: "Сколько раз bulk write в Mongo исчерпал ретраи.",
	})

	RetryAttempts = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "gw_notification_retry_attempts",
		Help:    "Число попыток retry для успешной операции (0 = с первой попытки).",
		Buckets: []float64{0, 1, 2, 3, 4, 5},
	})

	BatchSize = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "gw_notification_batch_size",
		Help:    "Размер батча при записи в Mongo.",
		Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000},
	})

	MongoWriteDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "gw_notification_mongo_write_duration_seconds",
		Help:    "Длительность одного bulk write в Mongo (включая retry).",
		Buckets: prometheus.DefBuckets,
	})

	ProcessingDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "gw_notification_processing_duration_seconds",
		Help:    "Полное время обработки батча.",
		Buckets: prometheus.DefBuckets,
	})
)
