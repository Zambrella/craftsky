package api_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/subscriptions"
)

type paywallStore struct {
	owner syntax.DID
	tier  subscriptions.Tier
	err   error
}

func (s *paywallStore) PrepareWebPaywall(_ context.Context, owner syntax.DID, tier subscriptions.Tier, _ time.Time) (subscriptions.WebPaywall, error) {
	s.owner = owner
	s.tier = tier
	return subscriptions.WebPaywall{APIKey: "strp_sb_fixture", AppUserID: "private-owner", Offering: string(tier)}, s.err
}
func TestWebPaywallOwnerScopeAndStrictBody(t *testing.T) {
	for _, body := range []string{`{"tier":"plus"}`, `{"tier":"plus","appUserId":"other"}`, `{"tier":"free"}`, `{"tier":"plus","offering":"other"}`} {
		store := &paywallStore{}
		r := httptest.NewRequest(http.MethodPost, "/v1/billing/web-paywall", strings.NewReader(body))
		r = r.WithContext(middleware.WithDID(r.Context(), syntax.DID("did:plc:owner")))
		w := httptest.NewRecorder()
		api.PrepareWebPaywallHandler(store, time.Now).ServeHTTP(w, r)
		if body == `{"tier":"plus"}` {
			if w.Code != 200 || store.owner != "did:plc:owner" || store.tier != subscriptions.TierPlus || w.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("wrong paywall scope")
			}
		} else if w.Code != 400 || store.owner != "" {
			t.Fatal("accepted caller-selected identity/catalog")
		}
	}
}
func TestWebPaywallErrorsDoNotExposeProviderPayloads(t *testing.T) {
	for _, err := range []error{subscriptions.ErrWebCheckoutPending, subscriptions.ErrSubscriptionAlreadyExists, subscriptions.ErrSubscriptionReconciliationRequired, errors.New("private-provider-canary")} {
		store := &paywallStore{err: err}
		r := httptest.NewRequest(http.MethodPost, "/v1/billing/web-paywall", strings.NewReader(`{"tier":"business"}`))
		r = r.WithContext(middleware.WithDID(r.Context(), syntax.DID("did:plc:owner")))
		w := httptest.NewRecorder()
		api.PrepareWebPaywallHandler(store, time.Now).ServeHTTP(w, r)
		if w.Code < 400 || strings.Contains(w.Body.String(), "private-provider-canary") {
			t.Fatal("unsafe provider error")
		}
	}
}
