package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"slices"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"

	"social.craftsky/appview/internal/federatedhttp"
)

const maxRegistrationOAuthErrorBytes int64 = 64 << 10

type RegistrationOAuthStage string

const (
	RegistrationOAuthStageDiscovery RegistrationOAuthStage = "discovery"
	RegistrationOAuthStageMetadata  RegistrationOAuthStage = "metadata"
	RegistrationOAuthStagePAR       RegistrationOAuthStage = "par"
	RegistrationOAuthStageToken     RegistrationOAuthStage = "token"
)

type RegistrationOAuthFailureCode string

const (
	RegistrationOAuthProviderUnavailable RegistrationOAuthFailureCode = "providerUnavailable"
	RegistrationOAuthIncomplete          RegistrationOAuthFailureCode = "registrationIncomplete"
)

type RegistrationOAuthError struct {
	Stage             RegistrationOAuthStage
	Code              RegistrationOAuthFailureCode
	StatusCode        int
	Transient         bool
	Prompted          bool
	IssuanceUncertain bool
}

func (err *RegistrationOAuthError) Error() string {
	return "registration OAuth request failed"
}

type RegistrationAuthMetadata struct {
	oauth.AuthServerMetadata
	PromptValuesSupported []string `json:"prompt_values_supported,omitempty"`
}

func decodeRegistrationAuthMetadata(body []byte, issuer string) (RegistrationAuthMetadata, error) {
	var metadata RegistrationAuthMetadata
	if err := json.Unmarshal(body, &metadata); err != nil {
		return RegistrationAuthMetadata{}, err
	}
	if err := metadata.AuthServerMetadata.Validate(issuer); err != nil {
		return RegistrationAuthMetadata{}, err
	}
	return metadata, nil
}

func (metadata RegistrationAuthMetadata) UseCreatePrompt() bool {
	return slices.Contains(metadata.PromptValuesSupported, "create")
}

type RegistrationOAuthAdapter struct {
	app *oauth.ClientApp
}

type registrationTokenResult struct {
	response      *oauth.TokenResponse
	cleanupTokens *registrationCleanupTokens
}

type registrationCleanupTokens struct {
	accessToken  string
	refreshToken string
}

func NewRegistrationOAuthAdapter(app *oauth.ClientApp) (*RegistrationOAuthAdapter, error) {
	if app == nil || app.Client == nil || app.Client.Transport == nil || app.Config == nil {
		return nil, errors.New("registration OAuth dependencies are unavailable")
	}
	return &RegistrationOAuthAdapter{app: app}, nil
}

func (adapter *RegistrationOAuthAdapter) ResolveAuthorizationServer(
	ctx context.Context,
	providerOrigin string,
) (RegistrationAuthMetadata, error) {
	if adapter == nil || adapter.app == nil || adapter.app.Resolver == nil || adapter.app.Resolver.Client == nil {
		return RegistrationAuthMetadata{}, newRegistrationOAuthError(RegistrationOAuthStageDiscovery, 0, false)
	}
	providerURL, err := url.Parse(providerOrigin)
	if err != nil {
		return RegistrationAuthMetadata{}, newRegistrationOAuthError(RegistrationOAuthStageDiscovery, 0, false)
	}
	protectedResourceURL := "https://" + providerURL.Hostname() + "/.well-known/oauth-protected-resource"
	protectedResource, err := adapter.fetchMetadata(ctx, RegistrationOAuthStageDiscovery, protectedResourceURL)
	issuer := ""
	if err != nil {
		var failure *RegistrationOAuthError
		if !errors.As(err, &failure) || failure.StatusCode != http.StatusNotFound {
			return RegistrationAuthMetadata{}, err
		}
		// A configured provider may already be an authorization-server issuer.
		issuer = providerOrigin
	} else {
		var protectedResourceDocument struct {
			AuthorizationServers []string `json:"authorization_servers"`
		}
		if err := json.Unmarshal(protectedResource, &protectedResourceDocument); err != nil || len(protectedResourceDocument.AuthorizationServers) == 0 {
			return RegistrationAuthMetadata{}, newRegistrationOAuthError(RegistrationOAuthStageDiscovery, http.StatusOK, false)
		}
		issuer = protectedResourceDocument.AuthorizationServers[0]
	}
	issuerURL, err := url.Parse(issuer)
	if err != nil || issuerURL.Scheme != "https" || issuerURL.Hostname() == "" || issuerURL.Port() != "" {
		return RegistrationAuthMetadata{}, newRegistrationOAuthError(RegistrationOAuthStageDiscovery, http.StatusOK, false)
	}
	metadataURL := "https://" + issuerURL.Hostname() + "/.well-known/oauth-authorization-server"
	body, err := adapter.fetchMetadata(ctx, RegistrationOAuthStageMetadata, metadataURL)
	if err != nil {
		return RegistrationAuthMetadata{}, err
	}
	metadata, err := decodeRegistrationAuthMetadata(body, issuer)
	if err != nil {
		return RegistrationAuthMetadata{}, newRegistrationOAuthError(RegistrationOAuthStageMetadata, http.StatusOK, false)
	}
	return metadata, nil
}

