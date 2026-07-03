package observability

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/jackc/pgx/v5"
	"social.craftsky/appview/internal/auth"
)

func TestClassifyErrorUsesBoundedSentinels(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		ctx          EventContext
		wantCategory string
		wantCode     string
	}{
		{name: "craftsky auth token", err: auth.ErrAuthTokenInvalid, wantCategory: "auth", wantCode: "auth.session_invalid"},
		{name: "oauth session missing", err: auth.ErrOAuthSessionNotFound, wantCategory: "auth", wantCode: "auth.oauth_session_not_found"},
		{name: "pds session expired", err: auth.ErrPDSSessionExpired, wantCategory: "auth", wantCode: "auth.pds_session_expired"},
		{name: "record missing", err: auth.ErrRecordNotFound, wantCategory: "not_found", wantCode: "pds.record_not_found"},
		{name: "deadline", err: context.DeadlineExceeded, wantCategory: "timeout", wantCode: "timeout.deadline_exceeded"},
		{name: "network", err: &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused for did:plc:raw")}, wantCategory: "network", wantCode: "network.unavailable"},
		{name: "db not found", err: pgx.ErrNoRows, ctx: EventContext{"component": "db"}, wantCategory: "not_found", wantCode: "db.no_rows"},
		{name: "validation context", err: errors.New("raw validation body did:plc:raw"), ctx: EventContext{"error_category": "validation"}, wantCategory: "validation", wantCode: "appview.validation"},
		{name: "tap context", err: errors.New("raw tap payload did:plc:raw"), ctx: EventContext{"component": "tap"}, wantCategory: "tap", wantCode: "tap.error"},
		{name: "unexpected", err: errors.New("raw upstream text did:plc:raw secret-token"), wantCategory: "unexpected", wantCode: "appview.unexpected"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyError(tt.err, tt.ctx)
			if got.Category != tt.wantCategory || got.Code != tt.wantCode {
				t.Fatalf("ClassifyError = category=%q code=%q, want category=%q code=%q", got.Category, got.Code, tt.wantCategory, tt.wantCode)
			}
			if strings.Contains(got.Message, "did:") || strings.Contains(got.Message, "secret") || strings.Contains(got.Message, "connection refused") {
				t.Fatalf("classifier message includes raw detail: %#v", got)
			}
		})
	}
}

func TestCaptureErrorUsesClassifiedSentinelWithoutRawErrorDetails(t *testing.T) {
	transport := &sentry.MockTransport{}
	observer := New(Config{
		Env:             "test",
		SentryDSN:       "https://public@example.invalid/1",
		SentryTransport: transport,
	})

	observer.CaptureError(context.Background(), EventContext{
		"component":      "http",
		"operation":      "search.posts",
		"failure_stage":  "handler",
		"route_pattern":  "/v1/search/posts",
		"error_category": "validation",
	}, errors.New("raw error included did:plc:raw secret-token request body"))
	if !observer.Flush(50 * time.Millisecond) {
		t.Fatal("Flush returned false")
	}

	events := transport.Events()
	if len(events) != 1 {
		t.Fatalf("captured %d events, want 1", len(events))
	}
	event := events[0]
	if event.Exception[0].Value != "appview.validation" {
		t.Fatalf("exception value = %q, want appview.validation", event.Exception[0].Value)
	}
	if event.Tags["error_code"] != "appview.validation" || event.Tags["error_category"] != "validation" || event.Tags["failure_stage"] != "handler" {
		t.Fatalf("event tags missing classifier fields: %#v", event.Tags)
	}
	for _, forbidden := range []string{"did:plc:raw", "secret-token", "request body", "raw error"} {
		if strings.Contains(event.Message, forbidden) || strings.Contains(event.Exception[0].Value, forbidden) {
			t.Fatalf("event contains forbidden raw value %q: %#v", forbidden, event)
		}
		for key, value := range event.Tags {
			if strings.Contains(key, forbidden) || strings.Contains(value, forbidden) {
				t.Fatalf("event tag contains forbidden raw value %q: %s=%s", forbidden, key, value)
			}
		}
	}
}
