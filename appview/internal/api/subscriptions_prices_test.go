package api_test

import (
	"context"
	"errors"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"net/http"
	"net/http/httptest"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/subscriptions"
	"strings"
	"testing"
)

type priceStore struct {
	calls int
	err   error
}

func (s *priceStore) WebPrices(context.Context) ([]subscriptions.WebPrice, error) {
	s.calls++
	return []subscriptions.WebPrice{{Tier: subscriptions.TierPlus, Amount: 249, Currency: "gbp", TaxBehavior: "inclusive"}}, s.err
}
func TestWebPricesAreAuthenticatedPublicCatalogAndFailIndependently(t *testing.T) {
	for _, tc := range []struct {
		name          string
		authenticated bool
		err           error
		status        int
	}{
		{"catalog", true, nil, 200}, {"no account", false, nil, 500}, {"provider unavailable", true, errors.New("private-provider-canary"), 503},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &priceStore{err: tc.err}
			request := httptest.NewRequest(http.MethodGet, "/v1/billing/web-prices", nil)
			if tc.authenticated {
				request = request.WithContext(middleware.WithDID(request.Context(), syntax.DID("did:plc:member")))
			}
			response := httptest.NewRecorder()
			api.GetWebSubscriptionPricesHandler(store).ServeHTTP(response, request)
			if response.Code != tc.status {
				t.Fatalf("status=%d", response.Code)
			}
			if !tc.authenticated && store.calls != 0 {
				t.Fatal("queried catalog without account scope")
			}
			if strings.Contains(response.Body.String(), "private-provider-canary") {
				t.Fatal("provider cause exposed")
			}
			if tc.status == 200 && !strings.Contains(response.Body.String(), `"amount":249`) {
				t.Fatalf("price absent: %s", response.Body.String())
			}
		})
	}
}
