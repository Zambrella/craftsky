package revenuecat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"social.craftsky/appview/internal/subscriptions"
)

const defaultMaxResponseBytes int64 = 1024 * 1024

type ClientConfig struct {
	BaseURL          string
	APIKey           string
	ProjectID        string
	PageLimit        int
	MaxPages         int
	MaxResponseBytes int64
}

type Client struct {
	config ClientConfig
	base   *url.URL
	http   *http.Client
}

func NewClient(config ClientConfig, httpClient *http.Client) (*Client, error) {
	base, err := url.Parse(config.BaseURL)
	if err != nil || base.Scheme == "" || base.Host == "" || base.User != nil || base.RawQuery != "" || base.Fragment != "" {
		return nil, errors.New("invalid RevenueCat base URL")
	}
	if config.APIKey == "" || config.ProjectID == "" {
		return nil, errors.New("incomplete RevenueCat client configuration")
	}
	if config.PageLimit <= 0 {
		config.PageLimit = 100
	}
	if config.MaxPages <= 0 {
		config.MaxPages = 20
	}
	if config.MaxResponseBytes <= 0 {
		config.MaxResponseBytes = defaultMaxResponseBytes
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	client := *httpClient
	client.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &Client{config: config, base: base, http: &client}, nil
}

func (c *Client) ListCustomerSubscriptions(ctx context.Context, customerID uuid.UUID) (subscriptions.CompleteSnapshot, error) {
	pageURL := *c.base
	pageURL.Path = strings.TrimRight(pageURL.Path, "/") + "/projects/" + url.PathEscape(c.config.ProjectID) +
		"/customers/" + url.PathEscape(customerID.String()) + "/subscriptions"
	query := pageURL.Query()
	query.Set("limit", strconv.Itoa(c.config.PageLimit))
	pageURL.RawQuery = query.Encode()
	endpointPath := pageURL.EscapedPath()

	result := subscriptions.CompleteSnapshot{}
	seen := make(map[string]struct{}, c.config.MaxPages)
	for page := 0; page < c.config.MaxPages; page++ {
		pageKey := pageURL.String()
		if _, duplicate := seen[pageKey]; duplicate {
			return subscriptions.CompleteSnapshot{}, errors.New("RevenueCat pagination loop detected")
		}
		seen[pageKey] = struct{}{}
		response, err := c.fetchPage(ctx, pageURL.String())
		var status *ProviderError
		if page == 0 && errors.As(err, &status) && status.StatusCode() == http.StatusNotFound {
			// Web checkout has no SDK login to create a first-time customer.
			// Create the exact server-owned UUID, then re-read: a project/path
			// misconfiguration must never be mistaken for an empty account.
			if createErr := c.createCustomer(ctx, customerID); createErr != nil {
				return subscriptions.CompleteSnapshot{}, createErr
			}
			response, err = c.fetchPage(ctx, pageURL.String())
		}
		if err != nil {
			return subscriptions.CompleteSnapshot{}, err
		}
		for _, item := range response.Items {
			snapshot, err := item.snapshot()
			if err != nil {
				return subscriptions.CompleteSnapshot{}, err
			}
			if item.Store == "promotional" {
				entitlements, err := c.promotionalEntitlements(ctx, item)
				if err != nil {
					return subscriptions.CompleteSnapshot{}, err
				}
				snapshot.Entitlements = entitlements
			}
			result.Subscriptions = append(result.Subscriptions, snapshot)
		}
		// RevenueCat documents both an omitted and a null next_page as terminal.
		// A present but empty/invalid URL must fail pagination instead.
		if response.NextPage == nil {
			result.Complete = true
			return result, nil
		}
		next, err := c.resolveNextPage(*response.NextPage, endpointPath)
		if err != nil {
			return subscriptions.CompleteSnapshot{}, fmt.Errorf("invalid RevenueCat pagination URL: %w", err)
		}
		pageURL = next
	}
	return subscriptions.CompleteSnapshot{}, errors.New("RevenueCat pagination limit exceeded")
}

func (c *Client) resolveNextPage(raw, endpointPath string) (url.URL, error) {
	reference, err := url.Parse(raw)
	if err != nil || reference.User != nil || reference.Fragment != "" || reference.Opaque != "" {
		return url.URL{}, errors.New("invalid pagination URL")
	}
	next := c.base.ResolveReference(reference)
	if !strings.EqualFold(next.Scheme, c.base.Scheme) || !strings.EqualFold(next.Host, c.base.Host) || next.EscapedPath() != endpointPath {
		return url.URL{}, errors.New("pagination URL outside configured endpoint")
	}
	query, err := url.ParseQuery(next.RawQuery)
	if err != nil || len(query["starting_after"]) != 1 || query.Get("starting_after") == "" {
		return url.URL{}, errors.New("invalid pagination cursor")
	}
	return *next, nil
}

type subscriptionPage struct {
	Items    []subscriptionResource `json:"items"`
	NextPage *string                `json:"next_page"`
}

type subscriptionResource struct {
	StoreSubscriptionIdentifier string           `json:"store_subscription_identifier"`
	ID                          string           `json:"id"`
	ProductID                   json.RawMessage  `json:"product_id"`
	Store                       string           `json:"store"`
	Environment                 string           `json:"environment"`
	Status                      string           `json:"status"`
	GivesAccess                 *bool            `json:"gives_access"`
	PendingPayment              *bool            `json:"pending_payment"`
	AutoRenewalStatus           string           `json:"auto_renewal_status"`
	StartsAt                    *int64           `json:"starts_at"`
	CurrentPeriodStartsAt       *int64           `json:"current_period_starts_at"`
	CurrentPeriodEndsAt         json.RawMessage  `json:"current_period_ends_at"`
	EndsAt                      json.RawMessage  `json:"ends_at"`
	Entitlements                *entitlementPage `json:"entitlements"`
	PendingChanges              *struct {
		Product *struct {
			ID string `json:"id"`
		} `json:"product"`
	} `json:"pending_changes"`
}

func (c *Client) fetchPage(ctx context.Context, endpoint string) (subscriptionPage, error) {
	var page subscriptionPage
	if err := c.fetchJSON(ctx, endpoint, &page); err != nil {
		return subscriptionPage{}, err
	}
	if page.Items == nil {
		return subscriptionPage{}, errors.New("invalid RevenueCat response")
	}
	return page, nil
}

func (c *Client) fetchJSON(ctx context.Context, endpoint string, destination any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return &ProviderError{stage: "create", cause: err}
	}
	request.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	request.Header.Set("Accept", "application/json")
	response, err := c.http.Do(request)
	if err != nil {
		return &ProviderError{stage: "request", cause: err}
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return &ProviderError{stage: "response", statusCode: response.StatusCode}
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, c.config.MaxResponseBytes+1))
	if err != nil {
		return &ProviderError{stage: "read", statusCode: response.StatusCode, cause: err}
	}
	if int64(len(body)) > c.config.MaxResponseBytes {
		return &ProviderError{stage: "oversized", statusCode: response.StatusCode}
	}
	if err := json.Unmarshal(body, destination); err != nil {
		return &ProviderError{stage: "decode", statusCode: response.StatusCode, cause: err}
	}
	return nil
}

