package instagram

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/testdb"
)

type recordingExpiredModerationRestoration struct {
	calls int
	limit int
}

type recordingExpiredModerationReceipts struct {
	calls int
	limit int
}

func (r *recordingExpiredModerationReceipts) SweepExpiredIdempotencyReceipts(
	_ context.Context,
	limit int,
) (int, error) {
	r.calls++
	r.limit = limit
	return 0, nil
}

func (r *recordingExpiredModerationRestoration) EnqueueExpiredModerationRestorations(
	_ context.Context,
	limit int,
) (int, error) {
	r.calls++
	r.limit = limit
	return 0, nil
}

func TestRetentionServiceEnqueuesExpiredModerationRestoration(t *testing.T) {
	pool, now := newRetentionTest(t)
	restoration := &recordingExpiredModerationRestoration{}
	receipts := &recordingExpiredModerationReceipts{}
	service := NewRetentionService(
		pool,
		func() time.Time { return now },
		RetentionServiceOptions{
			Restoration:        restoration,
			ModerationReceipts: receipts,
		},
	)
	if _, err := service.Run(context.Background(), 499); err != nil {
		t.Fatal(err)
	}
	if restoration.calls != 1 || restoration.limit != 499 {
		t.Fatalf(
			"restoration calls=%d limit=%d, want 1/499",
			restoration.calls,
			restoration.limit,
		)
	}
	if receipts.calls != 1 || receipts.limit != 499 {
		t.Fatalf(
			"receipt sweep calls=%d limit=%d, want 1/499",
			receipts.calls,
			receipts.limit,
		)
	}
}

