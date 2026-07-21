package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/instagram"
	"social.craftsky/appview/internal/middleware"
)

func TestInstagramAccountGetReportsLocalStateIndependentOfIntegrationAvailability(t *testing.T) {
	t.Parallel()

	alice := syntax.DID("did:plc:instagram-account-alice")
	verifiedAt := time.Date(2026, 7, 19, 11, 12, 13, 0, time.FixedZone("test", 60*60))
	store := &stubInstagramAccountStore{account: &instagram.AccountView{
		State:                instagram.LinkMembershipInactive,
		Username:             "private.synthetic.username",
		Discoverable:         false,
		ConflictPending:      true,
		ReactivationRequired: true,
		VerifiedAt:           verifiedAt,
	}}
	available := false
	handler := GetInstagramAccountHandler(store, func() bool { return available }, slog.Default())

	first := serveInstagramAccountRequest(t, handler, http.MethodGet, "/v1/migrations/instagram/account", "", alice)
	if first.Code != http.StatusOK {
		t.Fatalf("GET status = %d; body=%s", first.Code, first.Body.String())
	}
	firstBody := decodeInstagramAccountMap(t, first)
	assertInstagramAccountStatus(t, firstBody, false, map[string]any{
		"state":                "membershipInactive",
		"username":             "private.synthetic.username",
		"discoverable":         false,
		"conflictPending":      true,
		"reactivationRequired": true,
		"verifiedAt":           "2026-07-19T10:12:13Z",
	})

	available = true
	second := serveInstagramAccountRequest(t, handler, http.MethodGet, "/v1/migrations/instagram/account", "", alice)
	secondBody := decodeInstagramAccountMap(t, second)
	assertInstagramAccountStatus(t, secondBody, true, firstBody["account"])
	if len(store.getOwners) != 2 || store.getOwners[0] != alice || store.getOwners[1] != alice {
		t.Fatalf("GET owners = %v, want Alice twice", store.getOwners)
	}

	store.account = nil
	available = false
	empty := serveInstagramAccountRequest(t, handler, http.MethodGet, "/v1/migrations/instagram/account", "", alice)
	emptyBody := decodeInstagramAccountMap(t, empty)
	assertInstagramAccountStatus(t, emptyBody, false, nil)
}

