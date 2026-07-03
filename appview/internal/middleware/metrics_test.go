package middleware

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/getsentry/sentry-go"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/observability"
)

func TestHTTPMetrics_InFlightGaugeIsNonZeroDuringActiveRequest(t *testing.T) {
	recorder := observability.NewInMemoryMetricRecorder()
	observer := observability.New(observability.Config{Env: "test", MetricRecorder: recorder})
	entered := make(chan struct{})
	release := make(chan struct{})

	mux := http.NewServeMux()
	mux.Handle("GET /blocked", HTTPInFlight(observer)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		<-release
		w.WriteHeader(http.StatusNoContent)
	})))
	handler := HTTPMetrics(observer)(mux)

	done := make(chan struct{})
	go func() {
		defer close(done)
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/blocked", nil))
	}()

	<-entered
	close(release)
	<-done

	var sawInFlight, sawCompleted bool
	for _, call := range recorder.Calls() {
		if call.Attributes["route_pattern"] != "/blocked" {
			continue
		}
		switch call.Name {
		case "craftsky_appview_http_requests_in_flight":
			sawInFlight = true
		case "craftsky_appview_http_requests_total":
			sawCompleted = true
		}
		if err := observability.ValidateMetricCall(call); err != nil {
			t.Fatalf("metric call failed validation: %v; call=%#v", err, call)
		}
	}
	if !sawInFlight || !sawCompleted {
		t.Fatalf("metric calls did not include in-flight/completed route pattern calls: %#v", recorder.Calls())
	}
}

func TestHTTPMetrics_CapturesNonPanic5xxResponseInSentry(t *testing.T) {
	transport := &sentry.MockTransport{}
	observer := observability.New(observability.Config{
		Env:             "test",
		SentryDSN:       "https://public@example.invalid/1",
		SentryTransport: transport,
	})

	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/internal/{did}", func(w http.ResponseWriter, r *http.Request) {
		envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "internal server error", GetRunID(r.Context()), nil)
	})
	mux.HandleFunc("GET /v1/not-found", func(w http.ResponseWriter, r *http.Request) {
		envelope.WriteError(w, http.StatusNotFound, "not_found", "not found", GetRunID(r.Context()), nil)
	})
	handler := Logging(slog.New(slog.NewTextHandler(io.Discard, nil)))(HTTPMetrics(observer)(mux))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/internal/did:plc:raw?cursor=secret", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", rec.Code, rec.Body.String())
	}

	notFoundRec := httptest.NewRecorder()
	handler.ServeHTTP(notFoundRec, httptest.NewRequest(http.MethodGet, "/v1/not-found", nil))
	if notFoundRec.Code != http.StatusNotFound {
		t.Fatalf("not found status = %d, want 404", notFoundRec.Code)
	}

	if !observer.Flush(50 * time.Millisecond) {
		t.Fatal("observer Flush returned false")
	}
	events := transport.Events()
	if len(events) != 1 {
		t.Fatalf("captured %d Sentry events, want 1", len(events))
	}
	tags := events[0].Tags
	for key, want := range map[string]string{
		"component":         "http",
		"route_pattern":     "/v1/internal/{did}",
		"http_method":       "GET",
		"http_status":       "500",
		"http_status_class": "5xx",
		"error_category":    "server",
		"error_code":        "appview.server",
		"result":            "error",
	} {
		if tags[key] != want {
			t.Fatalf("Sentry tag %q = %q, want %q; all tags=%#v", key, tags[key], want, tags)
		}
	}
	if _, ok := tags["run_id"]; ok {
		t.Fatalf("Sentry event retained high-cardinality run_id tag: %#v", tags)
	}
	for _, forbidden := range []string{"did:plc:raw", "cursor=secret", "internal server error", "not_found"} {
		if strings.Contains(events[0].Message, forbidden) {
			t.Fatalf("Sentry message contains forbidden value %q: %#v", forbidden, events[0])
		}
		for key, value := range tags {
			if strings.Contains(key, forbidden) || strings.Contains(value, forbidden) {
				t.Fatalf("Sentry tag contains forbidden value %q: %q=%q", forbidden, key, value)
			}
		}
	}
}

