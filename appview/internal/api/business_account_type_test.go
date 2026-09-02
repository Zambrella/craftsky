package api_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/business"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/testdb"
)

type businessMemberChecker struct{}

func (businessMemberChecker) IsCurrentMember(context.Context, syntax.DID) (bool, error) {
	return true, nil
}

type businessLifecycleReader map[syntax.DID]ownerlifecycle.Lifecycle

func (r businessLifecycleReader) Get(_ context.Context, did syntax.DID) (ownerlifecycle.Lifecycle, error) {
	return r[did], nil
}

func TestBusinessAccountTypeMutationAuthenticationAndOwnership(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000061_business_account_types.up.sql")
	if err != nil {
		t.Fatalf("read account type migration: %v", err)
	}
	pool := testdb.WithSchema(t, `
		CREATE TABLE owner_lifecycles (
			owner_did TEXT PRIMARY KEY, state TEXT NOT NULL, generation BIGINT NOT NULL,
			auth_epoch BIGINT NOT NULL, transition_reason TEXT NOT NULL,
			transitioned_at TIMESTAMPTZ NOT NULL, terminal_at TIMESTAMPTZ,
			purge_completed_at TIMESTAMPTZ, created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		);
	`+string(migration))
	ctx := context.Background()
	alice := syntax.DID("did:plc:alice")
	bob := syntax.DID("did:plc:bob")
	departed := syntax.DID("did:plc:departed")
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at)
		VALUES ($1,'active',1,1,'test',now(),now(),now()),
		       ($2,'active',1,1,'test',now(),now(),now()),
		       ($3,'departed',1,1,'test',now(),now(),now())
	`, alice, bob, departed); err != nil {
		t.Fatalf("seed owner lifecycles: %v", err)
	}
	store := business.NewStore(pool)
	if err := store.PutAccountType(middleware.WithOwnerGeneration(ctx, 1), bob, business.AccountTypeBusiness); err != nil {
		t.Fatalf("seed Bob account type: %v", err)
	}
	lifecycles := businessLifecycleReader{
		alice:    {Owner: alice, State: ownerlifecycle.StateActive, Generation: 1},
		bob:      {Owner: bob, State: ownerlifecycle.StateActive, Generation: 1},
		departed: {Owner: departed, State: ownerlifecycle.StateDeparted, Generation: 1},
	}

	current := businessAccountTypeHandler(store, alice, lifecycles)
	response := serveBusinessAccountType(current, `{"accountType":"business"}`, true)
	if response.Code != http.StatusOK {
		t.Fatalf("current member status = %d, body=%s", response.Code, response.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body["accountType"] != "business" {
		t.Fatalf("current member body = %s, error=%v", response.Body.String(), err)
	}
	if got, err := store.ReadAccountType(ctx, alice); err != nil || got != business.AccountTypeBusiness {
		t.Fatalf("Alice account type = %q, error=%v", got, err)
	}

	response = serveBusinessAccountType(current, `{"accountType":"regular","ownerDid":"did:plc:bob"}`, true)
	assertBusinessAccountTypeError(t, response, http.StatusBadRequest, "unexpected_field")
	if got, err := store.ReadAccountType(ctx, bob); err != nil || got != business.AccountTypeBusiness {
		t.Fatalf("Bob account type changed = %q, error=%v", got, err)
	}

	response = serveBusinessAccountType(current, `{"accountType":"regular"}`, false)
	assertBusinessAccountTypeError(t, response, http.StatusUnauthorized, "unauthorized")

	response = serveBusinessAccountType(businessAccountTypeHandler(store, departed, lifecycles), `{"accountType":"business"}`, true)
	assertBusinessAccountTypeError(t, response, http.StatusNotFound, "profile_not_found")
	if got, err := store.ReadAccountType(ctx, departed); err != nil || got != business.AccountTypeRegular {
		t.Fatalf("departed account type = %q, error=%v", got, err)
	}

	response = serveBusinessAccountType(current, `{"accountType":"pro"}`, true)
	assertBusinessAccountTypeError(t, response, http.StatusUnprocessableEntity, "validation_failed")
}

func businessAccountTypeHandler(store *business.Store, did syntax.DID, lifecycles businessLifecycleReader) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := api.PutBusinessAccountTypeHandler(store)
	handler = middleware.CurrentMember(businessMemberChecker{}, logger, lifecycles)(handler)
	return middleware.Authenticated(&auth.MockAuthService{DefaultDID: did}, logger, middleware.DevAuthPolicy{Mode: middleware.DevAuthDisabled})(handler)
}

func serveBusinessAccountType(handler http.Handler, body string, authenticated bool) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPut, "/v1/profiles/me/account-type", strings.NewReader(body))
	if authenticated {
		req.Header.Set("Authorization", "Bearer test-session")
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	return response
}

func assertBusinessAccountTypeError(t *testing.T, response *httptest.ResponseRecorder, wantStatus int, wantCode string) {
	t.Helper()
	if response.Code != wantStatus {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, wantStatus, response.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body["error"] != wantCode {
		t.Fatalf("error body = %s, decode error=%v", response.Body.String(), err)
	}
}
