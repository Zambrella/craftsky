package instagrammeta

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestWebhookHandlerUnsafeDiagnosticsLogsRawBodyAndReduction(t *testing.T) {
	t.Parallel()

	const (
		secret    = "synthetic-app-secret"
		official  = "synthetic-official"
		bodyValue = "synthetic-sensitive-webhook-body"
	)
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	digests, err := NewDigestCodec(bytes.Repeat([]byte{0x73}, 32), func(input string) (string, error) {
		return strings.ToUpper(strings.TrimSpace(input)), nil
	})
	if err != nil {
		t.Fatalf("NewDigestCodec: %v", err)
	}
	reducer, err := NewPayloadReducer(official, digests)
	if err != nil {
		t.Fatalf("NewPayloadReducer: %v", err)
	}
	handler, err := NewWebhookHandler(WebhookHandlerConfig{
		AppSecret:       []byte(secret),
		VerifyToken:     "synthetic-verify-token",
		Reducer:         reducer,
		Sink:            &recordingWebhookSink{},
		Logger:          logger,
		UnsafeDebugLogs: true,
	})
	if err != nil {
		t.Fatalf("NewWebhookHandler: %v", err)
	}
	body := []byte(`{"object":"instagram","diagnostic":"` + bodyValue + `","entry":[]}`)
	request := httptest.NewRequest(http.MethodPost, "/integrations/instagram/webhook", bytes.NewReader(body))
	request.Header.Set("X-Hub-Signature-256", signWebhookBody([]byte(secret), body))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %q", response.Code, response.Body.String())
	}
	if !strings.Contains(logs.String(), bodyValue) {
		t.Fatalf("debug logs omitted raw webhook body: %s", logs.String())
	}
	if !strings.Contains(logs.String(), `"supported_event_count":0`) {
		t.Fatalf("debug logs omitted reduction count: %s", logs.String())
	}
}

func TestWebhookHandlerAcknowledgesOnlyAfterSignedReducedWorkIsPersisted(t *testing.T) {
	t.Parallel()

	const (
		appSecret     = "synthetic-app-secret"
		official      = "synthetic-official"
		sender        = "synthetic-sender"
		messageID     = "synthetic-message-id"
		challenge     = "CSKY-2345-6789-ABCD-E"
		rawBodyCanary = "synthetic-unrelated-raw-canary"
	)
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	sink := &recordingWebhookSink{}
	handler := newTestWebhookHandler(t, []byte(appSecret), official, sink, func() time.Time { return now })
	body := []byte(`{
  "object":"instagram",
  "private_unknown":"` + rawBodyCanary + `",
  "entry":[{"id":"` + official + `","messaging":[{
    "sender":{"id":"` + sender + `"},
    "recipient":{"id":"` + official + `"},
    "timestamp":1721386800123,
    "message":{"mid":"` + messageID + `","text":"` + challenge + `"}
  }]}]
}`)

	request := httptest.NewRequest(http.MethodPost, "/integrations/instagram/webhook", bytes.NewReader(body))
	request.Header.Set("X-Hub-Signature-256", signWebhookBody([]byte(appSecret), body))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %q", response.Code, response.Body.String())
	}
	if sink.calls != 1 || len(sink.items) != 1 || !sink.at.Equal(now) {
		t.Fatalf("sink = calls %d items %d at %s", sink.calls, len(sink.items), sink.at)
	}
	if sink.items[0].SenderIGSID != sender || sink.items[0].OfficialAccountID != official {
		t.Fatalf("reduced work identity mismatch: %v", sink.items[0])
	}
	for _, private := range []string{rawBodyCanary, messageID, challenge, sender, official, appSecret} {
		if strings.Contains(response.Body.String(), private) {
			t.Fatalf("response leaked %q: %s", private, response.Body.String())
		}
	}
}

