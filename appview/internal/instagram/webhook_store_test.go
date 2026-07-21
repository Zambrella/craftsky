package instagram

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"social.craftsky/appview/internal/integrations/instagrammeta"
	"social.craftsky/appview/internal/testdb"
)

func TestWebhookStoreEnqueueDeduplicatesMinimalBatchAtomically(t *testing.T) {
	store := newWebhookTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	first := syntheticWebhookItem(0x11, 0x21, "synthetic-sender-1", now.Add(-time.Minute))
	second := syntheticWebhookItem(0x12, 0x22, "synthetic-sender-2", now)

	inserted, err := store.EnqueueWebhookWork(ctx, []instagrammeta.WorkItem{first, second}, now)
	if err != nil {
		t.Fatalf("EnqueueWebhookWork: %v", err)
	}
	if inserted != 2 {
		t.Fatalf("inserted = %d, want 2", inserted)
	}
	inserted, err = store.EnqueueWebhookWork(ctx, []instagrammeta.WorkItem{second, first}, now.Add(time.Second))
	if err != nil {
		t.Fatalf("EnqueueWebhookWork duplicates: %v", err)
	}
	if inserted != 0 {
		t.Fatalf("duplicate inserted = %d, want 0", inserted)
	}

	var count int
	if err := store.pool.QueryRow(ctx, `SELECT count(*) FROM instagram_webhook_work`).Scan(&count); err != nil {
		t.Fatalf("count work: %v", err)
	}
	if count != 2 {
		t.Fatalf("work count = %d, want 2", count)
	}
	var (
		messageVersion, challengeVersion int
		messageDigest, challengeDigest   []byte
		sender, account, status          string
		eventAt, nextAttempt             time.Time
		attempts                         int
	)
	if err := store.pool.QueryRow(ctx, `
		SELECT message_digest_version, message_digest, sender_igsid,
		       official_account_id, challenge_digest_version,
		       challenge_digest, event_at, status, attempts, next_attempt_at
		FROM instagram_webhook_work
		WHERE message_digest = $1
	`, first.MessageIDDigest.Value[:]).Scan(
		&messageVersion, &messageDigest, &sender, &account,
		&challengeVersion, &challengeDigest, &eventAt, &status,
		&attempts, &nextAttempt,
	); err != nil {
		t.Fatalf("inspect work: %v", err)
	}
	if messageVersion != first.MessageIDDigest.Version || !bytes.Equal(messageDigest, first.MessageIDDigest.Value[:]) ||
		challengeVersion != first.ChallengeDigest.Version || !bytes.Equal(challengeDigest, first.ChallengeDigest.Value[:]) {
		t.Fatal("stored digests do not match reduced work")
	}
	if sender != first.SenderIGSID || account != first.OfficialAccountID || status != string(WebhookWorkQueued) || attempts != 0 {
		t.Fatalf("stored work = sender/account/status/attempts %q/%q/%q/%d", sender, account, status, attempts)
	}
	if !eventAt.Equal(first.EventAt) || !nextAttempt.Equal(now) {
		t.Fatalf("stored times = event %s next %s", eventAt, nextAttempt)
	}

	third := syntheticWebhookItem(0x13, 0x23, "synthetic-sender-3", now)
	invalid := third
	invalid.ChallengeDigest = instagrammeta.KeyedDigest{}
	if _, err := store.EnqueueWebhookWork(ctx, []instagrammeta.WorkItem{third, invalid}, now); err == nil {
		t.Fatal("EnqueueWebhookWork accepted an invalid batch")
	}
	if err := store.pool.QueryRow(ctx, `SELECT count(*) FROM instagram_webhook_work`).Scan(&count); err != nil {
		t.Fatalf("count after invalid batch: %v", err)
	}
	if count != 2 {
		t.Fatalf("invalid batch partially inserted; count = %d", count)
	}
}

