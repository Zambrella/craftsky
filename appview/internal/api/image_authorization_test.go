package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/imageupload"
	"social.craftsky/appview/internal/middleware"
)

type fakeImageUploadAuthorizationIssuer struct {
	authorization imageupload.Authorization
	err           error
	owner         syntax.DID
	sessionID     string
}

func (f *fakeImageUploadAuthorizationIssuer) IssueUpload(_ context.Context, owner syntax.DID, sessionID string) (imageupload.Authorization, error) {
	f.owner = owner
	f.sessionID = sessionID
	return f.authorization, f.err
}

func TestImageUploadAuthorizationHandler_ReturnsOnlyEphemeralAuthorization(t *testing.T) {
	t.Parallel()
	expiresAt := time.Date(2026, 9, 3, 12, 30, 0, 0, time.UTC)
	issuer := &fakeImageUploadAuthorizationIssuer{authorization: imageupload.Authorization{
		Token: "service-jwt", ExpiresAt: expiresAt, UploadEndpoint: "https://pds.example/xrpc/com.atproto.repo.uploadBlob",
	}}
	handler := api.ImageUploadAuthorizationHandler(issuer)
	request := httptest.NewRequest(http.MethodPost, "/v1/blobs/images/authorization", nil)
	ctx := middleware.WithDID(request.Context(), syntax.DID("did:plc:alice"))
	ctx = middleware.WithOAuthSessionID(ctx, "session-alice")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request.WithContext(ctx))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if recorder.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control = %q", recorder.Header().Get("Cache-Control"))
	}
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body) != 3 || body["uploadEndpoint"] != "https://pds.example/xrpc/com.atproto.repo.uploadBlob" || body["token"] != "service-jwt" || body["expiresAt"] != expiresAt.Format(time.RFC3339) {
		t.Fatalf("response = %#v", body)
	}
	if issuer.owner != syntax.DID("did:plc:alice") || issuer.sessionID != "session-alice" {
		t.Fatalf("issuer owner=%q session=%q", issuer.owner, issuer.sessionID)
	}
}

func TestImageUploadAuthorizationHandler_FailsClosed(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		withDID     bool
		withSession bool
		issuerErr   error
		wantStatus  int
	}{
		{name: "missing DID", wantStatus: http.StatusInternalServerError},
		{name: "missing session", withDID: true, wantStatus: http.StatusUnauthorized},
		{name: "issuer unavailable", withDID: true, withSession: true, issuerErr: errors.New("oauth-token dpop-key pds-url"), wantStatus: http.StatusBadGateway},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			issuer := &fakeImageUploadAuthorizationIssuer{err: test.issuerErr}
			handler := api.ImageUploadAuthorizationHandler(issuer)
			request := httptest.NewRequest(http.MethodPost, "/v1/blobs/images/authorization", nil)
			ctx := request.Context()
			if test.withDID {
				ctx = middleware.WithDID(ctx, syntax.DID("did:plc:alice"))
			}
			if test.withSession {
				ctx = middleware.WithOAuthSessionID(ctx, "session-alice")
			}
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request.WithContext(ctx))
			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
			}
			for _, secret := range []string{"oauth-token", "dpop-key", "pds-url"} {
				if body := recorder.Body.String(); test.issuerErr != nil && strings.Contains(body, secret) {
					t.Fatalf("response leaked %q: %s", secret, body)
				}
			}
		})
	}
}
