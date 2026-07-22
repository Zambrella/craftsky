package instagram

import (
	"bytes"
	"context"
	"errors"
	"net/url"
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

func TestVerificationServiceCreatesRedeemsConfirmsAndReplays(t *testing.T) {
	pool := verificationServicePool(t)
	store := NewVerificationStore(pool)
	codec, err := NewChallengeCodec(bytes.NewReader(bytes.Repeat([]byte{0x01}, 128)), bytes.Repeat([]byte{0x71}, 32))
	if err != nil {
		t.Fatalf("codec: %v", err)
	}
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	ids := sequentialUUIDs(0x10)
	dmURL, _ := url.Parse("https://www.instagram.com/direct/t/synthetic")
	service, err := NewVerificationService(VerificationServiceOptions{
		Store: store, Codec: codec, Now: func() time.Time { return now },
		NewID: ids, TTL: 10 * time.Minute, DMURL: dmURL,
		HMACKey: bytes.Repeat([]byte{0x71}, 32), Available: true,
	})
	if err != nil {
		t.Fatalf("service: %v", err)
	}
	owner := syntax.DID("did:plc:synthetic-alice")

	created, err := service.CreateVerification(context.Background(), owner)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.Attempt.State != AttemptPendingDM || !created.Attempt.ExpiresAt.Equal(now.Add(10*time.Minute)) || created.Challenge == "" {
		t.Fatalf("created state=%q expiry=%s challengeEmpty=%t", created.Attempt.State, created.Attempt.ExpiresAt, created.Challenge == "")
	}
	digest, err := codec.Digest(created.Challenge)
	if err != nil {
		t.Fatalf("digest: %v", err)
	}
	if _, err := store.RedeemVerificationChallenge(context.Background(), digest, "100000000000001", now.Add(time.Minute)); err != nil {
		t.Fatalf("redeem: %v", err)
	}
	if err := store.SetVerificationCandidate(context.Background(), created.Attempt.ID, "Synthetic.Candidate", now.Add(2*time.Minute)); err != nil {
		t.Fatalf("candidate: %v", err)
	}
	now = now.Add(3 * time.Minute)
	confirmed, err := service.ConfirmVerification(context.Background(), owner, created.Attempt.ID, true)
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if confirmed.State != AttemptConfirmed || confirmed.Account.State != LinkActive || confirmed.Account.Username != "synthetic.candidate" || !confirmed.Account.Discoverable {
		t.Fatalf("confirmed = %+v", confirmed)
	}
	replayed, err := service.ConfirmVerification(context.Background(), owner, created.Attempt.ID, true)
	if err != nil {
		t.Fatalf("confirm replay: %v", err)
	}
	if replayed != confirmed {
		t.Fatalf("replay = %+v, want %+v", replayed, confirmed)
	}

	var challengeCount, candidateCount int
	if err := pool.QueryRow(context.Background(), `
		SELECT count(challenge_digest), count(candidate_igsid)
		FROM instagram_verification_attempts WHERE id = $1
	`, created.Attempt.ID).Scan(&challengeCount, &candidateCount); err != nil {
		t.Fatalf("inspect terminal attempt: %v", err)
	}
	if challengeCount != 0 || candidateCount != 0 {
		t.Fatalf("terminal attempt retained challenge=%d candidate=%d", challengeCount, candidateCount)
	}
	var reconciliationJobs int
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*) FROM instagram_reconciliation_jobs
		WHERE owner_did=$1 AND reason='instagramLinkConfirmed' AND status='queued'
	`, owner).Scan(&reconciliationJobs); err != nil {
		t.Fatalf("inspect confirmed-link reconciliation: %v", err)
	}
	if reconciliationJobs != 1 {
		t.Fatalf("confirmed-link reconciliation jobs = %d, want 1", reconciliationJobs)
	}

	// A later DM proves the same stable IGSID again after an Instagram rename.
	// This is the validated refresh path: it changes the current username,
	// retires old-handle work, and queues only the new-handle reconciliation.
	now = now.Add(time.Minute)
	refresh, err := service.CreateVerification(context.Background(), owner)
	if err != nil {
		t.Fatalf("create refresh: %v", err)
	}
	refreshDigest, err := codec.Digest(refresh.Challenge)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.RedeemVerificationChallenge(context.Background(), refreshDigest, "100000000000001", now.Add(time.Minute)); err != nil {
		t.Fatalf("redeem refresh: %v", err)
	}
	if err := store.SetVerificationCandidate(context.Background(), refresh.Attempt.ID, "Renamed.Candidate", now.Add(2*time.Minute)); err != nil {
		t.Fatalf("candidate refresh: %v", err)
	}
	now = now.Add(3 * time.Minute)
	refreshed, err := service.ConfirmVerification(context.Background(), owner, refresh.Attempt.ID, true)
	if err != nil {
		t.Fatalf("confirm refresh: %v", err)
	}
	if refreshed.Account.Username != "renamed.candidate" || !refreshed.Account.Discoverable {
		t.Fatalf("refreshed account=%+v", refreshed.Account)
	}
	var refreshedJobs, ignoredConfirmedJobs int
	if err := pool.QueryRow(context.Background(), `
		SELECT
			count(*) FILTER (WHERE reason='instagramUsernameRefreshed' AND status='queued'),
			count(*) FILTER (WHERE reason='instagramLinkConfirmed' AND status='ignored')
		FROM instagram_reconciliation_jobs WHERE owner_did=$1
	`, owner).Scan(&refreshedJobs, &ignoredConfirmedJobs); err != nil {
		t.Fatal(err)
	}
	if refreshedJobs != 1 || ignoredConfirmedJobs != 1 {
		t.Fatalf("refreshed jobs=%d ignored confirmed jobs=%d", refreshedJobs, ignoredConfirmedJobs)
	}
}

func TestVerificationServiceCollisionDoesNotTransferExistingIdentity(t *testing.T) {
	pool := verificationServicePool(t)
	store := NewVerificationStore(pool)
	key := bytes.Repeat([]byte{0x72}, 32)
	codec, err := NewChallengeCodec(bytes.NewReader(bytes.Repeat([]byte{0x02}, 256)), key)
	if err != nil {
		t.Fatalf("codec: %v", err)
	}
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	dmURL, _ := url.Parse("https://www.instagram.com/direct/t/synthetic")
	service, err := NewVerificationService(VerificationServiceOptions{
		Store: store, Codec: codec, Now: func() time.Time { return now },
		NewID: sequentialUUIDs(0x40), TTL: 10 * time.Minute, DMURL: dmURL,
		HMACKey: key, Available: true,
	})
	if err != nil {
		t.Fatalf("service: %v", err)
	}
	ctx := context.Background()
	alice := syntax.DID("did:plc:synthetic-alice")
	bob := syntax.DID("did:plc:synthetic-bob")

	verifyCandidate := func(owner syntax.DID, username string) uuid.UUID {
		t.Helper()
		created, err := service.CreateVerification(ctx, owner)
		if err != nil {
			t.Fatalf("create %s: %v", owner, err)
		}
		digest, _ := codec.Digest(created.Challenge)
		if _, err := store.RedeemVerificationChallenge(ctx, digest, "100000000000099", now.Add(time.Minute)); err != nil {
			t.Fatalf("redeem %s: %v", owner, err)
		}
		if err := store.SetVerificationCandidate(ctx, created.Attempt.ID, username, now.Add(2*time.Minute)); err != nil {
			t.Fatalf("candidate %s: %v", owner, err)
		}
		return created.Attempt.ID
	}

	aliceAttempt := verifyCandidate(alice, "alice.crafts")
	now = now.Add(3 * time.Minute)
	if _, err := service.ConfirmVerification(ctx, alice, aliceAttempt, true); err != nil {
		t.Fatalf("confirm Alice: %v", err)
	}
	var aliceLinkID uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT id FROM instagram_account_links WHERE owner_did=$1 AND state='active'`, alice).Scan(&aliceLinkID); err != nil {
		t.Fatalf("read Alice link: %v", err)
	}
	dependentSuggestionID := uuid.MustParse("00000000-0000-0000-0000-00000000004e")
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_follow_suggestions(
			id,importer_did,target_did,state,reason,accepting_since,created_at,updated_at
		) VALUES($1,'did:plc:synthetic-importer',$2,'accepting','verifiedInstagramFollow',$3,$3,$3)
	`, dependentSuggestionID, alice, now); err != nil {
		t.Fatalf("seed collision suggestion: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO pds_follow_operations(
			id,suggestion_id,owner_did,target_did,rkey,status,attempt_count,created_at,updated_at
		) VALUES(
			'00000000-0000-0000-0000-00000000004f',$1,
			'did:plc:synthetic-importer',$2,'3kycollision','writing',1,$3,$3
		)
	`, dependentSuggestionID, alice, now); err != nil {
		t.Fatalf("seed collision follow operation: %v", err)
	}
	seedLifecycleNotification(
		t,
		pool,
		uuid.MustParse("00000000-0000-0000-0000-000000000050"),
		syntax.DID("did:plc:synthetic-importer"),
		dependentSuggestionID,
		"00000000-0000-0000-0000-000000000051",
		"leased",
		now,
	)
	bobAttempt := verifyCandidate(bob, "alice.crafts")
	now = now.Add(3 * time.Minute)
	if _, err := service.ConfirmVerification(ctx, bob, bobAttempt, true); err != ErrInstagramLinkConflict {
		t.Fatalf("confirm Bob error = %v, want link conflict", err)
	}

	var owner, state string
	var discoverable, conflictPending bool
	igsidDigest := digestPrivateIdentifier(key, "igsid", "100000000000099")
	if err := pool.QueryRow(ctx, `
		SELECT owner_did, state, discoverable, conflict_pending
		FROM instagram_account_links
		WHERE igsid_digest = $1
	`, igsidDigest.Value[:]).Scan(&owner, &state, &discoverable, &conflictPending); err != nil {
		t.Fatalf("inspect authoritative link: %v", err)
	}
	if owner != alice.String() || state != "active" || discoverable || !conflictPending {
		t.Fatalf("authoritative link owner=%s state=%s discoverable=%t conflict=%t", owner, state, discoverable, conflictPending)
	}
	var bobLinks, conflicts int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM instagram_account_links WHERE owner_did = $1`, bob).Scan(&bobLinks); err != nil {
		t.Fatalf("count Bob links: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM instagram_link_conflicts WHERE state = 'open'`).Scan(&conflicts); err != nil {
		t.Fatalf("count conflicts: %v", err)
	}
	if bobLinks != 0 || conflicts != 1 {
		t.Fatalf("Bob links=%d conflicts=%d", bobLinks, conflicts)
	}
	var suggestionState InstagramSuggestionState
	var followStatus, eventState, deliveryStatus, jobStatus string
	if err := pool.QueryRow(ctx, `
		SELECT
			(SELECT state FROM instagram_follow_suggestions WHERE id=$1),
			(SELECT status FROM pds_follow_operations WHERE suggestion_id=$1),
			(SELECT state FROM notification_events WHERE id='00000000-0000-0000-0000-000000000050'),
			(SELECT status FROM push_deliveries WHERE id='00000000-0000-0000-0000-000000000051'),
			(SELECT status FROM instagram_reconciliation_jobs WHERE link_id=$2 ORDER BY created_at DESC LIMIT 1)
	`, dependentSuggestionID, aliceLinkID).Scan(&suggestionState, &followStatus, &eventState, &deliveryStatus, &jobStatus); err != nil {
		t.Fatalf("inspect collision dependents: %v", err)
	}
	if suggestionState != SuggestionInvalidated || followStatus != "failed" || eventState != "retracted" || deliveryStatus != "cancelled" || jobStatus != "ignored" {
		t.Fatalf("collision dependents suggestion=%s follow=%s event=%s delivery=%s job=%s", suggestionState, followStatus, eventState, deliveryStatus, jobStatus)
	}
}

