package instagram

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/testdb"
)

func TestImportStoreCreatesVerifiedAdditiveSourcesRetainedUntilUnlink(t *testing.T) {
	store, pool := newImportTestStore(t)
	ctx := context.Background()
	owner := syntax.DID("did:plc:synthetic-alice")
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	insertVerifiedImportOwner(t, pool, owner, "synthetic.alice", now)

	first, err := store.CreateImport(ctx, CreateImportParams{
		ID:       uuid.MustParse("00000000-0000-0000-0000-000000000201"),
		OwnerDID: owner, SourceType: ImportSourceInstagramJSON,
		Entries: []ImportEntry{
			{Username: " Alice.Crafts "},
			{Username: "@alice.crafts"},
			{Username: "Bob_9"},
		},
		Now: now,
	})
	if err != nil {
		t.Fatalf("create first: %v", err)
	}
	if first.Import.State != ImportActive || first.Counts.Following != 2 || first.InitialSuggestionCount != 0 {
		t.Fatalf("first result = %+v", first)
	}
	second, err := store.CreateImport(ctx, CreateImportParams{
		ID:       uuid.MustParse("00000000-0000-0000-0000-000000000202"),
		OwnerDID: owner, SourceType: ImportSourceManual,
		Entries: []ImportEntry{{Username: "charlie"}},
		Now:     now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("create second: %v", err)
	}
	if second.Import.ID == first.Import.ID {
		t.Fatal("additive import reused the first source")
	}

	var imports, handles int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM instagram_graph_imports WHERE owner_did = $1`, owner).Scan(&imports); err != nil {
		t.Fatalf("count imports: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM instagram_graph_handles`).Scan(&handles); err != nil {
		t.Fatalf("count handles: %v", err)
	}
	if imports != 2 || handles != 3 {
		t.Fatalf("imports=%d handles=%d, want 2/3", imports, handles)
	}
}

