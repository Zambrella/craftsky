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
		if err != nil {
			return subscriptions.CompleteSnapshot{}, err
		}
		for _, item := range response.Items {
			result.Subscriptions = append(result.Subscriptions, item.snapshot())
		}
		if response.NextPage == nil || *response.NextPage == "" {
			result.Complete = true
			return result, nil
		}
		next, err := c.resolveNextPage(*response.NextPage, endpointPath)
		if err != nil {
			return subscriptions.CompleteSnapshot{}, errors.New("invalid RevenueCat pagination URL")
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
	ID                    string `json:"id"`
	ProductID             string `json:"product_id"`
	Store                 string `json:"store"`
	Environment           string `json:"environment"`
	Status                string `json:"status"`
	GivesAccess           bool   `json:"gives_access"`
	PendingPayment        bool   `json:"pending_payment"`
	AutoRenewalStatus     string `json:"auto_renewal_status"`
	StartsAt              *int64 `json:"starts_at"`
	CurrentPeriodStartsAt *int64 `json:"current_period_starts_at"`
	CurrentPeriodEndsAt   *int64 `json:"current_period_ends_at"`
	EndsAt                *int64 `json:"ends_at"`
	PendingChanges        *struct {
		Product *struct {
			ID string `json:"id"`
		} `json:"product"`
	} `json:"pending_changes"`
}

func (c *Client) fetchPage(ctx context.Context, endpoint string) (subscriptionPage, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return subscriptionPage{}, fmt.Errorf("create RevenueCat request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	request.Header.Set("Accept", "application/json")
	response, err := c.http.Do(request)
	if err != nil {
		return subscriptionPage{}, errors.New("RevenueCat request failed")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return subscriptionPage{}, fmt.Errorf("RevenueCat request returned status %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, c.config.MaxResponseBytes+1))
	if err != nil {
		return subscriptionPage{}, errors.New("read RevenueCat response")
	}
	if int64(len(body)) > c.config.MaxResponseBytes {
		return subscriptionPage{}, errors.New("RevenueCat response exceeds configured limit")
	}
	var page subscriptionPage
	if err := json.Unmarshal(body, &page); err != nil {
		return subscriptionPage{}, errors.New("invalid RevenueCat response")
	}
	return page, nil
}

func (resource subscriptionResource) snapshot() subscriptions.ProviderSubscriptionSnapshot {
	pendingProductID := ""
	if resource.PendingChanges != nil && resource.PendingChanges.Product != nil {
		pendingProductID = resource.PendingChanges.Product.ID
	}
	return subscriptions.ProviderSubscriptionSnapshot{
		ID: resource.ID, ProductID: resource.ProductID, PendingProductID: pendingProductID, Store: resource.Store,
		Environment: resource.Environment, Status: resource.Status,
		GivesAccess: resource.GivesAccess, PendingPayment: resource.PendingPayment,
		AutoRenewalStatus: resource.AutoRenewalStatus,
		StartsAt:          millisTime(resource.StartsAt), CurrentPeriodStartsAt: millisTime(resource.CurrentPeriodStartsAt),
		CurrentPeriodEndsAt: millisTime(resource.CurrentPeriodEndsAt), EndsAt: millisTime(resource.EndsAt),
	}
}

func millisTime(value *int64) *time.Time {
	if value == nil {
		return nil
	}
	timestamp := time.UnixMilli(*value).UTC()
	return &timestamp
}
