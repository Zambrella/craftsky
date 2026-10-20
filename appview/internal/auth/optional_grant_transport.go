package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"

	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
)

var errBaseScopesLost = errors.New("required base OAuth grant was lost")

// The pinned Indigo refresh implementation updates tokens but omits the
// response's effective scope. Track it before Indigo's credential persistence
// callback, then gate every resource dispatch, including SDK challenge retries.
// This transport is scoped to one fenced operation and adds no retries.
type optionalGrantTransport struct {
	base     http.RoundTripper
	session  *oauth.ClientSession
	required []string
	mu       sync.RWMutex
	scopes   []string
}

func trackOptionalGrant(session *oauth.ClientSession, required []string) {
	client := *session.Client
	base := client.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	client.Transport = &optionalGrantTransport{
		base: base, session: session,
		required: slices.Clone(required), scopes: slices.Clone(session.Data.Scopes),
	}
	session.Client = &client
}

func (transport *optionalGrantTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	isToken := request.Method == http.MethodPost && request.URL.String() == transport.session.Data.AuthServerTokenEndpoint
	isRevocation := request.Method == http.MethodPost && request.URL.String() == transport.session.Data.AuthServerRevocationEndpoint
	if !isToken && !isRevocation {
		if err := transport.checkGrant(); err != nil {
			return nil, err
		}
	}
	response, err := transport.base.RoundTrip(request)
	if err != nil || !isToken || response.StatusCode != http.StatusOK {
		return response, err
	}
	const maximumTokenResponse = 64 << 10
	body, readErr := io.ReadAll(io.LimitReader(response.Body, maximumTokenResponse+1))
	closeErr := response.Body.Close()
	if readErr != nil {
		return nil, readErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if len(body) > maximumTokenResponse {
		return nil, errors.New("OAuth refresh response exceeds size limit")
	}
	var grant struct {
		Scope *string `json:"scope"`
	}
	if err := json.Unmarshal(body, &grant); err != nil {
		return nil, err
	}
	if grant.Scope != nil {
		scopes := strings.Split(*grant.Scope, " ")
		transport.mu.Lock()
		transport.scopes = scopes
		transport.mu.Unlock()
		// RefreshTokens holds the SDK session mutex across this transport and
		// its persistence callback. Publish grants with the rotated tokens.
		transport.session.Data.Scopes = slices.Clone(scopes)
	}
	response.Body = io.NopCloser(bytes.NewReader(body))
	return response, nil
}

func (transport *optionalGrantTransport) checkGrant() error {
	transport.mu.RLock()
	raw := strings.Join(transport.scopes, " ")
	transport.mu.RUnlock()
	if !validRegistrationTokenScopes(raw, transport.session.Config.Scopes) {
		return errors.Join(errBaseScopesLost, ErrPDSSessionExpired, ErrPDSPermissionsRequired)
	}
	if !validRegistrationTokenScopes(raw, transport.required) {
		return &optionalCapabilityFailure{cause: ErrPDSPermissionsRequired}
	}
	return nil
}

type optionalResourceTransport struct {
	grant *optionalGrantTransport
	base  http.RoundTripper
}

func (transport optionalResourceTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if err := transport.grant.checkGrant(); err != nil {
		return nil, err
	}
	return transport.base.RoundTrip(request)
}

// ResourceClient retains the purpose-specific destination/body/redirect policy
// while sharing the grant updated by the separate OAuth refresh transport.
func (transport *optionalGrantTransport) ResourceClient(purpose *http.Client) *http.Client {
	client := *purpose
	base := client.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	client.Transport = optionalResourceTransport{grant: transport, base: base}
	return &client
}

func CapabilityAPIClient(session *oauth.ClientSession, purpose *http.Client) (*atclient.APIClient, error) {
	if session == nil || session.Client == nil || purpose == nil {
		return nil, ErrPDSPermissionsRequired
	}
	grant, ok := session.Client.Transport.(*optionalGrantTransport)
	if !ok {
		return nil, ErrPDSPermissionsRequired
	}
	client := session.APIClient()
	client.Client = grant.ResourceClient(purpose)
	return client, nil
}
