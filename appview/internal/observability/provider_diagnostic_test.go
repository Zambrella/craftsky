package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"google.golang.org/api/option"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ctxkeys"
	"social.craftsky/appview/internal/integrations/revenuecat"
	"social.craftsky/appview/internal/linkpreview"
)

type diagnosticPreviewFetcher struct{}

func (diagnosticPreviewFetcher) FetchPage(context.Context, string) (*http.Response, *url.URL, error) {
	return &http.Response{StatusCode: http.StatusServiceUnavailable, Body: io.NopCloser(strings.NewReader("PRIVATE_PREVIEW_BODY")), Header: http.Header{"X-Private": {"PRIVATE_HEADER"}}}, nil, nil
}
func (fetcher diagnosticPreviewFetcher) FetchJSON(ctx context.Context, target string) (*http.Response, *url.URL, error) {
	return fetcher.FetchPage(ctx, target)
}
func (fetcher diagnosticPreviewFetcher) FetchImage(ctx context.Context, target string) (*http.Response, *url.URL, error) {
	return fetcher.FetchPage(ctx, target)
}

func TestLinkPreviewFailuresRetainOnlyUpstreamStatus(t *testing.T) {
	for _, target := range []string{"https://example.invalid/PRIVATE_PATH?token=PRIVATE_CAPABILITY", "https://www.youtube.com/watch?v=dQw4w9WgXcQ"} {
		_, cause := linkpreview.NewService(diagnosticPreviewFetcher{}).FetchPreview(context.Background(), target)
		if !errors.Is(cause, linkpreview.ErrUpstream) {
			t.Fatalf("upstream taxonomy lost: %v", cause)
		}
		var local bytes.Buffer
		transport := &sentry.MockTransport{}
		observer := New(Config{Logger: slog.New(NewDiagnosticHandler(slog.NewJSONHandler(&local, nil))), LogsEnabled: true, SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
		observer.ObservePrivateFailure(context.Background(), cause, "did:plc:initiator", "", "link.preview", "fetch", "error", 1)
		observer.Flush(time.Second)
		issues := 0
		for _, event := range transport.Events() {
			if len(event.Exception) == 0 {
				continue
			}
			issues++
			if event.Tags["upstream_status"] != "503" || event.Exception[len(event.Exception)-1].Type != "*linkpreview.UpstreamError" {
				t.Fatalf("upstream evidence lost: %+v", event)
			}
		}
		if issues != 1 {
			t.Fatalf("issues=%d want1", issues)
		}
		serialized, err := json.Marshal(transport.Events())
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(local.String(), "503") || !strings.Contains(string(serialized), "link preview upstream failed (HTTP 503)") {
			t.Fatal("selected explanation/status missing")
		}
		if strings.Contains(local.String()+string(serialized), "PRIVATE_") || strings.Contains(local.String()+string(serialized), target) {
			t.Fatal("preview target/body escaped")
		}
	}
}

func TestRevenueCatNativeCausesAndStatusSurviveProtectedSinks(t *testing.T) {
	for _, test := range []struct {
		name                         string
		status                       int
		body, causeType, explanation string
	}{
		{"provider rejection", http.StatusServiceUnavailable, "PRIVATE_PROVIDER recipient@example.invalid", "*revenuecat.ProviderError", "RevenueCat response rejected"},
		{"invalid JSON", http.StatusOK, "!PRIVATE_PROVIDER", "*json.SyntaxError", "decode RevenueCat response failed"},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Private", "PRIVATE_HEADER")
				w.WriteHeader(test.status)
				_, _ = io.WriteString(w, test.body)
			}))
			defer server.Close()
			client, err := revenuecat.NewClient(revenuecat.ClientConfig{BaseURL: server.URL, APIKey: "PRIVATE_CREDENTIAL", ProjectID: "PRIVATE_PROJECT"}, server.Client())
			if err != nil {
				t.Fatal(err)
			}
			_, cause := client.ListCustomerSubscriptions(context.Background(), uuid.MustParse("40000000-0000-4000-8000-000000000001"))
			if cause == nil {
				t.Fatal("fixture did not fail")
			}
			var local bytes.Buffer
			transport := &sentry.MockTransport{}
			observer := New(Config{Env: "test", Logger: slog.New(NewDiagnosticHandler(slog.NewJSONHandler(&local, nil))), LogsEnabled: true, SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
			ctx := ctxkeys.WithRunID(context.Background(), "10000000-0000-4000-8000-000000000001")
			observer.ObservePrivateFailure(ctx, cause, "did:plc:initiator", "20000000-0000-4000-8000-000000000002", "revenuecat.reconciliation", "provider", "terminal", 1)
			observer.Flush(time.Second)
			serialized, err := json.Marshal(transport.Events())
			if err != nil {
				t.Fatal(err)
			}
			issues := 0
			for _, event := range transport.Events() {
				if len(event.Exception) == 0 {
					continue
				}
				issues++
				if event.Exception[0].Type != test.causeType || event.Tags["upstream_status"] != fmt.Sprint(test.status) {
					t.Fatalf("native cause or upstream status lost: %+v", event)
				}
				if event.Contexts["diagnostic"]["operation_account_did"] != "did:plc:initiator" || event.Contexts["correlation"]["run_id"] == nil {
					t.Fatal("failure context lost")
				}
			}
			if issues != 1 {
				t.Fatalf("issues=%d want1", issues)
			}
			for _, wanted := range []string{test.explanation, test.causeType, "revenuecat.reconciliation", fmt.Sprint(test.status)} {
				if !strings.Contains(local.String(), wanted) || !strings.Contains(string(serialized), wanted) {
					t.Fatalf("missing positive field %q", wanted)
				}
			}
			for _, forbidden := range []string{"PRIVATE_", "recipient@example.invalid", server.URL, "40000000-0000-4000-8000-000000000001"} {
				if strings.Contains(local.String()+string(serialized), forbidden) {
					t.Fatalf("private provider value escaped: %s", forbidden)
				}
			}
		})
	}
}

