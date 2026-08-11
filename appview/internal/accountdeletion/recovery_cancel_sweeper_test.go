package accountdeletion

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/testdb"
)

func TestFormerBearerIsOneTimeStatusRecoveryOnly(t *testing.T) {
	pool := testdb.WithSchema(t, "")
	applyAllAccountDeletionTestMigrations(t, pool)
	ctx := context.Background()
	now := time.Date(2026, 8, 10, 15, 5, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:alice")
	jobID := uuid.MustParse("10000000-0000-4000-8000-000000000077")
	formerBearer := "former-alice-bearer"
	if _, err := pool.Exec(ctx, `INSERT INTO oauth_sessions(account_did,session_id,data) VALUES($1,'deletion-oauth','{}')`, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO account_deletion_operations(
			id,owner_did,state,phase,accepted_at,deletion_oauth_session_id,next_attempt_at
		) VALUES($2,$1,'active','queued',$3,'deletion-oauth',$3)
	`, owner, jobID, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO account_deletion_recovery_credentials(token_hash,job_id,owner_did,device_id)
		VALUES($3,$2,$1,'alice-phone')
	`, owner, jobID, HashSecret(formerBearer)); err != nil {
		t.Fatal(err)
	}
	store := NewStore(pool, func() time.Time { return now })
	signer, err := NewStatusCapabilitySigner(bytes.Repeat([]byte{5}, 32), func() time.Time { return now }, bytes.NewReader(bytes.Repeat([]byte{6}, 128)))
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewAppService(AppServiceOptions{
		Pool: pool, Store: store, Signer: signer,
		OAuth: &databaseOAuthStarter{pool: pool, state: "unused", requestURI: "urn:unused"},
		Now:   func() time.Time { return now }, Random: bytes.NewReader(bytes.Repeat([]byte{8}, 128)),
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := service.Recover(ctx, formerBearer, "alice-phone")
	if err != nil {
		t.Fatal(err)
	}
	if result.JobID != jobID.String() || result.StatusToken == "" || result.Status != StatusActive {
		t.Fatalf("recovery result = %+v", result)
	}
	if _, err := service.Recover(ctx, formerBearer, "alice-phone"); !errors.Is(err, ErrRecoveryUnauthorized) {
		t.Fatalf("second recovery error = %v, want ErrRecoveryUnauthorized", err)
	}
	if pending, err := auth.NewCraftskySessionStore(pool, time.Minute).PendingDeletion(ctx, formerBearer); err != nil || !pending {
		t.Fatalf("ordinary auth pending lookup = %v, %v", pending, err)
	}
}

func TestCancelIntentRemovesStatusAndDeletionOAuthWithoutAccepting(t *testing.T) {
	pool := testdb.WithSchema(t, "")
	applyAllAccountDeletionTestMigrations(t, pool)
	ctx := context.Background()
	now := time.Date(2026, 8, 10, 15, 5, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:alice")
	jobID := uuid.MustParse("10000000-0000-4000-8000-000000000078")
	statusToken := "intent-status"
	if _, err := pool.Exec(ctx, `INSERT INTO oauth_sessions(account_did,session_id,data) VALUES($1,'intent-oauth','{}')`, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO account_deletion_operations(
			id,owner_did,state,reauth_oauth_session_id,confirmation_handle_hash,intent_expires_at
		) VALUES($2,$1,'intent','intent-oauth',$3,$4)
	`, owner, jobID, HashSecret("@alice.test"), now.Add(10*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO account_deletion_status_credentials(token_hash,job_id,owner_did,device_id,expires_at)
		VALUES($3,$2,$1,'alice-phone',$4)
	`, owner, jobID, HashSecret(statusToken), now.Add(30*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	store := NewStore(pool, func() time.Time { return now })
	if err := store.CancelIntent(ctx, jobID, owner, statusToken); err != nil {
		t.Fatal(err)
	}
	var operationCount, oauthCount, statusCount int
	if err := pool.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM account_deletion_operations WHERE id=$1),
		(SELECT count(*) FROM oauth_sessions WHERE account_did=$2 AND session_id='intent-oauth'),
		(SELECT count(*) FROM account_deletion_status_credentials WHERE job_id=$1)
	`, jobID, owner).Scan(&operationCount, &oauthCount, &statusCount); err != nil {
		t.Fatal(err)
	}
	if operationCount != 0 || oauthCount != 0 || statusCount != 0 {
		t.Fatalf("remaining operation/oauth/status = %d/%d/%d", operationCount, oauthCount, statusCount)
	}
	if err := store.CancelIntent(ctx, jobID, owner, statusToken); err != nil {
		t.Fatalf("duplicate cancel must be idempotent: %v", err)
	}
}

func TestAuditSweeperUsesExactExpiryBoundary(t *testing.T) {
	pool := testdb.WithSchema(t, "")
	applyAllAccountDeletionTestMigrations(t, pool)
	ctx := context.Background()
	now := time.Date(2026, 8, 10, 15, 5, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `
		INSERT INTO account_deletion_audits(job_id,did,accepted_at,terminal_at,outcome,expires_at) VALUES
		('10000000-0000-4000-8000-000000000081','did:plc:expired',$1,$1,'deleted',$1),
		('10000000-0000-4000-8000-000000000082','did:plc:future',$1,$1,'deleted',$2)
	`, now, now.Add(time.Microsecond)); err != nil {
		t.Fatal(err)
	}
	sweeper := NewAuditSweeper(pool, func() time.Time { return now })
	metrics := &recordingDeletionMetrics{}
	sweeper.SetTelemetry(NewDeletionTelemetry(nil, metrics))
	deleted, err := sweeper.Sweep(ctx, 100)
	if err != nil || deleted != 1 {
		t.Fatalf("sweep = %d, %v", deleted, err)
	}
	if got := countDeletionEvent(deletionEventNames(metrics.events), "auditExpired"); got != 1 {
		t.Fatalf("production audit-expiry telemetry events=%d, want 1", got)
	}
}