func TestRetentionServiceArchivesModerationOutboxAndBoundsDIDFreeHistory(t *testing.T) {
	pool, now := newRetentionTest(t)
	ctx := context.Background()
	cutoff := now.Add(-30 * 24 * time.Hour)
	historyCutoff := now.Add(-365 * 24 * time.Hour)

	if _, err := pool.Exec(ctx, `
		INSERT INTO moderation_outputs(
			id,source_did,subject_type,subject_did,value,action,created_at,indexed_at
		) VALUES
			('retention-no-work','did:plc:moderator','account','did:plc:no-work','hide','negate',$1,$1),
			('retention-cancelled','did:plc:moderator','account','did:plc:cancelled','hide','negate',$1,$1),
			('retention-completed','did:plc:moderator','account','did:plc:completed','hide','negate',$1,$1),
			('retention-future','did:plc:moderator','account','did:plc:future','hide','negate',$1,$1),
			('retention-pending','did:plc:moderator','account','did:plc:pending','hide','negate',$1,$1),
			('retention-processing','did:plc:moderator','account','did:plc:processing','hide','negate',$1,$1)
	`, cutoff.Add(-24*time.Hour)); err != nil {
		t.Fatalf("seed moderation outputs: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_reconciliation_jobs(
			id,owner_did,target_did,reason,status,next_attempt_at,
			lease_token,lease_expires_at,terminal_at,created_at,updated_at
		) VALUES
			('50000000-0000-4000-8000-000000000001','did:plc:completed',NULL,
			 'moderationCleared:retention-completed','completed',$1,NULL,NULL,$1,$2,$1),
			('50000000-0000-4000-8000-000000000002','did:plc:processing',NULL,
			 'moderationCleared:retention-processing','processing',$1,
			 '50000000-0000-4000-8000-000000000003',$3,NULL,$2,$1)
	`, now, cutoff.Add(-24*time.Hour), now.Add(time.Hour)); err != nil {
		t.Fatalf("seed moderation reconciliation jobs: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO moderation_restoration_outbox(
			moderation_output_id,target_did,status,reconciliation_job_id,
			created_at,processed_at
		) VALUES
			('retention-no-work','did:plc:no-work','no_work',NULL,$1,$2),
			('retention-cancelled','did:plc:cancelled','cancelled_target_terminal',NULL,$1,$2),
			('retention-completed','did:plc:completed','queued',
			 '50000000-0000-4000-8000-000000000001',$1,$2),
			('retention-future','did:plc:future','no_work',NULL,$1,$2+interval '1 microsecond'),
			('retention-pending','did:plc:pending','pending',NULL,$1,NULL),
			('retention-processing','did:plc:processing','queued',
			 '50000000-0000-4000-8000-000000000002',$1,$2)
	`, cutoff.Add(-24*time.Hour), cutoff); err != nil {
		t.Fatalf("seed moderation restoration outbox: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO moderation_idempotency_receipts(
			request_key_hash,request_fingerprint,output_id,output_status,
			created_at,expires_at
		) VALUES (
			decode(repeat('11',32),'hex'),decode(repeat('22',32),'hex'),
			'retention-no-work','indexed',$1::timestamptz,$1::timestamptz+interval '24 hours'
		)
	`, cutoff.Add(-24*time.Hour)); err != nil {
		t.Fatalf("seed moderation receipt: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO moderation_restoration_history(
			moderation_output_id,outcome,processed_at,archived_at
		) VALUES
			('expired-history','no_work',$1::timestamptz-interval '1 day',$1::timestamptz),
			('future-history','no_work',$1::timestamptz-interval '1 day',$1::timestamptz+interval '1 microsecond')
	`, historyCutoff); err != nil {
		t.Fatalf("seed moderation restoration history: %v", err)
	}

	service := NewRetentionService(pool, func() time.Time { return now }, RetentionServiceOptions{})
	stats, err := service.Run(ctx, 500)
	if err != nil {
		t.Fatalf("run moderation retention: %v", err)
	}
	if stats.ModerationOutboxArchived != 3 || stats.ModerationHistoryPurged != 1 {
		t.Fatalf("moderation retention stats = %+v", stats)
	}

	var archivedLive, archivedParents, receipts int
	if err := pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM moderation_restoration_outbox
			 WHERE moderation_output_id=ANY($1::text[])),
			(SELECT count(*) FROM moderation_outputs
			 WHERE id=ANY($1::text[])),
			(SELECT count(*) FROM moderation_idempotency_receipts
			 WHERE output_id='retention-no-work')
	`, []string{"retention-no-work", "retention-cancelled", "retention-completed"}).Scan(
		&archivedLive,
		&archivedParents,
		&receipts,
	); err != nil {
		t.Fatalf("read archived moderation rows: %v", err)
	}
	if archivedLive != 0 || archivedParents != 0 || receipts != 1 {
		t.Fatalf(
			"archived live=%d parents=%d receipts=%d, want 0/0/1",
			archivedLive,
			archivedParents,
			receipts,
		)
	}

	var keptLive, keptParents, queuedHistory, expiredHistory, futureHistory int
	if err := pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM moderation_restoration_outbox
			 WHERE moderation_output_id=ANY($1::text[])),
			(SELECT count(*) FROM moderation_outputs
			 WHERE id=ANY($1::text[])),
			(SELECT count(*) FROM moderation_restoration_history
			 WHERE moderation_output_id=ANY($2::text[])),
			(SELECT count(*) FROM moderation_restoration_history
			 WHERE moderation_output_id='expired-history'),
			(SELECT count(*) FROM moderation_restoration_history
			 WHERE moderation_output_id='future-history')
	`, []string{"retention-future", "retention-pending", "retention-processing"},
		[]string{"retention-no-work", "retention-cancelled", "retention-completed"}).Scan(
		&keptLive,
		&keptParents,
		&queuedHistory,
		&expiredHistory,
		&futureHistory,
	); err != nil {
		t.Fatalf("read retained moderation rows: %v", err)
	}
	if keptLive != 3 || keptParents != 3 || queuedHistory != 3 || expiredHistory != 0 || futureHistory != 1 {
		t.Fatalf(
			"kept live=%d parents=%d history=%d expired=%d future=%d, want 3/3/3/0/1",
			keptLive,
			keptParents,
			queuedHistory,
			expiredHistory,
			futureHistory,
		)
	}
}

