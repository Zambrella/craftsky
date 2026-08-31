package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/ctxkeys"
)

const registrationSecretSentinels = "provider-email-sentinel provider-password-sentinel auth-code-sentinel par-uri-sentinel access-token-sentinel refresh-token-sentinel dpop-private-key-sentinel handoff-code-sentinel provider-error-text-sentinel"

type registrationSecurityFlow struct {
	err error
}

func (flow registrationSecurityFlow) StartLogin(context.Context, syntax.Handle, HandoffMode, string, string) (string, error) {
	return "", fmt.Errorf("unexpected login")
}

func (flow registrationSecurityFlow) StartRegistration(context.Context, HandoffMode, string, string) (string, error) {
	return "", flow.err
}

func (registrationSecurityFlow) CompleteCallback(context.Context, url.Values, OAuthCallbackFinalizer) error {
	return nil
}

func TestRegistrationStartLogsAndEnvelopeRedactSecrets(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	failure := fmt.Errorf("%s: %w", registrationSecretSentinels, &RegistrationOAuthError{
		Stage: RegistrationOAuthStagePAR,
		Code:  RegistrationOAuthProviderUnavailable,
	})
	handlers := &HTTPHandlers{OAuthFlow: registrationSecurityFlow{err: failure}, Logger: logger}
	request := httptest.NewRequest(http.MethodPost, "/v1/auth/registrations", strings.NewReader(`{"handoffMode":"verified_link"}`))
	request = request.WithContext(ctxkeys.WithRunID(request.Context(), "request-registration-safe"))
	request = request.WithContext(ctxkeys.WithDeviceID(request.Context(), "stable-device"))
	response := httptest.NewRecorder()

	handlers.RegistrationHandler().ServeHTTP(response, request)

	if response.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502; body=%s", response.Code, response.Body.String())
	}
	var problem envelope.Error
	if err := json.Unmarshal(response.Body.Bytes(), &problem); err != nil {
		t.Fatal(err)
	}
	if problem.Error != "registration_provider_unavailable" || problem.RequestID != "request-registration-safe" {
		t.Fatalf("error envelope = %+v", problem)
	}
	combined := logs.String() + response.Body.String()
	for _, sentinel := range strings.Fields(registrationSecretSentinels) {
		if strings.Contains(combined, sentinel) {
			t.Fatalf("registration telemetry retained %q: %s", sentinel, combined)
		}
	}
	for _, safe := range []string{`"operation":"registration.start"`, `"stage":"par"`, `"error_category":"providerUnavailable"`, `"run_id":"request-registration-safe"`} {
		if !strings.Contains(logs.String(), safe) {
			t.Fatalf("registration log missing %s: %s", safe, logs.String())
		}
	}
}
