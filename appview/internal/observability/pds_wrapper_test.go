package observability

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/getsentry/sentry-go"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ownerlifecycle"
)

type fakePDSClient struct {
	getErr    error
	createErr error
	uploadErr error
}

type fakeListPDSClient struct {
	fakePDSClient
	records []auth.PDSRecord
	cursor  string
}

type fakeEffectPDSClient struct {
	fakePDSClient
	calls int
}

type fakeConditionalPDSClient struct {
	fakePDSClient
	calls       int
	putCalls    int
	expectedCID syntax.CID
}

func (client *fakeConditionalPDSClient) PutRecordWithSwap(
	_ context.Context,
	_ syntax.DID,
	_ string,
	_ string,
	_ any,
	expectedCID syntax.CID,
) error {
	client.putCalls++
	client.expectedCID = expectedCID
	return nil
}

func (client *fakeConditionalPDSClient) DeleteRecordWithSwap(
	_ context.Context,
	_ syntax.DID,
	_ string,
	_ string,
	expectedCID syntax.CID,
) error {
	client.calls++
	client.expectedCID = expectedCID
	return nil
}

type fakeConditionalEffectPDSClient struct {
	fakePDSClient
	purpose *fakeConditionalPDSClient
}

func (client *fakeConditionalEffectPDSClient) WithActiveEffects(
	ctx context.Context,
	_ []ownerlifecycle.ExpectedOwner,
	operation auth.ActiveEffectPDSOperation,
) error {
	return operation(ctx, client.purpose)
}

func (f *fakeEffectPDSClient) WithActiveEffects(
	ctx context.Context,
	expected []ownerlifecycle.ExpectedOwner,
	operation auth.ActiveEffectPDSOperation,
) error {
	f.calls++
	return operation(ctx, f.fakePDSClient)
}

func (f fakeListPDSClient) ListRecords(context.Context, syntax.DID, string, string, int) ([]auth.PDSRecord, string, error) {
	return f.records, f.cursor, nil
}

func TestWrapPDSFactoryPreservesRecordListingCapability(t *testing.T) {
	observer := New(Config{Env: "test", MetricRecorder: NewInMemoryMetricRecorder()})
	want := []auth.PDSRecord{{URI: "at://did:plc:alice/app.bsky.graph.block/one", CID: "bafy-one"}}
	wrappedFactory := observer.WrapPDSFactory(func(context.Context, syntax.DID, string) (auth.PDSClient, error) {
		return fakeListPDSClient{records: want, cursor: "next-page"}, nil
	})

	client, err := wrappedFactory(context.Background(), syntax.DID("did:plc:alice"), "session")
	if err != nil {
		t.Fatalf("wrapped factory: %v", err)
	}
	lister, ok := client.(auth.PDSRecordLister)
	if !ok {
		t.Fatal("wrapped PDS client lost PDSRecordLister capability")
	}
	records, cursor, err := lister.ListRecords(context.Background(), syntax.DID("did:plc:alice"), "app.bsky.graph.block", "", 100)
	if err != nil {
		t.Fatalf("ListRecords: %v", err)
	}
	if len(records) != 1 || records[0].URI != want[0].URI || cursor != "next-page" {
		t.Fatalf("records/cursor = %+v/%q, want %+v/%q", records, cursor, want, "next-page")
	}
}

func TestWrapPDSFactoryPreservesActiveEffectCapability(t *testing.T) {
	observer := New(Config{Env: "test", MetricRecorder: NewInMemoryMetricRecorder()})
	inner := &fakeEffectPDSClient{}
	wrappedFactory := observer.WrapPDSFactory(func(context.Context, syntax.DID, string) (auth.PDSClient, error) {
		return inner, nil
	})

	client, err := wrappedFactory(context.Background(), syntax.DID("did:plc:alice"), "session")
	if err != nil {
		t.Fatalf("wrapped factory: %v", err)
	}
	boundary, ok := client.(auth.ActiveEffectPDSBoundary)
	if !ok {
		t.Fatal("wrapped PDS client lost ActiveEffectPDSBoundary capability")
	}
	var callbackClient auth.PDSClient
	err = boundary.WithActiveEffects(
		context.Background(),
		[]ownerlifecycle.ExpectedOwner{{Owner: syntax.DID("did:plc:alice"), Generation: 2}},
		func(_ context.Context, purposeClient auth.PDSClient) error {
			callbackClient = purposeClient
			return purposeClient.PutRecord(
				context.Background(),
				syntax.DID("did:plc:alice"),
				"social.craftsky.feed.post",
				"3lobserved",
				map[string]any{"text": "observed"},
			)
		},
	)
	if err != nil {
		t.Fatalf("WithActiveEffects: %v", err)
	}
	if inner.calls != 1 || callbackClient == nil {
		t.Fatalf("active effect delegation calls/client = %d/%T", inner.calls, callbackClient)
	}
}

