package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTPPort         string
	HTTPReadTimeout  time.Duration
	HTTPWriteTimeout time.Duration

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	JWTSecret string
	JWTTTL    time.Duration

	ExchangerAddr    string
	ExchangerTimeout time.Duration
	RatesCacheTTL    time.Duration

	KafkaBrokers           []string
	KafkaTopic             string
	LargeTransferThreshold float64

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
		HTTPPort:      os.Getenv("HTTP_PORT"),
		DBHost:        os.Getenv("POSTGRES_HOST"),
		DBPort:        os.Getenv("POSTGRES_PORT"),
		DBUser:        os.Getenv("POSTGRES_USER"),
		DBPassword:    os.Getenv("POSTGRES_PASSWORD"),
		DBName:        os.Getenv("POSTGRES_DB"),
		DBSSLMode:     os.Getenv("POSTGRES_SSLMODE"),
		JWTSecret:     os.Getenv("JWT_SECRET"),
		ExchangerAddr: os.Getenv("EXCHANGER_ADDR"),
		KafkaTopic:    os.Getenv("KAFKA_TOPIC"),
	}

	// на случай, если у нас несколько брокеров будут разнесены по хостам
	// например: KAFKA_BROKERS=host1:9092,host2:9092
	if raw := os.Getenv("KAFKA_BROKERS"); raw != "" {
		for _, b := range strings.Split(raw, ",") {
			if b = strings.TrimSpace(b); b != "" {
				cfg.KafkaBrokers = append(cfg.KafkaBrokers, b)
			}
		}
	}

	var err error

	// парсим параметры в нормальный для гошки вид
	if cfg.HTTPReadTimeout, err = parseDuration("HTTP_READ_TIMEOUT", os.Getenv("HTTP_READ_TIMEOUT")); err != nil {
		return nil, err
	}
	if cfg.HTTPWriteTimeout, err = parseDuration("HTTP_WRITE_TIMEOUT", os.Getenv("HTTP_WRITE_TIMEOUT")); err != nil {
		return nil, err
	}
	if cfg.JWTTTL, err = parseDuration("JWT_TTL", os.Getenv("JWT_TTL")); err != nil {
		return nil, err
	}
	if cfg.ExchangerTimeout, err = parseDuration("EXCHANGER_TIMEOUT", os.Getenv("EXCHANGER_TIMEOUT")); err != nil {
		return nil, err
	}
	if cfg.RatesCacheTTL, err = parseDuration("RATES_CACHE_TTL", os.Getenv("RATES_CACHE_TTL")); err != nil {
		return nil, err
	}
	if cfg.ShutdownTimeout, err = parseDuration("SHUTDOWN_TIMEOUT", os.Getenv("SHUTDOWN_TIMEOUT")); err != nil {
		return nil, err
	}

	if cfg.LogLevel, err = parseLogLevel(os.Getenv("LOG_LEVEL")); err != nil {
		return nil, err
	}

	if cfg.LargeTransferThreshold, err = parseFloat("LARGE_TRANSFER_THRESHOLD", os.Getenv("LARGE_TRANSFER_THRESHOLD")); err != nil {
		return nil, err
	}

	return cfg, cfg.validate()
}

func parseDuration(key, v string) (time.Duration, error) {
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return d, nil
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

func parseFloat(key, v string) (float64, error) {
	var f float64
	if _, err := fmt.Sscanf(v, "%f", &f); err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return f, nil
}

// формируем строку подключения
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName, c.DBSSLMode,
	)
}

// проверяем все ли параметры указаны в файле config.env
// если что-то не указано, то название параметра попадает в слайс missing
func (c *Config) validate() error {
	missing := []string{}
	for key, val := range map[string]string{
		"HTTP_PORT":                c.HTTPPort,
		"DB_HOST":                  c.DBHost,
		"DB_PORT":                  c.DBPort,
		"DB_USER":                  c.DBUser,
		"DB_PASSWORD":              c.DBPassword,
		"DB_NAME":                  c.DBName,
		"DB_SSLMODE":               c.DBSSLMode,
		"JWT_SECRET":               c.JWTSecret,
		"EXCHANGER_ADDR":           c.ExchangerAddr,
		"KAFKA_TOPIC":              c.KafkaTopic,
		"LOG_LEVEL":                os.Getenv("LOG_LEVEL"),
		"HTTP_READ_TIMEOUT":        os.Getenv("HTTP_READ_TIMEOUT"),
		"HTTP_WRITE_TIMEOUT":       os.Getenv("HTTP_WRITE_TIMEOUT"),
		"JWT_TTL":                  os.Getenv("JWT_TTL"),
		"EXCHANGER_TIMEOUT":        os.Getenv("EXCHANGER_TIMEOUT"),
		"RATES_CACHE_TTL":          os.Getenv("RATES_CACHE_TTL"),
		"SHUTDOWN_TIMEOUT":         os.Getenv("SHUTDOWN_TIMEOUT"),
		"LARGE_TRANSFER_THRESHOLD": os.Getenv("LARGE_TRANSFER_THRESHOLD"),
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
	if c.ShutdownTimeout <= 0 {
		return fmt.Errorf("SHUTDOWN_TIMEOUT must be positive")
	}
	return nil
}
