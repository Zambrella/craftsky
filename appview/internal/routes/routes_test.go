package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/business"
	"social.craftsky/appview/internal/ctxkeys"
	"social.craftsky/appview/internal/languages"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/pdseffects"
	"social.craftsky/appview/internal/testdb"
	"social.craftsky/appview/internal/testlog"
)

const routeModerationDDL = `
CREATE TABLE moderation_outputs (
    id                 TEXT PRIMARY KEY,
    source_did         TEXT NOT NULL,
    subject_type       TEXT NOT NULL,
    subject_did        TEXT NOT NULL,
    subject_collection TEXT,
    subject_rkey       TEXT,
    subject_uri        TEXT,
    value              TEXT NOT NULL,
    action             TEXT NOT NULL,
    internal_reason    TEXT,
    expires_at         TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL,
    indexed_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE moderation_idempotency_receipts (
    request_key_hash BYTEA PRIMARY KEY,
    request_fingerprint BYTEA NOT NULL,
    output_id TEXT NOT NULL,
    output_status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE owner_lifecycles (
    owner_did TEXT PRIMARY KEY,
    state TEXT NOT NULL,
    generation BIGINT NOT NULL,
    auth_epoch BIGINT NOT NULL,
    transition_reason TEXT NOT NULL,
    transitioned_at TIMESTAMPTZ NOT NULL,
    terminal_at TIMESTAMPTZ,
    purge_completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);
INSERT INTO owner_lifecycles(
    owner_did,state,generation,auth_epoch,transition_reason,
    transitioned_at,created_at,updated_at
) VALUES
    ('did:plc:labeler','active',1,1,'test',now(),now(),now()),
    ('did:plc:bob','active',1,1,'test',now(),now(),now());
`

type routeTimeoutBody struct{}

func (routeTimeoutBody) Read([]byte) (int, error) { return 0, routeTimeoutError{} }
func (routeTimeoutBody) Close() error             { return nil }

type routeTimeoutError struct{}

func (routeTimeoutError) Error() string   { return "request body timed out" }
func (routeTimeoutError) Timeout() bool   { return true }
func (routeTimeoutError) Temporary() bool { return true }

func newRouteModerationStore(t *testing.T, pool *pgxpool.Pool) *api.ModerationStore {
	t.Helper()
	lifecycles := newRouteOwnerLifecycleStore(t, pool)
	store, err := api.NewModerationStore(pool, lifecycles)
	if err != nil {
		t.Fatalf("moderation store: %v", err)
	}
	return store
}

func newRouteOwnerLifecycleStore(t *testing.T, pool *pgxpool.Pool) *ownerlifecycle.Store {
	t.Helper()
	fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		t.Fatalf("owner fencer: %v", err)
	}
	lifecycles, err := ownerlifecycle.NewStore(pool, fencer, time.Now)
	if err != nil {
		t.Fatalf("owner lifecycle store: %v", err)
	}
	return lifecycles
}

// stubResolver is a minimal api.HandleResolver used by the routing
// tests so they don't depend on the real PLC directory.
type stubResolver struct{ handle syntax.Handle }

func (s stubResolver) ResolveHandle(_ context.Context, _ syntax.DID) (syntax.Handle, error) {
	return s.handle, nil
}
func (s stubResolver) ResolveDID(_ context.Context, _ syntax.Handle) (syntax.DID, error) {
	return "", nil
}

var _ api.HandleResolver = stubResolver{}

type routeAccountTypeReader struct{}

func (routeAccountTypeReader) ReadAccountTypes(_ context.Context, dids []syntax.DID) (map[syntax.DID]business.AccountType, error) {
	values := make(map[syntax.DID]business.AccountType, len(dids))
	for _, did := range dids {
		values[did] = business.AccountTypeBusiness
	}
	return values, nil
}

type recordingRegistrationFlow struct {
	registrationCalls int
	mode              auth.HandoffMode
	loopbackURI       string
	deviceID          string
	registrationErr   error
}

func (*recordingRegistrationFlow) StartLogin(context.Context, syntax.Handle, auth.HandoffMode, string, string) (string, error) {
	return "", nil
}

func (flow *recordingRegistrationFlow) StartRegistration(_ context.Context, mode auth.HandoffMode, loopbackURI, deviceID string) (string, error) {
	flow.registrationCalls++
	flow.mode = mode
	flow.loopbackURI = loopbackURI
	flow.deviceID = deviceID
	return "https://auth.example/authorize?request_uri=urn%3Aexample%3Apar", flow.registrationErr
}

func (*recordingRegistrationFlow) CompleteCallback(context.Context, url.Values, auth.OAuthCallbackFinalizer) error {
	return nil
}

func testDeps() *Dependencies {
	return &Dependencies{
		Config:           Config{Env: EnvDev, AllowedOrigins: []string{"*"}},
		Logger:           testlog.Discard(),
		AuthService:      &auth.MockAuthService{DefaultDID: "did:plc:test"},
		HandleResolver:   stubResolver{handle: syntax.Handle("stub-handle.example")},
		SuspensionReader: unsuspendedReader{},
	}
}