func (adapter *RegistrationOAuthAdapter) fetchMetadata(
	ctx context.Context,
	stage RegistrationOAuthStage,
	documentURL string,
) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, documentURL, nil)
	if err != nil {
		return nil, newRegistrationOAuthError(stage, 0, false)
	}
	if adapter.app.Resolver.UserAgent != "" {
		request.Header.Set("User-Agent", adapter.app.Resolver.UserAgent)
	}
	response, err := adapter.app.Resolver.Client.Do(request)
	if err != nil {
		kind := federatedhttp.Classify(err)
		transient := kind != federatedhttp.KindDestinationRejected &&
			kind != federatedhttp.KindRedirectRejected && kind != federatedhttp.KindResponseTooLarge
		return nil, newRegistrationOAuthError(stage, 0, transient)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		transient := response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= http.StatusInternalServerError
		return nil, newRegistrationOAuthError(stage, response.StatusCode, transient)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxRegistrationOAuthErrorBytes+1))
	if err != nil || int64(len(body)) > maxRegistrationOAuthErrorBytes {
		return nil, newRegistrationOAuthError(stage, response.StatusCode, false)
	}
	return body, nil
}

func (adapter *RegistrationOAuthAdapter) SendAuthorizationRequest(
	ctx context.Context,
	metadata RegistrationAuthMetadata,
	scopes []string,
) (*oauth.AuthRequestData, error) {
	appCopy := *adapter.app
	clientCopy := *adapter.app.Client
	next := clientCopy.Transport
	clientCopy.Transport = &registrationOAuthTransport{
		next:     next,
		parURL:   metadata.PushedAuthorizationRequestEndpoint,
		prompted: metadata.UseCreatePrompt(),
	}
	appCopy.Client = &clientCopy
	return appCopy.SendAuthRequest(ctx, &metadata.AuthServerMetadata, scopes, "")
}

func (adapter *RegistrationOAuthAdapter) SendInitialTokenRequest(
	ctx context.Context,
	authorizationCode string,
	request oauth.AuthRequestData,
) (registrationTokenResult, error) {
	appCopy := *adapter.app
	clientCopy := *adapter.app.Client
	transport := &registrationTokenTransport{
		next: clientCopy.Transport, tokenURL: request.AuthServerTokenEndpoint,
	}
	clientCopy.Transport = transport
	appCopy.Client = &clientCopy
	token, err := appCopy.SendInitialTokenRequest(ctx, authorizationCode, request)
	if err != nil && transport.receivedSuccess {
		return registrationTokenResult{cleanupTokens: transport.cleanupTokens},
			newRegistrationTokenError(http.StatusOK, false, true)
	}
	return registrationTokenResult{response: token}, err
}

func newRegistrationOAuthAdapter(app *oauth.ClientApp) (*RegistrationOAuthAdapter, error) {
	return NewRegistrationOAuthAdapter(app)
}

type registrationOAuthTransport struct {
	next     http.RoundTripper
	parURL   string
	prompted bool
}

type registrationTokenTransport struct {
	next            http.RoundTripper
	tokenURL        string
	receivedSuccess bool
	cleanupTokens   *registrationCleanupTokens
}

func (transport *registrationTokenTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if transport == nil || transport.next == nil || request == nil || request.URL == nil ||
		request.Method != http.MethodPost || request.URL.String() != transport.tokenURL {
		return nil, newRegistrationTokenError(0, false, false)
	}
	response, err := transport.next.RoundTrip(request)
	if err != nil {
		return nil, newRegistrationTokenError(0, true, true)
	}
	if response == nil || response.Body == nil {
		return nil, newRegistrationTokenError(0, true, true)
	}
	if response.StatusCode == http.StatusOK {
		transport.receivedSuccess = true
		body, readErr := io.ReadAll(io.LimitReader(response.Body, maxRegistrationOAuthErrorBytes+1))
		_ = response.Body.Close()
		if readErr != nil || int64(len(body)) > maxRegistrationOAuthErrorBytes {
			return nil, newRegistrationTokenError(http.StatusOK, false, true)
		}
		transport.cleanupTokens = decodeRegistrationCleanupTokens(body)
		response.Body = io.NopCloser(bytes.NewReader(body))
		response.ContentLength = int64(len(body))
		return response, nil
	}
	status := response.StatusCode
	if status == http.StatusBadRequest && response.Header.Get("DPoP-Nonce") != "" &&
		isRegistrationDPoPNonceError(response) {
		return response, nil
	}
	discardRegistrationOAuthResponse(response)
	transient := status == http.StatusTooManyRequests || status >= http.StatusInternalServerError
	return nil, newRegistrationTokenError(status, transient, transient)
}

