package main

import (
	"context"

	"github.com/google/uuid"
	"github.com/ricehub-io/payments/internal/polar"
	paymentv1 "github.com/ricehub-io/proto/gen/go/payment/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type paymentServer struct {
	paymentv1.UnimplementedPaymentServiceServer
	polar *polar.Polar
}

func NewPaymentServer(polar *polar.Polar) *paymentServer {
	return &paymentServer{
		paymentv1.UnimplementedPaymentServiceServer{},
		polar,
	}
}

func (s *paymentServer) CreateCheckout(
	ctx context.Context,
	req *paymentv1.CreateCheckoutRequest,
) (*paymentv1.CreateCheckoutResponse, error) {
	if _, err := uuid.Parse(req.UserId); err != nil {
		return nil, status.Error(codes.InvalidArgument, "user_id must be a valid uuid")
	}
	if _, err := uuid.Parse(req.ProductId); err != nil {
		return nil, status.Error(codes.InvalidArgument, "product_id must be a valid uuid")
	}

	url, err := s.polar.CreateCheckoutSession(req.UserId, req.ProductId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not create checkout session: %v", err)
	}

	return &paymentv1.CreateCheckoutResponse{CheckoutUrl: url}, nil
}
