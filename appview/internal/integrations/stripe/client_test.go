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

func TestEmbeddedManagedCheckoutUsesReturnURLAndClientSecret(t *testing.T) {
	var create url.Values
	customer := uuid.New()
	client := stripeTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/account" {
			_, _ = w.Write([]byte(`{"id":"acct_test"}`))
			return
		}
		_ = r.ParseForm()
		create = r.Form
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "cs_test_embedded", "ui_mode": "embedded", "client_secret": "cs_test_embedded_secret_canary", "status": "open", "livemode": false, "metadata": map[string]string{"app_user_id": customer.String()}, "managed_payments": map[string]bool{"enabled": true}, "automatic_tax": map[string]any{"liability": map[string]string{"type": "stripe"}}})
	})
	session, err := client.CreateCheckout(t.Context(), subscriptions.CheckoutRequest{Embedded: true, OperationID: uuid.New(), CustomerID: customer, AccountID: "acct_test", PriceID: "price_plus", SuccessURL: "https://app.example/return"})
	if err != nil || session.ClientSecret != "cs_test_embedded_secret_canary" || !session.Managed || !session.StripeTaxLiability {
		t.Fatalf("embedded response err=%v", err)
	}
	for key, want := range map[string]string{"ui_mode": "embedded", "return_url": "https://app.example/return", "redirect_on_completion": "if_required", "managed_payments[enabled]": "true"} {
		if create.Get(key) != want {
			t.Errorf("%s mismatch", key)
		}
	}
	if create.Has("success_url") || create.Has("cancel_url") {
		t.Fatal("hosted return fields sent to embedded Checkout")
	}
}

func TestSubscriptionCustomerChecksMerchantEnvironmentAndIdentifier(t *testing.T) {
	for _, test := range []struct {
		name, merchant, id, customer string
		live                         bool
		want                         bool
	}{
		{"owned", "acct_test", "sub_owner", "cus_owner", false, true},
		{"other merchant", "acct_other", "sub_owner", "cus_owner", false, false},
		{"live", "acct_test", "sub_owner", "cus_owner", true, false},
		{"other subscription", "acct_test", "sub_other", "cus_owner", false, false},
		{"invalid customer", "acct_test", "sub_owner", "bad", false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := stripeTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/account" {
					_ = json.NewEncoder(w).Encode(map[string]any{"id": test.merchant})
					return
				}
				if r.URL.Path != "/subscriptions/sub_owner" {
					t.Errorf("unexpected endpoint %s", r.URL.Path)
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"id": test.id, "customer": test.customer, "livemode": test.live})
			})
			got, err := client.SubscriptionCustomer(t.Context(), "sub_owner")
			if test.want {
				if err != nil || got != "cus_owner" {
					t.Fatalf("customer lookup: %v", err)
				}
			} else if err == nil {
				t.Fatal("mismatch accepted")
			}
		})
	}
}

func TestSubscriptionItemManagementResolvesParentAndFailsClosed(t *testing.T) {
	for _, test := range []struct {
		name, item, parent, merchant, customer string
		live, want                             bool
	}{
		{"owned", "si_owner", "sub_owner", "acct_test", "cus_owner", false, true},
		{"wrong item", "si_other", "sub_owner", "acct_test", "cus_owner", false, false},
		{"invalid parent", "si_owner", "cus_owner", "acct_test", "cus_owner", false, false},
		{"missing parent", "si_owner", "", "acct_test", "cus_owner", false, false},
		{"wrong merchant", "si_owner", "sub_owner", "acct_other", "cus_owner", false, false},
		{"wrong environment", "si_owner", "sub_owner", "acct_test", "cus_owner", true, false},
		{"invalid customer", "si_owner", "sub_owner", "acct_test", "invalid", false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			portalRequests := 0
			client := stripeTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/account":
					_ = json.NewEncoder(w).Encode(map[string]any{"id": test.merchant})
				case "/subscription_items/si_owner":
					_ = json.NewEncoder(w).Encode(map[string]any{"id": test.item, "subscription": test.parent})
				case "/subscriptions/sub_owner":
					_ = json.NewEncoder(w).Encode(map[string]any{"id": "sub_owner", "customer": test.customer, "livemode": test.live})
				case "/billing_portal/sessions":
					portalRequests++
					_ = r.ParseForm()
					if r.Form.Get("customer") != "cus_owner" {
						t.Error("incorrect portal customer")
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"url": "https://billing.stripe.com/p/session/test_owner"})
				default:
					t.Errorf("unexpected request %s", r.URL.Path)
					w.WriteHeader(400)
				}
			})
			customer, err := client.SubscriptionCustomer(t.Context(), "si_owner")
			if test.want {
				if err != nil || customer != "cus_owner" {
					t.Fatalf("item resolution: %v", err)
				}
				if _, err = client.CreateManagement(t.Context(), customer, "https://app.example/billing"); err != nil {
					t.Fatal(err)
				}
				if portalRequests != 1 {
					t.Fatal("portal was not opened")
				}
			} else if err == nil || portalRequests != 0 {
				t.Fatal("mismatch accepted")
			}
		})
	}
}
