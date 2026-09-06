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

type fakeOAuthAuthorityVerifier struct {
	authority auth.OAuthAuthority
	err       error
	calls     atomic.Int64
}

type matchingOAuthAuthorityVerifier struct{}

type oauthAuthorityVerifierFunc func(context.Context, syntax.DID) (auth.OAuthAuthority, error)

type recordingAuthorityObserver struct {
	operation string
	result    string
	reason    string
	calls     int
}

func (observer *recordingAuthorityObserver) ObserveAuthorityVerification(operation, result, reason string, _ time.Duration) {
	observer.operation = operation
	observer.result = result
	observer.reason = reason
	observer.calls++
}

func (resolve oauthAuthorityVerifierFunc) ResolveCurrent(
	ctx context.Context,
	did syntax.DID,
) (auth.OAuthAuthority, error) {
	return resolve(ctx, did)
}

func (matchingOAuthAuthorityVerifier) ResolveCurrent(
	_ context.Context,
	did syntax.DID,
) (auth.OAuthAuthority, error) {
	return auth.OAuthAuthority{
		DID: did, PDSOrigin: "https://pds.example.com", IssuerOrigin: "https://auth.example.com",
	}, nil
}

func (verifier *fakeOAuthAuthorityVerifier) ResolveCurrent(
	_ context.Context,
	_ syntax.DID,
) (auth.OAuthAuthority, error) {
	verifier.calls.Add(1)
	return verifier.authority, verifier.err
}

