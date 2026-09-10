package subscriptions

import (
	"os"
	"strings"
	"testing"
)

func TestBillingPersistenceContainsNoRawPayloadOrCredentialColumns(t *testing.T) {
	for _, path := range []string{"../../migrations/000069_subscription_accounts.up.sql", "../../migrations/000070_revenuecat_events.up.sql"} {
		contents, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		normalized := strings.ToLower(string(contents))
		for _, prohibited := range []string{"raw_body", "raw_payload", "receipt", "payment_detail", "api_key", "signing_secret", "authorization_header"} {
			if strings.Contains(normalized, prohibited) {
				t.Fatalf("%s contains prohibited billing persistence field %q", path, prohibited)
			}
		}
	}
}
