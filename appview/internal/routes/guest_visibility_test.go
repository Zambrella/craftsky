package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// IT-007 / FR-008 / AC-008, AC-014: both actual registrations remain private.
func TestGuestVisibilityRoutesRequireAuthentication(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())
	for _, method := range []string{http.MethodGet, http.MethodPut} {
		body := ""
		if method == http.MethodPut {
			body = `{"hideFromLoggedOut":true,"expectedCid":"profile-cid"}`
		}
		request := httptest.NewRequest(method, "/v1/preferences/logged-out-visibility", strings.NewReader(body))
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("%s status=%d body=%s", method, response.Code, response.Body.String())
		}
	}
}
