package api_test

import (
	"context"
	"errors"
	"fmt"
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

type webCheckoutStore struct {
	calls int
	owner syntax.DID
	tier  subscriptions.Tier
	err   error
}

func (s *webCheckoutStore) WebCheckoutURL(_ context.Context, owner syntax.DID, tier subscriptions.Tier, _ time.Time) (string, error) {
	s.calls++
	s.owner = owner
	s.tier = tier
	return "https://checkout.stripe.com/c/pay/cs_live_fixture#opaque", s.err
}

func TestWebCheckoutUsesAuthenticatedIdentityAndStrictTierBody(t *testing.T) {
	for _, tc := range []struct {
		name, body    string
		providerError error
		status        int
		calls         int
		code          string
	}{
		{"identified checkout", `{"tier":"plus"}`, nil, http.StatusOK, 1, ""},
		{"customer injection", `{"tier":"plus","customerId":"other"}`, nil, http.StatusBadRequest, 0, "invalid_request"},
		{"DID injection", `{"tier":"business","did":"did:plc:other"}`, nil, http.StatusBadRequest, 0, "invalid_request"},
		{"invalid tier", `{"tier":"free"}`, nil, http.StatusBadRequest, 0, "invalid_request"},
		{"unconfigured", `{"tier":"plus"}`, subscriptions.ErrWebCheckoutUnavailable, http.StatusServiceUnavailable, 1, "web_checkout_unavailable"},
		{"existing subscription", `{"tier":"business"}`, subscriptions.ErrSubscriptionAlreadyExists, http.StatusConflict, 1, "subscription_already_exists"},
		{"pending checkout", `{"tier":"business"}`, subscriptions.ErrWebCheckoutPending, http.StatusConflict, 1, "web_checkout_pending"},
		{"stale subscription", `{"tier":"plus"}`, subscriptions.ErrSubscriptionReconciliationRequired, http.StatusConflict, 1, "subscription_reconciliation_required"},
		{"provider internal", `{"tier":"plus"}`, errors.New("private-canary"), http.StatusInternalServerError, 1, "internal_error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &webCheckoutStore{err: tc.providerError}
			handler := api.CreateWebSubscriptionCheckoutHandler(store, time.Now)
			request := httptest.NewRequest(http.MethodPost, "/v1/billing/web-checkout", strings.NewReader(tc.body))
			request = request.WithContext(middleware.WithDID(request.Context(), syntax.DID("did:plc:authenticated")))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != tc.status || store.calls != tc.calls {
				t.Fatalf("status=%d body=%s calls=%d", response.Code, response.Body.String(), store.calls)
			}
			if store.calls > 0 && store.owner != "did:plc:authenticated" {
				t.Fatalf("checkout owner=%s", store.owner)
			}
			if tc.code != "" && !strings.Contains(response.Body.String(), `"error":"`+tc.code+`"`) {
				t.Fatalf("error body=%s", response.Body.String())
			}
			if strings.Contains(response.Body.String(), "private-canary") {
				t.Fatal("private cause leaked")
			}
			if tc.status == http.StatusOK && response.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("checkout identity URL was cacheable")
			}
		})
	}
}

type webManagementStore struct {
	owner syntax.DID
	calls int
	err   error
}

func (s *webManagementStore) WebManagementURL(_ context.Context, owner syntax.DID, _ time.Time) (string, error) {
	s.calls++
	s.owner = owner
	return "https://billing.stripe.com/p/session/test_customer", s.err
}
func TestWebManagementUsesAuthenticatedCustomerAndRejectsBody(t *testing.T) {
	for _, tc := range []struct {
		body   string
		err    error
		status int
		calls  int
	}{
		{"", nil, http.StatusOK, 1}, {`{"customer":"cus_other"}`, nil, http.StatusBadRequest, 0}, {"", subscriptions.ErrWebManagementUnavailable, http.StatusServiceUnavailable, 1}, {"", subscriptions.ErrWebCheckoutUnavailable, http.StatusServiceUnavailable, 1},
	} {
		t.Run(tc.body+fmt.Sprint(tc.status), func(t *testing.T) {
			store := &webManagementStore{err: tc.err}
			handler := api.CreateWebSubscriptionManagementHandler(store, time.Now)
			request := httptest.NewRequest(http.MethodPost, "/v1/billing/web-management", strings.NewReader(tc.body))
			request = request.WithContext(middleware.WithDID(request.Context(), syntax.DID("did:plc:owner")))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != tc.status || store.calls != tc.calls {
				t.Fatalf("status=%d calls=%d body=%s", response.Code, store.calls, response.Body.String())
			}
			if store.calls > 0 && store.owner != "did:plc:owner" {
				t.Fatal("owner mismatch")
			}
			if tc.status == http.StatusOK && response.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("portal URL cacheable")
			}
		})
	}
}
