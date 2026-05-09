package server

import (
	"context"
	"testing"

	paymentv1 "github.com/ricehub-io/proto/gen/go/payment/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	validUserID    = "550e8400-e29b-41d4-a716-446655440000"
	validProductID = "550e8400-e29b-41d4-a716-446655440001"
)

func newTestServer(db subscriptionStore, polar checkoutCreator) *PaymentServiceServer {
	return NewPaymentServiceServer(zap.NewNop(), db, polar)
}

func grpcCode(err error) codes.Code {
	st, _ := status.FromError(err)
	return st.Code()
}

func TestCreateCheckout_InvalidUserID(t *testing.T) {
	s := newTestServer(&fakeDB{}, &fakePolar{})

	_, err := s.CreateCheckout(context.Background(), &paymentv1.CreateCheckoutRequest{
		UserId:    "not-a-uuid",
		ProductId: validProductID,
	})

	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpcCode(err))
	assert.Contains(t, status.Convert(err).Message(), "user_id")
}

func TestCreateCheckout_EmptyUserID(t *testing.T) {
	s := newTestServer(&fakeDB{}, &fakePolar{})

	_, err := s.CreateCheckout(context.Background(), &paymentv1.CreateCheckoutRequest{
		UserId:    "",
		ProductId: validProductID,
	})

	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpcCode(err))
}

func TestCreateCheckout_InvalidProductID(t *testing.T) {
	s := newTestServer(&fakeDB{}, &fakePolar{})

	_, err := s.CreateCheckout(context.Background(), &paymentv1.CreateCheckoutRequest{
		UserId:    validUserID,
		ProductId: "not-a-uuid",
	})

	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpcCode(err))
	assert.Contains(t, status.Convert(err).Message(), "product_id")
}

func TestCreateCheckout_DBError_ReturnsInternal(t *testing.T) {
	s := newTestServer(&fakeDB{hasSubErr: errInternal}, &fakePolar{})

	_, err := s.CreateCheckout(context.Background(), &paymentv1.CreateCheckoutRequest{
		UserId:    validUserID,
		ProductId: validProductID,
	})

	require.Error(t, err)
	assert.Equal(t, codes.Internal, grpcCode(err))
}

func TestCreateCheckout_AlreadyHasSubscription(t *testing.T) {
	polar := &fakePolar{url: "https://checkout.example.com"}
	s := newTestServer(&fakeDB{hasSub: true}, polar)

	_, err := s.CreateCheckout(context.Background(), &paymentv1.CreateCheckoutRequest{
		UserId:    validUserID,
		ProductId: validProductID,
	})

	require.Error(t, err)
	assert.Equal(t, codes.AlreadyExists, grpcCode(err))
	assert.Equal(t, 0, polar.calls, "polar.CreateCheckoutSession must not be called")
}

func TestCreateCheckout_PolarError_ReturnsInternal(t *testing.T) {
	s := newTestServer(&fakeDB{hasSub: false}, &fakePolar{err: errInternal})

	_, err := s.CreateCheckout(context.Background(), &paymentv1.CreateCheckoutRequest{
		UserId:    validUserID,
		ProductId: validProductID,
	})

	require.Error(t, err)
	assert.Equal(t, codes.Internal, grpcCode(err))
}

func TestCreateCheckout_Success(t *testing.T) {
	const checkoutURL = "https://checkout.polar.sh/abc123"
	s := newTestServer(&fakeDB{hasSub: false}, &fakePolar{url: checkoutURL})

	resp, err := s.CreateCheckout(context.Background(), &paymentv1.CreateCheckoutRequest{
		UserId:    validUserID,
		ProductId: validProductID,
	})

	require.NoError(t, err)
	assert.Equal(t, checkoutURL, resp.CheckoutUrl)
}
