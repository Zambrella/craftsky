package auth_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ctxkeys"
)

type capacityOAuthFlow struct{}

func (capacityOAuthFlow) StartLogin(context.Context, syntax.Handle, auth.HandoffMode, string, string) (string, error) {
	return "", auth.ErrAuthRequestCapacity
}

func (capacityOAuthFlow) CompleteCallback(context.Context, url.Values, auth.OAuthCallbackFinalizer) error {
	return nil
}

func TestLoginHandlerMapsPendingCapacityToRetryableServiceUnavailable(t *testing.T) {
	handlers := &auth.HTTPHandlers{
		OAuthFlow: capacityOAuthFlow{},
		Logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	request := httptest.NewRequest(http.MethodPost, "/v1/auth/login", strings.NewReader(
		`{"handle":"alice.example","handoffMode":"verified_link"}`,
	))
	request = request.WithContext(ctxkeys.WithDeviceID(request.Context(), "device-capacity"))
	response := httptest.NewRecorder()

	handlers.LoginHandler().ServeHTTP(response, request)

	expectEnvelopeError(t, response, http.StatusServiceUnavailable, "authentication_capacity_exhausted")
	retryAfter, err := strconv.Atoi(response.Header().Get("Retry-After"))
	if err != nil || retryAfter <= 0 {
		t.Fatalf("Retry-After=%q, want positive integer seconds", response.Header().Get("Retry-After"))
	}
	if response.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("Content-Type=%q, want application/json", response.Header().Get("Content-Type"))
	}
}

var _ auth.OAuthFlowCoordinator = capacityOAuthFlow{}
