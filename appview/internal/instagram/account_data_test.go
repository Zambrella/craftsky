package instagram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/testdb"
)

var (
	lifecycleAlice = syntax.DID("did:plc:synthetic-lifecycle-alice")
	lifecycleBob   = syntax.DID("did:plc:synthetic-lifecycle-bob")
	lifecycleCarol = syntax.DID("did:plc:synthetic-lifecycle-carol")
	lifecycleDave  = syntax.DID("did:plc:synthetic-lifecycle-dave")

	lifecycleAliceAttempt  = uuid.MustParse("10000000-0000-0000-0000-000000000001")
	lifecycleAliceLink     = uuid.MustParse("20000000-0000-0000-0000-000000000001")
	lifecycleBobLink       = uuid.MustParse("20000000-0000-0000-0000-000000000002")
	lifecycleAliceImport   = uuid.MustParse("30000000-0000-0000-0000-000000000001")
	lifecycleBobImport     = uuid.MustParse("30000000-0000-0000-0000-000000000002")
	lifecycleOwnerPending  = uuid.MustParse("40000000-0000-0000-0000-000000000001")
	lifecycleTargetPending = uuid.MustParse("40000000-0000-0000-0000-000000000002")
	lifecycleAccepted      = uuid.MustParse("40000000-0000-0000-0000-000000000003")
	lifecycleAliceEvent    = uuid.MustParse("50000000-0000-0000-0000-000000000001")
	lifecycleBobEvent      = uuid.MustParse("50000000-0000-0000-0000-000000000002")
)

