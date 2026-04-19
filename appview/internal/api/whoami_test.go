package api

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/middleware"
)

// authTestMock is an inline AuthService that returns a fixed DID. It lets
// whoami_test.go inject a DID into the request context via the real
// Authenticated middleware, without depending on internal/auth.
type authTestMock struct{ did string }

func (m *authTestMock) Authenticate(ctx context.Context, token string) (auth.AuthInfo, error) {
	return auth.AuthInfo{DID: m.did}, nil
}

func TestWhoAmI_ReturnsDIDFromContext(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	outer := middleware.Authenticated(&authTestMock{did: "did:plc:alice"}, logger)(WhoAmIHandler())

	req := httptest.NewRequest("GET", "/whoami", nil)
	req.Header.Set("Authorization", "Bearer anything")
	rec := httptest.NewRecorder()
	outer.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	var body struct {
		DID string `json:"did"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.DID != "did:plc:alice" {
		t.Errorf("did = %q, want did:plc:alice", body.DID)
	}
}

func TestWhoAmI_WithoutDIDInContextReturns500(t *testing.T) {
	// Call the handler directly without running Authenticated — a routing
	// bug that's worth failing loudly on rather than silently returning
	// {"did":""}.
	h := WhoAmIHandler()
	req := httptest.NewRequest("GET", "/whoami", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
}
