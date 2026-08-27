package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"social.craftsky/appview/internal/linkpreview"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/testdb"
)

// IT-005: the preview route is always registered and auth, device, method,
// body, and disabled-feature rejection all occur before preview service work.
func TestAddRoutesLinkPreviewAdmission(t *testing.T) {
	pool := testdb.WithSchema(t, `
		CREATE TABLE craftsky_profiles (did TEXT PRIMARY KEY);
		INSERT INTO craftsky_profiles (did) VALUES ('did:plc:test');
	`)
	finalURL, _ := url.Parse("https://final.example/pattern")
	service := &routeLinkPreviewService{preview: linkpreview.Preview{URL: finalURL, Title: "Pattern"}}
	newMux := func(enabled bool) (*http.ServeMux, http.Handler) {
		deps := testDeps()
		deps.DB = pool
		deps.Config.LinkPreviewsEnabled = enabled
		deps.LinkPreviews = service
		mux := http.NewServeMux()
		AddRoutes(context.Background(), mux, deps)
		catalogue, err := NewV1Catalogue(V1RoutePolicies(deps.Config.Env, deps.Config))
		if err != nil {
			t.Fatalf("build route catalogue: %v", err)
		}
		return mux, catalogue.RoutingHandler(mux)
	}

	disabledMux, _ := newMux(false)
	probe := httptest.NewRequest(http.MethodPost, "/v1/link-previews", nil)
	if _, pattern := disabledMux.Handler(probe); pattern != "POST /v1/link-previews" {
		t.Fatalf("registered pattern = %q, want POST /v1/link-previews", pattern)
	}

	for _, tt := range []struct {
		name    string
		method  string
		body    string
		auth    bool
		device  bool
		enabled bool
		want    int
	}{
		{name: "missing auth", method: http.MethodPost, body: `{"url":"https://source.example"}`, device: true, enabled: true, want: http.StatusUnauthorized},
		{name: "missing device", method: http.MethodPost, body: `{"url":"https://source.example"}`, auth: true, enabled: true, want: http.StatusBadRequest},
		{name: "disabled", method: http.MethodPost, body: `{"url":"https://source.example"}`, auth: true, device: true, want: http.StatusServiceUnavailable},
		{name: "wrong method", method: http.MethodGet, auth: true, device: true, enabled: true, want: http.StatusMethodNotAllowed},
		{name: "malformed body", method: http.MethodPost, body: `{}`, auth: true, device: true, enabled: true, want: http.StatusBadRequest},
		{name: "admitted", method: http.MethodPost, body: `{"url":"https://source.example"}`, auth: true, device: true, enabled: true, want: http.StatusOK},
	} {
		t.Run(tt.name, func(t *testing.T) {
			before := service.calls
			request := httptest.NewRequest(tt.method, "/v1/link-previews", strings.NewReader(tt.body))
			if tt.auth {
				request.Header.Set("Authorization", "Bearer anything")
			}
			if tt.device {
				request.Header.Set("X-Craftsky-Device-Id", "device-test")
			}
			recorder := httptest.NewRecorder()
			_, handler := newMux(tt.enabled)
			handler.ServeHTTP(recorder, request)
			if recorder.Code != tt.want {
				t.Fatalf("status = %d, want %d; body=%s", recorder.Code, tt.want, recorder.Body.String())
			}
			wantCalls := before
			if tt.name == "admitted" {
				wantCalls++
			}
			if service.calls != wantCalls {
				t.Fatalf("service calls = %d, want %d", service.calls, wantCalls)
			}
			if recorder.Code >= 400 && recorder.Code != http.StatusMethodNotAllowed {
				assertErrorEnvelope(t, recorder.Body.Bytes())
			}
		})
	}
}

