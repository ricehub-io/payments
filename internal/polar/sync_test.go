package polar

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/polarsource/polar-go/models/components"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/ricehub-io/payments/internal/config"
)

func TestRetryDelay(t *testing.T) {
	tests := []struct {
		failures int
		want     time.Duration
	}{
		{0, 10 * time.Minute},
		{1, 10 * time.Minute},
		{2, 10 * time.Minute},
		{3, 20 * time.Minute},
		{4, 30 * time.Minute},
		{10, 90 * time.Minute},
	}

	for _, tc := range tests {
		got := retryDelay(tc.failures)
		assert.Equal(t, tc.want, got, "retryDelay(%d)", tc.failures)
	}
}

func newTestPolar(db store) *Polar {
	p := &Polar{
		logger: zap.NewNop(),
		cfg:    &config.Config{Environment: "dev"},
		db:     db,
	}
	p.listSubs = func(_ context.Context) ([]components.Subscription, error) {
		return nil, nil
	}
	return p
}

func TestSync_HappyPath(t *testing.T) {
	now := time.Now()
	userID1, userID2 := uuid.New(), uuid.New()
	uid1Str, uid2Str := userID1.String(), userID2.String()

	db := &fakeStore{}
	p := newTestPolar(db)
	p.listSubs = func(_ context.Context) ([]components.Subscription, error) {
		return []components.Subscription{
			{
				CurrentPeriodStart: now,
				CurrentPeriodEnd:   now.Add(30 * 24 * time.Hour),
				Customer:           components.SubscriptionCustomer{ExternalID: &uid1Str},
			},
			{
				CurrentPeriodStart: now,
				CurrentPeriodEnd:   now.Add(30 * 24 * time.Hour),
				Customer:           components.SubscriptionCustomer{ExternalID: &uid2Str},
			},
		}, nil
	}

	err := p.sync()
	require.NoError(t, err)

	assert.Len(t, db.insertSubCalls, 2)
	assert.Len(t, db.updateExpiredExceptCalls, 1)
	seen := db.updateExpiredExceptCalls[0]
	assert.Contains(t, seen, userID1)
	assert.Contains(t, seen, userID2)
}

func TestSync_NilExternalIDSkipped(t *testing.T) {
	userID := uuid.New()
	uid := userID.String()

	db := &fakeStore{}
	p := newTestPolar(db)
	p.listSubs = func(_ context.Context) ([]components.Subscription, error) {
		return []components.Subscription{
			{Customer: components.SubscriptionCustomer{ExternalID: nil}},
			{Customer: components.SubscriptionCustomer{ExternalID: &uid}},
		}, nil
	}

	require.NoError(t, p.sync())
	assert.Len(t, db.insertSubCalls, 1)
	assert.Equal(t, userID, db.insertSubCalls[0].UserID)

	seen := db.updateExpiredExceptCalls[0]
	assert.Len(t, seen, 1)
	assert.Contains(t, seen, userID)
}

func TestSync_InvalidUUIDExternalIDSkipped(t *testing.T) {
	bad := "not-a-uuid"
	db := &fakeStore{}
	p := newTestPolar(db)
	p.listSubs = func(_ context.Context) ([]components.Subscription, error) {
		return []components.Subscription{
			{Customer: components.SubscriptionCustomer{ExternalID: &bad}},
		}, nil
	}

	require.NoError(t, p.sync())
	assert.Len(t, db.insertSubCalls, 0)
	seen := db.updateExpiredExceptCalls[0]
	assert.Len(t, seen, 0)
}

func TestSync_InsertErrorContinuesOtherSubs(t *testing.T) {
	uid1, uid2 := uuid.New().String(), uuid.New().String()
	insertCount := 0
	db := &fakeStore{}

	p := newTestPolar(db)
	p.listSubs = func(_ context.Context) ([]components.Subscription, error) {
		return []components.Subscription{
			{Customer: components.SubscriptionCustomer{ExternalID: &uid1}},
			{Customer: components.SubscriptionCustomer{ExternalID: &uid2}},
		}, nil
	}

	db.insertSubErr = errors.New("db error")
	originalInsert := db.InsertSubscription
	_ = originalInsert
	db.insertSubErr = nil

	callDB := &countingStore{failOnCall: 0}
	p.db = callDB

	require.NoError(t, p.sync())
	assert.Equal(t, 2, insertCount+callDB.insertCount, "both subs should be attempted")
	assert.Equal(t, 1, callDB.insertErrCount, "one insert should have failed")
}

func TestSync_UpdateExpiredErrorDoesNotPropagate(t *testing.T) {
	db := &fakeStore{updateExpiredExceptErr: errors.New("db error")}
	p := newTestPolar(db)

	require.NoError(t, p.sync())
}

func TestSync_ListSubsErrorPropagatesToCaller(t *testing.T) {
	db := &fakeStore{}
	p := newTestPolar(db)
	p.listSubs = func(_ context.Context) ([]components.Subscription, error) {
		return nil, errors.New("network error")
	}

	err := p.sync()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "list subscriptions")
}

func TestSync_EmptySubsListCallsUpdateWithEmptySlice(t *testing.T) {
	db := &fakeStore{}
	p := newTestPolar(db)
	p.listSubs = func(_ context.Context) ([]components.Subscription, error) {
		return []components.Subscription{}, nil
	}

	require.NoError(t, p.sync())
	require.Len(t, db.updateExpiredExceptCalls, 1)
	assert.Len(t, db.updateExpiredExceptCalls[0], 0)
}

// countingStore is a specialized fake that fails only its first InsertSubscription call.
type countingStore struct {
	mu             sync.Mutex
	insertCount    int
	insertErrCount int
	failOnCall     int
	fakeStore
}

func (c *countingStore) InsertSubscription(ctx context.Context, userID uuid.UUID, start, end time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	call := c.insertCount
	c.insertCount++
	if call == c.failOnCall {
		c.insertErrCount++
		return errors.New("simulated insert error")
	}
	return nil
}
