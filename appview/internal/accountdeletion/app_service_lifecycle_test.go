package accountdeletion

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/testdb"
)

func TestAppServiceOwnsDeletionCredentialAcrossLifecycle(t *testing.T) {
	pool := testdb.WithSchema(t, "")
	applyAllAccountDeletionTestMigrations(t, pool)
	ctx := context.Background()
	now := time.Date(2026, 8, 14, 19, 0, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:deletion-lifecycle")
	fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	owners, err := ownerlifecycle.NewStore(pool, fencer, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	deletionStore := NewStore(pool, func() time.Time { return now })
	children, err := auth.NewCraftskySessionStoreWithConfig(pool, auth.CraftskySessionConfig{
		Inactivity:            24 * time.Hour,
		ActivityWriteInterval: time.Hour,
		RecoveryAuthorization: deletionStore,
	})
	if err != nil {
		t.Fatal(err)
	}
	sessions, err := auth.NewSessionLifecycleService(auth.SessionLifecycleOptions{
		Pool: pool, Owners: owners, Sessions: children, DeletionExemption: deletionStore,
		Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	authStore := auth.NewPostgresAuthStore(pool, auth.StoreConfig{
		SessionAbsoluteLifetime: 24 * time.Hour,
		AuthRequestExpiry:       10 * time.Minute,
		OwnerLifecycles:         owners,
		EndpointValidator:       deletionTestEndpointValidator{},
	})
	starter := &recordingDeletionOAuthStarter{authURL: "https://auth.example/authorize"}
	departureCalled := false
	service, err := NewAppService(AppServiceOptions{
		Pool: pool, Store: deletionStore, OAuth: starter,
		Owners: owners, Sessions: sessions, OAuthStore: authStore,
		DepartureParticipant: func(
			_ context.Context,
			_ pgx.Tx,
			before ownerlifecycle.Lifecycle,
			after ownerlifecycle.Lifecycle,
		) error {
			if before.State != ownerlifecycle.StateActive ||
				after.State != ownerlifecycle.StateDeletionPending {
				t.Fatalf("departure transition = %s -> %s", before.State, after.State)
			}
			departureCalled = true
			return nil
		},
		Now: func() time.Time { return now }, IntentTTL: 10 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,
			transitioned_at,created_at,updated_at
		) VALUES($1,'active',1,1,'test',$2,$2,$2)
	`, owner, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO atproto_identity_cache(did,handle,handle_lower,resolved_at,updated_at)
		VALUES($1,'alice.example','alice.example',$2,$2)
	`, owner, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO oauth_sessions(
			account_did,session_id,data,lifecycle_state,owner_generation,auth_epoch,
			row_version,absolute_expires_at,created_at,updated_at
		) VALUES($1,'ordinary-parent','{}','active',1,1,1,$2,$3,$3)
	`, owner, time.Now().UTC().Add(24*time.Hour), now); err != nil {
		t.Fatal(err)
	}
	ordinaryToken, err := children.Create(ctx, owner.String(), "ordinary-parent", "device-delete")
	if err != nil {
		t.Fatal(err)
	}

	intent, err := service.CreateIntent(ctx, CreateIntentParams{Owner: owner, DeviceID: "device-delete"})
	if err != nil {
		t.Fatal(err)
	}
	if !departureCalled {
		t.Fatal("account deletion intent did not run bounded departure participant")
	}
	jobID := uuid.MustParse(intent.JobID)
	if starter.owner != owner || starter.jobID != jobID || starter.deviceID != "device-delete" {
		t.Fatalf("OAuth start scope = owner %s job %s device %q", starter.owner, starter.jobID, starter.deviceID)
	}
	assertDeletionLifecycle(t, pool, owner, "deletion_pending", 2, 1)
	assertDeletionAuthState(t, pool, owner, "ordinary-parent", "active")
	if _, err := children.Lookup(ctx, ordinaryToken); !errors.Is(err, auth.ErrCraftskySessionNotFound) {
		t.Fatalf("ordinary lookup while deletion is pending = %v, want unavailable", err)
	}
	if info, err := children.LookupRecovery(ctx, ordinaryToken); err != nil || info.DID != owner {
		t.Fatalf("recovery lookup while deletion is pending = %+v, %v", info, err)
	}

	var operationGeneration int64
	if err := pool.QueryRow(ctx, `
		SELECT owner_generation FROM account_deletion_operations WHERE id=$1
	`, jobID).Scan(&operationGeneration); err != nil {
		t.Fatal(err)
	}
	if operationGeneration != 1 {
		t.Fatalf("originating operation generation = %d, want 1", operationGeneration)
	}

	var attempt auth.CallbackAttempt
	var proof string
	err = owners.WithExistingAuth(ctx, owner, func(authCtx context.Context, authority ownerlifecycle.Lifecycle) error {
		requestCtx := auth.WithAccountDeletionAuthRequestAuthority(
			authCtx, owner, authority.Generation, authority.AuthEpoch, jobID, "device-delete",
		)
		request := oauth.AuthRequestData{
			State: "deletion-parent", RequestURI: "urn:request:deletion-parent",
			AccountDID: &owner, AuthServerURL: "https://auth.example",
			AuthServerTokenEndpoint: "https://auth.example/oauth/token",
		}
		if err := authStore.SaveAuthRequestInfo(requestCtx, request); err != nil {
			return err
		}
		attemptID, err := authStore.BeginExchange(authCtx, request.State)
		if err != nil {
			return err
		}
		attempt = auth.CallbackAttempt{
			State: request.State, AttemptID: attemptID, Owner: owner,
			OwnerGeneration: authority.Generation, AuthEpoch: authority.AuthEpoch,
			Purpose: auth.AccountDeletionOAuthPurpose,
		}
		callbackCtx := auth.WithCallbackAttempt(authCtx, attempt)
		if err := authStore.SaveSession(callbackCtx, deletionOAuthSession(owner, request.State)); err != nil {
			return err
		}
		result, err := service.CompleteAttempt(callbackCtx, auth.AccountDeletionAuthRequest{
			Purpose: auth.AccountDeletionOAuthPurpose, JobID: jobID.String(), Owner: owner,
		}, attempt)
		if err != nil {
			return err
		}
		if result.Proof == "" {
			return errors.New("empty deletion proof")
		}
		proof = result.Proof
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	assertDeletionAuthState(t, pool, owner, attempt.State, "deletion_only")
	var credentialGeneration int64
	if err := pool.QueryRow(ctx, `
		SELECT deletion_credential_generation
		FROM account_deletion_operations WHERE id=$1
	`, jobID).Scan(&credentialGeneration); err != nil {
		t.Fatal(err)
	}
	if credentialGeneration != 1 {
		t.Fatalf("credential generation = %d, want 1", credentialGeneration)
	}

	if err := service.Accept(ctx, AcceptParams{
		JobID: jobID.String(), Owner: owner,
		ReauthProof: proof, ConfirmationHandle: "@alice.example",
	}); err != nil {
		t.Fatal(err)
	}
	assertDeletionLifecycle(t, pool, owner, "deleting", 3, 2)
	assertDeletionAuthState(t, pool, owner, "ordinary-parent", "revocation_pending")
	assertDeletionAuthState(t, pool, owner, attempt.State, "deletion_only")
	var boundSession string
	if err := pool.QueryRow(ctx, `
		SELECT deletion_oauth_session_id FROM account_deletion_operations WHERE id=$1
	`, jobID).Scan(&boundSession); err != nil {
		t.Fatal(err)
	}
	if boundSession != attempt.State {
		t.Fatalf("accepted credential = %q, want %q", boundSession, attempt.State)
	}
	if err := sessions.RevokeAllForDID(ctx, owner); err != nil {
		t.Fatal(err)
	}
	assertDeletionLifecycle(t, pool, owner, "deleting", 3, 3)
	assertDeletionAuthState(t, pool, owner, attempt.State, "deletion_only")
	var deletionEpoch int64
	if err := pool.QueryRow(ctx, `
		SELECT auth_epoch FROM oauth_sessions WHERE account_did=$1 AND session_id=$2
	`, owner, attempt.State).Scan(&deletionEpoch); err != nil {
		t.Fatal(err)
	}
	if deletionEpoch != 3 {
		t.Fatalf("accepted deletion credential auth epoch = %d, want 3", deletionEpoch)
	}

	leaseToken := uuid.New()
	if _, err := pool.Exec(ctx, `
		UPDATE account_deletion_operations
		SET lease_owner='test',lease_token=$2,lease_expires_at=$3,next_attempt_at=$4
		WHERE id=$1
	`, jobID, leaseToken, now.Add(time.Minute), now); err != nil {
		t.Fatal(err)
	}
	if err := service.CompleteAccepted(ctx, ClaimedOperation{
		JobID: jobID, Owner: owner, OwnerGeneration: 1,
		LeaseToken: leaseToken, LeaseExpiresAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	assertDeletionLifecycle(t, pool, owner, "departed", 4, 4)
	assertDeletionAuthState(t, pool, owner, attempt.State, "revocation_pending")
	assertCount(t, pool, `SELECT count(*) FROM account_deletion_operations WHERE id=$1`, jobID, 0)
}

func TestCancelDeletionIntentRestoresOrdinarySessionAndRevokesOnlyDeletionCredential(t *testing.T) {
	pool := testdb.WithSchema(t, "")
	applyAllAccountDeletionTestMigrations(t, pool)
	ctx := context.Background()
	now := time.Date(2026, 8, 14, 20, 0, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:deletion-cancel")
	fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	owners, err := ownerlifecycle.NewStore(pool, fencer, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	deletionStore := NewStore(pool, func() time.Time { return now })
	children, err := auth.NewCraftskySessionStoreWithConfig(pool, auth.CraftskySessionConfig{
		Inactivity:            24 * time.Hour,
		ActivityWriteInterval: time.Hour,
		RecoveryAuthorization: deletionStore,
	})
	if err != nil {
		t.Fatal(err)
	}
	sessions, err := auth.NewSessionLifecycleService(auth.SessionLifecycleOptions{
		Pool: pool, Owners: owners, Sessions: children, DeletionExemption: deletionStore,
		Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	authStore := auth.NewPostgresAuthStore(pool, auth.StoreConfig{
		SessionAbsoluteLifetime: 24 * time.Hour,
		AuthRequestExpiry:       10 * time.Minute,
		OwnerLifecycles:         owners,
		EndpointValidator:       deletionTestEndpointValidator{},
	})
	service, err := NewAppService(AppServiceOptions{
		Pool: pool, Store: deletionStore,
		OAuth:  &recordingDeletionOAuthStarter{authURL: "https://auth.example/authorize"},
		Owners: owners, Sessions: sessions, OAuthStore: authStore,
		DepartureParticipant: noOpDeletionDepartureParticipant,
		Now:                  func() time.Time { return now }, IntentTTL: 10 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,
			transitioned_at,created_at,updated_at
		) VALUES($1,'active',1,1,'test',$2,$2,$2)
	`, owner, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO atproto_identity_cache(did,handle,handle_lower,resolved_at,updated_at)
		VALUES($1,'cancel.example','cancel.example',$2,$2)
	`, owner, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_profiles(did,crafts,record_cid)
		VALUES($1,'{}','bafy-cancel-profile')
	`, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO oauth_sessions(
			account_did,session_id,data,lifecycle_state,owner_generation,auth_epoch,
			row_version,absolute_expires_at,created_at,updated_at
		) VALUES($1,'ordinary-parent','{}','active',1,1,1,$2,$3,$3)
	`, owner, time.Now().UTC().Add(24*time.Hour), now); err != nil {
		t.Fatal(err)
	}
	ordinaryToken, err := children.Create(ctx, owner.String(), "ordinary-parent", "device-cancel")
	if err != nil {
		t.Fatal(err)
	}
	intent, err := service.CreateIntent(ctx, CreateIntentParams{Owner: owner, DeviceID: "device-cancel"})
	if err != nil {
		t.Fatal(err)
	}
	jobID := uuid.MustParse(intent.JobID)
	if _, err := pool.Exec(ctx, `
		INSERT INTO oauth_sessions(
			account_did,session_id,data,lifecycle_state,owner_generation,auth_epoch,
			row_version,absolute_expires_at,deletion_operation_id,
			deletion_credential_generation,created_at,updated_at
		) VALUES($1,'deletion-parent','{}','deletion_only',2,1,1,$3,$2,1,$4,$4)
	`, owner, jobID, now.Add(24*time.Hour), now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE account_deletion_operations
		SET reauth_oauth_session_id='deletion-parent',
		    deletion_credential_generation=1,intent_proof_hash=decode(repeat('22',32),'hex')
		WHERE id=$1
	`, jobID); err != nil {
		t.Fatal(err)
	}

	if err := service.CancelIntent(ctx, intent.JobID, owner); err != nil {
		t.Fatal(err)
	}

	assertDeletionLifecycle(t, pool, owner, "active", 3, 1)
	assertDeletionAuthState(t, pool, owner, "ordinary-parent", "active")
	assertDeletionAuthState(t, pool, owner, "deletion-parent", "revocation_pending")
	if info, err := children.Lookup(ctx, ordinaryToken); err != nil || info.DID != owner {
		t.Fatalf("ordinary session after cancellation = %+v, %v", info, err)
	}
	assertCount(t, pool, `SELECT count(*) FROM account_deletion_operations WHERE id=$1`, jobID, 0)

	secondIntent, err := service.CreateIntent(ctx, CreateIntentParams{Owner: owner, DeviceID: "device-cancel"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM craftsky_profiles WHERE did=$1`, owner); err != nil {
		t.Fatal(err)
	}
	now = now.Add(11 * time.Minute)
	if err := service.ExpireIntent(ctx, uuid.MustParse(secondIntent.JobID), owner); err != nil {
		t.Fatal(err)
	}
	assertDeletionLifecycle(t, pool, owner, "departed", 5, 2)
	assertDeletionAuthState(t, pool, owner, "ordinary-parent", "revocation_pending")
	if _, err := children.LookupRecovery(ctx, ordinaryToken); !errors.Is(err, auth.ErrCraftskySessionNotFound) {
		t.Fatalf("departed canceled session lookup = %v, want unavailable", err)
	}
}

func noOpDeletionDepartureParticipant(
	context.Context,
	pgx.Tx,
	ownerlifecycle.Lifecycle,
	ownerlifecycle.Lifecycle,
) error {
	return nil
}

type recordingDeletionOAuthStarter struct {
	authURL  string
	owner    syntax.DID
	jobID    uuid.UUID
	deviceID string
}

func (starter *recordingDeletionOAuthStarter) StartAccountDeletion(
	_ context.Context,
	owner syntax.DID,
	_ syntax.Handle,
	jobID uuid.UUID,
	deviceID string,
) (string, error) {
	starter.owner = owner
	starter.jobID = jobID
	starter.deviceID = deviceID
	return starter.authURL, nil
}

type deletionTestEndpointValidator struct{}

func (deletionTestEndpointValidator) ValidateOrigin(_ context.Context, raw string) (*url.URL, error) {
	return url.Parse(raw)
}

func (deletionTestEndpointValidator) ValidateOAuthEndpoint(_ context.Context, issuer, endpoint string) (*url.URL, error) {
	issuerURL, err := url.Parse(issuer)
	if err != nil {
		return nil, err
	}
	endpointURL, err := url.Parse(endpoint)
	if err != nil || endpointURL.Scheme != issuerURL.Scheme || endpointURL.Host != issuerURL.Host {
		return nil, errors.New("cross-origin endpoint")
	}
	return endpointURL, nil
}

func deletionOAuthSession(owner syntax.DID, sessionID string) oauth.ClientSessionData {
	return oauth.ClientSessionData{
		AccountDID: owner, SessionID: sessionID,
		HostURL: "https://pds.example", AuthServerURL: "https://auth.example",
		AuthServerTokenEndpoint: "https://auth.example/oauth/token",
	}
}

func assertDeletionLifecycle(
	t *testing.T,
	pool interface {
		QueryRow(context.Context, string, ...any) pgx.Row
	},
	owner syntax.DID,
	state string,
	generation int64,
	authEpoch int64,
) {
	t.Helper()
	var gotState string
	var gotGeneration, gotEpoch int64
	if err := pool.QueryRow(context.Background(), `
		SELECT state,generation,auth_epoch FROM owner_lifecycles WHERE owner_did=$1
	`, owner).Scan(&gotState, &gotGeneration, &gotEpoch); err != nil {
		t.Fatal(err)
	}
	if gotState != state || gotGeneration != generation || gotEpoch != authEpoch {
		t.Fatalf("lifecycle = %s/%d/%d, want %s/%d/%d", gotState, gotGeneration, gotEpoch, state, generation, authEpoch)
	}
}

func assertDeletionAuthState(
	t *testing.T,
	pool interface {
		QueryRow(context.Context, string, ...any) pgx.Row
	},
	owner syntax.DID,
	sessionID string,
	want string,
) {
	t.Helper()
	var got string
	if err := pool.QueryRow(context.Background(), `
		SELECT lifecycle_state FROM oauth_sessions WHERE account_did=$1 AND session_id=$2
	`, owner, sessionID).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("OAuth parent %q state = %q, want %q", sessionID, got, want)
	}
}
