package main

import (
	"fmt"
	"log"
	"net"

	"github.com/ricehub-io/payments/internal/config"
	"github.com/ricehub-io/payments/internal/db"
	"github.com/ricehub-io/payments/internal/logging"
	"github.com/ricehub-io/payments/internal/polar"
	"github.com/ricehub-io/payments/internal/server"
	paymentv1 "github.com/ricehub-io/proto/gen/go/payment/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("Could not create a new config: %v", err)
	}

	logger, err := logging.Init(zap.InfoLevel, cfg.Environment == "prod", cfg.SentryDSN)
	if err != nil {
		log.Fatalf("Could not initialize logging: %v", err)
	}
	defer logging.Sync(logger)

	if err := run(cfg, logger); err != nil {
		logger.Fatal("Run failed", zap.Error(err))
	}
}

func run(cfg *config.Config, logger *zap.Logger) error {
	isProd := cfg.Environment == "prod"
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

	port := fmt.Sprintf(":%d", cfg.Port)
	lis, err := net.Listen("tcp", port)
	if err != nil {
		return fmt.Errorf("net listen: %w", err)
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

	logger.Sugar().Infof("gRPC server available at http://127.0.0.1%s", port)
	if err := grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("grpc serve: %w", err)
	}

	return nil
}
