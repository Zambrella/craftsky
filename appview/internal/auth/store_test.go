package auth_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/federatedhttp"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/testdb"
)

// authSchemaDDL is the full OAuth + Craftsky-sessions schema used by every
// test in this package.
//
// IMPORTANT: includes the sibling columns on oauth_auth_requests
// (handoff_mode, loopback_redirect_uri) per Appendix A's decision.
const authSchemaDDL = `
	CREATE TABLE craftsky_profiles (
		did TEXT PRIMARY KEY,
		created_at TIMESTAMPTZ NOT NULL DEFAULT now()
	);
	CREATE TABLE oauth_sessions (
		account_did TEXT NOT NULL,
		session_id  TEXT NOT NULL,
		data        JSONB NOT NULL,
		created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
		updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
		PRIMARY KEY (account_did, session_id)
	);
	CREATE TABLE oauth_auth_requests (
		state                  TEXT NOT NULL PRIMARY KEY,
		data                   JSONB NOT NULL,
		created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
		handoff_mode           TEXT NOT NULL DEFAULT 'deep_link',
		loopback_redirect_uri  TEXT,
		device_id              TEXT,
		purpose                TEXT NOT NULL DEFAULT 'login',
		account_deletion_owner_did TEXT,
		account_deletion_job_id UUID,
		CONSTRAINT oauth_auth_requests_purpose_check
			CHECK (purpose IN ('login', 'accountDeletion')),
		CONSTRAINT oauth_auth_requests_account_deletion_metadata_check
			CHECK (
				(purpose = 'login' AND account_deletion_owner_did IS NULL AND account_deletion_job_id IS NULL)
				OR
				(purpose = 'accountDeletion' AND account_deletion_owner_did IS NOT NULL AND account_deletion_job_id IS NOT NULL)
			)
	);
	CREATE TABLE craftsky_sessions (
		token_hash        BYTEA NOT NULL PRIMARY KEY,
		account_did       TEXT NOT NULL,
		oauth_session_id  TEXT NOT NULL,
		device_label      TEXT,
		last_device_id    TEXT,
		created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
		last_seen_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
		revoked_at        TIMESTAMPTZ,
		FOREIGN KEY (account_did, oauth_session_id)
			REFERENCES oauth_sessions (account_did, session_id)
			ON DELETE CASCADE
	);
	CREATE TABLE account_deletion_operations (
		id UUID PRIMARY KEY,
		owner_did TEXT NOT NULL UNIQUE,
		state TEXT NOT NULL,
		accepted_at TIMESTAMPTZ,
		reauth_oauth_session_id TEXT,
		deletion_oauth_session_id TEXT,
		attempt_count INTEGER NOT NULL DEFAULT 0,
		next_attempt_at TIMESTAMPTZ,
		error_category TEXT,
		intent_proof_hash BYTEA,
		confirmation_handle_hash BYTEA,
		intent_expires_at TIMESTAMPTZ,
		lease_owner TEXT,
		lease_token UUID,
		lease_expires_at TIMESTAMPTZ,
		owner_generation BIGINT NOT NULL DEFAULT 1,
		updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
		FOREIGN KEY (owner_did, deletion_oauth_session_id)
			REFERENCES oauth_sessions(account_did, session_id),
		FOREIGN KEY (owner_did, reauth_oauth_session_id)
			REFERENCES oauth_sessions(account_did, session_id)
	);
`

// withAuthSchema returns a pool scoped to a fresh schema seeded with the
// full auth schema. Thin convenience wrapper over testdb.WithSchema that
// bakes in authSchemaDDL so the many call sites stay tidy.
func withAuthSchema(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := testdb.WithSchema(t, authSchemaDDL)
	for _, path := range []string{
		"../../migrations/000038_owner_auth_lifecycle.up.sql",
		"../../migrations/000052_dev_oauth_scheme.up.sql",
		"../../migrations/000064_provider_first_registration.up.sql",
	} {
		migration, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read auth migration %s: %v", path, err)
		}
		if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
			t.Fatalf("apply auth migration %s: %v", path, err)
		}
	}
	return pool
}

func testStoreConfig() auth.StoreConfig {
	return auth.StoreConfig{
		SessionExpiry:                180 * 24 * time.Hour,
		SessionAbsoluteLifetime:      180 * 24 * time.Hour,
		SessionInactivity:            30 * 24 * time.Hour,
		AuthRequestExpiry:            30 * time.Minute,
		PendingAuthRequestCapacity:   4096,
		AuthRequestTerminalRetention: 24 * time.Hour,
		Logger:                       slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
		EndpointValidator:            testOAuthEndpointValidator{},
	}
}

type testOAuthEndpointValidator struct{}

type failingOAuthEndpointValidator struct {
	err error
}

func (validator failingOAuthEndpointValidator) ValidateOrigin(
	context.Context,
	string,
) (*url.URL, error) {
	return nil, validator.err
}

func (validator failingOAuthEndpointValidator) ValidateOAuthEndpoint(
	context.Context,
	string,
	string,
) (*url.URL, error) {
	return nil, validator.err
}

func (testOAuthEndpointValidator) ValidateOrigin(_ context.Context, raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil ||
		u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return nil, &federatedhttp.Error{Kind: federatedhttp.KindDestinationRejected}
	}
	return u, nil
}

