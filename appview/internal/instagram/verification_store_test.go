package instagram

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/testdb"
)

func TestVerificationStoreCreatesOneActiveAttemptAndSupersedesSensitiveState(t *testing.T) {
	store := newVerificationTestStore(t)
	ctx := context.Background()
	owner := syntax.DID("did:plc:synthetic-alice")
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	firstDigest := syntheticChallengeDigest(0x11)
	secondDigest := syntheticChallengeDigest(0x22)

	first, err := store.CreateVerificationAttempt(ctx, CreateVerificationAttemptParams{
		ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		OwnerDID:  owner,
		Digest:    firstDigest,
		ExpiresAt: now.Add(10 * time.Minute),
		Now:       now,
	})
	if err != nil {
		t.Fatalf("create first: %v", err)
	}
	if first.State != AttemptPendingDM || first.OwnerDID != owner {
		t.Fatalf("first attempt = %+v", first)
	}

	second, err := store.CreateVerificationAttempt(ctx, CreateVerificationAttemptParams{
		ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		OwnerDID:  owner,
		Digest:    secondDigest,
		ExpiresAt: now.Add(11 * time.Minute),
		Now:       now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("create second: %v", err)
	}
	if second.State != AttemptPendingDM {
		t.Fatalf("second state = %q", second.State)
	}

	gotFirst, err := store.GetVerificationAttempt(ctx, owner, first.ID, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("get first: %v", err)
	}
	if gotFirst.State != AttemptSuperseded || gotFirst.Digest != nil || gotFirst.CandidateIGSID != "" || gotFirst.CandidateUsername != "" {
		t.Fatalf("superseded first retained state: %+v", gotFirst)
	}

	if _, err := store.GetVerificationAttempt(ctx, syntax.DID("did:plc:synthetic-bob"), second.ID, now); !errors.Is(err, ErrInstagramResourceNotFound) {
		t.Fatalf("foreign read error = %v, want privacy-safe not found", err)
	}
	if got := second.String(); strings.Contains(got, owner.String()) || strings.Contains(got, strings.Repeat("22", 32)) {
		t.Fatalf("attempt diagnostic leaked private values: %q", got)
	}
}

func TestVerificationStoreExpiresCancelsAndPrivacyNoOps(t *testing.T) {
	store := newVerificationTestStore(t)
	ctx := context.Background()
	alice := syntax.DID("did:plc:synthetic-alice")
	bob := syntax.DID("did:plc:synthetic-bob")
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)

	attempt, err := store.CreateVerificationAttempt(ctx, CreateVerificationAttemptParams{
		ID:        uuid.MustParse("00000000-0000-0000-0000-000000000003"),
		OwnerDID:  alice,
		Digest:    syntheticChallengeDigest(0x33),
		ExpiresAt: now.Add(time.Minute),
		Now:       now,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := store.CancelVerificationAttempt(ctx, bob, attempt.ID, now.Add(10*time.Second)); err != nil {
		t.Fatalf("foreign cancel must be a no-op: %v", err)
	}
	stillPending, err := store.GetVerificationAttempt(ctx, alice, attempt.ID, now.Add(10*time.Second))
	if err != nil || stillPending.State != AttemptPendingDM {
		t.Fatalf("foreign cancel changed owned row: %+v, %v", stillPending, err)
	}

	missing := uuid.MustParse("00000000-0000-0000-0000-000000000099")
	if err := store.CancelVerificationAttempt(ctx, alice, missing, now); err != nil {
		t.Fatalf("absent cancel must be a no-op: %v", err)
	}

	expired, err := store.GetVerificationAttempt(ctx, alice, attempt.ID, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("get at exact expiry: %v", err)
	}
	if expired.State != AttemptExpired || expired.Digest != nil {
		t.Fatalf("expired attempt = %+v", expired)
	}
	if err := store.CancelVerificationAttempt(ctx, alice, attempt.ID, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("terminal cancel replay: %v", err)
	}
}

func TestVerificationStoreRedeemsChallengeExactlyOnce(t *testing.T) {
	store := newVerificationTestStore(t)
	ctx := context.Background()
	owner := syntax.DID("did:plc:synthetic-alice")
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	digest := syntheticChallengeDigest(0x44)

	attempt, err := store.CreateVerificationAttempt(ctx, CreateVerificationAttemptParams{
		ID:        uuid.MustParse("00000000-0000-0000-0000-000000000004"),
		OwnerDID:  owner,
		Digest:    digest,
		ExpiresAt: now.Add(10 * time.Minute),
		Now:       now,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	redeemed, err := store.RedeemVerificationChallenge(ctx, digest, "synthetic-igsid-canary", now.Add(time.Minute))
	if err != nil {
		t.Fatalf("redeem: %v", err)
	}
	if redeemed.ID != attempt.ID || redeemed.State != AttemptProcessing || redeemed.Digest != nil {
		t.Fatalf("redeemed attempt = %+v", redeemed)
	}
	if redeemed.CandidateIGSID != "synthetic-igsid-canary" {
		t.Fatal("redeemed attempt did not retain the temporary sender candidate")
	}
	if _, err := store.RedeemVerificationChallenge(ctx, digest, "synthetic-other-igsid", now.Add(2*time.Minute)); !errors.Is(err, ErrInstagramResourceNotFound) {
		t.Fatalf("replay error = %v, want non-disclosing not found", err)
	}

	if err := store.SetVerificationCandidate(ctx, attempt.ID, "synthetic.candidate", now.Add(2*time.Minute)); err != nil {
		t.Fatalf("set candidate: %v", err)
	}
	pending, err := store.GetVerificationAttempt(ctx, owner, attempt.ID, now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("get candidate: %v", err)
	}
	if pending.State != AttemptPendingConfirmation || pending.CandidateUsername != "synthetic.candidate" {
		t.Fatalf("pending confirmation = %+v", pending)
	}
}

func newVerificationTestStore(t *testing.T) *VerificationStore {
	t.Helper()
	migration, err := os.ReadFile("../../migrations/000023_instagram_migration.up.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	pool := testdb.WithSchema(t, string(migration))
	return NewVerificationStore(pool)
}

func syntheticChallengeDigest(value byte) ChallengeDigest {
	digest := ChallengeDigest{Version: ChallengeDigestVersion}
	copy(digest.Value[:], bytes.Repeat([]byte{value}, len(digest.Value)))
	return digest
}
