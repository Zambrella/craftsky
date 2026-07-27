package instagram

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestOperatorServiceListsOnlyOpaqueOpenConflictsInStableBoundedPages(t *testing.T) {
	pool, now := newRetentionTest(t)
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		id := uuid.MustParse(fmt.Sprintf("51000000-0000-0000-0000-%012d", i+1))
		if _, err := pool.Exec(ctx, `
			INSERT INTO instagram_link_conflicts(id,state,opened_at,expires_at,created_at,updated_at)
			VALUES($1,'open',$2,$3,$2,$2)
		`, id, now.Add(time.Duration(i)*time.Second), now.AddDate(1, 0, 0)); err != nil {
			t.Fatalf("seed conflict %d: %v", i, err)
		}
	}
	service := newOperatorTestService(t, pool, now)
	page, next, err := service.ListOpenConflicts(ctx, 2, uuid.Nil)
	if err != nil {
		t.Fatalf("list first conflict page: %v", err)
	}
	if len(page) != 2 || next == uuid.Nil || page[0].ID.String() >= page[1].ID.String() {
		t.Fatalf("first conflict page=%+v next=%s", page, next)
	}
	for _, item := range page {
		diagnostic := fmt.Sprintf("%v %+v %#v", item, item, item)
		for _, forbidden := range []string{"did:plc", "username", "igsid", "digest", "claimant", "existing"} {
			if strings.Contains(strings.ToLower(diagnostic), forbidden) {
				t.Fatalf("operator conflict output leaked %q: %s", forbidden, diagnostic)
			}
		}
	}
	second, final, err := service.ListOpenConflicts(ctx, 2, next)
	if err != nil || len(second) != 1 || final != uuid.Nil {
		t.Fatalf("second conflict page=%+v next=%s err=%v", second, final, err)
	}
	if _, _, err := service.ListOpenConflicts(ctx, MaxOperatorBatch+1, uuid.Nil); !errors.Is(err, ErrInvalidOperatorBatch) {
		t.Fatalf("limit 501 error=%v", err)
	}
}

