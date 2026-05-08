package polar

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/polarsource/polar-go/models/components"
	svix "github.com/svix/svix-webhooks/go"
)

type webhookEvent struct {
	Type components.WebhookEventType `json:"type"`
	Data json.RawMessage             `json:"data"`
}

func (p *Polar) StartWebhookHandler() error {
	r := gin.Default()
	if err := r.SetTrustedProxies(nil); err != nil {
		return fmt.Errorf("gin set trusted proxies: %w", err)
	}

	r.POST("/webhook", p.handleWebhookEvent)

	portStr := fmt.Sprintf(":%d", p.cfg.WebhookPort)
	log.Println("Webhook handler available at http://127.0.0.1" + portStr)
	if err := r.Run(portStr); err != nil {
		return fmt.Errorf("router run: %w", err)
	}

	return nil
}

func (p *Polar) handleWebhookEvent(c *gin.Context) {
	bytes, err := c.GetRawData()
	if err != nil {
		log.Printf("could not read webhook body: %v", err)
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

	headers := http.Header{}
	headers.Set("webhook-id", webhookID)
	headers.Set("webhook-timestamp", webhookTimestamp)
	headers.Set("webhook-signature", webhookSignature)

	if err := wh.Verify(bytes, headers); err != nil {
		c.String(http.StatusForbidden, "could not verify webhook")
		return
	}

	if err := p.processWebhookEvent(c.Request.Context(), webhookID, bytes); err != nil {
		log.Printf("could not process webhook event: %v", err)
		c.String(http.StatusInternalServerError, "could not process webhook event")
		return
	}

	c.Data(http.StatusOK, "application/json", bytes)
}

func (p *Polar) processWebhookEvent(
	ctx context.Context,
	webhookID string,
	body []byte,
) error {
	var event webhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return fmt.Errorf("body unmarshal json: %w", err)
	}

	switch event.Type {
	case components.WebhookEventTypeSubscriptionActive:
		if err := p.db.InsertWebhookEvent(ctx, webhookID, string(event.Type), event.Data); err != nil {
			log.Printf("could not insert webhook event log: %v", err)
		}

		if err := p.handleSubscriptionActive(ctx, event.Data); err != nil {
			if dbErr := p.db.UpdateWebhookEventError(ctx, webhookID, err.Error()); dbErr != nil {
				log.Printf("could not set webhook event error message: %v", dbErr)
			}

			return fmt.Errorf("handle subscription active: %w", err)
		}

		if err := p.db.UpdateWebhookEventProcessed(ctx, webhookID); err != nil {
			log.Printf("could not set webhook event as processed: %v", err)
		}
	case components.WebhookEventTypeOrderPaid:
		// TODO
	default:
		log.Printf("[WARNING] Received webhook event with unsupported type: %v", event.Type)
	}

	return nil
}

func (p *Polar) handleSubscriptionActive(ctx context.Context, rawPayload json.RawMessage) error {
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

	log.Printf("New user subscription inserted: %v", payload)

	return nil
}
