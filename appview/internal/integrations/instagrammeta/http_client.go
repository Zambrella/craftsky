package instagrammeta

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	InstagramGraphBaseURL      = "https://graph.instagram.com"
	MaxProviderTimeout         = 5 * time.Second
	MaxProviderResponseBytes   = 64 << 10
	MaxProviderConcurrentCalls = 20
	maxProviderRetryAfter      = 5 * time.Minute
)

var graphAPIVersionPattern = regexp.MustCompile(`^v[0-9]+\.[0-9]+$`)

type HTTPClientConfig struct {
	HTTPClient        *http.Client  `json:"-"`
	BaseURL           string        `json:"-"`
	APIVersion        string        `json:"-"`
	AccessToken       string        `json:"-"`
	OfficialAccountID string        `json:"-"`
	RequestTimeout    time.Duration `json:"-"`
	ResponseLimit     int64         `json:"-"`
	MaxConcurrent     int           `json:"-"`
}

func (HTTPClientConfig) String() string {
	return "instagram HTTP client config [REDACTED]"
}

func (HTTPClientConfig) GoString() string {
	return "instagram HTTP client config [REDACTED]"
}

type HTTPClient struct {
	httpClient        *http.Client
	baseURL           *url.URL
	apiVersion        string
	accessToken       string
	officialAccountID string
	timeout           time.Duration
	responseLimit     int64
	concurrency       chan struct{}
}

func (*HTTPClient) String() string {
	return "instagram HTTP client [REDACTED]"
}

func (*HTTPClient) GoString() string {
	return "instagram HTTP client [REDACTED]"
}

func NewHTTPClient(config HTTPClientConfig) (*HTTPClient, error) {
	if config.HTTPClient == nil {
		return nil, errors.New("Instagram HTTP client is required")
	}
	if !graphAPIVersionPattern.MatchString(config.APIVersion) {
		return nil, errors.New("Instagram Graph API version is invalid")
	}
	if config.AccessToken == "" {
		return nil, errors.New("Instagram access token is required")
	}
	if !validProviderID(config.OfficialAccountID) {
		return nil, errors.New("official Instagram account ID is required")
	}

	baseURLString := config.BaseURL
	if baseURLString == "" {
		baseURLString = InstagramGraphBaseURL
	}
	baseURL, err := url.Parse(baseURLString)
	if err != nil || baseURL.Scheme != "https" || baseURL.Host == "" || baseURL.User != nil || baseURL.RawQuery != "" || baseURL.Fragment != "" {
		return nil, errors.New("Instagram Graph base URL is invalid")
	}
	baseURL.Path = strings.TrimRight(baseURL.Path, "/")

	timeout := config.RequestTimeout
	if timeout == 0 {
		timeout = MaxProviderTimeout
	}
	if timeout < 0 || timeout > MaxProviderTimeout {
		return nil, errors.New("Instagram provider timeout exceeds the fixed maximum")
	}
	responseLimit := config.ResponseLimit
	if responseLimit == 0 {
		responseLimit = MaxProviderResponseBytes
	}
	if responseLimit < 0 || responseLimit > MaxProviderResponseBytes {
		return nil, errors.New("Instagram provider response limit exceeds the fixed maximum")
	}
	maxConcurrent := config.MaxConcurrent
	if maxConcurrent == 0 {
		maxConcurrent = MaxProviderConcurrentCalls
	}
	if maxConcurrent < 0 || maxConcurrent > MaxProviderConcurrentCalls {
		return nil, errors.New("Instagram provider concurrency exceeds the fixed maximum")
	}
	httpClient := *config.HTTPClient
	httpClient.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}

	return &HTTPClient{
		httpClient:        &httpClient,
		baseURL:           baseURL,
		apiVersion:        config.APIVersion,
		accessToken:       config.AccessToken,
		officialAccountID: config.OfficialAccountID,
		timeout:           timeout,
		responseLimit:     responseLimit,
		concurrency:       make(chan struct{}, maxConcurrent),
	}, nil
}

