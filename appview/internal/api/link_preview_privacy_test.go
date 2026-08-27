package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"

	"social.craftsky/appview/internal/linkpreview"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/observability"
)

func TestIT018LinkPreviewDiagnosticsExcludeSensitiveInputs(t *testing.T) {
	t.Parallel()
	const canary = "private.example/path?token=secret"
	observer := &recordingLinkPreviewObserver{}
	handler := LinkPreviewHandler(
		linkPreviewErrorService{err: errors.New(canary)},
		true,
		observer,
	)
	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/link-previews",
		strings.NewReader(`{"url":"https://`+canary+`"}`),
	)

	handler.ServeHTTP(httptest.NewRecorder(), request)

	if len(observer.events) != 1 {
		t.Fatalf("events = %d, want 1", len(observer.events))
	}
	event := observer.events[0]
	if strings.Contains(strings.Join([]string{event.stage, event.result, event.errorClass}, ":"), canary) {
		t.Fatalf("telemetry leaked canary: %+v", event)
	}
	if event.stage != "fetch" || event.result != "failed" || event.errorClass != "upstream" || event.status != http.StatusBadGateway {
		t.Fatalf("event = %+v", event)
	}
}

func TestAT009LinkPreviewSuccessTelemetryExcludesResponseContent(t *testing.T) {
	t.Parallel()
	const canary = "private-content-canary"
	observer := &recordingLinkPreviewObserver{}
	finalURL, err := url.Parse("https://final.example/" + canary)
	if err != nil {
		t.Fatal(err)
	}
	handler := LinkPreviewHandler(linkPreviewStaticService{preview: linkpreview.Preview{
		URL: finalURL, Title: canary, Description: canary,
		Thumbnail: &linkpreview.Thumbnail{Bytes: []byte(canary), MIMEType: "image/png", Width: 1, Height: 1},
	}}, true, observer)
	request := httptest.NewRequest(http.MethodPost, "/v1/link-previews", strings.NewReader(`{"url":"https://source.example/path"}`))

	handler.ServeHTTP(httptest.NewRecorder(), request)

	if len(observer.events) != 1 {
		t.Fatalf("events = %d, want 1", len(observer.events))
	}
	event := observer.events[0]
	if event.stage != "complete" || event.result != "success" || event.errorClass != "none" || event.bytes != len(canary) {
		t.Fatalf("event = %+v", event)
	}
	if strings.Contains(strings.Join([]string{event.stage, event.result, event.errorClass}, ":"), canary) {
		t.Fatalf("success telemetry leaked response content: %+v", event)
	}
}

func TestIR024RealLinkPreviewLifecycleExcludesCanariesFromApplicableSinks(t *testing.T) {
	t.Parallel()
	canaries := []string{
		"source-lifecycle-canary.example/private?token=secret",
		"final-lifecycle-canary.example/redirect",
		"title-lifecycle-canary",
		"description-lifecycle-canary",
		"thumbnail-lifecycle-canary",
		"dependency-lifecycle-canary",
		"bearer-lifecycle-canary",
		"device-lifecycle-canary",
		"parse-lifecycle-canary",
		"image-lifecycle-canary.example",
	}

	collector := newLifecycleTelemetryCollector()
	service, lifecycle := newScriptedLifecyclePreviewService(t, canaries)
	mux := http.NewServeMux()
	mux.Handle("POST /v1/link-previews", LinkPreviewHandler(service, true, collector.observer))
	handler := middleware.Logging(collector.logger)(middleware.HTTPMetrics(collector.observer)(mux))
	for _, test := range []struct {
		target string
		status int
	}{
		{target: "http://" + canaries[0], status: http.StatusOK},
		{target: "http://parse-lifecycle-canary.example/failure", status: http.StatusUnprocessableEntity},
		{target: "http://dns-lifecycle-canary.example/failure", status: http.StatusBadGateway},
		{target: "http://dial-lifecycle-canary.example/failure", status: http.StatusBadGateway},
	} {
		request := httptest.NewRequest(
			http.MethodPost,
			"/v1/link-previews",
			strings.NewReader(`{"url":"`+test.target+`"}`),
		)
		request.Header.Set("Authorization", "Bearer "+canaries[6])
		request.Header.Set("X-Craftsky-Device-Id", canaries[7])
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		if recorder.Code != test.status {
			t.Fatalf("target %q status = %d, want %d; body = %s", test.target, recorder.Code, test.status, recorder.Body.String())
		}
	}
	lifecycle.assertExercised(t)
	if !collector.observer.Flush(time.Second) {
		t.Fatal("observer flush failed")
	}

	captured := collector.String(t)
	for _, canary := range canaries {
		if strings.Contains(captured, canary) {
			t.Fatalf("link-preview lifecycle telemetry leaked %q:\n%s", canary, captured)
		}
	}
	if collector.localLogs.Len() == 0 || len(collector.metrics.Calls()) == 0 || len(collector.transport.Events()) == 0 {
		t.Fatalf("expected local logs, metrics, and Sentry trace/error events; captured=%s", captured)
	}
	collector.assertSentryErrorAndTransaction(t)
	if len(collector.externalLogs.events) != 0 {
		t.Fatalf("link-preview lifecycle unexpectedly emitted external logs: %s", collector.externalLogs.String())
	}
}

