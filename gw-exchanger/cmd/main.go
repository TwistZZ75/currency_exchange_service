package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"gw-exchanger/internal/config"
	"gw-exchanger/internal/server"
	"gw-exchanger/internal/storages/postgres"
	"gw-exchanger/pkg/logger"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "proto_exchange/exchange"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "gw-exchanger: %v\n", err)
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

	log := logger.New(cfg.LogLevel)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	storage, err := postgres.NewPool(ctx, cfg.DSN())
	if err != nil {
		return fmt.Errorf("init storage: %w", err)
	}
	defer storage.Close()
	log.Info("storage connected", "db", cfg.DBName, "host", cfg.DBHost)

	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		return fmt.Errorf("listen on :%s: %w", cfg.GRPCPort, err)
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(logger.LoggingInterceptor(log)),
	)

	pb.RegisterExchangeServiceServer(grpcServer, server.NewServer(storage, log))

	healthSrv := health.NewServer()
	healthSrv.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(grpcServer, healthSrv)
	reflection.Register(grpcServer)

	errCh := make(chan error, 1)
	go func() {
		log.Info("grpc server started", "port", cfg.GRPCPort)
		if serveErr := grpcServer.Serve(lis); serveErr != nil &&
			!errors.Is(serveErr, grpc.ErrServerStopped) {
			errCh <- serveErr
		}
	}()

	select {
	case <-ctx.Done():
		log.Info("shutdown signal received")
	case err := <-errCh:
		return fmt.Errorf("grpc serve: %w", err)
	}

	// Graceful shutdown.
	done := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		log.Info("grpc server stopped gracefully")
	case <-time.After(cfg.ShutDownTimeout):
		log.Warn("graceful shutdown timeout, forcing stop")
		grpcServer.Stop()
	}

	return nil
}
