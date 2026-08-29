package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/business"
)

func TestBusinessSummaryContractDecoratesEveryVisibleIdentity(t *testing.T) {
	reader := &summaryAccountTypeReader{values: map[syntax.DID]business.AccountType{
		"did:plc:business": business.AccountTypeBusiness,
	}}
	handler := api.NewIdentityAccountTypeHydrator(reader).Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"profile":{"did":"did:plc:regular","handle":"regular.test"},
			"search":{"items":[{"did":"did:plc:business","handle":"business.test"}]},
			"relationship":{"items":[{"did":"did:plc:regular","handle":"regular.test"}]},
			"post":{"author":{"did":"did:plc:business","handle":"business.test"}},
			"reply":{"author":{"did":"did:plc:regular","handle":"regular.test"}},
			"quote":{"post":{"author":{"did":"did:plc:business","handle":"business.test"}}},
			"notification":{"actor":{"available":true,"did":"did:plc:regular","handle":"regular.test"}}
		}`))
	}))

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/test", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", response.Code, response.Body.String())
	}
	var body any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	assertVisibleSummaryAccountTypes(t, body)
	if len(reader.calls) != 1 || len(reader.calls[0]) != 2 {
		t.Fatalf("account-type batch calls = %v, want one call with two unique DIDs", reader.calls)
	}
}

func hydrateProductionSummaries(handler http.Handler, values map[syntax.DID]business.AccountType) http.Handler {
	return api.NewIdentityAccountTypeHydrator(&summaryAccountTypeReader{values: values}).Handler(handler)
}

func assertVisibleSummaryAccountTypes(t *testing.T, value any) {
	t.Helper()
	switch value := value.(type) {
	case map[string]any:
		did, hasDID := value["did"].(string)
		handle, hasHandle := value["handle"].(string)
		if hasDID && hasHandle && handle != "" {
			want := "regular"
			if did == "did:plc:business" {
				want = "business"
			}
			if got := value["accountType"]; got != want {
				t.Errorf("identity %s accountType = %v, want %s", did, got, want)
			}
		}
		for _, child := range value {
			assertVisibleSummaryAccountTypes(t, child)
		}
	case []any:
		for _, child := range value {
			assertVisibleSummaryAccountTypes(t, child)
		}
	}
}
