package middleware

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"social.craftsky/appview/internal/auth"
)

// discardLogger returns a slog.Logger that drops everything. Used by tests
// that assert HTTP behaviour without caring about log output.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// passthroughHandler captures the DID seen in context and responds 200.
func passthroughHandler(didSeen *string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*didSeen, _ = GetDID(r.Context())
		w.WriteHeader(http.StatusOK)
	})
}

func TestAuthenticated_RejectsMissingHeader(t *testing.T) {
	var seen string
	h := Authenticated(&auth.MockAuthService{DefaultDID: "did:plc:default"}, discardLogger())(passthroughHandler(&seen))
	req := httptest.NewRequest("GET", "/whoami", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestAuthenticated_RejectsMalformedHeader(t *testing.T) {
	var seen string
	h := Authenticated(&auth.MockAuthService{DefaultDID: "did:plc:default"}, discardLogger())(passthroughHandler(&seen))
	req := httptest.NewRequest("GET", "/whoami", nil)
	req.Header.Set("Authorization", "Token abc")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestAuthenticated_RejectsEmptyBearer(t *testing.T) {
	var seen string
	h := Authenticated(&auth.MockAuthService{DefaultDID: "did:plc:default"}, discardLogger())(passthroughHandler(&seen))
	req := httptest.NewRequest("GET", "/whoami", nil)
	req.Header.Set("Authorization", "Bearer ")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestAuthenticated_MockSuccessUsesDefaultDID(t *testing.T) {
	var seen string
	h := Authenticated(&auth.MockAuthService{DefaultDID: "did:plc:default"}, discardLogger())(passthroughHandler(&seen))
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
	var seen string
	h := Authenticated(&auth.MockAuthService{DefaultDID: "did:plc:default"}, discardLogger())(passthroughHandler(&seen))
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

// errorAuthSvc always returns an auth error regardless of token, standing in
// for any service that rejects every token (e.g. invalid token, revoked, etc.).
type errorAuthSvc struct{ err error }

func (e *errorAuthSvc) Authenticate(_ context.Context, _ string) (auth.AuthInfo, error) {
	return auth.AuthInfo{}, e.err
}

func TestAuthenticated_AlwaysErroringServiceReturns401(t *testing.T) {
	var seen string
	h := Authenticated(&errorAuthSvc{err: auth.ErrAuthTokenInvalid}, discardLogger())(passthroughHandler(&seen))
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

// fakeAuthSvc is a minimal AuthService that returns a fixed DID and session ID.
type fakeAuthSvc struct {
	did    string
	sessID string
}

func (f *fakeAuthSvc) Authenticate(_ context.Context, _ string) (auth.AuthInfo, error) {
	return auth.AuthInfo{DID: f.did, SessionID: f.sessID}, nil
}

func TestAuthenticatedInjectsOAuthSessionID(t *testing.T) {
	svc := &fakeAuthSvc{did: "did:plc:xyz", sessID: "sess-123"}
	var gotDID, gotSID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotDID, _ = GetDID(r.Context())
		gotSID, _ = GetOAuthSessionID(r.Context())
	})
	h := Authenticated(svc, discardLogger())(next)
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("Authorization", "Bearer t")
	h.ServeHTTP(httptest.NewRecorder(), req)
	if gotDID != "did:plc:xyz" || gotSID != "sess-123" {
		t.Fatalf("ctx mismatch: did=%q sid=%q", gotDID, gotSID)
	}
}
