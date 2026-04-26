package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
)

type stubResolver struct {
	handle syntax.Handle
	err    error
}

func (s stubResolver) ResolveHandle(_ context.Context, _ syntax.DID) (syntax.Handle, error) {
	return s.handle, s.err
}
func (s stubResolver) ResolveDID(_ context.Context, _ syntax.Handle) (syntax.DID, error) {
	return "", nil
}

// testLogger returns a slog.Logger that discards output but preserves
// the structured-log pipeline. Cheap and doesn't pollute test output.
func testLogger(t *testing.T) *slog.Logger {
	t.Helper()
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestWhoAmI_HappyPath(t *testing.T) {
	h := WhoAmIHandler(stubResolver{handle: syntax.Handle("alice.example")}, testLogger(t))
	req := httptest.NewRequest(http.MethodGet, "/v1/whoami", nil)
	req = req.WithContext(middleware.WithDID(req.Context(), "did:plc:abc"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rr.Code, rr.Body.String())
	}
	var body WhoAmIResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.DID != "did:plc:abc" {
		t.Errorf("did = %q", body.DID)
	}
	if body.Handle != "alice.example" {
		t.Errorf("handle = %q", body.Handle)
	}
}

func TestWhoAmI_DirectoryUnavailable(t *testing.T) {
	h := WhoAmIHandler(stubResolver{handle: "", err: errors.New("plc down")}, testLogger(t))
	req := httptest.NewRequest(http.MethodGet, "/v1/whoami", nil)
	req = req.WithContext(middleware.WithDID(req.Context(), "did:plc:abc"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", rr.Code)
	}
	var env envelope.Error
	if err := json.Unmarshal(rr.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if env.Error != "identity_unavailable" {
		t.Errorf("code = %q", env.Error)
	}
}

func TestWhoAmI_NoDIDInContext(t *testing.T) {
	h := WhoAmIHandler(stubResolver{handle: syntax.Handle("unused")}, testLogger(t))
	req := httptest.NewRequest(http.MethodGet, "/v1/whoami", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rr.Code)
	}
	var env envelope.Error
	if err := json.Unmarshal(rr.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if env.Error != "internal_error" {
		t.Errorf("code = %q", env.Error)
	}
}