// AT-007: the registered endpoint composes authenticated admission, outbound
// URL policy, feature enablement, and dedicated limits before preview work.
func TestLinkPreviewEndpointSecurityStack(t *testing.T) {
	pool := testdb.WithSchema(t, `
		CREATE TABLE craftsky_profiles (did TEXT PRIMARY KEY);
		INSERT INTO craftsky_profiles (did) VALUES ('did:plc:test');
	`)
	finalURL, _ := url.Parse("https://final.example/pattern")
	newStack := func(enabled bool, perToken int) (http.Handler, *routeLinkPreviewService) {
		service := &routeLinkPreviewService{preview: linkpreview.Preview{URL: finalURL, Title: "Pattern"}}
		deps := testDeps()
		deps.DB = pool
		deps.Config.LinkPreviewsEnabled = enabled
		deps.LinkPreviews = service
		deps.RateLimiter = middleware.NewLocalRateLimiter(middleware.RateLimitConfig{Classes: map[middleware.RateClass]middleware.ClassLimit{
			middleware.RateClassLinkPreview: {Window: time.Hour, PerToken: perToken, PerDevice: perToken},
		}}, func() time.Time { return time.Unix(1_000, 0) })
		mux := http.NewServeMux()
		AddRoutes(context.Background(), mux, deps)
		catalogue, err := NewV1Catalogue(V1RoutePolicies(deps.Config.Env, deps.Config))
		if err != nil {
			t.Fatalf("build route catalogue: %v", err)
		}
		return catalogue.RoutingHandler(mux), service
	}
	serve := func(handler http.Handler, raw string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodPost, "/v1/link-previews", strings.NewReader(`{"url":"`+raw+`"}`))
		request.Header.Set("Authorization", "Bearer anything")
		request.Header.Set("X-Craftsky-Device-Id", "device-test")
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		return recorder
	}

	disabledMux, disabledService := newStack(false, 60)
	if recorder := serve(disabledMux, "https://source.example"); recorder.Code != http.StatusServiceUnavailable || disabledService.calls != 0 {
		t.Fatalf("disabled status/calls = %d/%d", recorder.Code, disabledService.calls)
	}
	forbiddenMux, forbiddenService := newStack(true, 60)
	if recorder := serve(forbiddenMux, "http://127.0.0.1/private"); recorder.Code != http.StatusUnprocessableEntity || forbiddenService.calls != 0 {
		t.Fatalf("forbidden status/calls = %d/%d", recorder.Code, forbiddenService.calls)
	}
	rateMux, rateService := newStack(true, 1)
	if recorder := serve(rateMux, "https://source.example/one"); recorder.Code != http.StatusOK {
		t.Fatalf("first status = %d; body=%s", recorder.Code, recorder.Body.String())
	}
	if recorder := serve(rateMux, "https://source.example/two"); recorder.Code != http.StatusTooManyRequests || rateService.calls != 1 {
		t.Fatalf("limited status/calls = %d/%d; body=%s", recorder.Code, rateService.calls, recorder.Body.String())
	}
}

func TestIT018LinkPreviewAdmissionTelemetryExcludesRequestCanaries(t *testing.T) {
	pool := testdb.WithSchema(t, `
		CREATE TABLE craftsky_profiles (did TEXT PRIMARY KEY);
		INSERT INTO craftsky_profiles (did) VALUES ('did:plc:test');
	`)
	canaries := []string{
		"url-canary.invalid/private", "query-canary=secret", "redirect-canary.invalid/final",
		"title-canary", "description-canary", "thumbnail-byte-canary",
		"post-text-canary", "bearer-token-canary", "device-id-canary",
	}
	var logs bytes.Buffer
	metrics := observability.NewInMemoryMetricRecorder()
	observer := observability.New(observability.Config{
		Env:            "test",
		MetricRecorder: metrics,
	})
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	finalURL, _ := url.Parse("https://" + canaries[2])
	service := &routeLinkPreviewService{preview: linkpreview.Preview{
		URL: finalURL, Title: canaries[3], Description: canaries[4],
		Thumbnail: &linkpreview.Thumbnail{Bytes: []byte(canaries[5]), MIMEType: "image/png", Width: 1, Height: 1},
	}}
	build := func(enabled bool, limit int) http.Handler {
		deps := testDeps()
		deps.DB = pool
		deps.Logger = logger
		deps.Observability = observer
		deps.Config.LinkPreviewsEnabled = enabled
		deps.LinkPreviews = service
		deps.RateLimiter = middleware.NewLocalRateLimiter(middleware.RateLimitConfig{Classes: map[middleware.RateClass]middleware.ClassLimit{
			middleware.RateClassLinkPreview: {Window: time.Hour, PerToken: limit, PerDevice: limit},
		}}, func() time.Time { return time.Unix(1_000, 0) })
		mux := http.NewServeMux()
		AddRoutes(context.Background(), mux, deps)
		catalogue, err := NewV1Catalogue(V1RoutePolicies(deps.Config.Env, deps.Config))
		if err != nil {
			t.Fatal(err)
		}
		return catalogue.RoutingHandler(mux)
	}
	serve := func(handler http.Handler, auth, device bool) {
		body := `{"url":"https://` + canaries[0] + `?` + canaries[1] + `"}`
		request := httptest.NewRequest(http.MethodPost, "/v1/link-previews", strings.NewReader(body))
		request.Header.Set("X-Post-Canary", canaries[6])
		if auth {
			request.Header.Set("Authorization", "Bearer "+canaries[7])
		}
		if device {
			request.Header.Set("X-Craftsky-Device-Id", canaries[8])
		}
		handler.ServeHTTP(httptest.NewRecorder(), request)
	}
	serve(build(true, 10), false, true)
	serve(build(true, 10), true, false)
	serve(build(false, 10), true, true)
	rateHandler := build(true, 1)
	serve(rateHandler, true, true)
	serve(rateHandler, true, true)
	metricJSON, _ := json.Marshal(metrics.Calls())
	captured := logs.String() + string(metricJSON)
	for _, canary := range canaries {
		if strings.Contains(captured, canary) {
			t.Fatalf("admission telemetry leaked %q:\n%s", canary, captured)
		}
	}
}

type routeLinkPreviewService struct {
	preview linkpreview.Preview
	calls   int
}

func (s *routeLinkPreviewService) FetchPreview(context.Context, string) (linkpreview.Preview, error) {
	s.calls++
	return s.preview, nil
}