func TestAdminModerationRoutesUseOnlyDedicatedModeratorAuthentication(t *testing.T) {
	const sensitive = "SENTINEL_ADMIN_CREDENTIAL"
	var logOutput bytes.Buffer
	recorder := observability.NewInMemoryMetricRecorder()
	sink := &routeModerationLogSink{}
	observer := observability.New(observability.Config{
		SentryDSN: "https://public@example.invalid/1", LogsEnabled: true, MetricsEnabled: true,
		MetricRecorder: recorder, LogSink: sink, Logger: slog.New(slog.NewJSONHandler(&logOutput, nil)),
	})
	deps := testDeps()
	deps.Config.ModerationAdminEnabled = true
	deps.Config.ModerationAdminToken = Secret(sensitive)
	deps.Config.ModerationAdminActorID = "operator-1"
	deps.Config.ModerationAdminSourceSystem = "admin-api"
	deps.Config.EnableDevModeration = true
	deps.Config.DevModerationToken = "SENTINEL_DEVELOPMENT_SECRET"
	deps.Observability = observer
	deps.Logger = slog.New(slog.NewJSONHandler(&logOutput, nil))
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, deps)

	denials := []struct {
		name          string
		authorization string
		devToken      string
	}{
		{name: "missing"},
		{name: "member", authorization: "Bearer SENTINEL_MEMBER_SESSION"},
		{name: "development", devToken: "SENTINEL_DEVELOPMENT_SECRET"},
		{name: "revoked", authorization: "Bearer SENTINEL_REVOKED_ADMIN_SECRET"},
		{name: "malformed", authorization: sensitive},
	}
	for index, test := range denials {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/v1/admin/moderation/cases", nil)
			request.Header.Set("Authorization", test.authorization)
			request.Header.Set("X-Craftsky-Dev-Moderation-Token", test.devToken)
			request.Header.Set("X-Dev-DID", "did:plc:SENTINEL_MEMBER_DID")
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			wantBody := `{"error":"moderator_authentication_failed","message":"moderator authentication failed","requestId":""}` + "\n"
			if response.Code != http.StatusUnauthorized || response.Body.String() != wantBody {
				t.Fatalf("status/body = %d/%s", response.Code, response.Body.String())
			}
			failures := moderationMetricCalls(recorder.Calls(), "craftsky_appview_moderation_admin_auth_total", map[string]string{"result": "failure"})
			if failures != index+1 {
				t.Fatalf("failure metrics after denial %d = %d", index+1, failures)
			}
		})
	}
	if len(sink.events) != 5 || sink.events[3].level != slog.LevelWarn || sink.events[4].level != slog.LevelError {
		t.Fatalf("auth threshold log levels = %#v, want four warnings then an error", sink.events)
	}

	request := httptest.NewRequest(http.MethodGet, "/v1/admin/moderation/cases?state=closed", nil)
	request.Header.Set("Authorization", "Bearer "+sensitive)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"error":"invalid_request"`) {
		t.Fatalf("current admin credential status/body = %d/%s", response.Code, response.Body.String())
	}
	if got := moderationMetricCalls(recorder.Calls(), "craftsky_appview_moderation_admin_auth_total", map[string]string{"result": "success"}); got != 1 {
		t.Fatalf("success metrics = %d, want 1", got)
	}
	for _, call := range recorder.Calls() {
		if call.Name == "craftsky_appview_moderation_admin_auth_total" && len(call.Attributes) != 1 {
			t.Fatalf("admin auth metric attributes = %#v, want bounded result only", call.Attributes)
		}
	}
	observable := logOutput.String() + response.Body.String()
	for _, event := range sink.events {
		observable += event.message
		for key, value := range event.attrs {
			observable += key + fmt.Sprint(value)
		}
	}
	for _, value := range []string{sensitive, "SENTINEL_MEMBER_SESSION", "SENTINEL_DEVELOPMENT_SECRET", "SENTINEL_REVOKED_ADMIN_SECRET", "SENTINEL_MEMBER_DID", "did:plc:"} {
		if strings.Contains(observable, value) {
			t.Fatalf("admin route response/log telemetry leaked %q: %s", value, observable)
		}
	}
}

type routeModerationLogEvent struct {
	level   slog.Level
	message string
	attrs   observability.EventContext
}

type routeModerationLogSink struct{ events []routeModerationLogEvent }

func (s *routeModerationLogSink) Emit(_ context.Context, level slog.Level, message string, attrs observability.EventContext) {
	s.events = append(s.events, routeModerationLogEvent{level: level, message: message, attrs: attrs})
}

func moderationMetricCalls(calls []observability.MetricCall, name string, attrs map[string]string) int {
	count := 0
	for _, call := range calls {
		if call.Name != name || len(call.Attributes) != len(attrs) {
			continue
		}
		matches := true
		for key, value := range attrs {
			matches = matches && call.Attributes[key] == value
		}
		if matches {
			count++
		}
	}
	return count
}

type unsuspendedReader struct{}

func (unsuspendedReader) IsSuspended(context.Context, syntax.DID) (bool, error) { return false, nil }

func TestV1MiddlewareHydratesIdentityAccountType(t *testing.T) {
	observer := observability.New(observability.Config{Env: "test"})
	mw := v1Middleware{
		bodyLimit:           middleware.BodyLimitConfig{DefaultJSONBytes: defaultJSONBodyLimitBytes},
		observer:            observer,
		accountTypeHydrator: api.NewIdentityAccountTypeHydrator(routeAccountTypeReader{}),
	}
	handler := mw.wrap(RoutePolicy{AccessClass: AccessAnonymous, BodyKind: BodyNoBody}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"author":{"did":"did:plc:business","handle":"business.test"}}`))
	}))

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/test", nil))
	var body struct {
		Author struct {
			AccountType string `json:"accountType"`
		} `json:"author"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Author.AccountType != "business" {
		t.Fatalf("author accountType = %q, want business; body=%s", body.Author.AccountType, response.Body.String())
	}
}

func TestProviderRegistrationRouteContract(t *testing.T) {
	flow := &recordingRegistrationFlow{}
	deps := testDeps()
	deps.OAuthFlow = flow
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, deps)

	request := httptest.NewRequest(http.MethodPost, "/v1/auth/registrations", strings.NewReader(
		`{"handoffMode":"verified_link"}`,
	))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Craftsky-Device-Id", "device-registration-route")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", response.Code, response.Body.String())
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", response.Header().Get("Cache-Control"))
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body) != 1 || body["authUrl"] != "https://auth.example/authorize?request_uri=urn%3Aexample%3Apar" {
		t.Fatalf("response = %#v, want exact authUrl body", body)
	}
	if flow.registrationCalls != 1 || flow.mode != auth.HandoffVerifiedLink || flow.loopbackURI != "" || flow.deviceID != "device-registration-route" {
		t.Fatalf("registration call = count %d mode %q loopback %q device %q", flow.registrationCalls, flow.mode, flow.loopbackURI, flow.deviceID)
	}

	unversioned := httptest.NewRecorder()
	mux.ServeHTTP(unversioned, httptest.NewRequest(http.MethodPost, "/auth/registrations", strings.NewReader(`{"handoffMode":"verified_link"}`)))
	if unversioned.Code != http.StatusNotFound {
		t.Fatalf("unversioned status = %d, want 404", unversioned.Code)
	}

	providerOverride := httptest.NewRequest(http.MethodPost, "/v1/auth/registrations", strings.NewReader(
		`{"handoffMode":"verified_link","provider":"https://attacker.example"}`,
	))
	providerOverride.Header.Set("Content-Type", "application/json")
	providerOverride.Header.Set("X-Craftsky-Device-Id", "device-registration-route")
	providerOverride = providerOverride.WithContext(ctxkeys.WithRunID(providerOverride.Context(), "registration-route-request"))
	rejected := httptest.NewRecorder()
	mux.ServeHTTP(rejected, providerOverride)
	if rejected.Code != http.StatusBadRequest {
		t.Fatalf("provider override status = %d, want 400; body=%s", rejected.Code, rejected.Body.String())
	}
	var problem envelope.Error
	if err := json.Unmarshal(rejected.Body.Bytes(), &problem); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if problem.Error != "invalid_body" || problem.Message == "" || problem.RequestID == "" {
		t.Fatalf("error envelope = %+v, want invalid_body with message and requestId", problem)
	}
	if flow.registrationCalls != 1 {
		t.Fatalf("registration calls = %d after provider override, want 1", flow.registrationCalls)
	}
}

func TestProviderRegistrationRouteUsesAuthRateLimitAndCapacityOutcome(t *testing.T) {
	flow := &recordingRegistrationFlow{}
	deps := testDeps()
	deps.OAuthFlow = flow
	deps.RateLimiter = middleware.NewLocalRateLimiter(middleware.RateLimitConfig{Classes: map[middleware.RateClass]middleware.ClassLimit{
		middleware.RateClassAuth: {Window: time.Minute, PerDevice: 1},
	}}, func() time.Time { return time.Unix(100, 0) })
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, deps)

	post := func(deviceID string) *httptest.ResponseRecorder {
		t.Helper()
		request := httptest.NewRequest(http.MethodPost, "/v1/auth/registrations", strings.NewReader(
			`{"handoffMode":"verified_link"}`,
		))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("X-Craftsky-Device-Id", deviceID)
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		return response
	}

	if first := post("registration-limited-device"); first.Code != http.StatusOK {
		t.Fatalf("first status = %d, want 200; body=%s", first.Code, first.Body.String())
	}
	limited := post("registration-limited-device")
	if limited.Code != http.StatusTooManyRequests {
		t.Fatalf("limited status = %d, want 429; body=%s", limited.Code, limited.Body.String())
	}
	retryAfter, err := strconv.Atoi(limited.Header().Get("Retry-After"))
	if err != nil || retryAfter < 1 || retryAfter > 60 {
		t.Fatalf("limited Retry-After = %q, want 1..60 seconds", limited.Header().Get("Retry-After"))
	}
	if flow.registrationCalls != 1 {
		t.Fatalf("provider calls after rate rejection = %d, want 1", flow.registrationCalls)
	}

	flow.registrationErr = auth.ErrAuthRequestCapacity
	capacity := post("registration-capacity-device")
	if capacity.Code != http.StatusServiceUnavailable {
		t.Fatalf("capacity status = %d, want 503; body=%s", capacity.Code, capacity.Body.String())
	}
	var problem envelope.Error
	if err := json.Unmarshal(capacity.Body.Bytes(), &problem); err != nil {
		t.Fatalf("decode capacity envelope: %v", err)
	}
	if problem.Error != "authentication_capacity_exhausted" {
		t.Fatalf("capacity error = %q, want authentication_capacity_exhausted", problem.Error)
	}
	capacityRetry, err := strconv.Atoi(capacity.Header().Get("Retry-After"))
	if err != nil || capacityRetry < 1 || capacityRetry > 60 {
		t.Fatalf("capacity Retry-After = %q, want 1..60 seconds", capacity.Header().Get("Retry-After"))
	}
}

func TestAddRoutes_V1WhoAmIAuthenticatedReturnsDIDAndHandle(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest("GET", "/v1/whoami", nil)
	req.Header.Set("Authorization", "Bearer anything")
	req.Header.Set("X-Dev-DID", "did:plc:from-header")
	req.Header.Set("X-Craftsky-Device-Id", "dev-test")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body struct {
		DID    string `json:"did"`
		Handle string `json:"handle"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body not valid JSON: %v", err)
	}
	if body.DID != "did:plc:from-header" {
		t.Errorf("body.did = %q, want did:plc:from-header", body.DID)
	}
	if body.Handle != "stub-handle.example" {
		t.Errorf("body.handle = %q, want stub-handle.example", body.Handle)
	}
	var topLevel map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &topLevel); err != nil {
		t.Fatalf("body not valid JSON object: %v", err)
	}
	if _, ok := topLevel["data"]; ok {
		t.Fatalf("success response has synthetic data wrapper: %s", rec.Body.String())
	}
}

