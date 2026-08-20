package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/ctxkeys"
	"social.craftsky/appview/internal/ownerlifecycle"
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

func TestCurrentMemberEmbedsActiveOwnerGeneration(t *testing.T) {
	t.Parallel()

	alice := syntax.DID("did:plc:synthetic-alice")
	checker := &stubCurrentMemberChecker{current: map[syntax.DID]bool{alice: true}}
	lifecycles := stubOwnerLifecycleReader{row: ownerlifecycle.Lifecycle{
		Owner: alice, State: ownerlifecycle.StateActive, Generation: 9, AuthEpoch: 4,
	}}
	handler := CurrentMember(checker, slog.Default(), lifecycles)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		generation, ok := GetOwnerGeneration(r.Context())
		if !ok || generation != 9 {
			t.Fatalf("owner generation = %d/%t", generation, ok)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/v1/scheduled-post-media/one", nil)
	req = req.WithContext(WithDID(req.Context(), alice))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d", rr.Code)
	}
}

type stubOwnerLifecycleReader struct {
	row ownerlifecycle.Lifecycle
	err error
}

func (stub stubOwnerLifecycleReader) Get(context.Context, syntax.DID) (ownerlifecycle.Lifecycle, error) {
	return stub.row, stub.err
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

func TestCurrentMemberDetachesUnreadBodyOnEveryRejection(t *testing.T) {
	alice := syntax.DID("did:plc:synthetic-alice")
	tests := []struct {
		name       string
		context    func(context.Context) context.Context
		checker    CurrentMemberChecker
		lifecycles []OwnerLifecycleReader
		wantStatus int
	}{
		{
			name:       "authenticated DID missing",
			context:    func(ctx context.Context) context.Context { return ctx },
			checker:    &stubCurrentMemberChecker{},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "membership store unavailable",
			context:    func(ctx context.Context) context.Context { return WithDID(ctx, alice) },
			checker:    &stubCurrentMemberChecker{err: errors.New("database unavailable")},
			wantStatus: http.StatusServiceUnavailable,
		},
		{
			name:       "not a current member",
			context:    func(ctx context.Context) context.Context { return WithDID(ctx, alice) },
			checker:    &stubCurrentMemberChecker{},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "lifecycle store unavailable",
			context:    func(ctx context.Context) context.Context { return WithDID(ctx, alice) },
			checker:    &stubCurrentMemberChecker{current: map[syntax.DID]bool{alice: true}},
			lifecycles: []OwnerLifecycleReader{stubOwnerLifecycleReader{err: errors.New("database unavailable")}},
			wantStatus: http.StatusServiceUnavailable,
		},
		{
			name:       "owner lifecycle inactive",
			context:    func(ctx context.Context) context.Context { return WithDID(ctx, alice) },
			checker:    &stubCurrentMemberChecker{current: map[syntax.DID]bool{alice: true}},
			lifecycles: []OwnerLifecycleReader{stubOwnerLifecycleReader{row: ownerlifecycle.Lifecycle{Owner: alice, State: ownerlifecycle.StateDeparted}}},
			wantStatus: http.StatusNotFound,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			probe := &countingBodyReader{reader: strings.NewReader(`{"text":"unread"}`)}
			handler := CurrentMember(test.checker, slog.Default(), test.lifecycles...)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				t.Fatal("next handler called for rejected member")
			}))
			request := httptest.NewRequest(http.MethodPost, "/v1/posts", probe)
			request = request.WithContext(test.context(request.Context()))
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)

			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", recorder.Code, test.wantStatus, recorder.Body.String())
			}
			if probe.bytesRead != 0 {
				t.Fatalf("body bytes read = %d, want 0", probe.bytesRead)
			}
			if request.Body != http.NoBody || !request.Close || recorder.Header().Get("Connection") != "close" {
				t.Fatalf("unread body was not detached: body=%T request.Close=%t headers=%v", request.Body, request.Close, recorder.Header())
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