func TestWebhookHandlerRejectsForgedMalformedAndOversizedDeliveriesBeforePersistence(t *testing.T) {
	t.Parallel()

	const secret = "synthetic-app-secret"
	validBody := []byte(`{
  "object":"instagram",
  "entry":[{"id":"official","messaging":[{
    "sender":{"id":"sender"},"recipient":{"id":"official"},
    "timestamp":1721386800123,
    "message":{"mid":"message","text":"CSKY-2345-6789-ABCD-E"}
  }]}]
}`)
	for _, test := range []struct {
		name       string
		body       []byte
		signBody   []byte
		duplicate  bool
		wantStatus int
	}{
		{name: "mutated exact bytes", body: append(append([]byte(nil), validBody...), ' '), signBody: validBody, wantStatus: http.StatusUnauthorized},
		{name: "missing signature", body: validBody, wantStatus: http.StatusUnauthorized},
		{name: "duplicate signature", body: validBody, signBody: validBody, duplicate: true, wantStatus: http.StatusUnauthorized},
		{name: "signed malformed JSON", body: []byte(`{"object":`), signBody: []byte(`{"object":`), wantStatus: http.StatusBadRequest},
		{name: "oversized", body: bytes.Repeat([]byte{'x'}, MaxWebhookBodyBytes+1), signBody: bytes.Repeat([]byte{'x'}, MaxWebhookBodyBytes+1), wantStatus: http.StatusRequestEntityTooLarge},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			sink := &recordingWebhookSink{}
			handler := newTestWebhookHandler(t, []byte(secret), "official", sink, time.Now)
			request := httptest.NewRequest(http.MethodPost, "/integrations/instagram/webhook", bytes.NewReader(test.body))
			if test.signBody != nil {
				request.Header.Add("X-Hub-Signature-256", signWebhookBody([]byte(secret), test.signBody))
				if test.duplicate {
					request.Header.Add("X-Hub-Signature-256", signWebhookBody([]byte(secret), test.signBody))
				}
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
			if sink.calls != 0 {
				t.Fatalf("sink calls = %d, want 0", sink.calls)
			}
			if strings.Contains(response.Body.String(), "CSKY-") || strings.Contains(response.Body.String(), secret) {
				t.Fatalf("generic response leaked private input: %q", response.Body.String())
			}
		})
	}
}

func TestWebhookHandlerEnforcesEventBatchAndDurableAcknowledgement(t *testing.T) {
	t.Parallel()

	const secret = "synthetic-app-secret"
	overLimit := supportedPayload(t, MaxSupportedEvents+1)
	sink := &recordingWebhookSink{}
	handler := newTestWebhookHandler(t, []byte(secret), "official", sink, time.Now)
	request := httptest.NewRequest(http.MethodPost, "/integrations/instagram/webhook", bytes.NewReader(overLimit))
	request.Header.Set("X-Hub-Signature-256", signWebhookBody([]byte(secret), overLimit))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusRequestEntityTooLarge || sink.calls != 0 {
		t.Fatalf("over event limit = status %d sink calls %d", response.Code, sink.calls)
	}

	body := []byte(`{"object":"instagram","entry":[]}`)
	sink.err = errors.New("synthetic database failure without private input")
	request = httptest.NewRequest(http.MethodPost, "/integrations/instagram/webhook", bytes.NewReader(body))
	request.Header.Set("X-Hub-Signature-256", signWebhookBody([]byte(secret), body))
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusInternalServerError || sink.calls != 1 {
		t.Fatalf("durability failure = status %d sink calls %d", response.Code, sink.calls)
	}
}

