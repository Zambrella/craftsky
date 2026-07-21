package instagram

import (
	"bytes"
	"context"
	"net/url"
	"os"
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
	migration, err := os.ReadFile("../../migrations/000023_instagram_migration.up.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	return testdb.WithSchema(t, string(migration))
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
