package accountdeletion

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"social.craftsky/appview/internal/testdb"
)

func TestAcceptanceBindsFreshOAuthBeforeRevokingOrdinaryAccessAndFinalizesWithoutResidue(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, "")
	applyAllAccountDeletionTestMigrations(t, pool)
	ctx := context.Background()
	now := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:alice")
	jobID := uuid.MustParse("10000000-0000-4000-8000-000000000001")
	store := NewStore(pool, func() time.Time { return now })

	if _, err := pool.Exec(ctx, `
		INSERT INTO oauth_sessions(account_did,session_id,data)
		VALUES($1,'ordinary-oauth','{}'),($1,'deletion-oauth','{}');
		INSERT INTO craftsky_sessions(token_hash,account_did,oauth_session_id)
		VALUES(decode(repeat('01',32),'hex'),$1,'ordinary-oauth');
	`, owner); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateIntent(ctx, IntentRecord{
		JobID: jobID, Owner: owner,
		ConfirmationHandleHash: HashSecret("@alice.test"),
		ExpiresAt:              now.Add(10 * time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.CompleteReauthentication(ctx, jobID, owner, "deletion-oauth", HashSecret("proof")); err != nil {
		t.Fatal(err)
	}
	accepted, err := store.Accept(ctx, AcceptanceRequest{
		JobID: jobID, Owner: owner, ReauthProof: "proof", ConfirmationHandle: "@alice.test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if accepted.Status != StatusActive || accepted.DeletionOAuthSessionID != "deletion-oauth" {
		t.Fatalf("accepted operation = %+v", accepted)
	}
	if got, err := store.BoundOAuthSession(ctx, jobID, owner); err != nil || got != "deletion-oauth" {
		t.Fatalf("worker OAuth authority = (%q, %v)", got, err)
	}
	for _, scope := range []struct {
		jobID uuid.UUID
		owner syntax.DID
	}{
		{jobID: uuid.MustParse("10000000-0000-4000-8000-000000000009"), owner: owner},
		{jobID: jobID, owner: syntax.DID("did:plc:bob")},
	} {
		if _, err := store.BoundOAuthSession(ctx, scope.jobID, scope.owner); !errors.Is(err, ErrBoundOAuthUnauthorized) {
			t.Fatalf("cross-scope worker OAuth error = %v", err)
		}
	}
	// A concurrent duplicate that authenticated before revocation is harmless.
	if _, err := store.Accept(ctx, AcceptanceRequest{JobID: jobID, Owner: owner}); err != nil {
		t.Fatalf("duplicate acceptance: %v", err)
	}
	assertCount(t, pool, `SELECT count(*) FROM craftsky_sessions WHERE account_did=$1`, owner, 0)
	assertCount(t, pool, `SELECT count(*) FROM oauth_sessions WHERE account_did=$1`, owner, 1)

	claimed, found, err := store.ClaimDue(ctx, "worker", time.Minute)
	if err != nil || !found {
		t.Fatalf("claim = (%+v, %t, %v)", claimed, found, err)
	}
	if err := store.CompleteAttempt(ctx, claimed); err != nil {
		t.Fatal(err)
	}
	assertCount(t, pool, `SELECT count(*) FROM account_deletion_operations WHERE owner_did=$1`, owner, 0)
	assertCount(t, pool, `SELECT count(*) FROM oauth_sessions WHERE account_did=$1`, owner, 0)
}

func TestPendingLoginRefreshesWorkerOAuthWithoutMintingMembership(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, "")
	applyAllAccountDeletionTestMigrations(t, pool)
	ctx := context.Background()
	now := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:alice")
	store := NewStore(pool, func() time.Time { return now })
	if _, err := pool.Exec(ctx, `
		INSERT INTO oauth_sessions(account_did,session_id,data)
		VALUES($1,'old-deletion-oauth','{}'),($1,'fresh-login-oauth','{}');
		INSERT INTO account_deletion_operations(
			id,owner_did,state,accepted_at,deletion_oauth_session_id,next_attempt_at
		) VALUES('10000000-0000-4000-8000-000000000002',$1,'retrying',$2,'old-deletion-oauth',$2);
	`, owner, now); err != nil {
		t.Fatal(err)
	}
	refreshed, err := store.RefreshBoundOAuthFromLogin(ctx, owner, "fresh-login-oauth")
	if err != nil || !refreshed {
		t.Fatalf("refresh = (%t, %v)", refreshed, err)
	}
	if got, err := store.BoundOAuthSession(ctx, uuid.MustParse("10000000-0000-4000-8000-000000000002"), owner); err != nil || got != "fresh-login-oauth" {
		t.Fatalf("bound OAuth = (%q, %v)", got, err)
	}
	assertCount(t, pool, `SELECT count(*) FROM oauth_sessions WHERE account_did=$1`, owner, 1)
	assertCount(t, pool, `SELECT count(*) FROM craftsky_sessions WHERE account_did=$1`, owner, 0)
}

func assertCount(t *testing.T, pool *pgxpool.Pool, query string, value any, want int) {
	t.Helper()
	var got int
	if err := pool.QueryRow(context.Background(), query, value).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("count = %d, want %d for %s", got, want, query)
	}
}
