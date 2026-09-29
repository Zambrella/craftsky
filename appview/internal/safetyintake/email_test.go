package safetyintake

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/testdb"
)

const intakePreStateDDL = `
CREATE TABLE moderation_cases (id UUID PRIMARY KEY, owner_did TEXT NOT NULL);
CREATE TABLE moderation_appeal_correspondence (
 id UUID PRIMARY KEY, case_id UUID NOT NULL REFERENCES moderation_cases(id), source_system TEXT NOT NULL,
 replay_id TEXT NOT NULL, sender_reference_hash BYTEA, received_at TIMESTAMPTZ NOT NULL, created_at TIMESTAMPTZ NOT NULL,
 UNIQUE(source_system,replay_id)
);
`

func TestAcceptEmailMinimizesExternalProvenanceWithoutReporterDID(t *testing.T) {
	pool := testdb.WithSchema(t, intakePreStateDDL)
	applyIntakeMigration(t, pool)
	ctx := context.Background()
	now := time.Date(2030, 5, 1, 12, 0, 0, 0, time.UTC)
	store := NewStore(pool, func() time.Time { return now.Add(time.Second) })
	input := EmailMetadata{
		ProviderMessageReference: "provider-message-001", ReceivedAt: now,
		CanonicalSubject: "at://did:plc:subject/social.craftsky.feed.post/abc",
		Kind:             KindComplaint, Urgency: UrgencyPriority, OwnerActorID: "intake-owner",
		SenderContact: "Reporter.Person@example.invalid", AttachmentStatus: AttachmentNone,
	}
	accepted, err := store.AcceptEmail(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := store.AcceptEmail(ctx, input)
	if err != nil || !replay.Replayed || replay.ID != accepted.ID {
		t.Fatalf("replay = %+v, %v", replay, err)
	}
	if accepted.Reference == "" || accepted.ContactReference == "" || strings.Contains(accepted.ContactReference, "reporter.person") {
		t.Fatalf("unsafe intake result = %+v", accepted)
	}
	var count, reporterColumns int
	var source, contact string
	if err := pool.QueryRow(ctx, `SELECT count(*),min(canonical_subject),min(contact_reference) FROM external_safety_intakes`).Scan(&count, &source, &contact); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM information_schema.columns WHERE table_schema=current_schema() AND table_name LIKE 'external_safety%' AND column_name='reporter_did'`).Scan(&reporterColumns); err != nil {
		t.Fatal(err)
	}
	if count != 1 || source != input.CanonicalSubject || contact != accepted.ContactReference || reporterColumns != 0 {
		t.Fatalf("count=%d subject=%s contact=%s reporterColumns=%d", count, source, contact, reporterColumns)
	}
}

func applyIntakeMigration(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	migration, err := os.ReadFile("../../migrations/000080_external_safety_intake.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
		t.Fatalf("apply intake migration: %v", err)
	}
}
