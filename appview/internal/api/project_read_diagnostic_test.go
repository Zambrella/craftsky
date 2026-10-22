package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/getsentry/sentry-go"
	"github.com/jackc/pgx/v5/pgconn"
	"log/slog"
	"net"
	"net/http/httptest"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/observability"
	"strings"
	"testing"
	"time"
)

func TestProjectReadRecurrenceRetainsStageAndRequestCorrelation(t *testing.T) {
	for _, stage := range []string{"relationship", "identity"} {
		t.Run(stage, func(t *testing.T) {
			var local bytes.Buffer
			logger := slog.New(observability.NewDiagnosticHandler(slog.NewJSONHandler(&local, nil)))
			transport := &sentry.MockTransport{}
			observer := observability.New(observability.Config{Env: "test", Logger: logger, SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
			store, resolver := &fakePostStore{}, fakeResolver{}
			wantStatus, wantCode, wantType := 500, "internal_error", "PgError"
			if stage == "relationship" {
				store.relationshipStateErr = &pgconn.PgError{Code: "08006", Message: "private-query-canary"}
			} else {
				store.projectListRows = []*api.PostRow{{URI: "at://did:plc:alice/social.craftsky.feed.post/p", DID: "did:plc:alice"}}
				resolver.err = &net.DNSError{Err: "private-query-canary", Name: "private-host-canary"}
				wantStatus, wantCode, wantType = 502, "identity_unavailable", "DNSError"
			}
			handler := middleware.Logging(logger)(middleware.HTTPMetrics(observer)(api.ListProjectsByAuthorHandler(store, resolver, logger, nil)))
			req := authedReq("GET", "/v1/profiles/@did:plc:alice/projects", "", "did:plc:viewer")
			req.SetPathValue("handleOrDid", "did:plc:alice")
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			observer.Flush(time.Second)
			events := transport.Events()
			requestID := rec.Header().Get("X-Request-ID")
			if rec.Code != wantStatus || !strings.Contains(rec.Body.String(), wantCode) || requestID == "" || !strings.Contains(rec.Body.String(), requestID) || len(events) != 1 {
				t.Fatalf("response/ownership changed: %d issues=%d", rec.Code, len(events))
			}
			if events[0].Tags["operation"] != "project.author.list" || events[0].Tags["failure_stage"] != stage || events[0].Contexts["correlation"]["run_id"] != requestID {
				t.Fatalf("lost context: %v", events[0].Tags)
			}
			serialized, err := json.Marshal(events)
			if err != nil {
				t.Fatal(err)
			}
			for _, sink := range []string{local.String(), string(serialized)} {
				for _, want := range []string{requestID, "project.author.list", stage, wantType} {
					if !strings.Contains(sink, want) {
						t.Fatalf("missing %q", want)
					}
				}
				for _, private := range []string{"private-query-canary", "private-host-canary"} {
					if strings.Contains(sink, private) {
						t.Fatalf("protected value escaped: %s", private)
					}
				}
			}
		})
	}
}

func TestMentionIdentityFailureRetainsCauseAndSingleIssue(t *testing.T) {
	var local bytes.Buffer
	logger := slog.New(observability.NewDiagnosticHandler(slog.NewJSONHandler(&local, nil)))
	transport := &sentry.MockTransport{}
	observer := observability.New(observability.Config{Env: "test", Logger: logger, SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
	cause := fmt.Errorf("%w: %w", api.ErrMentionIdentityUnavailable, &net.DNSError{Err: "PRIVATE_DNS_REASON", Name: "PRIVATE_HOST"})
	handler := middleware.Logging(logger)(middleware.HTTPMetrics(observer)(api.ResolveFacetMentionHandler(fakeFacetMentionResolver{err: cause}, logger)))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, authedReq("GET", "/v1/facets/mentions/resolve?handle=private.example", "", "did:plc:viewer"))
	observer.Flush(time.Second)
	events := transport.Events()
	if rec.Code != 503 || len(events) != 1 {
		t.Fatalf("response/ownership changed: status=%d issues=%d", rec.Code, len(events))
	}
	if events[0].Tags["operation"] != "facet.mention.resolve" || events[0].Tags["failure_stage"] != "identity" {
		t.Fatal("missing stage")
	}
	payload, _ := json.Marshal(events)
	for _, sink := range []string{local.String(), string(payload)} {
		if !strings.Contains(sink, "DNSError") || !strings.Contains(sink, rec.Header().Get("X-Request-ID")) {
			t.Fatal("cause/correlation lost")
		}
		for _, private := range []string{"PRIVATE_DNS_REASON", "PRIVATE_HOST", "private.example"} {
			if strings.Contains(sink, private) {
				t.Fatal("protected value escaped")
			}
		}
	}
}
