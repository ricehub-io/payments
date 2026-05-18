package polar

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
)

type fakeStore struct {
	mu sync.Mutex

	insertSubCalls            []insertSubCall
	insertSubErr              error
	updateExpiredExceptCalls  [][]uuid.UUID
	updateExpiredExceptResult int64
	updateExpiredExceptErr    error
	insertWebhookCalls        []insertWebhookCall
	insertWebhookErr          error
	updateWebhookErrCalls     []updateWebhookErrCall
	updateWebhookErrErr       error
	updateWebhookProcCalls    []string
	updateWebhookProcErr      error
}

type insertSubCall struct {
	UserID      uuid.UUID
	PeriodStart time.Time
	PeriodEnd   time.Time
}

type insertWebhookCall struct {
	WebhookID string
	EventType string
	Payload   json.RawMessage
}

type updateWebhookErrCall struct {
	WebhookID string
	ErrMsg    string
}

func (f *fakeStore) InsertSubscription(
	_ context.Context,
	userID uuid.UUID,
	periodStart, periodEnd time.Time,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.insertSubCalls = append(f.insertSubCalls, insertSubCall{userID, periodStart, periodEnd})
	return f.insertSubErr
}

func (f *fakeStore) UpdateSubscriptionExpiredExcept(
	_ context.Context,
	userIDs []uuid.UUID,
) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.updateExpiredExceptCalls = append(f.updateExpiredExceptCalls, userIDs)
	return f.updateExpiredExceptResult, f.updateExpiredExceptErr
}

func (f *fakeStore) InsertWebhookEvent(
	_ context.Context,
	webhookID, eventType string,
	payload json.RawMessage,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.insertWebhookCalls = append(f.insertWebhookCalls, insertWebhookCall{
		webhookID, eventType, payload,
	})
	return f.insertWebhookErr
}

func (f *fakeStore) UpdateWebhookEventError(_ context.Context, webhookID, errMsg string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.updateWebhookErrCalls = append(f.updateWebhookErrCalls, updateWebhookErrCall{
		webhookID, errMsg,
	})
	return f.updateWebhookErrErr
}

func (f *fakeStore) UpdateWebhookEventProcessed(_ context.Context, webhookID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.updateWebhookProcCalls = append(f.updateWebhookProcCalls, webhookID)
	return f.updateWebhookProcErr
}
