package instagram

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/integrations/instagrammeta"
)

func TestVerificationWebhookRedeemerSurvivesCrashAfterDigestClear(t *testing.T) {
	verification, queue, redeemer := newWebhookRedemptionTestStores(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	digest := syntheticChallengeDigest(0x71)
	attempt := createWebhookRedemptionAttempt(t, verification, "00000000-0000-0000-0000-000000000071", "did:plc:synthetic-redemption", digest, now)
	work := enqueueAndClaimWebhookRedemptionWork(t, queue, "00000000-0000-0000-0000-000000000171", digest, "synthetic-igsid-71", now)

	first, err := redeemer.RedeemWebhookChallenge(ctx, WebhookRedemptionRequest{
		WorkID:          work.ID,
		LeaseToken:      work.LeaseToken,
		ChallengeDigest: digest,
		SenderIGSID:     work.SenderIGSID,
		Now:             now.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("first redemption: %v", err)
	}
	if first.AttemptID != attempt.ID || first.OwnerDID != attempt.OwnerDID {
		t.Fatalf("first redemption = %+v", first)
	}

	var mappedAttempt uuid.UUID
	var attemptDigestCount int
	if err := verification.pool.QueryRow(ctx, `
		SELECT w.verification_attempt_id, count(a.challenge_digest)
		FROM instagram_webhook_work w
		JOIN instagram_verification_attempts a ON a.id = w.verification_attempt_id
		WHERE w.id = $1
		GROUP BY w.verification_attempt_id
	`, work.ID).Scan(&mappedAttempt, &attemptDigestCount); err != nil {
		t.Fatalf("inspect durable redemption: %v", err)
	}
	if mappedAttempt != attempt.ID || attemptDigestCount != 0 {
		t.Fatalf("durable redemption mapped %s with digest count %d", mappedAttempt, attemptDigestCount)
	}

	// Simulate a crash after challenge redemption but before durable work
	// completion. Recovery receives a new lease and no challenge digest remains
	// on the verification attempt.
	retryAt := now.Add(3 * time.Second)
	if err := queue.RetryWebhookWork(ctx, work.ID, work.LeaseToken, retryAt, now.Add(2*time.Second)); err != nil {
		t.Fatalf("schedule recovery: %v", err)
	}
	reclaimed, err := queue.ClaimWebhookWork(ctx, 1, retryAt)
	if err != nil || len(reclaimed) != 1 {
		t.Fatalf("reclaim work = %d, %v", len(reclaimed), err)
	}
	second, err := redeemer.RedeemWebhookChallenge(ctx, WebhookRedemptionRequest{
		WorkID:          reclaimed[0].ID,
		LeaseToken:      reclaimed[0].LeaseToken,
		ChallengeDigest: digest,
		SenderIGSID:     reclaimed[0].SenderIGSID,
		Now:             retryAt,
	})
	if err != nil {
		t.Fatalf("replay redemption: %v", err)
	}
	if second != first {
		t.Fatalf("replay redemption = %+v, want %+v", second, first)
	}

	if err := redeemer.SetWebhookCandidate(ctx, attempt.ID, "Synthetic.Candidate", retryAt); err != nil {
		t.Fatalf("set candidate: %v", err)
	}
	if err := redeemer.SetWebhookCandidate(ctx, attempt.ID, "synthetic.candidate", retryAt.Add(time.Second)); err != nil {
		t.Fatalf("idempotent candidate replay: %v", err)
	}
	if err := redeemer.SetWebhookCandidate(ctx, attempt.ID, "different.candidate", retryAt.Add(time.Second)); !errors.Is(err, ErrInstagramStateTransition) {
		t.Fatalf("candidate rebind error = %v, want state transition", err)
	}
}

func TestVerificationWebhookRedeemerConcurrentReplayCannotRebindWork(t *testing.T) {
	verification, queue, redeemer := newWebhookRedemptionTestStores(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 19, 13, 0, 0, 0, time.UTC)
	digest := syntheticChallengeDigest(0x72)
	attempt := createWebhookRedemptionAttempt(t, verification, "00000000-0000-0000-0000-000000000072", "did:plc:synthetic-concurrent", digest, now)
	work := enqueueAndClaimWebhookRedemptionWork(t, queue, "00000000-0000-0000-0000-000000000172", digest, "synthetic-igsid-72", now)
	request := WebhookRedemptionRequest{
		WorkID:          work.ID,
		LeaseToken:      work.LeaseToken,
		ChallengeDigest: digest,
		SenderIGSID:     work.SenderIGSID,
		Now:             now.Add(time.Second),
	}

	const callers = 12
	start := make(chan struct{})
	results := make(chan WebhookRedemption, callers)
	errs := make(chan error, callers)
	var group sync.WaitGroup
	for range callers {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			result, err := redeemer.RedeemWebhookChallenge(ctx, request)
			results <- result
			errs <- err
		}()
	}
	close(start)
	group.Wait()
	close(results)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent replay: %v", err)
		}
	}
	for result := range results {
		if result.AttemptID != attempt.ID || result.OwnerDID != attempt.OwnerDID {
			t.Fatalf("concurrent result = %+v", result)
		}
	}

	otherDigest := syntheticChallengeDigest(0x73)
	createWebhookRedemptionAttempt(t, verification, "00000000-0000-0000-0000-000000000073", "did:plc:synthetic-other", otherDigest, now)
	request.ChallengeDigest = otherDigest
	if _, err := redeemer.RedeemWebhookChallenge(ctx, request); !errors.Is(err, ErrInstagramResourceNotFound) {
		t.Fatalf("work rebind error = %v, want privacy-safe not found", err)
	}
	var stillMapped uuid.UUID
	if err := verification.pool.QueryRow(ctx, `
		SELECT verification_attempt_id FROM instagram_webhook_work WHERE id = $1
	`, work.ID).Scan(&stillMapped); err != nil {
		t.Fatalf("read stable mapping: %v", err)
	}
	if stillMapped != attempt.ID {
		t.Fatalf("work rebound to %s, want %s", stillMapped, attempt.ID)
	}
}