func TestHTTPMetrics_ExportsSentryTransactionWithRoutePattern(t *testing.T) {
	transport := &sentry.MockTransport{}
	observer := observability.New(observability.Config{
		Env:              "test",
		SentryDSN:        "https://public@example.invalid/1",
		SentryTransport:  transport,
		TracingEnabled:   true,
		TracesSampleRate: 1,
	})

	var traceID, spanID string
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/posts/{did}/{rkey}", func(w http.ResponseWriter, r *http.Request) {
		traceID, spanID = observability.TraceIDs(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})
	handler := Logging(slog.New(slog.NewTextHandler(io.Discard, nil)))(HTTPMetrics(observer)(mux))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/posts/did:plc:alice/post1?cursor=secret", nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if traceID == "" || spanID == "" {
		t.Fatalf("handler TraceIDs = (%q, %q), want populated", traceID, spanID)
	}
	if !observer.Flush(50 * time.Millisecond) {
		t.Fatal("observer Flush returned false")
	}

	events := transport.Events()
	if len(events) != 1 {
		t.Fatalf("captured %d Sentry events, want 1 transaction", len(events))
	}
	event := events[0]
	if event.Transaction != "GET /v1/posts/{did}/{rkey}" {
		t.Fatalf("transaction = %q, want route pattern; event=%#v", event.Transaction, event)
	}
	if len(event.Spans) != 1 {
		t.Fatalf("HTTP transaction child spans = %d, want 1; event=%#v", len(event.Spans), event)
	}
	if event.Spans[0].Op != "http.handler" {
		t.Fatalf("HTTP child span op = %q, want http.handler; span=%#v", event.Spans[0].Op, event.Spans[0])
	}
	for key, want := range map[string]any{
		"component":     "http",
		"operation":     "http.handler",
		"route_pattern": "/v1/posts/{did}/{rkey}",
		"result":        "success",
	} {
		if got := event.Spans[0].Data[key]; got != want {
			t.Fatalf("HTTP child span data %q = %#v, want %#v; data=%#v", key, got, want, event.Spans[0].Data)
		}
	}
	for _, forbidden := range []string{"did:plc:alice", "post1", "cursor=secret"} {
		if strings.Contains(event.Transaction, forbidden) {
			t.Fatalf("transaction contains forbidden value %q: %#v", forbidden, event)
		}
		for _, span := range event.Spans {
			if strings.Contains(span.Op, forbidden) || strings.Contains(span.Description, forbidden) {
				t.Fatalf("span contains forbidden value %q: %#v", forbidden, span)
			}
		}
	}
}

func TestHTTPMetrics_DoesNotCaptureExpectedPDSFailureReturnedAs502(t *testing.T) {
	cases := []struct {
		name string
		err  error
	}{
		{name: "validation", err: &atclient.APIError{StatusCode: http.StatusBadRequest}},
		{name: "auth", err: &atclient.APIError{StatusCode: http.StatusUnauthorized}},
		{name: "forbidden", err: &atclient.APIError{StatusCode: http.StatusForbidden}},
		{name: "not found", err: auth.ErrRecordNotFound},
		{name: "rate limited", err: &atclient.APIError{StatusCode: http.StatusTooManyRequests}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			transport := &sentry.MockTransport{}
			observer := observability.New(observability.Config{
				Env:             "test",
				SentryDSN:       "https://public@example.invalid/1",
				SentryTransport: transport,
			})
			wrappedFactory := observer.WrapPDSFactory(func(context.Context, syntax.DID, string) (auth.PDSClient, error) {
				return fakeMetricsPDSClient{putErr: tc.err}, nil
			})

			mux := http.NewServeMux()
			mux.HandleFunc("POST /v1/posts", func(w http.ResponseWriter, r *http.Request) {
				client, err := wrappedFactory(r.Context(), syntax.DID("did:plc:alice"), "session-secret")
				if err == nil {
					err = client.PutRecord(
						r.Context(),
						syntax.DID("did:plc:alice"),
						"social.craftsky.feed.post",
						"post1",
						map[string]any{"text": "secret body"},
					)
				}
				if err != nil {
					envelope.WriteError(w, http.StatusBadGateway, "pds_write_failed", "failed to write to PDS", GetRunID(r.Context()), nil)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			})
			handler := Logging(slog.New(slog.NewTextHandler(io.Discard, nil)))(HTTPMetrics(observer)(mux))

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/posts", strings.NewReader(`{"text":"secret body"}`)))
			if rec.Code != http.StatusBadGateway {
				t.Fatalf("status = %d, want 502; body=%s", rec.Code, rec.Body.String())
			}

			if !observer.Flush(50 * time.Millisecond) {
				t.Fatal("observer Flush returned false")
			}
			if events := transport.Events(); len(events) != 0 {
				t.Fatalf("captured %d Sentry events, want 0; first=%#v", len(events), events[0])
			}
		})
	}
}

type fakeMetricsPDSClient struct {
	putErr error
}

func (f fakeMetricsPDSClient) GetRecord(context.Context, syntax.DID, string, string, any) (string, error) {
	return "", nil
}

func (f fakeMetricsPDSClient) PutRecord(context.Context, syntax.DID, string, string, any) error {
	return f.putErr
}

func (f fakeMetricsPDSClient) CreateRecord(context.Context, syntax.DID, string, any) (syntax.ATURI, syntax.CID, error) {
	return "", "", nil
}

func (f fakeMetricsPDSClient) DeleteRecord(context.Context, syntax.DID, string, string) error {
	return nil
}

func (f fakeMetricsPDSClient) UploadBlob(context.Context, string, []byte) (*auth.UploadedBlob, error) {
	return &auth.UploadedBlob{}, nil
}
