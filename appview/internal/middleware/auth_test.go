package middleware

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/auth"
)

// discardLogger returns a slog.Logger that drops everything. Used by tests
// that assert HTTP behaviour without caring about log output.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// passthroughHandler captures the DID seen in context and responds 200.
func passthroughHandler(didSeen *syntax.DID) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*didSeen, _ = GetDID(r.Context())
		w.WriteHeader(http.StatusOK)
	})
}

func TestAuthenticated_RejectsMissingHeader(t *testing.T) {
	var seen syntax.DID
	h := Authenticated(&auth.MockAuthService{DefaultDID: "did:plc:default"}, discardLogger(), DevAuthPolicy{Mode: DevAuthDisabled})(passthroughHandler(&seen))
	req := httptest.NewRequest("GET", "/whoami", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestAuthenticated_RejectsMalformedHeader(t *testing.T) {
	var seen syntax.DID
	h := Authenticated(&auth.MockAuthService{DefaultDID: "did:plc:default"}, discardLogger(), DevAuthPolicy{Mode: DevAuthDisabled})(passthroughHandler(&seen))
	req := httptest.NewRequest("GET", "/whoami", nil)
	req.Header.Set("Authorization", "Token abc")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestAuthenticated_RejectsEmptyBearer(t *testing.T) {
	var seen syntax.DID
	h := Authenticated(&auth.MockAuthService{DefaultDID: "did:plc:default"}, discardLogger(), DevAuthPolicy{Mode: DevAuthDisabled})(passthroughHandler(&seen))
	req := httptest.NewRequest("GET", "/whoami", nil)
	req.Header.Set("Authorization", "Bearer ")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestAuthenticated_MockSuccessUsesDefaultDID(t *testing.T) {
	var seen syntax.DID
	h := Authenticated(&auth.MockAuthService{DefaultDID: "did:plc:default"}, discardLogger(), DevAuthPolicy{Mode: DevAuthDisabled})(passthroughHandler(&seen))
	req := httptest.NewRequest("GET", "/whoami", nil)
	req.Header.Set("Authorization", "Bearer anything")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if seen != "did:plc:default" {
		t.Errorf("did seen = %q, want did:plc:default", seen)
	}
}

func TestAuthenticated_MockHonoursXDevDID(t *testing.T) {
	var seen syntax.DID
	h := Authenticated(&auth.MockAuthService{DefaultDID: "did:plc:default"}, discardLogger(), DevAuthPolicy{Mode: DevAuthLocal})(passthroughHandler(&seen))
	req := httptest.NewRequest("GET", "/whoami", nil)
	req.Header.Set("Authorization", "Bearer anything")
	req.Header.Set("X-Dev-DID", "did:plc:override")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if seen != "did:plc:override" {
		t.Errorf("did seen = %q, want did:plc:override", seen)
	}
}

// A malformed X-Dev-DID is rejected at the boundary rather than silently
// dropped — surfacing the bug to the dev who set the header.
func TestAuthenticated_RejectsMalformedXDevDID(t *testing.T) {
	var seen syntax.DID
	h := Authenticated(&auth.MockAuthService{DefaultDID: "did:plc:default"}, discardLogger(), DevAuthPolicy{Mode: DevAuthLocal})(passthroughHandler(&seen))
	req := httptest.NewRequest("GET", "/whoami", nil)
	req.Header.Set("Authorization", "Bearer anything")
	req.Header.Set("X-Dev-DID", "not-a-did")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestAuthenticatedDisabledRejectsXDevDID(t *testing.T) {
	var seen syntax.DID
	handler := Authenticated(
		&auth.MockAuthService{DefaultDID: "did:plc:default"},
		discardLogger(),
		DevAuthPolicy{Mode: DevAuthDisabled},
	)(passthroughHandler(&seen))
	request := httptest.NewRequest(http.MethodGet, "/v1/whoami", nil)
	request.Header.Set("Authorization", "Bearer valid-real-session")
	request.Header.Set("X-Dev-DID", "did:plc:override")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest || seen != "" {
		t.Fatalf("status=%d DID=%q; body=%s", recorder.Code, seen, recorder.Body.String())
	}
}

// errorAuthSvc always returns an auth error regardless of token, standing in
// for any service that rejects every token (e.g. invalid token, revoked, etc.).
type errorAuthSvc struct{ err error }

func (e *errorAuthSvc) Authenticate(_ context.Context, _ string) (auth.AuthInfo, error) {
	return auth.AuthInfo{}, e.err
}

func (e *errorAuthSvc) AuthenticateRecovery(_ context.Context, _ string) (auth.AuthInfo, error) {
	return auth.AuthInfo{}, e.err
}

func TestAuthenticated_AlwaysErroringServiceReturns401(t *testing.T) {
	var seen syntax.DID
	h := Authenticated(&errorAuthSvc{err: auth.ErrAuthTokenInvalid}, discardLogger(), DevAuthPolicy{Mode: DevAuthDisabled})(passthroughHandler(&seen))
	req := httptest.NewRequest("GET", "/whoami", nil)
	req.Header.Set("Authorization", "Bearer anything")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body not json: %v", err)
	}
	if body["error"] != "unauthorized" {
		t.Errorf("error = %v, want unauthorized", body["error"])
	}
}

func TestAuthenticated_InfrastructureFailureReturnsRetryable503(t *testing.T) {
	var seen syntax.DID
	h := Authenticated(
		&errorAuthSvc{err: context.DeadlineExceeded},
		discardLogger(),
		DevAuthPolicy{Mode: DevAuthDisabled},
	)(passthroughHandler(&seen))
	req := httptest.NewRequest(http.MethodGet, "/whoami", nil)
	req.Header.Set("Authorization", "Bearer anything")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Retry-After") != "5" {
		t.Fatalf("Retry-After = %q, want 5", rec.Header().Get("Retry-After"))
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body not json: %v", err)
	}
	if body["error"] != "authentication_unavailable" {
		t.Fatalf("error = %v, want authentication_unavailable", body["error"])
	}
	if seen != "" {
		t.Fatalf("downstream saw DID %q", seen)
	}
}

func TestAuthenticatedRecoveryUsesTheRecoveryContractAndPreservesErrorTaxonomy(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
	}{
		{name: "invalid", err: auth.ErrAuthTokenInvalid, status: http.StatusUnauthorized},
		{name: "database unavailable", err: context.DeadlineExceeded, status: http.StatusServiceUnavailable},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var seen syntax.DID
			handler := AuthenticatedRecovery(
				&errorAuthSvc{err: test.err}, discardLogger(), DevAuthPolicy{Mode: DevAuthDisabled},
			)(passthroughHandler(&seen))
			request := httptest.NewRequest(http.MethodDelete, "/v1/account-deletion/intents/job", nil)
			request.Header.Set("Authorization", "Bearer recovery")
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			if recorder.Code != test.status || seen != "" {
				t.Fatalf("status=%d DID=%q body=%s", recorder.Code, seen, recorder.Body.String())
			}
			if test.status == http.StatusServiceUnavailable && recorder.Header().Get("Retry-After") != "5" {
				t.Fatalf("Retry-After=%q", recorder.Header().Get("Retry-After"))
			}
		})
	}
}

