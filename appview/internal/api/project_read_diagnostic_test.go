package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/jackc/pgx/v5/pgconn"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/observability"
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

func TestPublicProfileIdentityFailureRetainsResolverCause(t *testing.T) {
	const route = "/v1/public/profiles/{handleOrDid}"
	var local bytes.Buffer
	logger := slog.New(observability.NewDiagnosticHandler(slog.NewJSONHandler(&local, nil)))
	transport := &sentry.MockTransport{}
	observer := observability.New(observability.Config{Env: "test", Logger: logger, SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
	store := &publicPageStub{}
	resolver := fakeResolver{err: &net.DNSError{
		Err: "PRIVATE_DNS_REASON_CANARY", Name: "PRIVATE_HOST_CANARY", IsTimeout: true,
	}}
	mux := http.NewServeMux()
	mux.Handle("GET "+route, api.PublicPagesHandler(store, resolver, "profile"))
	handler := middleware.Logging(logger)(middleware.HTTPMetrics(observer)(mux))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/v1/public/profiles/protected-handle.example", nil))
	observer.Flush(time.Second)

	events := transport.Events()
	if rec.Code != 503 || !strings.Contains(rec.Body.String(), "identity_unavailable") || store.calls != 0 || len(events) != 1 {
		t.Fatalf("response/ownership changed: status=%d reads=%d issues=%d", rec.Code, store.calls, len(events))
	}
	if events[0].Tags["operation"] != "api.PublicPagesHandler" || events[0].Tags["failure_stage"] != "identity" {
		t.Fatalf("missing profile identity stage: %v", events[0].Tags)
	}
	assertPublicReadTelemetry(t, local.String(), events, rec.Header().Get("X-Request-ID"), "DNS lookup failed", "protected-handle.example")
}

func TestPublicProjectQueryTimeoutIdentifiesCandidateStage(t *testing.T) {
	const queryCanary = "protected-query-canary"
	pool, store := publicReadFixture(t)
	tx, err := pool.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(t.Context())
	if _, err := tx.Exec(t.Context(), "LOCK TABLE craftsky_project_posts IN ACCESS EXCLUSIVE MODE"); err != nil {
		t.Fatal(err)
	}
	var local bytes.Buffer
	logger := slog.New(observability.NewDiagnosticHandler(slog.NewJSONHandler(&local, nil)))
	transport := &sentry.MockTransport{}
	observer := observability.New(observability.Config{Env: "test", Logger: logger, SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
	handler := middleware.Logging(logger)(middleware.HTTPMetrics(observer)(api.PublicBrowsingHandler(store, nil, "search/projects")))
	ctx, cancel := context.WithTimeout(t.Context(), 300*time.Millisecond)
	defer cancel()
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/v1/public/search/projects?q="+queryCanary, nil).WithContext(ctx))
	observer.Flush(time.Second)

	events := transport.Events()
	if rec.Code != 503 || !strings.Contains(rec.Body.String(), "unavailable") || len(events) != 1 {
		t.Fatalf("response/ownership changed: status=%d issues=%d", rec.Code, len(events))
	}
	if events[0].Tags["error_category"] != "timeout" {
		t.Fatalf("database deadline lost: %v", events[0].Tags)
	}
	assertPublicReadTelemetry(t, local.String(), events, rec.Header().Get("X-Request-ID"), "load public project search candidates", queryCanary)
}

func assertPublicReadTelemetry(t *testing.T, local string, events []*sentry.Event, requestID, fact, canary string) {
	t.Helper()
	serialized, err := json.Marshal(events)
	if err != nil {
		t.Fatal(err)
	}
	if requestID == "" || events[0].Contexts["correlation"]["run_id"] != requestID {
		t.Fatal("request correlation lost")
	}
	for _, sink := range []string{local, string(serialized)} {
		if !strings.Contains(sink, fact) || !strings.Contains(sink, requestID) {
			t.Fatalf("missing diagnostic fact %q or request ID", fact)
		}
		for _, protected := range []string{canary, "PRIVATE_DNS_REASON_CANARY", "PRIVATE_HOST_CANARY"} {
			if strings.Contains(sink, protected) {
				t.Fatalf("protected value escaped: %q", protected)
			}
		}
	}
}