func TestVerificationWebhookRedeemerAllowsOnlyOneWorkForAChallenge(t *testing.T) {
	verification, queue, redeemer := newWebhookRedemptionTestStores(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 19, 14, 0, 0, 0, time.UTC)
	digest := syntheticChallengeDigest(0x74)
	attempt := createWebhookRedemptionAttempt(t, verification, "00000000-0000-0000-0000-000000000074", "did:plc:synthetic-one-work", digest, now)

	items := []instagrammeta.WorkItem{
		webhookRedemptionWorkItem("00000000-0000-0000-0000-000000000174", digest, "synthetic-igsid-74", now),
		webhookRedemptionWorkItem("00000000-0000-0000-0000-000000000175", digest, "synthetic-igsid-74", now),
	}
	if inserted, err := queue.EnqueueWebhookWork(ctx, items, now); err != nil || inserted != 2 {
		t.Fatalf("enqueue duplicate challenge deliveries = %d, %v", inserted, err)
	}
	claimed, err := queue.ClaimWebhookWork(ctx, 2, now)
	if err != nil || len(claimed) != 2 {
		t.Fatalf("claim duplicate challenge deliveries = %d, %v", len(claimed), err)
	}

	start := make(chan struct{})
	errs := make(chan error, 2)
	var group sync.WaitGroup
	for _, work := range claimed {
		work := work
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			_, err := redeemer.RedeemWebhookChallenge(ctx, WebhookRedemptionRequest{
				WorkID:          work.ID,
				LeaseToken:      work.LeaseToken,
				ChallengeDigest: digest,
				SenderIGSID:     work.SenderIGSID,
				Now:             now.Add(time.Second),
			})
			errs <- err
		}()
	}
	close(start)
	group.Wait()
	close(errs)
	successes := 0
	notFound := 0
	for err := range errs {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrInstagramResourceNotFound):
			notFound++
		default:
			t.Fatalf("concurrent distinct work redemption: %v", err)
		}
	}
	if successes != 1 || notFound != 1 {
		t.Fatalf("redemption outcomes success=%d notFound=%d", successes, notFound)
	}
	var mappings int
	if err := verification.pool.QueryRow(ctx, `
		SELECT count(*) FROM instagram_webhook_work WHERE verification_attempt_id = $1
	`, attempt.ID).Scan(&mappings); err != nil {
		t.Fatalf("count mappings: %v", err)
	}
	if mappings != 1 {
		t.Fatalf("attempt mappings = %d, want one", mappings)
	}
}

