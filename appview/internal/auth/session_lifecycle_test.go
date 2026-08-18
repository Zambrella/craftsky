package auth_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ownerlifecycle"
)

func TestSessionLifecycleRevokeOneCommitsBeforeAuxiliaryCleanup(t *testing.T) {
	pool := withAuthSchema(t)
	owners := newAuthOwnerStore(t, pool)
	children, err := auth.NewCraftskySessionStoreWithConfig(pool, auth.CraftskySessionConfig{
		Inactivity: 24 * time.Hour, ActivityWriteInterval: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	service, err := auth.NewSessionLifecycleService(auth.SessionLifecycleOptions{
		Pool: pool, Owners: owners, Sessions: children, Now: time.Now,
	})
	if err != nil {
		t.Fatal(err)
	}
	owner := syntax.DID("did:plc:single-logout")
	tokenA := "single-token-a"
	tokenB := "single-token-b"
	seedActiveSessionFamily(t, pool, owner, "parent-single", tokenA, tokenB)

	if err := service.RevokeOne(context.Background(), owner, tokenA, "installation-a"); err != nil {
		t.Fatalf("revoke one: %v", err)
	}
	if _, err := children.Lookup(context.Background(), tokenA); !errors.Is(err, auth.ErrCraftskySessionNotFound) {
		t.Fatalf("revoked bearer remained usable: %v", err)
	}
	if _, err := children.Lookup(context.Background(), tokenB); err != nil {
		t.Fatalf("sibling bearer was revoked: %v", err)
	}
	var parentState string
	var jobs int
	if err := pool.QueryRow(context.Background(), `
		SELECT lifecycle_state FROM oauth_sessions WHERE account_did=$1 AND session_id='parent-single'
	`, owner).Scan(&parentState); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*) FROM auth_auxiliary_cleanup_jobs
		WHERE owner_did=$1 AND kind='installation_push' AND installation_id='installation-a' AND state='pending'
	`, owner).Scan(&jobs); err != nil {
		t.Fatal(err)
	}
	if parentState != "active" || jobs != 1 {
		t.Fatalf("single logout parent=%s jobs=%d", parentState, jobs)
	}
}

func TestSessionLifecycleRevokeAllAdvancesEpochAndInvalidatesEveryArtifact(t *testing.T) {
	pool := withAuthSchema(t)
	owners := newAuthOwnerStore(t, pool)
	children, err := auth.NewCraftskySessionStoreWithConfig(pool, auth.CraftskySessionConfig{
		Inactivity: 24 * time.Hour, ActivityWriteInterval: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	service, err := auth.NewSessionLifecycleService(auth.SessionLifecycleOptions{
		Pool: pool, Owners: owners, Sessions: children, Now: time.Now,
	})
	if err != nil {
		t.Fatal(err)
	}
	owner := syntax.DID("did:plc:all-logout")
	seedActiveSessionFamily(t, pool, owner, "parent-one", "all-token-one")
	seedAdditionalActiveParent(t, pool, owner, "parent-two", "all-token-two")
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO oauth_auth_requests(
			state,data,handoff_mode,purpose,device_id,owner_did,owner_generation,
			auth_epoch,request_uri,request_state
		) VALUES(
			'pending-auth','{}','verified_link','login','device-all',$1,1,1,
			'urn:request:pending-all','ready'
		);
	`, owner); err != nil {
		t.Fatal(err)
	}

	if err := service.RevokeAllForDID(context.Background(), owner); err != nil {
		t.Fatalf("revoke all: %v", err)
	}
	var epoch int64
	var activeParents, activeChildren, liveRequests, accountJobs int
	if err := pool.QueryRow(context.Background(), `SELECT auth_epoch FROM owner_lifecycles WHERE owner_did=$1`, owner).Scan(&epoch); err != nil {
		t.Fatal(err)
	}
	for query, target := range map[string]*int{
		`SELECT count(*) FROM oauth_sessions WHERE account_did=$1 AND lifecycle_state<>'revocation_pending'`: &activeParents,
		`SELECT count(*) FROM craftsky_sessions WHERE account_did=$1 AND lifecycle_state<>'revoked'`:         &activeChildren,
		`SELECT count(*) FROM oauth_auth_requests WHERE owner_did=$1 AND request_state<>'revoked'`:           &liveRequests,
		`SELECT count(*) FROM auth_auxiliary_cleanup_jobs WHERE owner_did=$1 AND kind='account_push'`:        &accountJobs,
	} {
		if err := pool.QueryRow(context.Background(), query, owner).Scan(target); err != nil {
			t.Fatal(err)
		}
	}
	if epoch != 2 || activeParents != 0 || activeChildren != 0 || liveRequests != 0 || accountJobs != 1 {
		t.Fatalf("logout-all epoch=%d parents=%d children=%d requests=%d jobs=%d",
			epoch, activeParents, activeChildren, liveRequests, accountJobs)
	}
}