func TestWrapPDSFactoryPreservesConditionalDeleteInsideActiveEffects(t *testing.T) {
	observer := New(Config{Env: "test", MetricRecorder: NewInMemoryMetricRecorder()})
	purpose := &fakeConditionalPDSClient{}
	inner := &fakeConditionalEffectPDSClient{purpose: purpose}
	wrapper := observer.WrapPDSFactory(func(context.Context, syntax.DID, string) (auth.PDSClient, error) {
		return inner, nil
	})
	client, err := wrapper(context.Background(), syntax.DID("did:plc:alice"), "session")
	if err != nil {
		t.Fatal(err)
	}
	boundary, ok := client.(auth.ActiveEffectPDSBoundary)
	if !ok {
		t.Fatal("wrapped PDS client lost ActiveEffectPDSBoundary")
	}
	err = boundary.WithActiveEffects(
		context.Background(),
		[]ownerlifecycle.ExpectedOwner{{Owner: syntax.DID("did:plc:alice"), Generation: 2}},
		func(ctx context.Context, callbackClient auth.PDSClient) error {
			deleter, ok := callbackClient.(auth.ConditionalPDSRecordDeleter)
			if !ok {
				return errors.New("observed callback client lost ConditionalPDSRecordDeleter")
			}
			return deleter.DeleteRecordWithSwap(
				ctx,
				syntax.DID("did:plc:alice"),
				"social.craftsky.feed.post",
				"3lconditional",
				syntax.CID("bafy-expected"),
			)
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if purpose.calls != 1 || purpose.expectedCID != "bafy-expected" {
		t.Fatalf("conditional delete calls/CID = %d/%q", purpose.calls, purpose.expectedCID)
	}
}

func TestWrapPDSFactoryPreservesConditionalPutInsideActiveEffects(t *testing.T) {
	observer := New(Config{Env: "test", MetricRecorder: NewInMemoryMetricRecorder()})
	purpose := &fakeConditionalPDSClient{}
	inner := &fakeConditionalEffectPDSClient{purpose: purpose}
	wrapper := observer.WrapPDSFactory(func(context.Context, syntax.DID, string) (auth.PDSClient, error) {
		return inner, nil
	})
	client, err := wrapper(context.Background(), syntax.DID("did:plc:alice"), "session")
	if err != nil {
		t.Fatal(err)
	}
	boundary := client.(auth.ActiveEffectPDSBoundary)
	err = boundary.WithActiveEffects(
		context.Background(),
		[]ownerlifecycle.ExpectedOwner{{Owner: syntax.DID("did:plc:alice"), Generation: 2}},
		func(ctx context.Context, callbackClient auth.PDSClient) error {
			putter, ok := callbackClient.(auth.ConditionalPDSRecordPutter)
			if !ok {
				return errors.New("observed callback client lost ConditionalPDSRecordPutter")
			}
			return putter.PutRecordWithSwap(
				ctx,
				syntax.DID("did:plc:alice"),
				"social.craftsky.actor.profile",
				"self",
				map[string]any{"displayName": "updated"},
				syntax.CID("bafy-prior"),
			)
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if purpose.putCalls != 1 || purpose.expectedCID != "bafy-prior" {
		t.Fatalf("conditional Put calls/CID = %d/%q", purpose.putCalls, purpose.expectedCID)
	}
}

func (f fakePDSClient) GetRecord(context.Context, syntax.DID, string, string, any) (string, error) {
	return "", f.getErr
}
func (f fakePDSClient) PutRecord(context.Context, syntax.DID, string, string, any) error {
	return nil
}
func (f fakePDSClient) CreateRecord(context.Context, syntax.DID, string, any) (syntax.ATURI, syntax.CID, error) {
	return "", "", f.createErr
}
func (f fakePDSClient) DeleteRecord(context.Context, syntax.DID, string, string) error {
	return nil
}
func (f fakePDSClient) UploadBlob(context.Context, string, []byte) (*auth.UploadedBlob, error) {
	if f.uploadErr != nil {
		return nil, f.uploadErr
	}
	return &auth.UploadedBlob{}, nil
}

func TestWrapPDSFactoryRecordsWriteTelemetry(t *testing.T) {
	recorder := NewInMemoryMetricRecorder()
	observer := New(Config{Env: "test", MetricRecorder: recorder})
	factoryErr := errors.New("token refresh failed")
	wrappedFactory := observer.WrapPDSFactory(func(context.Context, syntax.DID, string) (auth.PDSClient, error) {
		return fakePDSClient{createErr: factoryErr}, nil
	})

	client, err := wrappedFactory(context.Background(), syntax.DID("did:plc:writer"), "session-secret")
	if err != nil {
		t.Fatalf("wrapped factory: %v", err)
	}
	_, _, _ = client.CreateRecord(context.Background(), syntax.DID("did:plc:writer"), "social.craftsky.feed.post", map[string]any{"text": "secret body"})
	_, _ = client.UploadBlob(context.Background(), "image/png", []byte("raw media bytes"))

	failingFactory := observer.WrapPDSFactory(func(context.Context, syntax.DID, string) (auth.PDSClient, error) {
		return nil, factoryErr
	})
	_, _ = failingFactory(context.Background(), syntax.DID("did:plc:writer"), "session-secret")

	calls := recorder.Calls()
	for _, want := range []string{
		"craftsky_appview_pds_write_duration_seconds",
	} {
		if !metricCallsContain(calls, want) {
			t.Fatalf("metric calls missing %q: %#v", want, calls)
		}
	}
	for _, call := range calls {
		if err := ValidateMetricCall(call); err != nil {
			t.Fatalf("metric call failed validation: %v; call=%#v", err, call)
		}
	}
}

func TestWrapPDSFactoryEmitsLogsSpansAndSentryForUnexpectedFailures(t *testing.T) {
	var logs bytes.Buffer
	transport := &sentry.MockTransport{}
	observer := New(Config{
		Env:              "test",
		TracingEnabled:   true,
		TracesSampleRate: 1,
		SentryDSN:        "https://public@example.invalid/1",
		SentryTransport:  transport,
		Logger:           slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})),
	})
	pdsErr := errors.New("upstream leaked did:plc:writer in error")
	wrappedFactory := observer.WrapPDSFactory(func(context.Context, syntax.DID, string) (auth.PDSClient, error) {
		return fakePDSClient{createErr: pdsErr}, nil
	})

	client, err := wrappedFactory(context.Background(), syntax.DID("did:plc:writer"), "session-secret")
	if err != nil {
		t.Fatalf("wrapped factory: %v", err)
	}
	_, _, _ = client.CreateRecord(context.Background(), syntax.DID("did:plc:writer"), "social.craftsky.feed.post", map[string]any{"text": "secret body"})

	logged := logs.String()
	for _, want := range []string{
		`"msg":"pds write completed"`,
		`"component":"pds"`,
		`"operation":"post.create"`,
		`"failure_stage":"pds_request"`,
		`"result":"error"`,
		`"error_category":"unexpected"`,
	} {
		if !strings.Contains(logged, want) {
			t.Fatalf("PDS logs missing %q:\n%s", want, logged)
		}
	}
	for _, forbidden := range []string{"did:plc:writer", "session-secret", "secret body"} {
		if strings.Contains(logged, forbidden) {
			t.Fatalf("PDS logs contain sensitive value %q:\n%s", forbidden, logged)
		}
	}

	if !observer.Flush(50 * time.Millisecond) {
		t.Fatal("observer Flush returned false")
	}
	events := transport.Events()
	var errorEvents []*sentry.Event
	for _, event := range events {
		if event.Level == sentry.LevelError {
			errorEvents = append(errorEvents, event)
		}
	}
	if len(errorEvents) != 1 {
		t.Fatalf("captured %d Sentry error events, want 1; all events=%#v", len(errorEvents), events)
	}
	tags := errorEvents[0].Tags
	for key, want := range map[string]string{
		"component":      "pds",
		"operation":      "post.create",
		"failure_stage":  "pds_request",
		"result":         "error",
		"error_category": "unexpected",
		"error_code":     "appview.unexpected",
	} {
		if tags[key] != want {
			t.Fatalf("Sentry tag %q = %q, want %q; all tags=%#v", key, tags[key], want, tags)
		}
	}
	if tags["sentry_trace_id"] == "" || tags["sentry_span_id"] == "" {
		t.Fatalf("Sentry event missing trace/span IDs: %#v", tags)
	}
	for _, forbidden := range []string{"did:plc:writer", "session-secret", "secret body"} {
		if strings.Contains(errorEvents[0].Message, forbidden) {
			t.Fatalf("Sentry message contains forbidden value %q: %#v", forbidden, errorEvents[0])
		}
		for key, value := range tags {
			if strings.Contains(key, forbidden) || strings.Contains(value, forbidden) {
				t.Fatalf("Sentry tag contains forbidden value %q: %q=%q", forbidden, key, value)
			}
		}
	}
}

func TestWrapPDSFactoryInstrumentsProfileReadBeforeWriteFailures(t *testing.T) {
	recorder := NewInMemoryMetricRecorder()
	transport := &sentry.MockTransport{}
	observer := New(Config{
		Env:             "test",
		SentryDSN:       "https://public@example.invalid/1",
		SentryTransport: transport,
		MetricRecorder:  recorder,
	})
	pdsErr := errors.New("profile read failed for did:plc:writer with secret-token")
	wrappedFactory := observer.WrapPDSFactory(func(context.Context, syntax.DID, string) (auth.PDSClient, error) {
		return fakePDSClient{getErr: pdsErr}, nil
	})

	client, err := wrappedFactory(context.Background(), syntax.DID("did:plc:writer"), "session-secret")
	if err != nil {
		t.Fatalf("wrapped factory: %v", err)
	}
	_, _ = client.GetRecord(context.Background(), syntax.DID("did:plc:writer"), "app.bsky.actor.profile", "self", &map[string]any{})

	var sawProfileReadMetric bool
	for _, call := range recorder.Calls() {
		if call.Name == "craftsky_appview_pds_write_duration_seconds" &&
			call.Attributes["operation"] == "profile.put_bsky" &&
			call.Attributes["stage"] == "pds_request" &&
			call.Attributes["result"] == "error" &&
			call.Attributes["category"] == "unexpected" {
			sawProfileReadMetric = true
		}
		if err := ValidateMetricCall(call); err != nil {
			t.Fatalf("metric call failed validation: %v; call=%#v", err, call)
		}
	}
	if !sawProfileReadMetric {
		t.Fatalf("missing profile read-before-write PDS metric: %#v", recorder.Calls())
	}

	if !observer.Flush(50 * time.Millisecond) {
		t.Fatal("observer Flush returned false")
	}
	var errorEvents []*sentry.Event
	for _, event := range transport.Events() {
		if event.Level == sentry.LevelError {
			errorEvents = append(errorEvents, event)
		}
	}
	if len(errorEvents) != 1 {
		t.Fatalf("captured %d Sentry error events, want 1; all events=%#v", len(errorEvents), transport.Events())
	}
	tags := errorEvents[0].Tags
	for key, want := range map[string]string{
		"component":      "pds",
		"operation":      "profile.put_bsky",
		"failure_stage":  "pds_request",
		"result":         "error",
		"error_category": "unexpected",
		"error_code":     "appview.unexpected",
	} {
		if tags[key] != want {
			t.Fatalf("Sentry tag %q = %q, want %q; all tags=%#v", key, tags[key], want, tags)
		}
	}
	for _, forbidden := range []string{"did:plc:writer", "session-secret", "secret-token"} {
		if strings.Contains(errorEvents[0].Message, forbidden) || strings.Contains(errorEvents[0].Exception[0].Value, forbidden) {
			t.Fatalf("Sentry event contains forbidden value %q: %#v", forbidden, errorEvents[0])
		}
		for key, value := range tags {
			if strings.Contains(key, forbidden) || strings.Contains(value, forbidden) {
				t.Fatalf("Sentry tag contains forbidden value %q: %q=%q", forbidden, key, value)
			}
		}
	}
}