func TestWebhookStoreGuardedEnqueueLimitsOnlyUniqueNonRedeemableEvents(t *testing.T) {
	store := newWebhookTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	owner := "did:plc:synthetic-current-member"
	redeemable := syntheticWebhookItem(0x14, 0x24, "synthetic-valid-sender", now)
	if _, err := store.pool.Exec(ctx, `INSERT INTO craftsky_profiles (did) VALUES ($1)`, owner); err != nil {
		t.Fatalf("insert current member: %v", err)
	}
	if _, err := store.pool.Exec(ctx, `
		INSERT INTO instagram_verification_attempts (
			id, owner_did, state, challenge_digest_version,
			challenge_digest, expires_at, created_at, updated_at
		) VALUES ($1, $2, 'pendingDm', $3, $4, $5, $6, $6)
	`, uuid.New(), owner, redeemable.ChallengeDigest.Version,
		redeemable.ChallengeDigest.Value[:], now.Add(time.Minute), now); err != nil {
		t.Fatalf("insert redeemable attempt: %v", err)
	}

	firstInvalid := syntheticWebhookItem(0x15, 0x25, "synthetic-invalid-sender-1", now)
	secondInvalid := syntheticWebhookItem(0x16, 0x26, "synthetic-invalid-sender-2", now)
	overLimit := syntheticWebhookItem(0x17, 0x27, "synthetic-invalid-sender-3", now)
	limiter := &sequenceInvalidRedemptionLimiter{allowed: []bool{true, true, false}}
	items := []instagrammeta.WorkItem{redeemable, firstInvalid, secondInvalid, overLimit}
	inserted, err := store.EnqueueWebhookWorkGuarded(ctx, items, now, limiter)
	if err != nil {
		t.Fatalf("EnqueueWebhookWorkGuarded: %v", err)
	}
	if inserted != len(items) || limiter.calls != 3 {
		t.Fatalf("guarded enqueue = inserted %d limiter calls %d, want %d/3", inserted, limiter.calls, len(items))
	}

	// A replay is a durable no-op and must not consume invalid-redemption
	// quota, including for the already terminal over-limit message.
	inserted, err = store.EnqueueWebhookWorkGuarded(ctx, items, now.Add(time.Second), limiter)
	if err != nil {
		t.Fatalf("guarded replay: %v", err)
	}
	if inserted != 0 || limiter.calls != 3 {
		t.Fatalf("guarded replay = inserted %d limiter calls %d, want 0/3", inserted, limiter.calls)
	}

	var queued, ignored int
	if err := store.pool.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE status = 'queued'),
		       count(*) FILTER (WHERE status = 'ignored')
		FROM instagram_webhook_work
	`).Scan(&queued, &ignored); err != nil {
		t.Fatalf("count guarded work: %v", err)
	}
	if queued != 3 || ignored != 1 {
		t.Fatalf("guarded work states = queued %d ignored %d, want 3/1", queued, ignored)
	}

	var (
		status           WebhookWorkStatus
		reason           WebhookTerminalReason
		sender, official sql.NullString
		challengeVersion sql.NullInt16
		challenge        []byte
		terminalAt       sql.NullTime
	)
	if err := store.pool.QueryRow(ctx, `
		SELECT status, terminal_reason, sender_igsid, official_account_id,
		       challenge_digest_version, challenge_digest, terminal_at
		FROM instagram_webhook_work
		WHERE message_digest_version = $1 AND message_digest = $2
	`, overLimit.MessageIDDigest.Version, overLimit.MessageIDDigest.Value[:]).Scan(
		&status, &reason, &sender, &official, &challengeVersion, &challenge, &terminalAt,
	); err != nil {
		t.Fatalf("inspect terminal invalid work: %v", err)
	}
	if status != WebhookWorkIgnored || reason != WebhookReasonRateLimited ||
		sender.Valid || official.Valid || challengeVersion.Valid || len(challenge) != 0 ||
		!terminalAt.Valid || !terminalAt.Time.Equal(now) {
		t.Fatalf("terminal invalid work retained private worker fields: status=%q reason=%q sender=%t official=%t challengeVersion=%t challenge=%d terminal=%v",
			status, reason, sender.Valid, official.Valid, challengeVersion.Valid, len(challenge), terminalAt)
	}
}

func TestWebhookStoreGuardedEnqueueTreatsExpiredAndNonMemberChallengesAsInvalid(t *testing.T) {
	store := newWebhookTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)

	expired := syntheticWebhookItem(0x18, 0x28, "synthetic-expired-sender", now)
	nonMember := syntheticWebhookItem(0x19, 0x29, "synthetic-non-member-sender", now)
	for _, attempt := range []struct {
		owner     string
		item      instagrammeta.WorkItem
		expiresAt time.Time
	}{
		{owner: "did:plc:synthetic-expired-member", item: expired, expiresAt: now},
		{owner: "did:plc:synthetic-former-member", item: nonMember, expiresAt: now.Add(time.Minute)},
	} {
		if _, err := store.pool.Exec(ctx, `
			INSERT INTO instagram_verification_attempts (
				id, owner_did, state, challenge_digest_version,
				challenge_digest, expires_at, created_at, updated_at
			) VALUES ($1, $2, 'pendingDm', $3, $4, $5, $6, $6)
		`, uuid.New(), attempt.owner, attempt.item.ChallengeDigest.Version,
			attempt.item.ChallengeDigest.Value[:], attempt.expiresAt, now.Add(-time.Minute)); err != nil {
			t.Fatalf("insert boundary attempt: %v", err)
		}
	}
	if _, err := store.pool.Exec(ctx, `INSERT INTO craftsky_profiles (did) VALUES ('did:plc:synthetic-expired-member')`); err != nil {
		t.Fatalf("insert expired attempt member: %v", err)
	}

	limiter := &sequenceInvalidRedemptionLimiter{allowed: []bool{true, false}}
	inserted, err := store.EnqueueWebhookWorkGuarded(ctx, []instagrammeta.WorkItem{expired, nonMember}, now, limiter)
	if err != nil {
		t.Fatalf("EnqueueWebhookWorkGuarded: %v", err)
	}
	if inserted != 2 || limiter.calls != 2 {
		t.Fatalf("boundary guarded enqueue = inserted %d limiter calls %d, want 2/2", inserted, limiter.calls)
	}
	var status WebhookWorkStatus
	if err := store.pool.QueryRow(ctx, `
		SELECT status FROM instagram_webhook_work
		WHERE message_digest_version = $1 AND message_digest = $2
	`, nonMember.MessageIDDigest.Version, nonMember.MessageIDDigest.Value[:]).Scan(&status); err != nil {
		t.Fatalf("inspect non-member work: %v", err)
	}
	if status != WebhookWorkIgnored {
		t.Fatalf("non-member over-limit work status = %q, want ignored", status)
	}
}

func TestWebhookStoreGuardedEnqueueRollsBackEntireDeliveryOnLimiterFailure(t *testing.T) {
	store := newWebhookTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	items := []instagrammeta.WorkItem{
		syntheticWebhookItem(0x1a, 0x2a, "synthetic-invalid-sender-1", now),
		syntheticWebhookItem(0x1b, 0x2b, "synthetic-invalid-sender-2", now),
	}
	limiter := &sequenceInvalidRedemptionLimiter{
		allowed: []bool{true},
		errAt:   2,
		err:     errors.New("synthetic persistent limiter failure"),
	}
	if _, err := store.EnqueueWebhookWorkGuarded(ctx, items, now, limiter); !errors.Is(err, limiter.err) {
		t.Fatalf("EnqueueWebhookWorkGuarded error = %v, want limiter failure", err)
	}
	var count int
	if err := store.pool.QueryRow(ctx, `SELECT count(*) FROM instagram_webhook_work`).Scan(&count); err != nil {
		t.Fatalf("count rolled-back delivery: %v", err)
	}
	if count != 0 {
		t.Fatalf("limiter failure partially persisted %d rows", count)
	}
}

func TestWebhookStoreClaimsUniqueLeasesAndRecoversExpiredLease(t *testing.T) {
	store := newWebhookTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	item := syntheticWebhookItem(0x31, 0x41, "synthetic-sender", now.Add(-time.Minute))
	if inserted, err := store.EnqueueWebhookWork(ctx, []instagrammeta.WorkItem{item}, now); err != nil || inserted != 1 {
		t.Fatalf("enqueue = (%d, %v)", inserted, err)
	}

	claimed, err := store.ClaimWebhookWork(ctx, 1, now)
	if err != nil {
		t.Fatalf("first claim: %v", err)
	}
	if len(claimed) != 1 {
		t.Fatalf("first claim count = %d, want 1", len(claimed))
	}
	first := claimed[0]
	if first.Status != WebhookWorkProcessing || first.Attempts != 1 || first.LeaseToken == uuid.Nil {
		t.Fatalf("first claim = %+v", first)
	}
	if want := now.Add(WebhookLeaseDuration); !first.LeaseExpiresAt.Equal(want) {
		t.Fatalf("first lease expires = %s, want %s", first.LeaseExpiresAt, want)
	}
	if second, err := store.ClaimWebhookWork(ctx, 1, now); err != nil || len(second) != 0 {
		t.Fatalf("claim while leased = (%d, %v), want empty", len(second), err)
	}

	reclaimed, err := store.ClaimWebhookWork(ctx, 1, now.Add(WebhookLeaseDuration))
	if err != nil {
		t.Fatalf("reclaim: %v", err)
	}
	if len(reclaimed) != 1 {
		t.Fatalf("reclaim count = %d, want 1", len(reclaimed))
	}
	second := reclaimed[0]
	if second.ID != first.ID || second.Attempts != 2 || second.LeaseToken == first.LeaseToken {
		t.Fatalf("reclaimed work = %+v", second)
	}
	if err := store.CompleteWebhookWork(ctx, first.ID, first.LeaseToken, now.Add(WebhookLeaseDuration), WebhookReasonProcessed); !errors.Is(err, ErrWebhookLeaseLost) {
		t.Fatalf("stale completion error = %v, want %v", err, ErrWebhookLeaseLost)
	}
}

func TestWebhookStoreRetryAndTerminalTransitionsFenceLeaseAndClearSensitiveFields(t *testing.T) {
	store := newWebhookTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	items := []instagrammeta.WorkItem{
		syntheticWebhookItem(0x51, 0x61, "synthetic-retry-sender", now),
		syntheticWebhookItem(0x52, 0x62, "synthetic-ignore-sender", now),
		syntheticWebhookItem(0x53, 0x63, "synthetic-fail-sender", now),
	}
	if inserted, err := store.EnqueueWebhookWork(ctx, items, now); err != nil || inserted != len(items) {
		t.Fatalf("enqueue = (%d, %v)", inserted, err)
	}
	claimed, err := store.ClaimWebhookWork(ctx, len(items), now)
	if err != nil || len(claimed) != len(items) {
		t.Fatalf("claim = (%d, %v)", len(claimed), err)
	}
	bySender := make(map[string]WebhookWork, len(claimed))
	for _, item := range claimed {
		bySender[item.SenderIGSID] = item
	}

	retry := bySender["synthetic-retry-sender"]
	next := now.Add(time.Second)
	if err := store.RetryWebhookWork(ctx, retry.ID, retry.LeaseToken, next, now); err != nil {
		t.Fatalf("retry: %v", err)
	}
	assertWebhookWorkRow(t, store, retry.ID, WebhookWorkRetryable, true)
	if early, err := store.ClaimWebhookWork(ctx, 1, now); err != nil || len(early) != 0 {
		t.Fatalf("early retry claim = (%d, %v)", len(early), err)
	}
	reclaimed, err := store.ClaimWebhookWork(ctx, 1, next)
	if err != nil || len(reclaimed) != 1 || reclaimed[0].ID != retry.ID || reclaimed[0].Attempts != 2 {
		t.Fatalf("retry claim = (%+v, %v)", reclaimed, err)
	}
	if err := store.CompleteWebhookWork(ctx, reclaimed[0].ID, reclaimed[0].LeaseToken, next, WebhookReasonProcessed); err != nil {
		t.Fatalf("complete: %v", err)
	}
	assertWebhookWorkRow(t, store, retry.ID, WebhookWorkCompleted, false)

	ignored := bySender["synthetic-ignore-sender"]
	if err := store.IgnoreWebhookWork(ctx, ignored.ID, ignored.LeaseToken, now, WebhookReasonMembershipInactive); err != nil {
		t.Fatalf("ignore: %v", err)
	}
	assertWebhookWorkRow(t, store, ignored.ID, WebhookWorkIgnored, false)

	failed := bySender["synthetic-fail-sender"]
	if err := store.FailWebhookWork(ctx, failed.ID, failed.LeaseToken, now, WebhookReasonProviderPermanent); err != nil {
		t.Fatalf("fail: %v", err)
	}
	assertWebhookWorkRow(t, store, failed.ID, WebhookWorkFailed, false)
}

func TestWebhookStoreRateLimitedInvalidRedemptionIsTerminallyIgnored(t *testing.T) {
	store := newWebhookTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	item := syntheticWebhookItem(0x54, 0x64, "synthetic-limited-sender", now)
	if inserted, err := store.EnqueueWebhookWork(ctx, []instagrammeta.WorkItem{item}, now); err != nil || inserted != 1 {
		t.Fatalf("enqueue = (%d, %v)", inserted, err)
	}
	claimed, err := store.ClaimWebhookWork(ctx, 1, now)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim = (%d, %v)", len(claimed), err)
	}
	work := claimed[0]
	if err := store.IgnoreWebhookWork(ctx, work.ID, work.LeaseToken, now, WebhookReasonRateLimited); err != nil {
		t.Fatalf("ignore rate-limited redemption: %v", err)
	}
	assertWebhookWorkRow(t, store, work.ID, WebhookWorkIgnored, false)
}

func TestWebhookStoreTerminalizesAttemptAndAgeBoundariesBeforeClaim(t *testing.T) {
	store := newWebhookTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 19, 12, 15, 0, 0, time.UTC)
	items := []instagrammeta.WorkItem{
		syntheticWebhookItem(0x71, 0x81, "synthetic-attempt-bound", now),
		syntheticWebhookItem(0x72, 0x82, "synthetic-age-bound", now),
	}
	if _, err := store.EnqueueWebhookWork(ctx, items, now.Add(-WebhookMaxProcessingAge)); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if _, err := store.pool.Exec(ctx, `
		UPDATE instagram_webhook_work
		SET status = 'retryable', attempts = CASE
		        WHEN sender_igsid = 'synthetic-attempt-bound' THEN $1 ELSE 1 END,
		    processing_started_at = CASE
		        WHEN sender_igsid = 'synthetic-age-bound' THEN $2::timestamptz ELSE $3::timestamptz END,
		    next_attempt_at = $3
	`, WebhookMaxAttempts, now.Add(-WebhookMaxProcessingAge), now); err != nil {
		t.Fatalf("prepare boundaries: %v", err)
	}
	if claimed, err := store.ClaimWebhookWork(ctx, 2, now); err != nil || len(claimed) != 0 {
		t.Fatalf("claim terminal boundaries = (%d, %v), want empty", len(claimed), err)
	}

	rows, err := store.pool.Query(ctx, `
		SELECT status, terminal_reason, sender_igsid, challenge_digest
		FROM instagram_webhook_work ORDER BY terminal_reason
	`)
	if err != nil {
		t.Fatalf("inspect terminal rows: %v", err)
	}
	defer rows.Close()
	wantReasons := map[string]bool{string(WebhookReasonMaxAttempts): false, string(WebhookReasonMaxAge): false}
	for rows.Next() {
		var status WebhookWorkStatus
		var reason string
		var sender sql.NullString
		var challenge []byte
		if err := rows.Scan(&status, &reason, &sender, &challenge); err != nil {
			t.Fatalf("scan terminal row: %v", err)
		}
		if status != WebhookWorkFailed || sender.Valid || len(challenge) != 0 {
			t.Fatalf("terminal row retained sensitive fields: %q/%q/%t/%d", status, reason, sender.Valid, len(challenge))
		}
		if _, ok := wantReasons[reason]; !ok {
			t.Fatalf("unexpected terminal reason %q", reason)
		}
		wantReasons[reason] = true
	}
	for reason, seen := range wantReasons {
		if !seen {
			t.Errorf("terminal reason %q not found", reason)
		}
	}
}

func TestWebhookStoreConcurrentClaimsLeaseOneRowOnce(t *testing.T) {
	store := newWebhookTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	if _, err := store.EnqueueWebhookWork(ctx, []instagrammeta.WorkItem{
		syntheticWebhookItem(0x91, 0xa1, "synthetic-concurrent-sender", now),
	}, now); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	start := make(chan struct{})
	results := make(chan []WebhookWork, 2)
	errorsCh := make(chan error, 2)
	for range 2 {
		go func() {
			<-start
			claimed, err := store.ClaimWebhookWork(ctx, 1, now)
			results <- claimed
			errorsCh <- err
		}()
	}
	close(start)
	total := 0
	var lease uuid.UUID
	for range 2 {
		if err := <-errorsCh; err != nil {
			t.Fatalf("concurrent claim: %v", err)
		}
		claimed := <-results
		total += len(claimed)
		if len(claimed) == 1 {
			if lease != uuid.Nil && lease == claimed[0].LeaseToken {
				t.Fatal("concurrent claims reused a lease token")
			}
			lease = claimed[0].LeaseToken
		}
	}
	if total != 1 {
		t.Fatalf("total concurrent claims = %d, want 1", total)
	}
}

func TestSignedWebhookHandlerDurablyDeduplicatesBeforeAcknowledgement(t *testing.T) {
	store := newWebhookTestStore(t)
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	secret := []byte("synthetic-app-secret")
	digests, err := instagrammeta.NewDigestCodec(bytes.Repeat([]byte{0xd1}, 32), CanonicalizeChallenge)
	if err != nil {
		t.Fatalf("NewDigestCodec: %v", err)
	}
	reducer, err := instagrammeta.NewPayloadReducer("official", digests)
	if err != nil {
		t.Fatalf("NewPayloadReducer: %v", err)
	}
	handler, err := instagrammeta.NewWebhookHandler(instagrammeta.WebhookHandlerConfig{
		AppSecret:   secret,
		VerifyToken: "synthetic-verify-token",
		Reducer:     reducer,
		Sink:        store,
		Now:         func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("NewWebhookHandler: %v", err)
	}
	body := []byte(`{
  "object":"instagram",
  "entry":[{"id":"official","messaging":[{
    "sender":{"id":"sender"},"recipient":{"id":"official"},
    "timestamp":1721386800123,
    "message":{"mid":"synthetic-message","text":"CSKY-2345-6789-ABCD-E"}
  }]}]
}`)
	signature := signWebhookPayload(secret, body)
	for range 2 {
		request := httptest.NewRequest(http.MethodPost, "/integrations/instagram/webhook", bytes.NewReader(body))
		request.Header.Set("X-Hub-Signature-256", signature)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("webhook status = %d, body %q", response.Code, response.Body.String())
		}
	}
	var count int
	if err := store.pool.QueryRow(context.Background(), `SELECT count(*) FROM instagram_webhook_work`).Scan(&count); err != nil {
		t.Fatalf("count durable work: %v", err)
	}
	if count != 1 {
		t.Fatalf("durable work count = %d, want 1", count)
	}
}

func TestSignedWebhookHandlerAcknowledgesTerminalInvalidRedemptionExcess(t *testing.T) {
	store := newWebhookTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	secret := []byte("synthetic-app-secret")
	digests, err := instagrammeta.NewDigestCodec(bytes.Repeat([]byte{0xd2}, 32), CanonicalizeChallenge)
	if err != nil {
		t.Fatalf("NewDigestCodec: %v", err)
	}
	reducer, err := instagrammeta.NewPayloadReducer("official", digests)
	if err != nil {
		t.Fatalf("NewPayloadReducer: %v", err)
	}
	invalidLimiter := &sequenceInvalidRedemptionLimiter{allowed: []bool{true, true, false}}
	requestLimiter := &syntheticWebhookRequestLimiter{invalid: invalidLimiter}
	handler, err := instagrammeta.NewWebhookHandler(instagrammeta.WebhookHandlerConfig{
		AppSecret:   secret,
		VerifyToken: "synthetic-verify-token",
		Reducer:     reducer,
		Sink:        store,
		Limiter:     requestLimiter,
		Now:         func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("NewWebhookHandler: %v", err)
	}
	body := []byte(`{
  "object":"instagram",
  "entry":[{"id":"official","messaging":[
    {"sender":{"id":"sender-1"},"recipient":{"id":"official"},"timestamp":1721386800123,"message":{"mid":"message-1","text":"CSKY-2345-6789-ABCD-E"}},
    {"sender":{"id":"sender-2"},"recipient":{"id":"official"},"timestamp":1721386801123,"message":{"mid":"message-2","text":"CSKY-3456-789A-BCDE-F"}},
    {"sender":{"id":"sender-3"},"recipient":{"id":"official"},"timestamp":1721386802123,"message":{"mid":"message-3","text":"CSKY-4567-89AB-CDEF-G"}}
  ]}]
}`)
	for delivery := 1; delivery <= 2; delivery++ {
		request := httptest.NewRequest(http.MethodPost, "/integrations/instagram/webhook", bytes.NewReader(body))
		request.RemoteAddr = "198.51.100.7:4321"
		request.Header.Set("Forwarded", "for=203.0.113.9")
		request.Header.Set("X-Hub-Signature-256", signWebhookPayload(secret, body))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("delivery %d status = %d, body %q", delivery, response.Code, response.Body.String())
		}
	}
	if requestLimiter.sourceCalls != 2 {
		t.Fatalf("invalid source resolver calls = %d, want once per signed decoded delivery", requestLimiter.sourceCalls)
	}
	if invalidLimiter.calls != 3 {
		t.Fatalf("invalid limiter calls = %d, want unique events only", invalidLimiter.calls)
	}
	claimed, err := store.ClaimWebhookWork(ctx, 3, now)
	if err != nil {
		t.Fatalf("ClaimWebhookWork: %v", err)
	}
	if len(claimed) != 2 {
		t.Fatalf("claimable work = %d, want two allowed invalid events", len(claimed))
	}
	var (
		ignored   int
		sensitive int
	)
	if err := store.pool.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE status = 'ignored' AND terminal_reason = 'rateLimited'),
		       count(*) FILTER (WHERE status = 'ignored' AND (
		           sender_igsid IS NOT NULL OR official_account_id IS NOT NULL OR
		           challenge_digest_version IS NOT NULL OR challenge_digest IS NOT NULL
		       ))
		FROM instagram_webhook_work
	`).Scan(&ignored, &sensitive); err != nil {
		t.Fatalf("inspect terminal invalid webhook: %v", err)
	}
	if ignored != 1 || sensitive != 0 {
		t.Fatalf("terminal invalid webhook = ignored %d sensitive %d, want 1/0", ignored, sensitive)
	}
}

