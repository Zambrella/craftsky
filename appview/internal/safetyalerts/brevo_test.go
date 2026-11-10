package safetyalerts

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestBrevoPayloadAndResponsePrivacy(t *testing.T) {
	id := uuid.New()
	for _, tc := range []struct {
		status int
		body   string
		ok     bool
	}{
		{201, `{"messageId":"provider-canary"}`, true},
		{400, `{"code":"duplicate_parameter","message":"provider-private-canary"}`, true},
		{400, `{"code":"invalid_parameter","message":"provider-private-canary"}`, false},
		{429, `{"message":"provider-private-canary"}`, false},
		{503, `provider-private-canary`, false},
	} {
		t.Run(fmt.Sprint(tc.status, tc.ok), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" || r.Header.Get("api-key") != "credential-canary" {
					t.Error("missing provider method/auth")
				}
				raw, _ := io.ReadAll(r.Body)
				var body map[string]any
				if err := json.Unmarshal(raw, &body); err != nil {
					t.Fatal(err)
				}
				if len(body) != 5 || body["subject"] != "CraftSky: urgent safety review needed" || !strings.Contains(body["textContent"].(string), "https://appview.craftsky.social/admin/") {
					t.Errorf("unexpected payload: %s", raw)
				}
				if body["headers"].(map[string]any)["idempotencyKey"] != id.String() {
					t.Error("unstable idempotency key")
				}
				if body["sender"].(map[string]any)["email"] != mailbox || body["to"].([]any)[0].(map[string]any)["email"] != mailbox {
					t.Error("incorrect routing")
				}
				if strings.Contains(string(raw), "credential-canary") {
					t.Error("credential in payload")
				}
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, tc.body)
			}))
			defer server.Close()
			b, err := NewBrevo("credential-canary", "https://appview.craftsky.social", false)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(fmt.Sprintf("%+v %#v", b, b), "credential-canary") {
				t.Fatal("credential exposed in diagnostics")
			}
			b.endpoint = server.URL
			err = b.Send(t.Context(), id)
			if (err == nil) != tc.ok {
				t.Fatalf("Send: %v", err)
			}
			if err != nil && strings.Contains(err.Error(), "provider-private-canary") {
				t.Fatal("provider prose leaked")
			}
		})
	}
}

func TestBrevoRejectsRedirectsAndInvalidOrigins(t *testing.T) {
	for _, origin := range []string{"http://example.com", "https://user:pass@example.com", "https://example.com/?token=x", "https://example.com/#secret", "https://example.com/private"} {
		if _, err := NewBrevo("key", origin, false); err == nil {
			t.Errorf("accepted %s", origin)
		}
	}
	leaked := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { leaked = true }))
	defer target.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer redirect.Close()
	b, _ := NewBrevo("key", "https://appview.craftsky.social", false)
	b.endpoint = redirect.URL
	if err := b.Send(context.Background(), uuid.New()); err == nil || leaked {
		t.Fatal("redirect followed or considered delivered")
	}
}

func TestConsoleOriginDevelopmentBoundary(t *testing.T) {
	for _, origin := range []string{"http://localhost:18080", "http://127.0.0.1:18080", "http://[::1]:18080"} {
		if _, err := NewBrevo("key", origin, true); err != nil {
			t.Errorf("development loopback rejected: %v", err)
		}
		if _, err := NewBrevo("key", origin, false); err == nil {
			t.Error("production accepted loopback HTTP")
		}
	}
	for _, origin := range []string{"http://example.org", "http://localhost.example.org", "http://192.168.1.2", "http://127.0.0.1:0", "http://127.0.0.1:65536", "http://user:secret@localhost", "http://localhost/private", "http://localhost/?", "http://localhost/#", "http://localhost/%2f", " http://localhost", "http://localhost\\private"} {
		if _, err := NewBrevo("credential-canary", origin, true); err == nil || strings.Contains(err.Error(), "credential-canary") {
			t.Errorf("invalid origin accepted or credential leaked")
		}
	}
}

func TestBrevoDevelopmentConsoleLinkInPayload(t *testing.T) {
	b, err := NewBrevo("credential-canary", "http://127.0.0.1:18444/", true)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if !strings.Contains(body["textContent"].(string), "http://127.0.0.1:18444/admin/") {
			t.Error("incorrect console link")
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()
	b.endpoint = server.URL
	if err := b.Send(t.Context(), uuid.New()); err != nil {
		t.Fatal(err)
	}
}
