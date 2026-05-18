package polar

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	polargo "github.com/polarsource/polar-go"
	"github.com/polarsource/polar-go/models/components"
	"go.uber.org/zap"

	"github.com/ricehub-io/payments/internal/config"
)

type store interface {
	InsertSubscription(
		ctx context.Context,
		userID uuid.UUID,
		periodStart, periodEnd time.Time,
	) error
	UpdateSubscriptionExpiredExcept(ctx context.Context, userIDs []uuid.UUID) (int64, error)
	InsertWebhookEvent(
		ctx context.Context,
		webhookID, eventType string,
		payload json.RawMessage,
	) error
	UpdateWebhookEventError(ctx context.Context, webhookID, errMsg string) error
	UpdateWebhookEventProcessed(ctx context.Context, webhookID string) error
}

type Polar struct {
	logger *zap.Logger
	cfg    *config.Config
	db     store
	sdk    *polargo.Polar

	// HACK: for testing :<
	listSubs func(ctx context.Context) ([]components.Subscription, error)
}

func NewPolar(
	logger *zap.Logger,
	cfg *config.Config,
	db store,
) *Polar {
	opts := []polargo.SDKOption{polargo.WithSecurity(cfg.PolarToken)}
	if cfg.PolarSandbox {
		opts = append(opts, polargo.WithServer(polargo.ServerSandbox))
		logger.Warn("Using Polar in sandbox mode!")
	}

	sdk := polargo.New(opts...)

	p := &Polar{logger: logger, cfg: cfg, db: db, sdk: sdk}
	p.listSubs = p.ListSubscriptions
	return p
}
