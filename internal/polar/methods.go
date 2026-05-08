package polar

import (
	"context"
	"fmt"

	polargo "github.com/polarsource/polar-go"
	"github.com/polarsource/polar-go/models/components"
	"github.com/polarsource/polar-go/models/operations"
)

// CreateCheckoutSession creates a new checkout session for given user ID and product ID.
// The user ID is set as an external customer ID.
// Returns URL to the created checkout session.
func (p *Polar) CreateCheckoutSession(
	ctx context.Context,
	userID, productID string,
) (string, error) {
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

// ListSubscription fetches paginated list of all currently active subscriptions.
func (p *Polar) ListSubscriptions(ctx context.Context) ([]components.Subscription, error) {
	res, err := p.sdk.Subscriptions.List(ctx, operations.SubscriptionsListRequest{
		Active: polargo.Bool(true),
		Limit:  polargo.Int64(32),
	})
	if err != nil {
		return nil, fmt.Errorf("sdk subscriptions list: %w", err)
	}

	if res.ListResourceSubscription == nil {
		return nil, fmt.Errorf("list resource subscription is nil")
	}

	subCount := res.ListResourceSubscription.Pagination.TotalCount
	subs := make([]components.Subscription, 0, subCount)
	for {
		subs = append(subs, res.ListResourceSubscription.Items...)

		res, err = res.Next()
		if err != nil {
			return nil, fmt.Errorf("res next: %w", err)
		}

		if res == nil {
			break
		}
	}

	return subs, nil
}