func (v testOAuthEndpointValidator) ValidateOAuthEndpoint(
	ctx context.Context,
	issuer string,
	endpoint string,
) (*url.URL, error) {
	issuerURL, err := v.ValidateOrigin(ctx, issuer)
	if err != nil {
		return nil, err
	}
	endpointURL, err := url.Parse(endpoint)
	if err != nil || endpointURL.Scheme != "https" || endpointURL.Host == "" ||
		endpointURL.User != nil || endpointURL.RawQuery != "" || endpointURL.Fragment != "" ||
		endpointURL.Scheme != issuerURL.Scheme || endpointURL.Host != issuerURL.Host {
		return nil, &federatedhttp.Error{Kind: federatedhttp.KindDestinationRejected}
	}
	return endpointURL, nil
}

func validOAuthSession(owner syntax.DID, sessionID string) oauth.ClientSessionData {
	return oauth.ClientSessionData{
		AccountDID:                   owner,
		SessionID:                    sessionID,
		HostURL:                      "https://pds.example.com",
		AuthServerURL:                "https://auth.example.com",
		AuthServerTokenEndpoint:      "https://auth.example.com/oauth/token",
		AuthServerRevocationEndpoint: "https://auth.example.com/oauth/revoke",
	}
}

func TestStoreInitialSessionIsAttemptBoundPendingAndVersioned(t *testing.T) {
	pool := withAuthSchema(t)
	store := auth.NewPostgresAuthStore(pool, testStoreConfig())
	owner := syntax.DID("did:plc:pending-parent")
	seedAuthOwner(t, pool, owner)
	requestContext := auth.WithLoginAuthRequest(
		context.Background(), owner, 1, 1, auth.HandoffVerifiedLink, "device-pending", "",
	)
	request := oauth.AuthRequestData{
		State: "pending-parent-state", RequestURI: "urn:request:pending-parent",
		AuthServerURL: "https://bsky.social",
	}
	if err := store.SaveAuthRequestInfo(requestContext, request); err != nil {
		t.Fatal(err)
	}
	attemptID, err := store.BeginExchange(context.Background(), request.State)
	if err != nil {
		t.Fatal(err)
	}
	attempt := auth.CallbackAttempt{
		State: request.State, AttemptID: attemptID, Owner: owner,
		OwnerGeneration: 1, AuthEpoch: 1, Purpose: auth.LoginOAuthPurpose,
	}
	callbackContext := auth.WithCallbackAttempt(context.Background(), attempt)
	session := validOAuthSession(owner, request.State)
	if err := store.SaveSession(context.Background(), session); !errors.Is(err, auth.ErrCallbackAttemptInvalid) {
		t.Fatalf("unbound initial save error = %v", err)
	}
	if err := store.SaveSession(callbackContext, session); err != nil {
		t.Fatalf("attempt-bound initial save: %v", err)
	}
	if _, err := store.GetSession(context.Background(), owner, request.State); !errors.Is(err, auth.ErrOAuthSessionNotFound) {
		t.Fatalf("ordinary pending resume error = %v", err)
	}
	pending, err := store.ResumePendingOnboardingSession(callbackContext, attempt)
	if err != nil {
		t.Fatalf("callback pending resume: %v", err)
	}
	if pending.RowVersion != 1 || pending.LifecycleState != "pending_handoff" {
		t.Fatalf("pending parent = %+v", pending)
	}
	if _, err := pool.Exec(context.Background(), `
		UPDATE oauth_sessions SET lifecycle_state='active' WHERE account_did=$1 AND session_id=$2
	`, owner, request.State); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `
		UPDATE owner_lifecycles
		SET state='active',generation=generation+1,transition_reason='handoffConfirmed',
		    transitioned_at=now(),updated_at=now()
		WHERE owner_did=$1
	`, owner); err != nil {
		t.Fatal(err)
	}
	session.HostURL = "https://pds-rotated.example.com"
	version, err := store.SaveSessionVersion(context.Background(), session, 1)
	if err != nil || version != 2 {
		t.Fatalf("versioned refresh = %d, %v", version, err)
	}
	if _, err := store.SaveSessionVersion(context.Background(), session, 1); !errors.Is(err, auth.ErrSessionVersionChanged) {
		t.Fatalf("stale refresh error = %v", err)
	}

	malicious := session
	malicious.AuthServerRevocationEndpoint = "https://attacker.example/oauth/revoke"
	maliciousData, err := json.Marshal(malicious)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `
		UPDATE oauth_sessions SET data=$3
		WHERE account_did=$1 AND session_id=$2
	`, owner, request.State, maliciousData); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetSession(context.Background(), owner, request.State); !errors.Is(err, auth.ErrOAuthSessionEndpointInvalid) {
		t.Fatalf("malicious persisted endpoint error = %v", err)
	}
}

func TestStoreFailsClosedWithoutOAuthEndpointValidator(t *testing.T) {
	owner := syntax.DID("did:plc:endpoint-validator")
	state := "endpoint-validator-state"
	attempt := auth.CallbackAttempt{
		State: state, AttemptID: uuid.New(), Owner: owner,
		OwnerGeneration: 1, AuthEpoch: 1, Purpose: auth.LoginOAuthPurpose,
	}
	store := auth.NewPostgresAuthStore(nil, auth.StoreConfig{
		SessionAbsoluteLifetime: time.Hour,
	})
	err := store.SaveSession(auth.WithCallbackAttempt(context.Background(), attempt), validOAuthSession(owner, state))
	if !errors.Is(err, auth.ErrOAuthSessionEndpointInvalid) {
		t.Fatalf("SaveSession error = %v, want endpoint metadata invalid", err)
	}
}

