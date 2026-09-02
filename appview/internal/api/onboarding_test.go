package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/ctxkeys"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/testdb"
)

type fakeOnboardingStatusStore struct {
	status    api.OnboardingStatus
	dids      []syntax.DID
	err       error
	completed bool
}

func TestOnboardingStatusStoreCompletesPermanentlyAndIsolatesDIDs(t *testing.T) {
	t.Parallel()
	migration, err := os.ReadFile("../../migrations/000062_account_onboarding_completion.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, `
		CREATE TABLE owner_lifecycles (
			owner_did TEXT PRIMARY KEY,
			state TEXT NOT NULL,
			generation BIGINT NOT NULL,
			auth_epoch BIGINT NOT NULL DEFAULT 1,
			transition_reason TEXT NOT NULL DEFAULT 'test',
			transitioned_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			terminal_at TIMESTAMPTZ,
			purge_completed_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		INSERT INTO owner_lifecycles (owner_did, state, generation)
		VALUES ('did:plc:alice', 'active', 1), ('did:plc:bob', 'active', 1);
	`+string(migration))
	store := api.NewOnboardingStatusStore(pool)
	ctx := ownerlifecycle.WithExpectedGeneration(context.Background(), 1)
	alice := syntax.DID("did:plc:alice")
	bob := syntax.DID("did:plc:bob")

	initial, err := store.Status(ctx, alice)
	if err != nil || initial.Completed || initial.CompletedAt != nil {
		t.Fatalf("initial Alice status = %+v, %v", initial, err)
	}
	first, err := store.Complete(ctx, alice)
	if err != nil || !first.Completed || first.CompletedAt == nil {
		t.Fatalf("first completion = %+v, %v", first, err)
	}
	second, err := store.Complete(ctx, alice)
	if err != nil || second.CompletedAt == nil || !second.CompletedAt.Equal(*first.CompletedAt) {
		t.Fatalf("idempotent completion = %+v, %v; first = %+v", second, err, first)
	}
	bobStatus, err := store.Status(ctx, bob)
	if err != nil || bobStatus.Completed || bobStatus.CompletedAt != nil {
		t.Fatalf("Bob status = %+v, %v", bobStatus, err)
	}
	if _, err := pool.Exec(context.Background(), `
		UPDATE owner_lifecycles
		SET state = 'departed', generation = 2
		WHERE owner_did = $1
	`, alice); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Status(ctx, alice); !errors.Is(err, ownerlifecycle.ErrOwnerNotActive) {
		t.Fatalf("stale lifecycle status read error = %v, want %v", err, ownerlifecycle.ErrOwnerNotActive)
	}
}

func (s *fakeOnboardingStatusStore) Status(_ context.Context, did syntax.DID) (api.OnboardingStatus, error) {
	s.dids = append(s.dids, did)
	return s.status, s.err
}

func (s *fakeOnboardingStatusStore) Complete(_ context.Context, did syntax.DID) (api.OnboardingStatus, error) {
	s.dids = append(s.dids, did)
	s.completed = true
	return s.status, s.err
}

func TestGetOnboardingStatusReturnsIncompleteForAuthenticatedDID(t *testing.T) {
	t.Parallel()
	did := syntax.DID("did:plc:onboarding-incomplete")
	store := &fakeOnboardingStatusStore{status: api.OnboardingStatus{Completed: false}}
	request := httptest.NewRequest(http.MethodGet, "/v1/onboarding/status", nil)
	ctx := middleware.WithDID(request.Context(), did)
	request = request.WithContext(ctxkeys.WithRunID(ctx, "onboarding-status-request"))
	recorder := httptest.NewRecorder()

	api.GetOnboardingStatusHandler(store, nil).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response["completed"] != false {
		t.Fatalf("response = %#v", response)
	}
	if _, ok := response["completedAt"]; ok {
		t.Fatalf("incomplete response contains completedAt: %#v", response)
	}
	if len(store.dids) != 1 || store.dids[0] != did {
		t.Fatalf("store DIDs = %v, want authenticated DID", store.dids)
	}
}

func TestCompleteOnboardingReturnsAuthenticatedDIDAndCamelCaseTimestamp(t *testing.T) {
	t.Parallel()
	did := syntax.DID("did:plc:onboarding-complete")
	completedAt := time.Date(2026, 8, 31, 12, 30, 0, 0, time.UTC)
	store := &fakeOnboardingStatusStore{status: api.OnboardingStatus{
		Completed: true, CompletedAt: &completedAt,
	}}
	request := httptest.NewRequest(http.MethodPost, "/v1/onboarding/completion", nil)
	ctx := middleware.WithDID(request.Context(), did)
	request = request.WithContext(ctxkeys.WithRunID(ctx, "onboarding-complete-request"))
	recorder := httptest.NewRecorder()

	api.CompleteOnboardingHandler(store, nil).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response["completed"] != true || response["completedAt"] != completedAt.Format(time.RFC3339) {
		t.Fatalf("response = %#v", response)
	}
	if !store.completed || len(store.dids) != 1 || store.dids[0] != did {
		t.Fatalf("completion store state = completed:%t DIDs:%v", store.completed, store.dids)
	}
}

func TestOnboardingHandlerFailureUsesCanonicalRedactedEnvelope(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name      string
		method    string
		path      string
		operation string
		handler   func(api.OnboardingStatusService, *slog.Logger) http.Handler
	}{
		{name: "read", method: http.MethodGet, path: "/v1/onboarding/status", operation: "onboarding.status.read", handler: api.GetOnboardingStatusHandler},
		{name: "write", method: http.MethodPost, path: "/v1/onboarding/completion", operation: "onboarding.completion.write", handler: api.CompleteOnboardingHandler},
	} {
		t.Run(test.name, func(t *testing.T) {
			var logs bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&logs, nil))
			store := &fakeOnboardingStatusStore{err: errors.New("database password and did:plc:secret at 2026-08-31T12:30:00Z")}
			request := httptest.NewRequest(test.method, test.path, nil)
			ctx := middleware.WithDID(request.Context(), syntax.DID("did:plc:alice"))
			request = request.WithContext(ctxkeys.WithRunID(ctx, "onboarding-failure-request"))
			recorder := httptest.NewRecorder()

			test.handler(store, logger).ServeHTTP(recorder, request)

			if recorder.Code != http.StatusInternalServerError {
				t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
			}
			var response map[string]any
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if response["error"] != "internal_error" || response["message"] != "onboarding status unavailable" ||
				response["requestId"] != "onboarding-failure-request" || len(response) != 3 {
				t.Fatalf("response = %#v", response)
			}
			logged := logs.String()
			for _, want := range []string{test.operation, "error_category=store", "run_id=onboarding-failure-request"} {
				if !strings.Contains(logged, want) {
					t.Fatalf("log %q missing %q", logged, want)
				}
			}
			for _, secret := range []string{"did:plc:alice", "did:plc:secret", "database password", "2026-08-31T12:30:00Z"} {
				if strings.Contains(logged, secret) {
					t.Fatalf("log leaked %q: %s", secret, logged)
				}
			}
		})
	}
}
