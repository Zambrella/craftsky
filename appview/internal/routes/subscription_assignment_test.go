package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSubscriptionAssignmentRoutesAreRemoved(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())
	for _, method := range []string{http.MethodPut, http.MethodDelete} {
		request := httptest.NewRequest(method, "/v1/billing/licenses/40000000-0000-4000-8000-000000000031/assignment", strings.NewReader(`{"targetDid":"did:plc:other"}`))
		request.Header.Set("Authorization", "Bearer test-token")
		request.Header.Set("X-Craftsky-Device-Id", "device")
		request.Header.Set("X-Dev-DID", "did:plc:owner")
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		if response.Code != http.StatusNotFound {
			t.Fatalf("%s assignment=%d", method, response.Code)
		}
	}
}