func TestVerificationWebhookRedeemerDrivesWorkerToPendingConfirmation(t *testing.T) {
	verification, queue, redeemer := newWebhookRedemptionTestStores(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 19, 15, 0, 0, 0, time.UTC)
	digest := syntheticChallengeDigest(0x75)
	attempt := createWebhookRedemptionAttempt(t, verification, "00000000-0000-0000-0000-000000000075", "did:plc:synthetic-worker-adapter", digest, now)
	item := webhookRedemptionWorkItem("00000000-0000-0000-0000-000000000176", digest, "synthetic-igsid-75", now)
	if inserted, err := queue.EnqueueWebhookWork(ctx, []instagrammeta.WorkItem{item}, now); err != nil || inserted != 1 {
		t.Fatalf("enqueue worker item = %d, %v", inserted, err)
	}
	events := make([]string, 0, 4)
	worker, err := NewWebhookWorker(
		queue,
		redeemer,
		&fakeWebhookMembership{current: true, events: &events},
		&fakeMetaClient{username: "Synthetic.Worker", events: &events},
		WebhookWorkerOptions{Now: func() time.Time { return now.Add(time.Second) }},
	)
	if err != nil {
		t.Fatalf("new worker: %v", err)
	}
	processed, err := worker.ProcessBatch(ctx)
	if err != nil || processed != 1 {
		t.Fatalf("process batch = %d, %v", processed, err)
	}
	got, err := verification.GetVerificationAttempt(ctx, attempt.OwnerDID, attempt.ID, now.Add(time.Second))
	if err != nil {
		t.Fatalf("get transitioned attempt: %v", err)
	}
	if got.State != AttemptPendingConfirmation || got.CandidateIGSID != item.SenderIGSID || got.CandidateUsername != "synthetic.worker" {
		t.Fatalf("transitioned attempt = %+v", got)
	}
	var (
		status        WebhookWorkStatus
		mappedAttempt uuid.UUID
		sender        sql.NullString
	)
	if err := verification.pool.QueryRow(ctx, `
		SELECT status, verification_attempt_id, sender_igsid
		FROM instagram_webhook_work
	`).Scan(&status, &mappedAttempt, &sender); err != nil {
		t.Fatalf("inspect completed work: %v", err)
	}
	if status != WebhookWorkCompleted || mappedAttempt != attempt.ID || sender.Valid {
		t.Fatalf("completed work status=%q mapped=%s senderValid=%t", status, mappedAttempt, sender.Valid)
	}
}