func TestStorePreservesTransientOAuthEndpointValidationFailures(t *testing.T) {
	owner := syntax.DID("did:plc:transient-endpoint-validation")
	state := "transient-endpoint-validation-state"
	attempt := auth.CallbackAttempt{
		State: state, AttemptID: uuid.New(), Owner: owner,
		OwnerGeneration: 1, AuthEpoch: 1, Purpose: auth.LoginOAuthPurpose,
	}
	callbackContext := auth.WithCallbackAttempt(context.Background(), attempt)
	session := validOAuthSession(owner, state)
	dependencyErr := errors.New("resolver dependency unavailable")

	tests := []struct {
		name string
		err  error
		want error
	}{
		{
			name: "canceled",
			err: &federatedhttp.Error{
				Kind: federatedhttp.KindCanceled, Cause: context.Canceled,
			},
			want: context.Canceled,
		},
		{
			name: "deadline",
			err: &federatedhttp.Error{
				Kind: federatedhttp.KindTimeout, Cause: context.DeadlineExceeded,
			},
			want: context.DeadlineExceeded,
		},
		{
			name: "dependency failure",
			err: &federatedhttp.Error{
				Kind: federatedhttp.KindUpstreamFailure, Cause: dependencyErr,
			},
			want: dependencyErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := auth.NewPostgresAuthStore(nil, auth.StoreConfig{
				SessionAbsoluteLifetime: time.Hour,
				EndpointValidator:       failingOAuthEndpointValidator{err: tt.err},
			})
			err := store.SaveSession(callbackContext, session)
			if !errors.Is(err, tt.want) {
				t.Fatalf("SaveSession error = %v, want %v", err, tt.want)
			}
			if errors.Is(err, auth.ErrOAuthSessionEndpointInvalid) {
				t.Fatalf("SaveSession error = %v, transient failure became terminal metadata rejection", err)
			}
		})
	}
}

func TestStoreMapsOAuthEndpointPolicyRejectionToInvalidMetadata(t *testing.T) {
	owner := syntax.DID("did:plc:policy-rejected-endpoint")
	state := "policy-rejected-endpoint-state"
	attempt := auth.CallbackAttempt{
		State: state, AttemptID: uuid.New(), Owner: owner,
		OwnerGeneration: 1, AuthEpoch: 1, Purpose: auth.LoginOAuthPurpose,
	}
	store := auth.NewPostgresAuthStore(nil, auth.StoreConfig{
		SessionAbsoluteLifetime: time.Hour,
		EndpointValidator: failingOAuthEndpointValidator{err: &federatedhttp.Error{
			Kind: federatedhttp.KindDestinationRejected,
		}},
	})

	err := store.SaveSession(
		auth.WithCallbackAttempt(context.Background(), attempt),
		validOAuthSession(owner, state),
	)
	if !errors.Is(err, auth.ErrOAuthSessionEndpointInvalid) {
		t.Fatalf("SaveSession error = %v, want endpoint metadata invalid", err)
	}
}

func TestStoreBindsExactDeletionCredentialWithoutMintingChild(t *testing.T) {
	pool := withAuthSchema(t)
	owners := newAuthOwnerStore(t, pool)
	config := testStoreConfig()
	config.OwnerLifecycles = owners
	store := auth.NewPostgresAuthStore(pool, config)
	owner := syntax.DID("did:plc:deletion-bind")
	operationID := uuid.MustParse("00000000-0000-4000-8000-000000000822")
	state := "deletion-bind-state"
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,
			transitioned_at,created_at,updated_at
		) VALUES($1,'deletion_pending',2,1,'deletionIntent',now(),now(),now())
	`, owner); err != nil {
		t.Fatal(err)
	}
	err := owners.WithExistingAuth(context.Background(), owner, func(authCtx context.Context, authority ownerlifecycle.Lifecycle) error {
		requestCtx := auth.WithAccountDeletionAuthRequestAuthority(
			authCtx, owner, authority.Generation, authority.AuthEpoch, operationID, "device-delete",
		)
		if err := store.SaveAuthRequestInfo(requestCtx, oauth.AuthRequestData{
			State: state, RequestURI: "urn:request:deletion-bind", AccountDID: &owner,
		}); err != nil {
			return err
		}
		attemptID, err := store.BeginExchange(authCtx, state)
		if err != nil {
			return err
		}
		attempt := auth.CallbackAttempt{
			State: state, AttemptID: attemptID, Owner: owner,
			OwnerGeneration: authority.Generation, AuthEpoch: authority.AuthEpoch,
			Purpose: auth.AccountDeletionOAuthPurpose,
		}
		callbackCtx := auth.WithCallbackAttempt(authCtx, attempt)
		if err := store.SaveSession(callbackCtx, validOAuthSession(owner, state)); err != nil {
			return err
		}
		return store.BindDeletionCredential(callbackCtx, attempt, operationID, 4)
	})
	if err != nil {
		t.Fatalf("bind deletion credential: %v", err)
	}
	var lifecycle string
	var boundOperation uuid.UUID
	var generation int64
	var childCount int
	if err := pool.QueryRow(context.Background(), `
		SELECT lifecycle_state,deletion_operation_id,deletion_credential_generation
		FROM oauth_sessions WHERE account_did=$1 AND session_id=$2
	`, owner, state).Scan(&lifecycle, &boundOperation, &generation); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM craftsky_sessions WHERE account_did=$1`, owner).Scan(&childCount); err != nil {
		t.Fatal(err)
	}
	if lifecycle != "deletion_only" || boundOperation != operationID || generation != 4 || childCount != 0 {
		t.Fatalf("lifecycle=%s operation=%s generation=%d children=%d", lifecycle, boundOperation, generation, childCount)
	}
}

