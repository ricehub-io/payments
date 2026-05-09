//go:build integration

package db

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInsertSubscription_NewRow(t *testing.T) {
	truncateAll(t)
	ctx := context.Background()

	userID := uuid.New()
	start := time.Now().UTC().Truncate(time.Millisecond)
	end := start.Add(30 * 24 * time.Hour)

	err := testDB.InsertSubscription(ctx, userID, start, end)
	require.NoError(t, err)

	has, err := testDB.HasUserSubscription(ctx, userID)
	require.NoError(t, err)
	assert.True(t, has)
}

func TestInsertSubscription_UpsertUpdatesPeriodDates(t *testing.T) {
	truncateAll(t)
	ctx := context.Background()

	userID := uuid.New()
	now := time.Now().UTC()
	start1 := now
	end1 := now.Add(30 * 24 * time.Hour)

	require.NoError(t, testDB.InsertSubscription(ctx, userID, start1, end1))

	start2 := now.Add(30 * 24 * time.Hour)
	end2 := now.Add(60 * 24 * time.Hour)
	require.NoError(t, testDB.InsertSubscription(ctx, userID, start2, end2))

	has, err := testDB.HasUserSubscription(ctx, userID)
	require.NoError(t, err)
	assert.True(t, has)
}

func TestHasUserSubscription_EmptyTable(t *testing.T) {
	truncateAll(t)

	has, err := testDB.HasUserSubscription(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.False(t, has)
}

func TestHasUserSubscription_ActiveSub(t *testing.T) {
	truncateAll(t)
	ctx := context.Background()

	userID := uuid.New()
	require.NoError(t, testDB.InsertSubscription(
		ctx,
		userID,
		time.Now().UTC(),
		time.Now().UTC().Add(30*24*time.Hour),
	))

	has, err := testDB.HasUserSubscription(ctx, userID)
	require.NoError(t, err)
	assert.True(t, has)
}

func TestHasUserSubscription_OtherUserActive_ReturnsFalse(t *testing.T) {
	truncateAll(t)
	ctx := context.Background()

	other := uuid.New()
	require.NoError(t, testDB.InsertSubscription(
		ctx, other,
		time.Now().UTC(), time.Now().UTC().Add(30*24*time.Hour),
	))

	has, err := testDB.HasUserSubscription(ctx, uuid.New())
	require.NoError(t, err)
	assert.False(t, has, "unrelated user must not be reported as subscribed")
}

func TestHasUserSubscription_CanceledWithFuturePeriodEnd(t *testing.T) {
	truncateAll(t)
	ctx := context.Background()

	userID := uuid.New()
	require.NoError(t, testDB.InsertSubscription(ctx, userID,
		time.Now().UTC(), time.Now().UTC().Add(30*24*time.Hour)))

	_, err := testDB.pool.Exec(
		ctx,
		"UPDATE subscriptions SET status = 'canceled' WHERE user_id = $1",
		userID,
	)
	require.NoError(t, err)

	has, err := testDB.HasUserSubscription(ctx, userID)
	require.NoError(t, err)
	assert.True(t, has, "canceled sub with future period_end should still count as active")
}

func TestHasUserSubscription_CanceledWithPastPeriodEnd(t *testing.T) {
	truncateAll(t)
	ctx := context.Background()

	userID := uuid.New()
	require.NoError(t, testDB.InsertSubscription(ctx, userID,
		time.Now().UTC().Add(-60*24*time.Hour),
		time.Now().UTC().Add(-30*24*time.Hour),
	))

	_, err := testDB.pool.Exec(
		ctx,
		"UPDATE subscriptions SET status = 'canceled' WHERE user_id = $1",
		userID,
	)
	require.NoError(t, err)

	has, err := testDB.HasUserSubscription(ctx, userID)
	require.NoError(t, err)
	assert.False(t, has)
}

func TestHasUserSubscription_ExpiredSub(t *testing.T) {
	truncateAll(t)
	ctx := context.Background()

	userID := uuid.New()
	require.NoError(t, testDB.InsertSubscription(ctx, userID,
		time.Now().UTC(), time.Now().UTC().Add(30*24*time.Hour)))

	_, err := testDB.pool.Exec(ctx,
		"UPDATE subscriptions SET status = 'expired' WHERE user_id = $1", userID)
	require.NoError(t, err)

	has, err := testDB.HasUserSubscription(ctx, userID)
	require.NoError(t, err)
	assert.False(t, has)
}

func TestHasUserSubscription_ExpiredUserAActiveUserB(t *testing.T) {
	truncateAll(t)
	ctx := context.Background()

	userA, userB := uuid.New(), uuid.New()

	require.NoError(t, testDB.InsertSubscription(ctx, userA,
		time.Now().UTC(), time.Now().UTC().Add(30*24*time.Hour)))
	_, err := testDB.pool.Exec(ctx,
		"UPDATE subscriptions SET status = 'expired' WHERE user_id = $1", userA)
	require.NoError(t, err)

	require.NoError(t, testDB.InsertSubscription(ctx, userB,
		time.Now().UTC(), time.Now().UTC().Add(30*24*time.Hour)))

	hasA, err := testDB.HasUserSubscription(ctx, userA)
	require.NoError(t, err)
	assert.False(t, hasA, "expired user A must return false even when user B is active")

	hasB, err := testDB.HasUserSubscription(ctx, userB)
	require.NoError(t, err)
	assert.True(t, hasB)
}

func TestUpdateSubscriptionExpiredExcept_AllRowsMarkedWhenEmptyList(t *testing.T) {
	truncateAll(t)
	ctx := context.Background()

	for range 3 {
		require.NoError(t, testDB.InsertSubscription(ctx, uuid.New(),
			time.Now().UTC(), time.Now().UTC().Add(30*24*time.Hour)))
	}

	count, err := testDB.UpdateSubscriptionExpiredExcept(ctx, []uuid.UUID{})
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)
}