func TestVerificationServiceUsernameCollisionDoesNotTransferHiddenIdentity(t *testing.T) {
	pool := verificationServicePool(t)
	store := NewVerificationStore(pool)
	key := bytes.Repeat([]byte{0x73}, 32)
	codec, err := NewChallengeCodec(bytes.NewReader(bytes.Repeat([]byte{0x03}, 256)), key)
	if err != nil {
		t.Fatalf("codec: %v", err)
	}
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	dmURL, _ := url.Parse("https://www.instagram.com/direct/t/synthetic")
	service, err := NewVerificationService(VerificationServiceOptions{
		Store: store, Codec: codec, Now: func() time.Time { return now },
		NewID: sequentialUUIDs(0x60), TTL: 10 * time.Minute, DMURL: dmURL,
		HMACKey: key, Available: true,
	})
	if err != nil {
		t.Fatalf("service: %v", err)
	}
	ctx := context.Background()
	alice := syntax.DID("did:plc:synthetic-hidden-alice")
	bob := syntax.DID("did:plc:synthetic-hidden-bob")

	verifyCandidate := func(owner syntax.DID, igsid, username string) uuid.UUID {
		t.Helper()
		created, err := service.CreateVerification(ctx, owner)
		if err != nil {
			t.Fatalf("create %s: %v", owner, err)
		}
		digest, err := codec.Digest(created.Challenge)
		if err != nil {
			t.Fatalf("digest %s: %v", owner, err)
		}
		if _, err := store.RedeemVerificationChallenge(ctx, digest, igsid, now.Add(time.Minute)); err != nil {
			t.Fatalf("redeem %s: %v", owner, err)
		}
		if err := store.SetVerificationCandidate(ctx, created.Attempt.ID, username, now.Add(2*time.Minute)); err != nil {
			t.Fatalf("candidate %s: %v", owner, err)
		}
		return created.Attempt.ID
	}

	aliceAttempt := verifyCandidate(alice, "100000000000101", "Shared.Crafts")
	now = now.Add(3 * time.Minute)
	if _, err := service.ConfirmVerification(ctx, alice, aliceAttempt, false); err != nil {
		t.Fatalf("confirm hidden Alice link: %v", err)
	}
	bobAttempt := verifyCandidate(bob, "100000000000202", "shared.crafts")
	now = now.Add(3 * time.Minute)
	if _, err := service.ConfirmVerification(ctx, bob, bobAttempt, true); err != ErrInstagramLinkConflict {
		t.Fatalf("confirm Bob error = %v, want link conflict", err)
	}

	var owner, state string
	var discoverable, conflictPending bool
	if err := pool.QueryRow(ctx, `
		SELECT owner_did, state, discoverable, conflict_pending
		FROM instagram_account_links
		WHERE username_normalized = 'shared.crafts'
	`).Scan(&owner, &state, &discoverable, &conflictPending); err != nil {
		t.Fatalf("inspect authoritative username link: %v", err)
	}
	if owner != alice.String() || state != "active" || discoverable || !conflictPending {
		t.Fatalf("authoritative link owner=%s state=%s discoverable=%t conflict=%t", owner, state, discoverable, conflictPending)
	}
	var bobLinks, conflicts int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM instagram_account_links WHERE owner_did = $1`, bob).Scan(&bobLinks); err != nil {
		t.Fatalf("count Bob links: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM instagram_link_conflicts WHERE state = 'open'`).Scan(&conflicts); err != nil {
		t.Fatalf("count conflicts: %v", err)
	}
	if bobLinks != 0 || conflicts != 1 {
		t.Fatalf("Bob links=%d conflicts=%d", bobLinks, conflicts)
	}
}