func TestStore_SaveGetSession(t *testing.T) {
	pool := withAuthSchema(t)
	store := auth.NewPostgresAuthStore(pool, testStoreConfig())
	ctx := context.Background()

	sess := validOAuthSession(syntax.DID("did:plc:abc"), "sess-1")
	seedActiveStoredOAuthSession(t, pool, sess)
	got, err := store.GetSession(ctx, sess.AccountDID, sess.SessionID)
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if got.HostURL != sess.HostURL {
		t.Fatalf("HostURL: got %q want %q", got.HostURL, sess.HostURL)
	}
}

func TestStore_DeleteSession(t *testing.T) {
	pool := withAuthSchema(t)
	store := auth.NewPostgresAuthStore(pool, testStoreConfig())
	ctx := context.Background()

	sess := validOAuthSession(syntax.DID("did:plc:del"), "sess-del")
	seedActiveStoredOAuthSession(t, pool, sess)
	if err := store.DeleteSession(ctx, sess.AccountDID, sess.SessionID); err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}
	_, err := store.GetSession(ctx, sess.AccountDID, sess.SessionID)
	if err == nil {
		t.Fatal("expected ErrOAuthSessionNotFound after delete, got nil")
	}
	if !isNotFound(err) {
		t.Fatalf("expected ErrOAuthSessionNotFound, got: %v", err)
	}
}

func TestStore_SaveGetAuthRequest(t *testing.T) {
	pool := withAuthSchema(t)
	store := auth.NewPostgresAuthStore(pool, testStoreConfig())
	owner := syntax.DID("did:plc:request-owner")
	seedAuthOwner(t, pool, owner)
	ctx := auth.WithLoginAuthRequest(
		context.Background(), owner, 1, 1, auth.HandoffVerifiedLink, "device-request", "",
	)

	info := oauth.AuthRequestData{
		State:         "state-abc",
		RequestURI:    "urn:request:state-abc",
		AuthServerURL: "https://bsky.social",
		PKCEVerifier:  "verifier-xyz",
	}
	if err := store.SaveAuthRequestInfo(ctx, info); err != nil {
		t.Fatalf("SaveAuthRequestInfo: %v", err)
	}
	got, err := store.GetAuthRequestInfo(ctx, info.State)
	if err != nil {
		t.Fatalf("GetAuthRequestInfo: %v", err)
	}
	if got.AuthServerURL != info.AuthServerURL {
		t.Fatalf("AuthServerURL: got %q want %q", got.AuthServerURL, info.AuthServerURL)
	}
	if got.PKCEVerifier != info.PKCEVerifier {
		t.Fatalf("PKCEVerifier: got %q want %q", got.PKCEVerifier, info.PKCEVerifier)
	}
}

func TestStorePersistsAndReloadsOwnerlessRegistrationAuthRequest(t *testing.T) {
	pool := withAuthSchema(t)
	storeBeforeRestart := auth.NewPostgresAuthStore(pool, testStoreConfig())
	ctx := auth.WithRegistrationAuthRequest(
		context.Background(),
		"https://bsky.social",
		"https://auth.bsky.app",
		auth.HandoffVerifiedLink,
		"device-registration",
		"",
	)
	info := oauth.AuthRequestData{
		State:                   "registration-state",
		RequestURI:              "urn:request:registration",
		AuthServerURL:           "https://auth.bsky.app",
		PKCEVerifier:            "registration-verifier",
		DPoPPrivateKeyMultibase: "registration-dpop-private-key",
	}
	reservation, err := storeBeforeRestart.ReserveAuthRequestCapacity(context.Background())
	if err != nil {
		t.Fatalf("reserve ownerless registration request: %v", err)
	}
	if err := storeBeforeRestart.SaveRegistrationAuthRequest(ctx, reservation.ID, info); err != nil {
		t.Fatalf("save ownerless registration request: %v", err)
	}

	storeAfterRestart := auth.NewPostgresAuthStore(pool, testStoreConfig())
	metadata, err := storeAfterRestart.LoadAuthRequestMetadata(context.Background(), info.State)
	if err != nil {
		t.Fatalf("load ownerless registration metadata after restart: %v", err)
	}
	if metadata.Purpose != auth.RegistrationOAuthPurpose ||
		metadata.RegistrationProviderOrigin != "https://bsky.social" ||
		metadata.RegistrationIssuer != "https://auth.bsky.app" ||
		metadata.Owner != "" || metadata.OwnerGeneration != 0 || metadata.AuthEpoch != 0 ||
		metadata.HandoffMode != auth.HandoffVerifiedLink || metadata.DeviceID != "device-registration" ||
		metadata.RequestState != auth.AuthRequestReady {
		t.Fatalf("reloaded registration metadata = %+v", metadata)
	}
	reloaded, err := storeAfterRestart.GetAuthRequestInfo(context.Background(), info.State)
	if err != nil {
		t.Fatalf("load opaque registration request after restart: %v", err)
	}
	if reloaded.PKCEVerifier != info.PKCEVerifier ||
		reloaded.DPoPPrivateKeyMultibase != info.DPoPPrivateKeyMultibase {
		t.Fatalf("reloaded opaque registration request = %+v", reloaded)
	}

	var ownerDID, ownerGeneration, authEpoch any
	if err := pool.QueryRow(context.Background(), `
		SELECT owner_did,owner_generation,auth_epoch
		FROM oauth_auth_requests WHERE state=$1
	`, info.State).Scan(&ownerDID, &ownerGeneration, &authEpoch); err != nil {
		t.Fatalf("inspect registration owner authority: %v", err)
	}
	if ownerDID != nil || ownerGeneration != nil || authEpoch != nil {
		t.Fatalf("registration authority = %#v/%#v/%#v, want SQL NULL triple", ownerDID, ownerGeneration, authEpoch)
	}
}

