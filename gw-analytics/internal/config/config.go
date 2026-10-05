package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	KafkaBrokers  []string
	KafkaTopic    string
	KafkaDLQTopic string
	KafkaGroupID  string

	ClickHouseAddr     string
	ClickHouseDB       string
	ClickHouseUser     string
	ClickHousePassword string
	ClickHouseTimeout  time.Duration

	BatchSize    int
	BatchTimeout time.Duration
	QueueSize    int

	RetryMaxAttempts int
	RetryInitial     time.Duration
	RetryMax         time.Duration

	HTTPPort         string
	HTTPReadTimeout  time.Duration
	HTTPWriteTimeout time.Duration

	MetricsEnabled bool
	MetricsPort    string

	LogLevel        slog.Level
	ShutdownTimeout time.Duration
}

func Load(path string) (*Config, error) {
	if path != "" {
		if _, err := os.Stat(path); err == nil {
			if err := godotenv.Load(path); err != nil {
				return nil, fmt.Errorf("load %q: %w", path, err)
			}
		}
	}

	cfg := &Config{
		KafkaTopic:         os.Getenv("KAFKA_TOPIC"),
		KafkaDLQTopic:      os.Getenv("KAFKA_DLQ_TOPIC"),
		KafkaGroupID:       os.Getenv("KAFKA_GROUP_ID"),
		ClickHouseAddr:     os.Getenv("CLICKHOUSE_ADDR"),
		ClickHouseDB:       os.Getenv("CLICKHOUSE_DB"),
		ClickHouseUser:     os.Getenv("CLICKHOUSE_USER"),
		ClickHousePassword: os.Getenv("CLICKHOUSE_PASSWORD"),
		HTTPPort:           os.Getenv("HTTP_PORT"),
		MetricsPort:        os.Getenv("METRICS_PORT"),
	}

	if raw := os.Getenv("KAFKA_BROKERS"); raw != "" {
		for _, b := range strings.Split(raw, ",") {
			if b = strings.TrimSpace(b); b != "" {
				cfg.KafkaBrokers = append(cfg.KafkaBrokers, b)
			}
		}
	}

	var err error
	for _, p := range []struct {
		key string
		dst *time.Duration
	}{
		{"CLICKHOUSE_TIMEOUT", &cfg.ClickHouseTimeout},
		{"BATCH_TIMEOUT", &cfg.BatchTimeout},
		{"RETRY_INITIAL", &cfg.RetryInitial},
		{"RETRY_MAX", &cfg.RetryMax},
		{"HTTP_READ_TIMEOUT", &cfg.HTTPReadTimeout},
		{"HTTP_WRITE_TIMEOUT", &cfg.HTTPWriteTimeout},
		{"SHUTDOWN_TIMEOUT", &cfg.ShutdownTimeout},
	} {
		if *p.dst, err = parseDuration(p.key, os.Getenv(p.key)); err != nil {
			return nil, err
		}
	}

	if cfg.BatchSize, err = parseInt("BATCH_SIZE", os.Getenv("BATCH_SIZE")); err != nil {
		return nil, err
	}
	if cfg.QueueSize, err = parseInt("QUEUE_SIZE", os.Getenv("QUEUE_SIZE")); err != nil {
		return nil, err
	}
	if cfg.RetryMaxAttempts, err = parseInt("RETRY_MAX_ATTEMPTS", os.Getenv("RETRY_MAX_ATTEMPTS")); err != nil {
		return nil, err
	}
	if cfg.MetricsEnabled, err = parseBool("METRICS_ENABLED", os.Getenv("METRICS_ENABLED")); err != nil {
		return nil, err
	}
	if cfg.LogLevel, err = parseLogLevel(os.Getenv("LOG_LEVEL")); err != nil {
		return nil, err
	}

	return cfg, cfg.validate()
}

func (c *Config) validate() error {
	missing := []string{}
	for key, val := range map[string]string{
		"KAFKA_TOPIC":        c.KafkaTopic,
		"KAFKA_DLQ_TOPIC":    c.KafkaDLQTopic,
		"KAFKA_GROUP_ID":     c.KafkaGroupID,
		"CLICKHOUSE_ADDR":    c.ClickHouseAddr,
		"CLICKHOUSE_DB":      c.ClickHouseDB,
		"CLICKHOUSE_USER":    c.ClickHouseUser,
		"CLICKHOUSE_TIMEOUT": os.Getenv("CLICKHOUSE_TIMEOUT"),
		"BATCH_SIZE":         os.Getenv("BATCH_SIZE"),
		"BATCH_TIMEOUT":      os.Getenv("BATCH_TIMEOUT"),
		"QUEUE_SIZE":         os.Getenv("QUEUE_SIZE"),
		"RETRY_MAX_ATTEMPTS": os.Getenv("RETRY_MAX_ATTEMPTS"),
		"RETRY_INITIAL":      os.Getenv("RETRY_INITIAL"),
		"RETRY_MAX":          os.Getenv("RETRY_MAX"),
		"HTTP_PORT":          c.HTTPPort,
		"HTTP_READ_TIMEOUT":  os.Getenv("HTTP_READ_TIMEOUT"),
		"HTTP_WRITE_TIMEOUT": os.Getenv("HTTP_WRITE_TIMEOUT"),
		"LOG_LEVEL":          os.Getenv("LOG_LEVEL"),
		"SHUTDOWN_TIMEOUT":   os.Getenv("SHUTDOWN_TIMEOUT"),
	} {
		if val == "" {
			missing = append(missing, key)
		}
	}
	if len(c.KafkaBrokers) == 0 {
		missing = append(missing, "KAFKA_BROKERS")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required config keys: %s", strings.Join(missing, ", "))
	}
	if c.BatchSize <= 0 {
		return fmt.Errorf("BATCH_SIZE must be > 0")
	}
	if c.QueueSize <= 0 {
		return fmt.Errorf("QUEUE_SIZE must be > 0")
	}
	if c.RetryMaxAttempts <= 0 {
		return fmt.Errorf("RETRY_MAX_ATTEMPTS must be > 0")
	}
	if c.MetricsEnabled && c.MetricsPort == "" {
		return fmt.Errorf("METRICS_PORT is required when METRICS_ENABLED=true")
	}
	return nil
}

func parseDuration(key, v string) (time.Duration, error) {
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return d, nil
}

func parseInt(key, v string) (int, error) {
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return n, nil
}

func parseBool(key, v string) (bool, error) {
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("%s: %w", key, err)
	}
	return b, nil
}

func parseLogLevel(v string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	}
	return 0, fmt.Errorf("unknown LOG_LEVEL: %q", v)
}
