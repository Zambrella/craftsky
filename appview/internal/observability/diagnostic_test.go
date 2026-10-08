package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"runtime"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ctxkeys"
	"social.craftsky/appview/internal/integrations/instagrammeta"
	lexiconschema "social.craftsky/appview/internal/lexicon/schema"
)

// UT-003 / FR-001, RULE-003 / AC-021: SQL detail is arbitrary private prose.
func TestDiagnosticUnknownPrivateMessage(t *testing.T) {
	transport := &sentry.MockTransport{}
	observer := New(Config{Env: "test", SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
	err := &pgconn.PgError{Code: "23505", Message: "opaque linen sentence", Detail: "another private sentence", ConstraintName: "private_target_canary"}
	observer.CaptureError(context.Background(), EventContext{"component": "db", "operation": "post.read", "failure_stage": "query"}, err)
	observer.Flush(time.Second)
	events := transport.Events()
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}
	event := events[0]
	payload, _ := json.Marshal(event)
	for _, private := range []string{"opaque linen sentence", "another private sentence", "private_target_canary"} {
		if strings.Contains(string(payload), private) {
			t.Fatalf("private diagnostic leaked: %s", private)
		}
	}
	if len(event.Exception) != 1 || event.Exception[0].Type != "*pgconn.PgError" {
		t.Fatalf("concrete cause lost: %#v", event.Exception)
	}
	if !strings.Contains(string(payload), "23505") {
		t.Fatal("approved SQLSTATE lost")
	}
	if event.Tags["failure_stage"] != "query" {
		t.Fatalf("failure stage lost: %#v", event.Tags)
	}
	if event.Exception[0].Stacktrace != nil {
		t.Fatal("ordinary error acquired fabricated origin stack")
	}
}

// UT-001 / FR-001 / AC-001: an opaque wrapper never lends prose permission
// to its causes; reviewed static sentinels retain their actual explanations.
func TestDiagnosticKnownSafeCause(t *testing.T) {
	transport := &sentry.MockTransport{}
	observer := New(Config{Env: "test", SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
	err := errors.Join(fmt.Errorf("Bearer credential-canary: %w", auth.ErrRecordNotFound), context.DeadlineExceeded)
	observer.CaptureError(context.Background(), EventContext{"operation": "post.read", "result": "terminal"}, err)
	observer.Flush(time.Second)
	events := transport.Events()
	if len(events) != 1 {
		t.Fatalf("events = %d", len(events))
	}
	data, _ := json.Marshal(events[0])
	if strings.Contains(string(data), "credential-canary") {
		t.Fatal("opaque wrapper leaked")
	}
	exceptions := events[0].Exception
	if len(exceptions) != 4 {
		t.Fatalf("causes = %d, want joined/wrapper/two causes", len(exceptions))
	}
	for _, want := range []string{auth.ErrRecordNotFound.Error(), context.DeadlineExceeded.Error()} {
		found := false
		for _, cause := range exceptions {
			if cause.Value == want {
				found = true
			}
		}
		if !found {
			t.Fatalf("known-safe explanation %q lost: %#v", want, exceptions)
		}
	}
}

// UT-002 / FR-001 / AC-002: stack is captured while the panic is recovered.
func TestDiagnosticRecoveredStack(t *testing.T) {
	transport := &sentry.MockTransport{}
	observer := New(Config{Env: "test", SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
	func() {
		defer func() { observer.CapturePanic(context.Background(), EventContext{"component": "http"}, recover()) }()
		panicAtDiagnosticBoundary()
	}()
	observer.Flush(time.Second)
	events := transport.Events()
	if len(events) != 1 {
		t.Fatalf("events = %d", len(events))
	}
	exception := events[0].Exception[0]
	if exception.Stacktrace == nil {
		t.Fatal("recovery-time stack missing")
	}
	found := false
	for _, frame := range exception.Stacktrace.Frames {
		if strings.Contains(frame.AbsPath, "/Users/") {
			t.Fatal("local user path leaked")
		}
		if strings.Contains(frame.Function, "panicAtDiagnosticBoundary") {
			found = true
		}
	}
	if !found {
		t.Fatalf("useful recovery frame absent: %#v", exception.Stacktrace)
	}
	if exception.Type != "*runtime.TypeAssertionError" || !strings.Contains(exception.Value, "int, not string") {
		t.Fatalf("safe panic details lost: %#v", exception)
	}
}

func panicAtDiagnosticBoundary() {
	var value any = 1
	_ = value.(string)
}

// UT-004 / FR-002 / AC-001, AC-006: public context is independent of prose.
func TestDiagnosticPublicActorTarget(t *testing.T) {
	transport := &sentry.MockTransport{}
	observer := New(Config{Env: "test", SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
	observer.CaptureDiagnostic(context.Background(), DiagnosticInput{
		Error:   errors.New("opaque linen sentence"),
		Context: EventContext{"operation": "post.read"},
		Workflow: PublicRecordContext{
			ActorDID: syntax.DID("did:plc:actor"), TargetDID: syntax.DID("did:plc:target"),
			URI:    syntax.ATURI("at://did:plc:target/social.craftsky.feed.post/record"),
			Handle: syntax.Handle("target.example.invalid"), CID: syntax.CID("public-cid"),
			NSID: syntax.NSID("social.craftsky.feed.post"), RecordKey: syntax.RecordKey("record"),
		},
	})
	observer.Flush(time.Second)
	events := transport.Events()
	if len(events) != 1 {
		t.Fatalf("events=%d", len(events))
	}
	diagnostic := events[0].Contexts["diagnostic"]
	for key, want := range map[string]string{"actor_did": "did:plc:actor", "target_did": "did:plc:target", "record_uri": "at://did:plc:target/social.craftsky.feed.post/record", "handle": "target.example.invalid", "cid": "public-cid", "nsid": "social.craftsky.feed.post", "record_key": "record"} {
		if diagnostic[key] != want {
			t.Fatalf("%s=%v, want %s", key, diagnostic[key], want)
		}
		if _, ok := events[0].Tags[key]; ok {
			t.Fatalf("public identity became grouping tag: %s", key)
		}
	}
	data, _ := json.Marshal(events[0])
	if strings.Contains(string(data), "opaque linen sentence") {
		t.Fatal("public context authorized unknown prose")
	}
}

func TestDiagnosticAttemptedPublicIdentifier(t *testing.T) {
	transport := &sentry.MockTransport{}
	observer := New(Config{Env: "test", SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
	observer.CaptureDiagnostic(context.Background(), DiagnosticInput{
		Error:    errors.New("private validation detail"),
		Workflow: AttemptedPublicDIDContext{Value: "did:plc:bad!"},
	})
	observer.Flush(time.Second)
	events := transport.Events()
	if len(events) != 1 {
		t.Fatalf("events=%d", len(events))
	}
	fields := events[0].Contexts["diagnostic"]
	if fields["attempted_did"] != "did:plc:bad!" || fields["identifier_valid"] != false || fields["validation_reason"] != "invalid DID" {
		t.Fatalf("attempt not identified safely: %#v", fields)
	}
}

// UT-005 / FR-002, FR-011 / AC-006: matching bytes do not establish provenance.
func TestDiagnosticPublishedExcerptRequiresFailureProvenance(t *testing.T) {
	for _, tc := range []struct {
		name       string
		err        error
		workflow   WorkflowContext
		wantText   bool
		wantEvents int
	}{
		{"published parse failure", errors.New("opaque linen sentence"), PublishedRecordParseFailureContext{Text: "public linen pattern access_token=credential-canary"}, true, 1},
		{"draft", errors.New("public linen pattern access_token=credential-canary"), nil, false, 1},
		{"failed publication", errors.New("public linen pattern access_token=credential-canary"), nil, false, 1},
		{"routine success", nil, PublishedRecordParseFailureContext{Text: "public linen pattern access_token=credential-canary"}, false, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			transport := &sentry.MockTransport{}
			observer := New(Config{Env: "test", SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
			observer.CaptureDiagnostic(context.Background(), DiagnosticInput{Error: tc.err, Workflow: tc.workflow})
			observer.Flush(time.Second)
			events := transport.Events()
			if len(events) != tc.wantEvents {
				t.Fatalf("events=%d", len(events))
			}
			data, _ := json.Marshal(events)
			if strings.Contains(string(data), "public linen pattern") != tc.wantText {
				t.Fatalf("wrong excerpt selection: %s", data)
			}
			if strings.Contains(string(data), "credential-canary") {
				t.Fatal("credential leaked in excerpt")
			}
		})
	}
}

// UT-012 / FR-010, RULE-001–003: selected published text is protected too.
func TestDiagnosticAdmittedMixedFormats(t *testing.T) {
	text := "public linen did:plc:target at://did:plc:target/social.craftsky.feed.post/record " +
		"Cookie=session-cookie-canary email=user-email-canary@example.invalid " +
		"https://userinfo-canary:password-canary@example.invalid/private/capability-canary?X-Amz-Signature=signature-canary " +
		"code%3Dencoded-code-canary%26state%3Dencoded-state-canary " +
		"/Users/local-path-canary/project/file.dart " +
		"-----BEGIN PRIVATE KEY-----\nprivate-key-canary\n-----END PRIVATE KEY-----"
	got := sanitizeKnownDiagnosticText(text)
	for _, forbidden := range []string{"cookie-canary", "email-canary", "userinfo-canary", "password-canary", "capability-canary", "signature-canary", "code-canary", "state-canary", "local-path-canary", "private-key-canary"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("protected value %s leaked: %s", forbidden, got)
		}
	}
	for _, want := range []string{"public linen", "did:plc:target", "at://did:plc:target/social.craftsky.feed.post/record"} {
		if !strings.Contains(got, want) {
			t.Fatalf("useful public diagnostic lost: %s", got)
		}
	}
}

type cyclicDiagnosticError struct{}

func (*cyclicDiagnosticError) Error() string   { return "opaque cyclic prose" }
func (e *cyclicDiagnosticError) Unwrap() error { return e }

// UT-013 / FR-010, FR-011, NFR-002: termination and explicit omission.
func TestDiagnosticBounds(t *testing.T) {
	for _, size := range []int{MaxDiagnosticTextBytes - 1, MaxDiagnosticTextBytes, MaxDiagnosticTextBytes + 1} {
		input := strings.Repeat("x", size)
		got := sanitizeKnownDiagnosticText(input)
		if len(got) > MaxDiagnosticTextBytes || !utf8.ValidString(got) {
			t.Fatalf("invalid bound: %d", len(got))
		}
		if size > MaxDiagnosticTextBytes && !strings.HasSuffix(got, "[TRUNCATED]") {
			t.Fatal("omission marker absent")
		}
	}
	input := strings.Repeat("🧶", 1024) + " access_token=" + strings.Repeat("secret-canary", 512)
	got := sanitizeKnownDiagnosticText(input)
	if len(got) > MaxDiagnosticTextBytes || !utf8.ValidString(got) || strings.Contains(got, "secret-canary") {
		t.Fatal("UTF8/credential boundary unsafe")
	}
	done := make(chan []DiagnosticCause, 1)
	go func() { done <- DescribeError(&cyclicDiagnosticError{}, EventContext{}) }()
	select {
	case causes := <-done:
		if len(causes) > 8 || len(causes) < 2 || causes[len(causes)-1].Type != "DiagnosticOmission" {
			t.Fatalf("cycle omission absent: %#v", causes)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("cyclic causes did not terminate")
	}
}

// AT-007 / FR-010, RULE-001–003: final SDK enrichment cannot bypass policy.
func TestDiagnosticFinalSDKEvent(t *testing.T) {
	transport := &sentry.MockTransport{}
	observer := New(Config{Env: "test", SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
	observer.sentryHub.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetContext("unknown", sentry.Context{"nested": map[string]any{"draft": "private-draft-canary"}})
		scope.SetUser(sentry.User{Email: "email-canary@example.invalid", ID: "private-device-canary"})
		scope.SetTag("operation", "opaque private operation")
		scope.SetRequest(&http.Request{URL: mustDiagnosticURL(t, "https://user:password-canary@example.invalid/capability-canary?code=code-canary"), Header: http.Header{"Cookie": {"cookie-canary"}}})
		scope.AddBreadcrumb(&sentry.Breadcrumb{Message: "opaque private breadcrumb", Data: map[string]any{"draft": "private-draft-canary"}}, 50)
	})
	observer.CaptureDiagnostic(context.Background(), DiagnosticInput{Error: auth.ErrRecordNotFound, Context: EventContext{"operation": "post.read", "result": "terminal"}, Workflow: PublicRecordContext{TargetDID: syntax.DID("did:plc:target")}})
	observer.sentryHub.CaptureEvent(&sentry.Event{Message: "opaque private SDK prose", Modules: map[string]string{"private-module-canary": "private-version-canary"}, Exception: []sentry.Exception{{Type: "ProviderFailure", Value: "opaque private SDK prose", Stacktrace: &sentry.Stacktrace{Frames: []sentry.Frame{{Function: "loadPublicRecord", Filename: "/Users/path-canary/private/file.go", Vars: map[string]any{"draft": "private-draft-canary"}, ContextLine: "private-line-canary"}}}}}})
	observer.Flush(time.Second)
	events := transport.Events()
	if len(events) != 2 {
		t.Fatalf("events=%d", len(events))
	}
	data, _ := json.Marshal(events)
	for _, bad := range []string{"private-draft-canary", "email-canary", "private-device-canary", "password-canary", "capability-canary", "code-canary", "cookie-canary", "opaque private", "path-canary", "private-line-canary", "private-module-canary", "private-version-canary"} {
		if strings.Contains(string(data), bad) {
			t.Fatalf("final SDK leak %s: %s", bad, data)
		}
	}
	for _, want := range []string{"did:plc:target", "pds: record not found", "ProviderFailure", "loadPublicRecord"} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("positive lost %s: %s", want, data)
		}
	}
}
func mustDiagnosticURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	value, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func TestDiagnosticFinalRecordBudgets(t *testing.T) {
	transport := &sentry.MockTransport{}
	observer := New(Config{Env: "test", SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
	fields := EventContext{"operation": "post.read"}
	for key := range allowedEventContextKeys {
		if key != "operation" {
			fields[key] = strings.Repeat("x", 32768)
		}
	}
	observer.CaptureDiagnostic(context.Background(), DiagnosticInput{Error: errors.New("opaque private prose"), Context: fields, Workflow: PublicRecordContext{CID: syntax.CID(strings.Repeat("c", 32768)), TargetDID: syntax.DID("did:plc:target")}})
	observer.Flush(time.Second)
	for _, event := range transport.Events() {
		owned, _ := json.Marshal(struct {
			Message    string
			Tags       map[string]string
			Contexts   map[string]sentry.Context
			Exceptions []sentry.Exception
		}{event.Message, event.Tags, event.Contexts, event.Exception})
		if len(owned) > 65536 {
			t.Fatalf("owned event budget exceeded: %d", len(owned))
		}
		for _, cause := range event.Exception {
			if cause.Stacktrace != nil && len(cause.Stacktrace.Frames) > 64 {
				t.Fatal("frame count exceeded")
			}
		}
		if !strings.Contains(string(owned), "OMITTED") {
			t.Fatal("budget omission marker absent")
		}
	}
	var output bytes.Buffer
	logger := slog.New(NewDiagnosticHandler(slog.NewJSONHandler(&output, nil)))
	logger.Error("operation failed", slog.Any("causes", DescribeError(errors.New("private"), EventContext{})), slog.String("operation", strings.Repeat("x", 32768)), slog.Any("diagnostic", PublishedRecordParseFailureContext{Record: PublicRecordContext{TargetDID: syntax.DID("did:plc:target")}, Text: strings.Repeat("🧶", 10000)}))
	if output.Len() > 16384 {
		t.Fatalf("local budget exceeded: %d", output.Len())
	}
	if !strings.Contains(output.String(), "*errors.errorString") {
		t.Fatal("core cause type lost under budget pressure")
	}
}

// UT-006 / FR-003: the request ID is context, never an issue grouping tag.
func TestDiagnosticRequestCorrelationWithoutSampling(t *testing.T) {
	const id = "b1b5a67d-0480-40b3-a9dc-201354ef91ed"
	for _, dsn := range []string{"", "https://public@example.invalid/1"} {
		t.Run(fmt.Sprint(dsn != ""), func(t *testing.T) {
			var local bytes.Buffer
			transport := &sentry.MockTransport{}
			logger := slog.New(NewDiagnosticHandler(slog.NewJSONHandler(&local, nil)))
			observer := New(Config{Env: "test", Release: "test-release", SentryDSN: dsn, SentryTransport: transport, LogsEnabled: true, TracingEnabled: false, Logger: logger})
			ctx := ctxkeys.WithRunID(context.Background(), id)
			input := DiagnosticInput{Error: errors.New("opaque private failure"), Context: EventContext{"operation": "post.read"}}
			LogDiagnostic(ctx, logger, input)
			observer.CaptureDiagnostic(ctx, input)
			observer.Flush(time.Second)
			if !strings.Contains(local.String(), id) {
				t.Fatalf("local correlation lost: %s", local.String())
			}
			for _, event := range transport.Events() {
				if len(event.Logs) > 0 {
					if event.Logs[0].Attributes["run_id"].AsString() != id {
						t.Fatal("export correlation lost")
					}
					continue
				}
				if event.Contexts["correlation"]["run_id"] != id {
					t.Fatalf("issue correlation lost: %#v", event.Contexts)
				}
				if _, exists := event.Tags["run_id"]; exists {
					t.Fatal("request id became generic issue tag")
				}
				if event.Release != "test-release" || event.Environment != "test" {
					t.Fatal("process metadata lost")
				}
			}
		})
	}
	observer := New(Config{})
	observer.CaptureDiagnostic(context.Background(), DiagnosticInput{Error: errors.New("no request"), Context: EventContext{}})
	if _, ok := SanitizeEventContext(EventContext{})["run_id"]; ok {
		t.Fatal("fabricated missing correlation")
	}
}

// IT-004 / FR-003: configured tracing without a backend is not a real span.
func TestDiagnosticNoDSNDoesNotInventTrace(t *testing.T) {
	observer := New(Config{TracingEnabled: true})
	ctx, span := observer.StartSpan(context.Background(), SpanContext{Operation: "post.read", Component: "api"})
	traceID, spanID := TraceIDs(ctx)
	if traceID != "" || spanID != "" || span.Enabled() {
		t.Fatal("missing backend fabricated trace/span correlation")
	}
}

// UT-010 / FR-007: ownership is per occurrence, including disabled sinks.
func TestDiagnosticCaptureOwnershipPerOccurrence(t *testing.T) {
	cause := errors.New("same opaque failure")
	for _, dsn := range []string{"", "https://public@example.invalid/1"} {
		t.Run(fmt.Sprint(dsn != ""), func(t *testing.T) {
			transport := &sentry.MockTransport{}
			observer := New(Config{Env: "test", SentryDSN: dsn, SentryTransport: transport})
			for i := 0; i < 2; i++ {
				ctx := WithCaptureMarker(context.Background())
				observer.CaptureDiagnostic(ctx, DiagnosticInput{Error: cause, Context: EventContext{"operation": "post.read"}})
				if !CaptureRecorded(ctx) {
					t.Fatal("owner missing when remote unavailable")
				}
				observer.CaptureError(ctx, EventContext{"operation": "http.server"}, cause)
			}
			observer.Flush(time.Second)
			want := 0
			if dsn != "" {
				want = 2
			}
			if len(transport.Events()) != want {
				t.Fatalf("issues=%d want%d per occurrence", len(transport.Events()), want)
			}
		})
	}
}

// AT-006 / FR-009, RULE-002: a public account is useful only on a private
// operational failure; private targets and success history remain absent.
func TestPrivateOperationalFailureContext(t *testing.T) {
	var local bytes.Buffer
	transport := &sentry.MockTransport{}
	observer := New(Config{Env: "test", SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
	ctx := WithRequestDiagnosticContext(context.Background(), RequestDiagnosticContext{Path: "/v1/scheduled-posts/[REDACTED]", PublicTargets: false})
	cause := &pgconn.PgError{Code: "40001", Message: "opaque schedule text canary", Detail: "private recipient canary"}
	workflowID := "00000000-0000-4000-8000-000000000401"
	workflow := PrivateFailureContext(cause, syntax.DID("did:plc:alice"), workflowID)
	input := DiagnosticInput{Error: cause, Context: EventContext{"operation": "schedule.publish", "failure_stage": "prepare", "attempt": 2, "result": "terminal"}, Workflow: workflow}
	LogDiagnostic(ctx, slog.New(slog.NewJSONHandler(&local, nil)), input)
	observer.CaptureDiagnostic(ctx, input)
	observer.Flush(time.Second)
	if len(transport.Events()) != 1 {
		t.Fatal("terminal failure not captured")
	}
	payload, _ := json.Marshal(transport.Events()[0])
	for name, output := range map[string]string{"local": local.String(), "event": string(payload)} {
		for _, positive := range []string{"did:plc:alice", workflowID, "40001", "prepare"} {
			if !strings.Contains(output, positive) {
				t.Fatalf("%s omitted %s: %s", name, positive, output)
			}
		}
		for _, negative := range []string{"opaque schedule", "private recipient"} {
			if strings.Contains(output, negative) {
				t.Fatalf("%s leaked %s", name, negative)
			}
		}
	}
	if PrivateFailureContext(nil, syntax.DID("did:plc:alice"), workflowID) != nil {
		t.Fatal("success acquired private activity context")
	}
}

// IT-007: private HTTP faults may identify only the authenticated operation owner.
func TestPrivateRequestFailureSelectsAuthenticatedOwner(t *testing.T) {
	var local bytes.Buffer
	transport := &sentry.MockTransport{}
	observer := New(Config{SentryDSN: "https://public@example.invalid/1", SentryTransport: transport, Logger: slog.New(slog.NewJSONHandler(&local, nil))})
	ctx := WithRequestObserver(WithRequestDiagnosticContext(ctxkeys.WithDID(context.Background(), syntax.DID("did:plc:initiator")), RequestDiagnosticContext{Path: "/v1/scheduled-posts/[REDACTED]", PublicTargets: false}), observer)
	cause := &pgconn.PgError{Code: "40001", Message: "PRIVATE_DRAFT_TEXT", Detail: "did:plc:private-target"}
	ReportRequestFailure(ctx, cause, "schedule.save", "persistence")
	observer.Flush(time.Second)
	if len(transport.Events()) != 1 {
		t.Fatal("missing issue")
	}
	payload, _ := json.Marshal(transport.Events()[0])
	for _, output := range []string{local.String(), string(payload)} {
		for _, selected := range []string{"did:plc:initiator", "operation_account_did", "40001", "persistence"} {
			if !strings.Contains(output, selected) {
				t.Errorf("missing %s in %s", selected, output)
			}
		}
		for _, private := range []string{"PRIVATE_DRAFT_TEXT", "did:plc:private-target", "actor_did"} {
			if strings.Contains(output, private) {
				t.Errorf("leaked %s", private)
			}
		}
	}
}

func TestIT007InstagramProviderCauseReachesSelectedSinks(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(429)
		_, _ = w.Write([]byte(`{"error":{"code":613,"message":"PRIVATE_INSTAGRAM_PAYLOAD"}}`))
	}))
	defer server.Close()
	client, err := instagrammeta.NewHTTPClient(instagrammeta.HTTPClientConfig{HTTPClient: server.Client(), BaseURL: server.URL, APIVersion: "v99.0", AccessToken: "PRIVATE_TOKEN", OfficialAccountID: "PRIVATE_ACCOUNT"})
	if err != nil {
		t.Fatal(err)
	}
	_, cause := client.LookupUsername(context.Background(), "PRIVATE_RECIPIENT")
	var output bytes.Buffer
	observer := New(Config{Logger: slog.New(slog.NewJSONHandler(&output, nil))})
	observer.ObservePrivateFailure(context.Background(), cause, syntax.DID("did:plc:owner"), "40000000-0000-4000-8000-000000000001", "instagram.verify", "profile_lookup", "retry", 2)
	for _, selected := range []string{"429", "rateLimited", "ProviderError", "did:plc:owner"} {
		if !strings.Contains(output.String(), selected) {
			t.Errorf("missing %s in %s", selected, output.String())
		}
	}
	if strings.Contains(output.String(), "PRIVATE_") {
		t.Error("provider canary leaked")
	}
}

func TestUT015WorkflowAndRequestIDsDoNotChangeAggregation(t *testing.T) {
	transport := &sentry.MockTransport{}
	recorder := NewInMemoryMetricRecorder()
	observer := New(Config{Env: "test", SentryDSN: "https://public@example.invalid/1", SentryTransport: transport, MetricRecorder: recorder, TracingEnabled: true, TracesSampleRate: 1})
	var tags string
	seen := map[string]bool{}
	for i := 0; i < 20; i++ {
		id := uuid.NewString()
		actor := syntax.DID(fmt.Sprintf("did:plc:actor%d", i))
		workflowID := uuid.NewString()
		ctx := ctxkeys.WithRunID(context.Background(), id)
		ctx, span := observer.StartSpan(ctx, SpanContext{Operation: "http.server", Component: "http", Attributes: EventContext{"route_pattern": "/v1/scheduled-posts"}})
		recorder.HTTPRequestFinished(ctx, "POST", "/v1/scheduled-posts", 500, time.Millisecond, 40)
		observer.CaptureDiagnostic(ctx, DiagnosticInput{Error: &pgconn.PgError{Code: "40001", Message: "PRIVATE_" + id}, Context: EventContext{"operation": "schedule.publish", "failure_stage": "record_write", "result": "terminal"}, Workflow: PrivateFailureContext(errors.New("failure"), actor, workflowID)})
		span.Finish("error")
		observer.Flush(time.Second)
	}
	issues := 0
	transactions := 0
	for _, event := range transport.Events() {
		if event.Type == "transaction" {
			transactions++
			if event.Transaction != "http.server" && event.Transaction != "/v1/scheduled-posts" {
				t.Errorf("dynamic transaction %q", event.Transaction)
			}
			continue
		}
		issues++
		raw, _ := json.Marshal(event.Tags)
		if tags == "" {
			tags = string(raw)
		} else if tags != string(raw) {
			t.Errorf("dynamic tags %s vs %s", tags, raw)
		}
		if len(event.Fingerprint) > 0 {
			t.Fatal("dynamic grouping override")
		}
		correlation := event.Contexts["correlation"]
		diagnostic := event.Contexts["diagnostic"]
		if correlation["run_id"] == nil || diagnostic["workflow_ref"] == nil || diagnostic["operation_account_did"] == nil {
			t.Fatal("lost diagnostic distinction")
		}
		seen[fmt.Sprint(correlation["run_id"])] = true
	}
	if issues != 20 || transactions != 20 || len(seen) != 20 {
		t.Fatalf("issues=%d transactions=%d distinct=%d", issues, transactions, len(seen))
	}
	for _, call := range recorder.Calls() {
		for key, value := range call.Attributes {
			if strings.Contains(value, "did:") || strings.Contains(value, "40001") || key == "run_id" || key == "workflow_ref" {
				t.Fatalf("dynamic metric %#v", call)
			}
		}
	}
}

func TestUT016RecoverableRetriesSummarizeWithoutLosingTerminalCause(t *testing.T) {
	var local bytes.Buffer
	now := time.Unix(1000, 0)

	transport := &sentry.MockTransport{}
	observer := New(Config{DiagnosticClock: func() time.Time { return now }, SentryDSN: "https://public@example.invalid/1", SentryTransport: transport, Logger: slog.New(slog.NewJSONHandler(&local, nil))})
	workflow := uuid.NewString()
	emit := func(outcome, code string) {
		observer.ObservePrivateFailure(context.Background(), &pgconn.PgError{Code: code, Message: "PRIVATE_RETRY"}, syntax.DID("did:plc:owner"), workflow, "schedule.publish", "record_write", outcome, 1)
	}
	for i := 0; i < 10; i++ {
		emit("retry", "40001")
	}
	now = now.Add(30 * time.Second)
	emit("retry", "40001")
	for i := 0; i < 3; i++ {
		emit("retry", "40001")
	}
	emit("terminal", "08006")
	observer.Flush(time.Second)
	lines := strings.Split(strings.TrimSpace(local.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("emitted %d logs, want first/window/terminal", len(lines))
	}
	for i, want := range []float64{0, 9, 3} {
		var record map[string]any
		if err := json.Unmarshal([]byte(lines[i]), &record); err != nil {
			t.Fatal(err)
		}
		got, _ := record["suppressed_count"].(float64)
		if got != want {
			t.Errorf("record%d suppressed=%v want%v", i, got, want)
		}
	}
	if !strings.Contains(lines[2], "08006") || !strings.Contains(lines[2], "terminal") {
		t.Fatal("terminal cause lost")
	}
	if len(transport.Events()) != 1 {
		t.Fatalf("issues=%d", len(transport.Events()))
	}
	if strings.Contains(local.String(), "PRIVATE_RETRY") {
		t.Fatal("private retry prose leaked")
	}
}

func TestUT016RetryScopeCapacityExpiresAndUnexpectedOccurrencesSurvive(t *testing.T) {
	var local bytes.Buffer
	now := time.Unix(1000, 0)
	observer := New(Config{DiagnosticClock: func() time.Time { return now }, Logger: slog.New(slog.NewJSONHandler(&local, nil))})
	emit := func(workflow, outcome string) {
		observer.ObservePrivateFailure(context.Background(), errors.New("PRIVATE_CAPACITY"), syntax.DID("did:plc:owner"), workflow, "schedule.publish", "record_write", outcome, 1)
	}
	for i := 0; i < 128; i++ {
		workflow := uuid.NewString()
		emit(workflow, "retry")
		emit(workflow, "retry")
	}
	overflow := uuid.NewString()
	emit(overflow, "retry")
	emit(overflow, "retry")
	now = now.Add(61 * time.Second)
	emit(overflow, "retry")
	emit(overflow, "retry")
	emit(overflow, "terminal")
	emit(overflow, "error")
	emit(overflow, "error")
	lines := strings.Split(strings.TrimSpace(local.String()), "\n")
	if len(lines) != 134 {
		t.Fatalf("emitted %d, want134 including expiry and independent occurrences", len(lines))
	}
	if !strings.Contains(lines[131], `"suppressed_count":1`) {
		t.Fatal("terminal lost post-expiry summary")
	}
	if strings.Contains(local.String(), "PRIVATE_CAPACITY") {
		t.Fatal("private cause leaked")
	}
}

// SIM-T05 / IR-005: SDK thread enrichment must not bypass exception privacy.
func TestDiagnosticFinalSDKOmitsThreadPrivateData(t *testing.T) {
	transport := &sentry.MockTransport{}
	observer := New(Config{Env: "test", Release: "review-release", SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
	observer.sentryHub.CaptureEvent(&sentry.Event{Exception: []sentry.Exception{{Type: "ReviewFailure", Value: "private-exception-canary", Stacktrace: &sentry.Stacktrace{Frames: []sentry.Frame{{Function: "readRecord", Filename: "record.go", Lineno: 12}}}}}, Threads: []sentry.Thread{{ID: "1", Stacktrace: &sentry.Stacktrace{Frames: []sentry.Frame{{Function: "readRecord", AbsPath: "/Users/private-path-canary/file.go", Vars: map[string]any{"draft": "private-thread-canary"}, ContextLine: "private-source-canary"}}}}}})
	observer.Flush(time.Second)
	data, _ := json.Marshal(transport.Events())
	if !strings.Contains(string(data), "ReviewFailure") || !strings.Contains(string(data), "review-release") || !strings.Contains(string(data), "readRecord") || !strings.Contains(string(data), "record.go") {
		t.Fatal("positive fields absent")
	}
	for _, v := range []string{"private-thread-canary", "private-path-canary", "private-source-canary", "private-exception-canary"} {
		if strings.Contains(string(data), v) {
			t.Errorf("serialized SDK retained %s", v)
		}
	}
}

// SDK-T02: use the SDK's native chained mechanisms and supported origin stacks.
func TestSDKNativeCauseChainAndOriginStack(t *testing.T) {
	transport := &sentry.MockTransport{}
	observer := New(Config{Env: "test", SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
	pcs := make([]uintptr, 8)
	n := runtime.Callers(1, pcs)
	root := &sdkStackError{pcs: pcs[:n]}
	observer.CaptureDiagnostic(ctxkeys.WithRunID(context.Background(), "b1b5a67d-0480-40b3-a9dc-201354ef91ed"), DiagnosticInput{Error: fmt.Errorf("private-wrapper: %w", root), Context: EventContext{"operation": "post.read", "request_id": "b1b5a67d-0480-40b3-a9dc-201354ef91ed"}})
	observer.Flush(time.Second)
	event := transport.Events()[0]
	data, _ := json.Marshal(event)
	if len(event.Exception) != 2 || event.Exception[0].Stacktrace == nil || event.Exception[0].Mechanism == nil {
		t.Fatalf("native chain/origin stack missing: %#v", event.Exception)
	}
	if event.Exception[1].Stacktrace != nil {
		t.Fatal("stackless wrapper acquired a fabricated origin stack")
	}
	if !strings.Contains(string(data), "TestSDKNativeCauseChainAndOriginStack") || !strings.Contains(string(data), "b1b5a67d") {
		t.Fatalf("context lost: %s", data)
	}
	if strings.Contains(string(data), "private-wrapper") || strings.Contains(string(data), "private-root") {
		t.Fatalf("private prose leaked: %s", data)
	}
}

type sdkStackError struct{ pcs []uintptr }

func (*sdkStackError) Error() string           { return "private-root" }
func (e *sdkStackError) StackTrace() []uintptr { return e.pcs }

func TestSDKReviewedExplanationSurvivesBothSinks(t *testing.T) {
	transport := &sentry.MockTransport{}
	var local bytes.Buffer
	logger := slog.New(NewDiagnosticHandler(slog.NewJSONHandler(&local, nil)))
	observer := New(Config{Env: "test", Logger: logger, SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
	input := DiagnosticInput{Error: WrapError("claim scheduled posts: token=credential-canary", errors.New("private-scheduled-content")), Context: EventContext{"operation": "schedule.claim"}}
	LogDiagnostic(context.Background(), logger, input)
	observer.CaptureDiagnostic(context.Background(), input)
	observer.Flush(time.Second)
	data, _ := json.Marshal(transport.Events()[0])
	for name, payload := range map[string]string{"local": local.String(), "event": string(data)} {
		if !strings.Contains(payload, "claim scheduled posts") {
			t.Errorf("%s explanation missing: %s", name, payload)
		}
		for _, secret := range []string{"credential-canary", "private-scheduled-content"} {
			if strings.Contains(payload, secret) {
				t.Errorf("%s private value leaked: %s", name, payload)
			}
		}
	}
}

func TestSDKSafeBreadcrumbTimeline(t *testing.T) {
	transport := &sentry.MockTransport{}
	observer := New(Config{Env: "test", SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
	hub := observer.sentryHub.Clone()
	hub.AddBreadcrumb(&sentry.Breadcrumb{Category: "operation", Message: "post read started", Data: map[string]any{"operation": "post.read", "draft": "private-breadcrumb-draft"}}, nil)
	hub.AddBreadcrumb(&sentry.Breadcrumb{Category: "http", Message: "private-http-body", Data: map[string]any{"method": "GET", "status_code": 503, "url": "https://private.test/?token=private-url-token"}}, nil)
	hub.AddBreadcrumb(&sentry.Breadcrumb{Category: "unknown", Message: "private-unknown-prose"}, nil)
	observer.CaptureDiagnostic(sentry.SetHubOnContext(context.Background(), hub), DiagnosticInput{Error: errors.New("private-error"), Context: EventContext{"operation": "post.read"}})
	observer.Flush(time.Second)
	data, _ := json.Marshal(transport.Events()[0])
	if !strings.Contains(string(data), "post read started") || !strings.Contains(string(data), "503") {
		t.Fatalf("timeline lost: %s", data)
	}
	for _, private := range []string{"private-breadcrumb-draft", "private-http-body", "private-url-token", "private-unknown-prose", "private-error"} {
		if strings.Contains(string(data), private) {
			t.Fatalf("private timeline leaked: %s", data)
		}
	}
}

func TestSDKFormattingFailureRetainsTypedLocalFallback(t *testing.T) {
	var local bytes.Buffer
	observer := New(Config{Env: "test", Logger: slog.New(slog.NewJSONHandler(&local, nil)), SentryDSN: "https://public@example.invalid/1", SentryTransport: &sentry.MockTransport{}})
	observer.CaptureDiagnostic(context.Background(), DiagnosticInput{Error: &sdkBrokenError{}, Context: EventContext{"operation": "post.read"}})
	if !strings.Contains(local.String(), "sdkBrokenError") || !strings.Contains(local.String(), "post.read") {
		t.Fatalf("typed fallback lost: %s", local.String())
	}
	if strings.Contains(local.String(), "private-formatting-panic") {
		t.Fatal("formatting panic prose leaked")
	}
}

type sdkBrokenError struct{}

func (*sdkBrokenError) Error() string { panic("private-formatting-panic") }

// SDK-T06: hostile graphs still produce a bounded, typed issue.
func TestSDKCyclicCauseStillCapturesIssue(t *testing.T) {
	transport := &sentry.MockTransport{}
	observer := New(Config{Env: "test", SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
	observer.CaptureDiagnostic(context.Background(), DiagnosticInput{Error: &cyclicDiagnosticError{}, Context: EventContext{"operation": "post.read"}})
	observer.Flush(time.Second)
	if len(transport.Events()) != 1 {
		t.Fatalf("issue lost for cyclic cause: %d events", len(transport.Events()))
	}
	data, _ := json.Marshal(transport.Events()[0])
	if !strings.Contains(string(data), "cyclicDiagnosticError") || !strings.Contains(string(data), "DiagnosticOmission") {
		t.Fatalf("bounded typed chain missing: %s", data)
	}
	if strings.Contains(string(data), "opaque cyclic prose") {
		t.Fatal("private prose leaked")
	}
}

func TestExhaustedValidationRemainsActionable(t *testing.T) {
	var local bytes.Buffer
	logger := slog.New(NewDiagnosticHandler(slog.NewJSONHandler(&local, nil)))
	transport := &sentry.MockTransport{}
	observer := New(Config{Env: "test", Logger: logger, SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
	input := DiagnosticInput{Error: &lexiconschema.ValidationError{Cause: errors.New("private-validation-canary")}, Context: EventContext{"error_category": "validation", "result": "exhausted", "operation": "post.read"}}
	LogDiagnostic(context.Background(), logger, input)
	observer.CaptureDiagnostic(context.Background(), input)
	observer.Flush(time.Second)
	if !strings.Contains(local.String(), `"level":"ERROR"`) || len(transport.Events()) != 1 {
		t.Fatalf("exhausted failure downgraded: %s events=%d", local.String(), len(transport.Events()))
	}
	data, _ := json.Marshal(transport.Events())
	if strings.Contains(string(data)+local.String(), "private-validation-canary") {
		t.Error("validator prose leaked")
	}
}
