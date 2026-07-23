package instagram

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/testdb"
)

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

	service := NewRetentionService(pool, func() time.Time { return now })
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

	service := NewRetentionService(pool, func() time.Time { return now })
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
	service = NewRetentionService(pool, func() time.Time { return secondNow })
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

	service := NewRetentionService(pool, func() time.Time { return now })
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
		INSERT INTO instagram_follow_suggestions(
			id,importer_did,target_did,state,reason,terminal_at,created_at,updated_at
		) VALUES
			('41000000-0000-0000-0000-000000000001','did:plc:retention-s1','did:plc:target-s1','dismissed','verifiedInstagramFollow',$1::timestamptz-interval '90 days',$1::timestamptz-interval '100 days',$1::timestamptz-interval '90 days'),
			('41000000-0000-0000-0000-000000000002','did:plc:retention-s2','did:plc:target-s2','dismissed','verifiedInstagramFollow',$1::timestamptz-interval '90 days'+interval '1 microsecond',$1::timestamptz-interval '100 days',$1::timestamptz-interval '90 days'),
			('41000000-0000-0000-0000-000000000003','did:plc:retention-s3','did:plc:target-s3','accepted','verifiedInstagramFollow',$1::timestamptz-interval '1 year',$1::timestamptz-interval '13 months',$1::timestamptz-interval '1 year'),
			('41000000-0000-0000-0000-000000000004','did:plc:retention-s4','did:plc:target-s4','alreadyFollowing','verifiedInstagramFollow',$1::timestamptz-interval '1 year'+interval '1 microsecond',$1::timestamptz-interval '13 months',$1::timestamptz-interval '1 year')
	`, now); err != nil {
		t.Fatalf("seed suggestion retention: %v", err)
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

	service := NewRetentionService(pool, func() time.Time { return now })
	stats, err := service.Run(ctx, 500)
	if err != nil {
		t.Fatalf("run terminal retention: %v", err)
	}
	if stats.SuggestionsPurged != 2 || stats.ConflictsExpired != 1 || stats.ConflictsPurged != 1 || stats.RateBucketsPurged != 1 || stats.AuditsPurged != 1 {
		t.Fatalf("terminal stats = %+v", stats)
	}
	assertRetentionExists(t, pool, "instagram_follow_suggestions", "41000000-0000-0000-0000-000000000001", false)
	assertRetentionExists(t, pool, "instagram_follow_suggestions", "41000000-0000-0000-0000-000000000002", true)
	assertRetentionExists(t, pool, "instagram_follow_suggestions", "41000000-0000-0000-0000-000000000003", false)
	assertRetentionExists(t, pool, "instagram_follow_suggestions", "41000000-0000-0000-0000-000000000004", true)
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

func TestRetentionServicePurgesMatchEventsAndRetractedDeliveriesAtExactBoundaries(t *testing.T) {
	pool, now := newRetentionTest(t)
	ctx := context.Background()
	eventAtBoundary := uuid.MustParse("43000000-0000-0000-0000-000000000001")
	eventAfterBoundary := uuid.MustParse("43000000-0000-0000-0000-000000000002")
	retractedAtBoundary := uuid.MustParse("43000000-0000-0000-0000-000000000003")
	retractedAfterBoundary := uuid.MustParse("43000000-0000-0000-0000-000000000004")
	suggestions := []uuid.UUID{
		uuid.MustParse("43100000-0000-0000-0000-000000000001"),
		uuid.MustParse("43100000-0000-0000-0000-000000000002"),
		uuid.MustParse("43100000-0000-0000-0000-000000000003"),
		uuid.MustParse("43100000-0000-0000-0000-000000000004"),
	}
	for i, id := range suggestions {
		if _, err := pool.Exec(ctx, `
			INSERT INTO instagram_follow_suggestions(
				id,importer_did,target_did,state,reason,created_at,updated_at
			) VALUES($1,$2,$3,'pending','verifiedInstagramFollow',$4,$4)
		`, id, fmt.Sprintf("did:plc:retention-notification-%d", i), fmt.Sprintf("did:plc:retention-notification-target-%d", i), now); err != nil {
			t.Fatalf("seed notification suggestion %d: %v", i, err)
		}
	}
	seedLifecycleNotification(t, pool, eventAtBoundary, syntax.DID("did:plc:retention-event-a"), suggestions[0], "43200000-0000-0000-0000-000000000001", "cancelled", now.Add(-90*24*time.Hour))
	seedLifecycleNotification(t, pool, eventAfterBoundary, syntax.DID("did:plc:retention-event-b"), suggestions[1], "43200000-0000-0000-0000-000000000002", "cancelled", now.Add(-90*24*time.Hour+time.Microsecond))
	seedLifecycleNotification(t, pool, retractedAtBoundary, syntax.DID("did:plc:retention-delivery-a"), suggestions[2], "43200000-0000-0000-0000-000000000003", "cancelled", now.Add(-7*24*time.Hour))
	seedLifecycleNotification(t, pool, retractedAfterBoundary, syntax.DID("did:plc:retention-delivery-b"), suggestions[3], "43200000-0000-0000-0000-000000000004", "cancelled", now.Add(-7*24*time.Hour+time.Microsecond))
	if _, err := pool.Exec(ctx, `
		UPDATE notification_events
		SET state='retracted',retracted_at=activity_at,retraction_reason='synthetic_retention'
		WHERE id=ANY($1::uuid[])
	`, []uuid.UUID{retractedAtBoundary, retractedAfterBoundary}); err != nil {
		t.Fatalf("retract delivery events: %v", err)
	}

	service := NewRetentionService(pool, func() time.Time { return now })
	stats, err := service.Run(ctx, 500)
	if err != nil {
		t.Fatalf("run notification retention: %v", err)
	}
	if stats.NotificationsPurged != 1 || stats.DeliveriesPurged != 1 {
		t.Fatalf("notification retention stats=%+v", stats)
	}
	assertRetentionExists(t, pool, "notification_events", eventAtBoundary.String(), false)
	assertRetentionExists(t, pool, "notification_events", eventAfterBoundary.String(), true)
	assertRetentionExists(t, pool, "push_deliveries", "43200000-0000-0000-0000-000000000003", false)
	assertRetentionExists(t, pool, "push_deliveries", "43200000-0000-0000-0000-000000000004", true)
}

func newRetentionTest(t *testing.T) (*pgxpool.Pool, time.Time) {
	t.Helper()
	var ddl strings.Builder
	for _, path := range []string{
		"../../migrations/000021_appview_notifications.up.sql",
		"../../migrations/000022_notification_newness.up.sql",
		"../../migrations/000023_instagram_migration.up.sql",
		"../../migrations/000024_system_notifications.up.sql",
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
