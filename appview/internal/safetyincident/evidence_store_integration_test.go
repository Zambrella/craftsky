package safetyincident

import (
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/testdb"
)

func TestRestrictedEvidenceRequiresExplicitAuthorizedPreservationAndAuditsAccess(t *testing.T) {
	pool := testdb.WithSchema(t, evidenceTestPreStateDDL)
	applyEvidenceMigration(t, pool, "../../migrations/000078_safety_evidence_holds.up.sql")

	ctx := context.Background()
	now := time.Date(2030, 9, 22, 17, 0, 0, 0, time.UTC)
	incidentID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO safety_incidents(id,state,incident_kind,detected_at,updated_at) VALUES($1,'detected','imageMatch',$2,$2)`, incidentID, now); err != nil {
		t.Fatal(err)
	}
	objects := NewMemoryEvidenceStore()
	service := NewEvidenceService(pool, objects, func() time.Time { return now })
	admin := Actor{ID: "safety-admin", Role: RoleSafetyAdministrator}
	moderator := Actor{ID: "ordinary-moderator", Role: RoleModerator}
	benign := []byte("benign restricted evidence fixture")
	digest := sha256.Sum256(benign)

	if got := objects.ObjectCount(); got != 0 {
		t.Fatalf("default incident copied bytes: object count=%d", got)
	}
	if _, err := service.Preserve(ctx, moderator, PreserveCommand{IncidentID: incidentID, Bytes: benign, SHA256: digest, ContentType: "image/png", Reason: "approved procedure", ExpiresAt: now.Add(24 * time.Hour)}); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("moderator preserve error=%v, want unauthorized", err)
	}
	ref, err := service.Preserve(ctx, admin, PreserveCommand{IncidentID: incidentID, Bytes: benign, SHA256: digest, ContentType: "image/png", Reason: "approved procedure", ExpiresAt: now.Add(24 * time.Hour)})
	if err != nil {
		t.Fatalf("preserve evidence: %v", err)
	}
	if ref.ID == uuid.Nil || ref.ObjectKey == "" || objects.ObjectCount() != 1 {
		t.Fatalf("evidence ref=%+v objects=%d", ref, objects.ObjectCount())
	}

	if _, err := service.Access(ctx, moderator, AccessCommand{EvidenceID: ref.ID, Reason: "not authorized"}); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("moderator access error=%v, want unauthorized", err)
	}
	stream, err := service.Access(ctx, admin, AccessCommand{EvidenceID: ref.ID, Reason: "authority follow-up"})
	if err != nil {
		t.Fatalf("access evidence: %v", err)
	}
	got, err := io.ReadAll(stream)
	if closeErr := stream.Close(); err == nil {
		err = closeErr
	}
	if err != nil || !reflect.DeepEqual(got, benign) {
		t.Fatalf("read evidence=%q err=%v", got, err)
	}

	var allowed, denied int
	if err := pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE allowed),count(*) FILTER (WHERE NOT allowed) FROM safety_evidence_accesses WHERE evidence_id=$1`, ref.ID).Scan(&allowed, &denied); err != nil {
		t.Fatal(err)
	}
	if allowed != 1 || denied != 1 {
		t.Fatalf("access audit allowed/denied=%d/%d, want 1/1", allowed, denied)
	}
}

func TestEvidenceStoreContractHasNoPublicReadOrURLMethod(t *testing.T) {
	typeOfStore := reflect.TypeOf((*EvidenceStore)(nil)).Elem()
	for _, forbidden := range []string{"Get", "Open", "Read", "URL", "Presign"} {
		if _, ok := typeOfStore.MethodByName(forbidden); ok {
			t.Fatalf("EvidenceStore exposes forbidden method %s", forbidden)
		}
	}
	if _, ok := typeOfStore.MethodByName("Put"); !ok {
		t.Fatal("EvidenceStore does not expose Put")
	}
	if _, ok := typeOfStore.MethodByName("Delete"); !ok {
		t.Fatal("EvidenceStore does not expose Delete")
	}
}

const evidenceTestPreStateDDL = `
CREATE TABLE safety_incidents (
    id UUID PRIMARY KEY,
    incident_kind TEXT NOT NULL,
    state TEXT NOT NULL,
    detected_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);`

func applyEvidenceMigration(t *testing.T, pool *pgxpool.Pool, path string) {
	t.Helper()
	migration, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
		t.Fatalf("apply %s: %v", path, err)
	}
}
