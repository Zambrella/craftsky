package revenuecat

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"testing"
	"time"
)

func TestVerifyWebhookAuthenticationUsesExactRawBody(t *testing.T) {
	now := time.Unix(1788955200, 0)
	body := []byte(`{"event":{"id":"event-id"}}`)
	sign := func(at time.Time, value []byte) string {
		timestamp := strconv.FormatInt(at.Unix(), 10)
		mac := hmac.New(sha256.New, []byte("hmac-secret"))
		_, _ = mac.Write([]byte(timestamp + "."))
		_, _ = mac.Write(value)
		return "t=" + timestamp + ",v1=" + hex.EncodeToString(mac.Sum(nil))
	}
	valid := func() http.Header {
		headers := make(http.Header)
		headers.Set("Authorization", "Bearer exact-webhook-secret")
		headers.Set("X-RevenueCat-Webhook-Signature", sign(now, body))
		return headers
	}

	tests := []struct {
		name    string
		headers http.Header
		body    []byte
		valid   bool
	}{
		{name: "valid", headers: valid(), body: body, valid: true},
		{name: "missing authorization", headers: func() http.Header { h := valid(); h.Del("Authorization"); return h }(), body: body},
		{name: "wrong authorization", headers: func() http.Header { h := valid(); h.Set("Authorization", "Bearer wrong"); return h }(), body: body},
		{name: "duplicate authorization", headers: func() http.Header { h := valid(); h.Add("Authorization", "Bearer exact-webhook-secret"); return h }(), body: body},
		{name: "missing signature", headers: func() http.Header { h := valid(); h.Del("X-RevenueCat-Webhook-Signature"); return h }(), body: body},
		{name: "duplicate signature header", headers: func() http.Header { h := valid(); h.Add("X-RevenueCat-Webhook-Signature", sign(now, body)); return h }(), body: body},
		{name: "duplicate component", headers: func() http.Header {
			h := valid()
			h.Set("X-RevenueCat-Webhook-Signature", sign(now, body)+",v1=00")
			return h
		}(), body: body},
		{name: "malformed", headers: func() http.Header { h := valid(); h.Set("X-RevenueCat-Webhook-Signature", "invalid"); return h }(), body: body},
		{name: "stale", headers: func() http.Header {
			h := valid()
			h.Set("X-RevenueCat-Webhook-Signature", sign(now.Add(-5*time.Minute-time.Second), body))
			return h
		}(), body: body},
		{name: "future", headers: func() http.Header {
			h := valid()
			h.Set("X-RevenueCat-Webhook-Signature", sign(now.Add(5*time.Minute+time.Second), body))
			return h
		}(), body: body},
		{name: "mutated body", headers: valid(), body: append(append([]byte(nil), body...), ' ')},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := VerifyWebhookAuthentication(test.headers, test.body, "Bearer exact-webhook-secret", "hmac-secret", now, 5*time.Minute)
			if (err == nil) != test.valid {
				t.Fatalf("verification error = %v, valid=%t", err, test.valid)
			}
		})
	}
}