type scriptedLifecycle struct {
	mu       sync.Mutex
	resolved map[string]int
	requests []string
	canaries []string
	png      []byte
}

func newScriptedLifecyclePreviewService(t *testing.T, canaries []string) (*linkpreview.Service, *scriptedLifecycle) {
	t.Helper()
	var imageBytes bytes.Buffer
	if err := png.Encode(&imageBytes, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatalf("encode lifecycle thumbnail: %v", err)
	}
	lifecycle := &scriptedLifecycle{
		resolved: map[string]int{}, canaries: canaries,
		png: append(imageBytes.Bytes(), []byte(canaries[4])...),
	}
	transport := linkpreview.NewPinnedTransport(lifecycle, lifecycle)
	t.Cleanup(transport.CloseIdleConnections)
	client := &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	return linkpreview.NewService(linkpreview.NewFetcher(lifecycle, client)), lifecycle
}

func (lifecycle *scriptedLifecycle) LookupNetIP(_ context.Context, _, host string) ([]netip.Addr, error) {
	lifecycle.mu.Lock()
	lifecycle.resolved[host]++
	lifecycle.mu.Unlock()
	if host == "dns-lifecycle-canary.example" {
		return nil, errors.New(lifecycle.canaries[5])
	}
	if host == "dial-lifecycle-canary.example" {
		return []netip.Addr{netip.MustParseAddr("8.8.4.4")}, nil
	}
	return []netip.Addr{netip.MustParseAddr("8.8.8.8")}, nil
}

func (lifecycle *scriptedLifecycle) DialContext(_ context.Context, _, address string) (net.Conn, error) {
	if address == "8.8.4.4:80" {
		return nil, errors.New(lifecycle.canaries[5])
	}
	client, server := net.Pipe()
	go lifecycle.serve(server)
	return client, nil
}

func (lifecycle *scriptedLifecycle) serve(connection net.Conn) {
	defer connection.Close()
	request, err := http.ReadRequest(bufio.NewReader(connection))
	if err != nil {
		return
	}
	lifecycle.mu.Lock()
	lifecycle.requests = append(lifecycle.requests, request.Host+request.URL.RequestURI())
	lifecycle.mu.Unlock()

	var status, contentType string
	var body []byte
	var extraHeader string
	switch request.Host {
	case "source-lifecycle-canary.example":
		status = "302 Found"
		extraHeader = "Location: http://" + lifecycle.canaries[1] + "\r\n"
	case "final-lifecycle-canary.example":
		status, contentType = "200 OK", "text/html; charset=utf-8"
		body = []byte(`<head><title>` + lifecycle.canaries[2] + `</title><meta name="description" content="` + lifecycle.canaries[3] + `"><meta property="og:image" content="http://` + lifecycle.canaries[9] + `/bad"><meta property="og:image" content="http://` + lifecycle.canaries[9] + `/good"></head>`)
	case "image-lifecycle-canary.example":
		status, contentType = "200 OK", "image/png"
		if request.URL.Path == "/bad" {
			body = []byte(lifecycle.canaries[4] + "-corrupt")
		} else {
			body = lifecycle.png
		}
	case "parse-lifecycle-canary.example":
		status, contentType = "200 OK", "text/html; charset=x-"+lifecycle.canaries[8]
		body = []byte(`<head><title>` + lifecycle.canaries[8] + `</title></head>`)
	default:
		status = "500 Internal Server Error"
	}
	headers := extraHeader
	if contentType != "" {
		headers += "Content-Type: " + contentType + "\r\n"
	}
	_, _ = fmt.Fprintf(connection, "HTTP/1.1 %s\r\n%sContent-Length: %d\r\nConnection: close\r\n\r\n", status, headers, len(body))
	_, _ = connection.Write(body)
}

