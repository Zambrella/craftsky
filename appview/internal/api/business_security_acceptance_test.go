package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/getsentry/sentry-go"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/ctxkeys"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/pdseffects"
)

func TestBusinessSecurityAcceptance(t *testing.T) {
	canaries := []string{
		"https://authored-uri-canary.example:65535/item?variant=one#buy",
		"mailto:Email.Canary+orders@EXAMPLE.COM",
		"authored-free-text-canary",
		"authored-title-canary",
		"123456.78",
		"authored-location-canary",
	}
	var logs bytes.Buffer
	metrics := observability.NewInMemoryMetricRecorder()
	transport := &sentry.MockTransport{}
	observer := observability.New(observability.Config{
		Env: "test", SentryDSN: "https://public@example.invalid/1",
		SentryTransport: transport, TracingEnabled: true, TracesSampleRate: 1,
		MetricRecorder: metrics,
	})
	logger := slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))

	validProfileBodies := []string{
		businessSecurityProfileBody(t, canaries, "email", canaries[1], canaries[0]),
		businessSecurityProfileBody(t, canaries, "visit-website", canaries[0], canaries[0]),
	}
	for _, body := range validProfileBodies {
		effects := &businessProfileEffects{}
		handler := api.PutBusinessProfileHandler(func(context.Context, syntax.DID, string) (pdseffects.EffectExecutor, error) {
			return effects, nil
		})
		response := serveBusinessSecurityRequest(t, logger, observer, handler, http.MethodPut,
			"/v1/profiles/me/business", body, "*", "", "")
		if response.Code != http.StatusOK || len(effects.puts) != 1 {
			t.Fatalf("valid profile destination status/effects = %d/%d; body=%s", response.Code, len(effects.puts), response.Body.String())
		}
	}

	webRejected := []string{
		"http://example.com",
		"custom://example.com",
		"https://user:password@example.com",
		"https:///hostless",
		"https://localhost",
		"https://example.com.",
		"https://127.0.0.1",
		"https://bücher.example",
		"https://%65xample.com",
		"https://example.com:65536",
		"https://example.com/" + strings.Repeat("x", 2049),
	}
	for _, destination := range webRejected {
		effects := &businessProfileEffects{}
		handler := api.PutBusinessProfileHandler(func(context.Context, syntax.DID, string) (pdseffects.EffectExecutor, error) {
			return effects, nil
		})
		body := businessSecurityProfileBody(t, canaries, "visit-website", destination, canaries[0])
		response := serveBusinessSecurityRequest(t, logger, observer, handler, http.MethodPut,
			"/v1/profiles/me/business", body, "*", "", "")
		if response.Code != http.StatusUnprocessableEntity || len(effects.reads) != 0 || len(effects.puts) != 0 {
			t.Errorf("rejected web destination %q status/effects = %d/%d/%d", destination, response.Code, len(effects.reads), len(effects.puts))
		}
	}

	mailRejected := []string{
		"MAILTO:person@example.com",
		"mailto:person name@example.com",
		"mailto:person%2Bshop@example.com",
		"mailto:person@example.com?subject=private",
		"mailto:person@example.com#private",
		"mailto:one@example.com,two@example.com",
		"mailto:one@example.com;two@example.com",
		"mailto:person\n@example.com",
		"mailto:" + strings.Repeat("a", 309) + "@example.com",
	}
	for _, destination := range mailRejected {
		effects := &businessProfileEffects{}
		handler := api.PutBusinessProfileHandler(func(context.Context, syntax.DID, string) (pdseffects.EffectExecutor, error) {
			return effects, nil
		})
		body := businessSecurityProfileBody(t, canaries, "email", destination, canaries[0])
		response := serveBusinessSecurityRequest(t, logger, observer, handler, http.MethodPut,
			"/v1/profiles/me/business", body, "*", "", "")
		if response.Code != http.StatusUnprocessableEntity || len(effects.reads) != 0 || len(effects.puts) != 0 {
			t.Errorf("rejected mail destination %q status/effects = %d/%d/%d", destination, response.Code, len(effects.reads), len(effects.puts))
		}
	}
	productEffects := &businessProfileEffects{}
	productHandler := api.PutBusinessProfileHandler(func(context.Context, syntax.DID, string) (pdseffects.EffectExecutor, error) {
		return productEffects, nil
	})
	badProduct := businessSecurityProfileBody(t, canaries, "visit-website", canaries[0], "http://product.example/item")
	productResponse := serveBusinessSecurityRequest(t, logger, observer, productHandler, http.MethodPut,
		"/v1/profiles/me/business", badProduct, "*", "", "")
	if productResponse.Code != http.StatusUnprocessableEntity || len(productEffects.puts) != 0 {
		t.Errorf("rejected product destination status/puts = %d/%d", productResponse.Code, len(productEffects.puts))
	}

	eventEffects := newBusinessEventEffects()
	eventHandler := api.PostBusinessEventHandler(func(context.Context, syntax.DID, string) (pdseffects.EffectExecutor, error) {
		return eventEffects, nil
	}, func() time.Time { return businessEventNow })
	validEvent := strings.NewReplacer(
		"Fiber Fair", canaries[3],
		"Meet our makers", canaries[2],
		"Guild Hall", canaries[5],
		"https://event.example/details", canaries[0],
	).Replace(validBusinessEventBody(false))
	eventResponse := serveBusinessSecurityRequest(t, logger, observer, eventHandler, http.MethodPost,
		"/v1/events", validEvent, "", "", "")
	if eventResponse.Code != http.StatusCreated || len(eventEffects.puts) != 1 {
		t.Fatalf("valid event destination status/effects = %d/%d; body=%s", eventResponse.Code, len(eventEffects.puts), eventResponse.Body.String())
	}
	for _, destination := range []string{"http://event.example", "custom://event.example", "https://user:event@example.com", "https://localhost"} {
		effects := newBusinessEventEffects()
		handler := api.PostBusinessEventHandler(func(context.Context, syntax.DID, string) (pdseffects.EffectExecutor, error) {
			return effects, nil
		}, func() time.Time { return businessEventNow })
		body := strings.Replace(validBusinessEventBody(false), "https://event.example/details", destination, 1)
		response := serveBusinessSecurityRequest(t, logger, observer, handler, http.MethodPost,
			"/v1/events", body, "", "", "")
		if response.Code != http.StatusUnprocessableEntity || len(effects.puts) != 0 {
			t.Errorf("rejected event destination %q status/puts = %d/%d", destination, response.Code, len(effects.puts))
		}
	}
	registrationEffects := newBusinessEventEffects()
	registrationHandler := api.PostBusinessEventHandler(func(context.Context, syntax.DID, string) (pdseffects.EffectExecutor, error) {
		return registrationEffects, nil
	}, func() time.Time { return businessEventNow })
	badRegistration := strings.Replace(validBusinessEventBody(false), "https://tickets.example/register", "http://tickets.example/register", 1)
	registrationResponse := serveBusinessSecurityRequest(t, logger, observer, registrationHandler, http.MethodPost,
		"/v1/events", badRegistration, "", "", "")
	if registrationResponse.Code != http.StatusUnprocessableEntity || len(registrationEffects.puts) != 0 {
		t.Errorf("rejected registration destination status/puts = %d/%d", registrationResponse.Code, len(registrationEffects.puts))
	}

	if !observer.Flush(time.Second) {
		t.Fatal("observer flush failed")
	}
	calls := metrics.Calls()
	if len(calls) == 0 {
		t.Fatal("security acceptance emitted no HTTP metrics")
	}
	for _, call := range calls {
		if err := observability.ValidateMetricCall(call); err != nil {
			t.Fatalf("metric call failed validation: %v; call=%#v", err, call)
		}
	}
	telemetry, err := json.Marshal(struct {
		Logs    string
		Metrics []observability.MetricCall
		Traces  []*sentry.Event
	}{logs.String(), calls, transport.Events()})
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range append(canaries,
		"did:plc:owner",
		"at://did:plc:owner/social.craftsky.business.profile/self",
		"at://did:plc:owner/social.craftsky.business.event/",
	) {
		if strings.Contains(string(telemetry), forbidden) {
			t.Fatalf("security telemetry contains prohibited value %q: %s", forbidden, telemetry)
		}
	}
}

