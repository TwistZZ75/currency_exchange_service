package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"gw-analytics/internal/api"
	"gw-analytics/internal/config"
	"gw-analytics/internal/consumer"
	"gw-analytics/internal/service"
	"gw-analytics/internal/storage/clickhouse"
	"gw-analytics/pkg/logger"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func run() error {
	var configPath string
	flag.StringVar(&configPath, "c", "config.env", "path to config file")
	flag.Parse()

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	log := logger.NewLogger(cfg.LogLevel)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	storage, err := clickhouse.NewConn(ctx, cfg.ClickHouseAddr, cfg.ClickHouseDB, cfg.ClickHouseUser, cfg.ClickHousePassword, cfg.ClickHouseTimeout)
	if err != nil {
		return fmt.Errorf("init clickhouse: %w", err)
	}
	defer storage.Close()
	log.Info("clickhouse connected", "addr", cfg.ClickHouseAddr, "db", cfg.ClickHouseDB)

	reader := consumer.NewReader(cfg.KafkaBrokers, cfg.KafkaTopic, cfg.KafkaGroupID, cfg.QueueSize)
	defer reader.Close()
	dlq := consumer.NewDLQWriter(cfg.KafkaBrokers, cfg.KafkaDLQTopic)
	defer dlq.Close()
	log.Info("kafka ready",
		"brokers", cfg.KafkaBrokers,
		"topic", cfg.KafkaTopic,
		"dlq_topic", cfg.KafkaDLQTopic,
		"group", cfg.KafkaGroupID,
	)

	var metricsSrv *http.Server
	if cfg.MetricsEnabled {
		metricsSrv = startMetricsServer(cfg.MetricsPort, log)
	}

	h := api.NewHandlers(storage)
	r := api.NewRouter(h)
	httpSrv := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      r,
		ReadTimeout:  cfg.HTTPReadTimeout,
		WriteTimeout: cfg.HTTPWriteTimeout,
	}
	go func() {
		log.Info("http api started", "port", cfg.HTTPPort)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("http server failed", "error", err)
		}
	}()

	processor := service.NewProcessor(reader, storage, dlq, cfg, log)
	procErrCh := make(chan error, 1)
	go func() {
		log.Info("processor started",
			"batch_size", cfg.BatchSize,
			"batch_timeout", cfg.BatchTimeout,
			"queue_size", cfg.QueueSize,
		)
		procErrCh <- processor.Run(ctx)
	}()

	select {
	case <-ctx.Done():
		log.Info("shutdown signal received")
	case err := <-procErrCh:
		if err != nil {
			log.Error("processor exited with error", "error", err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		log.Warn("http shutdown", "error", err)
	}
	if metricsSrv != nil {
		_ = metricsSrv.Shutdown(shutdownCtx)
	}

	select {
	case err := <-procErrCh:
		if err != nil && !errors.Is(err, context.Canceled) {
			return err
		}
	case <-time.After(cfg.ShutdownTimeout):
		log.Warn("processor didn't stop in time")
	}

	log.Info("shutdown complete")
	return nil
}

func startMetricsServer(port string, log *slog.Logger) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		log.Info("metrics server started", "port", port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("metrics server failed", "error", err)
		}
	}()
	return srv
}
