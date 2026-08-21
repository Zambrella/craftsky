package auth

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
)

type devSchemePDSClient struct{}

func (devSchemePDSClient) GetRecord(context.Context, syntax.DID, string, string, any) (string, error) {
	return "", ErrRecordNotFound
}
func (devSchemePDSClient) PutRecord(context.Context, syntax.DID, string, string, any) error {
	return nil
}
func (devSchemePDSClient) CreateRecord(context.Context, syntax.DID, string, any) (syntax.ATURI, syntax.CID, error) {
	return "", "", nil
}
func (devSchemePDSClient) DeleteRecord(context.Context, syntax.DID, string, string) error {
	return nil
}
func (devSchemePDSClient) UploadBlob(context.Context, string, []byte) (*UploadedBlob, error) {
	return nil, nil
}

type devSchemeOnboardingWriter struct{}

func (devSchemeOnboardingWriter) PutOnboardingProfile(context.Context, PDSClient, OnboardingProfileWrite) error {
	return nil
}

func TestDevelopmentSchemeCallbacksUseOnlyFixedCodeAndDeletionProofURLs(t *testing.T) {
	owner := syntax.DID("did:plc:dev-scheme")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("login", func(t *testing.T) {
		handlers := &HTTPHandlers{
			Logger:         logger,
			AllowDevScheme: true,
			OAuthFlow: &recordingDeletionOAuthFlow{result: OAuthCallbackResult{
				Handle: syntax.Handle("alice.example"),
				Metadata: AuthRequestMetadata{
					Purpose: LoginOAuthPurpose, Owner: owner, OwnerGeneration: 1,
					AuthEpoch: 1, HandoffMode: HandoffDevScheme, DeviceID: "device-dev",
				},
				Attempt: CallbackAttempt{
					State: "oauth-dev", AttemptID: uuid.New(), Owner: owner,
					OwnerGeneration: 1, AuthEpoch: 1, Purpose: LoginOAuthPurpose,
				},
			}},
			NewPendingPDSClient: func(context.Context, CallbackAttempt) (PDSClient, error) {
				return devSchemePDSClient{}, nil
			},
			OnboardingProfile: devSchemeOnboardingWriter{},
			Handoffs:          &fakeHandoffCoordinator{},
		}

		response := httptest.NewRecorder()
		handlers.CallbackHandler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/oauth/callback?state=dev&code=synthetic", nil))
		if response.Code != http.StatusOK {
			t.Fatalf("development login callback status = %d, body=%s", response.Code, response.Body.String())
		}
		body := response.Body.String()
		if !strings.Contains(body, "craftsky-dev:///auth/complete?code=callback-code") {
			t.Fatalf("development login callback body = %s", body)
		}
		if strings.Contains(body, "token=") || strings.Contains(body, "Bearer ") {
			t.Fatalf("development login callback exposed a bearer: %s", body)
		}
	})

	t.Run("account deletion", func(t *testing.T) {
		callbacks := &recordingDeletionOAuthCallbacks{metadata: AccountDeletionAuthRequest{
			Purpose: AccountDeletionOAuthPurpose, JobID: "job-dev", Owner: owner,
		}}
		handlers := &HTTPHandlers{
			Logger:         logger,
			AllowDevScheme: true,
			OAuthFlow: &recordingDeletionOAuthFlow{result: OAuthCallbackResult{
				Session: oauthSession(owner, "oauth-deletion-dev"),
				Metadata: AuthRequestMetadata{
					Purpose: AccountDeletionOAuthPurpose, Owner: owner,
					JobID: uuid.MustParse("10000000-0000-4000-8000-000000000001"),
				},
				Attempt: CallbackAttempt{State: "oauth-deletion-dev", Owner: owner, Purpose: AccountDeletionOAuthPurpose},
			}},
			DeletionOAuthCallbacks: callbacks,
		}

		response := httptest.NewRecorder()
		handlers.CallbackHandler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/oauth/callback?state=dev-deletion&code=synthetic", nil))
		if response.Code != http.StatusOK {
			t.Fatalf("development deletion callback status = %d, body=%s", response.Code, response.Body.String())
		}
		body := response.Body.String()
		if !strings.Contains(body, "craftsky-dev:///account-deletion/reauth-complete?job-id=job-dev") ||
			!strings.Contains(body, "proof=proof-secret") {
			t.Fatalf("development deletion callback body = %s", body)
		}
		if strings.Contains(body, "token=") || strings.Contains(body, "Bearer ") {
			t.Fatalf("development deletion callback exposed a bearer: %s", body)
		}
	})
}

func oauthSession(owner syntax.DID, sessionID string) oauth.ClientSessionData {
	return oauth.ClientSessionData{AccountDID: owner, SessionID: sessionID}
}
