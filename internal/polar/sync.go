package polar

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	syncInterval  = 24 * time.Hour
	retryInterval = 10 * time.Minute
)

func (p *Polar) StartSyncThread() {
	logger := zap.L()

	attempts := 0
	for {
		logger.Info("Synchronizing...", zap.Int("attempts", attempts))
		if err := p.sync(); err != nil {
			retryIn := retryInterval
			attempts++
			if attempts >= 3 {
				retryIn *= time.Duration(attempts - 2)
			}

			logger.Error("Could not synchronize",
				zap.String("retry_in", retryIn.String()),
				zap.Error(err),
			)
			time.Sleep(retryIn)
			continue
		}
		attempts = 0
		logger.Info("Successfully synchronized", zap.String("sync_in", syncInterval.String()))
		time.Sleep(syncInterval)
	}
}

// sync synchronizes internal subscriptions state with Polar's state,
// where Polar is the source of truth. It does that by fetching active subscriptions
// and upserting them into database. All rows (subscriptions) that were not
// upserted are marked as expired.
func (p *Polar) sync() error {
	logger := zap.L()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	subs, err := p.ListSubscriptions(ctx)
	if err != nil {
		return fmt.Errorf("list subscriptions: %w", err)
	}

	// unaimeds: I'm not using a transaction unlike in the main API I did
	// because we still want to insert/delete other subscriptions no matter
	// if one of them was invalid.

	// upsert active
	seen := make([]uuid.UUID, 0, len(subs))
	for _, sub := range subs {
		userIDStr := sub.Customer.ExternalID
		if userIDStr == nil {
			continue
		}

		userID, err := uuid.Parse(*userIDStr)
		if err != nil {
			logger.Warn("Could not parse external customer ID",
				zap.Stringp("id", userIDStr),
				zap.Error(err),
			)
			continue
		}
		seen = append(seen, userID)

		if err := p.db.InsertSubscription(
			ctx, userID, sub.CurrentPeriodStart, sub.CurrentPeriodEnd,
		); err != nil {
			logger.Error("Could not insert subscription into database",
				zap.String("sub_id", sub.ID),
				zap.Stringp("user_id", userIDStr),
				zap.Error(err),
			)
			continue
		}
	}

	// mark unseen as expired
	expCount, err := p.db.UpdateSubscriptionExpiredExcept(ctx, seen)
	if err != nil {
		logger.Error("Could not mark subscriptions as expired", zap.Error(err))
	}

	logger.Sugar().Infof("Upserted %d subscriptions", len(seen))
	logger.Sugar().Infof("Marked %d subscriptions expired", expCount)

	return nil
}
