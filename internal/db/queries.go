package db

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// -- subscriptions --
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

// HasUserSubscription fetches whether user has an active subscription in db.
func (d *Database) HasUserSubscription(
	ctx context.Context,
	userID uuid.UUID,
) (has bool, err error) {
	const query = `
	SELECT EXISTS (
		SELECT 1
		FROM subscriptions
		WHERE
			user_id = $1 AND
			status = 'active' OR
			(status = 'canceled' AND current_period_end > now())
	)
	`
	err = d.pool.QueryRow(ctx, query, userID).Scan(&has)
	return
}

// UpdateSubscriptionExpiredExcept updates status to 'expired' for all rows
// that are not present in the given user ID list.
func (d *Database) UpdateSubscriptionExpiredExcept(
	ctx context.Context,
	userIDs []uuid.UUID,
) (int64, error) {
	const query = "UPDATE subscriptions SET status = 'expired' WHERE user_id <> ALL($1)"
	cmd, err := d.pool.Exec(ctx, query, userIDs)
	return cmd.RowsAffected(), err
}

// -- webhook_events --
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