func TestWebhookHandlerHonorsTightenedBodyAndEventLimits(t *testing.T) {
	t.Parallel()

	const secret = "synthetic-app-secret"
	sink := &recordingWebhookSink{}
	digests, err := NewDigestCodec(bytes.Repeat([]byte{0x73}, 32), func(input string) (string, error) {
		return strings.ToUpper(strings.TrimSpace(input)), nil
	})
	if err != nil {
		t.Fatalf("NewDigestCodec: %v", err)
	}
	reducer, err := NewPayloadReducer("official", digests)
	if err != nil {
		t.Fatalf("NewPayloadReducer: %v", err)
	}
	handler, err := NewWebhookHandler(WebhookHandlerConfig{
		AppSecret:      []byte(secret),
		VerifyToken:    "synthetic-verify-token",
		Reducer:        reducer,
		Sink:           sink,
		BodyLimitBytes: 1024,
		MaxEvents:      1,
	})
	if err != nil {
		t.Fatalf("NewWebhookHandler: %v", err)
	}

	twoEvents := supportedPayload(t, 2)
	request := httptest.NewRequest(http.MethodPost, "/integrations/instagram/webhook", bytes.NewReader(twoEvents))
	request.Header.Set("X-Hub-Signature-256", signWebhookBody([]byte(secret), twoEvents))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusRequestEntityTooLarge || sink.calls != 0 {
		t.Fatalf("tight event limit = status %d sink %d", response.Code, sink.calls)
	}

	oversized := bytes.Repeat([]byte{'x'}, 1025)
	request = httptest.NewRequest(http.MethodPost, "/integrations/instagram/webhook", bytes.NewReader(oversized))
	request.Header.Set("X-Hub-Signature-256", signWebhookBody([]byte(secret), oversized))
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusRequestEntityTooLarge || sink.calls != 0 {
		t.Fatalf("tight body limit = status %d sink %d", response.Code, sink.calls)
	}
}

