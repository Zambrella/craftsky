package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"social.craftsky/appview/internal/auth"
)

type chatActivityAuth struct{ ordinary, automatic int }

func (service *chatActivityAuth) Authenticate(context.Context, string) (auth.AuthInfo, error) {
	service.ordinary++
	return auth.AuthInfo{DID: "did:plc:automatic-owner", SessionID: "parent"}, nil
}

func (service *chatActivityAuth) AuthenticateWithoutActivity(_ context.Context, token string) (auth.AuthInfo, error) {
	service.automatic++
	if token != "valid" {
		return auth.AuthInfo{}, auth.ErrAuthTokenInvalid
	}
	return auth.AuthInfo{DID: "did:plc:automatic-owner", SessionID: "parent"}, nil
}

// IT-005 / FR-016, NFR-005 / AC-014, AC-022.
func TestChatAutomaticMiddlewareChoosesNoActivityAuthentication(t *testing.T) {
	service := &chatActivityAuth{}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if did, ok := GetDID(r.Context()); !ok || did != "did:plc:automatic-owner" {
			t.Fatal("automatic authentication lost owner")
		}
		w.WriteHeader(http.StatusNoContent)
	})
	handler := AuthenticatedWithoutActivity(service, discardLogger(), DevAuthPolicy{Mode: DevAuthDisabled})(next)
	for _, test := range []struct {
		token string
		want  int
	}{{"valid", http.StatusNoContent}, {"invalid", http.StatusUnauthorized}, {"", http.StatusUnauthorized}} {
		request := httptest.NewRequest(http.MethodGet, "/v1/chat/status", nil)
		if test.token != "" {
			request.Header.Set("Authorization", "Bearer "+test.token)
		}
		// No client-supplied activity header can turn an automatic route into
		// a session-extending authentication call.
		request.Header.Set("X-Craftsky-Session-Activity", "active")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != test.want {
			t.Fatalf("status=%d want=%d", response.Code, test.want)
		}
	}
	if service.ordinary != 0 || service.automatic != 2 {
		t.Fatalf("ordinary/automatic calls=%d/%d", service.ordinary, service.automatic)
	}
}