func businessSecurityProfileBody(t *testing.T, canaries []string, actionType, destination, productURI string) string {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"tagline": canaries[2], "hoursNote": canaries[2], "serviceArea": canaries[2],
		"location":      map[string]any{"country": "gb", "locality": canaries[5]},
		"primaryAction": map[string]any{"type": actionType, "destination": destination},
		"products": []map[string]any{{
			"title": canaries[3], "uri": productURI,
			"image": map[string]any{
				"image": map[string]any{
					"$type": "blob", "ref": map[string]any{"$link": "bafkreie3w2xq7u6rs5szu6vllsq5xh7y7uv3f6blql6uz4ep6txv6m4o6a"},
					"mimeType": "image/webp", "size": 1024,
				},
				"alt": canaries[2],
			},
			"price": map[string]any{"amount": canaries[4], "currency": "USD"},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func serveBusinessSecurityRequest(
	t *testing.T,
	logger *slog.Logger,
	observer *observability.Observer,
	handler http.Handler,
	method, target, body, ifMatch, did, rkey string,
) *httptest.ResponseRecorder {
	t.Helper()
	wrapped := middleware.Logging(logger)(middleware.HTTPMetrics(observer)(handler))
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	if ifMatch != "" {
		request.Header.Set("If-Match", ifMatch)
	}
	request.SetPathValue("did", did)
	request.SetPathValue("rkey", rkey)
	ctx := middleware.WithDID(request.Context(), "did:plc:owner")
	ctx = middleware.WithOwnerGeneration(ctx, 7)
	ctx = middleware.WithOAuthSessionID(ctx, "oauth-owner-session")
	ctx = ctxkeys.WithRunID(ctx, "security-request")
	request = request.WithContext(ctx)
	response := httptest.NewRecorder()
	wrapped.ServeHTTP(response, request)
	return response
}
