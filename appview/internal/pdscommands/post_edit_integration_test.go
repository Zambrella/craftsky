package pdscommands

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/atdata"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/ipfs/go-cid"
	"github.com/jackc/pgx/v5"
	"github.com/multiformats/go-multihash"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ingestion"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/recordjson"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

// IT-001: an edit uses its durable operation time and updates one existing record.
func TestPostEditUsesDurableCommandTime(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	service, pds, request := postEditCommandFixture(t, &now)
	wantTime := now.Format(time.RFC3339Nano)
	request.BuildPreparedRecord = func(record AuthoritativeRecord, prepared PreparedCommand) (json.RawMessage, error) {
		var value map[string]json.RawMessage
		if err := json.Unmarshal(record.Record, &value); err != nil {
			return nil, err
		}
		value["text"] = json.RawMessage(`"after"`)
		value["editedAt"], _ = json.Marshal(prepared.CreatedAt.UTC().Format(time.RFC3339Nano))
		return json.Marshal(value)
	}
	result, err := service.Put(ctx, request)
	if err != nil || result.State != CommandAccepted {
		t.Fatalf("edit = %+v, %v", result, err)
	}
	var accepted struct{ Text, CreatedAt, EditedAt string }
	if err := json.Unmarshal(result.ResponseBody, &accepted); err != nil {
		t.Fatal(err)
	}
	if accepted.Text != "after" || accepted.CreatedAt != "2026-01-01T00:00:00.000Z" || accepted.EditedAt != wantTime {
		t.Fatalf("accepted content = %s; want edit time %s and unchanged publication", result.ResponseBody, wantTime)
	}
	now = now.Add(time.Hour)
	replay, err := service.Put(ctx, request)
	if err != nil || string(replay.ResponseBody) != string(result.ResponseBody) || pds.applyCalls != 1 || len(pds.records) != 1 {
		t.Fatalf("replay = %+v, %v; calls=%d records=%d", replay, err, pds.applyCalls, len(pds.records))
	}
	if len(pds.lastWrites) != 1 || pds.lastWrites[0].Action != "update" || pds.lastWrites[0].URI != request.URI {
		t.Fatalf("edit writes = %+v", pds.lastWrites)
	}
}