func TestAddRoutes_V1WhoAmIWithoutAuthReturns401(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest("GET", "/v1/whoami", nil)
	req.Header.Set("X-Craftsky-Device-Id", "dev-test")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestAddRoutes_V1WhoAmIWithoutDeviceIDReturns400(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest("GET", "/v1/whoami", nil)
	req.Header.Set("Authorization", "Bearer anything")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "missing_device_id") {
		t.Errorf("body = %q, want containing 'missing_device_id'", rec.Body.String())
	}
}

func TestAddRoutes_MetricsEndpointIsRemoved(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 after /metrics removal; body=%s", rec.Code, rec.Body.String())
	}
	if body := rec.Body.String(); strings.Contains(body, "craftsky_appview") || strings.Contains(body, "# HELP") {
		t.Fatalf("/metrics returned metrics output after removal: %s", body)
	}
}

func TestAddRoutes_NotificationResolutionRouteIsRemoved(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest(
		http.MethodGet,
		"/v1/notifications/00000000-0000-0000-0000-000000000001",
		nil,
	)
	_, pattern := mux.Handler(req)
	if pattern != "/" {
		t.Fatalf("former notification resolution path matched %q, want fallback route", pattern)
	}
}

func TestAddRoutes_NoMetricsAuthBypassAndV1RoutesStillEnforceDevice(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	mux.ServeHTTP(metricsRec, metricsReq)
	if metricsRec.Code != http.StatusNotFound {
		t.Fatalf("/metrics status = %d, want 404 after removal; body=%s", metricsRec.Code, metricsRec.Body.String())
	}

	v1Req := httptest.NewRequest(http.MethodGet, "/v1/whoami", nil)
	v1Req.Header.Set("Authorization", "Bearer anything")
	v1Rec := httptest.NewRecorder()
	mux.ServeHTTP(v1Rec, v1Req)
	if v1Rec.Code == http.StatusOK {
		t.Fatalf("/v1/whoami without auth/device status = 200, want an auth/device error")
	}
	if !strings.Contains(v1Rec.Body.String(), "missing_device_id") {
		t.Fatalf("/v1/whoami body = %q, want missing_device_id", v1Rec.Body.String())
	}
}