func TestStoreBindsRegistrationAuthorityAndPendingParentAtomicallyUnderLifecycleRace(t *testing.T) {
	pool := withAuthSchema(t)
	fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	owners, err := ownerlifecycle.NewStore(pool, fencer, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	config := testStoreConfig()
	config.OwnerLifecycles = owners
	store := auth.NewPostgresAuthStore(pool, config)
	ctx := context.Background()
	owner := syntax.DID("did:plc:registration-bind-race")
	state := "registration-bind-race"
	session := validOAuthSession(owner, state)
	registrationContext := auth.WithRegistrationAuthRequest(
		ctx, "https://bsky.social", session.AuthServerURL,
		auth.HandoffVerifiedLink, "device-registration-bind-race", "",
	)
	reservation, err := store.ReserveAuthRequestCapacity(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveRegistrationAuthRequest(registrationContext, reservation.ID, oauth.AuthRequestData{
		State: state, RequestURI: "urn:request:" + state,
	}); err != nil {
		t.Fatal(err)
	}
	attemptID, err := store.BeginRegistrationExchange(ctx, state)
	if err != nil {
		t.Fatal(err)
	}
	credential, err := json.Marshal(session)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO oauth_unverified_credentials(request_state,data,status,eligible_at,expires_at)
		VALUES($1,$2,'held',now()+interval '1 hour',now()+interval '2 hours')
	`, state, credential); err != nil {
		t.Fatal(err)
	}

	transitionStarted := make(chan struct{})
	transitionDone := make(chan error, 1)
	err = owners.WithOnboardingAuth(ctx, owner, func(authCtx context.Context, lifecycle ownerlifecycle.Lifecycle) error {
		if err := store.BindRegistrationAuthority(authCtx, auth.RegistrationAuthorityBinding{
			State: state, AttemptID: attemptID, Owner: owner,
			OwnerGeneration: lifecycle.Generation, AuthEpoch: lifecycle.AuthEpoch,
			Session: session,
		}); err != nil {
			return err
		}
		go func() {
			close(transitionStarted)
			_, transitionErr := owners.AdvanceAuthEpoch(ctx, owner, lifecycle.Generation, "registrationRace")
			transitionDone <- transitionErr
		}()
		<-transitionStarted
		select {
		case transitionErr := <-transitionDone:
			return fmt.Errorf("lifecycle transition crossed registration bind fence: %w", transitionErr)
		case <-time.After(75 * time.Millisecond):
		}

		var requestOwner *string
		var requestGeneration, requestEpoch *int64
		var parentGeneration, parentEpoch int64
		var parentState string
		if err := pool.QueryRow(authCtx, `
			SELECT request.owner_did,request.owner_generation,request.auth_epoch,
			       parent.owner_generation,parent.auth_epoch,parent.lifecycle_state
			FROM oauth_auth_requests request
			JOIN oauth_sessions parent
			  ON parent.account_did=request.owner_did AND parent.session_id=request.state
			WHERE request.state=$1
		`, state).Scan(
			&requestOwner, &requestGeneration, &requestEpoch,
			&parentGeneration, &parentEpoch, &parentState,
		); err != nil {
			return err
		}
		if requestOwner == nil || *requestOwner != owner.String() ||
			requestGeneration == nil || *requestGeneration != lifecycle.Generation ||
			requestEpoch == nil || *requestEpoch != lifecycle.AuthEpoch ||
			parentGeneration != lifecycle.Generation || parentEpoch != lifecycle.AuthEpoch ||
			parentState != "pending_handoff" {
			return fmt.Errorf("attached request/parent authority mismatch")
		}
		var quarantine int
		if err := pool.QueryRow(authCtx, `
			SELECT count(*) FROM oauth_unverified_credentials WHERE request_state=$1
		`, state).Scan(&quarantine); err != nil {
			return err
		}
		if quarantine != 0 {
			return fmt.Errorf("bound registration retained %d quarantine rows", quarantine)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("bind registration authority: %v", err)
	}
	if err := <-transitionDone; err != nil {
		t.Fatalf("advance lifecycle after bind fence: %v", err)
	}

	staleState := "registration-bind-stale"
	staleSession := validOAuthSession(owner, staleState)
	reservation, err = store.ReserveAuthRequestCapacity(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveRegistrationAuthRequest(registrationContext, reservation.ID, oauth.AuthRequestData{
		State: staleState, RequestURI: "urn:request:" + staleState,
	}); err != nil {
		t.Fatal(err)
	}
	staleAttemptID, err := store.BeginRegistrationExchange(ctx, staleState)
	if err != nil {
		t.Fatal(err)
	}
	staleCredential, err := json.Marshal(staleSession)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO oauth_unverified_credentials(request_state,data,status,eligible_at,expires_at)
		VALUES($1,$2,'held',now()+interval '1 hour',now()+interval '2 hours')
	`, staleState, staleCredential); err != nil {
		t.Fatal(err)
	}
	err = owners.WithExistingAuth(ctx, owner, func(authCtx context.Context, _ ownerlifecycle.Lifecycle) error {
		return store.BindRegistrationAuthority(authCtx, auth.RegistrationAuthorityBinding{
			State: staleState, AttemptID: staleAttemptID, Owner: owner,
			OwnerGeneration: 1, AuthEpoch: 1, Session: staleSession,
		})
	})
	if !errors.Is(err, auth.ErrCallbackAttemptInvalid) {
		t.Fatalf("stale authority bind error=%v, want ErrCallbackAttemptInvalid", err)
	}

	var staleOwner *string
	var staleQuarantine, staleParents int
	if err := pool.QueryRow(ctx, `
		SELECT owner_did FROM oauth_auth_requests WHERE state=$1
	`, staleState).Scan(&staleOwner); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM oauth_unverified_credentials WHERE request_state=$1
	`, staleState).Scan(&staleQuarantine); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM oauth_sessions WHERE account_did=$1 AND session_id=$2
	`, owner, staleState).Scan(&staleParents); err != nil {
		t.Fatal(err)
	}
	if staleOwner != nil || staleQuarantine != 1 || staleParents != 0 {
		t.Fatalf("stale bind residue owner=%v quarantine=%d parents=%d, want nil/1/0", staleOwner, staleQuarantine, staleParents)
	}
}

