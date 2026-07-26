package tap

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

const defaultAdminTimeout = 5 * time.Second

// RepositoryTracker requests Tap's ordinary repository tracking/backfill.
// Repeated requests for the same DID are intentionally supported.
type RepositoryTracker interface {
	AddRepo(context.Context, syntax.DID) error
}

type AdminClient struct {
	baseURL string
	client  *http.Client
}

func NewAdminClient(tapURL string, client *http.Client) (*AdminClient, error) {
	baseURL, err := HTTPBaseURL(tapURL)
	if err != nil {
		return nil, err
	}
	if client == nil {
		client = &http.Client{Timeout: defaultAdminTimeout}
	} else if client.Timeout == 0 {
		clone := *client
		clone.Timeout = defaultAdminTimeout
		client = &clone
	}
	return &AdminClient{baseURL: baseURL, client: client}, nil
}

func HTTPBaseURL(tapURL string) (string, error) {
	u, err := url.Parse(tapURL)
	if err != nil {
		return "", fmt.Errorf("parse TAP_WS_URL: %w", err)
	}
	switch u.Scheme {
	case "ws":
		u.Scheme = "http"
	case "wss":
		u.Scheme = "https"
	case "http", "https":
	default:
		return "", fmt.Errorf("unsupported TAP_WS_URL scheme %q", u.Scheme)
	}
	u.Path = strings.TrimSuffix(u.Path, "/channel")
	u.RawQuery = ""
	u.Fragment = ""
	return strings.TrimRight(u.String(), "/"), nil
}

func (c *AdminClient) AddRepo(ctx context.Context, did syntax.DID) error {
	body, err := json.Marshal(map[string][]string{"dids": {did.String()}})
	if err != nil {
		return fmt.Errorf("encode Tap repository request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/repos/add", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build Tap repository request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("request Tap repository tracking: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("request Tap repository tracking: http %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}
	return nil
}

var _ RepositoryTracker = (*AdminClient)(nil)
