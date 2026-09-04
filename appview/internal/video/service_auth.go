package video

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	apiatproto "github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/auth"
)

const (
	videoUploadMethod        = "com.atproto.repo.uploadBlob"
	maxAuthorizationLifetime = 30 * time.Minute
)

var ErrServiceAuthorizationUnavailable = errors.New("video service authorization unavailable")

type UploadAuthorization struct {
	Token     string
	ExpiresAt time.Time
}

type serviceAuthRequest func(context.Context, *oauth.ClientSession, string, int64, string) (string, error)

type UploadAuthorizationIssuerOptions struct {
	Sessions  auth.OAuthSessionRunner
	PDSClient *http.Client
	Now       func() time.Time
	Lifetime  time.Duration
	request   serviceAuthRequest
}

type UploadAuthorizationIssuer struct {
	sessions  auth.OAuthSessionRunner
	pdsClient *http.Client
	now       func() time.Time
	lifetime  time.Duration
	request   serviceAuthRequest
}

func NewUploadAuthorizationIssuer(options UploadAuthorizationIssuerOptions) (*UploadAuthorizationIssuer, error) {
	if options.Sessions == nil || options.PDSClient == nil {
		return nil, ErrServiceAuthorizationUnavailable
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.Lifetime <= 0 || options.Lifetime > maxAuthorizationLifetime {
		options.Lifetime = maxAuthorizationLifetime
	}
	if options.request == nil {
		options.request = requestServiceAuthorization(options.PDSClient)
	}
	return &UploadAuthorizationIssuer{
		sessions: options.Sessions, pdsClient: options.PDSClient,
		now: options.Now, lifetime: options.Lifetime, request: options.request,
	}, nil
}

func (issuer *UploadAuthorizationIssuer) IssueUpload(ctx context.Context, owner syntax.DID, sessionID string) (UploadAuthorization, error) {
	if issuer == nil || issuer.sessions == nil || issuer.request == nil || owner == "" || sessionID == "" {
		return UploadAuthorization{}, ErrServiceAuthorizationUnavailable
	}
	expiresAt := time.Unix(issuer.now().UTC().Add(issuer.lifetime).Unix(), 0).UTC()
	var token string
	err := issuer.sessions.WithActiveSession(ctx, owner, sessionID, func(operationCtx context.Context, session *oauth.ClientSession) error {
		if session == nil || session.Data == nil {
			return ErrServiceAuthorizationUnavailable
		}
		audience, err := pdsAudience(session.Data.HostURL)
		if err != nil {
			return ErrServiceAuthorizationUnavailable
		}
		token, err = issuer.request(operationCtx, session, audience.String(), expiresAt.Unix(), videoUploadMethod)
		if err != nil || strings.TrimSpace(token) == "" {
			return ErrServiceAuthorizationUnavailable
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return UploadAuthorization{}, err
		}
		return UploadAuthorization{}, ErrServiceAuthorizationUnavailable
	}
	return UploadAuthorization{Token: token, ExpiresAt: expiresAt}, nil
}

func pdsAudience(rawURL string) (syntax.DID, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil ||
		(parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", ErrServiceAuthorizationUnavailable
	}
	return syntax.ParseDID("did:web:" + strings.ToLower(parsed.Hostname()))
}

func requestServiceAuthorization(client *http.Client) serviceAuthRequest {
	return func(ctx context.Context, session *oauth.ClientSession, audience string, expiry int64, method string) (string, error) {
		if session == nil || session.Data == nil || session.Config == nil || client == nil {
			return "", ErrServiceAuthorizationUnavailable
		}
		apiClient := session.APIClient()
		apiClient.Client = client
		output, err := apiatproto.ServerGetServiceAuth(ctx, apiClient, audience, expiry, method)
		if err != nil || output == nil {
			return "", ErrServiceAuthorizationUnavailable
		}
		return output.Token, nil
	}
}