func TestOperatorServiceResolvesConflictExplicitlyWithoutTransferringOwnership(t *testing.T) {
	for _, resolution := range []OperatorConflictResolution{ResolutionKeepExisting, ResolutionRevokeExisting} {
		t.Run(string(resolution), func(t *testing.T) {
			pool, now := newRetentionTest(t)
			ctx := context.Background()
			linkID := uuid.MustParse("52000000-0000-0000-0000-000000000001")
			attemptID := uuid.MustParse("52000000-0000-0000-0000-000000000002")
			conflictID := uuid.MustParse("52000000-0000-0000-0000-000000000003")
			digest := bytes.Repeat([]byte{0x62}, 32)
			if _, err := pool.Exec(ctx, `
				INSERT INTO instagram_account_links(
					id,owner_did,state,igsid,igsid_digest_version,igsid_digest,
					username,username_normalized,discoverable,conflict_pending,
					verified_at,created_at,updated_at
				) VALUES($1,'did:plc:operator-existing','active','synthetic-operator-igsid',1,$2,
					'synthetic.operator','synthetic.operator',false,true,$3,$3,$3)
			`, linkID, digest, now); err != nil {
				t.Fatalf("seed existing link: %v", err)
			}
			if _, err := pool.Exec(ctx, `
				INSERT INTO instagram_identity_claims(
					id,link_id,owner_did,state,igsid_digest_version,igsid_digest,claimed_at,created_at,updated_at
				) VALUES('52000000-0000-0000-0000-000000000004',$1,'did:plc:operator-existing','active',1,$2,$3,$3,$3)
			`, linkID, digest, now); err != nil {
				t.Fatalf("seed existing claim: %v", err)
			}
			if _, err := pool.Exec(ctx, `
				INSERT INTO instagram_verification_attempts(
					id,owner_did,state,expires_at,terminal_at,created_at,updated_at
				) VALUES($1,'did:plc:operator-claimant','conflicted',$2,$2,$2,$2)
			`, attemptID, now); err != nil {
				t.Fatalf("seed claimant attempt: %v", err)
			}
			if _, err := pool.Exec(ctx, `
				INSERT INTO instagram_link_conflicts(
					id,state,existing_link_id,claimant_attempt_id,igsid_digest_version,
					igsid_digest,opened_at,expires_at,created_at,updated_at
				) VALUES($1,'open',$2,$3,1,$4,$5,$6,$5,$5)
			`, conflictID, linkID, attemptID, digest, now, now.AddDate(1, 0, 0)); err != nil {
				t.Fatalf("seed conflict: %v", err)
			}

			service := newOperatorTestService(t, pool, now)
			result, err := service.ResolveConflict(ctx, conflictID, resolution)
			if err != nil || !result.Changed || result.ID != conflictID || result.State != resolution.State() {
				t.Fatalf("resolve conflict result=%+v err=%v", result, err)
			}
			replay, err := service.ResolveConflict(ctx, conflictID, resolution)
			if err != nil || replay.Changed {
				t.Fatalf("replay conflict result=%+v err=%v", replay, err)
			}

			var state InstagramConflictState
			var evidenceFields int
			if err := pool.QueryRow(ctx, `
				SELECT state,num_nonnulls(existing_link_id,claimant_attempt_id,claimant_link_id,igsid_digest)
				FROM instagram_link_conflicts WHERE id=$1
			`, conflictID).Scan(&state, &evidenceFields); err != nil {
				t.Fatalf("read resolved conflict: %v", err)
			}
			if state != resolution.State() || evidenceFields != 0 {
				t.Fatalf("resolved conflict state=%s evidence=%d", state, evidenceFields)
			}
			var claimantLinks int
			if err := pool.QueryRow(ctx, `SELECT count(*) FROM instagram_account_links WHERE owner_did='did:plc:operator-claimant'`).Scan(&claimantLinks); err != nil {
				t.Fatalf("count claimant links: %v", err)
			}
			if claimantLinks != 0 {
				t.Fatalf("resolution transferred ownership: claimant links=%d", claimantLinks)
			}
			var linkState InstagramLinkState
			var identityFields int
			if err := pool.QueryRow(ctx, `
				SELECT state,num_nonnulls(igsid,username,username_normalized)
				FROM instagram_account_links WHERE id=$1
			`, linkID).Scan(&linkState, &identityFields); err != nil {
				t.Fatalf("read existing link: %v", err)
			}
			if resolution == ResolutionKeepExisting && (linkState != LinkActive || identityFields != 3) {
				t.Fatalf("keep-existing link state=%s identity=%d", linkState, identityFields)
			}
			if resolution == ResolutionRevokeExisting && (linkState != LinkRevoked || identityFields != 0) {
				t.Fatalf("revoke-existing link state=%s identity=%d", linkState, identityFields)
			}
			var audits int
			if err := pool.QueryRow(ctx, `SELECT count(*) FROM instagram_audit_events WHERE subject_id=$1`, conflictID.String()).Scan(&audits); err != nil {
				t.Fatalf("count resolution audits: %v", err)
			}
			if audits != 1 {
				t.Fatalf("resolution audits=%d want=1", audits)
			}
		})
	}
}

