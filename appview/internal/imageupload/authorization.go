// Package imageupload authorizes immediate client-to-PDS image transfers.
package imageupload

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	apiatproto "github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/auth"
)

const Lifetime = 5 * time.Minute
const uploadMethod = "com.atproto.repo.uploadBlob"

var ErrUnavailable = errors.New("image upload authorization unavailable")

type Authorization struct {
	Token          string
	ExpiresAt      time.Time
	UploadEndpoint string
}

// String excludes the bearer capability from debug formatting.
func (Authorization) GoString() string { return "ImageUploadAuthorization(<private>)" }

func (Authorization) String() string { return "ImageUploadAuthorization(<private>)" }

type serviceAuthRequest func(context.Context, *oauth.ClientSession, string, int64, string) (string, error)
type Issuer struct {
	sessions auth.OAuthSessionRunner
	now      func() time.Time
	request  serviceAuthRequest
}

func NewIssuer(sessions auth.OAuthSessionRunner, client *http.Client) (*Issuer, error) {
	if sessions == nil || client == nil {
		return nil, ErrUnavailable
	}
	return &Issuer{sessions: sessions, now: time.Now, request: func(ctx context.Context, session *oauth.ClientSession, audience string, expiry int64, method string) (string, error) {
		if session.Config == nil {
			return "", ErrUnavailable
		}
		apiClient := session.APIClient()
		apiClient.Client = client
		output, err := apiatproto.ServerGetServiceAuth(ctx, apiClient, audience, expiry, method)
		if err != nil {
			return "", fmt.Errorf("request image upload authorization: %w", err)
		}
		if output == nil {
			return "", ErrUnavailable
		}
		return output.Token, nil
	}}, nil
}

func (issuer *Issuer) IssueUpload(ctx context.Context, owner syntax.DID, sessionID string) (Authorization, error) {
	if issuer == nil || issuer.sessions == nil || issuer.request == nil || owner == "" || sessionID == "" {
		return Authorization{}, ErrUnavailable
	}
	var result Authorization
	err := issuer.sessions.WithActiveSession(ctx, owner, sessionID, func(operationCtx context.Context, session *oauth.ClientSession) error {
		if session == nil || session.Data == nil || session.Data.AccountDID != owner {
			return ErrUnavailable
		}
		origin, err := url.Parse(session.Data.HostURL)
		if err != nil || origin.Scheme != "https" || origin.Hostname() == "" || origin.User != nil || (origin.Path != "" && origin.Path != "/") || origin.RawQuery != "" || origin.Fragment != "" {
			return ErrUnavailable
		}
		// Match the PDS audience used by the existing video authorization flow.
		audience, err := syntax.ParseDID("did:web:" + strings.ToLower(origin.Hostname()))
		if err != nil {
			return ErrUnavailable
		}
		result.ExpiresAt = time.Unix(issuer.now().Add(Lifetime).Unix(), 0).UTC()
		token, err := issuer.request(operationCtx, session, audience.String(), result.ExpiresAt.Unix(), uploadMethod)
		if err != nil {
			return err
		}
		if strings.TrimSpace(token) == "" {
			return ErrUnavailable
		}
		origin.Path = "/xrpc/com.atproto.repo.uploadBlob"
		result.Token, result.UploadEndpoint = token, origin.String()
		return nil
	})
	if err != nil {
		return Authorization{}, fmt.Errorf("authorize image upload: %w", err)
	}
	return result, nil
}
