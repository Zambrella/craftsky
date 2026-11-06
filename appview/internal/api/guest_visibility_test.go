package api_test

import (
	"context"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"net/http"
	"net/http/httptest"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/pdscommands"
	"strings"
	"testing"
)

type guestVisibilitySettingsFake struct {
	owner  syntax.DID
	calls  int
	update pdscommands.GuestVisibilityPutRequest
}

func (f *guestVisibilitySettingsFake) ReadGuestVisibility(_ context.Context, owner syntax.DID, _ int64, _ string) (pdscommands.GuestVisibilitySetting, error) {
	f.owner = owner
	f.calls++
	return pdscommands.GuestVisibilitySetting{HideFromLoggedOut: true, ExpectedCID: "profile-cid"}, nil
}
func (f *guestVisibilitySettingsFake) PutGuestVisibility(_ context.Context, r pdscommands.GuestVisibilityPutRequest) (pdscommands.CommandResult, error) {
	f.owner = r.Owner
	f.calls++
	f.update = r
	return pdscommands.CommandResult{TerminalResult: pdscommands.TerminalResult{State: pdscommands.CommandAccepted, HTTPStatus: 200, ResponseBody: []byte(`{"hideFromLoggedOut":false,"expectedCid":"new-cid"}`)}}, nil
}

// IT-007 / FR-008 / AC-008, AC-014.
func TestGuestVisibilitySettingsOwnerContract(t *testing.T) {
	for _, tc := range []struct {
		name, method, body string
		authenticated      bool
		status, calls      int
	}{
		{"guest cannot read", http.MethodGet, "", false, 401, 0},
		{"owner reads", http.MethodGet, "", true, 200, 1},
		{"owner saves", http.MethodPut, `{"hideFromLoggedOut":false,"expectedCid":"profile-cid"}`, true, 200, 1},
		{"owner cannot select another account", http.MethodPut, `{"hideFromLoggedOut":true,"expectedCid":"profile-cid","owner":"did:plc:other"}`, true, 400, 0},
		{"required boolean", http.MethodPut, `{"expectedCid":"profile-cid"}`, true, 400, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := &guestVisibilitySettingsFake{}
			request := httptest.NewRequest(tc.method, "/v1/preferences/logged-out-visibility", strings.NewReader(tc.body))
			request.Header.Set("Idempotency-Key", "018f4d5c-7a61-7d40-a1a2-777777777777")
			if tc.authenticated {
				ctx := middleware.WithDID(request.Context(), syntax.DID("did:plc:owner"))
				ctx = middleware.WithOwnerGeneration(ctx, 1)
				ctx = middleware.WithOAuthSessionID(ctx, "owner-session")
				request = request.WithContext(ctx)
			}
			response := httptest.NewRecorder()
			api.LoggedOutVisibilityHandler(fake).ServeHTTP(response, request)
			if response.Code != tc.status || fake.calls != tc.calls {
				t.Fatalf("status=%d calls=%d body=%s", response.Code, fake.calls, response.Body.String())
			}
			if fake.calls > 0 && fake.owner != "did:plc:owner" {
				t.Fatal("owner did not come from auth")
			}
			if tc.calls > 0 && tc.method == http.MethodPut && (fake.update.SessionID != "owner-session" || fake.update.OwnerGeneration != 1 || fake.update.HideFromLoggedOut) {
				t.Fatalf("wrong authenticated write: %+v", fake.update)
			}
			if response.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("owner settings response may be cached")
			}
		})
	}
}