func TestAddRoutes_BodyPolicyRunsThroughMux(t *testing.T) {
	t.Run("unknown-length body is not read before authentication", func(t *testing.T) {
		deps := testDeps()
		deps.Config.JSONBodyLimitBytes = 8
		mux := http.NewServeMux()
		AddRoutes(context.Background(), mux, deps)

		body := &bodyReadProbe{Reader: strings.NewReader("123456789")}
		req := httptest.NewRequest(http.MethodPost, "/v1/posts", body)
		req.Header.Set("X-Craftsky-Device-Id", "dev-test")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401; body=%s", rec.Code, rec.Body.String())
		}
		if body.bytesRead != 0 {
			t.Fatalf("body bytes read = %d before authentication, want 0", body.bytesRead)
		}
	})

	t.Run("no-body route rejects unexpected body before auth", func(t *testing.T) {
		deps := testDeps()
		mux := http.NewServeMux()
		AddRoutes(context.Background(), mux, deps)

		req := httptest.NewRequest(http.MethodGet, "/v1/whoami", strings.NewReader("{}"))
		req.Header.Set("Authorization", "Bearer anything")
		req.Header.Set("X-Craftsky-Device-Id", "dev-test")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "request_body_not_allowed") {
			t.Fatalf("body = %q, want request_body_not_allowed", rec.Body.String())
		}
	})
}

