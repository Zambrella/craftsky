package stripe

import (
	"encoding/json"
	"net/http"
	"social.craftsky/appview/internal/subscriptions"
	"sync/atomic"
	"testing"
)

func TestMonthlyPriceUsesVerifiedMerchantAndCachesPublicProjection(t *testing.T) {
	var calls atomic.Int32
	client := stripeTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path == "/account" {
			_, _ = w.Write([]byte(`{"id":"acct_test"}`))
			return
		}
		if r.URL.Path != "/prices/price_plus" {
			t.Errorf("unexpected catalog route %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"id":"price_plus","active":true,"livemode":false,"unit_amount":249,"currency":"gbp","type":"recurring","billing_scheme":"per_unit","tax_behavior":"inclusive","recurring":{"interval":"month","interval_count":1,"usage_type":"licensed"},"metadata":{"private":"canary"}}`))
	})
	for range 2 {
		price, err := client.MonthlyPrice(t.Context(), "price_plus")
		if err != nil || price != (subscriptions.WebPrice{Amount: 249, Currency: "gbp", TaxBehavior: "inclusive"}) {
			t.Fatalf("price=%#v err=%v", price, err)
		}
	}
	if calls.Load() != 2 {
		t.Fatalf("catalog was not cached: %d requests", calls.Load())
	}
}

func TestMonthlyPriceRejectsUnexpectedCatalogSemantics(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(map[string]any)
	}{
		{"wrong price", func(p map[string]any) { p["id"] = "price_other" }},
		{"inactive", func(p map[string]any) { p["active"] = false }},
		{"wrong mode", func(p map[string]any) { p["livemode"] = true }},
		{"unknown mode", func(p map[string]any) { delete(p, "livemode") }},
		{"no amount", func(p map[string]any) { p["unit_amount"] = nil }},
		{"negative amount", func(p map[string]any) { p["unit_amount"] = -10 }},
		{"annual", func(p map[string]any) { p["recurring"].(map[string]any)["interval"] = "year" }},
		{"multimonth", func(p map[string]any) { p["recurring"].(map[string]any)["interval_count"] = 3 }},
		{"metered", func(p map[string]any) { p["recurring"].(map[string]any)["usage_type"] = "metered" }},
		{"tiered", func(p map[string]any) { p["billing_scheme"] = "tiered" }},
		{"transformed", func(p map[string]any) { p["transform_quantity"] = map[string]any{"divide_by": 5} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := stripeTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/account" {
					_, _ = w.Write([]byte(`{"id":"acct_test"}`))
					return
				}
				p := map[string]any{"id": "price_plus", "active": true, "livemode": false, "unit_amount": 249, "currency": "gbp", "type": "recurring", "billing_scheme": "per_unit", "recurring": map[string]any{"interval": "month", "interval_count": 1, "usage_type": "licensed"}}
				tc.edit(p)
				_ = json.NewEncoder(w).Encode(p)
			})
			if _, err := client.MonthlyPrice(t.Context(), "price_plus"); err == nil {
				t.Fatal("accepted invalid monthly price")
			}
		})
	}
}