func TestVerificationServiceConcurrentUsernameCollisionCreatesOneAuthoritativeLink(t *testing.T) {
	pool := verificationServicePool(t)
	store := NewVerificationStore(pool)
	key := bytes.Repeat([]byte{0x74}, 32)
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	dmURL, _ := url.Parse("https://www.instagram.com/direct/t/synthetic")
	newService := func(entropy byte, idStart byte) (*VerificationService, *ChallengeCodec) {
		t.Helper()
		codec, err := NewChallengeCodec(bytes.NewReader(bytes.Repeat([]byte{entropy}, 128)), key)
		if err != nil {
			t.Fatalf("codec: %v", err)
		}
		service, err := NewVerificationService(VerificationServiceOptions{
			Store: store, Codec: codec, Now: func() time.Time { return now },
			NewID: sequentialUUIDs(idStart), TTL: 10 * time.Minute, DMURL: dmURL,
			HMACKey: key, Available: true,
		})
		if err != nil {
			t.Fatalf("service: %v", err)
		}
		return service, codec
	}
	aliceService, aliceCodec := newService(0x04, 0x80)
	bobService, bobCodec := newService(0x05, 0xa0)
	ctx := context.Background()
	alice := syntax.DID("did:plc:synthetic-concurrent-alice")
	bob := syntax.DID("did:plc:synthetic-concurrent-bob")

	prepare := func(service *VerificationService, codec *ChallengeCodec, owner syntax.DID, igsid string) uuid.UUID {
		t.Helper()
		created, err := service.CreateVerification(ctx, owner)
		if err != nil {
			t.Fatalf("create %s: %v", owner, err)
		}
		digest, err := codec.Digest(created.Challenge)
		if err != nil {
			t.Fatalf("digest %s: %v", owner, err)
		}
		if _, err := store.RedeemVerificationChallenge(ctx, digest, igsid, now.Add(time.Minute)); err != nil {
			t.Fatalf("redeem %s: %v", owner, err)
		}
		if err := store.SetVerificationCandidate(ctx, created.Attempt.ID, "Concurrent.Crafts", now.Add(2*time.Minute)); err != nil {
			t.Fatalf("candidate %s: %v", owner, err)
		}
		return created.Attempt.ID
	}
	aliceAttempt := prepare(aliceService, aliceCodec, alice, "100000000000301")
	bobAttempt := prepare(bobService, bobCodec, bob, "100000000000302")

	start := make(chan struct{})
	errorsByOwner := make(chan error, 2)
	var wg sync.WaitGroup
	for _, confirmation := range []struct {
		service *VerificationService
		owner   syntax.DID
		attempt uuid.UUID
	}{
		{aliceService, alice, aliceAttempt},
		{bobService, bob, bobAttempt},
	} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := confirmation.service.ConfirmVerification(ctx, confirmation.owner, confirmation.attempt, true)
			errorsByOwner <- err
		}()
	}
	close(start)
	wg.Wait()
	close(errorsByOwner)

	var successes, collisions int
	for err := range errorsByOwner {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrInstagramLinkConflict):
			collisions++
		default:
			t.Fatalf("unexpected confirmation error: %v", err)
		}
	}
	if successes != 1 || collisions != 1 {
		t.Fatalf("successes=%d collisions=%d, want one each", successes, collisions)
	}
	var links, conflicts int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM instagram_account_links
		WHERE username_normalized = 'concurrent.crafts'
		  AND state IN ('active', 'membershipInactive', 'disputed')
	`).Scan(&links); err != nil {
		t.Fatalf("count authoritative links: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM instagram_link_conflicts WHERE state = 'open'`).Scan(&conflicts); err != nil {
		t.Fatalf("count conflicts: %v", err)
	}
	if links != 1 || conflicts != 1 {
		t.Fatalf("links=%d conflicts=%d, want one each", links, conflicts)
	}
}