func TestWebhookHandlerAppliesSourceIPBeforeBodyAndGlobalAfterSignatureBeforeDecode(t *testing.T) {
	t.Parallel()

	const secret = "synthetic-app-secret"
	validBody := []byte(`{"object":"instagram","entry":[]}`)

	t.Run("source IP excess does not read or persist a partial body", func(t *testing.T) {
		limiter := &recordingWebhookLimiter{
			ipDecision: WebhookLimitDecision{Allowed: false, RetryAfter: 2100 * time.Millisecond},
		}
		reader := &recordingBodyReader{body: validBody}
		sink := &recordingWebhookSink{}
		handler := newRateLimitedTestWebhookHandler(t, []byte(secret), "official", sink, limiter)
		request := httptest.NewRequest(http.MethodPost, "/integrations/instagram/webhook", reader)
		request.Header.Set("X-Hub-Signature-256", signWebhookBody([]byte(secret), validBody))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)

		if response.Code != http.StatusTooManyRequests || response.Header().Get("Retry-After") != "3" {
			t.Fatalf("response = status %d retry %q", response.Code, response.Header().Get("Retry-After"))
		}
		if reader.read || limiter.ipCalls != 1 || limiter.globalCalls != 0 || sink.calls != 0 {
			t.Fatalf("order = body read %t ip %d global %d sink %d", reader.read, limiter.ipCalls, limiter.globalCalls, sink.calls)
		}
	})

	t.Run("invalid signature never consumes global ingress", func(t *testing.T) {
		limiter := &recordingWebhookLimiter{ipDecision: WebhookLimitDecision{Allowed: true}}
		handler := newRateLimitedTestWebhookHandler(t, []byte(secret), "official", &recordingWebhookSink{}, limiter)
		request := httptest.NewRequest(http.MethodPost, "/integrations/instagram/webhook", bytes.NewReader(validBody))
		request.Header.Set("X-Hub-Signature-256", "sha256=0000000000000000000000000000000000000000000000000000000000000000")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)

		if response.Code != http.StatusUnauthorized || limiter.ipCalls != 1 || limiter.globalCalls != 0 {
			t.Fatalf("response/order = status %d ip %d global %d", response.Code, limiter.ipCalls, limiter.globalCalls)
		}
	})

	t.Run("signed global excess wins before JSON decode and clamps retry", func(t *testing.T) {
		malformed := []byte(`{"object":`)
		limiter := &recordingWebhookLimiter{
			ipDecision:     WebhookLimitDecision{Allowed: true},
			globalDecision: WebhookLimitDecision{Allowed: false, RetryAfter: 10 * time.Minute},
		}
		sink := &recordingWebhookSink{}
		handler := newRateLimitedTestWebhookHandler(t, []byte(secret), "official", sink, limiter)
		request := httptest.NewRequest(http.MethodPost, "/integrations/instagram/webhook", bytes.NewReader(malformed))
		request.Header.Set("X-Hub-Signature-256", signWebhookBody([]byte(secret), malformed))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)

		if response.Code != http.StatusTooManyRequests || response.Header().Get("Retry-After") != "60" {
			t.Fatalf("response = status %d retry %q", response.Code, response.Header().Get("Retry-After"))
		}
		if limiter.ipCalls != 1 || limiter.globalCalls != 1 || sink.calls != 0 {
			t.Fatalf("order = ip %d global %d sink %d", limiter.ipCalls, limiter.globalCalls, sink.calls)
		}
	})

	t.Run("invalid redemption source is resolved only after bounded decode", func(t *testing.T) {
		limiter := &recordingWebhookLimiter{
			ipDecision:     WebhookLimitDecision{Allowed: true},
			globalDecision: WebhookLimitDecision{Allowed: true},
		}
		sink := &recordingWebhookSink{}
		handler := newRateLimitedTestWebhookHandler(t, []byte(secret), "official", sink, limiter)
		body := supportedPayload(t, 1)
		request := httptest.NewRequest(http.MethodPost, "/integrations/instagram/webhook", bytes.NewReader(body))
		request.Header.Set("X-Hub-Signature-256", signWebhookBody([]byte(secret), body))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)

		if response.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %q", response.Code, response.Body.String())
		}
		if limiter.ipCalls != 1 || limiter.globalCalls != 1 || limiter.invalidSourceCalls != 1 {
			t.Fatalf("limiter order = IP %d global %d invalid source %d", limiter.ipCalls, limiter.globalCalls, limiter.invalidSourceCalls)
		}
		if sink.guardedCalls != 1 || sink.calls != 0 {
			t.Fatalf("sink calls = guarded %d unguarded %d", sink.guardedCalls, sink.calls)
		}
	})

	t.Run("malformed signed payload never resolves invalid redemption source", func(t *testing.T) {
		malformed := []byte(`{"object":`)
		limiter := &recordingWebhookLimiter{
			ipDecision:     WebhookLimitDecision{Allowed: true},
			globalDecision: WebhookLimitDecision{Allowed: true},
		}
		sink := &recordingWebhookSink{}
		handler := newRateLimitedTestWebhookHandler(t, []byte(secret), "official", sink, limiter)
		request := httptest.NewRequest(http.MethodPost, "/integrations/instagram/webhook", bytes.NewReader(malformed))
		request.Header.Set("X-Hub-Signature-256", signWebhookBody([]byte(secret), malformed))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)

		if response.Code != http.StatusBadRequest || limiter.invalidSourceCalls != 0 || sink.guardedCalls != 0 {
			t.Fatalf("response/order = status %d invalid source %d guarded sink %d", response.Code, limiter.invalidSourceCalls, sink.guardedCalls)
		}
	})

	t.Run("invalid redemption source failure is generic and persists nothing", func(t *testing.T) {
		limiter := &recordingWebhookLimiter{
			ipDecision:       WebhookLimitDecision{Allowed: true},
			globalDecision:   WebhookLimitDecision{Allowed: true},
			invalidSourceErr: errors.New("synthetic private source-key failure"),
		}
		sink := &recordingWebhookSink{}
		handler := newRateLimitedTestWebhookHandler(t, []byte(secret), "official", sink, limiter)
		body := supportedPayload(t, 1)
		request := httptest.NewRequest(http.MethodPost, "/integrations/instagram/webhook", bytes.NewReader(body))
		request.Header.Set("X-Hub-Signature-256", signWebhookBody([]byte(secret), body))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)

		if response.Code != http.StatusServiceUnavailable || sink.guardedCalls != 0 ||
			strings.Contains(response.Body.String(), "source-key") {
			t.Fatalf("response/order = status %d body %q guarded sink %d", response.Code, response.Body.String(), sink.guardedCalls)
		}
	})
}

