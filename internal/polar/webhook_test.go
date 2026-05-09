package polar

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	svix "github.com/svix/svix-webhooks/go"
	"go.uber.org/zap"

	"github.com/ricehub-io/payments/internal/config"
)

const testWebhookSecret = "test-webhook-secret"

// signedRequest returns a POST /webhook request with valid svix signature headers.
func signedRequest(t *testing.T, body []byte) *http.Request {
	t.Helper()
	b64Secret := base64.StdEncoding.EncodeToString([]byte(testWebhookSecret))
	wh, err := svix.NewWebhook(b64Secret)
	require.NoError(t, err)

	msgID := uuid.New().String()
	ts := time.Now()
	sig, err := wh.Sign(msgID, ts, body)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("webhook-id", msgID)
	req.Header.Set("webhook-timestamp", strconv.FormatInt(ts.Unix(), 10))
	req.Header.Set("webhook-signature", sig)
	return req
}

// validSubPayload returns a complete subscription.active event JSON.
// externalID can be set to a valid UUID string or left empty (nil ExternalID in result).
func validSubPayload(externalID string) []byte {
	extIDField := "null"
	if externalID != "" {
		extIDField = fmt.Sprintf("%q", externalID)
	}
	ts := "2024-01-01T00:00:00Z"
	return []byte(fmt.Sprintf(`{
		"type": "subscription.active",
		"data": {
			"id": "sub_test",
			"created_at": %q,
			"amount": 1000,
			"currency": "usd",
			"recurring_interval": "month",
			"recurring_interval_count": 1,
			"status": "active",
			"current_period_start": %q,
			"current_period_end": "2024-02-01T00:00:00Z",
			"cancel_at_period_end": false,
			"customer_id": "cust_123",
			"product_id": "prod_123",
			"metadata": {},
			"customer": {
				"id": "cust_123",
				"created_at": %q,
				"metadata": {},
				"email_verified": false,
				"type": "individual",
				"name": null,
				"billing_address": null,
				"tax_id": null,
				"organization_id": "org_123",
				"avatar_url": "",
				"deleted_at": null,
				"external_id": %s
			},
			"product": {
				"id": "prod_123",
				"created_at": %q,
				"name": "Test Product",
				"description": "",
				"type": "individual",
				"is_archived": false,
				"prices": [],
				"media": [],
				"benefits": [],
				"metadata": {},
				"organization_id": "org_123"
			},
			"prices": [],
			"meters": [],
			"pending_update": null
		}
	}`, ts, ts, ts, extIDField, ts))
}

func newWebhookPolar(db store) *Polar {
	p := &Polar{
		logger: zap.NewNop(),
		cfg: &config.Config{
			Environment:        "dev",
			PolarWebhookSecret: testWebhookSecret,
		},
		db: db,
	}
	return p
}

func TestWebhook_ValidSubscriptionActive_Returns200(t *testing.T) {
	db := &fakeStore{}
	p := newWebhookPolar(db)
	r, err := p.newRouter(false)
	require.NoError(t, err)

	userID := uuid.New().String()
	body := validSubPayload(userID)
	req := signedRequest(t, body)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Len(t, db.insertWebhookCalls, 1)
	assert.Len(t, db.insertSubCalls, 1)
	assert.Len(t, db.updateWebhookProcCalls, 1)
	assert.Len(t, db.updateWebhookErrCalls, 0)
}

func TestWebhook_ValidUnknownEventType_Returns200NoDBWrites(t *testing.T) {
	db := &fakeStore{}
	p := newWebhookPolar(db)
	r, err := p.newRouter(false)
	require.NoError(t, err)

	body := []byte(`{"type": "checkout.created", "data": {}}`)
	req := signedRequest(t, body)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Len(t, db.insertWebhookCalls, 0)
	assert.Len(t, db.insertSubCalls, 0)
}