func TestSessionLifecycleOwnerTransitionPreservesOnlyExactDeletionCredential(t *testing.T) {
	pool := withAuthSchema(t)
	owners := newAuthOwnerStore(t, pool)
	children, err := auth.NewCraftskySessionStoreWithConfig(pool, auth.CraftskySessionConfig{
		Inactivity: 24 * time.Hour, ActivityWriteInterval: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	service, err := auth.NewSessionLifecycleService(auth.SessionLifecycleOptions{
		Pool: pool, Owners: owners, Sessions: children, Now: time.Now,
	})
	if err != nil {
		t.Fatal(err)
	}
	owner := syntax.DID("did:plc:deletion-transition")
	operationID := uuid.MustParse("00000000-0000-4000-8000-000000000821")
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,
			transitioned_at,created_at,updated_at
		) VALUES($1,'deletion_pending',2,1,'deletionIntent',now(),now(),now())
	`, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO oauth_sessions(
			account_did,session_id,data,lifecycle_state,owner_generation,auth_epoch,
			absolute_expires_at,deletion_operation_id,deletion_credential_generation,
			created_at,updated_at
		) VALUES
			($1,'ordinary-parent','{}','active',1,1,now()+interval '1 day',NULL,NULL,now(),now()),
			($1,'deletion-parent','{}','deletion_only',2,1,now()+interval '1 day',$2,3,now(),now());
	`, owner, operationID); err != nil {
		t.Fatal(err)
	}
	ordinaryToken := "ordinary-before-delete"
	ordinaryHash := sha256.Sum256([]byte(ordinaryToken))
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO craftsky_sessions(
			token_hash,account_did,oauth_session_id,lifecycle_state,auth_epoch,last_seen_at,idle_expires_at
		) VALUES($1,$2,'ordinary-parent','active',1,now(),now()+interval '1 day')
	`, ordinaryHash[:], owner); err != nil {
		t.Fatal(err)
	}

	updated, err := owners.TransitionWith(context.Background(), ownerlifecycle.TransitionRequest{
		Owner: owner, ExpectedGeneration: 2, To: ownerlifecycle.StateDeleting, Reason: "deletionAccepted",
	}, service.OwnerTransitionParticipant(&auth.DeletionCredentialBinding{
		OperationID: operationID, SessionID: "deletion-parent", CredentialGeneration: 3,
	}))
	if err != nil {
		t.Fatalf("accept deletion transition: %v", err)
	}
	if updated.State != ownerlifecycle.StateDeleting || updated.AuthEpoch != 2 {
		t.Fatalf("updated lifecycle=%+v", updated)
	}
	var deletionState, ordinaryState, childState string
	var deletionEpoch int64
	if err := pool.QueryRow(context.Background(), `SELECT lifecycle_state,auth_epoch FROM oauth_sessions WHERE account_did=$1 AND session_id='deletion-parent'`, owner).Scan(&deletionState, &deletionEpoch); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT lifecycle_state FROM oauth_sessions WHERE account_did=$1 AND session_id='ordinary-parent'`, owner).Scan(&ordinaryState); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT lifecycle_state FROM craftsky_sessions WHERE token_hash=$1`, ordinaryHash[:]).Scan(&childState); err != nil {
		t.Fatal(err)
	}
	if deletionState != "deletion_only" || deletionEpoch != 2 || ordinaryState != "revocation_pending" || childState != "revoked" {
		t.Fatalf("deletion=%s/%d ordinary=%s child=%s", deletionState, deletionEpoch, ordinaryState, childState)
	}
}

func seedActiveSessionFamily(t *testing.T, pool *pgxpool.Pool, owner syntax.DID, parentID string, tokens ...string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,
			transitioned_at,created_at,updated_at
		) VALUES($1,'active',1,1,'test',now(),now(),now())
	`, owner); err != nil {
		t.Fatal(err)
	}
	seedAdditionalActiveParent(t, pool, owner, parentID, tokens...)
}

func seedAdditionalActiveParent(t *testing.T, pool *pgxpool.Pool, owner syntax.DID, parentID string, tokens ...string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO oauth_sessions(
			account_did,session_id,data,lifecycle_state,owner_generation,auth_epoch,
			absolute_expires_at,created_at,updated_at
		) VALUES($1,$2,'{}','active',1,1,now()+interval '10 days',now(),now())
	`, owner, parentID); err != nil {
		t.Fatal(err)
	}
	for _, token := range tokens {
		hash := sha256.Sum256([]byte(token))
		if _, err := pool.Exec(context.Background(), `
			INSERT INTO craftsky_sessions(
				token_hash,account_did,oauth_session_id,lifecycle_state,auth_epoch,
				last_seen_at,idle_expires_at
			) VALUES($1,$2,$3,'active',1,now(),now()+interval '1 day')
		`, hash[:], owner, parentID); err != nil {
			t.Fatal(err)
		}
	}
}