func (resource subscriptionResource) snapshot() (subscriptions.ProviderSubscriptionSnapshot, error) {
	// Absence is not authoritative false/empty state. Reject malformed items
	// before allowing a complete snapshot to replace confirmed paid access.
	// product_id is explicitly nullable for promotional subscriptions; retaining
	// its raw value distinguishes that documented null from an omitted field.
	var productID *string
	if strings.TrimSpace(resource.ID) == "" || strings.TrimSpace(resource.Store) == "" ||
		strings.TrimSpace(resource.Environment) == "" || strings.TrimSpace(resource.Status) == "" ||
		resource.GivesAccess == nil || resource.PendingPayment == nil || len(resource.ProductID) == 0 {
		return subscriptions.ProviderSubscriptionSnapshot{}, errors.New("invalid RevenueCat subscription response")
	}
	if err := json.Unmarshal(resource.ProductID, &productID); err != nil {
		return subscriptions.ProviderSubscriptionSnapshot{}, &ProviderError{stage: "decode", statusCode: http.StatusOK, cause: err}
	}
	if productID != nil && strings.TrimSpace(*productID) == "" {
		return subscriptions.ProviderSubscriptionSnapshot{}, errors.New("invalid RevenueCat subscription response")
	}
	product := ""
	if productID != nil {
		product = *productID
	}
	periodEnd, err := nullableMillis(resource.CurrentPeriodEndsAt)
	if err != nil {
		return subscriptions.ProviderSubscriptionSnapshot{}, &ProviderError{stage: "decode", statusCode: http.StatusOK, cause: err}
	}
	endsAt, err := nullableMillis(resource.EndsAt)
	if err != nil {
		return subscriptions.ProviderSubscriptionSnapshot{}, &ProviderError{stage: "decode", statusCode: http.StatusOK, cause: err}
	}
	if resource.Store == "promotional" && (len(resource.CurrentPeriodEndsAt) == 0 || len(resource.EndsAt) == 0) {
		// A missing grant expiry must not turn malformed timed access into a
		// lifetime grant. Explicit null bounds are documented lifetime access.
		return subscriptions.ProviderSubscriptionSnapshot{}, errors.New("invalid RevenueCat subscription expiry")
	}
	pendingProductID := ""
	if resource.PendingChanges != nil && resource.PendingChanges.Product != nil {
		pendingProductID = resource.PendingChanges.Product.ID
	}
	return subscriptions.ProviderSubscriptionSnapshot{
		StoreSubscriptionIdentifier: resource.StoreSubscriptionIdentifier, ID: resource.ID, ProductID: product, PendingProductID: pendingProductID, Store: resource.Store,
		Environment: resource.Environment, Status: resource.Status,
		GivesAccess: *resource.GivesAccess, PendingPayment: *resource.PendingPayment,
		AutoRenewalStatus: resource.AutoRenewalStatus,
		StartsAt:          millisTime(resource.StartsAt), CurrentPeriodStartsAt: millisTime(resource.CurrentPeriodStartsAt),
		CurrentPeriodEndsAt: millisTime(periodEnd), EndsAt: millisTime(endsAt),
	}, nil
}