func TestProviderLogStatusDoesNotOverwriteAppViewResponseStatus(t *testing.T) {
	var local bytes.Buffer
	transport := &sentry.MockTransport{}
	observer := New(Config{Env: "test", Logger: slog.New(NewDiagnosticHandler(slog.NewJSONHandler(&local, nil))), LogsEnabled: true, SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
	cause := &atclient.APIError{StatusCode: 503, Name: "UpstreamFailure", Message: "PRIVATE_PROVIDER"}
	observer.Log(context.Background(), slog.LevelWarn, "provider request failed", EventContext{"component": "api", "operation": "post.read", "http_status": 502}, slog.Any("causes", DescribeError(cause, EventContext{})))
	observer.Flush(time.Second)
	logs := 0
	for _, event := range transport.Events() {
		if len(event.Exception) != 0 {
			t.Fatal("supporting log created an issue")
		}
		for _, log := range event.Logs {
			logs++
			if fmt.Sprint(log.Attributes["http_status"].AsInterface()) != "502" || fmt.Sprint(log.Attributes["upstream_status"].AsInterface()) != "503" {
				t.Fatalf("provider status overwrote response: %+v", log.Attributes)
			}
		}
	}
	if logs != 1 {
		t.Fatalf("logs=%d want1", logs)
	}
	serialized, err := json.Marshal(transport.Events())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(serialized)+local.String(), "PRIVATE_PROVIDER") {
		t.Fatal("provider body escaped")
	}
}

type diagnosticFCMTransport struct {
	status                      int
	platformCode, messagingCode string
}

func (transport diagnosticFCMTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if transport.status == 0 {
		return nil, errors.New("PRIVATE_TRANSPORT recipient@example.invalid")
	}
	body, _ := json.Marshal(map[string]any{"error": map[string]any{
		"status": transport.platformCode, "message": "PRIVATE_PROVIDER_MESSAGE recipient@example.invalid",
		"details": []any{map[string]any{"@type": "type.googleapis.com/google.firebase.fcm.v1.FcmError", "errorCode": transport.messagingCode}},
	}})
	return &http.Response{StatusCode: transport.status, Header: http.Header{"X-Private": {"PRIVATE_HEADER"}}, Body: io.NopCloser(bytes.NewReader(body)), Request: request}, nil
}

func TestFirebaseDiagnosticsKeepProviderCodesAcrossLocalAndSDKSinks(t *testing.T) {
	for _, test := range []struct {
		platform, messaging, want string
		status                    int
	}{
		{"UNAUTHENTICATED", "THIRD_PARTY_AUTH_ERROR", "fcm.THIRD_PARTY_AUTH_ERROR", 401},
		{"PERMISSION_DENIED", "SENDER_ID_MISMATCH", "fcm.SENDER_ID_MISMATCH", 403},
		{"INVALID_ARGUMENT", "INVALID_ARGUMENT", "fcm.INVALID_ARGUMENT", 400},
		{"NOT_FOUND", "UNREGISTERED", "fcm.UNREGISTERED", 404},
		{"UNAVAILABLE", "UNAVAILABLE", "fcm.UNAVAILABLE", 503},
		{"RESOURCE_EXHAUSTED", "QUOTA_EXCEEDED", "fcm.QUOTA_EXCEEDED", 429},
		{"PERMISSION_DENIED", "PRIVATE_UNKNOWN_CODE", "firebase.PERMISSION_DENIED", 403},
		{"", "", "firebase.UNKNOWN", 0},
	} {
		t.Run(test.want, func(t *testing.T) {
			app, err := firebase.NewApp(context.Background(), &firebase.Config{ProjectID: "PRIVATE_PROJECT"},
				option.WithoutAuthentication(), option.WithHTTPClient(&http.Client{Transport: diagnosticFCMTransport{test.status, test.platform, test.messaging}}))
			if err != nil {
				t.Fatal(err)
			}
			client, err := app.Messaging(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			_, original := client.Send(context.Background(), &messaging.Message{Topic: "PRIVATE_TOPIC"})
			if original == nil {
				t.Fatal("fixture did not produce a real SDK error")
			}
			cause := fmt.Errorf("PRIVATE_WRAPPER: %w", original)
			var local bytes.Buffer
			transport := &sentry.MockTransport{}
			logger := slog.New(NewDiagnosticHandler(slog.NewJSONHandler(&local, nil)))
			observer := New(Config{Env: "test", Logger: logger, LogsEnabled: true, SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
			ctx := ctxkeys.WithRunID(context.Background(), "10000000-0000-4000-8000-000000000001")
			observer.ObservePrivateFailure(ctx, cause, syntax.DID("did:plc:initiator"), "20000000-0000-4000-8000-000000000002", "push.send", "provider", "terminal", 1)
			observer.Flush(time.Second)
			data, err := json.Marshal(transport.Events())
			if err != nil {
				t.Fatal(err)
			}
			var issue *sentry.Event
			var foundLog bool
			for _, event := range transport.Events() {
				if len(event.Exception) > 0 {
					issue = event
				}
				for _, log := range event.Logs {
					if log.Attributes["provider_code"].AsString() == test.want {
						foundLog = true
					}
				}
			}
			wantStatus := fmt.Sprint(test.status)
			if test.status == 0 {
				wantStatus = ""
			}
			if issue == nil || issue.Tags["provider_code"] != test.want || issue.Tags["upstream_status"] != wantStatus || !foundLog {
				t.Fatalf("provider facts lost: issue=%+v log=%t", issue, foundLog)
			}
			if issue.Exception[0].Type != "*internal.FirebaseError" || issue.Contexts["correlation"]["run_id"] == nil || issue.Contexts["diagnostic"]["workflow_ref"] == nil {
				t.Fatal("original typed cause or correlation lost")
			}
			if test.status == 0 && !strings.Contains(string(data), "HTTP response unavailable") {
				t.Fatal("missing evidence that no HTTP response was available")
			}
			for _, want := range []string{test.want, "run_id", "push.send"} {
				if !strings.Contains(local.String(), want) {
					t.Fatalf("local output missing %s", want)
				}
			}
			for _, private := range []string{"PRIVATE_", "recipient@example.invalid"} {
				if strings.Contains(local.String()+string(data), private) {
					t.Fatalf("private data escaped: %s", private)
				}
			}
		})
	}
}

func TestPDSProviderFactsSurviveErrorAndTraceFiltering(t *testing.T) {
	var local bytes.Buffer
	transport := &sentry.MockTransport{}
	observer := New(Config{Env: "test", Logger: slog.New(NewDiagnosticHandler(slog.NewJSONHandler(&local, nil))), LogsEnabled: true, TracingEnabled: true, TracesSampleRate: 1, SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
	ctx, span := observer.StartSpan(context.Background(), SpanContext{Operation: "command.apply_writes", Component: "pds"})
	sentry.SpanFromContext(ctx).SetData("request_body", "PRIVATE_SPAN_BODY")
	ctx = WithCaptureMarker(ctx)
	cause := fmt.Errorf("PRIVATE_WRAPPER: %w", &atclient.APIError{StatusCode: 503, Name: "UpstreamFailure", Message: "PRIVATE_RECORD_PAYLOAD"})
	observer.observePDSWrite(ctx, PDSOperationCommandApply, PDSStagePDSRequest, cause, time.Millisecond)
	span.Finish("error")
	observer.Flush(time.Second)
	var issue, transaction *sentry.Event
	for _, event := range transport.Events() {
		if len(event.Exception) > 0 {
			issue = event
		}
		if event.Type == "transaction" {
			transaction = event
		}
	}
	if issue == nil || issue.Tags["provider_code"] != "pds.UpstreamFailure" || issue.Tags["upstream_status"] != "503" {
		t.Fatal("PDS issue lost upstream facts")
	}
	if transaction == nil {
		t.Fatal("trace missing")
	}
	traceData, ok := transaction.Contexts["trace"]["data"].(EventContext)
	if !ok || traceData["provider_code"] != "pds.UpstreamFailure" {
		t.Fatal("trace lost provider code")
	}
	data, _ := json.Marshal(transport.Events())
	if strings.Contains(local.String()+string(data), "PRIVATE_") {
		t.Fatal("PDS prose escaped")
	}
	unknown := DescribeError(&atclient.APIError{StatusCode: 502, Name: "PRIVATE_MACHINE_CODE", Message: "PRIVATE_PAYLOAD"}, nil)
	if unknown[0].ProviderCode != "" {
		t.Fatal("unknown upstream code admitted")
	}
	missing := DescribeError(fmt.Errorf("PRIVATE_WRAPPER: %w", auth.ErrPDSLatestCommitMissingCID), nil)
	if missing[1].Message != auth.ErrPDSLatestCommitMissingCID.Error() {
		t.Fatal("repository head validation reason lost")
	}
}
