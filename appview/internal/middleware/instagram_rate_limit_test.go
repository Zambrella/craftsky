package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/instagram"
)

func TestTrustedClientIPIgnoresForwardingHeadersFromUntrustedPeer(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/integrations/instagram/webhook", nil)
	req.RemoteAddr = "198.51.100.7:4321"
	req.Header.Set("Forwarded", "for=203.0.113.9")
	req.Header.Set("X-Forwarded-For", "192.0.2.44")

	got, err := TrustedClientIP(req, []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")})
	if err != nil {
		t.Fatalf("TrustedClientIP: %v", err)
	}
	if want := netip.MustParseAddr("198.51.100.7"); got != want {
		t.Fatalf("client IP = %s, want socket peer %s", got, want)
	}
}

func TestTrustedClientIPSelectsFirstUntrustedHopFromTrustedChain(t *testing.T) {
	t.Parallel()

	trusted := []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")}
	for _, test := range []struct {
		name   string
		header string
		value  string
	}{
		{name: "standard Forwarded", header: "Forwarded", value: `for=203.0.113.9;proto=https, for="10.0.0.2:8443"`},
		{name: "legacy X-Forwarded-For", header: "X-Forwarded-For", value: "203.0.113.9, 10.0.0.2"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodPost, "/integrations/instagram/webhook", nil)
			req.RemoteAddr = "10.0.0.3:443"
			req.Header.Set(test.header, test.value)
			got, err := TrustedClientIP(req, trusted)
			if err != nil {
				t.Fatalf("TrustedClientIP: %v", err)
			}
			if want := netip.MustParseAddr("203.0.113.9"); got != want {
				t.Fatalf("client IP = %s, want %s", got, want)
			}
		})
	}
}

func TestTrustedClientIPFallsBackToSocketPeerForMalformedTrustedHeader(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/integrations/instagram/webhook", nil)
	req.RemoteAddr = "10.0.0.3:443"
	req.Header.Set("Forwarded", `for="private-user-controlled-value"`)
	got, err := TrustedClientIP(req, []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")})
	if err != nil {
		t.Fatalf("TrustedClientIP: %v", err)
	}
	if want := netip.MustParseAddr("10.0.0.3"); got != want {
		t.Fatalf("client IP = %s, want conservative peer fallback %s", got, want)
	}
}

func TestInstagramPersistentRateLimitUsesPrivateDIDDeviceAndTrustedIPKeys(t *testing.T) {
	t.Parallel()

	limiter := &recordingInstagramLimiter{
		decisions: map[instagram.RateLimitScope]instagram.RateLimitDecision{
			instagram.RateLimitChallengeIP: {Allowed: false, RetryAfter: 2100 * time.Millisecond},
		},
	}
	rules := []InstagramRateLimitRule{
		{Scope: instagram.RateLimitChallengeDID, Identity: InstagramRateIdentityDID, Window: 15 * time.Minute, Limit: 5},
		{Scope: instagram.RateLimitChallengeDevice, Identity: InstagramRateIdentityDevice, Window: 15 * time.Minute, Limit: 10},
		{Scope: instagram.RateLimitChallengeIP, Identity: InstagramRateIdentityClientIP, Window: 15 * time.Minute, Limit: 30},
	}
	called := false
	handler := InstagramPersistentRateLimit(limiter, rules, []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")}, nil)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodPost, "/v1/migrations/instagram/verifications", nil)
	req.RemoteAddr = "10.0.0.3:443"
	req.Header.Set("Forwarded", "for=203.0.113.9, for=10.0.0.2")
	ctx := WithDID(req.Context(), syntax.DID("did:plc:synthetic-alice"))
	ctx = WithDeviceID(ctx, "synthetic-device")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if called {
		t.Fatal("limited request reached handler")
	}
	if rr.Code != http.StatusTooManyRequests || rr.Header().Get("Retry-After") != "3" {
		t.Fatalf("limited response = status %d retry %q", rr.Code, rr.Header().Get("Retry-After"))
	}
	var body envelope.Error
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil || body.Error != "rate_limited" {
		t.Fatalf("error envelope = %+v, %v", body, err)
	}
	want := []recordedInstagramRateKey{
		{scope: instagram.RateLimitChallengeDID, identifier: "did:plc:synthetic-alice"},
		{scope: instagram.RateLimitChallengeDevice, identifier: "synthetic-device"},
		{scope: instagram.RateLimitChallengeIP, identifier: "203.0.113.9"},
	}
	if len(limiter.keys) != len(want) {
		t.Fatalf("rate keys = %+v, want %+v", limiter.keys, want)
	}
	for i := range want {
		if limiter.keys[i] != want[i] {
			t.Fatalf("rate key[%d] = %+v, want %+v", i, limiter.keys[i], want[i])
		}
	}
}

func TestInstagramPersistentRateLimitFailsClosedWhenLimiterOrIdentityUnavailable(t *testing.T) {
	t.Parallel()

	rule := []InstagramRateLimitRule{{
		Scope: instagram.RateLimitConfirmationDID, Identity: InstagramRateIdentityDID,
		Window: time.Hour, Limit: 20,
	}}
	for _, test := range []struct {
		name    string
		limiter InstagramPersistentLimiter
	}{
		{name: "limiter unavailable"},
		{name: "DID unavailable", limiter: &recordingInstagramLimiter{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			called := false
			handler := InstagramPersistentRateLimit(test.limiter, rule, nil, nil)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				called = true
			}))
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/confirm", nil))
			if called || rr.Code != http.StatusServiceUnavailable {
				t.Fatalf("response = called %t status %d", called, rr.Code)
			}
		})
	}
}