func TestPrivateDataMembershipInactivationIsReversibleIdempotentAndConcurrent(t *testing.T) {
	service, pool, now := newPrivateDataTest(t)
	seedPrivateDataLifecycle(t, service, pool, now)

	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin profile lifecycle transaction: %v", err)
	}
	if err := service.InactivateMembershipTx(ctx, tx, lifecycleAlice, now.Add(-time.Hour)); err != nil {
		t.Fatalf("transactional membership inactivation: %v", err)
	}
	if err := tx.Rollback(ctx); err != nil {
		t.Fatalf("rollback profile lifecycle transaction: %v", err)
	}
	var beforeState InstagramLinkState
	if err := pool.QueryRow(ctx, `SELECT state FROM instagram_account_links WHERE id=$1`, lifecycleAliceLink).Scan(&beforeState); err != nil {
		t.Fatalf("read rolled-back lifecycle state: %v", err)
	}
	if beforeState != LinkActive {
		t.Fatalf("transactional inactivation escaped rollback: %s", beforeState)
	}

	const workers = 8
	errs := make(chan error, workers)
	var ready sync.WaitGroup
	ready.Add(workers)
	start := make(chan struct{})
	for range workers {
		go func() {
			ready.Done()
			<-start
			errs <- service.InactivateMembership(ctx, lifecycleAlice)
		}()
	}
	ready.Wait()
	close(start)
	for range workers {
		if err := <-errs; err != nil {
			t.Fatalf("concurrent membership inactivation: %v", err)
		}
	}

	var linkState InstagramLinkState
	var discoverable bool
	var linkInactiveAt time.Time
	if err := pool.QueryRow(ctx, `
		SELECT state, discoverable, membership_inactive_at
		FROM instagram_account_links WHERE id=$1
	`, lifecycleAliceLink).Scan(&linkState, &discoverable, &linkInactiveAt); err != nil {
		t.Fatalf("read inactive link: %v", err)
	}
	if linkState != LinkMembershipInactive || discoverable || !linkInactiveAt.Equal(now) {
		t.Fatalf("link state=%s discoverable=%t inactiveAt=%v", linkState, discoverable, linkInactiveAt)
	}

	var importState InstagramImportState
	var importInactiveAt time.Time
	if err := pool.QueryRow(ctx, `
		SELECT state, membership_inactive_at
		FROM instagram_graph_imports WHERE id=$1
	`, lifecycleAliceImport).Scan(&importState, &importInactiveAt); err != nil {
		t.Fatalf("read inactive import: %v", err)
	}
	if importState != ImportMembershipInactive || !importInactiveAt.Equal(now) {
		t.Fatalf("import state=%s inactiveAt=%v", importState, importInactiveAt)
	}

	var attemptState VerificationAttemptState
	var retryCode AttemptRetryCode
	var attemptSensitive int
	if err := pool.QueryRow(ctx, `
		SELECT state, retry_code,
		       num_nonnulls(challenge_digest, candidate_igsid, candidate_username)
		FROM instagram_verification_attempts WHERE id=$1
	`, lifecycleAliceAttempt).Scan(&attemptState, &retryCode, &attemptSensitive); err != nil {
		t.Fatalf("read rejected attempt: %v", err)
	}
	if attemptState != AttemptRejected || retryCode != RetryMembershipInactive || attemptSensitive != 0 {
		t.Fatalf("attempt state=%s retry=%s sensitive=%d", attemptState, retryCode, attemptSensitive)
	}

	var workStatus WebhookWorkStatus
	var workSensitive int
	if err := pool.QueryRow(ctx, `
		SELECT status,
		       num_nonnulls(sender_igsid, official_account_id, challenge_digest, lease_token)
		FROM instagram_webhook_work WHERE verification_attempt_id=$1
	`, lifecycleAliceAttempt).Scan(&workStatus, &workSensitive); err != nil {
		t.Fatalf("read cancelled webhook work: %v", err)
	}
	if workStatus != WebhookWorkIgnored || workSensitive != 0 {
		t.Fatalf("webhook state=%s sensitive=%d", workStatus, workSensitive)
	}

	states := readSuggestionStates(t, pool)
	if states[lifecycleOwnerPending] != SuggestionInvalidated ||
		states[lifecycleTargetPending] != SuggestionInvalidated ||
		states[lifecycleAccepted] != SuggestionAccepted {
		t.Fatalf("suggestion states after inactivation = %+v", states)
	}
	var acceptedOperation FollowOperationStatus
	if err := pool.QueryRow(ctx, `SELECT status FROM pds_follow_operations WHERE suggestion_id=$1`, lifecycleAccepted).Scan(&acceptedOperation); err != nil {
		t.Fatalf("read accepted PDS operation: %v", err)
	}
	if acceptedOperation != FollowOperationSucceeded {
		t.Fatalf("accepted PDS operation changed to %s", acceptedOperation)
	}

	for _, eventID := range []uuid.UUID{lifecycleAliceEvent, lifecycleBobEvent} {
		var eventState, deliveryState string
		if err := pool.QueryRow(ctx, `
			SELECT event.state, delivery.status
			FROM notification_events event
			JOIN push_deliveries delivery ON delivery.notification_id=event.id
			WHERE event.id=$1
		`, eventID).Scan(&eventState, &deliveryState); err != nil {
			t.Fatalf("read retracted notification %s: %v", eventID, err)
		}
		if eventState != "retracted" || deliveryState != "cancelled" {
			t.Fatalf("event %s state=%s delivery=%s", eventID, eventState, deliveryState)
		}
	}
	var jobStatus string
	if err := pool.QueryRow(ctx, `SELECT status FROM instagram_reconciliation_jobs WHERE owner_did=$1`, lifecycleAlice).Scan(&jobStatus); err != nil {
		t.Fatalf("read paused reconciliation: %v", err)
	}
	if jobStatus != "ignored" {
		t.Fatalf("reconciliation state=%s", jobStatus)
	}

	// Rejoining current membership is deliberately not a lifecycle transition.
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles(did) VALUES($1) ON CONFLICT DO NOTHING`, lifecycleAlice); err != nil {
		t.Fatalf("restore current membership: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT state FROM instagram_account_links WHERE id=$1`, lifecycleAliceLink).Scan(&linkState); err != nil {
		t.Fatalf("read link after rejoin: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT state FROM instagram_graph_imports WHERE id=$1`, lifecycleAliceImport).Scan(&importState); err != nil {
		t.Fatalf("read import after rejoin: %v", err)
	}
	if linkState != LinkMembershipInactive || importState != ImportMembershipInactive {
		t.Fatalf("rejoin silently restored link=%s import=%s", linkState, importState)
	}
}