func TestVerificationWebhookRedeemerTerminalTransitionsAreOwnedAndIdempotent(t *testing.T) {
	verification, queue, redeemer := newWebhookRedemptionTestStores(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 19, 16, 0, 0, 0, time.UTC)

	providerDigest := syntheticChallengeDigest(0x76)
	providerAttempt := createWebhookRedemptionAttempt(t, verification, "00000000-0000-0000-0000-000000000076", "did:plc:synthetic-provider-reject", providerDigest, now)
	providerWork := enqueueAndClaimWebhookRedemptionWork(t, queue, "00000000-0000-0000-0000-000000000177", providerDigest, "synthetic-igsid-76", now)
	if _, err := redeemer.RedeemWebhookChallenge(ctx, WebhookRedemptionRequest{
		WorkID: providerWork.ID, LeaseToken: providerWork.LeaseToken,
		ChallengeDigest: providerDigest, SenderIGSID: providerWork.SenderIGSID, Now: now.Add(time.Second),
	}); err != nil {
		t.Fatalf("redeem provider attempt: %v", err)
	}
	if err := redeemer.RejectWebhookAttempt(ctx, providerAttempt.ID, RetryInvalidProfileResponse, now.Add(2*time.Second)); err != nil {
		t.Fatalf("reject provider attempt: %v", err)
	}
	if err := redeemer.RejectWebhookAttempt(ctx, providerAttempt.ID, RetryInvalidProfileResponse, now.Add(3*time.Second)); err != nil {
		t.Fatalf("replay provider rejection: %v", err)
	}
	if err := redeemer.RejectWebhookAttempt(ctx, providerAttempt.ID, RetryProfileLookupUnavailable, now.Add(3*time.Second)); !errors.Is(err, ErrInstagramStateTransition) {
		t.Fatalf("provider rejection rebind error = %v", err)
	}

	membershipDigest := syntheticChallengeDigest(0x77)
	membershipAttempt := createWebhookRedemptionAttempt(t, verification, "00000000-0000-0000-0000-000000000077", "did:plc:synthetic-membership-reject", membershipDigest, now)
	membershipWork := enqueueAndClaimWebhookRedemptionWork(t, queue, "00000000-0000-0000-0000-000000000178", membershipDigest, "synthetic-igsid-77", now)
	if _, err := redeemer.RedeemWebhookChallenge(ctx, WebhookRedemptionRequest{
		WorkID: membershipWork.ID, LeaseToken: membershipWork.LeaseToken,
		ChallengeDigest: membershipDigest, SenderIGSID: membershipWork.SenderIGSID, Now: now.Add(time.Second),
	}); err != nil {
		t.Fatalf("redeem membership attempt: %v", err)
	}
	if err := redeemer.InactivateWebhookOwner(ctx, membershipAttempt.ID, syntax.DID("did:plc:synthetic-wrong-owner"), now.Add(2*time.Second)); !errors.Is(err, ErrInstagramResourceNotFound) {
		t.Fatalf("foreign inactivation error = %v", err)
	}
	if err := redeemer.InactivateWebhookOwner(ctx, membershipAttempt.ID, membershipAttempt.OwnerDID, now.Add(2*time.Second)); err != nil {
		t.Fatalf("inactivate owner: %v", err)
	}

	for _, test := range []struct {
		id   uuid.UUID
		code AttemptRetryCode
	}{
		{id: providerAttempt.ID, code: RetryInvalidProfileResponse},
		{id: membershipAttempt.ID, code: RetryMembershipInactive},
	} {
		var (
			state             VerificationAttemptState
			code              AttemptRetryCode
			candidateIGSID    sql.NullString
			candidateUsername sql.NullString
		)
		if err := verification.pool.QueryRow(ctx, `
			SELECT state, retry_code, candidate_igsid, candidate_username
			FROM instagram_verification_attempts WHERE id = $1
		`, test.id).Scan(&state, &code, &candidateIGSID, &candidateUsername); err != nil {
			t.Fatalf("inspect rejected attempt: %v", err)
		}
		if state != AttemptRejected || code != test.code || candidateIGSID.Valid || candidateUsername.Valid {
			t.Fatalf("rejected attempt state=%q code=%q igsid=%t username=%t", state, code, candidateIGSID.Valid, candidateUsername.Valid)
		}
	}
}

func TestVerificationWebhookRedeemerRejectsExpiredAttemptAndStaleLease(t *testing.T) {
	t.Run("attempt expiry is durable", func(t *testing.T) {
		verification, queue, redeemer := newWebhookRedemptionTestStores(t)
		ctx := context.Background()
		now := time.Date(2026, 7, 19, 17, 0, 0, 0, time.UTC)
		digest := syntheticChallengeDigest(0x78)
		attempt, err := verification.CreateVerificationAttempt(ctx, CreateVerificationAttemptParams{
			ID:        uuid.MustParse("00000000-0000-0000-0000-000000000078"),
			OwnerDID:  syntax.DID("did:plc:synthetic-expired-redemption"),
			Digest:    digest,
			ExpiresAt: now.Add(30 * time.Second),
			Now:       now,
		})
		if err != nil {
			t.Fatalf("create short attempt: %v", err)
		}
		work := enqueueAndClaimWebhookRedemptionWork(t, queue, "00000000-0000-0000-0000-000000000179", digest, "synthetic-igsid-78", now)
		if _, err := redeemer.RedeemWebhookChallenge(ctx, WebhookRedemptionRequest{
			WorkID: work.ID, LeaseToken: work.LeaseToken,
			ChallengeDigest: digest, SenderIGSID: work.SenderIGSID, Now: attempt.ExpiresAt,
		}); !errors.Is(err, ErrInstagramResourceNotFound) {
			t.Fatalf("expired redemption error = %v", err)
		}
		var (
			state      VerificationAttemptState
			digestRows int
			mapped     uuid.NullUUID
		)
		if err := verification.pool.QueryRow(ctx, `
			SELECT a.state, count(a.challenge_digest), w.verification_attempt_id
			FROM instagram_verification_attempts a
			JOIN instagram_webhook_work w ON w.id = $2
			WHERE a.id = $1
			GROUP BY a.state, w.verification_attempt_id
		`, attempt.ID, work.ID).Scan(&state, &digestRows, &mapped); err != nil {
			t.Fatalf("inspect expired redemption: %v", err)
		}
		if state != AttemptExpired || digestRows != 0 || mapped.Valid {
			t.Fatalf("expired redemption state=%q digests=%d mapped=%t", state, digestRows, mapped.Valid)
		}
	})

	t.Run("stale lease cannot bind", func(t *testing.T) {
		verification, queue, redeemer := newWebhookRedemptionTestStores(t)
		ctx := context.Background()
		now := time.Date(2026, 7, 19, 18, 0, 0, 0, time.UTC)
		digest := syntheticChallengeDigest(0x79)
		attempt := createWebhookRedemptionAttempt(t, verification, "00000000-0000-0000-0000-000000000079", "did:plc:synthetic-stale-lease", digest, now)
		work := enqueueAndClaimWebhookRedemptionWork(t, queue, "00000000-0000-0000-0000-000000000180", digest, "synthetic-igsid-79", now)
		if _, err := redeemer.RedeemWebhookChallenge(ctx, WebhookRedemptionRequest{
			WorkID: work.ID, LeaseToken: work.LeaseToken,
			ChallengeDigest: digest, SenderIGSID: work.SenderIGSID, Now: work.LeaseExpiresAt,
		}); !errors.Is(err, ErrInstagramResourceNotFound) {
			t.Fatalf("stale lease redemption error = %v", err)
		}
		var (
			state  VerificationAttemptState
			mapped uuid.NullUUID
		)
		if err := verification.pool.QueryRow(ctx, `
			SELECT a.state, w.verification_attempt_id
			FROM instagram_verification_attempts a
			JOIN instagram_webhook_work w ON w.id = $2
			WHERE a.id = $1
		`, attempt.ID, work.ID).Scan(&state, &mapped); err != nil {
			t.Fatalf("inspect stale lease: %v", err)
		}
		if state != AttemptPendingDM || mapped.Valid {
			t.Fatalf("stale lease changed state=%q mapped=%t", state, mapped.Valid)
		}
	})
}

