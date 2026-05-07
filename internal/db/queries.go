package db

import (
	"context"
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
	`
	_, err := d.pool.Exec(ctx, query, userID, periodStart, periodEnd)
	return err
}