func TestRetentionServiceExpiresAndPurgesAtExactBoundaries(t *testing.T) {
	pool, now := newRetentionTest(t)
	ctx := context.Background()

	digest := func(value byte) []byte { return bytes.Repeat([]byte{value}, 32) }
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_verification_attempts(
			id,owner_did,state,challenge_digest_version,challenge_digest,
			expires_at,terminal_at,created_at,updated_at
		) VALUES
			('01000000-0000-0000-0000-000000000001','did:plc:retention-a','pendingDm',1,$1,$4,NULL,$3,$3),
			('01000000-0000-0000-0000-000000000002','did:plc:retention-b','pendingDm',1,$2,$5,NULL,$3,$3),
			('01000000-0000-0000-0000-000000000003','did:plc:retention-c','cancelled',NULL,NULL,$3,$3-interval '30 days', $3-interval '31 days',$3-interval '30 days'),
			('01000000-0000-0000-0000-000000000004','did:plc:retention-d','cancelled',NULL,NULL,$3,$3-interval '30 days'+interval '1 microsecond', $3-interval '31 days',$3-interval '30 days')
	`, digest(1), digest(2), now, now, now.Add(time.Microsecond)); err != nil {
		t.Fatalf("seed attempts: %v", err)
	}

	service := NewRetentionService(pool, func() time.Time { return now }, RetentionServiceOptions{})
	stats, err := service.Run(ctx, 500)
	if err != nil {
		t.Fatalf("run retention: %v", err)
	}
	if stats.AttemptsTerminalized != 1 || stats.AttemptsPurged != 1 {
		t.Fatalf("attempt stats = %+v", stats)
	}

	var expiredState VerificationAttemptState
	var sensitive int
	if err := pool.QueryRow(ctx, `
		SELECT state,num_nonnulls(challenge_digest,candidate_igsid,candidate_username)
		FROM instagram_verification_attempts WHERE id='01000000-0000-0000-0000-000000000001'
	`).Scan(&expiredState, &sensitive); err != nil {
		t.Fatalf("read expired attempt: %v", err)
	}
	if expiredState != AttemptExpired || sensitive != 0 {
		t.Fatalf("expired attempt state=%s sensitive=%d", expiredState, sensitive)
	}
	assertRetentionExists(t, pool, "instagram_verification_attempts", "01000000-0000-0000-0000-000000000003", false)
	assertRetentionExists(t, pool, "instagram_verification_attempts", "01000000-0000-0000-0000-000000000004", true)

	var futureState VerificationAttemptState
	if err := pool.QueryRow(ctx, `SELECT state FROM instagram_verification_attempts WHERE id='01000000-0000-0000-0000-000000000002'`).Scan(&futureState); err != nil {
		t.Fatalf("read future attempt: %v", err)
	}
	if futureState != AttemptPendingDM {
		t.Fatalf("future attempt state=%s", futureState)
	}
}

func TestRetentionServiceClearsWebhookAndLinkIdentityThenPurgesTombstones(t *testing.T) {
	pool, now := newRetentionTest(t)
	ctx := context.Background()
	digest := bytes.Repeat([]byte{0x33}, 32)

	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_webhook_work(
			id,message_digest_version,message_digest,sender_igsid,official_account_id,
			event_at,status,next_attempt_at,terminal_at,terminal_reason,created_at,updated_at
		) VALUES
			('02000000-0000-0000-0000-000000000001',1,$1,NULL,'synthetic-official',$2::timestamptz,'completed',$2::timestamptz,$2::timestamptz-interval '7 days','processed',$2::timestamptz-interval '8 days',$2::timestamptz-interval '7 days'),
			('02000000-0000-0000-0000-000000000002',1,$3,NULL,'synthetic-official',$2::timestamptz,'completed',$2::timestamptz,$2::timestamptz-interval '7 days'+interval '1 microsecond','processed',$2::timestamptz-interval '8 days',$2::timestamptz-interval '7 days')
	`, digest, now, bytes.Repeat([]byte{0x34}, 32)); err != nil {
		t.Fatalf("seed webhook retention: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_account_links(
			id,owner_did,state,igsid,igsid_digest_version,igsid_digest,
			username,username_normalized,discoverable,verified_at,
			membership_inactive_at,created_at,updated_at
		) VALUES
			('03000000-0000-0000-0000-000000000001','did:plc:retention-link-a','membershipInactive','synthetic-igsid-a',1,$2,'synthetic.a','synthetic.a',false,$1::timestamptz-interval '2 years',$1::timestamptz-interval '1 year',$1::timestamptz-interval '2 years',$1::timestamptz-interval '1 year'),
			('03000000-0000-0000-0000-000000000002','did:plc:retention-link-b','membershipInactive','synthetic-igsid-b',1,$3,'synthetic.b','synthetic.b',false,$1::timestamptz-interval '2 years',$1::timestamptz-interval '1 year'+interval '1 microsecond',$1::timestamptz-interval '2 years',$1::timestamptz-interval '1 year')
	`, now, bytes.Repeat([]byte{0x35}, 32), bytes.Repeat([]byte{0x36}, 32)); err != nil {
		t.Fatalf("seed link retention: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_identity_claims(
			id,link_id,owner_did,state,igsid_digest_version,igsid_digest,claimed_at,created_at,updated_at
		) VALUES
			('03100000-0000-0000-0000-000000000001','03000000-0000-0000-0000-000000000001','did:plc:retention-link-a','active',1,$2,$1::timestamptz-interval '2 years',$1::timestamptz-interval '2 years',$1::timestamptz-interval '1 year'),
			('03100000-0000-0000-0000-000000000002','03000000-0000-0000-0000-000000000002','did:plc:retention-link-b','active',1,$3,$1::timestamptz-interval '2 years',$1::timestamptz-interval '2 years',$1::timestamptz-interval '1 year')
	`, now, bytes.Repeat([]byte{0x35}, 32), bytes.Repeat([]byte{0x36}, 32)); err != nil {
		t.Fatalf("seed claim retention: %v", err)
	}

	service := NewRetentionService(pool, func() time.Time { return now }, RetentionServiceOptions{})
	stats, err := service.Run(ctx, 500)
	if err != nil {
		t.Fatalf("run retention: %v", err)
	}
	if stats.WebhookPurged != 1 || stats.LinksMembershipExpired != 1 {
		t.Fatalf("retention stats = %+v", stats)
	}
	assertRetentionExists(t, pool, "instagram_webhook_work", "02000000-0000-0000-0000-000000000001", false)
	assertRetentionExists(t, pool, "instagram_webhook_work", "02000000-0000-0000-0000-000000000002", true)

	var state InstagramLinkState
	var identityFields int
	var revokedAt time.Time
	if err := pool.QueryRow(ctx, `
		SELECT state,num_nonnulls(igsid,username,username_normalized),revoked_at
		FROM instagram_account_links WHERE id='03000000-0000-0000-0000-000000000001'
	`).Scan(&state, &identityFields, &revokedAt); err != nil {
		t.Fatalf("read membership-expired link: %v", err)
	}
	if state != LinkRevoked || identityFields != 0 || !revokedAt.Equal(now) {
		t.Fatalf("membership-expired link state=%s fields=%d revoked=%v", state, identityFields, revokedAt)
	}
	var futureLinkState InstagramLinkState
	if err := pool.QueryRow(ctx, `SELECT state FROM instagram_account_links WHERE id='03000000-0000-0000-0000-000000000002'`).Scan(&futureLinkState); err != nil {
		t.Fatalf("read future inactive link: %v", err)
	}
	if futureLinkState != LinkMembershipInactive {
		t.Fatalf("future inactive link state=%s", futureLinkState)
	}

	secondNow := now.Add(90 * 24 * time.Hour)
	service = NewRetentionService(pool, func() time.Time { return secondNow }, RetentionServiceOptions{})
	if _, err := service.Run(ctx, 500); err != nil {
		t.Fatalf("purge link tombstone: %v", err)
	}
	assertRetentionExists(t, pool, "instagram_account_links", "03000000-0000-0000-0000-000000000001", false)
}