func TestOperatorServiceRevokesLinksAndRetriesOnlyRecoverableJobsIdempotently(t *testing.T) {
	pool, now := newRetentionTest(t)
	ctx := context.Background()
	linkID := uuid.MustParse("53000000-0000-0000-0000-000000000001")
	digest := bytes.Repeat([]byte{0x73}, 32)
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_account_links(
			id,owner_did,state,igsid,igsid_digest_version,igsid_digest,
			username,username_normalized,discoverable,verified_at,created_at,updated_at
		) VALUES($1,'did:plc:operator-revoke','active','synthetic-revoke-igsid',1,$2,
			'synthetic.revoke','synthetic.revoke',true,$3,$3,$3)
	`, linkID, digest, now); err != nil {
		t.Fatalf("seed revocable link: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_identity_claims(
			id,link_id,owner_did,state,igsid_digest_version,igsid_digest,claimed_at,created_at,updated_at
		) VALUES('53000000-0000-0000-0000-000000000002',$1,'did:plc:operator-revoke','active',1,$2,$3,$3,$3)
	`, linkID, digest, now); err != nil {
		t.Fatalf("seed revocable claim: %v", err)
	}
	reconciliationID := uuid.MustParse("53000000-0000-0000-0000-000000000003")
	webhookID := uuid.MustParse("53000000-0000-0000-0000-000000000004")
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_reconciliation_jobs(
			id,owner_did,reason,status,attempts,next_attempt_at,terminal_at,created_at,updated_at
		) VALUES($1,'did:plc:operator-job','syntheticOperatorRetry','failed',5,$2,$2,$2,$2)
	`, reconciliationID, now); err != nil {
		t.Fatalf("seed reconciliation job: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_webhook_work(
			id,message_digest_version,message_digest,event_at,status,attempts,next_attempt_at,
			terminal_at,terminal_reason,created_at,updated_at
		) VALUES($1,1,$2,$3,'failed',5,$3,$3,'maxAttempts',$3,$3)
	`, webhookID, bytes.Repeat([]byte{0x74}, 32), now); err != nil {
		t.Fatalf("seed webhook job: %v", err)
	}

	service := newOperatorTestService(t, pool, now)
	revoked, err := service.RevokeLink(ctx, linkID)
	if err != nil || !revoked.Changed {
		t.Fatalf("revoke link result=%+v err=%v", revoked, err)
	}
	replay, err := service.RevokeLink(ctx, linkID)
	if err != nil || replay.Changed {
		t.Fatalf("replay link revoke result=%+v err=%v", replay, err)
	}
	var identityFields int
	if err := pool.QueryRow(ctx, `SELECT num_nonnulls(igsid,username,username_normalized) FROM instagram_account_links WHERE id=$1`, linkID).Scan(&identityFields); err != nil {
		t.Fatalf("read revoked link: %v", err)
	}
	if identityFields != 0 {
		t.Fatalf("revoked link retained %d identity fields", identityFields)
	}

	job, err := service.InspectJob(ctx, OperatorJobReconciliation, reconciliationID)
	if err != nil || job.ID != reconciliationID || job.Status != "failed" {
		t.Fatalf("inspect job=%+v err=%v", job, err)
	}
	if diagnostic := fmt.Sprintf("%v %+v %#v", job, job, job); strings.Contains(diagnostic, "did:plc") || strings.Contains(diagnostic, "syntheticOperatorRetry") {
		t.Fatalf("job diagnostic leaked private state: %s", diagnostic)
	}
	retried, err := service.RetryJob(ctx, OperatorJobReconciliation, reconciliationID)
	if err != nil || !retried.Changed {
		t.Fatalf("retry job result=%+v err=%v", retried, err)
	}
	retriedAgain, err := service.RetryJob(ctx, OperatorJobReconciliation, reconciliationID)
	if err != nil || retriedAgain.Changed {
		t.Fatalf("retry job replay=%+v err=%v", retriedAgain, err)
	}
	if _, err := service.RetryJob(ctx, OperatorJobWebhook, webhookID); !errors.Is(err, ErrOperatorJobNotRetryable) {
		t.Fatalf("terminal webhook retry error=%v", err)
	}
	var status string
	var attempts int
	if err := pool.QueryRow(ctx, `SELECT status,attempts FROM instagram_reconciliation_jobs WHERE id=$1`, reconciliationID).Scan(&status, &attempts); err != nil {
		t.Fatalf("read retried job: %v", err)
	}
	if status != "queued" || attempts != 0 {
		t.Fatalf("retried job status=%s attempts=%d", status, attempts)
	}
}

func newOperatorTestService(t *testing.T, pool *pgxpool.Pool, now time.Time) *OperatorService {
	t.Helper()
	service, err := NewOperatorService(pool, bytes.Repeat([]byte{0x7f}, 32), func() time.Time { return now })
	if err != nil {
		t.Fatalf("new operator service: %v", err)
	}
	return service
}