func TestImportStoreRequiresVerificationAndRetainsEveryHandle(t *testing.T) {
	store, pool := newImportTestStore(t)
	ctx := context.Background()
	owner := syntax.DID("did:plc:synthetic-alice")
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)

	params := CreateImportParams{
		ID:       uuid.MustParse("00000000-0000-0000-0000-000000000203"),
		OwnerDID: owner, SourceType: ImportSourceInstagramJSON,
		Entries: []ImportEntry{
			{Username: "unmatched.following"},
			{Username: "private.following"},
		},
		Now: now,
	}
	if _, err := store.CreateImport(ctx, params); !errors.Is(err, ErrInstagramVerificationRequired) {
		t.Fatalf("unverified create error = %v", err)
	}
	insertVerifiedImportOwner(t, pool, owner, "synthetic.alice", now)
	created, err := store.CreateImport(ctx, params)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	var handles int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM instagram_graph_handles WHERE import_id = $1`, created.Import.ID).Scan(&handles); err != nil {
		t.Fatalf("count handles: %v", err)
	}
	if handles != 2 {
		t.Fatalf("verified import retained %d handles, want 2", handles)
	}
}

func TestImportStoreMembershipReactivationAndDeleteArePrivate(t *testing.T) {
	store, pool := newImportTestStore(t)
	ctx := context.Background()
	alice := syntax.DID("did:plc:synthetic-alice")
	bob := syntax.DID("did:plc:synthetic-bob")
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	id := uuid.MustParse("00000000-0000-0000-0000-000000000204")
	insertVerifiedImportOwner(t, pool, alice, "synthetic.alice", now)

	_, err := store.CreateImport(ctx, CreateImportParams{
		ID: id, OwnerDID: alice, SourceType: ImportSourceManual,
		Entries: []ImportEntry{{Username: "synthetic"}},
		Now:     now,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE instagram_graph_imports
		SET state = 'membershipInactive', membership_inactive_at = $2
		WHERE id = $1
	`, id, now.Add(time.Hour)); err != nil {
		t.Fatalf("inactivate: %v", err)
	}
	updated, err := store.UpdateImport(ctx, alice, id, UpdateImportParams{Reactivate: boolPointer(true), Now: now.Add(2 * time.Hour)})
	if err != nil {
		t.Fatalf("reactivate: %v", err)
	}
	if updated.State != ImportActive {
		t.Fatalf("reactivated import = %+v", updated)
	}
	var reconciliationJobs int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM instagram_reconciliation_jobs
		WHERE owner_did=$1 AND import_id=$2
		  AND reason='instagramImportReactivated' AND status='queued'
	`, alice, id).Scan(&reconciliationJobs); err != nil {
		t.Fatalf("inspect import reactivation reconciliation: %v", err)
	}
	if reconciliationJobs != 1 {
		t.Fatalf("import reactivation reconciliation jobs = %d, want 1", reconciliationJobs)
	}

	if err := store.DeleteImport(ctx, bob, id, now.Add(3*time.Hour)); err != nil {
		t.Fatalf("foreign delete: %v", err)
	}
	if _, err := store.GetImport(ctx, alice, id, now.Add(3*time.Hour)); err != nil {
		t.Fatalf("foreign delete removed import: %v", err)
	}
	if err := store.DeleteImport(ctx, alice, id, now.Add(4*time.Hour)); err != nil {
		t.Fatalf("owner delete: %v", err)
	}
	if err := store.DeleteImport(ctx, alice, id, now.Add(5*time.Hour)); err != nil {
		t.Fatalf("delete replay: %v", err)
	}
	if _, err := store.GetImport(ctx, alice, id, now.Add(5*time.Hour)); !errors.Is(err, ErrInstagramResourceNotFound) {
		t.Fatalf("get deleted error = %v", err)
	}
}

func TestImportStoreDeleteRetractsEveryUnsupportedDependent(t *testing.T) {
	store, pool := newImportTestStore(t)
	ctx := context.Background()
	owner := syntax.DID("did:plc:synthetic-import-delete")
	target := syntax.DID("did:plc:synthetic-import-target")
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	importID := uuid.MustParse("00000000-0000-0000-0000-000000000205")
	suggestionID := uuid.MustParse("00000000-0000-0000-0000-000000000206")
	insertVerifiedImportOwner(t, pool, owner, "synthetic.import.delete", now)

	if _, err := store.CreateImport(ctx, CreateImportParams{
		ID: importID, OwnerDID: owner, SourceType: ImportSourceManual,
		Entries: []ImportEntry{{Username: "synthetic.target"}},
		Now:     now,
	}); err != nil {
		t.Fatalf("create import: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_follow_suggestions(
			id,importer_did,target_did,state,reason,accepting_since,created_at,updated_at
		) VALUES($1,$2,$3,'accepting','verifiedInstagramFollow',$4,$4,$4)
	`, suggestionID, owner, target, now); err != nil {
		t.Fatalf("seed import suggestion: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_suggestion_sources(suggestion_id,import_id,created_at)
		VALUES($1,$2,$3)
	`, suggestionID, importID, now); err != nil {
		t.Fatalf("seed import suggestion source: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO pds_follow_operations(
			id,suggestion_id,owner_did,target_did,rkey,status,attempt_count,created_at,updated_at
		) VALUES(
			'00000000-0000-0000-0000-000000000207',$1,$2,$3,
			'3kyimportdelete','writing',1,$4,$4
		)
	`, suggestionID, owner, target, now); err != nil {
		t.Fatalf("seed import follow operation: %v", err)
	}
	seedLifecycleNotification(
		t,
		pool,
		uuid.MustParse("00000000-0000-0000-0000-000000000208"),
		owner,
		suggestionID,
		"00000000-0000-0000-0000-000000000209",
		"pending",
		now,
	)
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_reconciliation_jobs(
			id,owner_did,import_id,reason,status,next_attempt_at,created_at,updated_at
		) VALUES(
			'00000000-0000-0000-0000-00000000020a',$1,$2,
			'syntheticImportDelete','processing',$3,$3,$3
		)
	`, owner, importID, now); err != nil {
		t.Fatalf("seed import reconciliation: %v", err)
	}

	if err := store.DeleteImport(ctx, owner, importID, now.Add(time.Minute)); err != nil {
		t.Fatalf("delete import: %v", err)
	}
	var suggestionState InstagramSuggestionState
	var followStatus, eventState, deliveryStatus, jobStatus string
	if err := pool.QueryRow(ctx, `
		SELECT
			(SELECT state FROM instagram_follow_suggestions WHERE id=$1),
			(SELECT status FROM pds_follow_operations WHERE suggestion_id=$1),
			(SELECT state FROM notification_events WHERE id='00000000-0000-0000-0000-000000000208'),
			(SELECT status FROM push_deliveries WHERE id='00000000-0000-0000-0000-000000000209'),
			(SELECT status FROM instagram_reconciliation_jobs WHERE id='00000000-0000-0000-0000-00000000020a')
	`, suggestionID).Scan(&suggestionState, &followStatus, &eventState, &deliveryStatus, &jobStatus); err != nil {
		t.Fatalf("inspect deleted-import dependents: %v", err)
	}
	if suggestionState != SuggestionInvalidated || followStatus != "failed" || eventState != "retracted" || deliveryStatus != "cancelled" || jobStatus != "ignored" {
		t.Fatalf("dependents suggestion=%s follow=%s event=%s delivery=%s job=%s", suggestionState, followStatus, eventState, deliveryStatus, jobStatus)
	}
}

func TestImportStoreListsOwnedImportsWithStableSeekPagination(t *testing.T) {
	store, _ := newImportTestStore(t)
	ctx := context.Background()
	alice := syntax.DID("did:plc:synthetic-alice")
	bob := syntax.DID("did:plc:synthetic-bob")
	base := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	insertVerifiedImportOwner(t, store.pool, alice, "synthetic.alice", base)
	insertVerifiedImportOwner(t, store.pool, bob, "synthetic.bob", base)
	ids := []uuid.UUID{
		uuid.MustParse("00000000-0000-0000-0000-000000000211"),
		uuid.MustParse("00000000-0000-0000-0000-000000000212"),
		uuid.MustParse("00000000-0000-0000-0000-000000000213"),
	}
	for index, id := range ids {
		if _, err := store.CreateImport(ctx, CreateImportParams{
			ID: id, OwnerDID: alice, SourceType: ImportSourceManual,
			Entries: []ImportEntry{{Username: "synthetic"}},
			Now:     base.Add(time.Duration(index) * time.Minute),
		}); err != nil {
			t.Fatalf("create import %d: %v", index, err)
		}
	}
	if _, err := store.CreateImport(ctx, CreateImportParams{
		ID: uuid.MustParse("00000000-0000-0000-0000-000000000214"), OwnerDID: bob,
		SourceType: ImportSourceManual,
		Entries:    []ImportEntry{{Username: "foreign"}},
		Now:        base.Add(3 * time.Minute),
	}); err != nil {
		t.Fatalf("create foreign import: %v", err)
	}

	first, cursor, err := store.ListImports(ctx, alice, 2, nil, base.Add(4*time.Minute))
	if err != nil {
		t.Fatalf("list first page: %v", err)
	}
	if len(first) != 2 || first[0].ID != ids[2] || first[1].ID != ids[1] || cursor == nil {
		t.Fatalf("first=%+v cursor=%+v", first, cursor)
	}
	second, finalCursor, err := store.ListImports(ctx, alice, 2, cursor, base.Add(4*time.Minute))
	if err != nil {
		t.Fatalf("list second page: %v", err)
	}
	if len(second) != 1 || second[0].ID != ids[0] || finalCursor != nil {
		t.Fatalf("second=%+v cursor=%+v", second, finalCursor)
	}

	foreignCursor := &ImportCursor{CreatedAt: base.Add(3 * time.Minute), ID: uuid.MustParse("00000000-0000-0000-0000-000000000214")}
	if _, _, err := store.ListImports(ctx, alice, 2, foreignCursor, base.Add(4*time.Minute)); !errors.Is(err, ErrInvalidInstagramImportCursor) {
		t.Fatalf("foreign cursor error = %v", err)
	}
}

func newImportTestStore(t *testing.T) (*ImportStore, *pgxpool.Pool) {
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
	pool := testdb.WithSchema(t, migration.String())
	return NewImportStore(pool), pool
}

func boolPointer(value bool) *bool { return &value }

func insertVerifiedImportOwner(
	t *testing.T,
	pool *pgxpool.Pool,
	owner syntax.DID,
	username string,
	now time.Time,
) {
	t.Helper()
	insertAccountLink(t, pool, accountLinkFixture{
		ID:           uuid.New(),
		Owner:        owner,
		State:        LinkActive,
		IGSID:        "igsid-" + uuid.NewString(),
		Username:     username,
		Discoverable: true,
		VerifiedAt:   now,
		UpdatedAt:    now,
	})
}