func (lifecycle *scriptedLifecycle) assertExercised(t *testing.T) {
	t.Helper()
	lifecycle.mu.Lock()
	defer lifecycle.mu.Unlock()
	for _, host := range []string{
		"source-lifecycle-canary.example", "final-lifecycle-canary.example",
		"image-lifecycle-canary.example", "parse-lifecycle-canary.example",
		"dns-lifecycle-canary.example", "dial-lifecycle-canary.example",
	} {
		if lifecycle.resolved[host] == 0 {
			t.Errorf("resolver did not exercise %s", host)
		}
	}
	requests := strings.Join(lifecycle.requests, "\n")
	for _, request := range []string{
		"source-lifecycle-canary.example/private?token=secret",
		"final-lifecycle-canary.example/redirect",
		"image-lifecycle-canary.example/bad",
		"image-lifecycle-canary.example/good",
		"parse-lifecycle-canary.example/failure",
	} {
		if !strings.Contains(requests, request) {
			t.Errorf("scripted lifecycle did not request %q; requests:\n%s", request, requests)
		}
	}
}

type linkPreviewErrorService struct{ err error }

func (s linkPreviewErrorService) FetchPreview(context.Context, string) (linkpreview.Preview, error) {
	return linkpreview.Preview{}, s.err
}

type linkPreviewStaticService struct{ preview linkpreview.Preview }

func (s linkPreviewStaticService) FetchPreview(context.Context, string) (linkpreview.Preview, error) {
	return s.preview, nil
}

type recordedLinkPreviewEvent struct {
	stage, result, errorClass string
	status, redirects, bytes  int
	duration                  time.Duration
}

type recordingLinkPreviewObserver struct{ events []recordedLinkPreviewEvent }

func (o *recordingLinkPreviewObserver) ObserveLinkPreview(
	stage, result, errorClass string,
	status, redirects, bytes int,
	duration time.Duration,
) {
	o.events = append(o.events, recordedLinkPreviewEvent{
		stage: stage, result: result, errorClass: errorClass,
		status: status, redirects: redirects, bytes: bytes, duration: duration,
	})
}

type lifecycleTelemetryCollector struct {
	localLogs    bytes.Buffer
	logger       *slog.Logger
	metrics      *observability.InMemoryMetricRecorder
	externalLogs *recordingLifecycleLogSink
	transport    *sentry.MockTransport
	observer     *observability.Observer
}

func newLifecycleTelemetryCollector() *lifecycleTelemetryCollector {
	collector := &lifecycleTelemetryCollector{
		metrics:      observability.NewInMemoryMetricRecorder(),
		externalLogs: &recordingLifecycleLogSink{},
		transport:    &sentry.MockTransport{},
	}
	collector.logger = slog.New(slog.NewJSONHandler(&collector.localLogs, nil))
	collector.observer = observability.New(observability.Config{
		Env: "test", SentryDSN: "https://public@example.invalid/1",
		SentryTransport: collector.transport, TracingEnabled: true, TracesSampleRate: 1,
		LogsEnabled: true, MetricRecorder: collector.metrics, LogSink: collector.externalLogs,
		Logger: collector.logger,
	})
	return collector
}

func (collector *lifecycleTelemetryCollector) String(t *testing.T) string {
	t.Helper()
	events, err := json.Marshal(collector.transport.Events())
	if err != nil {
		t.Fatal(err)
	}
	metrics, err := json.Marshal(collector.metrics.Calls())
	if err != nil {
		t.Fatal(err)
	}
	return strings.Join([]string{
		collector.localLogs.String(), collector.externalLogs.String(), string(metrics), string(events),
	}, "\n")
}

func (collector *lifecycleTelemetryCollector) assertSentryErrorAndTransaction(t *testing.T) {
	t.Helper()
	var sawError, sawTransaction bool
	for _, event := range collector.transport.Events() {
		sawError = sawError || event.Level == sentry.LevelError
		sawTransaction = sawTransaction || event.Type == "transaction"
	}
	if !sawError || !sawTransaction {
		t.Fatalf("Sentry events missing error or transaction: %#v", collector.transport.Events())
	}
}

type recordingLifecycleLogSink struct {
	events []observability.EventContext
}

func (sink *recordingLifecycleLogSink) Emit(_ context.Context, _ slog.Level, _ string, attrs observability.EventContext) {
	sink.events = append(sink.events, attrs)
}

func (sink *recordingLifecycleLogSink) String() string {
	body, _ := json.Marshal(sink.events)
	return string(body)
}
