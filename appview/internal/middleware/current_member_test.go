package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/ctxkeys"
)

func TestCurrentMemberAllowsOnlyCurrentCraftskyProfiles(t *testing.T) {
	t.Parallel()

	alice := syntax.DID("did:plc:synthetic-alice")
	checker := &stubCurrentMemberChecker{current: map[syntax.DID]bool{alice: true}}
	called := false
	handler := CurrentMember(checker, slog.Default())(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/migrations/instagram/account", nil)
	req = req.WithContext(WithDID(req.Context(), alice))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent || !called {
		t.Fatalf("current member response status=%d called=%t", rr.Code, called)
	}
	if len(checker.seen) != 1 || checker.seen[0] != alice {
		t.Fatalf("membership checks = %v, want only Alice", checker.seen)
	}
}

func TestCurrentMemberUsesProfileNotFoundBoundary(t *testing.T) {
	t.Parallel()

	departed := syntax.DID("did:plc:synthetic-departed")
	handler := CurrentMember(&stubCurrentMemberChecker{}, slog.Default())(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler called for departed member")
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/migrations/instagram/account", nil)
	req = req.WithContext(ctxkeys.WithRunID(WithDID(req.Context(), departed), "synthetic-request-id"))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
	var body envelope.Error
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if body.Error != "profile_not_found" || body.Message != "profile not found" || body.RequestID == "" {
		t.Fatalf("error envelope = %+v", body)
	}
}

func TestCurrentMemberFailsClosedOnMissingIdentityOrStoreError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		context func(context.Context) context.Context
		checker CurrentMemberChecker
		status  int
		code    string
	}{
		{
			name:    "authenticated DID missing",
			context: func(ctx context.Context) context.Context { return ctx },
			checker: &stubCurrentMemberChecker{},
			status:  http.StatusInternalServerError,
			code:    "missing_authenticated_did",
		},
		{
			name: "membership store unavailable",
			context: func(ctx context.Context) context.Context {
				return WithDID(ctx, syntax.DID("did:plc:synthetic-alice"))
			},
			checker: &stubCurrentMemberChecker{err: errors.New("synthetic private database error")},
			status:  http.StatusServiceUnavailable,
			code:    "membership_unavailable",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := CurrentMember(tt.checker, slog.Default())(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				t.Fatal("next handler called")
			}))
			req := httptest.NewRequest(http.MethodGet, "/v1/migrations/instagram/account", nil)
			req = req.WithContext(tt.context(req.Context()))
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			if rr.Code != tt.status {
				t.Fatalf("status = %d, want %d", rr.Code, tt.status)
			}
			var body envelope.Error
			if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode error: %v", err)
			}
			if body.Error != tt.code {
				t.Fatalf("code = %q, want %q", body.Error, tt.code)
			}
			if body.Message == "synthetic private database error" {
				t.Fatal("store error leaked through public envelope")
			}
		})
	}
}

type stubCurrentMemberChecker struct {
	current map[syntax.DID]bool
	err     error
	seen    []syntax.DID
}

func (s *stubCurrentMemberChecker) IsCurrentMember(_ context.Context, did syntax.DID) (bool, error) {
	s.seen = append(s.seen, did)
	return s.current[did], s.err
}