func TestPrivateDataExportIsOwnerScopedAndOmitsInfrastructureSecrets(t *testing.T) {
	service, pool, now := newPrivateDataTest(t)
	seedPrivateDataLifecycle(t, service, pool, now)

	export, err := service.ExportOwnerData(context.Background(), lifecycleAlice)
	if err != nil {
		t.Fatalf("ExportOwnerData: %v", err)
	}
	if export.OwnerDID != lifecycleAlice || len(export.VerificationAttempts) != 1 ||
		len(export.AccountLinks) != 1 || len(export.Imports) != 1 ||
		len(export.Suggestions) != 2 || len(export.MatchNotifications) != 1 {
		t.Fatalf("unexpected private export shape: %+v", export)
	}
	if export.AccountLinks[0].InstagramUserID != "synthetic-alice-igsid" ||
		export.AccountLinks[0].Username != "synthetic.alice" {
		t.Fatalf("owner link missing from private export: %+v", export.AccountLinks[0])
	}
	if len(export.Imports[0].RetainedEntries) != 1 ||
		export.Imports[0].RetainedEntries[0].Username != "synthetic.carol" {
		t.Fatalf("owner retained entries = %+v", export.Imports[0].RetainedEntries)
	}
	for _, suggestion := range export.Suggestions {
		if suggestion.TargetDID == lifecycleAlice {
			t.Fatalf("other importer's target fact leaked into export: %+v", suggestion)
		}
	}
	encoded, err := json.Marshal(export)
	if err != nil {
		t.Fatalf("marshal private export: %v", err)
	}
	for _, forbidden := range []string{
		"synthetic.bob.private.handle", "synthetic-bob-igsid",
		"challengeDigest", "igsidDigest", "messageDigest", "rateLimit",
		"leaseToken", "routingId", "fcmToken",
	} {
		if bytes.Contains(encoded, []byte(forbidden)) {
			t.Fatalf("private export leaked %q: %s", forbidden, encoded)
		}
	}
	if diagnostic := fmt.Sprintf("%+v", export); strings.Contains(diagnostic, "synthetic.alice") || strings.Contains(diagnostic, "synthetic-alice-igsid") {
		t.Fatalf("private export diagnostic leaked identity: %s", diagnostic)
	}

	missing, err := service.ExportOwnerData(context.Background(), syntax.DID("did:plc:synthetic-lifecycle-missing"))
	if err != nil {
		t.Fatalf("export absent owner: %v", err)
	}
	if len(missing.AccountLinks) != 0 || len(missing.Imports) != 0 || len(missing.Suggestions) != 0 {
		t.Fatalf("absent export contains data: %+v", missing)
	}
}

