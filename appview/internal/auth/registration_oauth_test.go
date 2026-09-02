package auth

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/atcrypto"
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
)

func TestRegistrationAuthMetadataSelectsAdvertisedCreatePrompt(t *testing.T) {
	t.Parallel()

	issuer := "https://auth.registration.test"
	tests := []struct {
		name        string
		promptValue any
		wantPrompt  bool
		wantError   bool
	}{
		{name: "exact create", promptValue: []string{"create"}, wantPrompt: true},
		{name: "duplicate exact create", promptValue: []string{"create", "create"}, wantPrompt: true},
		{name: "case variants", promptValue: []string{"Create", "CREATE"}},
		{name: "absent"},
		{name: "empty", promptValue: []string{}},
		{name: "unrelated", promptValue: []string{"login", "consent"}},
		{name: "malformed scalar", promptValue: "create", wantError: true},
		{name: "malformed member", promptValue: []any{"create", 7}, wantError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			document := validRegistrationMetadataDocument(issuer)
			if test.promptValue != nil {
				document["prompt_values_supported"] = test.promptValue
			}
			body, err := json.Marshal(document)
			if err != nil {
				t.Fatalf("marshal metadata fixture: %v", err)
			}

			metadata, err := decodeRegistrationAuthMetadata(body, issuer)
			if test.wantError {
				if err == nil {
					t.Fatal("decodeRegistrationAuthMetadata accepted malformed prompt metadata")
				}
				return
			}
			if err != nil {
				t.Fatalf("decodeRegistrationAuthMetadata: %v", err)
			}
			if got := metadata.UseCreatePrompt(); got != test.wantPrompt {
				t.Fatalf("UseCreatePrompt() = %t, want %t", got, test.wantPrompt)
			}
		})
	}
}

