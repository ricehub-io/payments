package polar

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/polarsource/polar-go/models/components"
	"github.com/ricehub-io/payments/internal/logging"
	svix "github.com/svix/svix-webhooks/go"
	"go.uber.org/zap"
)

type webhookEvent struct {
	Type components.WebhookEventType `json:"type"`
	Data json.RawMessage             `json:"data"`
}

func (p *Polar) StartWebhookHandler() {
	if err := p.setupGin(); err != nil {
		p.logger.Error("Could not start webhook handler", zap.Error(err))
	}
}

func (p *Polar) setupGin() error {
	isProd := p.cfg.Environment == "prod"

	if isProd {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	if err := r.SetTrustedProxies(nil); err != nil {
		return fmt.Errorf("gin set trusted proxies: %w", err)
	}

	r.Use(
		ginzap.RecoveryWithZap(p.logger, true),
		ginzap.Ginzap(p.logger, time.RFC3339, true),
	)
	if isProd {
		r.Use(sentrygin.New(sentrygin.Options{Repanic: true}))
	}

	r.POST("/webhook", p.handleWebhookEvent)

	portStr := fmt.Sprintf(":%d", p.cfg.WebhookPort)
	p.logger.Sugar().Infof("HTTP webhook handler available at http://127.0.0.1" + portStr)
	if err := r.Run(portStr); err != nil {
		return fmt.Errorf("router run: %w", err)
	}

	return nil
}

func (p *Polar) handleWebhookEvent(c *gin.Context) {
	logger := logging.LoggerFromContext(c.Request.Context(), p.logger)

	bytes, err := c.GetRawData()
	if err != nil {
		logger.Error("Could not read webhook body", zap.Error(err))
		c.String(http.StatusBadRequest, "error reading request body: %v", err)
		return
	}

	webhookID := c.GetHeader("webhook-id")
	webhookTimestamp := c.GetHeader("webhook-timestamp")
	webhookSignature := c.GetHeader("webhook-signature")
	base64Secret := base64.StdEncoding.EncodeToString([]byte(p.cfg.PolarWebhookSecret))

	wh, err := svix.NewWebhook(base64Secret)
	if err != nil {
		c.String(http.StatusForbidden, "could not verify webhook")
		return
	}

	if hub := sentrygin.GetHubFromContext(c); hub != nil {
		hub.Scope().SetTag("webhook.id", webhookID)
	}

	headers := http.Header{}
	headers.Set("webhook-id", webhookID)
	headers.Set("webhook-timestamp", webhookTimestamp)
	headers.Set("webhook-signature", webhookSignature)

	if err := wh.Verify(bytes, headers); err != nil {
		c.String(http.StatusForbidden, "could not verify webhook")
		return
	}

	if err := p.processWebhookEvent(c.Request.Context(), logger, webhookID, bytes); err != nil {
		logger.Error("Could not process webhook event", zap.Error(err))
		c.String(http.StatusInternalServerError, "could not process webhook event")
		return
	}

	c.Data(http.StatusOK, "application/json", bytes)
}

func (p *Polar) processWebhookEvent(
	ctx context.Context,
	logger *zap.Logger,
	webhookID string,
	body []byte,
) error {
	var event webhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return fmt.Errorf("body unmarshal json: %w", err)
	}

	if hub := sentry.GetHubFromContext(ctx); hub != nil {
		hub.Scope().SetTag("webhook.event_type", string(event.Type))
	}

	switch event.Type {
	case components.WebhookEventTypeSubscriptionActive:
		if err := p.db.InsertWebhookEvent(ctx, webhookID, string(event.Type), event.Data); err != nil {
			logger.Error("Could not insert webhook event log", zap.Error(err))
		}

		if err := p.handleSubscriptionActive(ctx, logger, event.Data); err != nil {
			if dbErr := p.db.UpdateWebhookEventError(ctx, webhookID, err.Error()); dbErr != nil {
				logger.Error("Could not update webhook event error", zap.Error(dbErr))
			}

			return fmt.Errorf("handle subscription active: %w", err)
		}

		if err := p.db.UpdateWebhookEventProcessed(ctx, webhookID); err != nil {
			logger.Error("Could not update webhook event as processed", zap.Error(err))
		}
	default:
		logger.Warn("Received unhandled webhook event", zap.String("type", string(event.Type)))
	}

	return nil
}

func (p *Polar) handleSubscriptionActive(
	ctx context.Context,
	logger *zap.Logger,
	rawPayload json.RawMessage,
) error {
	var payload components.Subscription
	if err := payload.UnmarshalJSON(rawPayload); err != nil {
		return fmt.Errorf("payload unmarshal json: %w", err)
	}

	// unaimeds: im not sure how im gonna integrate this microservice with
	// both of our rest apis therefore i'm not validating the product id yet

	userIDStr := payload.Customer.ExternalID
	if userIDStr == nil {
		return fmt.Errorf("external customer id is nil")
	}

	userID, err := uuid.Parse(*userIDStr)
	if err != nil {
		return fmt.Errorf("userID uuid parse: %w", err)
	}

	if err := p.db.InsertSubscription(
		ctx, userID, payload.CurrentPeriodStart, payload.CurrentPeriodEnd,
	); err != nil {
		return fmt.Errorf("insert subscription: %w", err)
	}

	logger.Info("Inserted new subscription to database", zap.Stringp("user_id", userIDStr))

	return nil
}
