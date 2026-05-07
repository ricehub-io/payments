package polar

import (
	"context"
	"fmt"
	"time"

	"github.com/polarsource/polar-go/models/components"
)

// CreateCheckoutSession creates a new checkout session for given user ID and product ID.
// The user ID is set as an external customer ID.
// Returns URL to the created checkout session.
func (p *Polar) CreateCheckoutSession(userID, productID string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	res, err := p.sdk.Checkouts.Create(ctx, components.CheckoutCreate{
		Products: []string{productID},
		CustomerBillingAddress: &components.AddressInput{
			Country: components.AddressInputCountryAlpha2InputUs,
		},
		ExternalCustomerID: &userID,
	})
	if err != nil {
		return "", fmt.Errorf("sdk checkouts create: %w", err)
	}

	return res.Checkout.URL, nil
}

// ListSubscription fetches 100 most recent active subscriptions.
func (p *Polar) ListSubscription() error {
	// ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	// defer cancel()

	// res, err := p.sdk.Subscriptions.List(ctx, operations.SubscriptionsListRequest{
	// 	Active: polargo.Bool(true),
	// 	Limit:  polargo.Int64(100),
	// })
	// if err != nil {
	// 	return fmt.Errorf("sdk subscriptions list: %w", err)
	// }

	// if res.ListResourceSubscription == nil {
	// 	return fmt.Errorf("list resource subscription is nil")
	// }

	// for {

	// }

	return nil
}