func TestVerificationServiceSupersessionImmediatelyPurgesPlaintextIdentity(t *testing.T) {
	pool := verificationServicePool(t)
	store := NewVerificationStore(pool)
	key := bytes.Repeat([]byte{0x75}, 32)
	codec, err := NewChallengeCodec(bytes.NewReader(bytes.Repeat([]byte{0x06}, 256)), key)
	if err != nil {
		t.Fatalf("codec: %v", err)
	}
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	dmURL, _ := url.Parse("https://www.instagram.com/direct/t/synthetic")
	service, err := NewVerificationService(VerificationServiceOptions{
		Store: store, Codec: codec, Now: func() time.Time { return now },
		NewID: sequentialUUIDs(0xc0), TTL: 10 * time.Minute, DMURL: dmURL,
		HMACKey: key, Available: true,
	})
	if err != nil {
		t.Fatalf("service: %v", err)
	}
	ctx := context.Background()
	owner := syntax.DID("did:plc:synthetic-supersession")

	confirm := func(igsid, username string) uuid.UUID {
		t.Helper()
		created, err := service.CreateVerification(ctx, owner)
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		digest, err := codec.Digest(created.Challenge)
		if err != nil {
			t.Fatalf("digest: %v", err)
		}
		if _, err := store.RedeemVerificationChallenge(ctx, digest, igsid, now.Add(time.Minute)); err != nil {
			t.Fatalf("redeem: %v", err)
		}
		if err := store.SetVerificationCandidate(ctx, created.Attempt.ID, username, now.Add(2*time.Minute)); err != nil {
			t.Fatalf("candidate: %v", err)
		}
		now = now.Add(3 * time.Minute)
		if _, err := service.ConfirmVerification(ctx, owner, created.Attempt.ID, true); err != nil {
			t.Fatalf("confirm: %v", err)
		}
		var linkID uuid.UUID
		if err := pool.QueryRow(ctx, `
			SELECT id FROM instagram_account_links
			WHERE owner_did = $1 AND state = 'active'
		`, owner).Scan(&linkID); err != nil {
			t.Fatalf("read current link: %v", err)
		}
		return linkID
	}

	oldLinkID := confirm("100000000000401", "old.identity")
	dependentSuggestionID := uuid.MustParse("00000000-0000-0000-0000-0000000000ce")
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_follow_suggestions(
			id,importer_did,target_did,state,reason,accepting_since,created_at,updated_at
		) VALUES($1,'did:plc:synthetic-supersession-importer',$2,'accepting','verifiedInstagramFollow',$3,$3,$3)
	`, dependentSuggestionID, owner, now); err != nil {
		t.Fatalf("seed supersession suggestion: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO pds_follow_operations(
			id,suggestion_id,owner_did,target_did,rkey,status,attempt_count,created_at,updated_at
		) VALUES(
			'00000000-0000-0000-0000-0000000000cf',$1,
			'did:plc:synthetic-supersession-importer',$2,'3kysupersession','writing',1,$3,$3
		)
	`, dependentSuggestionID, owner, now); err != nil {
		t.Fatalf("seed supersession follow operation: %v", err)
	}
	seedLifecycleNotification(
		t,
		pool,
		uuid.MustParse("00000000-0000-0000-0000-0000000000d0"),
		syntax.DID("did:plc:synthetic-supersession-importer"),
		dependentSuggestionID,
		"00000000-0000-0000-0000-0000000000d1",
		"pending",
		now,
	)
	newLinkID := confirm("100000000000402", "new.identity")
	if oldLinkID == newLinkID {
		t.Fatal("supersession reused the prior link")
	}
	var state InstagramLinkState
	var plaintextFields int
	var purgeAt *time.Time
	if err := pool.QueryRow(ctx, `
		SELECT state, num_nonnulls(igsid, username, username_normalized), raw_identity_purge_at
		FROM instagram_account_links WHERE id = $1
	`, oldLinkID).Scan(&state, &plaintextFields, &purgeAt); err != nil {
		t.Fatalf("inspect superseded link: %v", err)
	}
	if state != LinkSuperseded || plaintextFields != 0 || purgeAt == nil || !purgeAt.Equal(now) {
		t.Fatalf("superseded link state=%s plaintextFields=%d purgeAt=%v", state, plaintextFields, purgeAt)
	}
	var suggestionState InstagramSuggestionState
	var followStatus, eventState, deliveryStatus, jobStatus string
	if err := pool.QueryRow(ctx, `
		SELECT
			(SELECT state FROM instagram_follow_suggestions WHERE id=$1),
			(SELECT status FROM pds_follow_operations WHERE suggestion_id=$1),
			(SELECT state FROM notification_events WHERE id='00000000-0000-0000-0000-0000000000d0'),
			(SELECT status FROM push_deliveries WHERE id='00000000-0000-0000-0000-0000000000d1'),
			(SELECT status FROM instagram_reconciliation_jobs WHERE link_id=$2 ORDER BY created_at DESC LIMIT 1)
	`, dependentSuggestionID, oldLinkID).Scan(&suggestionState, &followStatus, &eventState, &deliveryStatus, &jobStatus); err != nil {
		t.Fatalf("inspect supersession dependents: %v", err)
	}
	if suggestionState != SuggestionInvalidated || followStatus != "failed" || eventState != "retracted" || deliveryStatus != "cancelled" || jobStatus != "ignored" {
		t.Fatalf("supersession dependents suggestion=%s follow=%s event=%s delivery=%s job=%s", suggestionState, followStatus, eventState, deliveryStatus, jobStatus)
	}
}