// fakeAuthSvc is a minimal AuthService that returns a fixed DID and session ID.
type fakeAuthSvc struct {
	did    syntax.DID
	sessID string
}

func (f *fakeAuthSvc) Authenticate(_ context.Context, _ string) (auth.AuthInfo, error) {
	return auth.AuthInfo{DID: f.did, SessionID: f.sessID}, nil
}

func TestAuthenticatedInjectsOAuthSessionID(t *testing.T) {
	svc := &fakeAuthSvc{did: "did:plc:xyz", sessID: "sess-123"}
	var gotDID syntax.DID
	var gotSID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotDID, _ = GetDID(r.Context())
		gotSID, _ = GetOAuthSessionID(r.Context())
	})
	h := Authenticated(svc, discardLogger(), DevAuthPolicy{Mode: DevAuthDisabled})(next)
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("Authorization", "Bearer t")
	h.ServeHTTP(httptest.NewRecorder(), req)
	if gotDID != "did:plc:xyz" || gotSID != "sess-123" {
		t.Fatalf("ctx mismatch: did=%q sid=%q", gotDID, gotSID)
	}
}

type devContextAuthService struct{}

func (devContextAuthService) Authenticate(ctx context.Context, _ string) (auth.AuthInfo, error) {
	did, ok := auth.DevDIDFromContext(ctx)
	if !ok {
		return auth.AuthInfo{}, auth.ErrAuthTokenInvalid
	}
	return auth.AuthInfo{DID: did}, nil
}

func TestAuthenticatedRemoteDevRequiresSecret(t *testing.T) {
	const secret = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFG"
	tests := []struct {
		name       string
		credential string
		did        string
		status     int
	}{
		{name: "missing credential", did: "did:plc:remote", status: http.StatusUnauthorized},
		{name: "wrong credential", credential: "wrong-secret", did: "did:plc:remote", status: http.StatusUnauthorized},
		{name: "correct credential", credential: secret, did: "did:plc:remote", status: http.StatusOK},
		{name: "malformed DID after authorization", credential: secret, did: "not-a-did", status: http.StatusBadRequest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var seen syntax.DID
			handler := Authenticated(
				devContextAuthService{},
				discardLogger(),
				DevAuthPolicy{Mode: DevAuthRemote, Secret: secret},
			)(passthroughHandler(&seen))
			request := httptest.NewRequest("GET", "/v1/whoami", nil)
			request.Header.Set("Authorization", "Bearer ignored")
			request.Header.Set("X-Dev-DID", test.did)
			if test.credential != "" {
				request.Header.Set("X-Craftsky-Dev-Authorization", test.credential)
			}
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			if recorder.Code != test.status {
				t.Fatalf("status = %d, want %d; body=%s", recorder.Code, test.status, recorder.Body.String())
			}
			if test.status == http.StatusOK && seen != "did:plc:remote" {
				t.Fatalf("DID = %q", seen)
			}
			if strings.Contains(recorder.Body.String(), secret) {
				t.Fatal("response exposed dev authorization secret")
			}
		})
	}
}
