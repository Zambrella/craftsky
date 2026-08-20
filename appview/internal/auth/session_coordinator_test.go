package auth_test

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/atcrypto"
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/federatedhttp"
	"social.craftsky/appview/internal/ownerlifecycle"
)

type blockingOAuthEndpointValidator struct {
	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

func (validator *blockingOAuthEndpointValidator) ValidateOrigin(
	ctx context.Context,
	raw string,
) (*url.URL, error) {
	wait := false
	validator.once.Do(func() {
		wait = true
		close(validator.entered)
	})
	if wait {
		select {
		case <-validator.release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return testOAuthEndpointValidator{}.ValidateOrigin(ctx, raw)
}

func (validator *blockingOAuthEndpointValidator) ValidateOAuthEndpoint(
	ctx context.Context,
	issuer string,
	endpoint string,
) (*url.URL, error) {
	return testOAuthEndpointValidator{}.ValidateOAuthEndpoint(ctx, issuer, endpoint)
}

func TestOAuthSessionCoordinatorTerminalizesInvalidOrdinarySessionWithoutNetwork(t *testing.T) {
	pool := withAuthSchema(t)
	owners := newAuthOwnerStore(t, pool)
	storeConfig := testStoreConfig()
	storeConfig.OwnerLifecycles = owners
	store := auth.NewPostgresAuthStore(pool, storeConfig)
	owner := syntax.DID("did:plc:invalid-ordinary-endpoint")
	sessionID := "invalid-ordinary-parent"
	data := validOAuthSession(owner, sessionID)
	data.HostURL = "http://127.0.0.1:18181"

	if _, err := pool.Exec(context.Background(), `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,
			transitioned_at,created_at,updated_at
		) VALUES($1,'active',1,1,'test',now(),now(),now())
	`, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO oauth_sessions(
			account_did,session_id,data,lifecycle_state,owner_generation,auth_epoch,
			row_version,absolute_expires_at,created_at,updated_at
		) VALUES($1,$2,$3,'active',1,1,3,now()+interval '1 day',now(),now())
	`, owner, sessionID, data); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO craftsky_sessions(
			token_hash,account_did,oauth_session_id,lifecycle_state,auth_epoch,
			last_seen_at,idle_expires_at
		) VALUES($3,$1,$2,'active',1,now(),now()+interval '1 day')
	`, owner, sessionID, []byte("invalid-ordinary-child")); err != nil {
		t.Fatal(err)
	}

	var networkCalls atomic.Int64
	config := oauth.NewPublicConfig(
		"https://appview.example/oauth/client-metadata.json",
		"https://appview.example/oauth/callback",
		[]string{"atproto"},
	)
	coordinator, err := auth.NewOAuthSessionCoordinator(auth.OAuthSessionCoordinatorOptions{
		App: &oauth.ClientApp{
			Client: &http.Client{Transport: cleanupRoundTripFunc(func(*http.Request) (*http.Response, error) {
				networkCalls.Add(1)
				return nil, errors.New("unexpected OAuth/PDS network call")
			})},
			Config: &config,
		},
		Store: store, Owners: owners, OperationTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	operationCalled := false
	err = coordinator.WithActiveSession(
		context.Background(), owner, sessionID,
		func(context.Context, *oauth.ClientSession) error {
			operationCalled = true
			return nil
		},
	)
	if !errors.Is(err, auth.ErrOAuthSessionEndpointInvalid) {
		t.Fatalf("coordinated invalid endpoint error = %v", err)
	}
	if operationCalled || networkCalls.Load() != 0 {
		t.Fatalf("invalid endpoint reached operation=%t network calls=%d", operationCalled, networkCalls.Load())
	}

	var parentState, childState string
	var parentVersion int64
	var revocationRequestedAt, cleanupNextAttemptAt, childRevokedAt *time.Time
	if err := pool.QueryRow(context.Background(), `
		SELECT parent.lifecycle_state,parent.row_version,parent.revocation_requested_at,
		       parent.cleanup_next_attempt_at,child.lifecycle_state,child.revoked_at
		FROM oauth_sessions parent
		JOIN craftsky_sessions child
		  ON child.account_did=parent.account_did AND child.oauth_session_id=parent.session_id
		WHERE parent.account_did=$1 AND parent.session_id=$2
	`, owner, sessionID).Scan(
		&parentState, &parentVersion, &revocationRequestedAt,
		&cleanupNextAttemptAt, &childState, &childRevokedAt,
	); err != nil {
		t.Fatal(err)
	}
	if parentState != "revocation_pending" || parentVersion != 4 ||
		revocationRequestedAt == nil || cleanupNextAttemptAt == nil ||
		childState != "revoked" || childRevokedAt == nil {
		t.Fatalf(
			"invalid endpoint states parent=%s version=%d requested=%v cleanup=%v child=%s revoked=%v",
			parentState, parentVersion, revocationRequestedAt, cleanupNextAttemptAt, childState, childRevokedAt,
		)
	}
}

func TestOAuthSessionCoordinatorPreservesOrdinarySessionOnTransientEndpointValidationFailure(t *testing.T) {
	pool := withAuthSchema(t)
	owners := newAuthOwnerStore(t, pool)
	dependencyErr := errors.New("OAuth endpoint resolver unavailable")
	storeConfig := testStoreConfig()
	storeConfig.OwnerLifecycles = owners
	storeConfig.EndpointValidator = failingOAuthEndpointValidator{err: &federatedhttp.Error{
		Kind: federatedhttp.KindUpstreamFailure, Cause: dependencyErr,
	}}
	store := auth.NewPostgresAuthStore(pool, storeConfig)
	owner := syntax.DID("did:plc:transient-ordinary-endpoint")
	sessionID := "transient-ordinary-parent"
	data := validOAuthSession(owner, sessionID)

	if _, err := pool.Exec(context.Background(), `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,
			transitioned_at,created_at,updated_at
		) VALUES($1,'active',1,1,'test',now(),now(),now())
	`, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO oauth_sessions(
			account_did,session_id,data,lifecycle_state,owner_generation,auth_epoch,
			row_version,absolute_expires_at,created_at,updated_at
		) VALUES($1,$2,$3,'active',1,1,3,now()+interval '1 day',now(),now())
	`, owner, sessionID, data); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO craftsky_sessions(
			token_hash,account_did,oauth_session_id,lifecycle_state,auth_epoch,
			last_seen_at,idle_expires_at
		) VALUES($3,$1,$2,'active',1,now(),now()+interval '1 day')
	`, owner, sessionID, []byte("transient-ordinary-child")); err != nil {
		t.Fatal(err)
	}

	var networkCalls atomic.Int64
	config := oauth.NewPublicConfig(
		"https://appview.example/oauth/client-metadata.json",
		"https://appview.example/oauth/callback",
		[]string{"atproto"},
	)
	coordinator, err := auth.NewOAuthSessionCoordinator(auth.OAuthSessionCoordinatorOptions{
		App: &oauth.ClientApp{
			Client: &http.Client{Transport: cleanupRoundTripFunc(func(*http.Request) (*http.Response, error) {
				networkCalls.Add(1)
				return nil, errors.New("unexpected OAuth/PDS network call")
			})},
			Config: &config,
		},
		Store: store, Owners: owners, OperationTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	operationCalled := false
	err = coordinator.WithActiveSession(
		context.Background(), owner, sessionID,
		func(context.Context, *oauth.ClientSession) error {
			operationCalled = true
			return nil
		},
	)
	if !errors.Is(err, dependencyErr) {
		t.Fatalf("coordinated transient endpoint error = %v, want dependency failure", err)
	}
	if errors.Is(err, auth.ErrOAuthSessionEndpointInvalid) {
		t.Fatalf("coordinated transient endpoint error = %v, want retryable failure", err)
	}
	if operationCalled || networkCalls.Load() != 0 {
		t.Fatalf("transient endpoint reached operation=%t network calls=%d", operationCalled, networkCalls.Load())
	}

	var parentState, childState string
	var parentVersion int64
	var revocationRequestedAt, cleanupNextAttemptAt, childRevokedAt *time.Time
	if err := pool.QueryRow(context.Background(), `
		SELECT parent.lifecycle_state,parent.row_version,parent.revocation_requested_at,
		       parent.cleanup_next_attempt_at,child.lifecycle_state,child.revoked_at
		FROM oauth_sessions parent
		JOIN craftsky_sessions child
		  ON child.account_did=parent.account_did AND child.oauth_session_id=parent.session_id
		WHERE parent.account_did=$1 AND parent.session_id=$2
	`, owner, sessionID).Scan(
		&parentState, &parentVersion, &revocationRequestedAt,
		&cleanupNextAttemptAt, &childState, &childRevokedAt,
	); err != nil {
		t.Fatal(err)
	}
	if parentState != "active" || parentVersion != 3 ||
		revocationRequestedAt != nil || cleanupNextAttemptAt != nil ||
		childState != "active" || childRevokedAt != nil {
		t.Fatalf(
			"transient endpoint states parent=%s version=%d requested=%v cleanup=%v child=%s revoked=%v",
			parentState, parentVersion, revocationRequestedAt, cleanupNextAttemptAt, childState, childRevokedAt,
		)
	}
}

func TestOAuthSessionCoordinatorDoesNotTerminalizeCorrectedConcurrentSessionVersion(t *testing.T) {
	pool := withAuthSchema(t)
	owners := newAuthOwnerStore(t, pool)
	validator := &blockingOAuthEndpointValidator{
		entered: make(chan struct{}),
		release: make(chan struct{}),
	}
	storeConfig := testStoreConfig()
	storeConfig.OwnerLifecycles = owners
	storeConfig.EndpointValidator = validator
	store := auth.NewPostgresAuthStore(pool, storeConfig)
	owner := syntax.DID("did:plc:corrected-concurrent-endpoint")
	sessionID := "corrected-concurrent-parent"
	privateKey, err := atcrypto.GeneratePrivateKeyP256()
	if err != nil {
		t.Fatal(err)
	}
	invalidData := validOAuthSession(owner, sessionID)
	invalidData.HostURL = "http://127.0.0.1:18182"
	invalidData.DPoPPrivateKeyMultibase = privateKey.Multibase()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,
			transitioned_at,created_at,updated_at
		) VALUES($1,'active',1,1,'test',now(),now(),now())
	`, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO oauth_sessions(
			account_did,session_id,data,lifecycle_state,owner_generation,auth_epoch,
			row_version,absolute_expires_at,created_at,updated_at
		) VALUES($1,$2,$3,'active',1,1,3,now()+interval '1 day',now(),now())
	`, owner, sessionID, invalidData); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO craftsky_sessions(
			token_hash,account_did,oauth_session_id,lifecycle_state,auth_epoch,
			last_seen_at,idle_expires_at
		) VALUES($3,$1,$2,'active',1,now(),now()+interval '1 day')
	`, owner, sessionID, []byte("corrected-concurrent-child")); err != nil {
		t.Fatal(err)
	}

	var networkCalls atomic.Int64
	config := oauth.NewPublicConfig(
		"https://appview.example/oauth/client-metadata.json",
		"https://appview.example/oauth/callback",
		[]string{"atproto"},
	)
	coordinator, err := auth.NewOAuthSessionCoordinator(auth.OAuthSessionCoordinatorOptions{
		App: &oauth.ClientApp{
			Client: &http.Client{Transport: cleanupRoundTripFunc(func(*http.Request) (*http.Response, error) {
				networkCalls.Add(1)
				return nil, errors.New("unexpected OAuth/PDS network call")
			})},
			Config: &config,
		},
		Store: store, Owners: owners, OperationTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	operationCalls := make(chan struct{}, 1)
	done := make(chan error, 1)
	go func() {
		done <- coordinator.WithActiveSession(
			context.Background(), owner, sessionID,
			func(context.Context, *oauth.ClientSession) error {
				operationCalls <- struct{}{}
				return nil
			},
		)
	}()
	select {
	case <-validator.entered:
	case <-time.After(time.Second):
		t.Fatal("coordinator did not reach endpoint validation")
	}
	correctedData := validOAuthSession(owner, sessionID)
	correctedData.DPoPPrivateKeyMultibase = privateKey.Multibase()
	if _, err := pool.Exec(context.Background(), `
		UPDATE oauth_sessions
		SET data=$3,row_version=row_version+1,updated_at=now()
		WHERE account_did=$1 AND session_id=$2
	`, owner, sessionID, correctedData); err != nil {
		t.Fatal(err)
	}
	close(validator.release)
	if err := <-done; err != nil {
		t.Fatalf("coordinated corrected endpoint: %v", err)
	}
	if len(operationCalls) != 1 || networkCalls.Load() != 0 {
		t.Fatalf("corrected endpoint operation calls=%d network calls=%d", len(operationCalls), networkCalls.Load())
	}

	var parentState, childState string
	var parentVersion int64
	if err := pool.QueryRow(context.Background(), `
		SELECT parent.lifecycle_state,parent.row_version,child.lifecycle_state
		FROM oauth_sessions parent
		JOIN craftsky_sessions child
		  ON child.account_did=parent.account_did AND child.oauth_session_id=parent.session_id
		WHERE parent.account_did=$1 AND parent.session_id=$2
	`, owner, sessionID).Scan(&parentState, &parentVersion, &childState); err != nil {
		t.Fatal(err)
	}
	if parentState != "active" || parentVersion != 4 || childState != "active" {
		t.Fatalf("corrected endpoint states parent=%s version=%d child=%s", parentState, parentVersion, childState)
	}
}

func TestOAuthSessionCoordinatorInvalidDeletionEndpointRequiresReauthenticationWithoutNetwork(t *testing.T) {
	pool := withAuthSchema(t)
	owners := newAuthOwnerStore(t, pool)
	storeConfig := testStoreConfig()
	storeConfig.OwnerLifecycles = owners
	store := auth.NewPostgresAuthStore(pool, storeConfig)
	owner := syntax.DID("did:plc:invalid-deletion-endpoint")
	sessionID := "invalid-deletion-parent"
	operationID := uuid.New()
	leaseToken := uuid.New()
	credentialGeneration := int64(4)
	data := validOAuthSession(owner, sessionID)
	data.AuthServerTokenEndpoint = "https://attacker.example/oauth/token"
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,
			transitioned_at,created_at,updated_at
		) VALUES($1,'deleting',3,2,'test',now(),now(),now())
	`, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO oauth_sessions(
			account_did,session_id,data,lifecycle_state,owner_generation,auth_epoch,
			row_version,absolute_expires_at,deletion_operation_id,
			deletion_credential_generation,created_at,updated_at
		) VALUES($1,$2,$3,'deletion_only',2,2,5,now()+interval '1 day',$4,$5,now(),now())
	`, owner, sessionID, data, operationID, credentialGeneration); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO account_deletion_operations(
			id,owner_did,owner_generation,state,accepted_at,
			deletion_oauth_session_id,deletion_credential_generation,
			lease_owner,lease_token,lease_expires_at
		) VALUES($1,$2,1,'active',now(),$3,$4,'test',$5,now()+interval '1 minute')
	`, operationID, owner, sessionID, credentialGeneration, leaseToken); err != nil {
		t.Fatal(err)
	}

	var networkCalls atomic.Int64
	config := oauth.NewPublicConfig(
		"https://appview.example/oauth/client-metadata.json",
		"https://appview.example/oauth/callback",
		[]string{"atproto"},
	)
	coordinator, err := auth.NewOAuthSessionCoordinator(auth.OAuthSessionCoordinatorOptions{
		App: &oauth.ClientApp{
			Client: &http.Client{Transport: cleanupRoundTripFunc(func(*http.Request) (*http.Response, error) {
				networkCalls.Add(1)
				return nil, errors.New("unexpected OAuth/PDS network call")
			})},
			Config: &config,
		},
		Store: store, Owners: owners, OperationTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	authority := auth.DeletionSessionAuthority{
		DeletionCredentialBinding: auth.DeletionCredentialBinding{
			OperationID: operationID, SessionID: sessionID,
			CredentialGeneration: credentialGeneration,
		},
		OwnerGeneration: 1, LeaseToken: leaseToken,
	}
	operationCalled := false
	err = coordinator.WithDeletionSession(
		context.Background(), owner, authority,
		func(context.Context, *oauth.ClientSession) error {
			operationCalled = true
			return nil
		},
	)
	if !errors.Is(err, auth.ErrOAuthSessionEndpointInvalid) ||
		!errors.Is(err, auth.ErrDeletionReauthenticationRequired) {
		t.Fatalf("coordinated invalid deletion endpoint error = %v", err)
	}
	if operationCalled || networkCalls.Load() != 0 {
		t.Fatalf("invalid deletion endpoint reached operation=%t network calls=%d", operationCalled, networkCalls.Load())
	}

	var ownerState, operationState, errorCategory, parentState string
	var ownerGeneration, ownerAuthEpoch, operationOwnerGeneration, parentVersion int64
	var acceptedAt time.Time
	var operationSession *string
	var operationCredentialGeneration *int64
	var operationLeaseToken *uuid.UUID
	var parentOperationID *uuid.UUID
	var parentCredentialGeneration *int64
	if err := pool.QueryRow(context.Background(), `
		SELECT owner.state,owner.generation,owner.auth_epoch,
		       operation.state,operation.owner_generation,operation.accepted_at,
		       operation.deletion_oauth_session_id,operation.deletion_credential_generation,
		       operation.error_category,operation.lease_token,
		       parent.lifecycle_state,parent.row_version,parent.deletion_operation_id,
		       parent.deletion_credential_generation
		FROM owner_lifecycles owner
		JOIN account_deletion_operations operation ON operation.owner_did=owner.owner_did
		JOIN oauth_sessions parent ON parent.account_did=owner.owner_did
		WHERE owner.owner_did=$1 AND operation.id=$2 AND parent.session_id=$3
	`, owner, operationID, sessionID).Scan(
		&ownerState, &ownerGeneration, &ownerAuthEpoch,
		&operationState, &operationOwnerGeneration, &acceptedAt,
		&operationSession, &operationCredentialGeneration,
		&errorCategory, &operationLeaseToken,
		&parentState, &parentVersion, &parentOperationID,
		&parentCredentialGeneration,
	); err != nil {
		t.Fatal(err)
	}
	if ownerState != "deleting" || ownerGeneration != 3 || ownerAuthEpoch != 2 ||
		operationState != "reauth_required" || operationOwnerGeneration != 1 || acceptedAt.IsZero() ||
		operationSession != nil || operationCredentialGeneration != nil ||
		errorCategory != "reauthentication" || operationLeaseToken != nil ||
		parentState != "revocation_pending" || parentVersion != 6 ||
		parentOperationID != nil || parentCredentialGeneration != nil {
		t.Fatalf(
			"invalid deletion transition owner=%s/%d/%d operation=%s/%d accepted=%v session=%v credential=%v category=%s lease=%v parent=%s/%d/%v/%v",
			ownerState, ownerGeneration, ownerAuthEpoch,
			operationState, operationOwnerGeneration, acceptedAt,
			operationSession, operationCredentialGeneration, errorCategory, operationLeaseToken,
			parentState, parentVersion, parentOperationID, parentCredentialGeneration,
		)
	}
}

func TestOAuthSessionCoordinatorPreservesDeletionCredentialOnTransientEndpointValidationFailure(t *testing.T) {
	pool := withAuthSchema(t)
	owners := newAuthOwnerStore(t, pool)
	dependencyErr := errors.New("OAuth endpoint resolver unavailable")
	storeConfig := testStoreConfig()
	storeConfig.OwnerLifecycles = owners
	storeConfig.EndpointValidator = failingOAuthEndpointValidator{err: &federatedhttp.Error{
		Kind: federatedhttp.KindUpstreamFailure, Cause: dependencyErr,
	}}
	store := auth.NewPostgresAuthStore(pool, storeConfig)
	owner := syntax.DID("did:plc:transient-deletion-endpoint")
	sessionID := "transient-deletion-parent"
	operationID := uuid.New()
	leaseToken := uuid.New()
	credentialGeneration := int64(4)
	data := validOAuthSession(owner, sessionID)
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,
			transitioned_at,created_at,updated_at
		) VALUES($1,'deleting',3,2,'test',now(),now(),now())
	`, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO oauth_sessions(
			account_did,session_id,data,lifecycle_state,owner_generation,auth_epoch,
			row_version,absolute_expires_at,deletion_operation_id,
			deletion_credential_generation,created_at,updated_at
		) VALUES($1,$2,$3,'deletion_only',2,2,5,now()+interval '1 day',$4,$5,now(),now())
	`, owner, sessionID, data, operationID, credentialGeneration); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO account_deletion_operations(
			id,owner_did,owner_generation,state,accepted_at,
			deletion_oauth_session_id,deletion_credential_generation,
			lease_owner,lease_token,lease_expires_at
		) VALUES($1,$2,1,'active',now(),$3,$4,'test',$5,now()+interval '1 minute')
	`, operationID, owner, sessionID, credentialGeneration, leaseToken); err != nil {
		t.Fatal(err)
	}

	var networkCalls atomic.Int64
	config := oauth.NewPublicConfig(
		"https://appview.example/oauth/client-metadata.json",
		"https://appview.example/oauth/callback",
		[]string{"atproto"},
	)
	coordinator, err := auth.NewOAuthSessionCoordinator(auth.OAuthSessionCoordinatorOptions{
		App: &oauth.ClientApp{
			Client: &http.Client{Transport: cleanupRoundTripFunc(func(*http.Request) (*http.Response, error) {
				networkCalls.Add(1)
				return nil, errors.New("unexpected OAuth/PDS network call")
			})},
			Config: &config,
		},
		Store: store, Owners: owners, OperationTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	authority := auth.DeletionSessionAuthority{
		DeletionCredentialBinding: auth.DeletionCredentialBinding{
			OperationID: operationID, SessionID: sessionID,
			CredentialGeneration: credentialGeneration,
		},
		OwnerGeneration: 1, LeaseToken: leaseToken,
	}
	operationCalled := false
	err = coordinator.WithDeletionSession(
		context.Background(), owner, authority,
		func(context.Context, *oauth.ClientSession) error {
			operationCalled = true
			return nil
		},
	)
	if !errors.Is(err, dependencyErr) {
		t.Fatalf("coordinated transient deletion endpoint error = %v, want dependency failure", err)
	}
	if errors.Is(err, auth.ErrOAuthSessionEndpointInvalid) ||
		errors.Is(err, auth.ErrDeletionReauthenticationRequired) {
		t.Fatalf("coordinated transient deletion endpoint error = %v, want retryable failure", err)
	}
	if operationCalled || networkCalls.Load() != 0 {
		t.Fatalf("transient deletion endpoint reached operation=%t network calls=%d", operationCalled, networkCalls.Load())
	}

	var ownerState, operationState, parentState string
	var ownerGeneration, ownerAuthEpoch, operationOwnerGeneration, parentVersion int64
	var acceptedAt time.Time
	var operationSession *string
	var operationCredentialGeneration *int64
	var errorCategory *string
	var operationLeaseToken *uuid.UUID
	var parentOperationID *uuid.UUID
	var parentCredentialGeneration *int64
	if err := pool.QueryRow(context.Background(), `
		SELECT owner.state,owner.generation,owner.auth_epoch,
		       operation.state,operation.owner_generation,operation.accepted_at,
		       operation.deletion_oauth_session_id,operation.deletion_credential_generation,
		       operation.error_category,operation.lease_token,
		       parent.lifecycle_state,parent.row_version,parent.deletion_operation_id,
		       parent.deletion_credential_generation
		FROM owner_lifecycles owner
		JOIN account_deletion_operations operation ON operation.owner_did=owner.owner_did
		JOIN oauth_sessions parent ON parent.account_did=owner.owner_did
		WHERE owner.owner_did=$1 AND operation.id=$2 AND parent.session_id=$3
	`, owner, operationID, sessionID).Scan(
		&ownerState, &ownerGeneration, &ownerAuthEpoch,
		&operationState, &operationOwnerGeneration, &acceptedAt,
		&operationSession, &operationCredentialGeneration,
		&errorCategory, &operationLeaseToken,
		&parentState, &parentVersion, &parentOperationID,
		&parentCredentialGeneration,
	); err != nil {
		t.Fatal(err)
	}
	if ownerState != "deleting" || ownerGeneration != 3 || ownerAuthEpoch != 2 ||
		operationState != "active" || operationOwnerGeneration != 1 || acceptedAt.IsZero() ||
		operationSession == nil || *operationSession != sessionID ||
		operationCredentialGeneration == nil || *operationCredentialGeneration != credentialGeneration ||
		errorCategory != nil || operationLeaseToken == nil || *operationLeaseToken != leaseToken ||
		parentState != "deletion_only" || parentVersion != 5 ||
		parentOperationID == nil || *parentOperationID != operationID ||
		parentCredentialGeneration == nil || *parentCredentialGeneration != credentialGeneration {
		t.Fatalf(
			"transient deletion state owner=%s/%d/%d operation=%s/%d accepted=%v session=%v credential=%v category=%v lease=%v parent=%s/%d/%v/%v",
			ownerState, ownerGeneration, ownerAuthEpoch,
			operationState, operationOwnerGeneration, acceptedAt,
			operationSession, operationCredentialGeneration, errorCategory, operationLeaseToken,
			parentState, parentVersion, parentOperationID, parentCredentialGeneration,
		)
	}
}

func TestOAuthSessionCoordinatorDoesNotCorruptCorrectedDeletionCredentialVersion(t *testing.T) {
	pool := withAuthSchema(t)
	owners := newAuthOwnerStore(t, pool)
	validator := &blockingOAuthEndpointValidator{
		entered: make(chan struct{}),
		release: make(chan struct{}),
	}
	storeConfig := testStoreConfig()
	storeConfig.OwnerLifecycles = owners
	storeConfig.EndpointValidator = validator
	store := auth.NewPostgresAuthStore(pool, storeConfig)
	owner := syntax.DID("did:plc:corrected-deletion-endpoint")
	sessionID := "corrected-deletion-parent"
	operationID := uuid.New()
	leaseToken := uuid.New()
	credentialGeneration := int64(4)
	privateKey, err := atcrypto.GeneratePrivateKeyP256()
	if err != nil {
		t.Fatal(err)
	}
	invalidData := validOAuthSession(owner, sessionID)
	invalidData.AuthServerTokenEndpoint = "https://attacker.example/oauth/token"
	invalidData.DPoPPrivateKeyMultibase = privateKey.Multibase()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,
			transitioned_at,created_at,updated_at
		) VALUES($1,'deleting',3,2,'test',now(),now(),now())
	`, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO oauth_sessions(
			account_did,session_id,data,lifecycle_state,owner_generation,auth_epoch,
			row_version,absolute_expires_at,deletion_operation_id,
			deletion_credential_generation,created_at,updated_at
		) VALUES($1,$2,$3,'deletion_only',2,2,5,now()+interval '1 day',$4,$5,now(),now())
	`, owner, sessionID, invalidData, operationID, credentialGeneration); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO account_deletion_operations(
			id,owner_did,owner_generation,state,accepted_at,
			deletion_oauth_session_id,deletion_credential_generation,
			lease_owner,lease_token,lease_expires_at
		) VALUES($1,$2,1,'active',now(),$3,$4,'test',$5,now()+interval '1 minute')
	`, operationID, owner, sessionID, credentialGeneration, leaseToken); err != nil {
		t.Fatal(err)
	}

	var networkCalls atomic.Int64
	config := oauth.NewPublicConfig(
		"https://appview.example/oauth/client-metadata.json",
		"https://appview.example/oauth/callback",
		[]string{"atproto"},
	)
	coordinator, err := auth.NewOAuthSessionCoordinator(auth.OAuthSessionCoordinatorOptions{
		App: &oauth.ClientApp{
			Client: &http.Client{Transport: cleanupRoundTripFunc(func(*http.Request) (*http.Response, error) {
				networkCalls.Add(1)
				return nil, errors.New("unexpected OAuth/PDS network call")
			})},
			Config: &config,
		},
		Store: store, Owners: owners, OperationTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	authority := auth.DeletionSessionAuthority{
		DeletionCredentialBinding: auth.DeletionCredentialBinding{
			OperationID: operationID, SessionID: sessionID,
			CredentialGeneration: credentialGeneration,
		},
		OwnerGeneration: 1, LeaseToken: leaseToken,
	}
	operationCalls := make(chan struct{}, 1)
	done := make(chan error, 1)
	go func() {
		done <- coordinator.WithDeletionSession(
			context.Background(), owner, authority,
			func(context.Context, *oauth.ClientSession) error {
				operationCalls <- struct{}{}
				return nil
			},
		)
	}()
	select {
	case <-validator.entered:
	case <-time.After(time.Second):
		t.Fatal("deletion coordinator did not reach endpoint validation")
	}
	correctedData := validOAuthSession(owner, sessionID)
	correctedData.DPoPPrivateKeyMultibase = privateKey.Multibase()
	if _, err := pool.Exec(context.Background(), `
		UPDATE oauth_sessions
		SET data=$3,row_version=row_version+1,updated_at=now()
		WHERE account_did=$1 AND session_id=$2
	`, owner, sessionID, correctedData); err != nil {
		t.Fatal(err)
	}
	close(validator.release)
	if err := <-done; err != nil {
		t.Fatalf("coordinated corrected deletion endpoint: %v", err)
	}
	if len(operationCalls) != 1 || networkCalls.Load() != 0 {
		t.Fatalf("corrected deletion operation calls=%d network calls=%d", len(operationCalls), networkCalls.Load())
	}

	var operationState, operationSession, parentState string
	var operationCredentialGeneration, parentCredentialGeneration, parentVersion int64
	var operationLeaseToken, parentOperationID uuid.UUID
	if err := pool.QueryRow(context.Background(), `
		SELECT operation.state,operation.deletion_oauth_session_id,
		       operation.deletion_credential_generation,operation.lease_token,
		       parent.lifecycle_state,parent.row_version,parent.deletion_operation_id,
		       parent.deletion_credential_generation
		FROM account_deletion_operations operation
		JOIN oauth_sessions parent ON parent.account_did=operation.owner_did
		WHERE operation.id=$1 AND parent.account_did=$2 AND parent.session_id=$3
	`, operationID, owner, sessionID).Scan(
		&operationState, &operationSession,
		&operationCredentialGeneration, &operationLeaseToken,
		&parentState, &parentVersion, &parentOperationID,
		&parentCredentialGeneration,
	); err != nil {
		t.Fatal(err)
	}
	if operationState != "active" || operationSession != sessionID ||
		operationCredentialGeneration != credentialGeneration || operationLeaseToken != leaseToken ||
		parentState != "deletion_only" || parentVersion != 6 || parentOperationID != operationID ||
		parentCredentialGeneration != credentialGeneration {
		t.Fatalf(
			"corrected deletion authority operation=%s/%s/%d/%s parent=%s/%d/%s/%d",
			operationState, operationSession, operationCredentialGeneration, operationLeaseToken,
			parentState, parentVersion, parentOperationID, parentCredentialGeneration,
		)
	}
}

func TestOAuthSessionCoordinatorCombinesActiveEffectsAndSessionPersistence(t *testing.T) {
	pool := withAuthSchema(t)
	for _, path := range []string{
		"../../migrations/000039_owner_effects_terminal_purge.up.sql",
		"../../migrations/000045_tap_ingestion_durability.up.sql",
		"../../migrations/000049_pds_effect_action.up.sql",
		"../../migrations/000050_pds_effect_source_reconciliation.up.sql",
	} {
		migration, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
			t.Fatal(err)
		}
	}
	owners := newAuthOwnerStore(t, pool)
	storeConfig := testStoreConfig()
	storeConfig.OwnerLifecycles = owners
	store := auth.NewPostgresAuthStore(pool, storeConfig)
	privateKey, err := atcrypto.GeneratePrivateKeyP256()
	if err != nil {
		t.Fatal(err)
	}
	owner := syntax.DID("did:plc:combined-effect-session")
	participant := syntax.DID("did:plc:combined-effect-participant")
	sessionID := "combined-parent"
	data := validOAuthSession(owner, sessionID)
	data.AccessToken = "access-v1"
	data.RefreshToken = "refresh-v1"
	data.DPoPPrivateKeyMultibase = privateKey.Multibase()
	for _, item := range []struct {
		owner      syntax.DID
		generation int64
	}{
		{owner: owner, generation: 2},
		{owner: participant, generation: 4},
	} {
		if _, err := pool.Exec(context.Background(), `
			INSERT INTO owner_lifecycles(
				owner_did,state,generation,auth_epoch,transition_reason,
				transitioned_at,created_at,updated_at
			) VALUES($1,'active',$2,1,'test',now(),now(),now())
		`, item.owner, item.generation); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO oauth_sessions(
			account_did,session_id,data,lifecycle_state,owner_generation,auth_epoch,
			row_version,absolute_expires_at,created_at,updated_at
		) VALUES($1,$2,$3,'active',2,1,1,now()+interval '1 day',now(),now())
	`, owner, sessionID, data); err != nil {
		t.Fatal(err)
	}
	config := oauth.NewPublicConfig(
		"https://appview.example/oauth/client-metadata.json",
		"https://appview.example/oauth/callback",
		[]string{"atproto"},
	)
	coordinator, err := auth.NewOAuthSessionCoordinator(auth.OAuthSessionCoordinatorOptions{
		App:   &oauth.ClientApp{Client: http.DefaultClient, Config: &config},
		Store: store, Owners: owners, OperationTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	expected := []ownerlifecycle.ExpectedOwner{
		{Owner: participant, Generation: 4},
		{Owner: owner, Generation: 2},
	}
	err = coordinator.WithActiveEffectSession(
		context.Background(),
		expected,
		owner,
		sessionID,
		func(ctx context.Context, session *oauth.ClientSession) error {
			fingerprint := sha256.Sum256([]byte("combined auth effect"))
			if _, err := owners.CreateEffectAttempt(ctx, ownerlifecycle.NewEffectAttempt{
				OperationID: "combined-auth-effect", Owner: owner, OwnerGeneration: 2,
				Kind:               ownerlifecycle.EffectPDSRecord,
				DeterministicKey:   "at://did:plc:combined-effect-session/social.craftsky.feed.post/one",
				RequestFingerprint: fingerprint,
				RecordFingerprint:  fingerprint,
				RemoteDeadline:     time.Now().Add(time.Minute),
			}); err != nil {
				return err
			}
			session.Data.AccessToken = "access-v2"
			session.Data.RefreshToken = "refresh-v2"
			session.PersistSessionCallback(ctx, session.Data)
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	stored, err := store.LoadActiveSession(context.Background(), owner, sessionID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.RowVersion != 2 || stored.Data.RefreshToken != "refresh-v2" {
		t.Fatalf("stored combined session = version %d refresh %q", stored.RowVersion, stored.Data.RefreshToken)
	}
}

func TestOAuthSessionCoordinatorSerializesRefreshPersistenceAcrossPools(t *testing.T) {
	poolA := withAuthSchema(t)
	poolB, err := pgxpool.NewWithConfig(context.Background(), poolA.Config().Copy())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(poolB.Close)
	ownersA := newAuthOwnerStore(t, poolA)
	ownersB := newAuthOwnerStore(t, poolB)
	storeConfigA := testStoreConfig()
	storeConfigA.OwnerLifecycles = ownersA
	storeConfigB := testStoreConfig()
	storeConfigB.OwnerLifecycles = ownersB
	storeA := auth.NewPostgresAuthStore(poolA, storeConfigA)
	storeB := auth.NewPostgresAuthStore(poolB, storeConfigB)

	privateKey, err := atcrypto.GeneratePrivateKeyP256()
	if err != nil {
		t.Fatal(err)
	}
	owner := syntax.DID("did:plc:serialized-refresh")
	sessionID := "serialized-parent"
	data := validOAuthSession(owner, sessionID)
	data.AccessToken = "access-v1"
	data.RefreshToken = "refresh-v1"
	data.DPoPPrivateKeyMultibase = privateKey.Multibase()
	if _, err := poolA.Exec(context.Background(), `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,
			transitioned_at,created_at,updated_at
		) VALUES($1,'active',1,1,'test',now(),now(),now())
	`, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := poolA.Exec(context.Background(), `
		INSERT INTO oauth_sessions(
			account_did,session_id,data,lifecycle_state,owner_generation,auth_epoch,
			row_version,absolute_expires_at,created_at,updated_at
		) VALUES($1,$2,$3,'active',1,1,1,now()+interval '1 day',now(),now())
	`, owner, sessionID, data); err != nil {
		t.Fatal(err)
	}
	config := oauth.NewPublicConfig(
		"https://appview.example/oauth/client-metadata.json",
		"https://appview.example/oauth/callback",
		[]string{"atproto"},
	)
	app := &oauth.ClientApp{Client: http.DefaultClient, Config: &config}
	coordinatorA, err := auth.NewOAuthSessionCoordinator(auth.OAuthSessionCoordinatorOptions{
		App: app, Store: storeA, Owners: ownersA, OperationTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	coordinatorB, err := auth.NewOAuthSessionCoordinator(auth.OAuthSessionCoordinatorOptions{
		App: app, Store: storeB, Owners: ownersB, OperationTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}

	firstPersisted := make(chan struct{})
	releaseFirst := make(chan struct{})
	firstDone := make(chan error, 1)
	go func() {
		firstDone <- coordinatorA.WithActiveSession(context.Background(), owner, sessionID, func(ctx context.Context, session *oauth.ClientSession) error {
			if session.Data.AccessToken != "access-v1" {
				return errors.New("first operation did not load version one")
			}
			session.Data.AccessToken = "access-v2"
			session.Data.RefreshToken = "refresh-v2"
			session.PersistSessionCallback(ctx, session.Data)
			close(firstPersisted)
			<-releaseFirst
			return nil
		})
	}()
	<-firstPersisted

	secondEntered := make(chan struct{})
	secondDone := make(chan error, 1)
	go func() {
		secondDone <- coordinatorB.WithActiveSession(context.Background(), owner, sessionID, func(_ context.Context, session *oauth.ClientSession) error {
			if session.Data.AccessToken != "access-v2" || session.Data.RefreshToken != "refresh-v2" {
				return errors.New("second operation resumed stale rotating credentials")
			}
			close(secondEntered)
			return nil
		})
	}()
	select {
	case <-secondEntered:
		close(releaseFirst)
		t.Fatal("second coordinator crossed the first parent-session operation")
	case <-time.After(75 * time.Millisecond):
	}
	close(releaseFirst)
	if err := <-firstDone; err != nil {
		t.Fatalf("first coordinated operation: %v", err)
	}
	if err := <-secondDone; err != nil {
		t.Fatalf("second coordinated operation: %v", err)
	}

	stored, err := storeA.LoadActiveSession(context.Background(), owner, sessionID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.RowVersion != 2 || stored.Data.RefreshToken != "refresh-v2" {
		t.Fatalf("stored session = version %d refresh %q", stored.RowVersion, stored.Data.RefreshToken)
	}
}

func TestOAuthSessionCoordinatorResumesOnlyExactDeletionCredential(t *testing.T) {
	pool := withAuthSchema(t)
	owners := newAuthOwnerStore(t, pool)
	storeConfig := testStoreConfig()
	storeConfig.OwnerLifecycles = owners
	store := auth.NewPostgresAuthStore(pool, storeConfig)
	privateKey, err := atcrypto.GeneratePrivateKeyP256()
	if err != nil {
		t.Fatal(err)
	}
	owner := syntax.DID("did:plc:exact-deletion-session")
	sessionID := "deletion-parent"
	operationID := uuid.New()
	leaseToken := uuid.New()
	credentialGeneration := int64(4)
	data := validOAuthSession(owner, sessionID)
	data.AccessToken = "access-v1"
	data.RefreshToken = "refresh-v1"
	data.DPoPPrivateKeyMultibase = privateKey.Multibase()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,
			transitioned_at,created_at,updated_at
		) VALUES($1,'deleting',3,2,'test',now(),now(),now())
	`, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO oauth_sessions(
			account_did,session_id,data,lifecycle_state,owner_generation,auth_epoch,
			row_version,absolute_expires_at,deletion_operation_id,
			deletion_credential_generation,created_at,updated_at
		) VALUES($1,$2,$3,'deletion_only',2,2,1,now()+interval '1 day',$4,$5,now(),now())
	`, owner, sessionID, data, operationID, credentialGeneration); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO account_deletion_operations(
			id,owner_did,owner_generation,state,accepted_at,
			deletion_oauth_session_id,deletion_credential_generation,
			lease_owner,lease_token,lease_expires_at
		) VALUES($1,$2,1,'active',now(),$3,$4,'test',$5,now()+interval '1 minute')
	`, operationID, owner, sessionID, credentialGeneration, leaseToken); err != nil {
		t.Fatal(err)
	}
	config := oauth.NewPublicConfig(
		"https://appview.example/oauth/client-metadata.json",
		"https://appview.example/oauth/callback",
		[]string{"atproto"},
	)
	coordinator, err := auth.NewOAuthSessionCoordinator(auth.OAuthSessionCoordinatorOptions{
		App:   &oauth.ClientApp{Client: http.DefaultClient, Config: &config},
		Store: store, Owners: owners, OperationTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	authority := auth.DeletionSessionAuthority{
		DeletionCredentialBinding: auth.DeletionCredentialBinding{
			OperationID: operationID, SessionID: sessionID,
			CredentialGeneration: credentialGeneration,
		},
		OwnerGeneration: 1, LeaseToken: leaseToken,
	}
	if err := coordinator.WithDeletionSession(
		context.Background(), owner, authority,
		func(ctx context.Context, session *oauth.ClientSession) error {
			if session.Data.AccessToken != "access-v1" {
				return errors.New("deletion coordinator loaded the wrong credential")
			}
			session.Data.AccessToken = "access-v2"
			session.Data.RefreshToken = "refresh-v2"
			session.PersistSessionCallback(ctx, session.Data)
			return nil
		},
	); err != nil {
		t.Fatal(err)
	}
	var raw []byte
	var rowVersion int64
	if err := pool.QueryRow(context.Background(), `
		SELECT data,row_version FROM oauth_sessions
		WHERE account_did=$1 AND session_id=$2
	`, owner, sessionID).Scan(&raw, &rowVersion); err != nil {
		t.Fatal(err)
	}
	var persisted oauth.ClientSessionData
	if err := json.Unmarshal(raw, &persisted); err != nil {
		t.Fatal(err)
	}
	if rowVersion != 2 || persisted.RefreshToken != "refresh-v2" {
		t.Fatalf("persisted deletion session = version %d refresh %q", rowVersion, persisted.RefreshToken)
	}

	mismatched := authority
	mismatched.CredentialGeneration++
	called := false
	err = coordinator.WithDeletionSession(
		context.Background(), owner, mismatched,
		func(context.Context, *oauth.ClientSession) error { called = true; return nil },
	)
	if !errors.Is(err, auth.ErrOAuthSessionNotFound) || called {
		t.Fatalf("mismatched deletion binding = called %t error %v", called, err)
	}
}