// IT-005 / FR-010: temporary preparation dependencies fail before dispatch,
// leaving the same immutable operation retryable rather than terminally rejected.
func TestPostEditRetriesUnavailablePreparationWithoutRemoteEffect(t *testing.T) {
	now := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	service, pds, request := postEditCommandFixture(t, &now)
	builds := 0
	request.BuildRecord = func(record AuthoritativeRecord) (json.RawMessage, error) {
		builds++
		if builds == 1 {
			return nil, errors.Join(ErrDispatchUnavailable, errors.New("verification dependency unavailable"))
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(record.Record, &fields); err != nil {
			return nil, err
		}
		fields["text"] = json.RawMessage(`"after"`)
		return json.Marshal(fields)
	}
	if _, err := service.Put(context.Background(), request); !errors.Is(err, ErrDispatchUnavailable) || pds.applyCalls != 0 {
		t.Fatalf("temporary preparation became terminal or dispatched: err=%v writes=%d", err, pds.applyCalls)
	}
	result, err := service.Put(context.Background(), request)
	if err != nil || result.State != CommandAccepted || builds != 2 || pds.applyCalls != 1 {
		t.Fatalf("same-key retry failed: result=%+v err=%v builds=%d writes=%d", result, err, builds, pds.applyCalls)
	}
}

func postEditCommandFixture(t *testing.T, now *time.Time) (*AddressedCommandService, *postEditConditionalPDS, AddressedPutCommandRequest) {
	t.Helper()
	ctx := context.Background()
	pool := testdb.WithMigratedSchema(t)
	owner := syntax.DID("did:plc:post-edit-owner")
	if _, err := pool.Exec(ctx, `INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at) VALUES($1,'active',1,1,'test',$2,$2,$2)`, owner, *now); err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(StoreConfig{Pool: pool, Now: func() time.Time { return *now }, NewUUID: uuid.New})
	if err != nil {
		t.Fatal(err)
	}
	fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	lifecycles, err := ownerlifecycle.NewStore(pool, fencer, func() time.Time { return *now })
	if err != nil {
		t.Fatal(err)
	}
	pds := &postEditConditionalPDS{setCommandPDS: newSetCommandPDS(owner), revision: "3aaaaaaaaaaa2"}
	uri := syntax.ATURI("at://did:plc:post-edit-owner/social.craftsky.feed.post/3aaaaaaaaaaa2")
	original := map[string]any{
		"$type": "social.craftsky.feed.post", "text": "before", "sponsored": false, "createdAt": "2026-01-01T00:00:00.000Z",
	}
	originalCID, err := postEditTestRecordCID(original)
	if err != nil {
		t.Fatal(err)
	}
	pds.records[uri] = setCommandRecord{cid: originalCID, value: original}
	pds.head, err = postEditTestRecordCID(map[string]any{"head": 0})
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewAddressedCommandService(AddressedCommandServiceConfig{
		Store: store, Lifecycles: lifecycles, Now: func() time.Time { return *now }, Sleep: func(time.Duration) {},
		NewBoundary: func(context.Context, syntax.DID, string) (auth.ActiveEffectPDSBoundary, error) {
			return &setCommandBoundary{client: pds}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return service, pds, AddressedPutCommandRequest{
		StrictExpectedCID: true,
		Owner:             owner, OwnerGeneration: 1, SessionID: "session", OperationKind: "post.edit", OperationKey: uuid.New(), URI: uri, ExpectedCID: originalCID, Intent: json.RawMessage(`{"changes":{"text":"after"}}`),
		BuildRecord: func(record AuthoritativeRecord) (json.RawMessage, error) { return record.Record, nil },
		Accepted: func(record AuthoritativeRecord) (TerminalResult, error) {
			return TerminalResult{State: CommandAccepted, HTTPStatus: 200, ResponseBody: record.Record}, nil
		},
		Rejected: setCommandRejectedResult,
	}
}

// The edit fixture models the PDS's atomic update/record-CID conditions, not
// only a cached AppView comparison. Generic set-command fixtures remain intact.
type postEditConditionalPDS struct {
	*setCommandPDS
	beforeHead          func()
	conditionalAttempts int
	afterRecord         func()
	omitReceipt         bool
	revision            syntax.TID
}

func (pds *postEditConditionalPDS) GetRecord(_ context.Context, _ syntax.DID, collection, rkey string, out any) (string, error) {
	uri := syntax.ATURI("at://" + pds.owner.String() + "/" + collection + "/" + rkey)
	record, exists := pds.records[uri]
	if !exists {
		return "", auth.ErrRecordNotFound
	}
	raw, err := json.Marshal(record.value)
	if err != nil {
		return "", err
	}
	if pds.afterRecord != nil {
		pds.afterRecord()
	}
	return record.cid.String(), recordjson.Unmarshal(raw, out)
}

func (pds *postEditConditionalPDS) LatestCommit(ctx context.Context, owner syntax.DID) (syntax.CID, error) {
	if pds.beforeHead != nil {
		hook := pds.beforeHead
		pds.beforeHead = nil
		hook()
	}
	return pds.setCommandPDS.LatestCommit(ctx, owner)
}

func (pds *postEditConditionalPDS) LatestCommitMetadata(ctx context.Context, owner syntax.DID) (auth.RepositoryCommit, error) {
	head, err := pds.LatestCommit(ctx, owner)
	return auth.RepositoryCommit{CID: head, Revision: pds.revision}, err
}

func (pds *postEditConditionalPDS) ApplyWritesWithCommit(ctx context.Context, owner syntax.DID, head syntax.CID, writes []auth.RepositoryWrite) (auth.RepositoryWriteReceipt, error) {
	return pds.applyWrites(ctx, owner, head, writes)
}

func (pds *postEditConditionalPDS) ApplyWrites(ctx context.Context, owner syntax.DID, head syntax.CID, writes []auth.RepositoryWrite) error {
	_, err := pds.applyWrites(ctx, owner, head, writes)
	return err
}

func (pds *postEditConditionalPDS) applyWrites(_ context.Context, owner syntax.DID, head syntax.CID, writes []auth.RepositoryWrite) (auth.RepositoryWriteReceipt, error) {
	pds.conditionalAttempts++
	if head != pds.head {
		return auth.RepositoryWriteReceipt{}, auth.ErrRepositorySwapConflict
	}
	for _, write := range writes {
		uri := syntax.ATURI("at://" + owner.String() + "/" + write.Collection.String() + "/" + write.RKey.String())
		current, exists := pds.records[uri]
		if write.Action == "update" && (!exists || write.ExpectedCID == "" || current.cid != write.ExpectedCID) {
			return auth.RepositoryWriteReceipt{}, auth.ErrRepositorySwapConflict
		}
	}
	if pds.applyErr != nil {
		return auth.RepositoryWriteReceipt{}, pds.applyErr
	}
	records := make([]auth.RepositoryRecordResult, len(writes))
	for i, write := range writes {
		uri := syntax.ATURI("at://" + owner.String() + "/" + write.Collection.String() + "/" + write.RKey.String())
		recordCID, err := postEditTestRecordCID(write.Record)
		if err != nil {
			return auth.RepositoryWriteReceipt{}, err
		}
		records[i] = auth.RepositoryRecordResult{URI: uri, CID: recordCID}
	}
	pds.applyCalls++
	pds.lastWrites = make([]DispatchStep, len(writes))
	for i, write := range writes {
		pds.records[records[i].URI] = setCommandRecord{cid: records[i].CID, value: write.Record}
		pds.lastWrites[i] = DispatchStep{Action: write.Action, URI: records[i].URI, ExpectedCID: write.ExpectedCID}
	}
	pds.revision = syntax.NewTIDFromInteger(syntax.TID("3aaaaaaaaaaa2").Integer() + uint64(pds.applyCalls))
	var err error
	pds.head, err = postEditTestRecordCID(map[string]any{"head": pds.applyCalls, "previous": head.String()})
	if err != nil {
		return auth.RepositoryWriteReceipt{}, err
	}
	receipt := auth.RepositoryWriteReceipt{Commit: auth.RepositoryCommit{CID: pds.head, Revision: pds.revision}, Records: records}
	if pds.afterApply != nil {
		pds.afterApply()
	}
	if pds.loseNextResponse {
		pds.loseNextResponse = false
		return auth.RepositoryWriteReceipt{}, errors.New("response lost")
	}
	if pds.omitReceipt {
		return auth.RepositoryWriteReceipt{}, nil
	}
	return receipt, nil
}

func postEditTestRecordCID(value any) (syntax.CID, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	object, err := recordjson.Parse(raw)
	if err != nil {
		return "", err
	}
	encoded, err := atdata.MarshalCBOR(object)
	if err != nil {
		return "", err
	}
	identity, err := (cid.Prefix{Version: 1, Codec: cid.DagCBOR, MhType: multihash.SHA2_256, MhLength: 32}).Sum(encoded)
	return syntax.CID(identity.String()), err
}

// IT-004 / IT-006: even with a freshly fetched repository head, the record's
// own conditional update must reject a concurrent replacement or deletion.
func TestPostEditAtomicConditionPreventsOverwriteOrRecreation(t *testing.T) {
	for _, deleted := range []bool{false, true} {
		name := "concurrent edit"
		if deleted {
			name = "concurrent deletion"
		}
		t.Run(name, func(t *testing.T) {
			now := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
			service, pds, request := postEditCommandFixture(t, &now)
			request.BuildRecord = func(record AuthoritativeRecord) (json.RawMessage, error) {
				var fields map[string]json.RawMessage
				if err := json.Unmarshal(record.Record, &fields); err != nil {
					return nil, err
				}
				fields["text"] = json.RawMessage(`"my edit"`)
				return json.Marshal(fields)
			}
			pds.beforeHead = func() {
				pds.head = "bafy-later-head"
				if deleted {
					delete(pds.records, request.URI)
				} else {
					pds.records[request.URI] = setCommandRecord{cid: "bafy-external", value: map[string]any{"text": "external edit"}}
				}
			}
			var cause error
			request.Rejected = func(err error) TerminalResult { cause = err; return setCommandRejectedResult(err) }
			result, err := service.Put(context.Background(), request)
			if err != nil || result.State != CommandRejected || !errors.Is(cause, ErrRecordConflict) || pds.applyCalls != 0 || pds.conditionalAttempts != 1 {
				t.Fatalf("racing edit=%+v %v cause=%v writes=%d attempts=%d", result, err, cause, pds.applyCalls, pds.conditionalAttempts)
			}
			current, exists := pds.records[request.URI]
			if (deleted && exists) || (!deleted && (!exists || current.cid != "bafy-external")) {
				t.Fatalf("edit overwrote concurrent state: exists=%v record=%+v", exists, current)
			}
		})
	}
}

// IT-004: matching content does not authorize a stale first submission.
func TestPostEditRejectsStaleBaseBeforeBuildingEvenIfContentMatches(t *testing.T) {
	now := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	service, pds, request := postEditCommandFixture(t, &now)
	request.StrictExpectedCID = true
	current := pds.records[request.URI]
	current.cid = "bafy-newer"
	pds.records[request.URI] = current
	built := false
	request.BuildRecord = func(record AuthoritativeRecord) (json.RawMessage, error) { built = true; return record.Record, nil }
	var cause error
	request.Rejected = func(err error) TerminalResult { cause = err; return setCommandRejectedResult(err) }
	result, err := service.Put(context.Background(), request)
	if err != nil || result.State != CommandRejected || !errors.Is(cause, ErrRecordConflict) || built || pds.applyCalls != 0 {
		t.Fatalf("stale edit = %+v, %v; cause=%v built=%v writes=%d", result, err, cause, built, pds.applyCalls)
	}
}

// IT-005: recovery compares the exact dispatched body, never a merge against
// newly read content. A lost response must not silently absorb another edit.
func TestPostEditLostResponseRecoversStoredBodyWithoutRebuilding(t *testing.T) {
	now := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	service, pds, request := postEditCommandFixture(t, &now)
	request.StrictExpectedCID = true
	builds := 0
	request.BuildRecord = func(record AuthoritativeRecord) (json.RawMessage, error) {
		builds++
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(record.Record, &fields); err != nil {
			return nil, err
		}
		fields["text"] = json.RawMessage(`"after"`)
		return json.Marshal(fields)
	}
	var accepted AuthoritativeRecord
	request.Accepted = func(record AuthoritativeRecord) (TerminalResult, error) {
		accepted = record
		return TerminalResult{State: CommandAccepted, HTTPStatus: 200, ResponseBody: record.Record}, nil
	}
	pds.loseNextResponse = true
	first, err := service.Put(context.Background(), request)
	if err != nil || first.State != CommandAmbiguous {
		t.Fatalf("lost response = %+v, %v", first, err)
	}
	now = now.Add(time.Minute)
	recovered, err := service.Put(context.Background(), request)
	if err != nil || recovered.State != CommandAccepted || builds != 1 || pds.applyCalls != 1 || accepted.ObservationCID != pds.head || accepted.ObservationRevision != pds.revision || accepted.Revision != "" || accepted.CommitCID != "" {
		t.Fatalf("recovery = %+v, %v; builds=%d writes=%d accepted=%+v", recovered, err, builds, pds.applyCalls, accepted)
	}
	var outcome string
	var commitCID, commitRevision, observationCID, observationRevision *string
	err = service.store.withTransaction(context.Background(), func(tx pgx.Tx) error {
		return tx.QueryRow(context.Background(), `SELECT d.outcome,d.result_commit_cid,d.result_commit_revision,d.observation_cid,d.observation_revision
		FROM pds_command_dispatches d JOIN pds_commands c ON c.id=d.command_id WHERE c.operation_key=$1`, request.OperationKey).Scan(&outcome, &commitCID, &commitRevision, &observationCID, &observationRevision)
	})
	if err != nil || outcome != "accepted" || commitCID != nil || commitRevision != nil || observationCID == nil || *observationCID != pds.head.String() || observationRevision == nil || *observationRevision != pds.revision.String() {
		t.Fatalf("durable recovery evidence: outcome=%s commit=%v revision=%v observedCID=%v observedRev=%v error=%v", outcome, commitCID, commitRevision, observationCID, observationRevision, err)
	}

}

// IT-005: identical records may have different URIs; the operation key must
// bind the target identity as well as its content/version and changes.
func TestPostEditOperationKeyCannotTargetAnotherURI(t *testing.T) {
	now := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	service, pds, request := postEditCommandFixture(t, &now)
	original := pds.records[request.URI]
	request.BuildRecord = func(record AuthoritativeRecord) (json.RawMessage, error) {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(record.Record, &fields); err != nil {
			return nil, err
		}
		fields["text"] = json.RawMessage(`"after"`)
		return json.Marshal(fields)
	}
	result, err := service.Put(context.Background(), request)
	if err != nil || result.State != CommandAccepted {
		t.Fatalf("first edit=%+v %v", result, err)
	}
	other := request
	other.URI = "at://did:plc:post-edit-owner/social.craftsky.feed.post/3anotherpost"
	pds.records[other.URI] = original
	_, err = service.Put(context.Background(), other)
	if !errors.Is(err, ErrIdempotencyConflict) || pds.applyCalls != 1 || pds.records[other.URI].cid != original.cid {
		t.Fatalf("changed target reused operation: error=%v writes=%d other=%+v", err, pds.applyCalls, pds.records[other.URI])
	}
}

// IT-005: a successful write receipt remains accepted if another client writes
// immediately afterward; a later read must not replace its record/revision.
func TestPostEditAcknowledgedWriteKeepsReceiptWhenCurrentRecordChanges(t *testing.T) {
	now := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	service, pds, request := postEditCommandFixture(t, &now)
	request.BuildRecord = func(record AuthoritativeRecord) (json.RawMessage, error) {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(record.Record, &fields); err != nil {
			return nil, err
		}
		fields["text"] = json.RawMessage(`"my saved edit"`)
		return json.Marshal(fields)
	}
	var expectedCID, expectedCommit syntax.CID
	var expectedRevision syntax.TID
	var accepted AuthoritativeRecord
	request.Accepted = func(record AuthoritativeRecord) (TerminalResult, error) {
		accepted = record
		return TerminalResult{State: CommandAccepted, HTTPStatus: 200, ResponseBody: record.Record}, nil
	}
	pds.afterApply = func() {
		expectedCID = pds.records[request.URI].cid
		expectedCommit = pds.head
		expectedRevision = pds.revision
		pds.records[request.URI] = setCommandRecord{cid: "bafy-external", value: map[string]any{"text": "later external version"}}
		pds.head = "bafy-external-head"
		pds.revision = syntax.NewTIDFromInteger(expectedRevision.Integer() + 1)
	}
	result, err := service.Put(context.Background(), request)
	if err != nil || result.State != CommandAccepted || accepted.CID != expectedCID || accepted.CommitCID != expectedCommit || accepted.Revision != expectedRevision || accepted.ObservationRevision != "" || pds.applyCalls != 1 || pds.records[request.URI].cid != "bafy-external" {
		t.Fatalf("acknowledged edit lost receipt: result=%+v record=%+v error=%v writes=%d", result, accepted, err, pds.applyCalls)
	}
	var baseRevision, commitCID, commitRevision, observationRevision *string
	err = service.store.withTransaction(context.Background(), func(tx pgx.Tx) error {
		return tx.QueryRow(context.Background(), `SELECT d.repository_revision,d.result_commit_cid,to_jsonb(d)->>'result_commit_revision',to_jsonb(d)->>'observation_revision'
		FROM pds_command_dispatches d JOIN pds_commands c ON c.id=d.command_id WHERE c.operation_key=$1`, request.OperationKey).Scan(&baseRevision, &commitCID, &commitRevision, &observationRevision)
	})
	if err != nil || baseRevision == nil || *baseRevision != "3aaaaaaaaaaa2" || commitCID == nil || *commitCID != expectedCommit.String() || commitRevision == nil || *commitRevision != expectedRevision.String() || observationRevision != nil {
		t.Fatalf("durable write evidence: base=%v commit=%v revision=%v observation=%v error=%v", baseRevision, commitCID, commitRevision, observationRevision, err)
	}

}

// IT-005: a moving repository cannot supply a stable recovery bound.
func TestPostEditLostResponseWaitsForStableObservation(t *testing.T) {
	now := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	service, pds, request := postEditCommandFixture(t, &now)
	request.BuildRecord = func(record AuthoritativeRecord) (json.RawMessage, error) {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(record.Record, &fields); err != nil {
			return nil, err
		}
		fields["text"] = json.RawMessage(`"after"`)
		return json.Marshal(fields)
	}
	pds.loseNextResponse = true
	first, err := service.Put(context.Background(), request)
	if err != nil || first.State != CommandAmbiguous {
		t.Fatalf("lost response=%+v %v", first, err)
	}
	reads := 0
	pds.afterRecord = func() {
		reads++
		pds.revision = syntax.NewTIDFromInteger(pds.revision.Integer() + 1)
		pds.head, _ = postEditTestRecordCID(map[string]any{"head": reads + 10})
	}
	result, err := service.Put(context.Background(), request)
	if err != nil || result.State != CommandAmbiguous || reads != 3 || pds.applyCalls != 1 {
		t.Fatalf("unstable recovery=%+v %v reads=%d writes=%d", result, err, reads, pds.applyCalls)
	}
	pds.afterRecord = nil
	result, err = service.Put(context.Background(), request)
	if err != nil || result.State != CommandAccepted || pds.applyCalls != 1 {
		t.Fatalf("stable recovery=%+v %v writes=%d", result, err, pds.applyCalls)
	}
}

// IT-016: an incomplete acknowledgement cannot invent ordering evidence.
func TestPostEditMissingReceiptRemainsRecoverable(t *testing.T) {
	now := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	service, pds, request := postEditCommandFixture(t, &now)
	request.BuildRecord = func(record AuthoritativeRecord) (json.RawMessage, error) {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(record.Record, &fields); err != nil {
			return nil, err
		}
		fields["text"] = json.RawMessage(`"after"`)
		return json.Marshal(fields)
	}
	pds.omitReceipt = true
	first, err := service.Put(context.Background(), request)
	if err != nil || first.State != CommandAmbiguous || pds.applyCalls != 1 {
		t.Fatalf("missing receipt=%+v %v writes=%d", first, err, pds.applyCalls)
	}
	second, err := service.Put(context.Background(), request)
	if err != nil || second.State != CommandAccepted || pds.applyCalls != 1 {
		t.Fatalf("recovery=%+v %v writes=%d", second, err, pds.applyCalls)
	}
}

// IT-010: feature confirmation and command acceptance share one transaction.
func TestPostEditFinalizationRollsBackAndRecoversWithoutAnotherWrite(t *testing.T) {
	now := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	service, pds, request := postEditCommandFixture(t, &now)
	request.BuildRecord = func(record AuthoritativeRecord) (json.RawMessage, error) {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(record.Record, &fields); err != nil {
			return nil, err
		}
		fields["text"] = json.RawMessage(`"after"`)
		return json.Marshal(fields)
	}
	failure := errors.New("finalization failed")
	fail := true
	calls := 0
	request.FinalizeAccepted = func(ctx context.Context, tx pgx.Tx, prepared PreparedCommand, record AuthoritativeRecord) error {
		calls++
		if prepared.ID == uuid.Nil || record.CID == "" {
			t.Fatalf("missing finalization identity: %+v %+v", prepared, record)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO craftsky_profiles(did,record_cid) VALUES('did:plc:atomic-edit-confirmation','confirmed')`); err != nil {
			return err
		}
		if fail {
			return failure
		}
		return nil
	}
	first, err := service.Put(context.Background(), request)
	if !errors.Is(err, failure) || pds.applyCalls != 1 || calls != 1 {
		t.Fatalf("failed finalization=%+v %v writes=%d calls=%d", first, err, pds.applyCalls, calls)
	}
	var profiles int
	var state string
	err = service.store.withTransaction(context.Background(), func(tx pgx.Tx) error {
		return tx.QueryRow(context.Background(), `SELECT state,(SELECT count(*) FROM craftsky_profiles WHERE did='did:plc:atomic-edit-confirmation') FROM pds_commands WHERE operation_key=$1`, request.OperationKey).Scan(&state, &profiles)
	})
	if err != nil || state != "dispatching" || profiles != 0 {
		t.Fatalf("partial local acceptance: state=%s profiles=%d error=%v", state, profiles, err)
	}
	fail = false
	second, err := service.Put(context.Background(), request)
	if err != nil || second.State != CommandAccepted || pds.applyCalls != 1 || calls != 2 {
		t.Fatalf("recovery=%+v %v writes=%d calls=%d", second, err, pds.applyCalls, calls)
	}
	replay, err := service.Put(context.Background(), request)
	if err != nil || replay.State != CommandAccepted || calls != 2 {
		t.Fatalf("replay repeated finalization: %+v %v calls=%d", replay, err, calls)
	}
}

// IT-010: source arrival can precede the PDS acknowledgement (or its loss).
func TestPostEditSourceBeforeAcknowledgementIsConfirmed(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	service, pds, request := postEditCommandFixture(t, &now)
	if _, err := service.store.pool.Exec(ctx, `INSERT INTO craftsky_profiles(did,record_cid) VALUES($1,'profile')`, request.Owner); err != nil {
		t.Fatal(err)
	}
	source, err := ingestion.NewStore(service.store.pool, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	original := pds.records[request.URI]
	raw, _ := json.Marshal(original.value)
	event := tap.Event{ID: 1, URI: request.URI, DID: request.Owner, Collection: "social.craftsky.feed.post", Rkey: request.URI.RecordKey(), CID: original.cid, Rev: pds.revision, Action: "create", Live: true, Record: raw}
	if _, err := source.IngestRecord(ctx, event); err != nil {
		t.Fatal(err)
	}
	request.BuildPreparedRecord = func(record AuthoritativeRecord, prepared PreparedCommand) (json.RawMessage, error) {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(record.Record, &fields); err != nil {
			return nil, err
		}
		fields["text"] = json.RawMessage(`"after"`)
		fields["editedAt"], _ = json.Marshal(prepared.CreatedAt.UTC().Format(time.RFC3339Nano))
		return json.Marshal(fields)
	}
	var observedSource string
	pds.afterApply = func() {
		event.ID = 2
		event.Action = "update"
		event.CID = pds.records[request.URI].cid
		event.Rev = pds.revision
		event.Record, _ = json.Marshal(pds.records[request.URI].value)
		if _, err := source.IngestRecord(ctx, event); err != nil {
			t.Fatal(err)
		}
		if err := service.store.pool.QueryRow(ctx, `SELECT edit_time_source FROM craftsky_post_edit_state WHERE uri=$1`, request.URI).Scan(&observedSource); err != nil {
			t.Fatal(err)
		}
	}
	result, err := service.Put(ctx, request)
	if err != nil || result.State != CommandAccepted || observedSource != "confirmed" {
		t.Fatalf("source raced ack: result=%+v source=%s error=%v", result, observedSource, err)
	}
	// Once exact acceptance is known, restoring that old body at a newer
	// repository revision must not borrow its old local confirmation.
	acceptedRaw := append(json.RawMessage(nil), event.Record...)
	acceptedCID := event.CID
	var external map[string]any
	if err := recordjson.Unmarshal(event.Record, &external); err != nil {
		t.Fatal(err)
	}
	external["text"] = "external version"
	event.Record, _ = json.Marshal(external)
	event.CID, _ = postEditTestRecordCID(external)
	event.Rev = syntax.NewTIDFromInteger(event.Rev.Integer() + 1)
	event.ID++
	if _, err := source.IngestRecord(ctx, event); err != nil {
		t.Fatal(err)
	}
	event.Record = acceptedRaw
	event.CID = acceptedCID
	event.Rev = syntax.NewTIDFromInteger(event.Rev.Integer() + 1)
	event.ID++
	if _, err := source.IngestRecord(ctx, event); err != nil {
		t.Fatal(err)
	}
	if err := service.store.pool.QueryRow(ctx, `SELECT edit_time_source FROM craftsky_post_edit_state WHERE uri=$1`, request.URI).Scan(&observedSource); err != nil {
		t.Fatal(err)
	}
	if observedSource != "reported" {
		t.Fatalf("later restored body reused old confirmation: %s", observedSource)
	}

}
