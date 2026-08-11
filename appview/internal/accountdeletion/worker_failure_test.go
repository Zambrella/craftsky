package accountdeletion

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/testdb"
)

func TestPDSDeletionRestartConvergesAfterUncertainSideEffect(t *testing.T) {
	t.Parallel()

	up, err := os.ReadFile("../../migrations/000037_account_deletion.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, accountDeletionStorePreStateDDL)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, string(up)); err != nil {
		t.Fatal(err)
	}
	owner := syntax.DID("did:plc:alice")
	jobID := uuid.MustParse("10000000-0000-0000-0000-000000000041")
	now := time.Date(2026, 8, 10, 15, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `
		INSERT INTO account_deletion_operations(id,owner_did,state,phase,accepted_at)
		VALUES($1,$2,'active','removingCraftskyRecords',$3)
	`, jobID, owner, now); err != nil {
		t.Fatal(err)
	}
	pds := &uncertainDeletePDS{
		owner:               owner,
		record:              auth.PDSRecord{URI: syntax.ATURI("at://did:plc:alice/social.craftsky.feed.post/uncertain")},
		failAfterSideEffect: true,
	}
	store := NewStore(pool, func() time.Time { return now })
	deleter := NewPDSDeleter(pds, store, 20)
	if _, err := deleter.DeleteAll(ctx, jobID.String(), owner); err == nil {
		t.Fatal("uncertain PDS failure unexpectedly succeeded")
	}

	var deleteRequested bool
	if err := pool.QueryRow(ctx, `
		SELECT delete_requested_at IS NOT NULL FROM account_deletion_expected_records
		WHERE job_id=$1 AND uri=$2
	`, jobID, pds.deletedURI).Scan(&deleteRequested); err != nil {
		t.Fatal(err)
	}
	if !deleteRequested {
		t.Fatal("uncertain delete request was not durably marked")
	}

	restartedStore := NewStore(pool, func() time.Time { return now.Add(time.Minute) })
	restarted := NewPDSDeleter(pds, restartedStore, 20)
	result, err := restarted.DeleteAll(ctx, jobID.String(), owner)
	if err != nil || result.Listed != 0 {
		t.Fatalf("restart convergence = (%+v, %v)", result, err)
	}
	if TerminalSuccessEligible(TerminalGates{ExpectedRecordReceiptsComplete: false}) {
		t.Fatal("an uncertain side effect without an index receipt must not permit terminal success")
	}
}

func TestPDSDeletionDoesNotRunBeforeDeleteRequestMarkerPersists(t *testing.T) {
	t.Parallel()

	up, err := os.ReadFile("../../migrations/000037_account_deletion.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, accountDeletionStorePreStateDDL)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, string(up)); err != nil {
		t.Fatal(err)
	}
	owner := syntax.DID("did:plc:alice")
	jobID := uuid.MustParse("10000000-0000-0000-0000-000000000042")
	now := time.Date(2026, 8, 10, 15, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `
		INSERT INTO account_deletion_operations(id,owner_did,state,phase,accepted_at)
		VALUES($1,$2,'active','removingCraftskyRecords',$3)
	`, jobID, owner, now); err != nil {
		t.Fatal(err)
	}
	pds := &uncertainDeletePDS{
		owner: owner,
		record: auth.PDSRecord{
			URI: syntax.ATURI("at://did:plc:alice/social.craftsky.feed.post/marker"),
		},
	}
	store := NewStore(pool, func() time.Time { return now })
	registrar := &failOnceDeleteRequestMarker{store: store, fail: true}
	deleter := NewPDSDeleter(pds, registrar, 20)
	if _, err := deleter.DeleteAll(ctx, jobID.String(), owner); err == nil {
		t.Fatal("marker persistence failure unexpectedly succeeded")
	}
	if pds.record.URI == "" || pds.deletedURI != "" {
		t.Fatalf("PDS delete ran before marker persisted: record=%q deleted=%q", pds.record.URI, pds.deletedURI)
	}

	if _, err := deleter.DeleteAll(ctx, jobID.String(), owner); err != nil {
		t.Fatalf("retry after marker persistence failure: %v", err)
	}
	if pds.record.URI != "" || pds.deletedURI == "" {
		t.Fatalf("PDS retry did not delete record: record=%q deleted=%q", pds.record.URI, pds.deletedURI)
	}
	var marked bool
	if err := pool.QueryRow(ctx, `
		SELECT delete_requested_at IS NOT NULL
		FROM account_deletion_expected_records WHERE job_id=$1 AND uri=$2
	`, jobID, pds.deletedURI).Scan(&marked); err != nil {
		t.Fatal(err)
	}
	if !marked {
		t.Fatal("successful retry did not retain the durable delete marker")
	}
}

type failOnceDeleteRequestMarker struct {
	store *Store
	fail  bool
}

func (registrar *failOnceDeleteRequestMarker) RegisterExpected(
	ctx context.Context,
	jobID string,
	owner syntax.DID,
	uri syntax.ATURI,
	collection syntax.NSID,
) error {
	return registrar.store.RegisterExpected(ctx, jobID, owner, uri, collection)
}

func (registrar *failOnceDeleteRequestMarker) MarkDeleteRequested(
	ctx context.Context,
	jobID string,
	owner syntax.DID,
	uri syntax.ATURI,
) error {
	if registrar.fail {
		registrar.fail = false
		return errors.New("synthetic marker persistence failure")
	}
	return registrar.store.MarkDeleteRequested(ctx, jobID, owner, uri)
}

type uncertainDeletePDS struct {
	owner               syntax.DID
	record              auth.PDSRecord
	deletedURI          syntax.ATURI
	failAfterSideEffect bool
}

func (pds *uncertainDeletePDS) ListRecords(_ context.Context, repo syntax.DID, collection string, _ string, _ int) ([]auth.PDSRecord, string, error) {
	if repo != pds.owner {
		return nil, "", errors.New("wrong owner")
	}
	if pds.record.URI != "" && pds.record.URI.Collection() == syntax.NSID(collection) {
		return []auth.PDSRecord{pds.record}, "", nil
	}
	return nil, "", nil
}

func (pds *uncertainDeletePDS) DeleteRecord(_ context.Context, _ syntax.DID, _ string, _ string) error {
	pds.deletedURI = pds.record.URI
	pds.record = auth.PDSRecord{}
	if pds.failAfterSideEffect {
		pds.failAfterSideEffect = false
		return errors.New("synthetic connection loss after side effect")
	}
	return nil
}

var _ auth.DeletionPDSClient = (*uncertainDeletePDS)(nil)
