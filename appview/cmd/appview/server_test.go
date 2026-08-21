package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/app"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/instagram"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/observability"
)

type serverStubResolver struct{ handle syntax.Handle }

func (s serverStubResolver) ResolveHandle(_ context.Context, _ syntax.DID) (syntax.Handle, error) {
	return s.handle, nil
}
func (s serverStubResolver) ResolveDID(_ context.Context, _ syntax.Handle) (syntax.DID, error) {
	return "", nil
}

var _ api.HandleResolver = serverStubResolver{}

func TestNewServer_HTTPMetricsUseRoutePattern(t *testing.T) {
	recorder := observability.NewInMemoryMetricRecorder()
	observer := observability.New(observability.Config{Env: "test", MetricRecorder: recorder})
	deps := &app.Deps{
		Config: app.Config{
			Env:            app.EnvDev,
			AllowedOrigins: []string{"*"},
			DevDID:         "did:plc:test",
		},
		Logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		AuthService:    &auth.MockAuthService{DefaultDID: "did:plc:test"},
		HandleResolver: serverStubResolver{handle: syntax.Handle("stub.example")},
		Observability:  observer,
	}
	handler := NewServer(context.Background(), deps)

	req := httptest.NewRequest(http.MethodGet, "/v1/posts/did:plc:raw/rkey123?cursor=secret", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	calls := recorder.Calls()
	for _, call := range calls {
		if call.Name != "craftsky_appview_http_requests_total" {
			continue
		}
		if call.Attributes["route_pattern"] != "/v1/posts/{did}/{rkey}" {
			t.Fatalf("route_pattern = %q, want /v1/posts/{did}/{rkey}; call=%#v", call.Attributes["route_pattern"], call)
		}
		if err := observability.ValidateMetricCall(call); err != nil {
			t.Fatalf("HTTP metric call failed validation: %v; call=%#v", err, call)
		}
		return
	}
	t.Fatalf("missing HTTP request counter call: %#v", calls)
}

func TestNewServerRejectsUnexpectedHostBeforeRouting(t *testing.T) {
	deps := &app.Deps{
		Config: app.Config{
			Env:            app.EnvProd,
			AllowedOrigins: []string{"https://craftsky.social"},
			ExpectedHosts:  []string{"appview.craftsky.social"},
		},
		Logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		AuthService:   &auth.MockAuthService{DefaultDID: "did:plc:test"},
		Observability: observability.New(observability.Config{Env: "test"}),
	}
	handler := NewServer(context.Background(), deps)
	for _, path := range []string{"/v1", "/v1/not-a-route"} {
		t.Run(path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "https://attacker.invalid"+path, nil)
			request.Host = "attacker.invalid"
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusMisdirectedRequest {
				t.Fatalf("status = %d, want 421; body=%s", recorder.Code, recorder.Body.String())
			}
			if got := recorder.Header().Get("Content-Type"); got != "application/json" {
				t.Fatalf("Content-Type = %q", got)
			}
			var got envelope.Error
			if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
				t.Fatal(err)
			}
			if got.Error != "unexpected_host" || got.RequestID == "" {
				t.Fatalf("envelope = %#v", got)
			}
		})
	}
}

