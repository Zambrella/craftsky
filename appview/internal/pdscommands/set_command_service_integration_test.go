package pdscommands

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ingestion"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/testdb"
)

func TestSetCommandServiceReconcilesLostCreateAndRemovesEveryMatchingRecord(t *testing.T) {
	ctx := context.Background()
	pool := testdb.WithMigratedSchema(t)
	now := time.Date(2026, 9, 23, 21, 0, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:set-command-owner")
	target := syntax.DID("did:plc:set-command-target")
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at
		) VALUES($1,'active',3,1,'test',$3,$3,$3),($2,'active',5,1,'test',$3,$3,$3)
	`, owner, target, now); err != nil {
		t.Fatal(err)
	}
	commandStore, err := NewStore(StoreConfig{Pool: pool, Now: func() time.Time { return now }, NewUUID: uuid.New})
	if err != nil {
		t.Fatal(err)
	}
	fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	lifecycles, err := ownerlifecycle.NewStore(pool, fencer, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	pds := newSetCommandPDS(owner)
	boundary := &setCommandBoundary{client: pds}
	service, err := NewSetCommandService(SetCommandServiceConfig{
		Store: commandStore, Lifecycles: lifecycles,
		NewBoundary: func(context.Context, syntax.DID, string) (auth.ActiveEffectPDSBoundary, error) { return boundary, nil },
		Now:         func() time.Time { return now }, Sleep: func(time.Duration) {},
	})
	if err != nil {
		t.Fatal(err)
	}
	subject := "at://did:plc:set-command-target/social.craftsky.feed.post/one"
	base := SetCommandRequest{
		Owner: owner, OwnerGeneration: 3, Target: target, TargetGeneration: 5, SessionID: "session",
		Collection: "social.craftsky.feed.like", Intent: json.RawMessage(`{"subject":"` + subject + `"}`),
		Matches: func(record AuthoritativeRecord) bool {
			var value struct {
				Subject string `json:"subject"`
			}
			return json.Unmarshal(record.Record, &value) == nil && value.Subject == subject
		},
		CreateRecord: func(createdAt time.Time) (json.RawMessage, error) {
			return json.Marshal(map[string]any{"subject": subject, "createdAt": createdAt.Format(time.RFC3339Nano)})
		},
		AcceptedPresent: func(record AuthoritativeRecord, created bool) (TerminalResult, error) {
			status := 200
			if created {
				status = 201
			}
			body, _ := json.Marshal(map[string]string{"uri": record.URI.String(), "cid": record.CID.String()})
			return TerminalResult{State: CommandAccepted, HTTPStatus: status, ResponseBody: body}, nil
		},
		AcceptedAbsent: func() TerminalResult { return TerminalResult{State: CommandAccepted, HTTPStatus: 204} },
		Rejected:       setCommandRejectedResult,
	}

	pds.loseNextResponse = true
	like := base
	like.OperationKind = "like"
	like.OperationKey = uuid.MustParse("018f4d5c-7a61-7d40-a1a2-111111111111")
	like.SelectedRkey = "3aaaaaaaaaaa1"
	like.DesiredActive = true
	first, err := service.Execute(ctx, like)
	if err != nil {
		t.Fatal(err)
	}
	if first.State != CommandAmbiguous || first.RetryAfterSeconds != 1 || len(pds.records) != 1 {
		t.Fatalf("lost create result=%+v records=%d", first, len(pds.records))
	}
	second, err := service.Execute(ctx, like)
	if err != nil {
		t.Fatal(err)
	}
	if second.State != CommandAccepted || second.HTTPStatus != 201 || pds.applyCalls != 1 {
		t.Fatalf("reconciled create result=%+v applyCalls=%d", second, pds.applyCalls)
	}
	replayed, err := service.Execute(ctx, like)
	if err != nil || replayed.HTTPStatus != 201 || pds.applyCalls != 1 {
		t.Fatalf("accepted replay result=%+v err=%v applyCalls=%d", replayed, err, pds.applyCalls)
	}
	changedTargetGeneration := like
	changedTargetGeneration.TargetGeneration = 4
	if _, err := service.Execute(ctx, changedTargetGeneration); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("changed target generation error = %v, want idempotency conflict", err)
	}

	pds.putRecord("3aaaaaaaaaaa2", map[string]any{"subject": subject, "createdAt": now.Format(time.RFC3339Nano)})
	pds.putRecord("3aaaaaaaaaaa3", map[string]any{"subject": "at://did:plc:other/social.craftsky.feed.post/other"})
	unlike := base
	unlike.OperationKind = "unlike"
	unlike.OperationKey = uuid.MustParse("018f4d5c-7a61-7d40-a1a2-222222222222")
	unlike.CreateRecord = nil
	unlike.DesiredActive = false
	deleted, err := service.Execute(ctx, unlike)
	if err != nil {
		t.Fatal(err)
	}
	if deleted.State != CommandAccepted || deleted.HTTPStatus != 204 {
		t.Fatalf("unlike result = %+v", deleted)
	}
	if len(pds.lastWrites) != 2 || pds.lastWrites[0].URI > pds.lastWrites[1].URI {
		t.Fatalf("unlike writes = %+v, want two URI-sorted deletes", pds.lastWrites)
	}
	if len(pds.records) != 1 {
		t.Fatalf("records after unlike = %d, want unmatched record only", len(pds.records))
	}
	if boundary.expected[0] != (ownerlifecycle.ExpectedOwner{Owner: owner, Generation: 3}) ||
		boundary.expected[1] != (ownerlifecycle.ExpectedOwner{Owner: target, Generation: 5}) {
		t.Fatalf("lifecycle fence = %+v", boundary.expected)
	}

	applyCalls := pds.applyCalls
	stale := base
	stale.OperationKind = "stale-target-generation"
	stale.OperationKey = uuid.MustParse("018f4d5c-7a61-7d40-a1a2-333333333333")
	stale.SelectedRkey = "3aaaaaaaaaaa4"
	stale.DesiredActive = true
	stale.TargetGeneration = 4
	result, err := service.Execute(ctx, stale)
	if err != nil {
		t.Fatal(err)
	}
	if result.State != CommandRejected || pds.applyCalls != applyCalls {
		t.Fatalf("stale target result=%+v applyCalls=%d, want rejected before dispatch at %d", result, pds.applyCalls, applyCalls)
	}

	missing := base
	missing.Target = "did:plc:set-command-missing-target"
	missing.TargetGeneration = 1
	missing.OperationKind = "missing-target-generation"
	missing.OperationKey = uuid.MustParse("018f4d5c-7a61-7d40-a1a2-444444444444")
	missing.SelectedRkey = "3aaaaaaaaaaa5"
	missing.DesiredActive = true
	result, err = service.Execute(ctx, missing)
	if err != nil {
		t.Fatal(err)
	}
	if result.State != CommandRejected || pds.applyCalls != applyCalls {
		t.Fatalf("missing target result=%+v applyCalls=%d, want rejected before dispatch at %d", result, pds.applyCalls, applyCalls)
	}
}

func TestAmbiguousSetCreateDoesNotRecreateAfterExternalDelete(t *testing.T) {
	ctx := context.Background()
	pool := testdb.WithMigratedSchema(t)
	now := time.Date(2026, 9, 24, 21, 0, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:lost-set-create")
	if _, err := pool.Exec(ctx, `INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at) VALUES($1,'active',1,1,'test',$2,$2,$2)`, owner, now); err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(StoreConfig{Pool: pool, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	lifecycles, err := ownerlifecycle.NewStore(pool, fencer, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	pds := newSetCommandPDS(owner)
	service, err := NewSetCommandService(SetCommandServiceConfig{Store: store, Lifecycles: lifecycles, NewBoundary: func(context.Context, syntax.DID, string) (auth.ActiveEffectPDSBoundary, error) {
		return &setCommandBoundary{client: pds}, nil
	}, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	const subject = "did:plc:target"
	request := SetCommandRequest{
		Owner: owner, OwnerGeneration: 1, SessionID: "session", OperationKind: "profile.follow", OperationKey: uuid.New(), Collection: "app.bsky.graph.follow", DesiredActive: true, SelectedRkey: "3aaaaaaaaaaa1", Intent: json.RawMessage(`{"target":"did:plc:target"}`),
		Matches: func(record AuthoritativeRecord) bool {
			var value struct {
				Subject string `json:"subject"`
			}
			return json.Unmarshal(record.Record, &value) == nil && value.Subject == subject
		},
		CreateRecord: func(time.Time) (json.RawMessage, error) { return json.RawMessage(`{"subject":"did:plc:target"}`), nil },
		AcceptedPresent: func(AuthoritativeRecord, bool) (TerminalResult, error) {
			return TerminalResult{State: CommandAccepted, HTTPStatus: 200}, nil
		},
		AcceptedAbsent: func() TerminalResult { return TerminalResult{State: CommandAccepted, HTTPStatus: 204} },
		Rejected:       setCommandRejectedResult,
	}
	pds.loseNextResponse = true
	pds.afterApply = func() {
		for uri := range pds.records {
			delete(pds.records, uri)
		}
		pds.head = "bafy-head-external-delete"
		pds.afterApply = nil
	}
	first, err := service.Execute(ctx, request)
	if err != nil || first.State != CommandAmbiguous {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	second, err := service.Execute(ctx, request)
	if err != nil || second.State != CommandAmbiguous {
		t.Fatalf("retry=%+v err=%v", second, err)
	}
	if pds.applyCalls != 1 || len(pds.records) != 0 {
		t.Fatalf("remote calls=%d records=%d; retry recreated externally deleted follow", pds.applyCalls, len(pds.records))
	}
}

type failingSnapshotFallback struct{ calls int }

func (fallback *failingSnapshotFallback) FetchCollection(context.Context, syntax.DID, syntax.NSID) (ingestion.VerifiedRepositorySnapshot, error) {
	fallback.calls++
	return ingestion.VerifiedRepositorySnapshot{}, errors.New("snapshot unavailable")
}

type changingSetHeadPDS struct {
	*setCommandPDS
	calls int
}

func (pds *changingSetHeadPDS) LatestCommit(context.Context, syntax.DID) (syntax.CID, error) {
	pds.calls++
	if pds.calls == 1 {
		return "bafy-before", nil
	}
	return "bafy-after", nil
}

func TestSetCommandUsesConfiguredSnapshotFallbackWhenHeadChanges(t *testing.T) {
	owner := syntax.DID("did:plc:snapshot-fallback-owner")
	fallback := &failingSnapshotFallback{}
	service := &SetCommandService{snapshotFallback: fallback}
	transport, err := NewPDSTransport(&changingSetHeadPDS{setCommandPDS: newSetCommandPDS(owner)})
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = service.readCollection(context.Background(), transport, SetCommandRequest{Owner: owner, Collection: "app.bsky.graph.follow"})
	if err == nil || fallback.calls != 1 {
		t.Fatalf("snapshot fallback calls=%d err=%v", fallback.calls, err)
	}
}

func TestSetCommandResumesKnownInvalidSwapAfterInterruptedRetry(t *testing.T) {
	ctx := context.Background()
	pool := testdb.WithMigratedSchema(t)
	now := time.Date(2026, 9, 24, 22, 0, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:invalid-swap-recovery")
	if _, err := pool.Exec(ctx, `INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at) VALUES($1,'active',1,1,'test',$2,$2,$2)`, owner, now); err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(StoreConfig{Pool: pool, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	lifecycles, err := ownerlifecycle.NewStore(pool, fencer, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	pds := newSetCommandPDS(owner)
	pds.applyErr = auth.ErrRepositorySwapConflict
	crashOnRetry := true
	service, err := NewSetCommandService(SetCommandServiceConfig{
		Store: store, Lifecycles: lifecycles,
		NewBoundary: func(context.Context, syntax.DID, string) (auth.ActiveEffectPDSBoundary, error) {
			return &setCommandBoundary{client: pds}, nil
		},
		Sleep: func(time.Duration) {
			if crashOnRetry {
				panic("restart")
			}
		}, Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	request := SetCommandRequest{
		Owner: owner, OwnerGeneration: 1, SessionID: "session", OperationKind: "profile.follow", OperationKey: uuid.New(), Collection: "app.bsky.graph.follow", DesiredActive: true, SelectedRkey: "3aaaaaaaaaaa2", Intent: json.RawMessage(`{"subject":"did:plc:target"}`),
		Matches: func(record AuthoritativeRecord) bool {
			var value struct {
				Subject string `json:"subject"`
			}
			return json.Unmarshal(record.Record, &value) == nil && value.Subject == "did:plc:target"
		},
		CreateRecord: func(time.Time) (json.RawMessage, error) { return json.RawMessage(`{"subject":"did:plc:target"}`), nil },
		AcceptedPresent: func(AuthoritativeRecord, bool) (TerminalResult, error) {
			return TerminalResult{State: CommandAccepted, HTTPStatus: 200}, nil
		},
		AcceptedAbsent: func() TerminalResult { return TerminalResult{State: CommandAccepted, HTTPStatus: 204} }, Rejected: setCommandRejectedResult,
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Error("expected interruption after recorded InvalidSwap")
			}
		}()
		_, _ = service.Execute(ctx, request)
	}()
	crashOnRetry = false
	pds.applyErr = nil
	result, err := service.Execute(ctx, request)
	if err != nil || result.State != CommandAccepted || pds.applyCalls != 2 {
		t.Fatalf("known swap retry=%+v err=%v calls=%d", result, err, pds.applyCalls)
	}
}

func TestSetCommandLeavesPreparedCommandRetryableWhenLifecycleReadFails(t *testing.T) {
	ctx := context.Background()
	pool := testdb.WithMigratedSchema(t)
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:transient-lifecycle-owner")
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at)
		VALUES($1,'active',1,1,'test',$2,$2,$2)
	`, owner, now); err != nil {
		t.Fatal(err)
	}
	// Keep preparation available while simulating an unavailable lifecycle
	// lookup after the command has been inserted.
	commandStore, err := NewStore(StoreConfig{Pool: pool, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	lifecycles, err := ownerlifecycle.NewStore(pool, fencer, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewSetCommandService(SetCommandServiceConfig{
		Store: commandStore, Lifecycles: lifecycles,
		NewBoundary: func(context.Context, syntax.DID, string) (auth.ActiveEffectPDSBoundary, error) {
			return &setCommandBoundary{client: newSetCommandPDS(owner)}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	request := SetCommandRequest{
		Owner: owner, OwnerGeneration: 1, SessionID: "session", OperationKind: "test.unlike",
		OperationKey: uuid.New(), Collection: "social.craftsky.feed.like", DesiredActive: false,
		Intent:          json.RawMessage(`{"subjectUri":"at://did:plc:other/social.craftsky.feed.post/post1"}`),
		Matches:         func(AuthoritativeRecord) bool { return false },
		AcceptedPresent: func(AuthoritativeRecord, bool) (TerminalResult, error) { return TerminalResult{}, nil },
		AcceptedAbsent:  func() TerminalResult { return TerminalResult{State: CommandAccepted, HTTPStatus: 204} },
		Rejected:        setCommandRejectedResult,
	}
	if _, err := pool.Exec(ctx, `ALTER TABLE owner_lifecycles RENAME TO owner_lifecycles_unavailable`); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Execute(ctx, request); err == nil {
		t.Fatal("unavailable lifecycle lookup must return an error")
	}
	prepared, err := commandStore.LookupCommand(ctx, owner, request.OperationKind, request.OperationKey)
	if err != nil || prepared == nil || prepared.Result.State != CommandPrepared {
		t.Fatalf("command after transient failure = %+v, err=%v", prepared, err)
	}
	if _, err := pool.Exec(ctx, `ALTER TABLE owner_lifecycles_unavailable RENAME TO owner_lifecycles`); err != nil {
		t.Fatal(err)
	}
	// The test pool includes public in its search_path. Drop prepared queries
	// that resolved the missing relation against public during the outage.
	pool.Reset()
	if lifecycle, err := lifecycles.Get(ctx, owner); err != nil {
		t.Fatalf("restored lifecycle unavailable: %+v, %v", lifecycle, err)
	}
	result, err := service.Execute(ctx, request)
	if err != nil || result.State != CommandAccepted || result.HTTPStatus != 204 {
		t.Fatalf("retry after lifecycle recovery = %+v, err=%v", result, err)
	}
}

func setCommandRejectedResult(error) TerminalResult {
	return TerminalResult{State: CommandRejected, HTTPStatus: 409, ResponseBody: json.RawMessage(`{"error":"conflict"}`)}
}

type setCommandBoundary struct {
	client   auth.PDSClient
	expected []ownerlifecycle.ExpectedOwner
}

func (boundary *setCommandBoundary) WithActiveEffects(
	ctx context.Context,
	expected []ownerlifecycle.ExpectedOwner,
	operation auth.ActiveEffectPDSOperation,
) error {
	boundary.expected = append([]ownerlifecycle.ExpectedOwner(nil), expected...)
	return operation(ctx, boundary.client)
}

type setCommandRecord struct {
	cid   syntax.CID
	value any
}

type setCommandPDS struct {
	owner            syntax.DID
	head             syntax.CID
	records          map[syntax.ATURI]setCommandRecord
	applyCalls       int
	applyErr         error
	loseNextResponse bool
	afterApply       func()
	lastWrites       []DispatchStep
}

func newSetCommandPDS(owner syntax.DID) *setCommandPDS {
	return &setCommandPDS{owner: owner, head: "bafy-head-1", records: make(map[syntax.ATURI]setCommandRecord)}
}

func (pds *setCommandPDS) putRecord(rkey string, value any) {
	uri := syntax.ATURI("at://" + pds.owner.String() + "/social.craftsky.feed.like/" + rkey)
	pds.records[uri] = setCommandRecord{cid: syntax.CID("bafy-" + rkey), value: value}
}

func (pds *setCommandPDS) GetRecord(_ context.Context, _ syntax.DID, collection, rkey string, out any) (string, error) {
	uri := syntax.ATURI("at://" + pds.owner.String() + "/" + collection + "/" + rkey)
	record, ok := pds.records[uri]
	if !ok {
		return "", auth.ErrRecordNotFound
	}
	raw, _ := json.Marshal(record.value)
	if err := json.Unmarshal(raw, out); err != nil {
		return "", err
	}
	return record.cid.String(), nil
}

func (*setCommandPDS) PutRecord(context.Context, syntax.DID, string, string, any) error {
	return errors.New("unused")
}
func (*setCommandPDS) CreateRecord(context.Context, syntax.DID, string, any) (syntax.ATURI, syntax.CID, error) {
	return "", "", errors.New("unused")
}
func (*setCommandPDS) DeleteRecord(context.Context, syntax.DID, string, string) error {
	return errors.New("unused")
}
func (*setCommandPDS) UploadBlob(context.Context, string, []byte) (*auth.UploadedBlob, error) {
	return nil, errors.New("unused")
}

func (pds *setCommandPDS) ListRecords(_ context.Context, _ syntax.DID, collection, cursor string, _ int) ([]auth.PDSRecord, string, error) {
	if cursor != "" {
		return nil, "", nil
	}
	var records []auth.PDSRecord
	for uri, record := range pds.records {
		parsed, _ := syntax.ParseATURI(uri.String())
		if parsed.Collection().String() == collection {
			records = append(records, auth.PDSRecord{URI: uri, CID: record.cid, Value: record.value})
		}
	}
	sort.Slice(records, func(i, j int) bool { return records[i].URI < records[j].URI })
	return records, "", nil
}

func (pds *setCommandPDS) LatestCommit(context.Context, syntax.DID) (syntax.CID, error) {
	return pds.head, nil
}

func (pds *setCommandPDS) ApplyWrites(_ context.Context, _ syntax.DID, head syntax.CID, writes []auth.RepositoryWrite) error {
	if head != pds.head {
		return auth.ErrRepositorySwapConflict
	}
	pds.applyCalls++
	if pds.applyErr != nil {
		return pds.applyErr
	}
	pds.lastWrites = make([]DispatchStep, len(writes))
	for index, write := range writes {
		uri := syntax.ATURI("at://" + pds.owner.String() + "/" + write.Collection.String() + "/" + write.RKey.String())
		pds.lastWrites[index] = DispatchStep{Action: write.Action, URI: uri}
		switch write.Action {
		case "create", "update":
			pds.records[uri] = setCommandRecord{cid: syntax.CID("bafy-created-" + write.RKey.String()), value: write.Record}
		case "delete":
			delete(pds.records, uri)
		}
	}
	pds.head = syntax.CID("bafy-head-next")
	if pds.afterApply != nil {
		pds.afterApply()
	}
	if pds.loseNextResponse {
		pds.loseNextResponse = false
		return errors.New("response lost")
	}
	return nil
}

func (pds *setCommandPDS) DeleteRecordWithRepositorySwap(
	context.Context,
	syntax.DID,
	syntax.NSID,
	syntax.RecordKey,
	syntax.CID,
	syntax.CID,
) error {
	return auth.ErrApplyWritesUnsupported
}
