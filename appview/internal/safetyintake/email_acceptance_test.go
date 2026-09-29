package safetyintake

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/testdb"
)

func TestExternalAppealLinksToExistingModerationChronology(t *testing.T) {
	pool := testdb.WithSchema(t, intakePreStateDDL)
	applyIntakeMigration(t, pool)
	ctx := context.Background()
	now := time.Date(2030, 5, 2, 8, 0, 0, 0, time.UTC)
	store := NewStore(pool, func() time.Time { return now.Add(time.Minute) })
	owner := syntax.DID("did:plc:appeal-owner")
	caseID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO moderation_cases(id,owner_did) VALUES($1,$2)`, caseID, owner); err != nil {
		t.Fatal(err)
	}
	intake, err := store.AcceptEmail(ctx, EmailMetadata{
		ProviderMessageReference: "appeal-message-1", ReceivedAt: now, CanonicalSubject: "case:" + caseID.String(),
		Kind: KindAppeal, Urgency: UrgencyRoutine, OwnerActorID: "moderator", SenderContact: "owner@example.invalid",
		AttachmentStatus: AttachmentNone,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.BindAppeal(ctx, AppealLink{
		IntakeID: intake.ID, CaseID: caseID, VerifiedOwnerDID: syntax.DID("did:plc:wrong-owner"), ActorID: "moderator",
		SourceSystem: "externalEmail", ReplayID: "appeal-message-1", ReceivedAt: now,
	}); !errors.Is(err, ErrAppealOwnerMismatch) {
		t.Fatalf("wrong-owner error = %v", err)
	}
	correspondenceID, err := store.BindAppeal(ctx, AppealLink{
		IntakeID: intake.ID, CaseID: caseID, VerifiedOwnerDID: owner, ActorID: "moderator",
		SourceSystem: "externalEmail", ReplayID: "appeal-message-1", ReceivedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	replayedID, err := store.BindAppeal(ctx, AppealLink{
		IntakeID: intake.ID, CaseID: caseID, VerifiedOwnerDID: owner, ActorID: "moderator",
		SourceSystem: "externalEmail", ReplayID: "appeal-message-1", ReceivedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if replayedID != correspondenceID {
		t.Fatalf("replayed correspondence ID=%s, want %s", replayedID, correspondenceID)
	}
	var moderationRows, linkRows int
	if err := pool.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM moderation_appeal_correspondence WHERE id=$1 AND case_id=$2),
		(SELECT count(*) FROM external_safety_appeal_links WHERE appeal_correspondence_id=$1 AND intake_id=$3)`,
		correspondenceID, caseID, intake.ID).Scan(&moderationRows, &linkRows); err != nil {
		t.Fatal(err)
	}
	if moderationRows != 1 || linkRows != 1 {
		t.Fatalf("moderation/link chronology = %d/%d", moderationRows, linkRows)
	}
}

func TestRepeatedExternalMessagesAppendToCanonicalIntake(t *testing.T) {
	pool := testdb.WithSchema(t, intakePreStateDDL)
	applyIntakeMigration(t, pool)
	now := time.Date(2030, 5, 2, 9, 0, 0, 0, time.UTC)
	store := NewStore(pool, func() time.Time { return now })
	intake, err := store.AcceptEmail(context.Background(), EmailMetadata{
		ProviderMessageReference: "thread-first", ReceivedAt: now, CanonicalSubject: "https://craftsky.social/post/synthetic",
		Kind: KindAllegation, Urgency: UrgencyRoutine, OwnerActorID: "moderator", AttachmentStatus: AttachmentNone,
	})
	if err != nil {
		t.Fatal(err)
	}
	first, replayed, err := store.AppendCorrespondence(context.Background(), CorrespondenceMetadata{
		IntakeID: intake.ID, ProviderMessageReference: "thread-followup", ReceivedAt: now.Add(time.Minute), AttachmentStatus: AttachmentNone,
	})
	if err != nil || replayed {
		t.Fatalf("first correspondence = %s replayed=%t err=%v", first, replayed, err)
	}
	second, replayed, err := store.AppendCorrespondence(context.Background(), CorrespondenceMetadata{
		IntakeID: intake.ID, ProviderMessageReference: "thread-followup", ReceivedAt: now.Add(time.Minute), AttachmentStatus: AttachmentNone,
	})
	if err != nil || !replayed || second != first {
		t.Fatalf("replayed correspondence = %s replayed=%t err=%v", second, replayed, err)
	}
}

func TestExternalIntakeRejectsNonCraftSkyCanonicalURL(t *testing.T) {
	pool := testdb.WithSchema(t, intakePreStateDDL)
	applyIntakeMigration(t, pool)
	_, err := NewStore(pool, time.Now).AcceptEmail(context.Background(), EmailMetadata{
		ProviderMessageReference: "invalid-subject", ReceivedAt: time.Now(),
		CanonicalSubject: "https://attacker.invalid/post/one", Kind: KindAllegation,
		Urgency: UrgencyRoutine, OwnerActorID: "moderator", AttachmentStatus: AttachmentNone,
	})
	if !errors.Is(err, ErrInvalidEmailMetadata) {
		t.Fatalf("invalid canonical subject error = %v", err)
	}
}

func TestCorrespondenceReplayCannotCrossIntakes(t *testing.T) {
	pool := testdb.WithSchema(t, intakePreStateDDL)
	applyIntakeMigration(t, pool)
	now := time.Date(2030, 5, 2, 9, 0, 0, 0, time.UTC)
	store := NewStore(pool, func() time.Time { return now })
	firstIntake, err := store.AcceptEmail(context.Background(), EmailMetadata{
		ProviderMessageReference: "thread-one", ReceivedAt: now,
		CanonicalSubject: "https://craftsky.social/post/one", Kind: KindAllegation,
		Urgency: UrgencyRoutine, OwnerActorID: "moderator", AttachmentStatus: AttachmentNone,
	})
	if err != nil {
		t.Fatal(err)
	}
	secondIntake, err := store.AcceptEmail(context.Background(), EmailMetadata{
		ProviderMessageReference: "thread-two", ReceivedAt: now,
		CanonicalSubject: "https://craftsky.social/post/two", Kind: KindAllegation,
		Urgency: UrgencyRoutine, OwnerActorID: "moderator", AttachmentStatus: AttachmentNone,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.AppendCorrespondence(context.Background(), CorrespondenceMetadata{
		IntakeID: firstIntake.ID, ProviderMessageReference: "shared-followup",
		ReceivedAt: now.Add(time.Minute), AttachmentStatus: AttachmentNone,
	}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.AppendCorrespondence(context.Background(), CorrespondenceMetadata{
		IntakeID: secondIntake.ID, ProviderMessageReference: "shared-followup",
		ReceivedAt: now.Add(time.Minute), AttachmentStatus: AttachmentNone,
	}); !errors.Is(err, ErrCorrespondenceIntakeMismatch) {
		t.Fatalf("cross-intake replay error = %v", err)
	}
}
