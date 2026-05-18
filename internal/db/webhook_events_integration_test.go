//go:build integration

package db

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInsertWebhookEvent_Inserted(t *testing.T) {
	truncateAll(t)
	ctx := context.Background()

	payload := json.RawMessage(`{"key": "value"}`)
	err := testDB.InsertWebhookEvent(ctx, "wh_001", "subscription.active", payload)
	require.NoError(t, err)

	var count int
	var processedAt *time.Time
	var errCol *string
	err = testDB.pool.QueryRow(
		ctx,
		`SELECT COUNT(*), MAX(processed_at), MAX(error)
		FROM webhook_events WHERE polar_webhook_id = $1`,
		"wh_001",
	).Scan(&count, &processedAt, &errCol)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
	assert.Nil(t, processedAt)
	assert.Nil(t, errCol)
}

func TestInsertWebhookEvent_DuplicateIDReturnsError(t *testing.T) {
	truncateAll(t)
	ctx := context.Background()

	payload := json.RawMessage(`{}`)
	require.NoError(t, testDB.InsertWebhookEvent(ctx, "wh_dup", "subscription.active", payload))

	err := testDB.InsertWebhookEvent(ctx, "wh_dup", "subscription.active", payload)
	require.Error(t, err, "duplicate polar_webhook_id must violate unique constraint")
}

func TestInsertWebhookEvent_EmptyPayload(t *testing.T) {
	truncateAll(t)
	ctx := context.Background()

	err := testDB.InsertWebhookEvent(ctx, "wh_empty", "subscription.active", json.RawMessage(`{}`))
	require.NoError(t, err)
}

func TestInsertWebhookEvent_LargePayload(t *testing.T) {
	truncateAll(t)
	ctx := context.Background()

	large := `{"data": "` + strings.Repeat("x", 64*1024) + `"}`
	err := testDB.InsertWebhookEvent(ctx, "wh_large", "subscription.active", json.RawMessage(large))
	require.NoError(t, err)
}

func TestUpdateWebhookEventError_SetsErrorColumn(t *testing.T) {
	truncateAll(t)
	ctx := context.Background()

	require.NoError(
		t,
		testDB.InsertWebhookEvent(ctx, "wh_err_test", "subscription.active", json.RawMessage(`{}`)),
	)

	err := testDB.UpdateWebhookEventError(ctx, "wh_err_test", "something went wrong")
	require.NoError(t, err)

	var errMsg *string
	var processedAt *time.Time
	err = testDB.pool.QueryRow(ctx,
		"SELECT error, processed_at FROM webhook_events WHERE polar_webhook_id = $1",
		"wh_err_test",
	).Scan(&errMsg, &processedAt)
	require.NoError(t, err)
	require.NotNil(t, errMsg)
	assert.Equal(t, "something went wrong", *errMsg)
	assert.Nil(t, processedAt, "processed_at must remain NULL after an error update")
}

func TestUpdateWebhookEventError_NonExistentIDNoError(t *testing.T) {
	truncateAll(t)
	ctx := context.Background()

	err := testDB.UpdateWebhookEventError(ctx, "wh_nonexistent", "msg")
	require.NoError(t, err)
}

func TestUpdateWebhookEventProcessed_SetsProcessedAt(t *testing.T) {
	truncateAll(t)
	ctx := context.Background()
	before := time.Now().UTC()

	require.NoError(t,
		testDB.InsertWebhookEvent(ctx, "wh_proc", "subscription.active", json.RawMessage(`{}`)),
	)

	err := testDB.UpdateWebhookEventProcessed(ctx, "wh_proc")
	require.NoError(t, err)

	var processedAt *time.Time
	err = testDB.pool.QueryRow(ctx,
		"SELECT processed_at FROM webhook_events WHERE polar_webhook_id = $1",
		"wh_proc",
	).Scan(&processedAt)
	require.NoError(t, err)
	require.NotNil(t, processedAt)
	assert.WithinDuration(t, before, *processedAt, 5*time.Second)
}

func TestUpdateWebhookEventProcessed_NonExistentIDNoError(t *testing.T) {
	truncateAll(t)
	ctx := context.Background()

	err := testDB.UpdateWebhookEventProcessed(ctx, "wh_nonexistent")
	require.NoError(t, err)
}

func TestUpdateWebhookEventProcessed_AfterError_BothColumnsSet(t *testing.T) {
	truncateAll(t)
	ctx := context.Background()

	require.NoError(t,
		testDB.InsertWebhookEvent(ctx, "wh_both", "subscription.active", json.RawMessage(`{}`)),
	)
	require.NoError(t, testDB.UpdateWebhookEventError(ctx, "wh_both", "error msg"))
	require.NoError(t, testDB.UpdateWebhookEventProcessed(ctx, "wh_both"))

	var errMsg *string
	var processedAt *time.Time
	err := testDB.pool.QueryRow(ctx,
		"SELECT error, processed_at FROM webhook_events WHERE polar_webhook_id = $1",
		"wh_both",
	).Scan(&errMsg, &processedAt)
	require.NoError(t, err)
	assert.NotNil(t, errMsg)
	assert.NotNil(t, processedAt)
}
