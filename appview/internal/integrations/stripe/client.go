// Package stripe owns the narrow server-side Managed Payments checkout boundary.
package stripe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"social.craftsky/appview/internal/subscriptions"
)

const APIVersion = "2025-03-31.basil"

type Config struct{ SecretKey, AccountID, Environment, PortalConfigurationID string }
type Client struct {
	config Config
	http   *http.Client
	base   string
}

func (c *Client) String() string             { return "StripeClient{configuration:[REDACTED]}" }
func (c *Client) GoString() string           { return c.String() }
func (c *Client) Format(s fmt.State, _ rune) { _, _ = io.WriteString(s, c.String()) }
func NewClient(config Config, transport *http.Client) (*Client, error) {
	live := config.Environment == "production"
	if config.Environment != "sandbox" && !live {
		return nil, errors.New("invalid Stripe environment")
	}
	if config.AccountID == "" || !strings.HasPrefix(config.AccountID, "acct_") || (live && !strings.HasPrefix(config.SecretKey, "sk_live_")) || (!live && !strings.HasPrefix(config.SecretKey, "sk_test_")) {
		return nil, errors.New("Stripe credentials do not match configured account environment")
	}
	if transport == nil {
		transport = &http.Client{Timeout: 10 * time.Second}
	}
	client := *transport
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	if client.Timeout == 0 || client.Timeout > 30*time.Second {
		client.Timeout = 10 * time.Second
	}
	return &Client{config: config, http: &client, base: "https://api.stripe.com/v1"}, nil
}

type Session struct {
	ID              string            `json:"id"`
	URL             string            `json:"url"`
	Status          string            `json:"status"`
	LiveMode        *bool             `json:"livemode"`
	ExpiresAt       int64             `json:"expires_at"`
	Customer        string            `json:"customer"`
	Metadata        map[string]string `json:"metadata"`
	ManagedPayments struct {
		Enabled bool `json:"enabled"`
	} `json:"managed_payments"`
	AutomaticTax struct {
		Liability struct {
			Type string `json:"type"`
		} `json:"liability"`
	} `json:"automatic_tax"`
	Subscription json.RawMessage `json:"subscription"`
}

func (s Session) String() string             { return "StripeCheckoutSession{private:[REDACTED]}" }
func (s Session) GoString() string           { return s.String() }
func (s Session) Format(f fmt.State, _ rune) { _, _ = io.WriteString(f, s.String()) }
func (s Session) checkout() subscriptions.CheckoutSession {
	out := subscriptions.CheckoutSession{ID: s.ID, URL: s.URL, Status: s.Status, LiveMode: s.LiveMode != nil && *s.LiveMode, EnvironmentKnown: s.LiveMode != nil, ExpiresAt: time.Unix(s.ExpiresAt, 0).UTC(), CustomerID: s.Customer, AppUserID: s.Metadata["app_user_id"], Managed: s.ManagedPayments.Enabled, StripeTaxLiability: s.AutomaticTax.Liability.Type == "stripe"}
	var sub struct {
		Status   string            `json:"status"`
		Metadata map[string]string `json:"metadata"`
	}
	if json.Unmarshal(s.Subscription, &sub) == nil {
		out.SubscriptionCanceled = sub.Status == "canceled"
		out.SubscriptionAppUserID = sub.Metadata["app_user_id"]
	}
	return out
}
func (c *Client) CreateCheckout(ctx context.Context, request subscriptions.CheckoutRequest) (subscriptions.CheckoutSession, error) {
	// An idempotency key is scoped to one merchant. Verify the key's actual
	// merchant before dispatch so a credential/configuration change cannot create
	// the same durable operation in another account.
	var account struct {
		ID string `json:"id"`
	}
	if err := c.request(ctx, http.MethodGet, "/account", nil, "", &account); err != nil {
		return subscriptions.CheckoutSession{}, err
	}
	if account.ID != request.AccountID || request.AccountID != c.config.AccountID {
		return subscriptions.CheckoutSession{}, errors.New("Stripe account mismatch")
	}
	values := url.Values{"mode": {"subscription"}, "line_items[0][price]": {request.PriceID}, "line_items[0][quantity]": {"1"}, "managed_payments[enabled]": {"true"}, "metadata[app_user_id]": {request.CustomerID.String()}, "subscription_data[metadata][app_user_id]": {request.CustomerID.String()}, "success_url": {request.SuccessURL}, "cancel_url": {request.CancelURL}}
	var session Session
	if err := c.request(ctx, http.MethodPost, "/checkout/sessions", values, "craftsky-checkout-"+request.OperationID.String(), &session); err != nil {
		return subscriptions.CheckoutSession{}, err
	}
	if session.Status == "complete" {
		return c.GetCheckout(ctx, session.ID)
	}
	return session.checkout(), nil
}
func (c *Client) verifyAccount(ctx context.Context) error {
	var account struct {
		ID string `json:"id"`
	}
	if err := c.request(ctx, http.MethodGet, "/account", nil, "", &account); err != nil {
		return err
	}
	if account.ID != c.config.AccountID {
		return errors.New("Stripe account mismatch")
	}
	return nil
}

