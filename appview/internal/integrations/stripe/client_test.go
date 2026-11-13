package stripe

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/subscriptions"
)

func stripeTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	c, err := NewClient(Config{SecretKey: "sk_test_secret_canary", AccountID: "acct_test", Environment: "sandbox", PortalConfigurationID: "bpc_test"}, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	c.base = server.URL
	return c
}
func TestManagedCheckoutRequestHasServerOwnedIdentityAndIdempotency(t *testing.T) {
	customer := uuid.New()
	operation := uuid.New()
	calls := 0
	client := stripeTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk_test_secret_canary" || r.Header.Get("Stripe-Version") != APIVersion {
			t.Error("missing authentication/version")
		}
		if r.URL.Path == "/account" {
			_, _ = w.Write([]byte(`{"id":"acct_test"}`))
			return
		}
		calls++
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		expected := map[string]string{"mode": "subscription", "line_items[0][quantity]": "1", "line_items[0][price]": "price_plus", "managed_payments[enabled]": "true", "metadata[app_user_id]": customer.String(), "subscription_data[metadata][app_user_id]": customer.String(), "success_url": "https://app.example/success", "cancel_url": "https://app.example/cancel"}
		if len(r.Form) != len(expected) {
			t.Errorf("unexpected request fields=%v", r.Form)
		}
		for key, want := range expected {
			if r.Form.Get(key) != want {
				t.Errorf("%s mismatch", key)
			}
		}
		if r.Header.Get("Idempotency-Key") != "craftsky-checkout-"+operation.String() {
			t.Error("idempotency key lost")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "cs_test_123", "url": "https://checkout.stripe.com/c/pay/cs_test_123#opaque", "status": "open", "livemode": false, "expires_at": 2000000000, "metadata": map[string]string{"app_user_id": customer.String()}, "managed_payments": map[string]bool{"enabled": true}, "automatic_tax": map[string]any{"liability": map[string]string{"type": "stripe"}}})
	})
	request := subscriptions.CheckoutRequest{OperationID: operation, CustomerID: customer, AccountID: "acct_test", PriceID: "price_plus", SuccessURL: "https://app.example/success", CancelURL: "https://app.example/cancel"}
	for range 2 {
		session, err := client.CreateCheckout(t.Context(), request)
		if err != nil || !session.Managed || !session.StripeTaxLiability || session.AppUserID != customer.String() {
			t.Fatalf("checkout decode err=%v", err)
		}
	}
	if calls != 2 {
		t.Fatal("unexpected requests")
	}
}
func TestCheckoutRetrievalAndManagementVerifyActualMerchant(t *testing.T) {
	requests := 0
	client := stripeTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/account" {
			_, _ = w.Write([]byte(`{"id":"acct_other"}`))
			return
		}
		requests++
		w.WriteHeader(500)
	})
	if _, err := client.GetCheckout(t.Context(), "cs_test_valid"); err == nil {
		t.Fatal("other merchant read accepted")
	}
	if _, err := client.CreateManagement(t.Context(), "cus_owner", "https://app.example/billing"); err == nil {
		t.Fatal("other merchant portal accepted")
	}
	if requests != 0 {
		t.Fatal("dispatched past merchant mismatch")
	}
}
func TestStripeNeverForwardsCredentialsOnRedirect(t *testing.T) {
	forwarded := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { forwarded = true }))
	defer target.Close()
	client := stripeTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	})
	_, err := client.CreateCheckout(t.Context(), subscriptions.CheckoutRequest{AccountID: "acct_test"})
	var typed *Error
	if !errors.As(err, &typed) || typed.Status != http.StatusTemporaryRedirect || forwarded {
		t.Fatalf("redirect boundary status=%v forwarded=%v", err, forwarded)
	}
}
func TestManagementIsCustomerScopedAndRejectsEmailLoginDestination(t *testing.T) {
	client := stripeTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/account" {
			_, _ = w.Write([]byte(`{"id":"acct_test"}`))
			return
		}
		_ = r.ParseForm()
		if r.Form.Get("customer") != "cus_owner" || r.Form.Get("configuration") != "bpc_test" || r.Form.Get("return_url") != "https://app.example/billing" {
			t.Error("portal scope mismatch")
		}
		_, _ = w.Write([]byte(`{"url":"https://billing.stripe.com/p/login/test_shared_email"}`))
	})
	if _, err := client.CreateManagement(t.Context(), "cus_owner", "https://app.example/billing"); err == nil {
		t.Fatal("email-based portal accepted")
	}
}
func TestStripeErrorsPreserveCauseWithoutSerializedCapabilities(t *testing.T) {
	cause := &url.Error{Op: "Get", URL: "https://api.stripe.com/v1/checkout/sessions/cs_test_private_capability", Err: errors.New("credential_payload_canary")}
	wrapped := &Error{Cause: cause}
	var typed *url.Error
	if !errors.As(wrapped, &typed) || typed != cause {
		t.Fatal("original typed cause lost")
	}
	var output bytes.Buffer
	logger := slog.New(observability.NewDiagnosticHandler(slog.NewJSONHandler(&output, nil)))
	observability.LogDiagnostic(context.Background(), logger, observability.DiagnosticInput{Error: wrapped, Context: observability.EventContext{"component": "api", "operation": "billing.checkout", "failure_stage": "provider"}})
	serialized := output.String() + fmt.Sprintf("%v %#v", wrapped, wrapped)
	for _, canary := range []string{"cs_test_private_capability", "credential_payload_canary"} {
		if strings.Contains(serialized, canary) {
			t.Fatalf("protected provider data leaked: %s", canary)
		}
	}
	if !strings.Contains(output.String(), "billing.checkout") || !strings.Contains(output.String(), "provider") {
		t.Fatal("safe diagnostic context missing")
	}
}