func TestRetentionServiceKeepsVerifiedAccountImportsUntilExplicitUnlink(t *testing.T) {
	pool, now := newRetentionTest(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_graph_imports(
			id,owner_did,state,source_type,following_count,created_at,updated_at
		) VALUES(
			'10000000-0000-0000-0000-000000000001',
			'did:plc:retention-import-owner',
			'active','manual',1,$1::timestamptz-interval '10 years',$1
		)
	`, now); err != nil {
		t.Fatalf("seed retained import: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_graph_handles(
			import_id,username_normalized,matched,created_at
		) VALUES(
			'10000000-0000-0000-0000-000000000001',
			'synthetic.retained.handle',false,$1::timestamptz-interval '10 years'
		)
	`, now); err != nil {
		t.Fatalf("seed retained handle: %v", err)
	}

	service := NewRetentionService(pool, func() time.Time { return now }, RetentionServiceOptions{})
	if _, err := service.Run(ctx, 500); err != nil {
		t.Fatalf("run retention: %v", err)
	}
	var imports, handles int
	if err := pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM instagram_graph_imports
			  WHERE id='10000000-0000-0000-0000-000000000001'),
			(SELECT count(*) FROM instagram_graph_handles
			  WHERE import_id='10000000-0000-0000-0000-000000000001')
	`).Scan(&imports, &handles); err != nil {
		t.Fatalf("count retained import: %v", err)
	}
	if imports != 1 || handles != 1 {
		t.Fatalf("retained imports=%d handles=%d, want 1 each", imports, handles)
	}
}

func TestRetentionServicePurgesTerminalPrivateClassesAtExactBoundaries(t *testing.T) {
	pool, now := newRetentionTest(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_account_links(
			id,owner_did,state,igsid,igsid_digest_version,igsid_digest,
			username,username_normalized,discoverable,conflict_pending,
			verified_at,created_at,updated_at
		) VALUES(
			'40000000-0000-0000-0000-000000000001','did:plc:retention-target',
			'active','retention-target',1,$2,
			'retention.target','retention.target',true,false,$1,$1,$1
		)
	`, now, bytes.Repeat([]byte{0x40}, 32)); err != nil {
		t.Fatalf("seed suggestion retention link: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_graph_imports(
			id,owner_did,state,source_type,following_count,created_at,updated_at
		) VALUES(
			'40000000-0000-0000-0000-000000000002','did:plc:retention-s3',
			'active','manual',1,$1,$1
		)
	`, now); err != nil {
		t.Fatalf("seed suggestion retention import: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_private_suggestions(
			id,importer_did,target_did,importer_generation,target_generation,
			evidence_link_id,state,reason,terminal_at,created_at,updated_at
		) VALUES
			('41000000-0000-0000-0000-000000000001','did:plc:retention-s1','did:plc:target-s1',1,1,
			 '40000000-0000-0000-0000-000000000001','invalidated','verifiedInstagramFollow',
			 $1::timestamptz-interval '90 days',$1::timestamptz-interval '100 days',$1::timestamptz-interval '90 days'),
			('41000000-0000-0000-0000-000000000002','did:plc:retention-s2','did:plc:target-s2',1,1,
			 '40000000-0000-0000-0000-000000000001','invalidated','verifiedInstagramFollow',
			 $1::timestamptz-interval '90 days'+interval '1 microsecond',$1::timestamptz-interval '100 days',$1::timestamptz-interval '90 days'),
			('41000000-0000-0000-0000-000000000003','did:plc:retention-s3','did:plc:target-s3',1,1,
			 '40000000-0000-0000-0000-000000000001','dismissed','verifiedInstagramFollow',
			 $1::timestamptz-interval '1 year',$1::timestamptz-interval '13 months',$1::timestamptz-interval '1 year')
	`, now); err != nil {
		t.Fatalf("seed suggestion retention rows: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_private_suggestion_sources(suggestion_id,import_id,created_at)
		VALUES(
			'41000000-0000-0000-0000-000000000003',
			'40000000-0000-0000-0000-000000000002',$1
		)
	`, now); err != nil {
		t.Fatalf("seed suggestion retention source: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_link_conflicts(
			id,state,opened_at,expires_at,created_at,updated_at,igsid_digest_version,igsid_digest
		) VALUES
			('42000000-0000-0000-0000-000000000001','open',$1::timestamptz-interval '365 days',$1::timestamptz,$1::timestamptz-interval '365 days',$1::timestamptz-interval '365 days',1,$2),
			('42000000-0000-0000-0000-000000000002','open',$1::timestamptz-interval '365 days',$1::timestamptz+interval '1 microsecond',$1::timestamptz-interval '365 days',$1::timestamptz-interval '365 days',1,$3),
			('42000000-0000-0000-0000-000000000003','resolvedKeepExisting',$1::timestamptz-interval '2 years',$1::timestamptz-interval '2 years',$1::timestamptz-interval '2 years',$1::timestamptz-interval '365 days',NULL,NULL)
	`, now, bytes.Repeat([]byte{0x51}, 32), bytes.Repeat([]byte{0x52}, 32)); err != nil {
		t.Fatalf("seed conflict retention: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE instagram_link_conflicts SET resolved_at=$1::timestamptz-interval '365 days' WHERE id='42000000-0000-0000-0000-000000000003'`, now); err != nil {
		t.Fatalf("seed resolved conflict time: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_rate_limit_buckets(
			bucket_scope,key_version,key_digest,window_start,window_end,count,created_at,updated_at
		) VALUES('challenge_did',1,$2,$1::timestamptz-interval '25 hours',$1::timestamptz-interval '24 hours',1,$1::timestamptz-interval '25 hours',$1::timestamptz-interval '25 hours')
	`, now, bytes.Repeat([]byte{0x53}, 32)); err != nil {
		t.Fatalf("seed rate retention: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_audit_events(owner_did,action,subject_kind,subject_id,outcome,created_at)
		VALUES('did:plc:retention-audit','syntheticOld','link','opaque-old','completed',$1::timestamptz-interval '365 days')
	`, now); err != nil {
		t.Fatalf("seed audit retention: %v", err)
	}

	service := NewRetentionService(pool, func() time.Time { return now }, RetentionServiceOptions{})
	stats, err := service.Run(ctx, 500)
	if err != nil {
		t.Fatalf("run terminal retention: %v", err)
	}
	if stats.SuggestionsPurged != 1 || stats.ConflictsExpired != 1 || stats.ConflictsPurged != 1 || stats.RateBucketsPurged != 1 || stats.AuditsPurged != 1 {
		t.Fatalf("terminal stats = %+v", stats)
	}
	assertRetentionExists(t, pool, "instagram_private_suggestions", "41000000-0000-0000-0000-000000000001", false)
	assertRetentionExists(t, pool, "instagram_private_suggestions", "41000000-0000-0000-0000-000000000002", true)
	assertRetentionExists(t, pool, "instagram_private_suggestions", "41000000-0000-0000-0000-000000000003", true)
	assertRetentionExists(t, pool, "instagram_link_conflicts", "42000000-0000-0000-0000-000000000003", false)
	var conflictState InstagramConflictState
	var identityFields int
	if err := pool.QueryRow(ctx, `
		SELECT state,num_nonnulls(existing_link_id,claimant_attempt_id,claimant_link_id,igsid_digest,resolution_note_digest)
		FROM instagram_link_conflicts WHERE id='42000000-0000-0000-0000-000000000001'
	`).Scan(&conflictState, &identityFields); err != nil {
		t.Fatalf("read expired conflict: %v", err)
	}
	if conflictState != ConflictExpired || identityFields != 0 {
		t.Fatalf("expired conflict state=%s identityFields=%d", conflictState, identityFields)
	}
}

func newRetentionTest(t *testing.T) (*pgxpool.Pool, time.Time) {
	t.Helper()
	var ddl strings.Builder
	for _, path := range []string{
		"../../migrations/000014_moderation_flow.up.sql",
		"../../migrations/000021_appview_notifications.up.sql",
		"../../migrations/000022_notification_newness.up.sql",
		"../../migrations/000025_instagram_migration.up.sql",
		"../../migrations/000026_system_notifications.up.sql",
		"../../migrations/000029_notification_client_owned_destination.up.sql",
		"../../migrations/000030_instagram_automatic_follows.up.sql",
		"../../migrations/000031_instagram_automatic_follow_storage_names.up.sql",
		"../../migrations/000042_instagram_private_suggestions.up.sql",
		"../../migrations/000044_moderation_restoration_outbox.up.sql",
	} {
		migration, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read migration %s: %v", path, err)
		}
		ddl.Write(migration)
	}
	return testdb.WithSchema(t, ddl.String()), time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
}

func assertRetentionExists(t *testing.T, pool *pgxpool.Pool, table, id string, want bool) {
	t.Helper()
	var exists bool
	query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE id=$1)", table)
	if err := pool.QueryRow(context.Background(), query, id).Scan(&exists); err != nil {
		t.Fatalf("read %s %s: %v", table, id, err)
	}
	if exists != want {
		t.Fatalf("%s %s exists=%t want=%t", table, id, exists, want)
	}
}
