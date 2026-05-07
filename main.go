package main

import (
	"fmt"
	"log"
	"net"

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
	cfg, err := NewConfig()
	if err != nil {
		return fmt.Errorf("new config: %w", err)
	}

	polar := polar.NewPolar(cfg.PolarToken, cfg.PolarSandbox)

	port := ":" + cfg.Port
	lis, err := net.Listen("tcp", port)
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

	log.Printf("gRPC server available at 127.0.0.1%s", port)
	if err := grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("grpc serve: %w", err)
	}

	return nil
}