func TestWebhookHandlerRequiresGuardedSinkWhenIngressLimitingIsConfigured(t *testing.T) {
	t.Parallel()

	digests, err := NewDigestCodec(bytes.Repeat([]byte{0x73}, 32), func(input string) (string, error) {
		return strings.ToUpper(strings.TrimSpace(input)), nil
	})
	if err != nil {
		t.Fatalf("NewDigestCodec: %v", err)
	}
	reducer, err := NewPayloadReducer("official", digests)
	if err != nil {
		t.Fatalf("NewPayloadReducer: %v", err)
	}
	_, err = NewWebhookHandler(WebhookHandlerConfig{
		AppSecret:   []byte("synthetic-secret"),
		VerifyToken: "synthetic-token",
		Reducer:     reducer,
		Sink:        &unguardedWebhookSink{},
		Limiter:     &recordingWebhookLimiter{},
	})
	if err == nil || !strings.Contains(err.Error(), "guarded work sink") {
		t.Fatalf("NewWebhookHandler error = %v, want guarded sink requirement", err)
	}
}

func TestWebhookHandlerVerifiesCallbackWithoutReflectingInvalidQueries(t *testing.T) {
	t.Parallel()

	handler := newTestWebhookHandler(t, []byte("synthetic-secret"), "official", &recordingWebhookSink{}, time.Now)
	validURL := "/integrations/instagram/webhook?hub.mode=subscribe&hub.verify_token=synthetic-verify-token&hub.challenge=synthetic-callback"
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, validURL, nil))
	if response.Code != http.StatusOK || response.Body.String() != "synthetic-callback" {
		t.Fatalf("valid callback = status %d body %q", response.Code, response.Body.String())
	}

	for _, query := range []string{
		"?hub.mode=Subscribe&hub.verify_token=synthetic-verify-token&hub.challenge=private-invalid-challenge",
		"?hub.mode=subscribe&hub.verify_token=wrong-private-token&hub.challenge=private-invalid-challenge",
		"?hub.mode=subscribe&hub.verify_token=synthetic-verify-token&hub.verify_token=synthetic-verify-token&hub.challenge=private-invalid-challenge",
	} {
		response = httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/integrations/instagram/webhook"+query, nil))
		if response.Code != http.StatusForbidden {
			t.Errorf("invalid callback status = %d", response.Code)
		}
		if strings.Contains(response.Body.String(), "private-invalid-challenge") || strings.Contains(response.Body.String(), "wrong-private-token") {
			t.Errorf("invalid callback reflected private query: %q", response.Body.String())
		}
	}
}

type recordingWebhookSink struct {
	calls        int
	guardedCalls int
	items        []WorkItem
	at           time.Time
	err          error
}

type recordingWebhookLimiter struct {
	ipDecision         WebhookLimitDecision
	globalDecision     WebhookLimitDecision
	ipCalls            int
	globalCalls        int
	invalidSourceCalls int
	invalidLimiter     WebhookInvalidRedemptionLimiter
	invalidSourceErr   error
	err                error
}

func (l *recordingWebhookLimiter) AllowSourceIP(context.Context, *http.Request) (WebhookLimitDecision, error) {
	l.ipCalls++
	return l.ipDecision, l.err
}

func (l *recordingWebhookLimiter) AllowGlobal(context.Context) (WebhookLimitDecision, error) {
	l.globalCalls++
	return l.globalDecision, l.err
}