func TestUpdateSubscriptionExpiredExcept_SubsetPreserved(t *testing.T) {
	truncateAll(t)
	ctx := context.Background()

	keep := uuid.New()
	expire1, expire2 := uuid.New(), uuid.New()

	for _, uid := range []uuid.UUID{keep, expire1, expire2} {
		require.NoError(t, testDB.InsertSubscription(ctx, uid,
			time.Now().UTC(), time.Now().UTC().Add(30*24*time.Hour)))
	}

	count, err := testDB.UpdateSubscriptionExpiredExcept(ctx, []uuid.UUID{keep})
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	has, err := testDB.HasUserSubscription(ctx, keep)
	require.NoError(t, err)
	assert.True(t, has)
}

func TestUpdateSubscriptionExpiredExcept_AllInListNoneAffected(t *testing.T) {
	truncateAll(t)
	ctx := context.Background()

	uid1, uid2 := uuid.New(), uuid.New()
	for _, uid := range []uuid.UUID{uid1, uid2} {
		require.NoError(t, testDB.InsertSubscription(ctx, uid,
			time.Now().UTC(), time.Now().UTC().Add(30*24*time.Hour)))
	}

	count, err := testDB.UpdateSubscriptionExpiredExcept(ctx, []uuid.UUID{uid1, uid2})
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestUpdateSubscriptionExpiredExcept_IdempotentOnAlreadyExpired(t *testing.T) {
	truncateAll(t)
	ctx := context.Background()

	userID := uuid.New()
	require.NoError(t, testDB.InsertSubscription(ctx, userID,
		time.Now().UTC(), time.Now().UTC().Add(30*24*time.Hour)))
	_, err := testDB.pool.Exec(ctx,
		"UPDATE subscriptions SET status = 'expired' WHERE user_id = $1", userID)
	require.NoError(t, err)

	count, err := testDB.UpdateSubscriptionExpiredExcept(ctx, []uuid.UUID{})
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}
