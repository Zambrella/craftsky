package retention

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/safetyincident"
	"social.craftsky/appview/internal/testdb"
)

func TestWorkerDeletesDueEvidenceIdempotentlyAndPreservesActiveHolds(t *testing.T) {
	pool := testdb.WithSchema(t, retentionPreStateDDL)
	applyRetentionMigration(t, pool, "../../migrations/000078_safety_evidence_holds.up.sql")
	applyRetentionMigration(t, pool, "../../migrations/000083_safety_retention.up.sql")
	ctx := context.Background()
	now := time.Date(2030, 9, 22, 19, 0, 0, 0, time.UTC)
	store := safetyincident.NewMemoryEvidenceStore()
	service := safetyincident.NewEvidenceService(pool, store, func() time.Time { return now.Add(-48 * time.Hour) })
	admin := safetyincident.Actor{ID: "admin", Role: safetyincident.RoleSafetyAdministrator}
	incidentID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO safety_incidents(id) VALUES($1)`, incidentID); err != nil {
		t.Fatal(err)
	}
	preserve := func(label string) safetyincident.EvidenceRef {
		value := []byte("benign-" + label)
		ref, err := service.Preserve(ctx, admin, safetyincident.PreserveCommand{IncidentID: incidentID, Bytes: value, SHA256: sha256.Sum256(value), ContentType: "image/png", Reason: "approved", ExpiresAt: now.Add(-time.Hour)})
		if err != nil {
			t.Fatal(err)
		}
		return ref
	}
	due := preserve("due")
	held := preserve("held")
	holdID := uuid.New()
	if _, err := pool.Exec(ctx, `
		INSERT INTO safety_legal_holds(id,incident_id,basis,approved_by,created_by,expires_at,created_at)
		VALUES($1,$2,'basis','approver','creator',$3,$4)
	`, holdID, incidentID, now.Add(24*time.Hour), now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO safety_legal_hold_evidence(hold_id,evidence_id,created_at) VALUES($1,$2,$3)
	`, holdID, held.ID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	worker := NewWorker(pool, store, func() time.Time { return now })
	if count, err := worker.RunOnce(ctx, 10); err != nil || count != 1 {
		t.Fatalf("first run count=%d err=%v", count, err)
	}
	if count, err := worker.RunOnce(ctx, 10); err != nil || count != 0 {
		t.Fatalf("repeat run count=%d err=%v", count, err)
	}
	var dueDeleted, heldDeleted *time.Time
	if err := pool.QueryRow(ctx, `SELECT (SELECT deleted_at FROM safety_evidence WHERE id=$1),(SELECT deleted_at FROM safety_evidence WHERE id=$2)`, due.ID, held.ID).Scan(&dueDeleted, &heldDeleted); err != nil {
		t.Fatal(err)
	}
	if dueDeleted == nil || heldDeleted != nil || store.ObjectCount() != 1 {
		t.Fatalf("deleted due/held=%v/%v objects=%d", dueDeleted, heldDeleted, store.ObjectCount())
	}
	var completion int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM safety_retention_events WHERE item_reference=$1 AND outcome='deleted'`, due.ID).Scan(&completion); err != nil || completion != 1 {
		t.Fatalf("completion=%d err=%v", completion, err)
	}
}

func TestWorkerRetriesFailuresAndDeadLettersAtTheBound(t *testing.T) {
	pool := testdb.WithSchema(t, retentionPreStateDDL)
	applyRetentionMigration(t, pool, "../../migrations/000078_safety_evidence_holds.up.sql")
	applyRetentionMigration(t, pool, "../../migrations/000083_safety_retention.up.sql")
	ctx := context.Background()
	now := time.Date(2030, 9, 22, 19, 0, 0, 0, time.UTC)
	objects := &failingEvidenceStore{MemoryEvidenceStore: safetyincident.NewMemoryEvidenceStore()}
	service := safetyincident.NewEvidenceService(pool, objects, func() time.Time { return now.Add(-48 * time.Hour) })
	admin := safetyincident.Actor{ID: "admin", Role: safetyincident.RoleSafetyAdministrator}
	incidentID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO safety_incidents(id) VALUES($1)`, incidentID); err != nil {
		t.Fatal(err)
	}
	value := []byte("benign-dead-letter")
	ref, err := service.Preserve(ctx, admin, safetyincident.PreserveCommand{
		IncidentID: incidentID, Bytes: value, SHA256: sha256.Sum256(value), ContentType: "image/png",
		Reason: "approved", ExpiresAt: now.Add(-time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	clock := now
	worker := NewWorker(pool, objects, func() time.Time { return clock })
	for attempt := 1; attempt <= defaultMaxAttempts; attempt++ {
		if _, err := worker.RunOnce(ctx, 1); err != nil {
			t.Fatalf("attempt %d: %v", attempt, err)
		}
		clock = clock.Add(defaultRetryBackoff << (attempt - 1))
	}
	var state, category string
	var attempts, history int
	if err := pool.QueryRow(ctx, `SELECT state,safe_error_category,attempt_count FROM safety_retention_jobs
		WHERE item_reference=$1`, ref.ID).Scan(&state, &category, &attempts); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM safety_retention_attempts attempts
		JOIN safety_retention_jobs jobs ON jobs.id=attempts.job_id WHERE jobs.item_reference=$1`, ref.ID).Scan(&history); err != nil {
		t.Fatal(err)
	}
	if state != "deadLetter" || category != "objectDelete" || attempts != defaultMaxAttempts || history != defaultMaxAttempts {
		t.Fatalf("state=%s category=%s attempts/history=%d/%d", state, category, attempts, history)
	}
}

type failingEvidenceStore struct {
	*safetyincident.MemoryEvidenceStore
}

func (store *failingEvidenceStore) Delete(context.Context, safetyincident.ObjectRef) error {
	return errors.Join(safetyincident.ErrEvidenceStoreBoundary, errors.New("synthetic object failure"))
}

const retentionPreStateDDL = `
	CREATE TABLE safety_incidents(id UUID PRIMARY KEY);
	CREATE TABLE safety_incident_subjects(
		incident_id UUID NOT NULL REFERENCES safety_incidents(id) ON DELETE CASCADE,
		owner_did TEXT
	);`

func applyRetentionMigration(t *testing.T, pool *pgxpool.Pool, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), string(data)); err != nil {
		t.Fatal(err)
	}
}