func nullableMillis(raw json.RawMessage) (*int64, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var value *int64
	err := json.Unmarshal(raw, &value)
	return value, err
}

type entitlementPage struct {
	Items []struct {
		ID        string `json:"id"`
		ProjectID string `json:"project_id"`
		LookupKey string `json:"lookup_key"`
	} `json:"items"`
	NextPage *string `json:"next_page"`
}

// Grants have no product mapping. Only a complete, project-scoped entitlement
// list may identify their tier; partial nested pagination is not authoritative.
func (c *Client) promotionalEntitlements(ctx context.Context, resource subscriptionResource) ([]subscriptions.ProviderEntitlementSnapshot, error) {
	endpoint := *c.base
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/projects/" + url.PathEscape(c.config.ProjectID) +
		"/subscriptions/" + url.PathEscape(resource.ID) + "/entitlements"
	endpointPath := endpoint.EscapedPath()
	page := resource.Entitlements
	result := make([]subscriptions.ProviderEntitlementSnapshot, 0)
	seenPages := make(map[string]struct{}, c.config.MaxPages)
	seenEntitlements := make(map[string]struct{})
	for count := 0; count < c.config.MaxPages; count++ {
		if page == nil || page.Items == nil {
			return nil, errors.New("invalid RevenueCat promotional entitlements")
		}
		for _, item := range page.Items {
			if strings.TrimSpace(item.ID) == "" || strings.TrimSpace(item.LookupKey) == "" || item.ProjectID != c.config.ProjectID {
				return nil, errors.New("invalid RevenueCat promotional entitlement")
			}
			if _, exists := seenEntitlements[item.ID]; exists {
				return nil, errors.New("duplicate RevenueCat promotional entitlement")
			}
			seenEntitlements[item.ID] = struct{}{}
			result = append(result, subscriptions.ProviderEntitlementSnapshot{ID: item.ID, ProjectID: item.ProjectID, LookupKey: item.LookupKey})
		}
		if page.NextPage == nil {
			return result, nil
		}
		next, err := c.resolveNextPage(*page.NextPage, endpointPath)
		if err != nil {
			return nil, fmt.Errorf("invalid RevenueCat entitlement pagination URL: %w", err)
		}
		if _, duplicate := seenPages[next.String()]; duplicate {
			return nil, errors.New("RevenueCat entitlement pagination loop detected")
		}
		seenPages[next.String()] = struct{}{}
		if count+1 == c.config.MaxPages {
			break
		}
		page = new(entitlementPage)
		if err := c.fetchJSON(ctx, next.String(), page); err != nil {
			return nil, err
		}
	}
	return nil, errors.New("RevenueCat entitlement pagination limit exceeded")
}

func millisTime(value *int64) *time.Time {
	if value == nil {
		return nil
	}
	timestamp := time.UnixMilli(*value).UTC()
	return &timestamp
}

func (c *Client) createCustomer(ctx context.Context, customerID uuid.UUID) error {
	endpoint := *c.base
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/projects/" + url.PathEscape(c.config.ProjectID) + "/customers"
	body, err := json.Marshal(struct {
		ID string `json:"id"`
	}{ID: customerID.String()})
	if err != nil {
		return &ProviderError{stage: "encode", cause: err}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), strings.NewReader(string(body)))
	if err != nil {
		return &ProviderError{stage: "create", cause: err}
	}
	request.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	response, err := c.http.Do(request)
	if err != nil {
		return &ProviderError{stage: "request", cause: err}
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated && response.StatusCode != http.StatusConflict {
		return &ProviderError{stage: "response", statusCode: response.StatusCode}
	}
	return nil
}
