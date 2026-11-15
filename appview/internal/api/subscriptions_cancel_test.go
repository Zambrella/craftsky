package api_test

import (
	"context"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"net/http"
	"net/http/httptest"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/middleware"
	"strings"
	"testing"
	"time"
)

type cancellationStore struct {
	owner syntax.DID
	calls int
}

func (s *cancellationStore) CancelPendingWebCheckout(_ context.Context, owner syntax.DID, _ time.Time) error {
	s.owner = owner
	s.calls++
	return nil
}
func TestCancelCheckoutUsesOnlyAuthenticatedScope(t *testing.T) {
	for _, body := range []string{"", `{"sessionId":"other-session"}`} {
		t.Run(body, func(t *testing.T) {
			store := &cancellationStore{}
			request := httptest.NewRequest(http.MethodPost, "/v1/billing/web-checkout/cancel", strings.NewReader(body))
			request = request.WithContext(middleware.WithDID(request.Context(), syntax.DID("did:plc:owner")))
			response := httptest.NewRecorder()
			api.CancelPendingWebCheckoutHandler(store, time.Now).ServeHTTP(response, request)
			if body == "" {
				if response.Code != 200 || store.calls != 1 || store.owner != "did:plc:owner" {
					t.Fatal("wrong cancellation scope")
				}
			} else if response.Code != 400 || store.calls != 0 {
				t.Fatal("accepted caller-chosen session")
			}
		})
	}
}
