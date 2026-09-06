package video

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/auth"
)

const (
	videoLimitsMethod       = "app.bsky.video.getUploadLimits"
	limitsAuthorizationLife = time.Minute
	maxLimitsResponseBytes  = 64 * 1024
)

var ErrUploadLimitsUnavailable = errors.New("video upload limits unavailable")

type UploadLimits struct {
	CanUpload            bool
	RemainingDailyVideos *int64
	RemainingDailyBytes  *int64
	Reason               string
}

type UploadLimitsResponse struct {
	CanUpload            bool    `json:"canUpload"`
	RemainingDailyVideos *int64  `json:"remainingDailyVideos,omitempty"`
	RemainingDailyBytes  *int64  `json:"remainingDailyBytes,omitempty"`
	Error                *string `json:"error,omitempty"`
	Message              *string `json:"message,omitempty"`
}

type limitsRequest func(context.Context, *http.Client, string, string) (UploadLimitsResponse, error)

type UploadLimitsServiceOptions struct {
	Sessions      auth.OAuthSessionRunner
	PDSClient     *http.Client
	VideoClient   *http.Client
	ServiceDID    syntax.DID
	ServiceURL    string
	Now           func() time.Time
	authorize     serviceAuthRequest
	requestLimits limitsRequest
}

type UploadLimitsService struct {
	sessions      auth.OAuthSessionRunner
	serviceDID    syntax.DID
	serviceURL    string
	now           func() time.Time
	authorize     serviceAuthRequest
	videoClient   *http.Client
	requestLimits limitsRequest
}

func NewUploadLimitsService(options UploadLimitsServiceOptions) (*UploadLimitsService, error) {
	if options.Sessions == nil || options.PDSClient == nil || options.VideoClient == nil || options.ServiceDID == "" {
		return nil, ErrUploadLimitsUnavailable
	}
	serviceURL, err := serviceOrigin(options.ServiceURL)
	if err != nil {
		return nil, ErrUploadLimitsUnavailable
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.authorize == nil {
		options.authorize = requestServiceAuthorization(options.PDSClient)
	}
	if options.requestLimits == nil {
		options.requestLimits = requestUploadLimits
	}
	return &UploadLimitsService{
		sessions: options.Sessions, serviceDID: options.ServiceDID, serviceURL: serviceURL,
		now: options.Now, authorize: options.authorize, videoClient: options.VideoClient,
		requestLimits: options.requestLimits,
	}, nil
}

func (service *UploadLimitsService) Get(ctx context.Context, owner syntax.DID, sessionID string) (UploadLimits, error) {
	if service == nil || service.sessions == nil || service.authorize == nil || service.requestLimits == nil || owner == "" || sessionID == "" {
		return UploadLimits{}, ErrUploadLimitsUnavailable
	}
	var limits UploadLimits
	err := service.sessions.WithActiveSession(ctx, owner, sessionID, func(operationCtx context.Context, session *oauth.ClientSession) error {
		if session == nil || session.Data == nil {
			return ErrUploadLimitsUnavailable
		}
		expiry := service.now().UTC().Add(limitsAuthorizationLife).Unix()
		token, err := service.authorize(operationCtx, session, service.serviceDID.String(), expiry, videoLimitsMethod)
		if err != nil || strings.TrimSpace(token) == "" {
			return ErrUploadLimitsUnavailable
		}
		response, err := service.requestLimits(operationCtx, service.videoClient, service.serviceURL, token)
		if err != nil {
			return ErrUploadLimitsUnavailable
		}
		limits = normalizeUploadLimits(response)
		return nil
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return UploadLimits{}, err
		}
		return UploadLimits{}, ErrUploadLimitsUnavailable
	}
	return limits, nil
}

func normalizeUploadLimits(response UploadLimitsResponse) UploadLimits {
	limits := UploadLimits{
		CanUpload: response.CanUpload, RemainingDailyVideos: response.RemainingDailyVideos,
		RemainingDailyBytes: response.RemainingDailyBytes,
	}
	detail := strings.ToLower(pointerString(response.Error) + " " + pointerString(response.Message))
	switch {
	case strings.Contains(detail, "email"):
		limits.Reason = "email_unverified"
	case strings.Contains(detail, "unsupported"):
		limits.Reason = "provider_unsupported"
	case strings.Contains(detail, "quota") || strings.Contains(detail, "limit") || strings.Contains(detail, "daily"):
		limits.Reason = "quota_exhausted"
	case response.RemainingDailyVideos != nil && *response.RemainingDailyVideos <= 1:
		limits.Reason = "quota_exhausted"
	case response.RemainingDailyBytes != nil && *response.RemainingDailyBytes < 300_000_000:
		limits.Reason = "quota_exhausted"
	case !response.CanUpload || response.Error != nil:
		limits.Reason = "unknown"
	}
	if limits.Reason != "" {
		limits.CanUpload = false
	}
	return limits
}

func pointerString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func serviceOrigin(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil ||
		(parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", ErrUploadLimitsUnavailable
	}
	return strings.TrimSuffix(parsed.String(), "/"), nil
}

func requestUploadLimits(ctx context.Context, client *http.Client, endpoint, token string) (UploadLimitsResponse, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"/xrpc/"+videoLimitsMethod, nil)
	if err != nil {
		return UploadLimitsResponse{}, ErrUploadLimitsUnavailable
	}
	request.Header.Set("Authorization", "Bearer "+token)
	noRedirectClient := *client
	noRedirectClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	response, err := noRedirectClient.Do(request)
	if err != nil {
		return UploadLimitsResponse{}, ErrUploadLimitsUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return UploadLimitsResponse{}, ErrUploadLimitsUnavailable
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxLimitsResponseBytes+1))
	if err != nil || len(body) > maxLimitsResponseBytes {
		return UploadLimitsResponse{}, ErrUploadLimitsUnavailable
	}
	var output UploadLimitsResponse
	if err := json.Unmarshal(body, &output); err != nil {
		return UploadLimitsResponse{}, ErrUploadLimitsUnavailable
	}
	return output, nil
}