func TestPrivateDataScopedPurgeIsOwnerBoundAndPreservesAcceptedFollowLedger(t *testing.T) {
	service, pool, now := newPrivateDataTest(t)
	seedPrivateDataLifecycle(t, service, pool, now)
	ctx := context.Background()

	if err := service.PurgeImport(ctx, lifecycleAlice, lifecycleBobImport); err != nil {
		t.Fatalf("foreign import purge: %v", err)
	}
	assertRowExists(t, pool, "instagram_graph_imports", lifecycleBobImport, true)
	if err := service.PurgeImport(ctx, lifecycleAlice, lifecycleAliceImport); err != nil {
		t.Fatalf("owner import purge: %v", err)
	}
	if err := service.PurgeImport(ctx, lifecycleAlice, lifecycleAliceImport); err != nil {
		t.Fatalf("owner import purge replay: %v", err)
	}
	assertRowExists(t, pool, "instagram_graph_imports", lifecycleAliceImport, false)
	states := readSuggestionStates(t, pool)
	if states[lifecycleOwnerPending] != SuggestionInvalidated || states[lifecycleAccepted] != SuggestionAccepted {
		t.Fatalf("scoped import suggestion states = %+v", states)
	}
	assertFollowLedgerStatus(t, pool, lifecycleAccepted, FollowOperationSucceeded)

	if err := service.PurgeLink(ctx, lifecycleAlice, lifecycleBobLink); err != nil {
		t.Fatalf("foreign link purge: %v", err)
	}
	assertRowExists(t, pool, "instagram_account_links", lifecycleBobLink, true)
	if err := service.PurgeLink(ctx, lifecycleAlice, lifecycleAliceLink); err != nil {
		t.Fatalf("owner link purge: %v", err)
	}
	if err := service.PurgeLink(ctx, lifecycleAlice, lifecycleAliceLink); err != nil {
		t.Fatalf("owner link purge replay: %v", err)
	}
	assertRowExists(t, pool, "instagram_account_links", lifecycleAliceLink, false)
	states = readSuggestionStates(t, pool)
	if states[lifecycleTargetPending] != SuggestionInvalidated {
		t.Fatalf("target-dependent suggestion state=%s", states[lifecycleTargetPending])
	}
	assertFollowLedgerStatus(t, pool, lifecycleAccepted, FollowOperationSucceeded)
	assertRowExists(t, pool, "instagram_account_links", lifecycleBobLink, true)
}

