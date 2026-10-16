package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/getsentry/sentry-go"
	"google.golang.org/api/option"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ctxkeys"
)

type diagnosticFCMTransport struct {
	status                      int
	platformCode, messagingCode string
}

func (transport diagnosticFCMTransport) RoundTrip(request *http.Request) (*http.Response, error) {
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
			if issue == nil || issue.Tags["provider_code"] != test.want || issue.Tags["upstream_status"] != fmt.Sprint(test.status) || !foundLog {
				t.Fatalf("provider facts lost: issue=%+v log=%t", issue, foundLog)
			}
			if issue.Exception[0].Type != "*internal.FirebaseError" || issue.Contexts["correlation"]["run_id"] == nil || issue.Contexts["diagnostic"]["workflow_ref"] == nil {
				t.Fatal("original typed cause or correlation lost")
			}
			for _, want := range []string{test.want, "upstream_status", "run_id", "push.send"} {
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