func TestVerificationServiceDisabledFailsOnlyMetaDependentCreation(t *testing.T) {
	t.Parallel()

	service, err := NewVerificationService(VerificationServiceOptions{})
	if err != nil {
		t.Fatalf("disabled service: %v", err)
	}
	if _, err := service.CreateVerification(context.Background(), syntax.DID("did:plc:synthetic")); err != ErrVerificationUnavailable {
		t.Fatalf("create error = %v", err)
	}
	if err := service.CancelVerification(context.Background(), syntax.DID("did:plc:synthetic"), uuid.New()); err != nil {
		t.Fatalf("disabled cancellation should remain privacy-safe: %v", err)
	}
}

func verificationServicePool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	var migration strings.Builder
	for _, name := range []string{
		"000021_appview_notifications.up.sql",
		"000022_notification_newness.up.sql",
		"000023_instagram_migration.up.sql",
		"000024_system_notifications.up.sql",
	} {
		contents, err := os.ReadFile("../../migrations/" + name)
		if err != nil {
			t.Fatalf("read migration %s: %v", name, err)
		}
		migration.Write(contents)
		migration.WriteByte('\n')
	}
	return testdb.WithSchema(t, migration.String())
}

func sequentialUUIDs(start byte) func() uuid.UUID {
	next := start
	return func() uuid.UUID {
		var id uuid.UUID
		id[15] = next
		next++
		return id
	}
}
