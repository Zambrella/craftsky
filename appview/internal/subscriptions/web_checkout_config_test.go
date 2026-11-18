package subscriptions

import "testing"

func TestCheckoutReturnURLsPermitLoopbackOnlyInSandbox(t *testing.T) {
	for _, tc := range []struct {
		name, environment, destination string
		allowed                        bool
	}{
		{"production HTTPS", "production", "https://app.example/settings/subscriptions", true},
		{"sandbox HTTPS", "sandbox", "https://app.example/settings/subscriptions", true},
		{"sandbox loopback", "sandbox", "http://127.0.0.1:25432/settings/subscriptions", true},
		{"sandbox localhost", "sandbox", "http://localhost:25432/settings/subscriptions", true},
		{"sandbox IPv6 loopback", "sandbox", "http://[::1]:25432/settings/subscriptions", true},
		{"production loopback", "production", "http://127.0.0.1:25432/settings/subscriptions", false},
		{"unknown environment loopback", "", "http://localhost:25432/settings/subscriptions", false},
		{"sandbox remote HTTP", "sandbox", "http://app.example/settings/subscriptions", false},
		{"sandbox LAN HTTP", "sandbox", "http://192.168.1.10/settings/subscriptions", false},
		{"hostname prefix", "sandbox", "http://localhost.example/settings/subscriptions", false},
		{"credentials", "sandbox", "http://user:password@localhost/settings/subscriptions", false},
		{"query", "sandbox", "http://localhost/settings/subscriptions?return=elsewhere", false},
		{"fragment", "sandbox", "http://localhost/settings/subscriptions#return", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, field := range []string{"success", "cancel", "portal"} {
				config := WebCheckoutConfig{AccountID: "acct_test", PlusPriceID: "price_plus", BusinessPriceID: "price_business", SuccessURL: "https://app.example/success", CancelURL: "https://app.example/cancel", BillingReturnURL: "https://app.example/billing"}
				switch field {
				case "success":
					config.SuccessURL = tc.destination
				case "cancel":
					config.CancelURL = tc.destination
				case "portal":
					config.BillingReturnURL = tc.destination
				}
				if err := ValidateWebCheckoutConfig(config, tc.environment); (err == nil) != tc.allowed {
					t.Fatalf("%s accepted=%t, error=%v", field, err == nil, err)
				}
			}
		})
	}
}
