package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/ctxkeys"
)

func TestExpectedHostRejectsUnexpectedV1HostWithEnvelope(t *testing.T) {
	for _, path := range []string{"/v1", "/v1/whoami", "//v1/whoami", "/../v1/whoami"} {
		t.Run(path, func(t *testing.T) {
			reached := false
			next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reached = true })
			handler := ExpectedHost(ExpectedHostPolicy{
				Authorities: []string{"appview.craftsky.social"},
			})(next)
			request := httptest.NewRequest("GET", "https://attacker.invalid"+path, nil)
			request.Host = "attacker.invalid"
			request = request.WithContext(ctxkeys.WithRunID(context.Background(), "request-123"))
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			if reached {
				t.Fatal("unexpected Host reached the next handler")
			}
			if recorder.Code != http.StatusMisdirectedRequest {
				t.Fatalf("status = %d, want 421", recorder.Code)
			}
			var got envelope.Error
			if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
				t.Fatalf("decode error envelope: %v", err)
			}
			if got.Error != "unexpected_host" || got.RequestID != "request-123" {
				t.Fatalf("envelope = %#v", got)
			}
		})
	}
}

func TestExpectedHostAuthorityMatrix(t *testing.T) {
	tests := []struct {
		name   string
		host   string
		status int
	}{
		{name: "canonical", host: "appview.craftsky.social", status: http.StatusNoContent},
		{name: "case normalised", host: "APPVIEW.CRAFTSKY.SOCIAL", status: http.StatusNoContent},
		{name: "default HTTPS port", host: "appview.craftsky.social:443", status: http.StatusNoContent},
		{name: "alternate port", host: "appview.craftsky.social:8443", status: http.StatusMisdirectedRequest},
		{name: "alternate host", host: "other.craftsky.social", status: http.StatusMisdirectedRequest},
		{name: "empty port", host: "appview.craftsky.social:", status: http.StatusMisdirectedRequest},
		{name: "userinfo", host: "user@appview.craftsky.social", status: http.StatusMisdirectedRequest},
		{name: "trailing dot", host: "appview.craftsky.social.", status: http.StatusMisdirectedRequest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reached := false
			handler := ExpectedHost(ExpectedHostPolicy{Authorities: []string{"appview.craftsky.social"}})(
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					reached = true
					w.WriteHeader(http.StatusNoContent)
				}),
			)
			request := httptest.NewRequest("GET", "https://appview.craftsky.social/oauth/client-metadata.json", nil)
			request.Host = test.host
			request.Header.Set("Forwarded", "host=attacker.invalid")
			request.Header.Set("X-Forwarded-Host", "attacker.invalid")
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			if recorder.Code != test.status {
				t.Fatalf("status = %d, want %d", recorder.Code, test.status)
			}
			if reached != (test.status == http.StatusNoContent) {
				t.Fatalf("reached = %t", reached)
			}
		})
	}
}