func signWebhookPayload(secret, body []byte) string {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func assertWebhookWorkRow(t *testing.T, store *WebhookStore, id uuid.UUID, wantStatus WebhookWorkStatus, wantSensitive bool) {
	t.Helper()
	var (
		status           WebhookWorkStatus
		sender           sql.NullString
		challenge        []byte
		challengeVersion sql.NullInt16
		leaseToken       sql.NullString
	)
	if err := store.pool.QueryRow(context.Background(), `
		SELECT status, sender_igsid, challenge_digest_version,
		       challenge_digest, lease_token::text
		FROM instagram_webhook_work WHERE id = $1
	`, id).Scan(&status, &sender, &challengeVersion, &challenge, &leaseToken); err != nil {
		t.Fatalf("inspect work %s: %v", id, err)
	}
	if status != wantStatus {
		t.Fatalf("status = %q, want %q", status, wantStatus)
	}
	gotSensitive := sender.Valid || challengeVersion.Valid || len(challenge) > 0
	if gotSensitive != wantSensitive {
		t.Fatalf("sensitive fields present = %t, want %t", gotSensitive, wantSensitive)
	}
	if leaseToken.Valid {
		t.Fatalf("terminal/retry row retained lease token %s", leaseToken.String)
	}
}

func newWebhookTestStore(t *testing.T) *WebhookStore {
	t.Helper()
	migration, err := os.ReadFile("../../migrations/000023_instagram_migration.up.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	return NewWebhookStore(testdb.WithSchema(t, "CREATE TABLE craftsky_profiles (did TEXT PRIMARY KEY);\n"+string(migration)))
}

type sequenceInvalidRedemptionLimiter struct {
	allowed []bool
	calls   int
	errAt   int
	err     error
}

func (l *sequenceInvalidRedemptionLimiter) AllowInvalidRedemption(context.Context) (instagrammeta.WebhookLimitDecision, error) {
	l.calls++
	if l.errAt > 0 && l.calls == l.errAt {
		return instagrammeta.WebhookLimitDecision{}, l.err
	}
	allowed := true
	if l.calls <= len(l.allowed) {
		allowed = l.allowed[l.calls-1]
	}
	return instagrammeta.WebhookLimitDecision{Allowed: allowed}, nil
}

type syntheticWebhookRequestLimiter struct {
	invalid     instagrammeta.WebhookInvalidRedemptionLimiter
	sourceCalls int
}

func (*syntheticWebhookRequestLimiter) AllowSourceIP(context.Context, *http.Request) (instagrammeta.WebhookLimitDecision, error) {
	return instagrammeta.WebhookLimitDecision{Allowed: true}, nil
}

func (*syntheticWebhookRequestLimiter) AllowGlobal(context.Context) (instagrammeta.WebhookLimitDecision, error) {
	return instagrammeta.WebhookLimitDecision{Allowed: true}, nil
}

func (l *syntheticWebhookRequestLimiter) InvalidRedemptionSourceIP(*http.Request) (instagrammeta.WebhookInvalidRedemptionLimiter, error) {
	l.sourceCalls++
	return l.invalid, nil
}

func syntheticWebhookItem(messageByte, challengeByte byte, sender string, eventAt time.Time) instagrammeta.WorkItem {
	message := instagrammeta.KeyedDigest{Version: instagrammeta.DigestVersion}
	challenge := instagrammeta.KeyedDigest{Version: instagrammeta.DigestVersion}
	copy(message.Value[:], bytes.Repeat([]byte{messageByte}, len(message.Value)))
	copy(challenge.Value[:], bytes.Repeat([]byte{challengeByte}, len(challenge.Value)))
	return instagrammeta.WorkItem{
		MessageIDDigest:   message,
		SenderIGSID:       sender,
		OfficialAccountID: "synthetic-official-account",
		ChallengeDigest:   challenge,
		EventAt:           eventAt,
	}
}
