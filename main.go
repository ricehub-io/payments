package main

import (
	"fmt"
	"log"
	"net"

	"github.com/ricehub-io/payments/internal/config"
	"github.com/ricehub-io/payments/internal/db"
	"github.com/ricehub-io/payments/internal/polar"
	paymentv1 "github.com/ricehub-io/proto/gen/go/payment/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}

func run() error {
	cfg, err := config.NewConfig()
	if err != nil {
		return fmt.Errorf("new config: %w", err)
	}

	db, err := db.NewDatabase(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("new database: %w", err)
	}
	defer db.Close()

	polar := polar.NewPolar(cfg, db)
	go func() {
		if err := polar.StartWebhookHandler(); err != nil {
			log.Printf("[ERROR] Could not start webhook handler: %v", err)
		}
	}()
	go polar.StartSyncThread()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Port))
	if err != nil {
		return fmt.Errorf("net listen: %w", err)
	}

	paymentServer := NewPaymentServer(polar)
	grpcServer := grpc.NewServer()
	paymentv1.RegisterPaymentServiceServer(grpcServer, paymentServer)

	if cfg.Reflection {
		log.Println("[WARNING] Reflection is enabled!")
		reflection.Register(grpcServer)
	}

	log.Printf("gRPC server available at 127.0.0.1:%d", cfg.Port)
	if err := grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("grpc serve: %w", err)
	}

	return nil
}
