package server

import (
	"context"
	"net"
	"testing"

	paymentv1 "github.com/ricehub-io/proto/gen/go/payment/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"github.com/ricehub-io/payments/internal/logging"
)

const bufSize = 1024 * 1024

func newBufconnServer(t *testing.T, db subscriptionStore, polar checkoutCreator) paymentv1.PaymentServiceClient {
	t.Helper()
	lis := bufconn.Listen(bufSize)

	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(
		logging.ZapUnaryServerInterceptor(zap.NewNop()),
		logging.SentryUnaryServerInterceptor(zap.NewNop()),
	))
	paymentv1.RegisterPaymentServiceServer(srv, NewPaymentServiceServer(zap.NewNop(), db, polar))

	go func() {
		if err := srv.Serve(lis); err != nil {
			_ = err
		}
	}()
	t.Cleanup(func() {
		srv.GracefulStop()
		lis.Close()
	})

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })

	return paymentv1.NewPaymentServiceClient(conn)
}

func TestBufconn_Success(t *testing.T) {
	const checkoutURL = "https://checkout.polar.sh/bufconn-test"
	client := newBufconnServer(t,
		&fakeDB{hasSub: false},
		&fakePolar{url: checkoutURL},
	)

	resp, err := client.CreateCheckout(context.Background(), &paymentv1.CreateCheckoutRequest{
		UserId:    validUserID,
		ProductId: validProductID,
	})

	require.NoError(t, err)
	assert.Equal(t, checkoutURL, resp.CheckoutUrl)
}

func TestBufconn_InvalidArgument_PropagatesThroughInterceptors(t *testing.T) {
	client := newBufconnServer(t, &fakeDB{}, &fakePolar{})

	_, err := client.CreateCheckout(context.Background(), &paymentv1.CreateCheckoutRequest{
		UserId:    "not-a-uuid",
		ProductId: validProductID,
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}
