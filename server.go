package main

import (
	"context"

	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/ricehub-io/payments/internal/db"
	"github.com/ricehub-io/payments/internal/logging"
	"github.com/ricehub-io/payments/internal/polar"
	paymentv1 "github.com/ricehub-io/proto/gen/go/payment/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type paymentServer struct {
	paymentv1.UnimplementedPaymentServiceServer
	db    *db.Database
	polar *polar.Polar
}

func NewPaymentServer(db *db.Database, polar *polar.Polar) *paymentServer {
	return &paymentServer{
		paymentv1.UnimplementedPaymentServiceServer{},
		db,
		polar,
	}
}

func (s *paymentServer) CreateCheckout(
	ctx context.Context,
	req *paymentv1.CreateCheckoutRequest,
) (*paymentv1.CreateCheckoutResponse, error) {
	logger := logging.LoggerFromContext(ctx, zap.L())

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "user_id must be a valid uuid")
	}
	if _, err := uuid.Parse(req.ProductId); err != nil {
		return nil, status.Error(codes.InvalidArgument, "product_id must be a valid uuid")
	}

	if hub := sentry.GetHubFromContext(ctx); hub != nil {
		hub.Scope().SetUser(sentry.User{ID: req.UserId})
		hub.Scope().SetTag("product_id", req.ProductId)
	}

	hasSub, err := s.db.HasUserSubscription(ctx, userID)
	if err != nil {
		logger.Error("Could not check if user has subscription", zap.Error(err))
		return nil, status.Error(codes.Internal, "internal server error")
	}

	if hasSub {
		return nil, status.Error(codes.AlreadyExists, "user has an already active subscription")
	}

	url, err := s.polar.CreateCheckoutSession(ctx, req.UserId, req.ProductId)
	if err != nil {
		logger.Error("Could not create a checkout session", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "could not create checkout session: %v", err)
	}

	return &paymentv1.CreateCheckoutResponse{CheckoutUrl: url}, nil
}
