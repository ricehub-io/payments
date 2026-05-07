package db

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

func (d *Database) InsertSubscription(
	ctx context.Context,
	userID uuid.UUID,
	periodStart, periodEnd time.Time,
) error {
	const query = `
	INSERT INTO subscriptions (user_id, current_period_start, current_period_end)
	VALUES ($1, $2, $3)
	ON CONFLICT (user_id) DO UPDATE
		SET current_period_start = excluded.current_period_start,
			current_period_end = excluded.current_period_end
	`
	_, err := d.pool.Exec(ctx, query, userID, periodStart, periodEnd)
	return err
}

func (d *Database) InsertWebhookEvent(
	ctx context.Context,
	webhookID, eventType string,
	payload json.RawMessage,
) error {
	const query = `
	INSERT INTO webhook_events (polar_webhook_id, event_type, payload)
	VALUES ($1, $2, $3)
	`
	_, err := d.pool.Exec(ctx, query, webhookID, eventType, payload)
	return err
}

func (d *Database) UpdateWebhookEventError(
	ctx context.Context,
	webhookID, errMsg string,
) error {
	const query = "UPDATE webhook_events SET error = $2 WHERE polar_webhook_id = $1"
	_, err := d.pool.Exec(ctx, query, webhookID, errMsg)
	return err
}

func (d *Database) UpdateWebhookEventProcessed(ctx context.Context, webhookID string) error {
	const query = "UPDATE webhook_events SET processed_at = now() WHERE polar_webhook_id = $1"
	_, err := d.pool.Exec(ctx, query, webhookID)
	return err
}
