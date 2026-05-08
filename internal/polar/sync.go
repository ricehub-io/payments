package polar

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

func (p *Polar) StartSyncThread() {
	for {
		log.Println("Syncing...")
		if err := p.sync(); err != nil {
			log.Printf("[ERROR] Polar sync failed: %v", err)
		}
		time.Sleep(24 * time.Hour)
	}
}

// sync synchronizes internal subscriptions state with Polar's state,
// where Polar is the source of truth. It does that by fetching active subscriptions
// and upserting them into database. All rows (subscriptions) that were not
// upserted are marked as expired.
func (p *Polar) sync() error {
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
			log.Printf("[WARNING] Could not parse external ID: %v", err)
			continue
		}
		seen = append(seen, userID)

		if err := p.db.InsertSubscription(
			ctx, userID, sub.CurrentPeriodStart, sub.CurrentPeriodEnd,
		); err != nil {
			log.Printf("[ERROR] Could not insert subscription: %v", err)
			continue
		}
	}

	// mark unseen as expired
	expCount, err := p.db.UpdateSubscriptionExpiredExcept(ctx, seen)
	if err != nil {
		log.Printf("[ERROR] Could not mark subscriptions as expired: %v", err)
	}

	log.Printf("Upserted %d subscriptions", len(seen))
	log.Printf("Marked %d subscriptions as expired", expCount)

	return nil
}
