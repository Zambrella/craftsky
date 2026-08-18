package auth_test

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/atcrypto"
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ownerlifecycle"
)

func TestOAuthSessionCoordinatorCombinesActiveEffectsAndSessionPersistence(t *testing.T) {
	pool := withAuthSchema(t)
	for _, path := range []string{
		"../../migrations/000039_owner_effects_terminal_purge.up.sql",
		"../../migrations/000049_pds_effect_action.up.sql",
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