func newWebhookRedemptionTestStores(t *testing.T) (*VerificationStore, *WebhookStore, *VerificationWebhookRedeemer) {
	t.Helper()
	verification := newVerificationTestStore(t)
	queue := NewWebhookStore(verification.pool)
	redeemer, err := NewVerificationWebhookRedeemer(verification)
	if err != nil {
		t.Fatalf("new webhook redeemer: %v", err)
	}
	return verification, queue, redeemer
}

func createWebhookRedemptionAttempt(t *testing.T, store *VerificationStore, id, owner string, digest ChallengeDigest, now time.Time) *VerificationAttempt {
	t.Helper()
	attempt, err := store.CreateVerificationAttempt(context.Background(), CreateVerificationAttemptParams{
		ID:        uuid.MustParse(id),
		OwnerDID:  syntax.DID(owner),
		Digest:    digest,
		ExpiresAt: now.Add(10 * time.Minute),
		Now:       now,
	})
	if err != nil {
		t.Fatalf("create verification attempt: %v", err)
	}
	return attempt
}

func enqueueAndClaimWebhookRedemptionWork(t *testing.T, store *WebhookStore, messageID string, digest ChallengeDigest, sender string, now time.Time) WebhookWork {
	t.Helper()
	item := webhookRedemptionWorkItem(messageID, digest, sender, now)
	if inserted, err := store.EnqueueWebhookWork(context.Background(), []instagrammeta.WorkItem{item}, now); err != nil || inserted != 1 {
		t.Fatalf("enqueue webhook work = %d, %v", inserted, err)
	}
	claimed, err := store.ClaimWebhookWork(context.Background(), 1, now)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim webhook work = %d, %v", len(claimed), err)
	}
	return claimed[0]
}

func webhookRedemptionWorkItem(messageID string, digest ChallengeDigest, sender string, now time.Time) instagrammeta.WorkItem {
	messageDigest := instagrammeta.KeyedDigest{Version: instagrammeta.DigestVersion}
	parsedMessageID := uuid.MustParse(messageID)
	copy(messageDigest.Value[:], parsedMessageID[:])
	// Fill the remainder so the persistence validator never accepts an all-zero
	// synthetic message digest.
	copy(messageDigest.Value[16:], messageDigest.Value[:16])
	challengeDigest := instagrammeta.KeyedDigest{Version: digest.Version, Value: digest.Value}
	return instagrammeta.WorkItem{
		MessageIDDigest:   messageDigest,
		SenderIGSID:       sender,
		OfficialAccountID: "synthetic-official-account",
		ChallengeDigest:   challengeDigest,
		EventAt:           now,
	}
}