func TestInstagramWebhookRateLimiterUsesTrustedIPThenGlobalScopes(t *testing.T) {
	t.Parallel()

	limiter := &recordingInstagramLimiter{}
	adapter, err := NewInstagramWebhookRateLimiter(
		limiter,
		[]netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")},
		300,
		1000,
		30,
	)
	if err != nil {
		t.Fatalf("NewInstagramWebhookRateLimiter: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/integrations/instagram/webhook", nil)
	req.RemoteAddr = "10.0.0.3:443"
	req.Header.Set("Forwarded", "for=203.0.113.9, for=10.0.0.2")
	if decision, err := adapter.AllowSourceIP(req.Context(), req); err != nil || !decision.Allowed {
		t.Fatalf("AllowSourceIP = %+v, %v", decision, err)
	}
	if decision, err := adapter.AllowGlobal(req.Context()); err != nil || !decision.Allowed {
		t.Fatalf("AllowGlobal = %+v, %v", decision, err)
	}
	invalidLimiter, err := adapter.InvalidRedemptionSourceIP(req)
	if err != nil {
		t.Fatalf("InvalidRedemptionSourceIP: %v", err)
	}
	if decision, err := invalidLimiter.AllowInvalidRedemption(req.Context()); err != nil || !decision.Allowed {
		t.Fatalf("AllowInvalidRedemption = %+v, %v", decision, err)
	}
	want := []recordedInstagramRateKey{
		{scope: instagram.RateLimitWebhookIP, identifier: "203.0.113.9"},
		{scope: instagram.RateLimitWebhookGlobal, identifier: ""},
		{scope: instagram.RateLimitInvalidRedemptionIP, identifier: "203.0.113.9"},
	}
	if len(limiter.keys) != len(want) {
		t.Fatalf("keys = %+v, want %+v", limiter.keys, want)
	}
	for i := range want {
		if limiter.keys[i] != want[i] {
			t.Fatalf("key[%d] = %+v, want %+v", i, limiter.keys[i], want[i])
		}
	}
	lastAllow := limiter.allows[len(limiter.allows)-1]
	if lastAllow.scope != instagram.RateLimitInvalidRedemptionIP || lastAllow.window != 15*time.Minute || lastAllow.limit != 30 {
		t.Fatalf("invalid redemption allowance = %+v", lastAllow)
	}
	if rendered := fmt.Sprintf("%v %#v", invalidLimiter, invalidLimiter); strings.Contains(rendered, "203.0.113.9") || !strings.Contains(rendered, "REDACTED") {
		t.Fatalf("invalid limiter representation is not redacted: %q", rendered)
	}
}

func TestInstagramWebhookInvalidRedemptionLimiterIgnoresForwardedIPFromUntrustedPeer(t *testing.T) {
	t.Parallel()

	limiter := &recordingInstagramLimiter{}
	adapter, err := NewInstagramWebhookRateLimiter(
		limiter,
		[]netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")},
		300,
		1000,
		2,
	)
	if err != nil {
		t.Fatalf("NewInstagramWebhookRateLimiter: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/integrations/instagram/webhook", nil)
	req.RemoteAddr = "198.51.100.7:4321"
	req.Header.Set("Forwarded", "for=203.0.113.9")
	invalidLimiter, err := adapter.InvalidRedemptionSourceIP(req)
	if err != nil {
		t.Fatalf("InvalidRedemptionSourceIP: %v", err)
	}
	for call := 1; call <= 3; call++ {
		if _, err := invalidLimiter.AllowInvalidRedemption(req.Context()); err != nil {
			t.Fatalf("AllowInvalidRedemption call %d: %v", call, err)
		}
	}
	if got := limiter.keys[0]; got.scope != instagram.RateLimitInvalidRedemptionIP || got.identifier != "198.51.100.7" {
		t.Fatalf("invalid redemption key = %+v, want untrusted socket peer", got)
	}
	for _, allow := range limiter.allows {
		if allow.window != 15*time.Minute || allow.limit != 2 {
			t.Fatalf("invalid redemption allowance = %+v", allow)
		}
	}
}

type recordedInstagramRateKey struct {
	scope      instagram.RateLimitScope
	identifier string
}

type recordedInstagramAllow struct {
	scope  instagram.RateLimitScope
	window time.Duration
	limit  int
}

type recordingInstagramLimiter struct {
	keys      []recordedInstagramRateKey
	allows    []recordedInstagramAllow
	decisions map[instagram.RateLimitScope]instagram.RateLimitDecision
	err       error
}

func (l *recordingInstagramLimiter) Key(scope instagram.RateLimitScope, identifier []byte) (instagram.RateLimitKey, error) {
	l.keys = append(l.keys, recordedInstagramRateKey{scope: scope, identifier: string(identifier)})
	return instagram.RateLimitKey{}, l.err
}

func (l *recordingInstagramLimiter) Allow(_ context.Context, key instagram.RateLimitKey, window time.Duration, limit int) (instagram.RateLimitDecision, error) {
	if l.err != nil {
		return instagram.RateLimitDecision{}, l.err
	}
	if len(l.keys) == 0 {
		return instagram.RateLimitDecision{Allowed: true}, nil
	}
	scope := l.keys[len(l.keys)-1].scope
	l.allows = append(l.allows, recordedInstagramAllow{scope: scope, window: window, limit: limit})
	decision, ok := l.decisions[scope]
	if !ok {
		decision.Allowed = true
	}
	return decision, nil
}