func TestWebhook_InvalidSignature_Returns403(t *testing.T) {
	db := &fakeStore{}
	p := newWebhookPolar(db)
	r, err := p.newRouter(false)
	require.NoError(t, err)

	body := validSubPayload(uuid.New().String())
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("webhook-id", uuid.New().String())
	req.Header.Set("webhook-timestamp", strconv.FormatInt(time.Now().Unix(), 10))
	req.Header.Set("webhook-signature", "v1,invalidsignature")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestWebhook_MissingSignatureHeader_Returns403(t *testing.T) {
	db := &fakeStore{}
	p := newWebhookPolar(db)
	r, err := p.newRouter(false)
	require.NoError(t, err)

	body := validSubPayload(uuid.New().String())
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("webhook-id", uuid.New().String())
	req.Header.Set("webhook-timestamp", strconv.FormatInt(time.Now().Unix(), 10))
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestWebhook_MissingIDHeader_Returns403(t *testing.T) {
	db := &fakeStore{}
	p := newWebhookPolar(db)
	r, err := p.newRouter(false)
	require.NoError(t, err)

	body := validSubPayload(uuid.New().String())
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("webhook-timestamp", strconv.FormatInt(time.Now().Unix(), 10))
	req.Header.Set("webhook-signature", "v1,fakesig")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestWebhook_StaleTimestamp_Returns403(t *testing.T) {
	db := &fakeStore{}
	p := newWebhookPolar(db)
	r, err := p.newRouter(false)
	require.NoError(t, err)

	b64Secret := base64.StdEncoding.EncodeToString([]byte(testWebhookSecret))
	wh, err := svix.NewWebhook(b64Secret)
	require.NoError(t, err)

	body := validSubPayload(uuid.New().String())
	msgID := uuid.New().String()
	staleTS := time.Now().Add(-10 * time.Minute)
	sig, err := wh.Sign(msgID, staleTS, body)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("webhook-id", msgID)
	req.Header.Set("webhook-timestamp", strconv.FormatInt(staleTS.Unix(), 10))
	req.Header.Set("webhook-signature", sig)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestWebhook_MalformedJSON_Returns500(t *testing.T) {
	db := &fakeStore{}
	p := newWebhookPolar(db)
	r, err := p.newRouter(false)
	require.NoError(t, err)

	body := []byte(`not-json`)
	req := signedRequest(t, body)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestWebhook_SubscriptionActiveNilExternalID_Returns500(t *testing.T) {
	db := &fakeStore{}
	p := newWebhookPolar(db)
	r, err := p.newRouter(false)
	require.NoError(t, err)

	body := validSubPayload("")
	req := signedRequest(t, body)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Len(t, db.updateWebhookErrCalls, 1)
	assert.NotEmpty(t, db.updateWebhookErrCalls[0].ErrMsg)
	assert.Len(t, db.updateWebhookProcCalls, 0)
}

func TestWebhook_SubscriptionActiveNonUUIDExternalID_Returns500(t *testing.T) {
	db := &fakeStore{}
	p := newWebhookPolar(db)
	r, err := p.newRouter(false)
	require.NoError(t, err)

	body := validSubPayload("not-a-uuid")
	req := signedRequest(t, body)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Len(t, db.updateWebhookErrCalls, 1)
	assert.Len(t, db.updateWebhookProcCalls, 0)
}

func TestWebhook_InsertSubscriptionError_Returns500UpdatesError(t *testing.T) {
	db := &fakeStore{insertSubErr: fmt.Errorf("db failure")}
	p := newWebhookPolar(db)
	r, err := p.newRouter(false)
	require.NoError(t, err)

	body := validSubPayload(uuid.New().String())
	req := signedRequest(t, body)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Len(t, db.updateWebhookErrCalls, 1)
	assert.Len(t, db.updateWebhookProcCalls, 0)
}

func TestWebhook_InsertWebhookEventError_ProcessingContinues(t *testing.T) {
	db := &fakeStore{insertWebhookErr: fmt.Errorf("log insert failed")}
	p := newWebhookPolar(db)
	r, err := p.newRouter(false)
	require.NoError(t, err)

	body := validSubPayload(uuid.New().String())
	req := signedRequest(t, body)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Len(t, db.insertSubCalls, 1)
	assert.Len(t, db.updateWebhookProcCalls, 1)
}

func TestWebhook_DuplicateWebhookID_BothReturn200(t *testing.T) {
	db := &fakeStore{}
	p := newWebhookPolar(db)
	r, err := p.newRouter(false)
	require.NoError(t, err)

	body := validSubPayload(uuid.New().String())

	for i := 0; i < 2; i++ {
		req := signedRequest(t, body)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code, "request %d", i+1)
	}
}