func TestPrivateDataTerminalPurgeIsIdempotentConcurrentAndLeavesOtherOwners(t *testing.T) {
	service, pool, now := newPrivateDataTest(t)
	seedPrivateDataLifecycle(t, service, pool, now)
	ctx := context.Background()

	const workers = 6
	errs := make(chan error, workers)
	var ready sync.WaitGroup
	ready.Add(workers)
	start := make(chan struct{})
	for range workers {
		go func() {
			ready.Done()
			<-start
			errs <- service.PurgeOwner(ctx, lifecycleAlice)
		}()
	}
	ready.Wait()
	close(start)
	for range workers {
		if err := <-errs; err != nil {
			t.Fatalf("concurrent terminal purge: %v", err)
		}
	}

	ownerChecks := []struct {
		table, predicate string
	}{
		{"instagram_verification_attempts", "owner_did=$1"},
		{"instagram_account_links", "owner_did=$1"},
		{"instagram_identity_claims", "owner_did=$1"},
		{"instagram_graph_imports", "owner_did=$1"},
		{"instagram_reconciliation_jobs", "owner_did=$1 OR target_did=$1"},
		{"instagram_follow_suggestions", "importer_did=$1 OR target_did=$1"},
		{"pds_follow_operations", "owner_did=$1 OR target_did=$1"},
	}
	for _, check := range ownerChecks {
		var count int
		query := fmt.Sprintf("SELECT count(*) FROM %s WHERE %s", check.table, check.predicate)
		if err := pool.QueryRow(ctx, query, lifecycleAlice).Scan(&count); err != nil {
			t.Fatalf("count %s: %v", check.table, err)
		}
		if count != 0 {
			t.Fatalf("terminal purge left %d rows in %s", count, check.table)
		}
	}
	var workCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM instagram_webhook_work WHERE verification_attempt_id=$1`, lifecycleAliceAttempt).Scan(&workCount); err != nil {
		t.Fatalf("count purged webhook work: %v", err)
	}
	if workCount != 0 {
		t.Fatalf("terminal purge left webhook work=%d", workCount)
	}
	assertRowExists(t, pool, "instagram_account_links", lifecycleBobLink, true)
	assertRowExists(t, pool, "instagram_graph_imports", lifecycleBobImport, true)

	var bobEventState, bobDeliveryState string
	if err := pool.QueryRow(ctx, `
		SELECT event.state, delivery.status
		FROM notification_events event
		JOIN push_deliveries delivery ON delivery.notification_id=event.id
		WHERE event.id=$1
	`, lifecycleBobEvent).Scan(&bobEventState, &bobDeliveryState); err != nil {
		t.Fatalf("read cross-user notification after purge: %v", err)
	}
	if bobEventState != "retracted" || bobDeliveryState != "cancelled" {
		t.Fatalf("cross-user event=%s delivery=%s", bobEventState, bobDeliveryState)
	}
	assertRowExists(t, pool, "notification_events", lifecycleAliceEvent, false)

	var auditOwner, auditSubject *string
	if err := pool.QueryRow(ctx, `SELECT owner_did, subject_id FROM instagram_audit_events WHERE action='syntheticLifecycle'`).Scan(&auditOwner, &auditSubject); err != nil {
		t.Fatalf("read anonymized audit: %v", err)
	}
	if auditOwner != nil || auditSubject != nil {
		t.Fatalf("terminal audit retained owner=%v subject=%v", auditOwner, auditSubject)
	}
	assertOwnerRateBuckets(t, service, pool, lifecycleAlice, "synthetic-alice-igsid", false)
	assertOwnerRateBuckets(t, service, pool, lifecycleBob, "synthetic-bob-igsid", true)

	export, err := service.ExportOwnerData(ctx, lifecycleAlice)
	if err != nil {
		t.Fatalf("export after terminal purge: %v", err)
	}
	if len(export.VerificationAttempts)+len(export.AccountLinks)+len(export.Imports)+len(export.Suggestions)+len(export.MatchNotifications) != 0 {
		t.Fatalf("terminally purged export not empty: %+v", export)
	}
}

func newPrivateDataTest(t *testing.T) (*PrivateDataService, *pgxpool.Pool, time.Time) {
	t.Helper()
	var ddl strings.Builder
	ddl.WriteString(`CREATE TABLE craftsky_profiles (did TEXT PRIMARY KEY);`)
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
	pool := testdb.WithSchema(t, ddl.String())
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	limiter, err := NewPostgresRateLimiter(pool, bytes.Repeat([]byte{0x6c}, 32), func() time.Time { return now })
	if err != nil {
		t.Fatalf("new lifecycle limiter: %v", err)
	}
	return NewPrivateDataService(pool, limiter, func() time.Time { return now }), pool, now
}

func seedPrivateDataLifecycle(t *testing.T, service *PrivateDataService, pool *pgxpool.Pool, now time.Time) {
	t.Helper()
	ctx := context.Background()
	digestA := bytes.Repeat([]byte{0xa1}, 32)
	digestB := bytes.Repeat([]byte{0xb2}, 32)
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_profiles(did) VALUES($1),($2),($3),($4)
	`, lifecycleAlice, lifecycleBob, lifecycleCarol, lifecycleDave); err != nil {
		t.Fatalf("seed lifecycle profiles: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_verification_attempts(
			id,owner_did,state,candidate_igsid,expires_at,
			processing_started_at,created_at,updated_at
		) VALUES($1,$2,'processing','synthetic-alice-igsid',$3,$4,$4,$4)
	`, lifecycleAliceAttempt, lifecycleAlice, now.Add(10*time.Minute), now); err != nil {
		t.Fatalf("seed lifecycle attempt: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_webhook_work(
			id,verification_attempt_id,message_digest_version,message_digest,
			sender_igsid,official_account_id,challenge_digest_version,
			challenge_digest,event_at,status,attempts,next_attempt_at,
			created_at,updated_at
		) VALUES(
			'11000000-0000-0000-0000-000000000001',$1,1,$2,
			'synthetic-alice-igsid','synthetic-official',1,$3,$4,
			'queued',0,$4,$4,$4
		)
	`, lifecycleAliceAttempt, digestB, digestA, now); err != nil {
		t.Fatalf("seed lifecycle work: %v", err)
	}
	insertLifecycleLink(t, pool, lifecycleAliceLink, lifecycleAlice, "synthetic-alice-igsid", "synthetic.alice", digestA, now)
	insertLifecycleLink(t, pool, lifecycleBobLink, lifecycleBob, "synthetic-bob-igsid", "synthetic.bob", digestB, now)

	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_graph_imports(
			id,owner_did,state,source_type,retain_unmatched,
			retention_expires_at,following_count,follower_count,created_at,updated_at
		) VALUES
			($1,$2,'active','manual',true,$3,1,0,$4,$4),
			($5,$6,'active','manual',true,$3,1,0,$4,$4)
	`, lifecycleAliceImport, lifecycleAlice, now.AddDate(1, 0, 0), now,
		lifecycleBobImport, lifecycleBob); err != nil {
		t.Fatalf("seed lifecycle imports: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_graph_handles(
			import_id,username_normalized,direction,matched,retain_until,created_at
		) VALUES
			($1,'synthetic.carol','following',true,$2,$3),
			($4,'synthetic.bob.private.handle','following',true,$2,$3)
	`, lifecycleAliceImport, now.AddDate(1, 0, 0), now, lifecycleBobImport); err != nil {
		t.Fatalf("seed lifecycle handles: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_follow_suggestions(
			id,importer_did,target_did,state,reason,terminal_at,created_at,updated_at
		) VALUES
			($1,$2,$3,'pending','verifiedInstagramFollow',NULL,$4,$4),
			($5,$6,$2,'pending','verifiedInstagramFollow',NULL,$4,$4),
			($7,$2,$8,'accepted','verifiedInstagramFollow',$4,$4,$4)
	`, lifecycleOwnerPending, lifecycleAlice, lifecycleCarol, now,
		lifecycleTargetPending, lifecycleBob, lifecycleAccepted, lifecycleDave); err != nil {
		t.Fatalf("seed lifecycle suggestions: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_suggestion_sources(suggestion_id,import_id,created_at) VALUES
			($1,$2,$3),($4,$5,$3),($6,$2,$3)
	`, lifecycleOwnerPending, lifecycleAliceImport, now,
		lifecycleTargetPending, lifecycleBobImport, lifecycleAccepted); err != nil {
		t.Fatalf("seed lifecycle suggestion sources: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO pds_follow_operations(
			id,suggestion_id,owner_did,target_did,rkey,status,
			attempt_count,created_at,updated_at,completed_at
		) VALUES
			('41000000-0000-0000-0000-000000000001',$1,$2,$3,'3kysyntheticpending','writing',1,$4,$4,NULL),
			('41000000-0000-0000-0000-000000000003',$5,$2,$6,'3kysyntheticaccepted','succeeded',1,$4,$4,$4)
	`, lifecycleOwnerPending, lifecycleAlice, lifecycleCarol, now,
		lifecycleAccepted, lifecycleDave); err != nil {
		t.Fatalf("seed lifecycle follow operations: %v", err)
	}
	seedLifecycleNotification(t, pool, lifecycleAliceEvent, lifecycleAlice, lifecycleOwnerPending, "42000000-0000-0000-0000-000000000001", "pending", now)
	seedLifecycleNotification(t, pool, lifecycleBobEvent, lifecycleBob, lifecycleTargetPending, "42000000-0000-0000-0000-000000000002", "leased", now)
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_reconciliation_jobs(
			id,owner_did,target_did,link_id,import_id,reason,status,
			attempts,next_attempt_at,created_at,updated_at
		) VALUES(
			'60000000-0000-0000-0000-000000000001',$1,$3,$2,$4,
			'syntheticLifecycle','queued',0,$5,$5,$5
		)
	`, lifecycleAlice, lifecycleAliceLink, lifecycleCarol, lifecycleAliceImport, now); err != nil {
		t.Fatalf("seed lifecycle job: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_audit_events(owner_did,action,subject_kind,subject_id,outcome,created_at)
		VALUES($1,'syntheticLifecycle','link',$2,'synthetic',$3)
	`, lifecycleAlice, lifecycleAliceLink, now); err != nil {
		t.Fatalf("seed lifecycle audit: %v", err)
	}
	seedOwnerRateBuckets(t, service, lifecycleAlice, "synthetic-alice-igsid")
	seedOwnerRateBuckets(t, service, lifecycleBob, "synthetic-bob-igsid")
}

func insertLifecycleLink(t *testing.T, pool *pgxpool.Pool, id uuid.UUID, owner syntax.DID, igsid, username string, digest []byte, now time.Time) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO instagram_account_links(
			id,owner_did,state,igsid,igsid_digest_version,igsid_digest,
			username,username_normalized,discoverable,conflict_pending,
			verified_at,created_at,updated_at
		) VALUES($1,$2,'active',$3,1,$4,$5,$5,true,false,$6,$6,$6)
	`, id, owner, igsid, digest, username, now); err != nil {
		t.Fatalf("insert lifecycle link: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO instagram_identity_claims(
			id,link_id,owner_did,state,igsid_digest_version,igsid_digest,
			claimed_at,created_at,updated_at
		) VALUES(gen_random_uuid(),$1,$2,'active',1,$3,$4,$4,$4)
	`, id, owner, digest, now); err != nil {
		t.Fatalf("insert lifecycle claim: %v", err)
	}
}

func seedLifecycleNotification(t *testing.T, pool *pgxpool.Pool, eventID uuid.UUID, recipient syntax.DID, suggestionID uuid.UUID, deliveryID, deliveryStatus string, now time.Time) {
	t.Helper()
	ctx := context.Background()
	installationID := uuid.New()
	subscriptionID := uuid.New()
	parsedDeliveryID := uuid.MustParse(deliveryID)
	leaseOwner := any(nil)
	leaseExpiresAt := any(nil)
	if deliveryStatus == "leased" {
		leaseOwner = "synthetic-worker"
		leaseExpiresAt = now.Add(time.Minute)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO notification_preferences(account_did,category,scope,push_enabled,created_at,updated_at)
		VALUES($1,'instagramMatch','everyone',true,$2,$2)
		ON CONFLICT(account_did,category) DO NOTHING
	`, recipient, now); err != nil {
		t.Fatalf("seed lifecycle notification preference: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO notification_events(
			id,recipient_did,kind,category,subject_key,
			eligibility_scope,recipient_followed_actor,push_enabled_snapshot,
			state,first_activity_at,activity_at,indexed_at,
			initial_push_evaluated_at,system_count,system_count_capped,
			system_destination,system_group_key,coalesce_until
		) VALUES(
			$1,$2,'system','instagramMatch',$3,
			'everyone',false,true,'active',$4,$4,$4,$4,
			1,false,'instagramMigration',$3,$4::timestamptz + interval '5 minutes'
		)
	`, eventID, recipient, eventID.String(), now); err != nil {
		t.Fatalf("seed lifecycle notification event: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_notification_suggestions(notification_id,suggestion_id,created_at)
		VALUES($1,$2,$3)
	`, eventID, suggestionID, now); err != nil {
		t.Fatalf("seed lifecycle notification support: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO push_installations(id,device_id,platform,fcm_token,created_at,updated_at)
		VALUES($1,$2,'ios',$3,$4,$4)
	`, installationID, "device-"+eventID.String(), "token-"+eventID.String(), now); err != nil {
		t.Fatalf("seed lifecycle push installation: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO push_account_subscriptions(
			id,installation_id,account_did,routing_id,created_at,updated_at
		) VALUES($1,$2,$3,$4,$5,$5)
	`, subscriptionID, installationID, recipient, uuid.New(), now); err != nil {
		t.Fatalf("seed lifecycle push subscription: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO push_deliveries(
			id,notification_id,account_subscription_id,status,next_attempt_at,
			deadline_at,lease_owner,lease_expires_at,created_at,updated_at
		) VALUES($1,$2,$3,$4,$5,$5::timestamptz + interval '6 hours',$6,$7,$5,$5)
	`, parsedDeliveryID, eventID, subscriptionID, deliveryStatus, now, leaseOwner, leaseExpiresAt); err != nil {
		t.Fatalf("seed lifecycle push delivery: %v", err)
	}
}

func seedOwnerRateBuckets(t *testing.T, service *PrivateDataService, owner syntax.DID, igsid string) {
	t.Helper()
	for _, scopedIdentifier := range []struct {
		scope      RateLimitScope
		identifier string
	}{
		{RateLimitChallengeDID, owner.String()},
		{RateLimitConfirmationDID, owner.String()},
		{RateLimitImportDID, owner.String()},
		{RateLimitInvalidRedemptionIGSID, igsid},
		{RateLimitMetaLookupIGSID, igsid},
	} {
		key, err := service.rateLimiter.Key(scopedIdentifier.scope, []byte(scopedIdentifier.identifier))
		if err != nil {
			t.Fatalf("build lifecycle rate key: %v", err)
		}
		if _, err := service.rateLimiter.Allow(context.Background(), key, time.Hour, 10); err != nil {
			t.Fatalf("seed lifecycle rate bucket: %v", err)
		}
	}
}

func assertOwnerRateBuckets(t *testing.T, service *PrivateDataService, pool *pgxpool.Pool, owner syntax.DID, igsid string, want bool) {
	t.Helper()
	for _, scopedIdentifier := range []struct {
		scope      RateLimitScope
		identifier string
	}{
		{RateLimitChallengeDID, owner.String()},
		{RateLimitConfirmationDID, owner.String()},
		{RateLimitImportDID, owner.String()},
		{RateLimitInvalidRedemptionIGSID, igsid},
		{RateLimitMetaLookupIGSID, igsid},
	} {
		key, err := service.rateLimiter.Key(scopedIdentifier.scope, []byte(scopedIdentifier.identifier))
		if err != nil {
			t.Fatalf("build asserted rate key: %v", err)
		}
		var exists bool
		if err := pool.QueryRow(context.Background(), `
			SELECT EXISTS(
				SELECT 1 FROM instagram_rate_limit_buckets
				WHERE bucket_scope=$1 AND key_version=$2 AND key_digest=$3
			)
		`, key.scope, key.version, key.digest[:]).Scan(&exists); err != nil {
			t.Fatalf("read asserted rate key: %v", err)
		}
		if exists != want {
			t.Fatalf("rate bucket %s exists=%t want=%t", scopedIdentifier.scope, exists, want)
		}
	}
}

func readSuggestionStates(t *testing.T, pool *pgxpool.Pool) map[uuid.UUID]InstagramSuggestionState {
	t.Helper()
	rows, err := pool.Query(context.Background(), `SELECT id,state FROM instagram_follow_suggestions`)
	if err != nil {
		t.Fatalf("read suggestion states: %v", err)
	}
	defer rows.Close()
	states := make(map[uuid.UUID]InstagramSuggestionState)
	for rows.Next() {
		var id uuid.UUID
		var state InstagramSuggestionState
		if err := rows.Scan(&id, &state); err != nil {
			t.Fatalf("scan suggestion state: %v", err)
		}
		states[id] = state
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate suggestion states: %v", err)
	}
	return states
}

func assertFollowLedgerStatus(t *testing.T, pool *pgxpool.Pool, suggestionID uuid.UUID, want FollowOperationStatus) {
	t.Helper()
	var got FollowOperationStatus
	if err := pool.QueryRow(context.Background(), `SELECT status FROM pds_follow_operations WHERE suggestion_id=$1`, suggestionID).Scan(&got); err != nil {
		t.Fatalf("read follow ledger %s: %v", suggestionID, err)
	}
	if got != want {
		t.Fatalf("follow ledger %s=%s want=%s", suggestionID, got, want)
	}
}

func assertRowExists(t *testing.T, pool *pgxpool.Pool, table string, id uuid.UUID, want bool) {
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