func TestRegistrationOAuthPromptedPARDoesNotDowngrade(t *testing.T) {
	t.Parallel()

	const providerText = "provider-secret: prompt unsupported; retry without it"
	tests := []struct {
		name       string
		prompted   bool
		response   *http.Response
		transport  error
		wantCode   RegistrationOAuthFailureCode
		wantStatus int
	}{
		{
			name: "prompted generic 4xx", prompted: true,
			response: oauthErrorResponse(http.StatusBadRequest, "invalid_request", providerText),
			wantCode: RegistrationOAuthProviderUnavailable, wantStatus: http.StatusBadRequest,
		},
		{
			name: "prompted DPoP rejection", prompted: true,
			response: oauthErrorResponse(http.StatusBadRequest, "invalid_dpop_proof", providerText),
			wantCode: RegistrationOAuthProviderUnavailable, wantStatus: http.StatusBadRequest,
		},
		{
			name: "prompted scope rejection", prompted: true,
			response: oauthErrorResponse(http.StatusBadRequest, "invalid_scope", providerText),
			wantCode: RegistrationOAuthProviderUnavailable, wantStatus: http.StatusBadRequest,
		},
		{
			name:     "ordinary generic 4xx",
			response: oauthErrorResponse(http.StatusBadRequest, "invalid_request", providerText),
			wantCode: RegistrationOAuthIncomplete, wantStatus: http.StatusBadRequest,
		},
		{
			name: "timeout", prompted: true, transport: context.DeadlineExceeded,
			wantCode: RegistrationOAuthProviderUnavailable,
		},
		{
			name: "rate limited", prompted: true,
			response: oauthErrorResponse(http.StatusTooManyRequests, "slow_down", providerText),
			wantCode: RegistrationOAuthProviderUnavailable, wantStatus: http.StatusTooManyRequests,
		},
		{
			name: "server failure", prompted: true,
			response: oauthErrorResponse(http.StatusBadGateway, "server_error", providerText),
			wantCode: RegistrationOAuthProviderUnavailable, wantStatus: http.StatusBadGateway,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var attempts []url.Values
			base := roundTripFunc(func(request *http.Request) (*http.Response, error) {
				attempts = append(attempts, readPARForm(t, request))
				if test.transport != nil {
					return nil, test.transport
				}
				return test.response, nil
			})
			app := registrationTestClientApp(t, base)
			adapter, err := newRegistrationOAuthAdapter(app)
			if err != nil {
				t.Fatalf("newRegistrationOAuthAdapter: %v", err)
			}
			metadata := registrationTestMetadata(test.prompted)

			_, err = adapter.SendAuthorizationRequest(context.Background(), metadata, []string{"atproto", "transition:generic"})
			if err == nil {
				t.Fatal("SendAuthorizationRequest succeeded for failed PAR")
			}
			var failure *RegistrationOAuthError
			if !errors.As(err, &failure) {
				t.Fatalf("error type = %T, want *RegistrationOAuthError", err)
			}
			if failure.Stage != RegistrationOAuthStagePAR || failure.Code != test.wantCode ||
				failure.StatusCode != test.wantStatus || failure.Prompted != test.prompted {
				t.Fatalf("failure = %+v, want stage PAR code %q status %d prompted %t", failure, test.wantCode, test.wantStatus, test.prompted)
			}
			if strings.Contains(err.Error(), providerText) || strings.Contains(err.Error(), "invalid_scope") ||
				strings.Contains(err.Error(), "invalid_dpop_proof") || strings.Contains(err.Error(), "invalid_request") {
				t.Fatalf("error exposed or interpreted provider text: %q", err)
			}
			if len(attempts) != 1 {
				t.Fatalf("HTTP PAR attempts = %d, want exactly one", len(attempts))
			}
			wantPrompt := ""
			if test.prompted {
				wantPrompt = "create"
			}
			if got := attempts[0].Get("prompt"); got != wantPrompt {
				t.Fatalf("prompt = %q, want %q", got, wantPrompt)
			}
			if attempts[0].Get("login_hint") != "" {
				t.Fatalf("login_hint = %q, want absent", attempts[0].Get("login_hint"))
			}
		})
	}

	t.Run("one logical PAR preserves prompt and Indigo nonce replay", func(t *testing.T) {
		var forms []url.Values
		var dpop []string
		base := roundTripFunc(func(request *http.Request) (*http.Response, error) {
			forms = append(forms, readPARForm(t, request))
			dpop = append(dpop, request.Header.Get("DPoP"))
			if len(forms) == 1 {
				response := oauthErrorResponse(http.StatusBadRequest, "use_dpop_nonce", providerText)
				response.Header.Set("DPoP-Nonce", "nonce-2")
				return response, nil
			}
			response := &http.Response{
				StatusCode: http.StatusCreated,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"request_uri":"urn:example:par:1","expires_in":60}`)),
			}
			response.Header.Set("DPoP-Nonce", "nonce-2")
			return response, nil
		})
		app := registrationTestClientApp(t, base)
		adapter, err := newRegistrationOAuthAdapter(app)
		if err != nil {
			t.Fatalf("newRegistrationOAuthAdapter: %v", err)
		}

		info, err := adapter.SendAuthorizationRequest(
			context.Background(), registrationTestMetadata(true), []string{"atproto", "transition:generic"},
		)
		if err != nil {
			t.Fatalf("SendAuthorizationRequest: %v", err)
		}
		if len(forms) != 2 {
			t.Fatalf("nonce HTTP attempts = %d, want two within one logical PAR", len(forms))
		}
		for i, form := range forms {
			if form.Get("prompt") != "create" || form.Get("login_hint") != "" ||
				form.Get("code_challenge") == "" || form.Get("code_challenge_method") != "S256" ||
				form.Get("client_assertion") == "" || form.Get("client_assertion_type") != oauth.ClientAssertionJWTBearer {
				t.Fatalf("PAR form %d did not preserve prompt/PKCE/client assertion: %v", i+1, form)
			}
		}
		if forms[0].Encode() != forms[1].Encode() {
			t.Fatal("nonce replay changed the PAR form")
		}
		if dpop[0] == "" || dpop[1] == "" || dpop[0] == dpop[1] {
			t.Fatal("Indigo did not regenerate the DPoP proof for nonce replay")
		}
		if info.RequestURI != "urn:example:par:1" || info.PKCEVerifier == "" ||
			info.DPoPPrivateKeyMultibase == "" || info.DPoPAuthServerNonce != "nonce-2" {
			t.Fatalf("AuthRequestData lost Indigo state: %+v", info)
		}
	})
}

func registrationTestClientApp(t *testing.T, transport http.RoundTripper) *oauth.ClientApp {
	t.Helper()
	privateKey, err := atcrypto.GeneratePrivateKeyP256()
	if err != nil {
		t.Fatalf("generate client key: %v", err)
	}
	config := oauth.NewPublicConfig(
		"https://craftsky.test/oauth-client-metadata.json",
		"https://craftsky.test/oauth/callback",
		[]string{"atproto", "transition:generic"},
	)
	if err := config.SetClientSecret(privateKey, "registration-test-key"); err != nil {
		t.Fatalf("SetClientSecret: %v", err)
	}
	return &oauth.ClientApp{
		Client: &http.Client{Transport: transport},
		Config: &config,
	}
}

func registrationTestMetadata(prompted bool) RegistrationAuthMetadata {
	metadata := RegistrationAuthMetadata{
		AuthServerMetadata: oauth.AuthServerMetadata{
			Issuer:                             "https://auth.registration.test",
			AuthorizationEndpoint:              "https://auth.registration.test/authorize",
			TokenEndpoint:                      "https://auth.registration.test/token",
			PushedAuthorizationRequestEndpoint: "https://auth.registration.test/par",
		},
	}
	if prompted {
		metadata.PromptValuesSupported = []string{"create"}
	}
	return metadata
}

func readPARForm(t *testing.T, request *http.Request) url.Values {
	t.Helper()
	if request.Method != http.MethodPost || request.URL.String() != "https://auth.registration.test/par" ||
		request.Header.Get("Content-Type") != "application/x-www-form-urlencoded" || request.Header.Get("DPoP") == "" {
		t.Fatalf("unexpected PAR request: %s %s headers=%v", request.Method, request.URL, request.Header)
	}
	body, err := io.ReadAll(request.Body)
	if err != nil {
		t.Fatalf("read PAR body: %v", err)
	}
	form, err := url.ParseQuery(string(body))
	if err != nil {
		t.Fatalf("parse PAR form: %v", err)
	}
	return form
}

func oauthErrorResponse(status int, code, description string) *http.Response {
	body, err := json.Marshal(map[string]string{"error": code, "error_description": description})
	if err != nil {
		panic(err)
	}
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(string(body))),
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func validRegistrationMetadataDocument(issuer string) map[string]any {
	metadata := oauth.AuthServerMetadata{
		Issuer:                                     issuer,
		AuthorizationEndpoint:                      issuer + "/authorize",
		TokenEndpoint:                              issuer + "/token",
		ResponseTypesSupported:                     []string{"code"},
		GrantTypesSupported:                        []string{"authorization_code", "refresh_token"},
		CodeChallengeMethodsSupported:              []string{"S256"},
		TokenEndpointAuthMethodsSupoorted:          []string{"none", "private_key_jwt"},
		TokenEndpointAuthSigningAlgValuesSupported: []string{"ES256"},
		ScopesSupported:                            []string{"atproto"},
		AuthorizationReponseISSParameterSupported:  true,
		RequirePushedAuthorizationRequests:         true,
		PushedAuthorizationRequestEndpoint:         issuer + "/par",
		DPoPSigningAlgValuesSupported:              []string{"ES256"},
		ClientIDMetadataDocumentSupported:          true,
	}
	body, err := json.Marshal(metadata)
	if err != nil {
		panic(err)
	}
	var document map[string]any
	if err := json.Unmarshal(body, &document); err != nil {
		panic(err)
	}
	return document
}
