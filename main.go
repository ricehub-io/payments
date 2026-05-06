package main

import (
	"fmt"
	"log"
	"net"

	paymentv1 "github.com/ricehub-io/proto/gen/go/payment/v1"
	"google.golang.org/grpc"
)

const port = ":52344"

func main() {
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}

func run() error {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		return fmt.Errorf("net listen: %w", err)
	}

	grpcServer := grpc.NewServer()
	paymentv1.RegisterPaymentServiceServer(grpcServer, &paymentServer{})

	log.Printf("Server available at port %s", port)
	if err := grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("grpc serve: %w", err)
	}

	return nil
}
