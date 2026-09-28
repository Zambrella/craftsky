package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/ctxkeys"
)

func TestModeratorAuthenticationDerivesTrustedContext(t *testing.T) {
	handler := ModeratorAuthentication("correct horse battery staple 1234", "operator-1", "retool", nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		moderator, ok := ctxkeys.GetModerator(r.Context())
		if !ok || moderator.ActorID != "operator-1" || moderator.SourceSystem != "retool" {
			t.Fatalf("moderator context = %+v, %v", moderator, ok)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/v1/admin/moderation/cases", nil)
	request.Header.Set("Authorization", "Bearer correct horse battery staple 1234")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}

	for _, authorization := range []string{"", "Bearer wrong", "correct horse battery staple 1234"} {
		request := httptest.NewRequest(http.MethodGet, "/v1/admin/moderation/cases", nil)
		request.Header.Set("Authorization", authorization)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized || !strings.Contains(response.Body.String(), `"error":"moderator_authentication_failed"`) {
			t.Fatalf("authorization %q: status=%d body=%s", authorization, response.Code, response.Body.String())
		}
	}
}

func TestModeratorAuthenticationObservesSuccessAndFailureWithoutCredentials(t *testing.T) {
	observer := &moderatorAuthObserver{}
	handler := ModeratorAuthentication("SENTINEL_ADMIN_CREDENTIAL", "operator-1", "admin-api", nil, observer)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	valid := httptest.NewRequest(http.MethodGet, "/v1/admin/moderation/cases", nil)
	valid.Header.Set("Authorization", "Bearer SENTINEL_ADMIN_CREDENTIAL")
	handler.ServeHTTP(httptest.NewRecorder(), valid)
	invalid := httptest.NewRequest(http.MethodGet, "/v1/admin/moderation/cases", nil)
	invalid.Header.Set("Authorization", "Bearer SENTINEL_WRONG_CREDENTIAL")
	handler.ServeHTTP(httptest.NewRecorder(), invalid)

	if observer.successes != 1 || observer.failures != 1 {
		t.Fatalf("auth observations = successes %d failures %d, want one each", observer.successes, observer.failures)
	}
}

type moderatorAuthObserver struct {
	successes int
	failures  int
}

func (o *moderatorAuthObserver) ObserveModeratorAuthSuccess(time.Time) { o.successes++ }
func (o *moderatorAuthObserver) ObserveModeratorAuthFailure(time.Time) bool {
	o.failures++
	return false
}

func TestModeratorAuthenticationDoesNotTrustMemberOrDevelopmentCredentials(t *testing.T) {
	called := false
	handler := ModeratorAuthentication("admin-secret", "operator-1", "admin-api", nil)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))

	for _, test := range []struct {
		name          string
		authorization string
		devToken      string
	}{
		{name: "member bearer", authorization: "Bearer member-session-secret"},
		{name: "development moderation token", devToken: "development-secret"},
		{name: "revoked admin bearer", authorization: "Bearer old-admin-secret"},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/v1/admin/moderation/cases", nil)
			request.Header.Set("Authorization", test.authorization)
			request.Header.Set("X-Craftsky-Dev-Moderation-Token", test.devToken)
			request = request.WithContext(ctxkeys.WithDID(request.Context(), syntax.DID("did:plc:member-sensitive")))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusUnauthorized || !strings.Contains(response.Body.String(), `"error":"moderator_authentication_failed"`) {
				t.Fatalf("status/body = %d/%s", response.Code, response.Body.String())
			}
			for _, sensitive := range []string{"member-session-secret", "development-secret", "old-admin-secret", "did:plc:member-sensitive"} {
				if strings.Contains(response.Body.String(), sensitive) {
					t.Fatalf("response leaked %q: %s", sensitive, response.Body.String())
				}
			}
		})
	}
	if called {
		t.Fatal("protected handler was called")
	}
}