func TestOAuthSessionCoordinatorVerifiesCurrentAuthorityBeforeOperation(t *testing.T) {
	transientErr := errors.New("DID resolution timed out")
	tests := []struct {
		name            string
		currentPDS      string
		currentIssuer   string
		verifyErr       error
		wantExpired     bool
		wantOperation   bool
		wantParentState string
		wantResult      string
		wantReason      string
	}{
		{
			name:       "matching authority proceeds",
			currentPDS: "https://pds.example", currentIssuer: "https://issuer.example",
			wantOperation: true, wantParentState: "active", wantResult: "success", wantReason: "none",
		},
		{
			name:       "changed PDS is stale",
			currentPDS: "https://new-pds.example", currentIssuer: "https://issuer.example",
			wantExpired: true, wantParentState: "revocation_pending", wantResult: "mismatch", wantReason: "pds_changed",
		},
		{
			name:       "changed issuer is stale",
			currentPDS: "https://pds.example", currentIssuer: "https://new-issuer.example",
			wantExpired: true, wantParentState: "revocation_pending", wantResult: "mismatch", wantReason: "issuer_changed",
		},
		{
			name:      "unverified authority is retryable",
			verifyErr: transientErr, wantParentState: "active", wantResult: "error", wantReason: "resolve_failed",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pool := withAuthSchema(t)
			owners := newAuthOwnerStore(t, pool)
			storeConfig := testStoreConfig()
			storeConfig.OwnerLifecycles = owners
			store := auth.NewPostgresAuthStore(pool, storeConfig)
			owner := syntax.DID("did:plc:authority-check")
			sessionID := "authority-parent"
			data := validOAuthSession(owner, sessionID)
			data.HostURL = "https://pds.example"
			data.AuthServerURL = "https://issuer.example"
			data.AuthServerTokenEndpoint = "https://issuer.example/oauth/token"
			data.AuthServerRevocationEndpoint = "https://issuer.example/oauth/revoke"
			privateKey, err := atcrypto.GeneratePrivateKeyP256()
			if err != nil {
				t.Fatal(err)
			}
			data.DPoPPrivateKeyMultibase = privateKey.Multibase()

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
			`, owner, sessionID, []byte("authority-child")); err != nil {
				t.Fatal(err)
			}

			verifier := &fakeOAuthAuthorityVerifier{
				authority: auth.OAuthAuthority{
					DID: owner, PDSOrigin: test.currentPDS, IssuerOrigin: test.currentIssuer,
				},
				err: test.verifyErr,
			}
			config := oauth.NewPublicConfig(
				"https://appview.example/oauth/client-metadata.json",
				"https://appview.example/oauth/callback",
				[]string{"atproto"},
			)
			observer := &recordingAuthorityObserver{}
			coordinator, err := auth.NewOAuthSessionCoordinator(auth.OAuthSessionCoordinatorOptions{
				App:   &oauth.ClientApp{Client: http.DefaultClient, Config: &config},
				Store: store, Owners: owners, AuthorityVerifier: verifier,
				Observer:         observer,
				OperationTimeout: 5 * time.Second,
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
			if verifier.calls.Load() != 1 {
				t.Fatalf("authority verifier calls = %d, want 1", verifier.calls.Load())
			}
			if observer.calls != 1 || observer.operation != "session_select" || observer.result != test.wantResult || observer.reason != test.wantReason {
				t.Fatalf("authority metric = calls:%d operation:%q result:%q reason:%q", observer.calls, observer.operation, observer.result, observer.reason)
			}
			if test.wantExpired != errors.Is(err, auth.ErrPDSSessionExpired) {
				t.Fatalf("coordinator error = %v, expired = %t", err, test.wantExpired)
			}
			if test.verifyErr != nil && !errors.Is(err, test.verifyErr) {
				t.Fatalf("coordinator error = %v, want retryable %v", err, test.verifyErr)
			}
			if operationCalled != test.wantOperation {
				t.Fatalf("operation called = %t, want %t", operationCalled, test.wantOperation)
			}

			var parentState string
			if err := pool.QueryRow(context.Background(), `
				SELECT lifecycle_state FROM oauth_sessions
				WHERE account_did=$1 AND session_id=$2
			`, owner, sessionID).Scan(&parentState); err != nil {
				t.Fatal(err)
			}
			if parentState != test.wantParentState {
				t.Fatalf("parent state = %q, want %q", parentState, test.wantParentState)
			}
			if test.wantExpired {
				err = coordinator.WithActiveSession(context.Background(), owner, sessionID, func(context.Context, *oauth.ClientSession) error {
					t.Fatal("already-stale parent reached credential-bearing operation")
					return nil
				})
				if !errors.Is(err, auth.ErrPDSSessionExpired) {
					t.Fatalf("repeated stale selection error=%v, want expired", err)
				}
				if observer.calls != 2 || observer.result != "mismatch" || observer.reason != "already_stale" {
					t.Fatalf("repeated stale metric = calls:%d result:%q reason:%q", observer.calls, observer.result, observer.reason)
				}
				if verifier.calls.Load() != 1 {
					t.Fatalf("already-stale selection re-resolved authority %d times, want original check only", verifier.calls.Load())
				}
			}
		})
	}
}

func TestOAuthSessionCoordinatorFencesOnlyExactStaleParentUnderConcurrentRetries(t *testing.T) {
	pool := withAuthSchema(t)
	owners := newAuthOwnerStore(t, pool)
	storeConfig := testStoreConfig()
	storeConfig.OwnerLifecycles = owners
	store := auth.NewPostgresAuthStore(pool, storeConfig)
	owner := syntax.DID("did:plc:exact-stale-parent")
	otherOwner := syntax.DID("did:plc:same-parent-id-other-owner")
	const (
		generation  = int64(7)
		authEpoch   = int64(11)
		staleID     = "stale-parent"
		currentID   = "current-parent"
		correctedID = "corrected-parent"
	)
	for _, did := range []syntax.DID{owner, otherOwner} {
		if _, err := pool.Exec(context.Background(), `
			INSERT INTO owner_lifecycles(
				owner_did,state,generation,auth_epoch,transition_reason,
				transitioned_at,created_at,updated_at
			) VALUES($1,'active',$2,$3,'test',now(),now(),now())
		`, did, generation, authEpoch); err != nil {
			t.Fatal(err)
		}
	}
	privateKey, err := atcrypto.GeneratePrivateKeyP256()
	if err != nil {
		t.Fatal(err)
	}
	parentData := func(did syntax.DID, sessionID, pds, issuer string) oauth.ClientSessionData {
		data := validOAuthSession(did, sessionID)
		data.HostURL = pds
		data.AuthServerURL = issuer
		data.AuthServerTokenEndpoint = issuer + "/oauth/token"
		data.AuthServerRevocationEndpoint = issuer + "/oauth/revoke"
		data.DPoPPrivateKeyMultibase = privateKey.Multibase()
		return data
	}
	seedParent := func(did syntax.DID, sessionID string, version int64, data oauth.ClientSessionData, lastSeen time.Time) {
		t.Helper()
		if _, err := pool.Exec(context.Background(), `
			INSERT INTO oauth_sessions(
				account_did,session_id,data,lifecycle_state,owner_generation,auth_epoch,
				row_version,absolute_expires_at,created_at,updated_at
			) VALUES($1,$2,$3,'active',$4,$5,$6,now()+interval '1 day',now(),now())
		`, did, sessionID, data, generation, authEpoch, version); err != nil {
			t.Fatal(err)
		}
		for child := 1; child <= 2; child++ {
			if _, err := pool.Exec(context.Background(), `
				INSERT INTO craftsky_sessions(
					token_hash,account_did,oauth_session_id,lifecycle_state,auth_epoch,
					last_seen_at,idle_expires_at
				) VALUES(convert_to($1,'UTF8'),$2,$3,'active',$4,$5,now()+interval '1 day')
			`, did.String()+sessionID+string(rune('0'+child)), did, sessionID, authEpoch, lastSeen); err != nil {
				t.Fatal(err)
			}
		}
	}
	now := time.Now().UTC()
	seedParent(owner, staleID, 13, parentData(owner, staleID, "https://pds-a.example", "https://issuer-a.example"), now)
	seedParent(owner, currentID, 23, parentData(owner, currentID, "https://pds-b.example", "https://issuer-b.example"), now.Add(-time.Minute))
	seedParent(owner, correctedID, 17, parentData(owner, correctedID, "https://pds-a.example", "https://issuer-a.example"), now.Add(-2*time.Minute))
	seedParent(otherOwner, staleID, 31, parentData(otherOwner, staleID, "https://pds-a.example", "https://issuer-a.example"), now)
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO oauth_sessions(
			account_did,session_id,data,lifecycle_state,owner_generation,auth_epoch,
			row_version,absolute_expires_at,created_at,updated_at
		) VALUES($1,'obsolete-generation',$2,'active',$3,$4,41,
		         now()+interval '1 day',now(),now())
	`, owner, parentData(owner, "obsolete-generation", "https://pds-b.example", "https://issuer-b.example"), generation-1, authEpoch); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO craftsky_sessions(
			token_hash,account_did,oauth_session_id,lifecycle_state,auth_epoch,
			last_seen_at,idle_expires_at
		) VALUES(convert_to('obsolete-generation-child','UTF8'),$1,
		         'obsolete-generation','active',$2,$3,now()+interval '1 day')
	`, owner, authEpoch, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}

	config := oauth.NewPublicConfig(
		"https://appview.example/oauth/client-metadata.json",
		"https://appview.example/oauth/callback",
		[]string{"atproto"},
	)
	currentAuthority := auth.OAuthAuthority{
		DID: owner, PDSOrigin: "https://pds-b.example", IssuerOrigin: "https://issuer-b.example",
	}
	coordinator, err := auth.NewOAuthSessionCoordinator(auth.OAuthSessionCoordinatorOptions{
		App:   &oauth.ClientApp{Client: http.DefaultClient, Config: &config},
		Store: store, Owners: owners,
		AuthorityVerifier: &fakeOAuthAuthorityVerifier{authority: currentAuthority},
		OperationTimeout:  5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	for range 2 {
		go func() {
			<-start
			results <- coordinator.WithActiveSession(
				context.Background(), owner, staleID,
				func(context.Context, *oauth.ClientSession) error {
					return errors.New("stale operation must not run")
				},
			)
		}()
	}
	close(start)
	for range 2 {
		if err := <-results; !errors.Is(err, auth.ErrPDSSessionExpired) {
			t.Fatalf("concurrent stale result = %v, want ErrPDSSessionExpired", err)
		}
	}
	if err := coordinator.WithActiveSession(
		context.Background(), owner, staleID,
		func(context.Context, *oauth.ClientSession) error {
			return errors.New("repeated stale operation must not run")
		},
	); !errors.Is(err, auth.ErrPDSSessionExpired) {
		t.Fatalf("repeated stale result = %v, want ErrPDSSessionExpired", err)
	}

	selector := auth.NewBackgroundSessionSelector(pool)
	selected, err := selector.Select(context.Background(), owner)
	if err != nil {
		t.Fatal(err)
	}
	if selected != currentID {
		t.Fatalf("eligible background parent = %q, want %q", selected, currentID)
	}
	validOperationCalled := false
	if err := coordinator.WithActiveSession(
		context.Background(), owner, currentID,
		func(context.Context, *oauth.ClientSession) error {
			validOperationCalled = true
			return nil
		},
	); err != nil {
		t.Fatalf("independently valid parent: %v", err)
	}
	if !validOperationCalled {
		t.Fatal("independently valid parent operation did not run")
	}

	entered := make(chan struct{})
	release := make(chan struct{})
	var resolveCalls atomic.Int64
	correctedCoordinator, err := auth.NewOAuthSessionCoordinator(auth.OAuthSessionCoordinatorOptions{
		App:   &oauth.ClientApp{Client: http.DefaultClient, Config: &config},
		Store: store, Owners: owners,
		AuthorityVerifier: oauthAuthorityVerifierFunc(func(_ context.Context, did syntax.DID) (auth.OAuthAuthority, error) {
			if did != owner {
				t.Fatalf("resolved DID = %q, want %q", did, owner)
			}
			if resolveCalls.Add(1) == 1 {
				close(entered)
				<-release
			}
			return currentAuthority, nil
		}),
		OperationTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	correctedDone := make(chan error, 1)
	correctedOperationCalled := false
	go func() {
		correctedDone <- correctedCoordinator.WithActiveSession(
			context.Background(), owner, correctedID,
			func(context.Context, *oauth.ClientSession) error {
				correctedOperationCalled = true
				return nil
			},
		)
	}()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("coordinator did not reach stale authority proof")
	}
	correctedData := parentData(owner, correctedID, "https://pds-b.example", "https://issuer-b.example")
	if command, err := pool.Exec(context.Background(), `
		UPDATE oauth_sessions
		SET data=$3,row_version=row_version+1,updated_at=now()
		WHERE account_did=$1 AND session_id=$2 AND row_version=17
	`, owner, correctedID, correctedData); err != nil {
		t.Fatal(err)
	} else if command.RowsAffected() != 1 {
		t.Fatalf("corrected rows = %d, want 1", command.RowsAffected())
	}
	close(release)
	if err := <-correctedDone; err != nil {
		t.Fatalf("corrected parent result = %v, want success", err)
	}
	if !correctedOperationCalled || resolveCalls.Load() != 2 {
		t.Fatalf("corrected operation=%t authority calls=%d, want true/2", correctedOperationCalled, resolveCalls.Load())
	}

	type parentState struct {
		state                      string
		version, generation, epoch int64
	}
	readParent := func(did syntax.DID, sessionID string) parentState {
		t.Helper()
		var got parentState
		if err := pool.QueryRow(context.Background(), `
			SELECT lifecycle_state,row_version,owner_generation,auth_epoch
			FROM oauth_sessions WHERE account_did=$1 AND session_id=$2
		`, did, sessionID).Scan(&got.state, &got.version, &got.generation, &got.epoch); err != nil {
			t.Fatal(err)
		}
		return got
	}
	for _, check := range []struct {
		name string
		did  syntax.DID
		id   string
		want parentState
	}{
		{"stale exact parent", owner, staleID, parentState{"revocation_pending", 14, generation, authEpoch}},
		{"independent current parent", owner, currentID, parentState{"active", 23, generation, authEpoch}},
		{"corrected row version", owner, correctedID, parentState{"active", 18, generation, authEpoch}},
		{"same parent ID other DID", otherOwner, staleID, parentState{"active", 31, generation, authEpoch}},
	} {
		if got := readParent(check.did, check.id); got != check.want {
			t.Fatalf("%s state = %+v, want %+v", check.name, got, check.want)
		}
	}
	for _, check := range []struct {
		did      syntax.DID
		id, want string
	}{
		{owner, staleID, "revoked"},
		{owner, currentID, "active"},
		{owner, correctedID, "active"},
		{otherOwner, staleID, "active"},
	} {
		var states []string
		rows, err := pool.Query(context.Background(), `
			SELECT lifecycle_state FROM craftsky_sessions
			WHERE account_did=$1 AND oauth_session_id=$2 ORDER BY token_hash
		`, check.did, check.id)
		if err != nil {
			t.Fatal(err)
		}
		for rows.Next() {
			var state string
			if err := rows.Scan(&state); err != nil {
				rows.Close()
				t.Fatal(err)
			}
			states = append(states, state)
		}
		rows.Close()
		if len(states) != 2 || states[0] != check.want || states[1] != check.want {
			t.Fatalf("child states for %s/%s = %v, want [%s %s]", check.did, check.id, states, check.want, check.want)
		}
	}
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
		Store: store, Owners: owners, AuthorityVerifier: matchingOAuthAuthorityVerifier{},
		OperationTimeout: 5 * time.Second,
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
		Store: store, Owners: owners, AuthorityVerifier: matchingOAuthAuthorityVerifier{},
		OperationTimeout: 5 * time.Second,
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
		Store: store, Owners: owners, AuthorityVerifier: matchingOAuthAuthorityVerifier{},
		OperationTimeout: 5 * time.Second,
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
		Store: store, Owners: owners, AuthorityVerifier: matchingOAuthAuthorityVerifier{},
		OperationTimeout: 5 * time.Second,
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
		Store: store, Owners: owners, AuthorityVerifier: matchingOAuthAuthorityVerifier{},
		OperationTimeout: 5 * time.Second,
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
		Store: store, Owners: owners, AuthorityVerifier: matchingOAuthAuthorityVerifier{},
		OperationTimeout: 5 * time.Second,
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
		Store: store, Owners: owners, AuthorityVerifier: matchingOAuthAuthorityVerifier{},
		OperationTimeout: 5 * time.Second,
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
		App: app, Store: storeA, Owners: ownersA, AuthorityVerifier: matchingOAuthAuthorityVerifier{},
		OperationTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	coordinatorB, err := auth.NewOAuthSessionCoordinator(auth.OAuthSessionCoordinatorOptions{
		App: app, Store: storeB, Owners: ownersB, AuthorityVerifier: matchingOAuthAuthorityVerifier{},
		OperationTimeout: 5 * time.Second,
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
		Store: store, Owners: owners, AuthorityVerifier: matchingOAuthAuthorityVerifier{},
		OperationTimeout: 5 * time.Second,
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
