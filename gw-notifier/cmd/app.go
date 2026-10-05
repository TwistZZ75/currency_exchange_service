package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"gw-notifier/internal/config"
	"gw-notifier/internal/consumer"
	"gw-notifier/internal/service/processor"
	"gw-notifier/internal/storage/mongo"
	"gw-notifier/pkg/logger"
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

	storage, err := mongo.NewConn(ctx, cfg.MongoURI, cfg.MongoDB, cfg.MongoCollection, cfg.MongoTimeout)
	if err != nil {
		return fmt.Errorf("init mongo: %w", err)
	}
	log.Info("mongo connected", "db", cfg.MongoDB, "collection", cfg.MongoCollection)

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

	metricsSrv := startMetrics(cfg, log)

	processor := processor.NewProcessor(reader, storage, dlq, cfg, log)

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

	if metricsSrv != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = metricsSrv.Shutdown(shutdownCtx)
		cancel()
	}

	closeCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := storage.Close(closeCtx); err != nil {
		log.Warn("mongo close failed", "error", err)
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

func startMetrics(cfg *config.Config, log *slog.Logger) *http.Server {
	if !cfg.MetricsEnabled {
		return nil
	}
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	srv := &http.Server{
		Addr:              ":" + cfg.MetricsPort,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Info("metrics server started", "port", cfg.MetricsPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("metrics server failed", "error", err)
		}
	}()
	return srv
}