type bodyReadProbe struct {
	io.Reader
	bytesRead int
}

func (probe *bodyReadProbe) Read(buffer []byte) (int, error) {
	read, err := probe.Reader.Read(buffer)
	probe.bytesRead += read
	return read, err
}

func TestAddRoutes_RateLimitRejectsBeforeHandlerWork(t *testing.T) {
	deps := testDeps()
	deps.RateLimiter = middleware.NewLocalRateLimiter(middleware.RateLimitConfig{Classes: map[middleware.RateClass]middleware.ClassLimit{
		middleware.RateClassRead: {Window: time.Minute, PerDevice: 1},
	}}, func() time.Time { return time.Unix(100, 0) })
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, deps)

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/v1/whoami", nil)
		req.Header.Set("Authorization", "Bearer anything")
		req.Header.Set("X-Dev-DID", "did:plc:from-header")
		req.Header.Set("X-Craftsky-Device-Id", "dev-test")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if i == 0 && rec.Code != http.StatusOK {
			t.Fatalf("first status = %d, want 200; body=%s", rec.Code, rec.Body.String())
		}
		if i == 1 {
			if rec.Code != http.StatusTooManyRequests {
				t.Fatalf("second status = %d, want 429; body=%s", rec.Code, rec.Body.String())
			}
			if rec.Header().Get("Retry-After") == "" {
				t.Fatal("Retry-After header is empty")
			}
			if rec.Header().Get("X-RateLimit-Limit") != "" || rec.Header().Get("X-RateLimit-Remaining") != "" {
				t.Fatalf("unexpected public rate-limit headers: %v", rec.Header())
			}
			if !strings.Contains(rec.Body.String(), "rate_limited") {
				t.Fatalf("body = %q, want rate_limited", rec.Body.String())
			}
			if strings.Contains(rec.Body.String(), "stub-handle.example") {
				t.Fatalf("rate-limited response appears to include handler output: %q", rec.Body.String())
			}
		}
	}
}

func TestAddRoutes_TimelineRequiresDeviceID(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest("GET", "/v1/feed/timeline", nil)
	req.Header.Set("Authorization", "Bearer anything")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "missing_device_id") {
		t.Errorf("body = %q, want containing 'missing_device_id'", rec.Body.String())
	}
}

func TestAddRoutes_V1LoginWithoutDeviceIDReturns400(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest("POST", "/v1/auth/login", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "missing_device_id") {
		t.Errorf("body = %q, want containing 'missing_device_id'", rec.Body.String())
	}
}

func TestAddRoutes_LegacyUnprefixedWhoAmIReturns404(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest("GET", "/whoami", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 (legacy path should be gone)", rec.Code)
	}
}

func TestAddRoutes_HealthStaysUnprefixed(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest("GET", "/health", nil)
	_, pattern := mux.Handler(req)
	if pattern == "/" || pattern == "" {
		t.Errorf("pattern = %q; /health must be registered at a top-level path", pattern)
	}
}

func TestAddRoutes_OAuthClientMetadataStaysUnprefixed(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest("GET", "/oauth/client-metadata.json", nil)
	_, pattern := mux.Handler(req)
	if pattern == "/" || pattern == "" {
		t.Errorf("pattern = %q; /oauth/client-metadata.json must be registered", pattern)
	}
}

func TestAddRoutes_UnknownPathReturns404(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest("GET", "/does-not-exist", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestRoutes_DevModerationRouteUnavailableUnlessEnabled(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  Config
	}{
		{name: "prod with flag", cfg: Config{Env: EnvProd, EnableDevModeration: true, DevModerationToken: "secret"}},
		{name: "dev flag off", cfg: Config{Env: EnvDev, EnableDevModeration: false, DevModerationToken: "secret"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			deps := testDeps()
			deps.Config = tc.cfg
			mux := http.NewServeMux()
			AddRoutes(context.Background(), mux, deps)

			req := httptest.NewRequest(http.MethodPost, "/v1/dev/moderation/ozone-events", strings.NewReader(`{"subject":{"type":"post","did":"did:plc:bob","rkey":"rk"},"value":"hide","action":"apply"}`))
			req.Header.Set("X-Craftsky-Dev-Moderation-Token", "secret")
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)

			if rr.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404; body = %s", rr.Code, rr.Body.String())
			}
		})
	}
}

func TestRoutes_DevPanicRouteIsDevOnly(t *testing.T) {
	t.Run("dev route is registered", func(t *testing.T) {
		deps := testDeps()
		deps.Config = Config{Env: EnvDev}
		mux := http.NewServeMux()
		AddRoutes(context.Background(), mux, deps)

		defer func() {
			recovered := recover()
			if recovered == nil {
				t.Fatal("GET /v1/dev/panic did not panic")
			}
			if recovered != "synthetic appview dev panic" {
				t.Fatalf("panic = %#v, want synthetic appview dev panic", recovered)
			}
		}()
		mux.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/v1/dev/panic", nil))
	})

	t.Run("prod route is not registered", func(t *testing.T) {
		deps := testDeps()
		deps.Config = Config{Env: EnvProd}
		mux := http.NewServeMux()
		AddRoutes(context.Background(), mux, deps)

		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/dev/panic", nil))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404; body=%s", rec.Code, rec.Body.String())
		}
	})
}