func (c *HTTPClient) LookupUsername(ctx context.Context, senderIGSID string) (string, error) {
	if !validProviderID(senderIGSID) {
		return "", &ProviderError{kind: ProviderErrorPermanent}
	}
	endpoint := c.endpoint(senderIGSID)
	query := endpoint.Query()
	query.Set("fields", "username")
	endpoint.RawQuery = query.Encode()

	response, err := c.do(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	var profile struct {
		Username string `json:"username"`
	}
	if err := json.Unmarshal(response, &profile); err != nil || !validUsername(profile.Username) {
		return "", &ProviderError{kind: ProviderErrorInvalidResponse}
	}
	return strings.ToLower(profile.Username), nil
}

func (c *HTTPClient) SendReply(ctx context.Context, senderIGSID, text string) error {
	if !validProviderID(senderIGSID) || text == "" {
		return &ProviderError{kind: ProviderErrorPermanent}
	}
	payload, err := json.Marshal(replyRequest{
		Recipient: replyRecipient{ID: senderIGSID},
		Message:   replyMessage{Text: text},
	})
	if err != nil {
		return &ProviderError{kind: ProviderErrorPermanent}
	}
	_, err = c.do(ctx, http.MethodPost, c.endpoint(c.officialAccountID+"/messages"), payload)
	return err
}

type replyRequest struct {
	Recipient replyRecipient `json:"recipient"`
	Message   replyMessage   `json:"message"`
}

type replyRecipient struct {
	ID string `json:"id"`
}

type replyMessage struct {
	Text string `json:"text"`
}

func (c *HTTPClient) endpoint(path string) *url.URL {
	endpoint := *c.baseURL
	segments := strings.Split(path, "/")
	escaped := make([]string, 0, len(segments)+1)
	escaped = append(escaped, url.PathEscape(c.apiVersion))
	for _, segment := range segments {
		escaped = append(escaped, url.PathEscape(segment))
	}
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/" + strings.Join(escaped, "/")
	return &endpoint
}

func (c *HTTPClient) do(ctx context.Context, method string, endpoint *url.URL, body []byte) ([]byte, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	select {
	case c.concurrency <- struct{}{}:
		defer func() { <-c.concurrency }()
	case <-callCtx.Done():
		return nil, classifyContextError(callCtx.Err())
	}

	request, err := http.NewRequestWithContext(callCtx, method, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return nil, &ProviderError{kind: ProviderErrorPermanent}
	}
	request.Header.Set("Authorization", "Bearer "+c.accessToken)
	request.Header.Set("Accept", "application/json")
	if len(body) > 0 {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		if callCtx.Err() != nil {
			return nil, classifyContextError(callCtx.Err())
		}
		return nil, &ProviderError{kind: ProviderErrorTransient}
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(response.Body, c.responseLimit+1))
	if err != nil {
		if callCtx.Err() != nil {
			return nil, classifyContextError(callCtx.Err())
		}
		return nil, &ProviderError{kind: ProviderErrorTransient}
	}
	if int64(len(responseBody)) > c.responseLimit {
		return nil, &ProviderError{kind: ProviderErrorInvalidResponse}
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, classifyResponse(response.StatusCode, response.Header.Get("Retry-After"), responseBody)
	}
	return responseBody, nil
}

func classifyContextError(err error) error {
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	return &ProviderError{kind: ProviderErrorTransient}
}

func classifyStatus(status int, retryAfterValue string) error {
	switch status {
	case http.StatusUnauthorized, http.StatusForbidden:
		return &ProviderError{kind: ProviderErrorAuthentication}
	case http.StatusTooManyRequests:
		return &ProviderError{kind: ProviderErrorRateLimited, retryAfter: parseRetryAfter(retryAfterValue)}
	case http.StatusNotFound:
		return &ProviderError{kind: ProviderErrorNotFound}
	case http.StatusRequestTimeout:
		return &ProviderError{kind: ProviderErrorTransient}
	}
	if status >= http.StatusInternalServerError {
		return &ProviderError{kind: ProviderErrorTransient}
	}
	return &ProviderError{kind: ProviderErrorPermanent}
}

func classifyResponse(status int, retryAfterValue string, body []byte) error {
	var envelope struct {
		Error struct {
			Code         int  `json:"code"`
			ErrorSubcode int  `json:"error_subcode"`
			IsTransient  bool `json:"is_transient"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &envelope) == nil {
		switch envelope.Error.Code {
		case 10, 190, 200:
			return &ProviderError{kind: ProviderErrorAuthentication}
		case 4, 17, 32, 613:
			return &ProviderError{kind: ProviderErrorRateLimited, retryAfter: parseRetryAfter(retryAfterValue)}
		}
		if envelope.Error.Code == 100 && envelope.Error.ErrorSubcode == 33 {
			return &ProviderError{kind: ProviderErrorNotFound}
		}
		if envelope.Error.IsTransient {
			return &ProviderError{kind: ProviderErrorTransient}
		}
	}
	return classifyStatus(status, retryAfterValue)
}

func parseRetryAfter(value string) time.Duration {
	seconds, err := strconv.ParseInt(value, 10, 32)
	if err != nil || seconds <= 0 {
		return 0
	}
	retryAfter := time.Duration(seconds) * time.Second
	if retryAfter > maxProviderRetryAfter {
		return maxProviderRetryAfter
	}
	return retryAfter
}

func validUsername(username string) bool {
	if len(username) == 0 || len(username) > 30 {
		return false
	}
	for _, b := range []byte(username) {
		if (b < 'a' || b > 'z') && (b < 'A' || b > 'Z') && (b < '0' || b > '9') && b != '_' && b != '.' {
			return false
		}
	}
	return true
}
