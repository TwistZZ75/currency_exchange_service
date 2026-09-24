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
	GRPCPort        string
	DBPort          string
	DBUser          string
	DBHost          string
	DBPassword      string
	DBSSLMode       string
	DBName          string
	LogLevel        slog.Level
	ShutDownTimeout time.Duration
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
		GRPCPort:   os.Getenv("GRPC_PORT"),
		DBHost:     os.Getenv("POSTGRES_HOST"),
		DBPort:     os.Getenv("POSTGRES_PORT"),
		DBUser:     os.Getenv("POSTGRES_USER"),
		DBPassword: os.Getenv("POSTGRES_PASSWORD"),
		DBName:     os.Getenv("POSTGRES_DB"),
		DBSSLMode:  os.Getenv("POSTGRES_SSLMODE"),
	}

	level, err := parseLogLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		return nil, err
	}
	cfg.LogLevel = level

	timeout, err := time.ParseDuration(os.Getenv("SHUTDOWN_TIMEOUT"))
	if err != nil {
		return nil, fmt.Errorf("SHUTDOWN_TIMEOUT: %w", err)
	}
	cfg.ShutDownTimeout = timeout

	return cfg, cfg.validate()
}

func (c *Config) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName, c.DBSSLMode,
	)
}

func (c *Config) validate() error {
	missing := []string{}
	for key, val := range map[string]string{
		"GRPC_PORT":        c.GRPCPort,
		"DB_HOST":          c.DBHost,
		"DB_PORT":          c.DBPort,
		"DB_USER":          c.DBUser,
		"DB_PASSWORD":      c.DBPassword,
		"DB_NAME":          c.DBName,
		"DB_SSLMODE":       c.DBSSLMode,
		"LOG_LEVEL":        os.Getenv("LOG_LEVEL"),
		"SHUTDOWN_TIMEOUT": os.Getenv("SHUTDOWN_TIMEOUT"),
	} {
		if val == "" {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required config keys: %s", strings.Join(missing, ", "))
	}
	if c.ShutDownTimeout <= 0 {
		return fmt.Errorf("SHUTDOWN_TIMEOUT must be positive")
	}
	return nil
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