func TestStoreSavesDevSchemeAuthRequestWithoutClientRedirect(t *testing.T) {
	pool := withAuthSchema(t)
	store := auth.NewPostgresAuthStore(pool, testStoreConfig())
	owner := syntax.DID("did:plc:dev-request-owner")
	seedAuthOwner(t, pool, owner)
	ctx := auth.WithLoginAuthRequest(
		context.Background(), owner, 1, 1, auth.HandoffDevScheme, "device-dev", "",
	)
	info := oauth.AuthRequestData{
		State: "state-dev-scheme", RequestURI: "urn:request:dev-scheme",
		AuthServerURL: "https://bsky.social",
	}
	if err := store.SaveAuthRequestInfo(ctx, info); err != nil {
		t.Fatalf("SaveAuthRequestInfo development scheme: %v", err)
	}
	metadata, err := store.LoadAuthRequestMetadata(context.Background(), info.State)
	if err != nil {
		t.Fatalf("LoadAuthRequestMetadata development scheme: %v", err)
	}
	if metadata.HandoffMode != auth.HandoffDevScheme || metadata.LoopbackURI != "" || metadata.DeviceID != "device-dev" {
		t.Fatalf("development scheme metadata = %+v", metadata)
	}
}

func TestStoreRequiresCompleteAtomicAuthRequestMetadata(t *testing.T) {
	pool := withAuthSchema(t)
	store := auth.NewPostgresAuthStore(pool, testStoreConfig())
	info := oauth.AuthRequestData{
		State: "state-without-authority", RequestURI: "urn:request:missing",
		AuthServerURL: "https://bsky.social",
	}
	if err := store.SaveAuthRequestInfo(context.Background(), info); !errors.Is(err, auth.ErrAuthRequestMetadataInvalid) {
		t.Fatalf("SaveAuthRequestInfo error = %v, want metadata invalid", err)
	}
	var count int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM oauth_auth_requests`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("partial auth request rows = %d, want 0", count)
	}
}

func TestStoreStagesExchangeAndLogicallyConsumesWithoutLosingEvidence(t *testing.T) {
	pool := withAuthSchema(t)
	store := auth.NewPostgresAuthStore(pool, testStoreConfig())
	owner := syntax.DID("did:plc:exchange-owner")
	seedAuthOwner(t, pool, owner)
	ctx := auth.WithLoginAuthRequest(
		context.Background(), owner, 1, 1, auth.HandoffLoopback,
		"device-exchange", "http://127.0.0.1:31001/callback",
	)
	info := oauth.AuthRequestData{
		State: "exchange-state", RequestURI: "urn:request:exchange",
		AuthServerURL: "https://bsky.social",
	}
	if err := store.SaveAuthRequestInfo(ctx, info); err != nil {
		t.Fatalf("save auth request: %v", err)
	}

	attemptID, err := store.BeginExchange(context.Background(), info.State)
	if err != nil {
		t.Fatalf("begin exchange: %v", err)
	}
	if attemptID == uuid.Nil {
		t.Fatal("begin exchange returned nil attempt")
	}
	if err := store.DeleteAuthRequestInfo(context.Background(), info.State); err != nil {
		t.Fatalf("logical consume: %v", err)
	}
	loaded, err := store.GetAuthRequestInfo(context.Background(), info.State)
	if err != nil || loaded.State != info.State {
		t.Fatalf("Indigo callback load after staging = %+v, %v", loaded, err)
	}

	metadata, err := store.LoadAuthRequestMetadata(context.Background(), info.State)
	if err != nil {
		t.Fatalf("load retained metadata: %v", err)
	}
	if metadata.Owner != owner || metadata.OwnerGeneration != 1 || metadata.AuthEpoch != 1 ||
		metadata.HandoffMode != auth.HandoffLoopback || metadata.DeviceID != "device-exchange" ||
		metadata.ExchangeAttemptID != attemptID || metadata.RequestState != auth.AuthRequestExchangeStarted {
		t.Fatalf("retained metadata = %+v", metadata)
	}
	if err := store.MarkExchangeAmbiguous(context.Background(), info.State, attemptID); err != nil {
		t.Fatalf("mark exchange ambiguous: %v", err)
	}
	if _, err := store.GetAuthRequestInfo(context.Background(), info.State); !errors.Is(err, auth.ErrOAuthSessionNotFound) {
		t.Fatalf("ambiguous request replay error = %v, want not found", err)
	}
}

func TestStoreSavesAccountDeletionAuthRequestMetadataAtomically(t *testing.T) {
	pool := withAuthSchema(t)
	store := auth.NewPostgresAuthStore(pool, testStoreConfig())
	owner := syntax.DID("did:plc:alice")
	seedOwnerLifecycle(t, pool, owner, "deletion_pending")
	ctx := auth.WithAccountDeletionAuthRequestAuthority(
		context.Background(), owner, 1, 1,
		uuid.MustParse("10000000-0000-4000-8000-000000000091"), "device-deletion",
	)
	info := oauth.AuthRequestData{
		State: "deletion-state", AuthServerURL: "https://bsky.social",
		RequestURI: "urn:request:deletion",
	}
	if err := store.SaveAuthRequestInfo(ctx, info); err != nil {
		t.Fatalf("save deletion auth request: %v", err)
	}
	var purpose, storedOwner, jobID string
	if err := pool.QueryRow(context.Background(), `
		SELECT purpose,account_deletion_owner_did,account_deletion_job_id::text
		FROM oauth_auth_requests WHERE state=$1
	`, info.State).Scan(&purpose, &storedOwner, &jobID); err != nil {
		t.Fatalf("read deletion auth request metadata: %v", err)
	}
	if purpose != "accountDeletion" || storedOwner != "did:plc:alice" || jobID != "10000000-0000-4000-8000-000000000091" {
		t.Fatalf("atomic deletion metadata purpose=%q owner=%q job=%q", purpose, storedOwner, jobID)
	}
}

func TestStore_DeleteAuthRequest(t *testing.T) {
	pool := withAuthSchema(t)
	store := auth.NewPostgresAuthStore(pool, testStoreConfig())
	owner := syntax.DID("did:plc:delete-request")
	seedAuthOwner(t, pool, owner)
	ctx := auth.WithLoginAuthRequest(
		context.Background(), owner, 1, 1, auth.HandoffVerifiedLink, "device-delete", "",
	)

	info := oauth.AuthRequestData{
		State:         "state-del",
		RequestURI:    "urn:request:state-del",
		AuthServerURL: "https://bsky.social",
	}
	if err := store.SaveAuthRequestInfo(ctx, info); err != nil {
		t.Fatalf("SaveAuthRequestInfo: %v", err)
	}
	if err := store.DeleteAuthRequestInfo(ctx, info.State); err != nil {
		t.Fatalf("DeleteAuthRequestInfo: %v", err)
	}
	_, err := store.GetAuthRequestInfo(ctx, info.State)
	if err == nil {
		t.Fatal("expected ErrOAuthSessionNotFound after delete, got nil")
	}
	if !isNotFound(err) {
		t.Fatalf("expected ErrOAuthSessionNotFound, got: %v", err)
	}
}

func TestStore_ExpiredSessionsAreInvisibleUntilRevocationCleanup(t *testing.T) {
	pool := withAuthSchema(t)
	store := auth.NewPostgresAuthStore(pool, testStoreConfig())
	ctx := context.Background()
	owner := syntax.DID("did:plc:expired")
	seedActiveAuthOwner(t, pool, owner)
	sess := validOAuthSession(owner, "sess-expired")
	data, err := json.Marshal(sess)
	if err != nil {
		t.Fatal(err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO oauth_sessions (
			account_did,session_id,data,lifecycle_state,owner_generation,auth_epoch,
			row_version,absolute_expires_at,created_at,updated_at
		) VALUES ($1,$2,$3,'active',1,1,1,now()-interval '1 day',
		          now()-interval '200 days',now()-interval '200 days')
	`, owner, sess.SessionID, data)
	if err != nil {
		t.Fatalf("insert expired session: %v", err)
	}

	_, getErr := store.GetSession(ctx, owner, sess.SessionID)
	if getErr == nil {
		t.Fatal("expected ErrOAuthSessionNotFound for expired session, got nil")
	}
	if !isNotFound(getErr) {
		t.Fatalf("expected ErrOAuthSessionNotFound, got: %v", getErr)
	}

	// Expiry removes authority immediately, while lifecycle cleanup retains the
	// row until local revocation and remote-token cleanup have been coordinated.
	var count int
	if err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM oauth_sessions WHERE account_did = 'did:plc:expired'`,
	).Scan(&count); err != nil {
		t.Fatalf("count query: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected expired evidence to remain for cleanup, got %d rows", count)
	}
}

func TestStore_OAuthUpdatedAtDoesNotImplyCraftskyChildInactivity(t *testing.T) {
	pool := withAuthSchema(t)
	store := auth.NewPostgresAuthStore(pool, testStoreConfig())
	ctx := context.Background()
	owner := syntax.DID("did:plc:inactive")
	seedActiveAuthOwner(t, pool, owner)
	sess := validOAuthSession(owner, "sess-inactive")
	data, err := json.Marshal(sess)
	if err != nil {
		t.Fatal(err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO oauth_sessions (
			account_did,session_id,data,lifecycle_state,owner_generation,auth_epoch,
			row_version,absolute_expires_at,created_at,updated_at
		) VALUES ($1,$2,$3,'active',1,1,1,now()+interval '120 days',
		          now()-interval '60 days',now()-interval '60 days')
	`, owner, sess.SessionID, data)
	if err != nil {
		t.Fatalf("insert inactive session: %v", err)
	}

	got, getErr := store.GetSession(ctx, owner, sess.SessionID)
	if getErr != nil {
		t.Fatalf("GetSession: %v", getErr)
	}
	if got.SessionID != sess.SessionID {
		t.Fatalf("session ID = %q, want %q", got.SessionID, sess.SessionID)
	}
}

