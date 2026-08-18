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
	"github.com/google/uuid"
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
		OAuthFlow: &recordingDeletionOAuthFlow{result: OAuthCallbackResult{
			Session: oauth.ClientSessionData{AccountDID: owner, SessionID: "oauth-session-fresh"},
			Metadata: AuthRequestMetadata{
				Purpose: AccountDeletionOAuthPurpose, Owner: owner,
				JobID: uuid.MustParse("10000000-0000-4000-8000-000000000001"),
			},
			Attempt: CallbackAttempt{State: "oauth-session-fresh", Owner: owner, Purpose: AccountDeletionOAuthPurpose},
		}},
		DeletionOAuthCallbacks: callbacks,
		DeletionCompleteURL:    "https://craftsky.social/account-deletion/reauth-complete",
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
	if !strings.Contains(response.Body.String(), "https://craftsky.social/account-deletion/reauth-complete") ||
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
	if response.Code != http.StatusBadRequest || callbacks.rejectedSession != "" {
		t.Fatalf("cross-DID callback status = %d rejected = %q", response.Code, callbacks.rejectedSession)
	}
}

func TestLoginOAuthCallbackFailsClosedBeforeOrdinaryAccessWhenOwnerPendingDeletion(t *testing.T) {
	pdsCreated := false
	handlers := &HTTPHandlers{
		Logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		OAuthFlow: &recordingDeletionOAuthFlow{err: ErrOAuthOwnerIneligible},
		NewPDSClient: func(context.Context, syntax.DID, string) (PDSClient, error) {
			pdsCreated = true
			return nil, errors.New("pending deletion must not initialize membership")
		},
	}

	request := httptest.NewRequest(http.MethodGet, "/oauth/callback?state=login-state&code=synthetic", nil)
	response := httptest.NewRecorder()
	handlers.CallbackHandler().ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest ||
		!strings.Contains(response.Body.String(), "Sign-in could not be completed") {
		t.Fatalf("pending callback status = %d body = %s", response.Code, response.Body.String())
	}
	if pdsCreated {
		t.Fatal("pending-deletion callback created an ordinary PDS client")
	}
}

type recordingDeletionOAuthFlow struct {
	result OAuthCallbackResult
	err    error
}

func (*recordingDeletionOAuthFlow) StartLogin(context.Context, syntax.Handle, HandoffMode, string, string) (string, error) {
	return "", errors.New("unexpected login start")
}

func (flow *recordingDeletionOAuthFlow) CompleteCallback(
	ctx context.Context,
	_ url.Values,
	finalize OAuthCallbackFinalizer,
) error {
	if flow.err != nil {
		return flow.err
	}
	return finalize(ctx, flow.result)
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

func (callbacks *recordingDeletionOAuthCallbacks) CompleteAttempt(
	_ context.Context,
	request AccountDeletionAuthRequest,
	attempt CallbackAttempt,
) (AccountDeletionOAuthResult, error) {
	callbacks.completedDID = attempt.Owner
	callbacks.completedSession = attempt.State
	if callbacks.callbackDIDMismatch || attempt.Owner != request.Owner {
		return AccountDeletionOAuthResult{}, ErrDeletionReauthenticationRequired
	}
	return AccountDeletionOAuthResult{JobID: callbacks.metadata.JobID, Proof: "proof-secret"}, nil
}

func (callbacks *recordingDeletionOAuthCallbacks) Reject(_ context.Context, _ syntax.DID, sessionID string) error {
	callbacks.rejectedSession = sessionID
	return nil
}
