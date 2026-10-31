package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"social.craftsky/appview/internal/auth"
	"testing"
)

type chatActivityAuth struct{ normal, automatic int }

func (a *chatActivityAuth) Authenticate(context.Context, string) (auth.AuthInfo, error) {
	a.normal++
	return auth.AuthInfo{DID: "did:plc:chat-reader", SessionID: "parent"}, nil
}
func (a *chatActivityAuth) AuthenticateWithoutActivity(context.Context, string) (auth.AuthInfo, error) {
	a.automatic++
	return auth.AuthInfo{DID: "did:plc:chat-reader", SessionID: "parent"}, nil
}
func (a *chatActivityAuth) AuthenticateRecovery(ctx context.Context, token string) (auth.AuthInfo, error) {
	return a.Authenticate(ctx, token)
}

// IT-005 / FR-016: route policy, rather than a caller header, selects the
// identical credential validation path that does not extend idle expiry.
func TestChatStatusUsesAutomaticAuthentication(t *testing.T) {
	a := &chatActivityAuth{}
	deps := testDeps()
	m := buildV1Middleware(middlewareDependencies{Config: deps.Config, Logger: deps.Logger, AuthService: a, Suspension: deps.SuspensionReader, Eligibility: deps.EligibilityReader}, nil)
	m.member = func(h http.Handler) http.Handler { return h }
	m.deviceID = func(h http.Handler) http.Handler { return h }
	called := false
	h := m.wrap(mustPolicy("GET", "/v1/chat/status"), http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	r := httptest.NewRequest("GET", "/v1/chat/status", nil)
	r.Header.Set("Authorization", "Bearer synthetic")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if !called || a.normal != 0 || a.automatic != 1 {
		t.Fatalf("automatic=%d normal=%d reached=%t status=%d", a.automatic, a.normal, called, w.Code)
	}
}

// IT-019 / NFR-003: private chat routes enter the ordinary consumer
// authentication/device boundary and use the standard error envelope.
func TestChatStatusRouteRequiresAuthenticationAndDevice(t *testing.T) {
	assertPrivateChatRouteCredentials(t, "GET", "/v1/chat/status")
	assertPrivateChatRouteCredentials(t, "GET", "/v1/chat/conversations/direct-1/details")
}

func TestChatAuthorizationRouteRequiresAuthenticationAndDevice(t *testing.T) {
	assertPrivateChatRouteCredentials(t, "POST", "/v1/chat/authorization")
}

func TestChatSendRouteRequiresAuthenticationAndDevice(t *testing.T) {
	assertPrivateChatRouteCredentials(t, "POST", "/v1/chat/conversations/direct-1/messages")
}

func TestChatPreferenceRoutesRequireAuthenticationAndDevice(t *testing.T) {
	for _, route := range []struct{ method, path string }{{"GET", "/v1/chat/incoming-policy"}, {"GET", "/v1/chat/push-preferences"}, {"PUT", "/v1/chat/push-preferences"}} {
		t.Run(route.method+route.path, func(t *testing.T) { assertPrivateChatRouteCredentials(t, route.method, route.path) })
	}
}

func TestChatSyncRouteRequiresAutomaticAuthentication(t *testing.T) {
	assertPrivateChatRouteCredentials(t, "POST", "/v1/chat/sync")
	if !mustPolicy("POST", "/v1/chat/sync").NoActivity {
		t.Fatal("automatic heartbeat records user activity")
	}
}

func TestChatNotificationOpenRouteRequiresAuthenticationAndDevice(t *testing.T) {
	assertPrivateChatRouteCredentials(t, "GET", "/v1/chat/notification-opens/00000000-0000-4000-8000-000000000123")
}

// IT-019: all implemented consumer operations share the private normal
// credential/device boundary, including reads and protected receipts.
func TestChatConversationRoutesRequireAuthenticationAndDevice(t *testing.T) {
	for _, route := range []struct{ method, path string }{
		{"DELETE", "/v1/chat/conversations/direct-1/messages/own-1"},
		{"PUT", "/v1/chat/connection"}, {"GET", "/v1/chat/conversations"}, {"GET", "/v1/chat/availability"}, {"POST", "/v1/chat/conversations"}, {"GET", "/v1/chat/conversations/direct-1"}, {"GET", "/v1/chat/conversations/direct-1/details"}, {"GET", "/v1/chat/conversations/direct-1/messages"}, {"GET", "/v1/chat/operations/operation-1"}, {"PUT", "/v1/chat/conversations/direct-1/read"}, {"POST", "/v1/chat/conversations/direct-1/accept"}, {"PUT", "/v1/chat/conversations/direct-1/mute"}, {"DELETE", "/v1/chat/conversations/direct-1/mute"}, {"POST", "/v1/chat/conversations/direct-1/leave"}, {"POST", "/v1/chat/reports"}, {"GET", "/v1/chat/unread-counts"}, {"PUT", "/v1/chat/incoming-policy"},
	} {
		t.Run(route.method+" "+route.path, func(t *testing.T) { assertPrivateChatRouteCredentials(t, route.method, route.path) })
	}
}

func assertPrivateChatRouteCredentials(t *testing.T, method, path string) {
	t.Helper()
	mux := http.NewServeMux()
	AddRoutes(t.Context(), mux, testDeps())
	for _, test := range []struct {
		name, bearer string
		want         int
	}{
		{"missing bearer", "", http.StatusUnauthorized},
		{"missing device", "Bearer synthetic", http.StatusBadRequest},
	} {
		t.Run(test.name, func(t *testing.T) {
			r := httptest.NewRequest(method, path, nil)
			r.Header.Set("Authorization", test.bearer)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, r)
			if w.Code != test.want {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
			assertErrorEnvelope(t, w.Body.Bytes())
		})
	}
}
