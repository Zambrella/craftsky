package auth

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/testdb"
)

func TestAccountDeletionReauthenticationIsFreshOwnerBoundAndSingleUse(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 10, 15, 0, 0, 0, time.UTC)
	intent := AccountDeletionReauthIntent{
		JobID:          "job-alice",
		Owner:          syntax.DID("did:plc:alice"),
		ExpectedHandle: "alice.test",
		IssuedAt:       now,
		ExpiresAt:      now.Add(10 * time.Minute),
	}
	completion, err := CompleteAccountDeletionReauth(
		intent,
		intent.Owner,
		"oauth-session-fresh",
		"one-time-proof-secret",
		now.Add(time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	if completion.JobID != intent.JobID || completion.Owner != intent.Owner || completion.OAuthSessionID != "oauth-session-fresh" {
		t.Fatalf("completion binding = %+v", completion)
	}
	if string(completion.ProofHash[:]) == "one-time-proof-secret" {
		t.Fatal("reauthentication proof must be stored only as a one-way hash")
	}

	sessionID, err := ConsumeAccountDeletionReauth(intent, &completion, "one-time-proof-secret", "alice.test", now.Add(2*time.Minute))
	if err != nil || sessionID != completion.OAuthSessionID {
		t.Fatalf("matching proof consumption = (%q, %v)", sessionID, err)
	}
	if _, err := ConsumeAccountDeletionReauth(intent, &completion, "one-time-proof-secret", "alice.test", now.Add(2*time.Minute)); !errors.Is(err, ErrDeletionReauthReplayed) {
		t.Fatalf("replayed proof error = %v", err)
	}

	fresh, err := CompleteAccountDeletionReauth(intent, intent.Owner, "oauth-session-2", "proof-2", now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ConsumeAccountDeletionReauth(intent, &fresh, "proof-2", "Alice.test", now.Add(2*time.Minute)); !errors.Is(err, ErrDeletionConfirmationHandleMismatch) {
		t.Fatalf("handle mismatch error = %v", err)
	}
	if fresh.Consumed {
		t.Fatal("handle mismatch must not consume or advance the proof")
	}

	for _, test := range []struct {
		name        string
		intent      AccountDeletionReauthIntent
		callbackDID syntax.DID
		at          time.Time
	}{
		{name: "wrong DID", intent: intent, callbackDID: syntax.DID("did:plc:bob"), at: now.Add(time.Minute)},
		{name: "stale", intent: intent, callbackDID: intent.Owner, at: now.Add(-time.Second)},
		{name: "expired", intent: intent, callbackDID: intent.Owner, at: intent.ExpiresAt},
		{name: "canceled", intent: AccountDeletionReauthIntent{JobID: intent.JobID, Owner: intent.Owner, ExpectedHandle: intent.ExpectedHandle, IssuedAt: intent.IssuedAt, ExpiresAt: intent.ExpiresAt, Canceled: true}, callbackDID: intent.Owner, at: now.Add(time.Minute)},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := CompleteAccountDeletionReauth(test.intent, test.callbackDID, "oauth-rejected", "proof-rejected", test.at); !errors.Is(err, ErrDeletionReauthenticationRequired) {
				t.Fatalf("completion error = %v, want ErrDeletionReauthenticationRequired", err)
			}
		})
	}
}

