package observability_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/getsentry/sentry-go"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/notifications"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/push"
	"social.craftsky/appview/internal/testdb"
)

type privacyIntegrationSender struct {
	mu      sync.Mutex
	calls   []push.SendRequest
	failure string
}

func (s *privacyIntegrationSender) Send(_ context.Context, request push.SendRequest) (push.ProviderResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, request)
	if len(s.calls) == 1 {
		return push.ProviderResult{Class: push.ResultRetryable}, errors.New(s.failure)
	}
	return push.ProviderResult{Class: push.ResultSuccess}, nil
}

func (s *privacyIntegrationSender) Calls() []push.SendRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]push.SendRequest(nil), s.calls...)
}

func TestPushPrivacySentinelsAcrossRegistrationEnqueueDispatchAndTelemetry(t *testing.T) {
	pool := testdb.WithSchema(t, `
		CREATE TABLE bluesky_profiles(did TEXT PRIMARY KEY,display_name TEXT,avatar_cid TEXT);
		CREATE TABLE craftsky_posts(uri TEXT PRIMARY KEY,reply_root_uri TEXT,reply_parent_uri TEXT);
	`)
	migration, err := os.ReadFile("../../migrations/000021_appview_notifications.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
		t.Fatal(err)
	}

	const (
		tokenSentinel      = "SENTINEL_FCM_TOKEN"
		credentialSentinel = "SENTINEL_FIREBASE_CREDENTIAL"
		recipientDID       = "did:plc:privacyrecipient"
		actorDID           = "did:plc:privacyactor"
		handleSentinel     = "sentinel-handle.example"
		sourceURISentinel  = "at://did:plc:privacyactor/social.craftsky.feed.post/sentinel-source"
		subjectURISentinel = "at://did:plc:privacyactor/social.craftsky.feed.post/sentinel-subject"
		textSentinel       = "SENTINEL_POST_TEXT"
		titleSentinel      = "SENTINEL_PROJECT_TITLE"
		imageSentinel      = "https://images.invalid/SENTINEL_IMAGE.jpg"
		payloadSentinel    = `{"fullPayload":"SENTINEL_FULL_PAYLOAD"}`
		providerError      = "SENTINEL_PROVIDER_ERROR"
	)
	forbidden := []string{tokenSentinel, credentialSentinel, recipientDID, actorDID, handleSentinel, sourceURISentinel, subjectURISentinel, textSentinel, titleSentinel, imageSentinel, payloadSentinel, "SENTINEL_FULL_PAYLOAD", providerError}
	failureText := strings.Join(forbidden, " ")

	var stdout bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&stdout, nil))
	transport := &sentry.MockTransport{}
	recorder := observability.NewInMemoryMetricRecorder()
	observer := observability.New(observability.Config{
		Env: "test", SentryDSN: "https://public@example.invalid/1", SentryTransport: transport,
		LogsEnabled: true, MetricsEnabled: true, MetricRecorder: recorder, Logger: logger,
	})
	store := api.NewPostStore(pool)

	requestBody, _ := json.Marshal(map[string]string{"platform": "ios", "token": tokenSentinel})
	request := httptest.NewRequest(http.MethodPost, "/v1/notifications/devices", bytes.NewReader(requestBody))
	ctx := middleware.WithDID(request.Context(), syntax.DID(recipientDID))
	ctx = middleware.WithDeviceID(ctx, "privacy-device")
	request = request.WithContext(ctx)
	response := httptest.NewRecorder()
	api.RegisterNotificationDeviceHandler(store, logger).ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("registration status=%d body=%s", response.Code, response.Body.String())
	}
	var registration struct {
		AccountSubscriptionID string `json:"accountSubscriptionId"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &registration); err != nil || registration.AccountSubscriptionID == "" {
		t.Fatalf("registration response=%q err=%v", response.Body.String(), err)
	}
	forbidden = append(forbidden, registration.AccountSubscriptionID)

	if _, err := pool.Exec(context.Background(), `INSERT INTO bluesky_profiles(did,display_name) VALUES($1,'Alice')`, actorDID); err != nil {
		t.Fatal(err)
	}
	base := time.Now().UTC().Add(time.Second)
	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	activation := notifications.Activation{
		RecipientDID: syntax.DID(recipientDID), ActorDID: syntax.DID(actorDID), Category: notifications.Reply,
		SubjectKey: strings.Join([]string{handleSentinel, textSentinel, titleSentinel, imageSentinel, payloadSentinel}, "|"),
		SourceURI:  syntax.ATURI(sourceURISentinel), SourceCID: "sentinel-source-cid", SourceRkey: "sentinel-source",
		SubjectURI: syntax.ATURI(subjectURISentinel), SubjectCID: "sentinel-subject-cid", ActivityAt: base,
	}
	if err := notifications.NewService(observer).Activate(context.Background(), tx, activation); err != nil {
		_ = tx.Rollback(context.Background())
		t.Fatal(err)
	}
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}

	now := base.Add(time.Second)
	sender := &privacyIntegrationSender{failure: failureText}
	dispatcher := push.NewDispatcher(pool, sender, push.DispatcherOptions{Now: func() time.Time { return now }, BatchSize: 1, LeaseDuration: time.Minute, Observer: observer})
	if n, err := dispatcher.ProcessBatch(context.Background(), "appview"); err != nil || n != 1 {
		t.Fatalf("retry batch n=%d err=%v", n, err)
	}
	now = now.Add(2 * time.Second)
	if n, err := dispatcher.ProcessBatch(context.Background(), "appview"); err != nil || n != 1 {
		t.Fatalf("success batch n=%d err=%v", n, err)
	}
	if !observer.Flush(time.Second) {
		t.Fatal("observer flush failed")
	}

	calls := sender.Calls()
	if len(calls) != 2 || calls[0].Token != tokenSentinel || calls[1].Token != tokenSentinel {
		t.Fatalf("provider boundary calls=%+v", calls)
	}
	payloadBytes, err := json.Marshal(push.BuildPayload(calls[0].Category, calls[0].AccountSubscriptionID, calls[0].ActorDisplayName, calls[0].RoutingFacts))
	if err != nil {
		t.Fatal(err)
	}
	providerPayload := string(payloadBytes)
	for _, required := range []string{registration.AccountSubscriptionID, sourceURISentinel, subjectURISentinel} {
		if !strings.Contains(providerPayload, required) {
			t.Fatalf("provider payload omitted required routing fact %q: %s", required, providerPayload)
		}
	}
	for _, private := range []string{tokenSentinel, credentialSentinel, textSentinel, titleSentinel, imageSentinel, payloadSentinel, providerError} {
		if strings.Contains(providerPayload, private) {
			t.Fatalf("provider payload leaked private value %q: %s", private, providerPayload)
		}
	}
	var providerClass string
	if err := pool.QueryRow(context.Background(), `SELECT provider_result_class FROM push_deliveries`).Scan(&providerClass); err != nil {
		t.Fatal(err)
	}

	// Registration responses and provider payloads intentionally contain the
	// account binding and category-required public routing facts. Telemetry must
	// not copy any of those boundary values.
	observable := stdout.String() + "\n" + providerClass
	for _, call := range recorder.Calls() {
		observable += fmt.Sprintf("\n%+v", call)
	}
	events := transport.Events()
	if len(events) == 0 {
		t.Fatal("real retry/success flow emitted no safe Sentry log evidence")
	}
	for _, event := range events {
		observable += fmt.Sprintf("\n%+v", event.Logs)
	}
	for _, value := range forbidden {
		if strings.Contains(observable, value) {
			t.Fatalf("observable output leaked %q:\n%s", value, observable)
		}
	}
}