func TestNewServerAdmissionRunsBeforeUnexpectedHost(t *testing.T) {
	t.Parallel()

	deps := &app.Deps{
		Config: app.Config{
			Env:            app.EnvProd,
			AllowedOrigins: []string{"https://craftsky.social"},
			ExpectedHosts:  []string{"appview.craftsky.social"},
		},
		Logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		AuthService:   &auth.MockAuthService{DefaultDID: "did:plc:test"},
		Observability: observability.New(observability.Config{Env: "test"}),
	}
	admission := DefaultHandlerAdmissionConfig()
	admission.OuterRateLimits.Classes[middleware.RateClassOuter] = middleware.ClassLimit{
		Window: time.Minute,
		Global: 1,
	}
	handler, err := NewServerWithAdmission(context.Background(), deps, admission)
	if err != nil {
		t.Fatal(err)
	}

	for index := 0; index < 2; index++ {
		request := httptest.NewRequest(http.MethodGet, "https://attacker.invalid/v1/not-a-route", nil)
		request.Host = "attacker.invalid"
		request.RemoteAddr = "192.0.2.1:1234"
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)

		want := http.StatusMisdirectedRequest
		if index == 1 {
			want = http.StatusTooManyRequests
		}
		if recorder.Code != want {
			t.Fatalf("request %d status = %d, want %d; body=%s", index, recorder.Code, want, recorder.Body.String())
		}
		var body envelope.Error
		if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.RequestID == "" {
			t.Fatalf("request %d has empty request ID", index)
		}
	}
}

func TestNewServerOwnsV1FallbackMethodAndCanonicalPathContracts(t *testing.T) {
	t.Parallel()

	deps := &app.Deps{
		Config: app.Config{
			Env:            app.EnvDev,
			AllowedOrigins: []string{"https://app.craftsky.social"},
			DevDID:         "did:plc:test",
		},
		Logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		AuthService:    &auth.MockAuthService{DefaultDID: "did:plc:test"},
		HandleResolver: serverStubResolver{handle: syntax.Handle("stub.example")},
		Observability:  observability.New(observability.Config{Env: "test"}),
	}
	handler := NewServer(context.Background(), deps)

	tests := []struct {
		name      string
		method    string
		path      string
		wantCode  int
		wantError string
		wantAllow string
	}{
		{name: "unknown", method: http.MethodGet, path: "/v1/not-a-route", wantCode: http.StatusNotFound, wantError: "not_found"},
		{name: "wrong method", method: http.MethodPost, path: "/v1/whoami", wantCode: http.StatusMethodNotAllowed, wantError: "method_not_allowed", wantAllow: "GET"},
		{name: "HEAD rejected", method: http.MethodHead, path: "/v1/whoami", wantCode: http.StatusMethodNotAllowed, wantError: "method_not_allowed", wantAllow: "GET"},
		{name: "repeated slash", method: http.MethodGet, path: "/v1//whoami", wantCode: http.StatusNotFound, wantError: "not_found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)
			if recorder.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d; body=%s", recorder.Code, tt.wantCode, recorder.Body.String())
			}
			if got := recorder.Header().Get("Allow"); got != tt.wantAllow {
				t.Fatalf("Allow = %q, want %q", got, tt.wantAllow)
			}
			if recorder.Header().Get("Location") != "" {
				t.Fatalf("Location = %q, want no redirect", recorder.Header().Get("Location"))
			}
			var body envelope.Error
			if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body.Error != tt.wantError || body.RequestID == "" {
				t.Fatalf("envelope = %#v", body)
			}
		})
	}
}

func TestNewServerAllowsCatalogueDerivedPatchPreflight(t *testing.T) {
	t.Parallel()

	deps := &app.Deps{
		Config: app.Config{
			Env:            app.EnvDev,
			AllowedOrigins: []string{"https://app.craftsky.social"},
		},
		Logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		AuthService:   &auth.MockAuthService{DefaultDID: "did:plc:test"},
		Observability: observability.New(observability.Config{Env: "test"}),
	}
	handler := NewServer(context.Background(), deps)
	req := httptest.NewRequest(http.MethodOptions, "/v1/notifications/preferences", nil)
	req.Header.Set("Origin", "https://app.craftsky.social")
	req.Header.Set("Access-Control-Request-Method", http.MethodPatch)
	req.Header.Set("Access-Control-Request-Headers", "Authorization, Content-Type, X-Craftsky-Device-Id")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204; body=%s", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(got, http.MethodPatch) {
		t.Fatalf("Access-Control-Allow-Methods = %q, missing PATCH", got)
	}
}