func TestInstagramAccountPatchRequiresStrictExplicitSettings(t *testing.T) {
	t.Parallel()

	alice := syntax.DID("did:plc:instagram-settings-alice")
	store := &stubInstagramAccountStore{account: &instagram.AccountView{
		State:        instagram.LinkActive,
		Username:     "private.settings.username",
		Discoverable: false,
		VerifiedAt:   time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC),
	}}
	handler := PatchInstagramSettingsHandler(store, func() bool { return false }, slog.Default())

	tests := []struct {
		name string
		body string
	}{
		{name: "empty object", body: `{}`},
		{name: "unknown field", body: `{"unexpected":"private-input-canary"}`},
		{name: "multiple values", body: `{"discoverable":false}{"discoverable":true}`},
		{name: "reactivate without discovery choice", body: `{"reactivate":true}`},
		{name: "false reactivation", body: `{"reactivate":false,"discoverable":false}`},
		{name: "wrong type", body: `{"discoverable":"yes"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := len(store.updateCalls)
			rr := serveInstagramAccountRequest(t, handler, http.MethodPatch, "/v1/migrations/instagram/settings", tt.body, alice)
			assertInstagramAccountError(t, rr, http.StatusBadRequest, "invalid_request")
			if strings.Contains(rr.Body.String(), "private-input-canary") {
				t.Fatal("invalid request echoed a private input")
			}
			if len(store.updateCalls) != before {
				t.Fatalf("invalid request reached store: calls=%+v", store.updateCalls)
			}
		})
	}

	discovery := serveInstagramAccountRequest(t, handler, http.MethodPatch, "/v1/migrations/instagram/settings", `{"discoverable":false}`, alice)
	if discovery.Code != http.StatusOK {
		t.Fatalf("discovery PATCH status = %d; body=%s", discovery.Code, discovery.Body.String())
	}
	assertInstagramAccountStatus(t, decodeInstagramAccountMap(t, discovery), false, map[string]any{
		"state":                "active",
		"username":             "private.settings.username",
		"discoverable":         false,
		"conflictPending":      false,
		"reactivationRequired": false,
		"verifiedAt":           "2026-07-19T12:00:00Z",
	})
	if len(store.updateCalls) != 1 || store.updateCalls[0].owner != alice || store.updateCalls[0].patch.Discoverable == nil || *store.updateCalls[0].patch.Discoverable || store.updateCalls[0].patch.Reactivate != nil {
		t.Fatalf("discovery PATCH call = %+v", store.updateCalls)
	}

	reactivation := serveInstagramAccountRequest(t, handler, http.MethodPatch, "/v1/migrations/instagram/settings", `{"reactivate":true,"discoverable":true}`, alice)
	if reactivation.Code != http.StatusOK {
		t.Fatalf("reactivation PATCH status = %d; body=%s", reactivation.Code, reactivation.Body.String())
	}
	if len(store.updateCalls) != 2 || store.updateCalls[1].patch.Reactivate == nil || !*store.updateCalls[1].patch.Reactivate || store.updateCalls[1].patch.Discoverable == nil || !*store.updateCalls[1].patch.Discoverable {
		t.Fatalf("reactivation PATCH call = %+v", store.updateCalls)
	}
}

func TestInstagramAccountHandlersMapSafeErrorsWithoutPrivateDiagnostics(t *testing.T) {
	t.Parallel()

	alice := syntax.DID("did:plc:instagram-errors-alice")
	tests := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "invalid settings", err: instagram.ErrInvalidInstagramSettings, status: http.StatusBadRequest, code: "invalid_request"},
		{name: "missing link", err: instagram.ErrInstagramLinkNotFound, status: http.StatusNotFound, code: "instagram_link_not_found"},
		{name: "reactivation required", err: instagram.ErrInstagramReactivationRequired, status: http.StatusConflict, code: "instagram_reactivation_required"},
		{name: "link conflict", err: instagram.ErrInstagramLinkConflict, status: http.StatusConflict, code: "instagram_link_conflict"},
		{name: "internal", err: errors.New("private.username and 17841400000000123"), status: http.StatusInternalServerError, code: "internal_error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logs bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&logs, nil))
			store := &stubInstagramAccountStore{updateErr: fmt.Errorf("wrapped operation: %w", tt.err)}
			handler := PatchInstagramSettingsHandler(store, func() bool { return true }, logger)
			rr := serveInstagramAccountRequest(t, handler, http.MethodPatch, "/v1/migrations/instagram/settings", `{"discoverable":true}`, alice)
			assertInstagramAccountError(t, rr, tt.status, tt.code)
			combined := rr.Body.String() + logs.String()
			if strings.Contains(combined, "private.username") || strings.Contains(combined, "17841400000000123") {
				t.Fatalf("private identity leaked in diagnostics: %s", combined)
			}
		})
	}

	store := &stubInstagramAccountStore{getErr: errors.New("private.get.username 17841400000000456")}
	var logs bytes.Buffer
	get := GetInstagramAccountHandler(store, func() bool { return false }, slog.New(slog.NewTextHandler(&logs, nil)))
	rr := serveInstagramAccountRequest(t, get, http.MethodGet, "/v1/migrations/instagram/account", "", alice)
	assertInstagramAccountError(t, rr, http.StatusInternalServerError, "internal_error")
	if strings.Contains(rr.Body.String()+logs.String(), "private.get.username") || strings.Contains(rr.Body.String()+logs.String(), "17841400000000456") {
		t.Fatal("GET diagnostics exposed private Instagram identity")
	}
}

func TestInstagramAccountDeleteIsIdempotentAndOwnershipHiding(t *testing.T) {
	t.Parallel()

	alice := syntax.DID("did:plc:instagram-delete-alice")
	store := &stubInstagramAccountStore{}
	handler := DeleteInstagramAccountHandler(store, slog.Default())
	for range 2 {
		rr := serveInstagramAccountRequest(t, handler, http.MethodDelete, "/v1/migrations/instagram/account", "", alice)
		if rr.Code != http.StatusNoContent || rr.Body.Len() != 0 {
			t.Fatalf("DELETE status=%d body=%q, want empty 204", rr.Code, rr.Body.String())
		}
	}
	if len(store.revokeOwners) != 2 || store.revokeOwners[0] != alice || store.revokeOwners[1] != alice {
		t.Fatalf("revoke owners = %v, want Alice twice", store.revokeOwners)
	}

	missingDID := httptest.NewRecorder()
	handler.ServeHTTP(missingDID, httptest.NewRequest(http.MethodDelete, "/v1/migrations/instagram/account", nil))
	assertInstagramAccountError(t, missingDID, http.StatusInternalServerError, "missing_authenticated_did")
	if len(store.revokeOwners) != 2 {
		t.Fatal("missing authenticated DID reached account store")
	}

	var logs bytes.Buffer
	store.revokeErr = errors.New("private.delete.username 17841400000000789")
	handler = DeleteInstagramAccountHandler(store, slog.New(slog.NewTextHandler(&logs, nil)))
	failure := serveInstagramAccountRequest(t, handler, http.MethodDelete, "/v1/migrations/instagram/account", "", alice)
	assertInstagramAccountError(t, failure, http.StatusInternalServerError, "internal_error")
	if strings.Contains(failure.Body.String()+logs.String(), "private.delete.username") || strings.Contains(failure.Body.String()+logs.String(), "17841400000000789") {
		t.Fatal("DELETE diagnostics exposed private Instagram identity")
	}
}

func serveInstagramAccountRequest(t *testing.T, handler http.Handler, method, target, body string, did syntax.DID) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if did != "" {
		req = req.WithContext(middleware.WithDID(req.Context(), did))
	}
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

func decodeInstagramAccountMap(t *testing.T, rr *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, rr.Body.String())
	}
	return body
}

func assertInstagramAccountStatus(t *testing.T, body map[string]any, available bool, wantAccount any) {
	t.Helper()
	if len(body) != 2 || body["integrationAvailable"] != available {
		t.Fatalf("status response = %#v, want integrationAvailable=%t and exactly two fields", body, available)
	}
	wantJSON, err := json.Marshal(wantAccount)
	if err != nil {
		t.Fatal(err)
	}
	gotJSON, err := json.Marshal(body["account"])
	if err != nil {
		t.Fatal(err)
	}
	if string(gotJSON) != string(wantJSON) {
		t.Fatalf("account = %s, want %s", gotJSON, wantJSON)
	}
	if strings.Contains(string(gotJSON), "igsid") || strings.Contains(string(gotJSON), "178414") {
		t.Fatal("account response exposed an Instagram scoped ID")
	}
}

func assertInstagramAccountError(t *testing.T, rr *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if rr.Code != status {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, status, rr.Body.String())
	}
	var body envelope.Error
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error: %v; body=%s", err, rr.Body.String())
	}
	if body.Error != code {
		t.Fatalf("error code = %q, want %q; body=%s", body.Error, code, rr.Body.String())
	}
}

type stubInstagramAccountStore struct {
	account      *instagram.AccountView
	getErr       error
	updateErr    error
	revokeErr    error
	getOwners    []syntax.DID
	revokeOwners []syntax.DID
	updateCalls  []struct {
		owner syntax.DID
		patch instagram.AccountSettingsPatch
	}
}

func (s *stubInstagramAccountStore) GetAccount(_ context.Context, owner syntax.DID) (*instagram.AccountView, error) {
	s.getOwners = append(s.getOwners, owner)
	return s.account, s.getErr
}

func (s *stubInstagramAccountStore) UpdateSettings(_ context.Context, owner syntax.DID, patch instagram.AccountSettingsPatch) (*instagram.AccountView, error) {
	s.updateCalls = append(s.updateCalls, struct {
		owner syntax.DID
		patch instagram.AccountSettingsPatch
	}{owner: owner, patch: patch})
	return s.account, s.updateErr
}

func (s *stubInstagramAccountStore) RevokeAccount(_ context.Context, owner syntax.DID) error {
	s.revokeOwners = append(s.revokeOwners, owner)
	return s.revokeErr
}

var _ InstagramAccountStore = (*stubInstagramAccountStore)(nil)
