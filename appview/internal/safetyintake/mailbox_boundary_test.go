package safetyintake

import (
	"context"
	"errors"
	"testing"
	"time"

	"social.craftsky/appview/internal/testdb"
)

func TestMailboxBoundaryRequiresSafeMediaHandlingStatus(t *testing.T) {
	pool := testdb.WithSchema(t, intakePreStateDDL)
	applyIntakeMigration(t, pool)
	store := NewStore(pool, time.Now)
	base := EmailMetadata{
		ProviderMessageReference: "media-message", ReceivedAt: time.Now().UTC(),
		CanonicalSubject: "at://did:plc:subject/social.craftsky.feed.post/media",
		Kind:             KindIncident, Urgency: UrgencyUrgent, OwnerActorID: "safety-owner",
		SenderContact: "reporter@example.invalid", HasMediaAttachment: true,
	}
	if _, err := store.AcceptEmail(context.Background(), base); !errors.Is(err, ErrUnsafeAttachment) {
		t.Fatalf("unsafe attachment error = %v", err)
	}
	base.AttachmentStatus = AttachmentQuarantined
	accepted, err := store.AcceptEmail(context.Background(), base)
	if err != nil {
		t.Fatal(err)
	}
	var status string
	var byteColumns int
	if err := pool.QueryRow(context.Background(), `SELECT attachment_status FROM external_safety_intakes WHERE id=$1`, accepted.ID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM information_schema.columns
		WHERE table_schema=current_schema() AND table_name LIKE 'external_safety%' AND data_type='bytea'`).Scan(&byteColumns); err != nil {
		t.Fatal(err)
	}
	if status != "quarantined" || byteColumns != 0 {
		t.Fatalf("status=%s byteColumns=%d", status, byteColumns)
	}
}
