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

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/getsentry/sentry-go"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/routes"
)

// IR-002 / IT-017: seed loading consumes identity errors and must retain their
// original typed chain and available stack at one protected request owner.
func TestEditPostSeedIdentityFailureDiagnostics(t *testing.T) {
	testEditPostSeedFailureDiagnostics(t, false)
}
func TestEditPostSeedPDSFailureDiagnostics(t *testing.T) {
	testEditPostSeedFailureDiagnostics(t, true)
}
func testEditPostSeedFailureDiagnostics(t *testing.T, pdsFailure bool) {
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
	resolver := fakeResolver{err: cause}
	stage := "identity"
	status := 502
	code := "identity_unavailable"
	load := func(context.Context, syntax.DID, syntax.RecordKey, int64, string) (*api.PostEditSnapshot, error) {
		return store.snapshot, store.err
	}
	if pdsFailure {
		resolver = fakeResolver{handleFor: "alice.example"}
		stage = "query"
		status = 503
		code = "pds_unavailable"
		load = func(context.Context, syntax.DID, syntax.RecordKey, int64, string) (*api.PostEditSnapshot, error) {
			return nil, cause
		}
	}
	handler := middleware.Logging(logger, catalogue)(middleware.HTTPMetrics(observer)(api.GetPostEditHandler(store, resolver, load)))
	request := httptest.NewRequest(http.MethodGet, "/v1/posts/did:plc:alice/3existingpost/edit", nil)
	request.SetPathValue("did", "did:plc:alice")
	request.SetPathValue("rkey", "3existingpost")
	ctx := middleware.WithDID(request.Context(), "did:plc:alice")
	ctx = middleware.WithOwnerGeneration(ctx, 1)
	ctx = middleware.WithOAuthSessionID(ctx, "PRIVATE_SESSION_CANARY")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request.WithContext(ctx))
	observer.Flush(time.Second)
	requestID := response.Header().Get("X-Request-ID")
	if response.Code != status || !strings.Contains(response.Body.String(), `"error":"`+code+`"`) || requestID == "" || !strings.Contains(response.Body.String(), requestID) {
		t.Fatalf("response changed: %d %s", response.Code, response.Body.String())
	}
	events := transport.Events()
	if len(events) != 1 {
		t.Fatalf("issues=%d, want one original-cause owner; logs=%s", len(events), logs.String())
	}
	if events[0].Tags["operation"] != "api.GetPostEditHandler" || events[0].Tags["failure_stage"] != stage {
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
		for _, positive := range []string{requestID, "api.GetPostEditHandler", stage, "net.DNSError", "did:plc:alice"} {
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
