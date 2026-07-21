package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/app"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/instagram"
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
