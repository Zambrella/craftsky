package revenuecat

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/subscriptions"
	"social.craftsky/appview/internal/testdb"
)

func TestWebhookHandlerRollsBackInterruptedDurableCommit(t *testing.T) {
	accounts, err := os.ReadFile("../../../migrations/000069_subscription_accounts.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	events, err := os.ReadFile("../../../migrations/000070_revenuecat_events.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, string(accounts)+string(events))
	ctx := context.Background()
	store := subscriptions.NewStore(pool)
	account, _, err := store.EnsureAccount(ctx, syntax.DID("did:plc:webhook-commit-owner"))
	if err != nil {
		t.Fatal(err)
	}
	lock, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := lock.Exec(ctx, `SELECT id FROM billing_accounts WHERE id=$1 FOR UPDATE`, account.ID); err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1788955200, 0)
	metrics := observability.NewInMemoryMetricRecorder()
	var logs bytes.Buffer
	observer := observability.New(observability.Config{MetricRecorder: metrics, Logger: slog.New(slog.NewJSONHandler(&logs, nil))})
	const rawCanary = "raw-receipt-payment-canary"
	const authCanary = "authorization-private-canary"
	const signatureCanary = "signature-private-canary"
	handler, err := NewWebhookHandler(WebhookConfig{
		Authorization: authCanary, SigningSecret: signatureCanary, BodyLimit: 1024,
		IngressDeadline: 20 * time.Millisecond, SignatureTolerance: time.Minute, Observer: observer,
	}, store, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	body := `{"api_version":"1.0","event":{"id":"commit-interrupted","type":"RENEWAL","app_user_id":"` + account.RevenueCatAppUserID.String() + `","event_timestamp_ms":1788955200000,"receipt":"` + rawCanary + `","payment":{"secret":"` + rawCanary + `"}}}`
	timestamp := strconv.FormatInt(now.Unix(), 10)
	mac := hmac.New(sha256.New, []byte(signatureCanary))
	_, _ = mac.Write([]byte(timestamp + "." + body))
	request := httptest.NewRequest(http.MethodPost, "/integrations/revenuecat/webhook", strings.NewReader(body))
	request.Header.Set("Authorization", authCanary)
	request.Header.Set("X-RevenueCat-Webhook-Signature", "t="+timestamp+",v1="+hex.EncodeToString(mac.Sum(nil)))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("interrupted commit response = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
	if err := lock.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	var eventCount int
	var generation int64
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM revenuecat_events`).Scan(&eventCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT requested_generation FROM billing_accounts WHERE id=$1`, account.ID).Scan(&generation); err != nil {
		t.Fatal(err)
	}
	if eventCount != 0 || generation != 0 {
		t.Fatalf("interrupted commit persisted event/generation %d/%d", eventCount, generation)
	}
	prohibitedOutput := logs.String() + response.Body.String()
	for _, canary := range []string{rawCanary, authCanary, signatureCanary, account.RevenueCatAppUserID.String(), "commit-interrupted"} {
		if strings.Contains(prohibitedOutput, canary) {
			t.Fatalf("public boundary output leaked %q: %s", canary, prohibitedOutput)
		}
		for _, call := range metrics.Calls() {
			for _, value := range call.Attributes {
				if strings.Contains(value, canary) {
					t.Fatalf("metric leaked %q: %#v", canary, call)
				}
			}
		}
	}
}
