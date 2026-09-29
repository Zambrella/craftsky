package accountdeletion

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/safetyincident"
	"social.craftsky/appview/internal/testdb"
)

func TestAccountDeletionPreservesOnlyActiveInScopeHeldEvidence(t *testing.T) {
	pool := testdb.WithSchema(t, `
		CREATE TABLE safety_incidents(id UUID PRIMARY KEY);
		CREATE TABLE safety_incident_subjects(
			incident_id UUID NOT NULL REFERENCES safety_incidents(id) ON DELETE CASCADE,
			owner_did TEXT,
			PRIMARY KEY(incident_id)
		);`)
	applyMigrationFile(t, pool, "../../migrations/000075_safety_evidence_holds.up.sql")
	ctx := context.Background()
	now := time.Date(2030, 9, 22, 20, 0, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:deleting-owner")
	store := safetyincident.NewMemoryEvidenceStore()
	type fixture struct {
		name        string
		activeHold  bool
		expiredHold bool
		outOfScope  bool
		id          uuid.UUID
		incidentID  uuid.UUID
	}
	fixtures := []fixture{{name: "unheld"}, {name: "active", activeHold: true}, {name: "expired", expiredHold: true}, {name: "out-of-scope", outOfScope: true}}
	for index := range fixtures {
		item := &fixtures[index]
		item.id = uuid.New()
		item.incidentID = uuid.New()
		if _, err := pool.Exec(ctx, `INSERT INTO safety_incidents(id) VALUES($1)`, item.incidentID); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO safety_incident_subjects(incident_id,owner_did) VALUES($1,$2)`, item.incidentID, owner); err != nil {
			t.Fatal(err)
		}
		key := item.incidentID.String() + "/" + item.id.String()
		if _, err := store.Put(ctx, safetyincident.RestrictedObject{Key: key, Bytes: []byte("benign-" + item.name)}); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO safety_evidence(id,incident_id,object_key,integrity_sha256,content_type,byte_size,preservation_reason,created_by,retention_expires_at,created_at) VALUES($1,$2,$3,decode(repeat('01',32),'hex'),'image/png',10,'approved','admin',$4,$5)`, item.id, item.incidentID, key, now.Add(time.Hour), now.Add(-time.Hour)); err != nil {
			t.Fatal(err)
		}
		if item.activeHold || item.expiredHold || item.outOfScope {
			holdID := uuid.New()
			expires := now.Add(time.Hour)
			if item.expiredHold {
				expires = now.Add(-time.Minute)
			}
			if _, err := pool.Exec(ctx, `INSERT INTO safety_legal_holds(id,incident_id,basis,approved_by,created_by,expires_at,created_at) VALUES($1,$2,'basis','approver','creator',$3,$4)`, holdID, item.incidentID, expires, now.Add(-time.Hour)); err != nil {
				t.Fatal(err)
			}
			if !item.outOfScope {
				if _, err := pool.Exec(ctx, `INSERT INTO safety_legal_hold_evidence(hold_id,evidence_id,created_at) VALUES($1,$2,$3)`, holdID, item.id, now.Add(-time.Hour)); err != nil {
					t.Fatal(err)
				}
			}
		}
	}

	if err := PurgeOwnerSafetyEvidence(ctx, pool, store, owner, now); err != nil {
		t.Fatal(err)
	}
	for _, item := range fixtures {
		var deleted *time.Time
		if err := pool.QueryRow(ctx, `SELECT deleted_at FROM safety_evidence WHERE id=$1`, item.id).Scan(&deleted); err != nil {
			t.Fatal(err)
		}
		if item.activeHold && deleted != nil {
			t.Fatalf("active held evidence %s was deleted", item.name)
		}
		if !item.activeHold && deleted == nil {
			t.Fatalf("eligible evidence %s survived", item.name)
		}
		var retainedOwner *string
		if err := pool.QueryRow(ctx, `SELECT owner_did FROM safety_incident_subjects WHERE incident_id=$1`, item.incidentID).Scan(&retainedOwner); err != nil {
			t.Fatal(err)
		}
		if item.activeHold && (retainedOwner == nil || *retainedOwner != owner.String()) {
			t.Fatalf("active held subject owner %s was not preserved", item.name)
		}
		if !item.activeHold && retainedOwner != nil {
			t.Fatalf("eligible subject owner %s survived account deletion", item.name)
		}
	}
	if store.ObjectCount() != 1 {
		t.Fatalf("objects=%d, want only active-held object", store.ObjectCount())
	}
}

func TestAccountDeletionSerializesEvidenceDeletionAgainstHoldCreation(t *testing.T) {
	pool := testdb.WithSchema(t, `
		CREATE TABLE safety_incidents(id UUID PRIMARY KEY);
		CREATE TABLE safety_incident_subjects(
			incident_id UUID NOT NULL REFERENCES safety_incidents(id) ON DELETE CASCADE,
			owner_did TEXT
		);
	`)
	applyMigrationFile(t, pool, "../../migrations/000075_safety_evidence_holds.up.sql")
	ctx := context.Background()
	now := time.Date(2030, 7, 8, 9, 0, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:serializedelete")
	incidentID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO safety_incidents(id) VALUES($1)`, incidentID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO safety_incident_subjects(incident_id,owner_did) VALUES($1,$2)`, incidentID, owner); err != nil {
		t.Fatal(err)
	}
	objects := &blockingEvidenceStore{
		MemoryEvidenceStore: safetyincident.NewMemoryEvidenceStore(),
		started:             make(chan struct{}),
		release:             make(chan struct{}),
	}
	admin := safetyincident.Actor{ID: "admin", Role: safetyincident.RoleSafetyAdministrator}
	service := safetyincident.NewEvidenceService(pool, objects, func() time.Time { return now.Add(-time.Hour) })
	value := []byte("benign-race-evidence")
	evidence, err := service.Preserve(ctx, admin, safetyincident.PreserveCommand{
		IncidentID: incidentID, Bytes: value, SHA256: sha256.Sum256(value), ContentType: "image/png",
		Reason: "approved", ExpiresAt: now.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	purgeDone := make(chan error, 1)
	go func() { purgeDone <- PurgeOwnerSafetyEvidence(ctx, pool, objects, owner, now) }()
	<-objects.started
	holdDone := make(chan error, 1)
	go func() {
		_, holdErr := safetyincident.NewHoldService(pool).Create(ctx, admin, safetyincident.HoldRequest{
			IncidentID: incidentID, EvidenceIDs: []uuid.UUID{evidence.ID}, Basis: "legal",
			ApprovedBy: "approver", CreatedBy: admin.ID, CreatedAt: now, ExpiresAt: now.Add(time.Hour),
		})
		holdDone <- holdErr
	}()
	select {
	case err := <-holdDone:
		t.Fatalf("hold completed while deletion held the evidence lock: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	close(objects.release)
	if err := <-purgeDone; err != nil {
		t.Fatal(err)
	}
	if err := <-holdDone; !errors.Is(err, safetyincident.ErrInvalidHold) {
		t.Fatalf("hold after deletion error=%v", err)
	}
}

type blockingEvidenceStore struct {
	*safetyincident.MemoryEvidenceStore
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func (store *blockingEvidenceStore) Delete(ctx context.Context, ref safetyincident.ObjectRef) error {
	store.once.Do(func() { close(store.started) })
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-store.release:
	}
	return store.MemoryEvidenceStore.Delete(ctx, ref)
}

func applyMigrationFile(t *testing.T, pool *pgxpool.Pool, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), string(data)); err != nil {
		t.Fatal(err)
	}
}