func decodeRegistrationCleanupTokens(body []byte) *registrationCleanupTokens {
	var payload struct {
		AccessToken  json.RawMessage `json:"access_token"`
		RefreshToken json.RawMessage `json:"refresh_token"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil
	}
	var tokens registrationCleanupTokens
	_ = json.Unmarshal(payload.AccessToken, &tokens.accessToken)
	_ = json.Unmarshal(payload.RefreshToken, &tokens.refreshToken)
	if tokens.accessToken == "" && tokens.refreshToken == "" {
		return nil
	}
	return &tokens
}

func (transport *registrationOAuthTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if transport == nil || transport.next == nil || request == nil || request.URL == nil ||
		request.Method != http.MethodPost || request.URL.String() != transport.parURL {
		return nil, newRegistrationPARError(0, transport != nil && transport.prompted)
	}

	body, err := io.ReadAll(io.LimitReader(request.Body, maxRegistrationOAuthErrorBytes+1))
	if err != nil || int64(len(body)) > maxRegistrationOAuthErrorBytes {
		return nil, newRegistrationPARError(0, transport.prompted)
	}
	_ = request.Body.Close()
	form, err := url.ParseQuery(string(body))
	if err != nil || form.Get("login_hint") != "" || form.Get("prompt") != "" {
		return nil, newRegistrationPARError(0, transport.prompted)
	}
	if transport.prompted {
		form.Set("prompt", "create")
	}
	request = request.Clone(request.Context())
	setRegistrationOAuthBody(request, []byte(form.Encode()))

	response, err := transport.next.RoundTrip(request)
	if err != nil {
		return nil, newRegistrationPARError(0, transport.prompted)
	}
	if response == nil || response.Body == nil {
		return nil, newRegistrationPARError(0, transport.prompted)
	}
	if response.StatusCode == http.StatusOK || response.StatusCode == http.StatusCreated {
		return response, nil
	}

	status := response.StatusCode
	if status == http.StatusBadRequest && response.Header.Get("DPoP-Nonce") != "" {
		if isRegistrationDPoPNonceError(response) {
			return response, nil
		}
	} else {
		discardRegistrationOAuthResponse(response)
	}
	return nil, newRegistrationPARError(status, transport.prompted)
}

func setRegistrationOAuthBody(request *http.Request, body []byte) {
	request.Body = io.NopCloser(bytes.NewReader(body))
	request.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
	request.ContentLength = int64(len(body))
}

func isRegistrationDPoPNonceError(response *http.Response) bool {
	body, err := io.ReadAll(io.LimitReader(response.Body, maxRegistrationOAuthErrorBytes+1))
	_ = response.Body.Close()
	if err != nil || int64(len(body)) > maxRegistrationOAuthErrorBytes {
		return false
	}
	var payload struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &payload); err != nil || payload.Error != "use_dpop_nonce" {
		return false
	}
	sanitized := []byte(`{"error":"use_dpop_nonce"}`)
	response.Body = io.NopCloser(bytes.NewReader(sanitized))
	response.ContentLength = int64(len(sanitized))
	return true
}

func discardRegistrationOAuthResponse(response *http.Response) {
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, maxRegistrationOAuthErrorBytes+1))
	_ = response.Body.Close()
}

func newRegistrationPARError(status int, prompted bool) *RegistrationOAuthError {
	transient := status == 0 || status == http.StatusTooManyRequests || status >= http.StatusInternalServerError
	code := RegistrationOAuthIncomplete
	if prompted || transient {
		code = RegistrationOAuthProviderUnavailable
	}
	return &RegistrationOAuthError{
		Stage: RegistrationOAuthStagePAR, Code: code, StatusCode: status,
		Transient: transient, Prompted: prompted,
	}
}

func newRegistrationTokenError(status int, transient, issuanceUncertain bool) *RegistrationOAuthError {
	code := RegistrationOAuthIncomplete
	if transient {
		code = RegistrationOAuthProviderUnavailable
	}
	return &RegistrationOAuthError{
		Stage: RegistrationOAuthStageToken, Code: code, StatusCode: status,
		Transient: transient, IssuanceUncertain: issuanceUncertain,
	}
}

func newRegistrationOAuthError(stage RegistrationOAuthStage, status int, transient bool) *RegistrationOAuthError {
	code := RegistrationOAuthIncomplete
	if transient {
		code = RegistrationOAuthProviderUnavailable
	}
	return &RegistrationOAuthError{
		Stage: stage, Code: code, StatusCode: status, Transient: transient,
	}
}

var _ error = (*RegistrationOAuthError)(nil)
