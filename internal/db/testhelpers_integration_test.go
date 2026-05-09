//go:build integration

package db

import (
	"context"
	"testing"
)

func truncateAll(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	_, err := testDB.pool.Exec(ctx, "TRUNCATE subscriptions, webhook_events RESTART IDENTITY")
	if err != nil {
		t.Fatalf("truncateAll: %v", err)
	}
}