func TestInstagramWebhookWorkerLoopRetriesErrorsAndDrainsBacklogWithoutPollingDelay(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	processor := &scriptedInstagramBatchProcessor{
		results: []instagramBatchResult{
			{err: context.DeadlineExceeded},
			{processed: 1},
			{cancel: cancel},
		},
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		runInstagramWebhookWorker(ctx, processor, slog.New(slog.NewTextHandler(io.Discard, nil)), time.Millisecond)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Instagram worker loop did not stop after cancellation")
	}
	if processor.calls != 3 {
		t.Fatalf("ProcessBatch calls = %d, want 3", processor.calls)
	}
}

func TestScheduledWorkerLoopRunsImmediatelyDrainsAndStopsOnCancellation(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	processor := &scriptedInstagramBatchProcessor{results: []instagramBatchResult{
		{err: context.DeadlineExceeded}, {processed: 1}, {cancel: cancel},
	}}
	done := make(chan struct{})
	go func() {
		defer close(done)
		runScheduledWorker(ctx, processor, slog.New(slog.NewTextHandler(io.Discard, nil)), time.Millisecond, "publication")
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("scheduled worker did not stop")
	}
	if processor.calls != 3 {
		t.Fatalf("scheduled worker calls=%d, want 3", processor.calls)
	}
}

func TestInstagramReconciliationWorkerLoopUsesBoundedBatchAndDrainsBacklog(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	processor := &scriptedInstagramReconciliationProcessor{
		results: []instagramBatchResult{
			{err: context.DeadlineExceeded},
			{processed: 1},
			{cancel: cancel},
		},
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		runInstagramReconciliationWorker(
			ctx,
			processor,
			slog.New(slog.NewTextHandler(io.Discard, nil)),
			100,
			time.Millisecond,
		)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Instagram reconciliation worker loop did not stop after cancellation")
	}
	if processor.calls != 3 {
		t.Fatalf("ProcessBatch calls = %d, want 3", processor.calls)
	}
	for _, limit := range processor.limits {
		if limit != 100 {
			t.Fatalf("ProcessBatch limit = %d, want 100", limit)
		}
	}
}

func TestInstagramRetentionRunsImmediatelyAndStopsOnCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	runner := &scriptedInstagramRetentionRunner{cancel: cancel}
	done := make(chan struct{})
	go func() {
		defer close(done)
		runInstagramRetention(
			ctx,
			runner,
			slog.New(slog.NewTextHandler(io.Discard, nil)),
			500,
			time.Hour,
		)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Instagram retention loop did not stop after cancellation")
	}
	if runner.calls != 1 || runner.batch != 500 {
		t.Fatalf("retention calls=%d batch=%d, want 1/500", runner.calls, runner.batch)
	}
}

type instagramBatchResult struct {
	processed int
	err       error
	cancel    context.CancelFunc
}

type scriptedInstagramBatchProcessor struct {
	results []instagramBatchResult
	calls   int
}

func (p *scriptedInstagramBatchProcessor) ProcessBatch(context.Context) (int, error) {
	result := p.results[p.calls]
	p.calls++
	if result.cancel != nil {
		result.cancel()
	}
	return result.processed, result.err
}

type scriptedInstagramReconciliationProcessor struct {
	results []instagramBatchResult
	limits  []int
	calls   int
}

type scriptedInstagramRetentionRunner struct {
	cancel context.CancelFunc
	calls  int
	batch  int
}

func (r *scriptedInstagramRetentionRunner) Run(_ context.Context, batch int) (instagram.RetentionStats, error) {
	r.calls++
	r.batch = batch
	if r.cancel != nil {
		r.cancel()
	}
	return instagram.RetentionStats{}, nil
}

func (p *scriptedInstagramReconciliationProcessor) ProcessBatch(_ context.Context, limit int) (int, error) {
	result := p.results[p.calls]
	p.calls++
	p.limits = append(p.limits, limit)
	if result.cancel != nil {
		result.cancel()
	}
	return result.processed, result.err
}