func (l *recordingWebhookLimiter) InvalidRedemptionSourceIP(*http.Request) (WebhookInvalidRedemptionLimiter, error) {
	l.invalidSourceCalls++
	if l.invalidSourceErr != nil {
		return nil, l.invalidSourceErr
	}
	if l.err != nil {
		return nil, l.err
	}
	if l.invalidLimiter == nil {
		l.invalidLimiter = &recordingInvalidRedemptionLimiter{decision: WebhookLimitDecision{Allowed: true}}
	}
	return l.invalidLimiter, nil
}

type recordingInvalidRedemptionLimiter struct {
	calls    int
	decision WebhookLimitDecision
	err      error
}

func (l *recordingInvalidRedemptionLimiter) AllowInvalidRedemption(context.Context) (WebhookLimitDecision, error) {
	l.calls++
	return l.decision, l.err
}

type recordingBodyReader struct {
	body []byte
	read bool
}

func (r *recordingBodyReader) Read(p []byte) (int, error) {
	r.read = true
	if len(r.body) == 0 {
		return 0, io.EOF
	}
	n := copy(p, r.body)
	r.body = r.body[n:]
	return n, nil
}

func (s *recordingWebhookSink) EnqueueWebhookWork(_ context.Context, items []WorkItem, at time.Time) (int, error) {
	s.calls++
	s.items = append([]WorkItem(nil), items...)
	s.at = at
	return len(items), s.err
}

func (s *recordingWebhookSink) EnqueueWebhookWorkGuarded(_ context.Context, items []WorkItem, at time.Time, _ WebhookInvalidRedemptionLimiter) (int, error) {
	s.guardedCalls++
	s.items = append([]WorkItem(nil), items...)
	s.at = at
	return len(items), s.err
}

type unguardedWebhookSink struct{}

func (*unguardedWebhookSink) EnqueueWebhookWork(context.Context, []WorkItem, time.Time) (int, error) {
	return 0, nil
}

func newTestWebhookHandler(t *testing.T, secret []byte, official string, sink WebhookWorkSink, now func() time.Time) *WebhookHandler {
	t.Helper()
	digests, err := NewDigestCodec(bytes.Repeat([]byte{0x73}, 32), func(input string) (string, error) {
		input = strings.ToUpper(strings.TrimSpace(input))
		if input != "CSKY-2345-6789-ABCD-E" {
			return "", ErrInvalidDigestInput
		}
		return input, nil
	})
	if err != nil {
		t.Fatalf("NewDigestCodec: %v", err)
	}
	reducer, err := NewPayloadReducer(official, digests)
	if err != nil {
		t.Fatalf("NewPayloadReducer: %v", err)
	}
	handler, err := NewWebhookHandler(WebhookHandlerConfig{
		AppSecret:   secret,
		VerifyToken: "synthetic-verify-token",
		Reducer:     reducer,
		Sink:        sink,
		Now:         now,
	})
	if err != nil {
		t.Fatalf("NewWebhookHandler: %v", err)
	}
	return handler
}

func newRateLimitedTestWebhookHandler(t *testing.T, secret []byte, official string, sink WebhookWorkSink, limiter WebhookRequestLimiter) *WebhookHandler {
	t.Helper()
	digests, err := NewDigestCodec(bytes.Repeat([]byte{0x73}, 32), func(input string) (string, error) {
		return strings.ToUpper(strings.TrimSpace(input)), nil
	})
	if err != nil {
		t.Fatalf("NewDigestCodec: %v", err)
	}
	reducer, err := NewPayloadReducer(official, digests)
	if err != nil {
		t.Fatalf("NewPayloadReducer: %v", err)
	}
	handler, err := NewWebhookHandler(WebhookHandlerConfig{
		AppSecret:   secret,
		VerifyToken: "synthetic-verify-token",
		Reducer:     reducer,
		Sink:        sink,
		Limiter:     limiter,
		Now:         time.Now,
	})
	if err != nil {
		t.Fatalf("NewWebhookHandler: %v", err)
	}
	return handler
}

func signWebhookBody(secret, body []byte) string {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
