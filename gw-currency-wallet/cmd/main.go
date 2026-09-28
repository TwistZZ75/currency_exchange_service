package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"gw-currency-wallet/internal/auth"
	"gw-currency-wallet/internal/config"
	exchanger "gw-currency-wallet/internal/exchanger_client"
	"gw-currency-wallet/internal/handlers"
	"gw-currency-wallet/internal/kafka"
	"gw-currency-wallet/internal/router"
	"gw-currency-wallet/internal/service"
	"gw-currency-wallet/internal/storages/postgres"
	"gw-currency-wallet/pkg/logger"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "gw-currency-wallet: %v\n", err)
		os.Exit(1)
	}
}

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

	// Storage
	storage, err := postgres.NewPool(ctx, cfg.DSN())
	if err != nil {
		return fmt.Errorf("init storage: %w", err)
	}
	defer storage.Close()
	log.Info("storage connected", "db", cfg.DBName, "host", cfg.DBHost)

	//  exchanger gRPC + cache
	exCli, err := exchanger.NewClient(cfg.ExchangerAddr)
	if err != nil {
		return fmt.Errorf("init exchanger client: %w", err)
	}
	defer exCli.Close()
	rates := exchanger.NewCachedRates(exCli, cfg.RatesCacheTTL)
	log.Info("exchanger client connected", "addr", cfg.ExchangerAddr)

	//  Kafka producer
	producer := kafka.NewProducer(cfg.KafkaBrokers, cfg.KafkaTopic, cfg.LargeTransferThreshold)
	defer producer.Close()
	log.Info("kafka producer ready", "brokers", cfg.KafkaBrokers, "topic", cfg.KafkaTopic)

	//  JWT
	jwtMgr := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTTTL)

	//  Service
	svc := &service.Service{
		Storage:        storage,
		JWT:            jwtMgr,
		Rates:          rates,
		Producer:       producer,
		Logger:         log,
		LargeThreshold: cfg.LargeTransferThreshold,
	}

	//  HTTP
	h := handlers.NewHandler(svc, log)
	r := router.New(h, jwtMgr, log)

	srv := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      r,
		ReadTimeout:  cfg.HTTPReadTimeout,
		WriteTimeout: cfg.HTTPWriteTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("http server started", "port", cfg.HTTPPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Info("shutdown signal received")
	case err := <-errCh:
		return fmt.Errorf("http serve: %w", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Warn("graceful shutdown failed", "error", err)
	} else {
		log.Info("http server stopped gracefully")
	}
	return nil
}