func TestStore_ExpiredAuthRequestsCleanedUp(t *testing.T) {
	pool := withAuthSchema(t)
	store := auth.NewPostgresAuthStore(pool, testStoreConfig())
	ctx := context.Background()
	owner := syntax.DID("did:plc:expired-auth-request")
	seedAuthOwner(t, pool, owner)
	requestCtx := auth.WithLoginAuthRequest(
		ctx, owner, 1, 1, auth.HandoffVerifiedLink, "device-expired", "",
	)
	if err := store.SaveAuthRequestInfo(requestCtx, oauth.AuthRequestData{
		State: "state-expired", RequestURI: "urn:request:state-expired",
	}); err != nil {
		t.Fatalf("save auth request: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE oauth_auth_requests
		SET created_at=now()-interval '60 minutes'
		WHERE state='state-expired'
	`); err != nil {
		t.Fatalf("age auth request: %v", err)
	}
	if _, err := store.SweepAuthRequests(ctx, 10); err != nil {
		t.Fatalf("sweep expired auth request: %v", err)
	}

	_, getErr := store.GetAuthRequestInfo(ctx, "state-expired")
	if getErr == nil {
		t.Fatal("expected ErrOAuthSessionNotFound for expired auth request, got nil")
	}
	if !isNotFound(getErr) {
		t.Fatalf("expected ErrOAuthSessionNotFound, got: %v", getErr)
	}

	var count int
	if err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM oauth_auth_requests WHERE state = 'state-expired'`,
	).Scan(&count); err != nil {
		t.Fatalf("count query: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 rows after auth request cleanup, got %d", count)
	}
}

func TestStore_SaveSessionVersionUpdatesTimestamp(t *testing.T) {
	pool := withAuthSchema(t)
	store := auth.NewPostgresAuthStore(pool, testStoreConfig())
	ctx := context.Background()

	sess := validOAuthSession(syntax.DID("did:plc:ts"), "sess-ts")
	seedActiveStoredOAuthSession(t, pool, sess)

	var before time.Time
	if err := pool.QueryRow(ctx,
		`SELECT updated_at FROM oauth_sessions WHERE account_did = $1 AND session_id = $2`,
		sess.AccountDID.String(), sess.SessionID,
	).Scan(&before); err != nil {
		t.Fatalf("read updated_at before: %v", err)
	}

	// Small sleep to ensure the clock advances enough for the DB timestamp to differ.
	time.Sleep(50 * time.Millisecond)

	// Update the session with a different field value.
	sess.HostURL = "https://pds2.example.com"
	version, err := store.SaveSessionVersion(ctx, sess, 1)
	if err != nil {
		t.Fatalf("SaveSessionVersion: %v", err)
	}
	if version != 2 {
		t.Fatalf("row version = %d, want 2", version)
	}

	var after time.Time
	if err := pool.QueryRow(ctx,
		`SELECT updated_at FROM oauth_sessions WHERE account_did = $1 AND session_id = $2`,
		sess.AccountDID.String(), sess.SessionID,
	).Scan(&after); err != nil {
		t.Fatalf("read updated_at after: %v", err)
	}

	if !after.After(before) {
		t.Fatalf("updated_at did not advance: before=%v after=%v", before, after)
	}
}

// isNotFound is a thin alias around errors.Is for the not-found sentinel,
// kept so the assertion sites read cleanly. The whole point of having
// ErrOAuthSessionNotFound exported is that callers can errors.Is against it —
// don't use string comparison.
func isNotFound(err error) bool {
	return errors.Is(err, auth.ErrOAuthSessionNotFound)
}

func seedAuthOwner(t *testing.T, pool *pgxpool.Pool, owner syntax.DID) {
	t.Helper()
	seedOwnerLifecycle(t, pool, owner, "departed")
}

func seedActiveAuthOwner(t *testing.T, pool *pgxpool.Pool, owner syntax.DID) {
	t.Helper()
	seedOwnerLifecycle(t, pool, owner, "active")
}

func seedOwnerLifecycle(t *testing.T, pool *pgxpool.Pool, owner syntax.DID, state string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,
			transitioned_at,created_at,updated_at
		) VALUES($1,$2,1,1,'testFixture',now(),now(),now())
		ON CONFLICT (owner_did) DO NOTHING
	`, owner, state)
	if err != nil {
		t.Fatalf("seed auth owner: %v", err)
	}
	var gotState string
	var generation, authEpoch int64
	if err := pool.QueryRow(context.Background(), `
		SELECT state,generation,auth_epoch FROM owner_lifecycles WHERE owner_did=$1
	`, owner).Scan(&gotState, &generation, &authEpoch); err != nil {
		t.Fatalf("read seeded auth owner: %v", err)
	}
	if gotState != state || generation != 1 || authEpoch != 1 {
		t.Fatalf("seeded auth owner = %s generation %d epoch %d, want %s generation 1 epoch 1",
			gotState, generation, authEpoch, state)
	}
}

func seedActiveOAuthSession(t *testing.T, pool *pgxpool.Pool, owner, sessionID string) {
	t.Helper()
	seedActiveStoredOAuthSession(t, pool, validOAuthSession(syntax.DID(owner), sessionID))
}

func seedActiveStoredOAuthSession(t *testing.T, pool *pgxpool.Pool, sess oauth.ClientSessionData) {
	t.Helper()
	seedActiveAuthOwner(t, pool, sess.AccountDID)
	data, err := json.Marshal(sess)
	if err != nil {
		t.Fatalf("marshal OAuth session: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO oauth_sessions(
			account_did,session_id,data,lifecycle_state,owner_generation,auth_epoch,
			row_version,absolute_expires_at
		) VALUES($1,$2,$3,'active',1,1,1,now()+interval '180 days')
	`, sess.AccountDID, sess.SessionID, data); err != nil {
		t.Fatalf("seed active OAuth session: %v", err)
	}
}
