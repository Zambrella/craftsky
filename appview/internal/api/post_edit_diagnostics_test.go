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
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/jackc/pgx/v5/pgconn"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/pdscommands"
	"social.craftsky/appview/internal/routes"
)

// IR-002 / IT-017: seed loading consumes identity errors and must retain their
// original typed chain and available stack at one protected request owner.
func TestEditPostSeedIdentityFailureDiagnostics(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(observability.NewDiagnosticHandler(slog.NewJSONHandler(&logs, nil)))
	transport := &sentry.MockTransport{}
	observer := observability.New(observability.Config{Env: "test", SentryDSN: "https://public@example.invalid/1", SentryTransport: transport, Logger: logger})
	catalogue, err := routes.NewV1Catalogue(routes.V1RoutePolicies(routes.EnvProd, routes.Config{}))
	if err != nil {
		t.Fatal(err)
	}
	pcs := make([]uintptr, 32)
	pcs = pcs[:runtime.Callers(1, pcs)]
	cause := &postEditSeedIdentityFailure{
		cause: fmt.Errorf("PRIVATE_EDIT_CANARY Bearer CREDENTIAL_CANARY: %w", &net.DNSError{Err: "PRIVATE_DNS_CANARY", Name: "PRIVATE_HOST_CANARY"}),
		pcs:   pcs,
	}
	store := &seededPostEditStore{fakePostStore: &fakePostStore{}, snapshot: &api.PostEditSnapshot{
		Post:   &api.PostRow{URI: "at://did:plc:alice/social.craftsky.feed.post/3existingpost", DID: "did:plc:alice", Rkey: "3existingpost", Text: "PRIVATE_EDIT_CANARY"},
		Record: json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"PRIVATE_EDIT_CANARY","sponsored":false,"createdAt":"2026-01-01T00:00:00Z"}`),
	}}
	handler := middleware.Logging(logger, catalogue)(middleware.HTTPMetrics(observer)(api.GetPostEditHandler(store, fakeResolver{err: cause})))
	request := httptest.NewRequest(http.MethodGet, "/v1/posts/did:plc:alice/3existingpost/edit", nil)
	request.SetPathValue("did", "did:plc:alice")
	request.SetPathValue("rkey", "3existingpost")
	ctx := middleware.WithDID(request.Context(), "did:plc:alice")
	ctx = middleware.WithOAuthSessionID(ctx, "PRIVATE_SESSION_CANARY")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request.WithContext(ctx))
	observer.Flush(time.Second)
	requestID := response.Header().Get("X-Request-ID")
	if response.Code != 502 || !strings.Contains(response.Body.String(), `"error":"identity_unavailable"`) || requestID == "" || !strings.Contains(response.Body.String(), requestID) {
		t.Fatalf("response changed: %d %s", response.Code, response.Body.String())
	}
	events := transport.Events()
	if len(events) != 1 {
		t.Fatalf("issues=%d, want one original-cause owner; logs=%s", len(events), logs.String())
	}
	if events[0].Tags["operation"] != "api.GetPostEditHandler" || events[0].Tags["failure_stage"] != "identity" {
		t.Fatalf("missing seed failure context: %+v", events[0].Tags)
	}
	hasStack := false
	for _, exception := range events[0].Exception {
		if exception.Stacktrace != nil && len(exception.Stacktrace.Frames) > 0 {
			hasStack = true
		}
	}
	if !hasStack {
		t.Fatal("original available stack lost")
	}
	serialized, err := json.Marshal(events)
	if err != nil {
		t.Fatal(err)
	}
	for _, sink := range []string{logs.String(), string(serialized)} {
		for _, positive := range []string{requestID, "api.GetPostEditHandler", "identity", "net.DNSError", "did:plc:alice"} {
			if !strings.Contains(sink, positive) {
				t.Fatalf("diagnostic lost %s: %s", positive, sink)
			}
		}
	}
	combined := logs.String() + string(serialized) + response.Body.String()
	for _, canary := range []string{"PRIVATE_EDIT_CANARY", "CREDENTIAL_CANARY", "PRIVATE_DNS_CANARY", "PRIVATE_HOST_CANARY", "PRIVATE_SESSION_CANARY"} {
		if strings.Contains(combined, canary) {
			t.Fatalf("protected value reached sink: %s", canary)
		}
	}
}

type postEditSeedIdentityFailure struct {
	cause error
	pcs   []uintptr
}

func (e *postEditSeedIdentityFailure) Error() string         { return e.cause.Error() }
func (e *postEditSeedIdentityFailure) Unwrap() error         { return e.cause }
func (e *postEditSeedIdentityFailure) StackTrace() []uintptr { return e.pcs }

// IT-017: a response-only read failure cannot roll back accepted publication,
// but unexpected failures keep their cause at one protected diagnostic owner.
func TestEditPostAcceptedRefreshFailureDiagnostics(t *testing.T) {
	unexpected := func() error {
		return fmt.Errorf("PRIVATE_EDIT_CANARY Bearer CREDENTIAL_CANARY: %w", &pgconn.PgError{Code: "08006", Message: "PRIVATE_DATABASE_CANARY"})
	}
	for _, tc := range []struct {
		name   string
		stage  string
		err    error
		issues int
	}{
		{"unexpected", "query", unexpected(), 1},
		{"identity", "identity", unexpected(), 1},
		{"engagement", "engagement", unexpected(), 1},
		{"quote", "quote", unexpected(), 1},
		{"pending", "query", api.ErrPostEditPending, 0},
		{"unavailable", "query", api.ErrPostNotFound, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var logs bytes.Buffer
			transport := &sentry.MockTransport{}
			logger := slog.New(slog.NewJSONHandler(&logs, nil))
			observer := observability.New(observability.Config{Env: "test", SentryDSN: "https://public@example.invalid/1", SentryTransport: transport, Logger: logger})
			catalogue, err := routes.NewV1Catalogue(routes.V1RoutePolicies(routes.EnvProd, routes.Config{}))
			if err != nil {
				t.Fatal(err)
			}
			commands := postEditStoredResultExecutor{}
			store := &seededPostEditStore{fakePostStore: &fakePostStore{}, snapshot: &api.PostEditSnapshot{
				Post: &api.PostRow{URI: "at://did:plc:alice/social.craftsky.feed.post/3existingpost", DID: "did:plc:alice", Rkey: "3existingpost", CID: "bafyreif7dhriwzuzrl3gsrxjo3mzxt5jdls7yo3jwoashfx74og6n3ck4m", Text: "PRIVATE_EDIT_CANARY"},
			}}
			resolver := fakeResolver{handleFor: "alice.example"}
			switch tc.stage {
			case "query":
				store.err = tc.err
			case "identity":
				resolver.err = tc.err
			case "engagement":
				store.engagementErr = tc.err
			case "quote":
				uri, cid := "at://did:plc:quoted/social.craftsky.feed.post/quoted", "bafyquoted"
				store.snapshot.Post.QuoteURI = &uri
				store.snapshot.Post.QuoteCID = &cid
				store.quoteViewsErr = tc.err
			}
			handler := middleware.Logging(logger, catalogue)(middleware.HTTPMetrics(observer)(api.CommandEditPostHandler(store, commands, resolver, api.DefaultMediaLimits())))
			request := httptest.NewRequest(http.MethodPatch, "/v1/posts/did:plc:alice/3existingpost", strings.NewReader(`{"changes":{"text":"PRIVATE_EDIT_CANARY"}}`))
			request.SetPathValue("did", "did:plc:alice")
			request.SetPathValue("rkey", "3existingpost")
			request.Header.Set("If-Match", "bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq")
			request.Header.Set("Idempotency-Key", "018f4d5c-7a61-7d40-a1a2-777777777777")
			ctx := middleware.WithDID(request.Context(), "did:plc:alice")
			ctx = middleware.WithOwnerGeneration(ctx, 1)
			ctx = middleware.WithOAuthSessionID(ctx, "PRIVATE_SESSION_CANARY")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request.WithContext(ctx))
			observer.Flush(time.Second)
			var result api.PostEditResult
			if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if response.Code != 200 || !result.ProjectionPending || result.Post != nil || !result.Changed {
				t.Fatalf("accepted result changed: %d %s", response.Code, response.Body.String())
			}
			events := transport.Events()
			if len(events) != tc.issues {
				t.Fatalf("issues=%d, want=%d logs=%s", len(events), tc.issues, logs.String())
			}
			serialized, _ := json.Marshal(events)
			combined := logs.String() + string(serialized)
			if tc.issues > 0 {
				for _, positive := range []string{"08006", "pgconn.PgError", "did:plc:alice", "at://did:plc:alice/social.craftsky.feed.post/3existingpost", "accepted_refresh_" + tc.stage} {
					if !strings.Contains(combined, positive) {
						t.Fatalf("diagnostic lost %s: %s", positive, combined)
					}
				}
			}
			for _, canary := range []string{"PRIVATE_EDIT_CANARY", "CREDENTIAL_CANARY", "PRIVATE_DATABASE_CANARY", "PRIVATE_SESSION_CANARY"} {
				if strings.Contains(combined, canary) {
					t.Fatalf("protected value reached sink: %s", canary)
				}
			}
		})
	}
}

type postEditStoredResultExecutor struct{}

func (postEditStoredResultExecutor) Put(context.Context, pdscommands.AddressedPutCommandRequest) (pdscommands.CommandResult, error) {
	return pdscommands.CommandResult{TerminalResult: pdscommands.TerminalResult{State: pdscommands.CommandAccepted, HTTPStatus: 200, ResponseBody: []byte(`{"post":null,"changed":true,"acceptedCid":"bafyreif7dhriwzuzrl3gsrxjo3mzxt5jdls7yo3jwoashfx74og6n3ck4m","editedAt":"2026-10-26T12:00:00Z","editTimeSource":"confirmed","projectionPending":true}`)}}, nil
}
