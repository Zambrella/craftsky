package revenuecat

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"social.craftsky/appview/internal/subscriptions"
)

type recordingEventStore struct {
	events []subscriptions.RevenueCatEvent
	err    error
}

type blockingRequestBody struct {
	closed chan struct{}
	once   sync.Once
}

func (body *blockingRequestBody) Read([]byte) (int, error) {
	<-body.closed
	return 0, context.Canceled
}

func (body *blockingRequestBody) Close() error {
	body.once.Do(func() { close(body.closed) })
	return nil
}

type recordingWebhookObserver struct {
	outcomes []string
}

func (observer *recordingWebhookObserver) ObserveRevenueCatWebhook(_ context.Context, outcome string, _ time.Duration) {
	observer.outcomes = append(observer.outcomes, outcome)
}

func (store *recordingEventStore) AcceptRevenueCatEvent(_ context.Context, event subscriptions.RevenueCatEvent, _ time.Time) (bool, error) {
	store.events = append(store.events, event)
	return len(store.events) == 1, store.err
}

func TestWebhookHandlerAuthenticatesBoundsAndQueuesSanitizedEvents(t *testing.T) {
	now := time.Unix(1788955200, 0)
	store := &recordingEventStore{}
	observer := &recordingWebhookObserver{}
	handler, err := NewWebhookHandler(WebhookConfig{
		Authorization:      "Bearer exact-webhook-secret",
		SigningSecret:      "hmac-secret",
		BodyLimit:          512,
		IngressDeadline:    2 * time.Second,
		SignatureTolerance: 5 * time.Minute,
		Observer:           observer,
	}, store, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	signedRequest := func(body string) *http.Request {
		timestamp := strconv.FormatInt(now.Unix(), 10)
		mac := hmac.New(sha256.New, []byte("hmac-secret"))
		_, _ = mac.Write([]byte(timestamp + "." + body))
		request := httptest.NewRequest(http.MethodPost, "/integrations/revenuecat/webhook", strings.NewReader(body))
		request.Header.Set("Authorization", "Bearer exact-webhook-secret")
		request.Header.Set("X-RevenueCat-Webhook-Signature", "t="+timestamp+",v1="+hex.EncodeToString(mac.Sum(nil)))
		return request
	}
	serve := func(request *http.Request) *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response
	}

	body := `{"api_version":"1.0","event":{"id":"event-unknown","type":"FUTURE_EVENT","app_user_id":"10000000-0000-0000-0000-000000000001","event_timestamp_ms":1788955200123,"receipt":"must-not-cross-boundary"}}`
	if response := serve(signedRequest(body)); response.Code != http.StatusOK {
		t.Fatalf("valid unknown event = %d, body=%s", response.Code, response.Body.String())
	}
	if response := serve(signedRequest(body)); response.Code != http.StatusOK {
		t.Fatalf("duplicate event = %d, body=%s", response.Code, response.Body.String())
	}
	if len(store.events) != 2 || store.events[0].ID != "event-unknown" || store.events[0].Type != "FUTURE_EVENT" ||
		store.events[0].RevenueCatAppUserID.String() != "10000000-0000-0000-0000-000000000001" {
		t.Fatalf("sanitized events = %+v", store.events)
	}

	invalidAuth := signedRequest(body)
	invalidAuth.Header.Set("Authorization", "Bearer wrong")
	if response := serve(invalidAuth); response.Code != http.StatusUnauthorized || strings.Contains(response.Body.String(), "wrong") {
		t.Fatalf("invalid auth = %d %s", response.Code, response.Body.String())
	}
	if response := serve(signedRequest(`{"event":`)); response.Code != http.StatusBadRequest {
		t.Fatalf("malformed body = %d %s", response.Code, response.Body.String())
	}
	oversized := signedRequest(`{"padding":"` + strings.Repeat("x", 600) + `"}`)
	if response := serve(oversized); response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized body = %d %s", response.Code, response.Body.String())
	}
	store.err = context.DeadlineExceeded
	failureBody := strings.Replace(body, "event-unknown", "event-failure", 1)
	if response := serve(signedRequest(failureBody)); response.Code != http.StatusServiceUnavailable {
		t.Fatalf("persistence failure = %d %s", response.Code, response.Body.String())
	}
	if len(store.events) != 3 {
		t.Fatalf("invalid requests reached store: %d events", len(store.events))
	}
	wantOutcomes := []string{"accepted", "duplicate", "authentication_failed", "malformed", "oversized", "store_error"}
	if strings.Join(observer.outcomes, ",") != strings.Join(wantOutcomes, ",") {
		t.Fatalf("webhook outcomes = %v, want %v", observer.outcomes, wantOutcomes)
	}
}

func TestWebhookIngressDeadlineCoversBodyRead(t *testing.T) {
	store := &recordingEventStore{}
	handler, err := NewWebhookHandler(WebhookConfig{
		Authorization: "authorization", SigningSecret: "signature",
		BodyLimit: 512, IngressDeadline: 10 * time.Millisecond,
		SignatureTolerance: time.Minute,
	}, store, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	body := &blockingRequestBody{closed: make(chan struct{})}
	request := httptest.NewRequest(http.MethodPost, "/integrations/revenuecat/webhook", body)
	response := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		handler.ServeHTTP(response, request)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(250 * time.Millisecond):
		_ = body.Close()
		t.Fatal("handler did not enforce the ingress deadline while reading the body")
	}
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("deadline response = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
	if len(store.events) != 0 {
		t.Fatalf("deadline persisted %d events", len(store.events))
	}
}
