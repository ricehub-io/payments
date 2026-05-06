package main

import (
	"context"

	paymentv1 "github.com/ricehub-io/proto/gen/go/payment/v1"
)

type paymentServer struct {
	paymentv1.UnimplementedPaymentServiceServer
}

func (s *paymentServer) CreatePayment(
	ctx context.Context,
	req *paymentv1.CreatePaymentRequest,
) (*paymentv1.CreatePaymentResponse, error) {
	return &paymentv1.CreatePaymentResponse{}, nil
}