func TestRoutes_DevModerationRouteRequiresToken(t *testing.T) {
	deps := testDeps()
	deps.Config = Config{Env: EnvDev, EnableDevModeration: true, DevModerationToken: "secret-token"}
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, deps)

	for _, tc := range []struct {
		name  string
		token string
	}{
		{name: "missing"},
		{name: "invalid", token: "wrong"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/v1/dev/moderation/ozone-events", strings.NewReader(`{"subject":{"type":"post","did":"did:plc:bob","rkey":"rk"},"value":"hide","action":"apply"}`))
			if tc.token != "" {
				req.Header.Set("X-Craftsky-Dev-Moderation-Token", tc.token)
			}
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)

			if rr.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want 403; body = %s", rr.Code, rr.Body.String())
			}
			if !strings.Contains(rr.Body.String(), "invalid_dev_moderation_token") {
				t.Fatalf("body = %q, want invalid_dev_moderation_token", rr.Body.String())
			}
		})
	}
}

func TestRoutes_DevModerationRouteUsesCatalogueBodyAdmission(t *testing.T) {
	newMux := func() *http.ServeMux {
		deps := testDeps()
		deps.Config = Config{
			Env:                 EnvDev,
			EnableDevModeration: true,
			DevModerationToken:  "secret-token",
			JSONBodyLimitBytes:  128,
		}
		mux := http.NewServeMux()
		AddRoutes(context.Background(), mux, deps)
		return mux
	}

	for _, tc := range []struct {
		name       string
		body       io.ReadCloser
		length     int64
		transfer   []string
		wantStatus int
		wantCode   string
	}{
		{
			name:       "declared oversized body",
			body:       io.NopCloser(strings.NewReader(strings.Repeat("x", 129))),
			length:     129,
			wantStatus: http.StatusRequestEntityTooLarge,
			wantCode:   "request_body_too_large",
		},
		{
			name:       "chunked oversized body",
			body:       io.NopCloser(strings.NewReader(`{"padding":"` + strings.Repeat("x", 256) + `"}`)),
			length:     -1,
			transfer:   []string{"chunked"},
			wantStatus: http.StatusRequestEntityTooLarge,
			wantCode:   "request_body_too_large",
		},
		{
			name:       "timed out body",
			body:       routeTimeoutBody{},
			length:     -1,
			transfer:   []string{"chunked"},
			wantStatus: http.StatusRequestTimeout,
			wantCode:   "request_body_timeout",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/v1/dev/moderation/ozone-events", tc.body)
			req.ContentLength = tc.length
			req.TransferEncoding = tc.transfer
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Craftsky-Dev-Moderation-Token", "secret-token")
			req.Header.Set("Idempotency-Key", "route-body-policy-0001")
			recorder := httptest.NewRecorder()
			newMux().ServeHTTP(recorder, req)

			if recorder.Code != tc.wantStatus || !strings.Contains(recorder.Body.String(), tc.wantCode) {
				t.Fatalf("status/body = %d %q, want %d containing %q", recorder.Code, recorder.Body.String(), tc.wantStatus, tc.wantCode)
			}
		})
	}
}

