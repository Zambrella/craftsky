package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/ctxkeys"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/testdb"
)

type fakeProfilePinStore struct {
	state api.ProfilePinState
	err   error

	owner      syntax.DID
	targetDID  syntax.DID
	targetRkey syntax.RecordKey
	targetURI  syntax.ATURI
}

func (f *fakeProfilePinStore) Read(context.Context, syntax.DID) (api.ProfilePinState, error) {
	return f.state, f.err
}

func (f *fakeProfilePinStore) Pin(_ context.Context, owner, targetDID syntax.DID, targetRkey syntax.RecordKey) (api.ProfilePinMutationResult, error) {
	f.owner, f.targetDID, f.targetRkey = owner, targetDID, targetRkey
	return api.ProfilePinMutationResult{State: f.state}, f.err
}

func (f *fakeProfilePinStore) Unpin(_ context.Context, owner syntax.DID, targetURI syntax.ATURI) (api.ProfilePinMutationResult, error) {
	f.owner, f.targetURI = owner, targetURI
	return api.ProfilePinMutationResult{State: f.state}, f.err
}

func TestProfilePinHandlersReturnAuthoritativeBodylessContracts(t *testing.T) {
	standard := syntax.ATURI("at://did:plc:alice/social.craftsky.feed.post/standard-a")
	project := syntax.ATURI("at://did:plc:alice/social.craftsky.feed.post/project-a")

	t.Run("read both slots", func(t *testing.T) {
		store := &fakeProfilePinStore{state: api.ProfilePinState{StandardPostURI: &standard, ProjectPostURI: &project}}
		response := httptest.NewRecorder()
		api.GetProfilePinsHandler(store).ServeHTTP(response, profilePinRequest(http.MethodGet, "/v1/profiles/me/pins"))
		assertProfilePinSuccess(t, response, standard.String(), project.String())
	})

	t.Run("pin returns nullable authoritative state", func(t *testing.T) {
		store := &fakeProfilePinStore{state: api.ProfilePinState{StandardPostURI: &standard}}
		request := profilePinRequest(http.MethodPut, "/v1/posts/did:plc:alice/standard-a/pin")
		request.SetPathValue("did", "did:plc:alice")
		request.SetPathValue("rkey", "standard-a")
		response := httptest.NewRecorder()
		api.PinProfilePostHandler(store).ServeHTTP(response, request)
		assertProfilePinSuccess(t, response, standard.String(), "")
		if store.owner != "did:plc:alice" || store.targetDID != "did:plc:alice" || store.targetRkey != "standard-a" {
			t.Fatalf("pin call = owner %q target %q/%q", store.owner, store.targetDID, store.targetRkey)
		}
	})

	t.Run("stale unpin returns newer state", func(t *testing.T) {
		newer := syntax.ATURI("at://did:plc:alice/social.craftsky.feed.post/standard-b")
		store := &fakeProfilePinStore{state: api.ProfilePinState{StandardPostURI: &newer}}
		request := profilePinRequest(http.MethodDelete, "/v1/posts/did:plc:alice/standard-a/pin")
		request.SetPathValue("did", "did:plc:alice")
		request.SetPathValue("rkey", "standard-a")
		response := httptest.NewRecorder()
		api.UnpinProfilePostHandler(store).ServeHTTP(response, request)
		assertProfilePinSuccess(t, response, newer.String(), "")
		if store.owner != "did:plc:alice" || store.targetURI != standard {
			t.Fatalf("unpin call = owner %q uri %q", store.owner, store.targetURI)
		}
	})

	for _, test := range []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "forbidden", err: api.ErrProfilePinForbidden, status: http.StatusForbidden, code: "forbidden"},
		{name: "not found", err: api.ErrProfilePinTargetNotFound, status: http.StatusNotFound, code: "post_not_found"},
		{name: "not allowed", err: api.ErrProfilePinNotAllowed, status: http.StatusUnprocessableEntity, code: "pin_not_allowed"},
		{name: "internal", err: errors.New("database unavailable"), status: http.StatusInternalServerError, code: "internal_error"},
	} {
		t.Run("pin error "+test.name, func(t *testing.T) {
			store := &fakeProfilePinStore{err: test.err}
			request := profilePinRequest(http.MethodPut, "/v1/posts/did:plc:alice/standard-a/pin")
			request.SetPathValue("did", "did:plc:alice")
			request.SetPathValue("rkey", "standard-a")
			response := httptest.NewRecorder()
			api.PinProfilePostHandler(store).ServeHTTP(response, request)
			assertProfilePinError(t, response, test.status, test.code)
		})
	}

	t.Run("invalid identifiers", func(t *testing.T) {
		request := profilePinRequest(http.MethodPut, "/v1/posts/not-a-did/bad%20key/pin")
		request.SetPathValue("did", "not-a-did")
		request.SetPathValue("rkey", "bad key")
		response := httptest.NewRecorder()
		api.PinProfilePostHandler(&fakeProfilePinStore{}).ServeHTTP(response, request)
		assertProfilePinError(t, response, http.StatusBadRequest, "invalid_identifier")
	})
}

func profilePinRequest(method, target string) *http.Request {
	request := httptest.NewRequest(method, target, strings.NewReader(""))
	ctx := middleware.WithDID(request.Context(), syntax.DID("did:plc:alice"))
	ctx = ctxkeys.WithRunID(ctx, "profile-pin-request")
	return request.WithContext(ctx)
}

