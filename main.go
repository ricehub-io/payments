package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	paymentv1 "github.com/ricehub-io/proto/gen/go/payment/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/ricehub-io/payments/internal/config"
	"github.com/ricehub-io/payments/internal/db"
	"github.com/ricehub-io/payments/internal/logging"
	"github.com/ricehub-io/payments/internal/polar"
	"github.com/ricehub-io/payments/internal/server"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Server run failed: %v", err)
	}
}

func run() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cfg, err := config.NewConfig(ctx)
	if err != nil {
		return fmt.Errorf("new config: %w", err)
	}

	isProd := cfg.Environment == "prod"
	logger, err := logging.Init(zap.InfoLevel, isProd, cfg.SentryDSN)
	if err != nil {
		return fmt.Errorf("initializing logging: %w", err)
	}
	defer logging.Sync(logger)

	if !isProd {
		logger.Warn("Running in development environment!")
	}

	db, err := db.NewDatabase(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("new database: %w", err)
	}
	defer db.Close()

	polar := polar.NewPolar(logger, cfg, db)
	go polar.StartWebhookHandler()
	go polar.StartSyncThread()

	lisCfg := net.ListenConfig{KeepAlive: 5 * time.Minute}
	lis, err := lisCfg.Listen(ctx, "tcp", fmt.Sprintf(":%d", cfg.Port))
	if err != nil {
		return fmt.Errorf("listening on tcp: %w", err)
	}

	paymentServer := server.NewPaymentServiceServer(logger, db, polar)
	grpcServer := grpc.NewServer(grpc.ChainUnaryInterceptor(
		logging.ZapUnaryServerInterceptor(logger),
		logging.SentryUnaryServerInterceptor(logger),
	))
	paymentv1.RegisterPaymentServiceServer(grpcServer, paymentServer)

	if !isProd {
		logger.Info("Server reflection enabled")
		reflection.Register(grpcServer)
	}

	logger.Sugar().Infof("gRPC server available on %s", lis.Addr())
	if err := grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("serving grpc server: %w", err)
	}

	return nil
}