var sessionID = regexp.MustCompile(`^cs_(?:test_|live_)[A-Za-z0-9]+$`)

func (c *Client) GetCheckout(ctx context.Context, id string) (subscriptions.CheckoutSession, error) {
	if err := c.verifyAccount(ctx); err != nil {
		return subscriptions.CheckoutSession{}, err
	}
	if !sessionID.MatchString(id) {
		return subscriptions.CheckoutSession{}, errors.New("invalid stored Stripe session")
	}
	var session Session
	if err := c.request(ctx, http.MethodGet, "/checkout/sessions/"+id+"?expand[]=subscription", nil, "", &session); err != nil {
		return subscriptions.CheckoutSession{}, err
	}
	return session.checkout(), nil
}
func (c *Client) CreateManagement(ctx context.Context, customer, returnURL string) (string, error) {
	if err := c.verifyAccount(ctx); err != nil {
		return "", err
	}
	if !regexp.MustCompile(`^cus_[A-Za-z0-9]+$`).MatchString(customer) {
		return "", errors.New("invalid Stripe customer")
	}
	values := url.Values{"customer": {customer}, "return_url": {returnURL}}
	if c.config.PortalConfigurationID != "" {
		values.Set("configuration", c.config.PortalConfigurationID)
	}
	var result struct {
		URL      string `json:"url"`
		LiveMode *bool  `json:"livemode"`
	}
	if err := c.request(ctx, http.MethodPost, "/billing_portal/sessions", values, "", &result); err != nil {
		return "", err
	}
	// Portal session objects do not expose livemode; ownership comes from the
	// verified completed Checkout session and authenticated merchant credential.
	if !subscriptions.ValidStripeManagementURL(result.URL) {
		return "", errors.New("invalid Stripe management destination")
	}
	return result.URL, nil
}

// Error retains a typed cause without publishing provider payloads or URLs.
type Error struct {
	Status int
	Cause  error
}

func (e *Error) Error() string              { return fmt.Sprintf("Stripe request failed (status %d)", e.Status) }
func (e *Error) Unwrap() error              { return e.Cause }
func (e *Error) GoString() string           { return e.Error() }
func (e *Error) Format(s fmt.State, _ rune) { _, _ = io.WriteString(s, e.Error()) }
func (c *Client) request(ctx context.Context, method, path string, values url.Values, key string, result any) error {
	var body io.Reader
	if values != nil {
		body = strings.NewReader(values.Encode())
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, body)
	if err != nil {
		return &Error{Cause: err}
	}
	req.Header.Set("Authorization", "Bearer "+c.config.SecretKey)
	req.Header.Set("Stripe-Version", APIVersion)
	req.Header.Set("Accept", "application/json")
	if values != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	response, err := c.http.Do(req)
	if err != nil {
		return &Error{Cause: err}
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return &Error{Status: response.StatusCode}
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, 256*1024+1))
	if err != nil {
		return &Error{Cause: err}
	}
	if len(raw) > 256*1024 {
		return errors.New("Stripe response exceeds limit")
	}
	if err = json.Unmarshal(raw, result); err != nil {
		return &Error{Cause: err}
	}
	return nil
}

var _ subscriptions.CheckoutProvider = (*Client)(nil)