func TestOAuthCallbackUsesDeletionOnlyPurposeWithoutMintingOrdinaryAccess(t *testing.T) {
	t.Parallel()

	owner := syntax.DID("did:plc:alice")
	callbacks := &recordingDeletionOAuthCallbacks{
		metadata: AccountDeletionAuthRequest{
			Purpose: AccountDeletionOAuthPurpose,
			JobID:   "job-alice",
			Owner:   owner,
		},
	}
	pdsCreated := false
	handlers := &HTTPHandlers{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		ProcessOAuthCallback: func(context.Context, url.Values) (*oauth.ClientSessionData, error) {
			return &oauth.ClientSessionData{AccountDID: owner, SessionID: "oauth-session-fresh"}, nil
		},
		DeletionOAuthCallbacks: callbacks,
		NewPDSClient: func(context.Context, syntax.DID, string) (PDSClient, error) {
			pdsCreated = true
			return nil, errors.New("ordinary PDS client must not be created")
		},
	}

	request := httptest.NewRequest(http.MethodGet, "/oauth/callback?state=deletion-state&code=synthetic", nil)
	response := httptest.NewRecorder()
	handlers.CallbackHandler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("deletion callback status = %d, body = %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "craftsky:///account-deletion/reauth-complete") ||
		!strings.Contains(response.Body.String(), "job-id=job-alice") ||
		!strings.Contains(response.Body.String(), "proof=proof-secret") {
		t.Fatalf("deletion callback body = %s", response.Body.String())
	}
	if pdsCreated {
		t.Fatal("deletion callback created an ordinary PDS client")
	}
	if callbacks.completedSession != "oauth-session-fresh" || callbacks.completedDID != owner {
		t.Fatalf("deletion completion = DID %q session %q", callbacks.completedDID, callbacks.completedSession)
	}

	callbacks.callbackDIDMismatch = true
	request = httptest.NewRequest(http.MethodGet, "/oauth/callback?state=deletion-state&code=synthetic", nil)
	response = httptest.NewRecorder()
	handlers.CallbackHandler().ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || callbacks.rejectedSession != "oauth-session-fresh" {
		t.Fatalf("cross-DID callback status = %d rejected = %q", response.Code, callbacks.rejectedSession)
	}
}

func TestNormalOAuthCallbackReturnsCoarsePendingDeletionWithoutOrdinaryAccess(t *testing.T) {
	owner := syntax.DID("did:plc:alice")
	pool := testdb.WithSchema(t, `
		CREATE TABLE oauth_auth_requests(
			state TEXT PRIMARY KEY,
			data JSONB NOT NULL,
			handoff_mode TEXT NOT NULL DEFAULT 'deep_link',
			loopback_redirect_uri TEXT,
			device_id TEXT
		)
	`)
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO oauth_auth_requests(state,data,handoff_mode)
		VALUES('login-state','{}','deep_link')
	`); err != nil {
		t.Fatal(err)
	}
	policy := &recordingPendingLoginPolicy{}
	pdsCreated := false
	handlers := &HTTPHandlers{
		Pool: pool, Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		ProcessOAuthCallback: func(context.Context, url.Values) (*oauth.ClientSessionData, error) {
			return &oauth.ClientSessionData{AccountDID: owner, SessionID: "ordinary-oauth-must-be-deleted"}, nil
		},
		DeletionPendingLogin: policy,
		NewPDSClient: func(context.Context, syntax.DID, string) (PDSClient, error) {
			pdsCreated = true
			return nil, errors.New("pending deletion must not initialize membership")
		},
	}

	request := httptest.NewRequest(http.MethodGet, "/oauth/callback?state=login-state&code=synthetic", nil)
	response := httptest.NewRecorder()
	handlers.CallbackHandler().ServeHTTP(response, request)

	if response.Code != http.StatusOK ||
		!strings.Contains(response.Body.String(), "craftsky:///auth/complete") ||
		!strings.Contains(response.Body.String(), "account_deletion_pending") {
		t.Fatalf("pending callback status = %d body = %s", response.Code, response.Body.String())
	}
	if pdsCreated || policy.rejectedSession != "" || policy.sessionID != "ordinary-oauth-must-be-deleted" {
		t.Fatalf("pending callback pdsCreated=%v rejected=%q", pdsCreated, policy.rejectedSession)
	}
}

type recordingPendingLoginPolicy struct {
	result          AccountDeletionPendingLogin
	rejectedSession string
	sessionID       string
}

func (policy *recordingPendingLoginPolicy) PendingLogin(_ context.Context, _ syntax.DID, sessionID, _ string) (AccountDeletionPendingLogin, bool, error) {
	policy.sessionID = sessionID
	return policy.result, true, nil
}

func (policy *recordingPendingLoginPolicy) Reject(_ context.Context, _ syntax.DID, sessionID string) error {
	policy.rejectedSession = sessionID
	return nil
}

type recordingDeletionOAuthCallbacks struct {
	metadata            AccountDeletionAuthRequest
	callbackDIDMismatch bool
	completedDID        syntax.DID
	completedSession    string
	rejectedSession     string
}

func (callbacks *recordingDeletionOAuthCallbacks) RequestForState(context.Context, string) (AccountDeletionAuthRequest, bool, error) {
	return callbacks.metadata, true, nil
}

func (callbacks *recordingDeletionOAuthCallbacks) Complete(_ context.Context, request AccountDeletionAuthRequest, did syntax.DID, sessionID string) (AccountDeletionOAuthResult, error) {
	callbacks.completedDID = did
	callbacks.completedSession = sessionID
	if callbacks.callbackDIDMismatch || did != request.Owner {
		return AccountDeletionOAuthResult{}, ErrDeletionReauthenticationRequired
	}
	return AccountDeletionOAuthResult{JobID: request.JobID, Proof: "proof-secret"}, nil
}

func (callbacks *recordingDeletionOAuthCallbacks) Reject(_ context.Context, _ syntax.DID, sessionID string) error {
	callbacks.rejectedSession = sessionID
	return nil
}