func assertProfilePinSuccess(t *testing.T, response *httptest.ResponseRecorder, standardURI, projectURI string) {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", response.Code, response.Body.String())
	}
	if response.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("content type = %q", response.Header().Get("Content-Type"))
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body) != 2 {
		t.Fatalf("response keys = %v, want only two slot keys", body)
	}
	if got, _ := body["standardPostUri"].(string); got != standardURI {
		if standardURI != "" || body["standardPostUri"] != nil {
			t.Fatalf("standardPostUri = %#v, want %q/null", body["standardPostUri"], standardURI)
		}
	}
	if got, _ := body["projectPostUri"].(string); got != projectURI {
		if projectURI != "" || body["projectPostUri"] != nil {
			t.Fatalf("projectPostUri = %#v, want %q/null", body["projectPostUri"], projectURI)
		}
	}
}

func assertProfilePinError(t *testing.T, response *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if response.Code != status {
		t.Fatalf("status = %d, want %d body=%s", response.Code, status, response.Body.String())
	}
	var body envelope.Error
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if body.Error != code || body.Message == "" || body.RequestID != "profile-pin-request" {
		t.Fatalf("error body = %+v, want code %q and request ID", body, code)
	}
}

func TestProfilePinStoreEnforcesNewTargetPolicyButAllowsRetainedUnpin(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000035_profile_pins.up.sql")
	if err != nil {
		t.Fatalf("read profile pin migration: %v", err)
	}
	pool := testdb.WithSchema(t, profilePinStoreTestDDL+string(migration))
	ctx := context.Background()
	owner := syntax.DID("did:plc:alice")
	store := api.NewProfilePinStore(pool)

	for _, test := range []struct {
		name      string
		targetDID syntax.DID
		rkey      syntax.RecordKey
		wantErr   error
	}{
		{name: "another author", targetDID: syntax.DID("did:plc:bob"), rkey: syntax.RecordKey("other"), wantErr: api.ErrProfilePinForbidden},
		{name: "missing", targetDID: owner, rkey: syntax.RecordKey("missing"), wantErr: api.ErrProfilePinTargetNotFound},
		{name: "moderation hidden", targetDID: owner, rkey: syntax.RecordKey("hidden"), wantErr: api.ErrProfilePinTargetNotFound},
		{name: "comment", targetDID: owner, rkey: syntax.RecordKey("comment"), wantErr: api.ErrProfilePinNotAllowed},
		{name: "partial reply", targetDID: owner, rkey: syntax.RecordKey("partial-reply"), wantErr: api.ErrProfilePinNotAllowed},
		{name: "project quote", targetDID: owner, rkey: syntax.RecordKey("project-quote"), wantErr: api.ErrProfilePinNotAllowed},
		{name: "project missing materialization", targetDID: owner, rkey: syntax.RecordKey("project-missing"), wantErr: api.ErrProfilePinNotAllowed},
		{name: "standard with project materialization", targetDID: owner, rkey: syntax.RecordKey("standard-materialized"), wantErr: api.ErrProfilePinNotAllowed},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := store.Pin(ctx, owner, test.targetDID, test.rkey)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Pin() error = %v, want %v", err, test.wantErr)
			}
			state, readErr := store.Read(ctx, owner)
			if readErr != nil {
				t.Fatalf("read state after rejected pin: %v", readErr)
			}
			assertProfilePinState(t, state, "", "")
		})
	}
	if _, err := store.Pin(
		ctx,
		syntax.DID("did:plc:dave"),
		syntax.DID("did:plc:dave"),
		syntax.RecordKey("nonmember"),
	); !errors.Is(err, api.ErrProfilePinTargetNotFound) {
		t.Fatalf("non-member Pin() error = %v, want %v", err, api.ErrProfilePinTargetNotFound)
	}

	for _, test := range []struct {
		name string
		rkey syntax.RecordKey
		slot api.ProfilePinSlot
	}{
		{name: "standard", rkey: syntax.RecordKey("standard-a"), slot: api.ProfilePinSlotStandard},
		{name: "quote", rkey: syntax.RecordKey("quote"), slot: api.ProfilePinSlotStandard},
		{name: "project", rkey: syntax.RecordKey("project-a"), slot: api.ProfilePinSlotProject},
	} {
		t.Run("allows "+test.name, func(t *testing.T) {
			mutation, err := store.Pin(ctx, owner, owner, test.rkey)
			if err != nil {
				t.Fatalf("Pin(): %v", err)
			}
			if mutation.Slot != test.slot {
				t.Fatalf("slot = %q, want %q", mutation.Slot, test.slot)
			}
			if _, err := store.Unpin(ctx, owner, syntax.ATURI("at://did:plc:alice/social.craftsky.feed.post/"+test.rkey.String())); err != nil {
				t.Fatalf("reset pin: %v", err)
			}
		})
	}

	if _, err := store.Pin(ctx, owner, owner, syntax.RecordKey("standard-a")); err != nil {
		t.Fatalf("seed retained pin: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO moderation_outputs (
			id, source_did, subject_type, subject_did, subject_uri, value, action
		) VALUES (
			'retained-hidden', 'did:plc:moderator', 'post', 'did:plc:alice',
			'at://did:plc:alice/social.craftsky.feed.post/standard-a', 'takedown', 'apply'
		)
	`); err != nil {
		t.Fatalf("hide retained target: %v", err)
	}
	removed, err := store.Unpin(ctx, owner, syntax.ATURI("at://did:plc:alice/social.craftsky.feed.post/standard-a"))
	if err != nil {
		t.Fatalf("unpin retained hidden target: %v", err)
	}
	if removed.Operation != api.ProfilePinOperationUnpin {
		t.Fatalf("hidden-target unpin operation = %q", removed.Operation)
	}
	assertProfilePinState(t, removed.State, "", "")
}
