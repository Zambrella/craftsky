package federatedhttp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const (
	protectedResourceMetadataPath = "/.well-known/oauth-protected-resource"
	authorizationMetadataPath     = "/.well-known/oauth-authorization-server"
)

// OAuthMetadataClient returns the metadata-purpose client with an additional
// response boundary. The Indigo resolver validates the metadata document's
// feature flags, while this transport validates every metadata-derived
// destination before Indigo can use it for PAR, token, refresh, or revocation
// requests.
func (b *Boundary) OAuthMetadataClient(profile Profile) (*http.Client, error) {
	if profile.Purpose != PurposeOAuthMetadata {
		return nil, fmt.Errorf("federated http: OAuth metadata client requires metadata purpose")
	}
	client, err := b.Client(profile)
	if err != nil {
		return nil, err
	}
	client.Transport = &oauthMetadataTransport{
		next:   client.Transport,
		policy: b.policy,
	}
	return client, nil
}

type oauthMetadataTransport struct {
	next   http.RoundTripper
	policy *Policy
}

func (transport *oauthMetadataTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if transport == nil || transport.next == nil || transport.policy == nil ||
		request == nil || request.URL == nil {
		return nil, metadataError(KindDestinationRejected, nil)
	}
	response, err := transport.next.RoundTrip(request)
	if err != nil {
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}
		return nil, err
	}
	if response == nil || response.Body == nil || response.StatusCode != http.StatusOK {
		return response, nil
	}
	if request.URL.Path != protectedResourceMetadataPath &&
		request.URL.Path != authorizationMetadataPath {
		return response, nil
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, MaxOAuthMetadataResponseBytes+1))
	closeErr := response.Body.Close()
	if err != nil {
		return nil, metadataError(KindUpstreamFailure, err)
	}
	if closeErr != nil {
		return nil, metadataError(KindUpstreamFailure, closeErr)
	}
	if int64(len(body)) > MaxOAuthMetadataResponseBytes {
		return nil, metadataError(KindResponseTooLarge, nil)
	}
	if err := transport.validate(request, body); err != nil {
		return nil, err
	}
	response.Body = io.NopCloser(bytes.NewReader(body))
	response.ContentLength = int64(len(body))
	return response, nil
}

func (transport *oauthMetadataTransport) validate(request *http.Request, body []byte) error {
	switch request.URL.Path {
	case protectedResourceMetadataPath:
		var metadata struct {
			AuthorizationServers []string `json:"authorization_servers"`
		}
		if err := json.Unmarshal(body, &metadata); err != nil {
			return metadataError(KindUpstreamFailure, err)
		}
		if len(metadata.AuthorizationServers) == 0 {
			return metadataError(KindDestinationRejected, nil)
		}
		for _, server := range metadata.AuthorizationServers {
			if _, err := transport.policy.ValidateOrigin(request.Context(), server); err != nil {
				return metadataError(Classify(err), err)
			}
		}
		return nil

	case authorizationMetadataPath:
		var metadata struct {
			Issuer                             string `json:"issuer"`
			AuthorizationEndpoint              string `json:"authorization_endpoint"`
			TokenEndpoint                      string `json:"token_endpoint"`
			PushedAuthorizationRequestEndpoint string `json:"pushed_authorization_request_endpoint"`
			RevocationEndpoint                 string `json:"revocation_endpoint"`
		}
		if err := json.Unmarshal(body, &metadata); err != nil {
			return metadataError(KindUpstreamFailure, err)
		}
		requestOrigin := (&url.URL{Scheme: request.URL.Scheme, Host: request.URL.Host}).String()
		issuer, err := transport.policy.ValidateOrigin(request.Context(), metadata.Issuer)
		if err != nil {
			return metadataError(Classify(err), err)
		}
		if issuer.String() != requestOrigin {
			return metadataError(KindDestinationRejected, nil)
		}
		endpoints := []string{
			metadata.AuthorizationEndpoint,
			metadata.TokenEndpoint,
			metadata.PushedAuthorizationRequestEndpoint,
		}
		if metadata.RevocationEndpoint != "" {
			endpoints = append(endpoints, metadata.RevocationEndpoint)
		}
		for _, endpoint := range endpoints {
			if endpoint == "" {
				return metadataError(KindDestinationRejected, nil)
			}
			if _, err := transport.policy.ValidateOAuthEndpoint(
				request.Context(), metadata.Issuer, endpoint,
			); err != nil {
				return metadataError(Classify(err), err)
			}
		}
		return nil
	default:
		return nil
	}
}

func metadataError(kind Kind, cause error) error {
	return &Error{Kind: kind, Purpose: PurposeOAuthMetadata, Cause: cause}
}