func TestRoutes_DevModerationRoutePersistsValidOutput(t *testing.T) {
	pool := testdb.WithSchema(t, routeModerationDDL)
	deps := testDeps()
	deps.DB = pool
	deps.ModerationStore = newRouteModerationStore(t, pool)
	deps.Config = Config{
		Env:                         EnvDev,
		EnableDevModeration:         true,
		DevModerationToken:          "secret-token",
		DevLabelerDID:               "did:plc:labeler",
		TrustedModerationSourceDIDs: []string{"did:plc:labeler"},
	}
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, deps)

	req := httptest.NewRequest(http.MethodPost, "/v1/dev/moderation/ozone-events", strings.NewReader(`{
		"subject":{"type":"post","did":"did:plc:bob","rkey":"3lf2abc"},
		"value":"hide",
		"action":"apply",
		"internalReason":"private fixture"
	}`))
	req.Header.Set("X-Craftsky-Dev-Moderation-Token", "secret-token")
	req.Header.Set("Idempotency-Key", "route-valid-output-0001")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", rr.Code, rr.Body.String())
	}
	var body struct {
		OutputID string `json:"outputId"`
		Status   string `json:"status"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("body not valid JSON: %v", err)
	}
	if body.OutputID == "" || body.Status != "indexed" {
		t.Fatalf("body = %+v, want outputId and indexed status", body)
	}

	var count int
	var subjectURI, internalReason string
	if err := pool.QueryRow(context.Background(), `SELECT count(*)::int, max(subject_uri), max(internal_reason) FROM moderation_outputs`).Scan(&count, &subjectURI, &internalReason); err != nil {
		t.Fatalf("query persisted output: %v", err)
	}
	if count != 1 || subjectURI != "at://did:plc:bob/social.craftsky.feed.post/3lf2abc" || internalReason != "private fixture" {
		t.Fatalf("persisted count=%d subjectURI=%q internalReason=%q", count, subjectURI, internalReason)
	}
}

func TestRoutes_DevModerationRouteRejectsInvalidWithoutMutation(t *testing.T) {
	pool := testdb.WithSchema(t, routeModerationDDL)
	deps := testDeps()
	deps.DB = pool
	deps.ModerationStore = newRouteModerationStore(t, pool)
	deps.Config = Config{
		Env:                         EnvDev,
		EnableDevModeration:         true,
		DevModerationToken:          "secret-token",
		DevLabelerDID:               "did:plc:labeler",
		TrustedModerationSourceDIDs: []string{"did:plc:labeler"},
	}
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, deps)

	req := httptest.NewRequest(http.MethodPost, "/v1/dev/moderation/ozone-events", strings.NewReader(`{
		"sourceDid":"did:plc:untrusted",
		"subject":{"type":"account","did":"did:plc:bob"},
		"value":"warn",
		"action":"apply"
	}`))
	req.Header.Set("X-Craftsky-Dev-Moderation-Token", "secret-token")
	req.Header.Set("Idempotency-Key", "route-invalid-output-0001")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body = %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "untrusted_moderation_source") {
		t.Fatalf("body = %q, want untrusted_moderation_source", rr.Body.String())
	}
	var count int
	if err := pool.QueryRow(context.Background(), `SELECT count(*)::int FROM moderation_outputs`).Scan(&count); err != nil {
		t.Fatalf("count moderation outputs: %v", err)
	}
	if count != 0 {
		t.Fatalf("stored outputs = %d, want 0", count)
	}
}

// IT-007: the shared interaction path keeps distinct read and mutation registrations.
func TestPostInteractionRouteMethodsCoexist(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())
	for _, target := range []struct {
		name string
		path string
	}{
		{name: "likes", path: "/v1/posts/did:plc:alice/root/likes"},
		{name: "reposts", path: "/v1/posts/did:plc:alice/root/reposts"},
		{name: "quotes", path: "/v1/posts/did:plc:alice/root/quotes"},
	} {
		t.Run(target.name, func(t *testing.T) {
			for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodDelete} {
				if target.name == "quotes" && method != http.MethodGet {
					continue
				}
				_, pattern := mux.Handler(httptest.NewRequest(method, target.path, nil))
				if pattern == "" || pattern == "/" {
					t.Fatalf("%s %s pattern = %q, want registered route", method, target.path, pattern)
				}
			}
		})
	}
}

const postInteractionReadRouteDDL = `
CREATE TABLE craftsky_profiles (
	did TEXT PRIMARY KEY,
	record_cid TEXT NOT NULL
);
CREATE TABLE bluesky_profiles (
	did TEXT PRIMARY KEY,
	display_name TEXT,
	description TEXT,
	avatar_cid TEXT,
	avatar_mime TEXT
);
CREATE TABLE craftsky_posts (
	uri TEXT PRIMARY KEY,
	did TEXT NOT NULL,
	rkey TEXT NOT NULL,
	cid TEXT NOT NULL,
	text TEXT NOT NULL,
	sponsored BOOLEAN NOT NULL DEFAULT false,
	facets JSONB,
	images JSONB,
	record JSONB NOT NULL,
	reply_root_uri TEXT,
	reply_root_cid TEXT,
	reply_parent_uri TEXT,
	reply_parent_cid TEXT,
	quote_uri TEXT,
	quote_cid TEXT,
	tags TEXT[] NOT NULL DEFAULT '{}',
	langs TEXT[] NOT NULL DEFAULT '{}',
	created_at TIMESTAMPTZ NOT NULL,
	indexed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	external_import_source TEXT,
	profile_sort_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	is_project BOOLEAN NOT NULL DEFAULT false,
	project_craft_type TEXT
);
CREATE TABLE craftsky_project_posts (
	uri TEXT PRIMARY KEY,
	raw_project JSONB NOT NULL
);
CREATE TABLE craftsky_post_mentions (
	post_uri TEXT NOT NULL,
	mentioned_did TEXT NOT NULL
);
CREATE TABLE craftsky_likes (
	uri TEXT PRIMARY KEY,
	did TEXT NOT NULL,
	subject_uri TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL,
	deleted_at TIMESTAMPTZ
);
CREATE TABLE craftsky_reposts (
	uri TEXT PRIMARY KEY,
	did TEXT NOT NULL,
	subject_uri TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL,
	deleted_at TIMESTAMPTZ
);
CREATE TABLE actor_mutes (
	owner_did TEXT NOT NULL,
	subject_did TEXT NOT NULL,
	PRIMARY KEY (owner_did, subject_did)
);
CREATE TABLE atproto_blocks (
	uri TEXT PRIMARY KEY,
	blocker_did TEXT NOT NULL,
	subject_did TEXT NOT NULL
);
CREATE TABLE atproto_identity_cache (
	did TEXT PRIMARY KEY,
	handle TEXT NOT NULL
);
CREATE TABLE moderation_outputs (
	id TEXT PRIMARY KEY,
	source_did TEXT NOT NULL,
	subject_type TEXT NOT NULL,
	subject_did TEXT NOT NULL,
	subject_uri TEXT,
	value TEXT NOT NULL,
	action TEXT NOT NULL,
	expires_at TIMESTAMPTZ,
	indexed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE account_language_preferences (
	account_did TEXT PRIMARY KEY,
	primary_language TEXT NOT NULL,
	content_languages TEXT[] NOT NULL
);
`

// REG-004: successful production-mux interaction reads cannot reach PDS effects.
func TestPostInteractionReadRoutesSucceedWithoutPDSEffects(t *testing.T) {
	ctx := context.Background()
	pool := testdb.WithSchema(t, postInteractionReadRouteDDL)
	viewer := "did:plc:interactionrouteviewer"
	owner := "did:plc:interactionrouteowner"
	rkey := "route"
	uri := "at://" + owner + "/social.craftsky.feed.post/" + rkey
	statements := []struct {
		query string
		args  []any
	}{
		{`INSERT INTO craftsky_profiles (did, record_cid) VALUES ($1, 'viewer-cid'), ($2, 'owner-cid')`, []any{viewer, owner}},
		{`INSERT INTO account_language_preferences (account_did, primary_language, content_languages) VALUES ($1, 'en', ARRAY['en'])`, []any{viewer}},
		{`INSERT INTO craftsky_posts (uri, did, rkey, cid, text, record, created_at) VALUES ($1, $2, $3, 'route-cid', 'route target', '{}', now())`, []any{uri, owner, rkey}},
	}
	for _, statement := range statements {
		if _, err := pool.Exec(ctx, statement.query, statement.args...); err != nil {
			t.Fatalf("seed production route fixture: %v", err)
		}
	}

	pdsCalls := 0
	deps := testDeps()
	deps.DB = pool
	deps.LanguagePreferences = languages.NewStore(pool)
	deps.NewPDSEffects = func(context.Context, syntax.DID, string) (pdseffects.EffectExecutor, error) {
		pdsCalls++
		return nil, errors.New("PDS effects must not be constructed for reads")
	}
	mux := http.NewServeMux()
	AddRoutes(ctx, mux, deps)
	for _, suffix := range []string{"likes", "reposts", "quotes"} {
		request := httptest.NewRequest(http.MethodGet, "/v1/posts/"+owner+"/"+rkey+"/"+suffix, nil)
		request.Header.Set("Authorization", "Bearer interaction-test")
		request.Header.Set("X-Craftsky-Device-Id", "interaction-test-device")
		request.Header.Set("X-Dev-DID", viewer)
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("GET %s status/body = %d/%s, want 200", suffix, response.Code, response.Body.String())
		}
	}
	if pdsCalls != 0 {
		t.Fatalf("PDS effect factory calls = %d, want 0", pdsCalls)
	}
}

func TestScheduledImageValidatorUsesRouteObserver(t *testing.T) {
	recorder := observability.NewInMemoryMetricRecorder()
	observer := observability.New(observability.Config{MetricRecorder: recorder})
	validator := newScheduledImageValidator(api.DefaultImageDecodeLimits(), observer)
	value := image.NewRGBA(image.Rect(0, 0, 1, 1))
	value.Set(0, 0, color.White)
	var payload bytes.Buffer
	if err := jpeg.Encode(&payload, value, nil); err != nil {
		t.Fatalf("encode JPEG fixture: %v", err)
	}

	if _, err := validator.Validate(
		context.Background(), "image/jpeg", payload.Bytes(),
	); err != nil {
		t.Fatalf("Validate error = %v, want nil", err)
	}
	for _, call := range recorder.Calls() {
		if call.Name == "craftsky_appview_scheduled_image_validations_total" {
			return
		}
	}
	t.Fatalf("scheduled image metric missing: %#v", recorder.Calls())
}

func TestSearchProjectsRouteRejectsBrowseFilters(t *testing.T) {
	pool := testdb.WithSchema(t, `
		CREATE TABLE craftsky_profiles (did TEXT PRIMARY KEY);
		INSERT INTO craftsky_profiles (did) VALUES ('did:plc:test');
	`)
	deps := testDeps()
	deps.DB = pool
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, deps)
	req := httptest.NewRequest(http.MethodGet, "/v1/search/projects?q=sock&craftType=knitting&material=alpaca", nil)
	req.Header.Set("Authorization", "Bearer anything")
	req.Header.Set("X-Craftsky-Device-Id", "dev-test")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body=%s, want 400", rr.Code, rr.Body.String())
	}
	assertErrorEnvelope(t, rr.Body.Bytes())
	if !strings.Contains(rr.Body.String(), "validation_error") {
		t.Fatalf("body = %s, want validation_error", rr.Body.String())
	}
}

func assertErrorEnvelope(t *testing.T, body []byte) {
	t.Helper()
	var env map[string]any
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("body not valid JSON: %v; body=%s", err, string(body))
	}
	for _, key := range []string{"error", "message", "requestId"} {
		if _, ok := env[key]; !ok {
			t.Fatalf("error envelope missing %s: %v", key, env)
		}
	}
}
