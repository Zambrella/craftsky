package pdscommands

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/instagram"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/postfence"
	"social.craftsky/appview/internal/postpublication"
)

func TestInstagramImportCreateOnlyIdentityAndConfirmedHistory(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	addressed, pds, base := postEditCommandFixture(t, &now)
	service, err := NewAppendCommandService(AppendCommandServiceConfig{Store: addressed.store, Lifecycles: addressed.lifecycles, NewBoundary: addressed.newBoundary, NewRecordKey: func() (syntax.RecordKey, error) { t.Fatal("import allocated ordinary identity"); return "", nil }, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	const sourceKey = "abcdef0123456789abcdef0123456789"
	rkey, err := instagram.PostImportRecordKey(sourceKey, time.Date(2020, 6, 15, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	history := instagram.NewPostImportStore(addressed.store.pool)
	request := AppendCommandRequest{Owner: base.Owner, OwnerGeneration: 1, SessionID: base.SessionID, OperationKind: "post.create", OperationKey: uuid.New(), Collection: "social.craftsky.feed.post", SelectedRkey: rkey, Intent: json.RawMessage(`{"text":"historical caption"}`),
		BuildRecord: func(time.Time) (json.RawMessage, error) {
			return json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"historical caption","createdAt":"2020-06-15T12:00:00Z","externalImport":{"source":"instagram"},"sponsored":false}`), nil
		},
		Accepted: func(record AuthoritativeRecord) (TerminalResult, error) {
			return TerminalResult{State: CommandAccepted, HTTPStatus: 201, ResponseBody: record.Record}, nil
		}, Rejected: setCommandRejectedResult,
	}
	request.SelectRecordKey = func(ctx context.Context, tx pgx.Tx) (syntax.RecordKey, error) {
		return instagram.ReservePostImport(ctx, tx, base.Owner, sourceKey, rkey, service.newRecordKey)
	}
	request.FinalizeAccepted = func(ctx context.Context, tx pgx.Tx, command PreparedCommand, record AuthoritativeRecord) error {
		return instagram.FinalizePostImport(ctx, tx, base.Owner, instagram.PostImport{SourceKey: sourceKey, URI: record.URI, Kind: "project", OperationKey: request.OperationKey})
	}
	pds.loseNextResponse = true
	result, err := service.Execute(ctx, request)
	if err != nil || result.State != CommandAmbiguous || pds.applyCalls != 1 {
		t.Fatalf("ambiguous=%+v err=%v writes=%d", result, err, pds.applyCalls)
	}
	items, err := history.Lookup(ctx, base.Owner, []string{sourceKey})
	if err != nil || len(items) != 0 {
		t.Fatalf("unconfirmed import marked posted: %+v %v", items, err)
	}
	result, err = service.Execute(ctx, request)
	if err != nil || result.State != CommandAccepted || pds.applyCalls != 1 {
		t.Fatalf("recovery=%+v err=%v writes=%d", result, err, pds.applyCalls)
	}
	items, err = history.Lookup(ctx, base.Owner, []string{sourceKey})
	if err != nil || len(items) != 1 || items[0].Kind != "project" {
		t.Fatalf("confirmed history=%+v %v", items, err)
	}
	other, err := history.Lookup(ctx, "did:plc:other-owner", []string{sourceKey})
	if err != nil || len(other) != 0 {
		t.Fatalf("history crossed accounts=%+v %v", other, err)
	}

	// Another draft or browser can only address the same create-only URI.
	// Changed captions cannot replace it or create a second post.
	changed := request
	changed.OperationKey = uuid.New()
	changed.SelectRecordKey = nil
	changed.Intent = json.RawMessage(`{"text":"edited caption"}`)
	changed.BuildRecord = func(time.Time) (json.RawMessage, error) {
		return json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"edited caption","createdAt":"2020-06-15T12:00:00Z"}`), nil
	}
	result, err = service.Execute(ctx, changed)
	if err != nil || result.State != CommandRejected || pds.applyCalls != 1 {
		t.Fatalf("duplicate changed source=%+v err=%v writes=%d", result, err, pds.applyCalls)
	}
	for _, record := range pds.records {
		encoded, _ := json.Marshal(record.value)
		if strings.Contains(string(encoded), "edited caption") {
			t.Fatal("duplicate replaced imported record")
		}
	}
	// Private source history survives replay-body compaction.
	now = now.Add(31 * 24 * time.Hour)
	if _, err := addressed.store.CompactExpired(ctx, 20); err != nil {
		t.Fatal(err)
	}
	items, err = history.Lookup(ctx, base.Owner, []string{sourceKey})
	if err != nil || len(items) != 1 {
		t.Fatalf("history lost at compaction=%+v %v", items, err)
	}
}

func TestInstagramImportCanBeRecreatedAfterConfirmedDeletion(t *testing.T) {
	for _, scenario := range []struct {
		name          string
		loseResponse  bool
		originalState CommandState
	}{
		{name: "confirmed publication", originalState: CommandAccepted},
		{name: "unconfirmed publication", loseResponse: true, originalState: CommandAmbiguous},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			testInstagramImportRecreation(t, scenario.loseResponse, scenario.originalState)
		})
	}
}

func testInstagramImportRecreation(t *testing.T, loseResponse bool, originalState CommandState) {
	t.Helper()
	ctx := context.Background()
	now := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	addressed, pds, base := postEditCommandFixture(t, &now)
	pool := addressed.store.pool
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_account_links (
			id, owner_did, state, igsid, igsid_digest_version, igsid_digest, username, username_normalized, verified_at
		) VALUES ($1, $2, 'active', 'synthetic-id', 1, $3, 'synthetic.importer', 'synthetic.importer', $4)
	`, uuid.New(), base.Owner, make([]byte, 32), now); err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(StoreConfig{
		Pool: pool, Lifecycles: addressed.lifecycles,
		Now: func() time.Time { return now }, NewUUID: uuid.New,
	})
	if err != nil {
		t.Fatal(err)
	}
	const sourceKey = "abcdef0123456789abcdef0123456789"
	const historicalDate = "2020-06-15T12:00:00Z"
	createdAt, err := time.Parse(time.RFC3339, historicalDate)
	if err != nil {
		t.Fatal(err)
	}
	originalRkey, err := instagram.PostImportRecordKey(sourceKey, createdAt)
	if err != nil {
		t.Fatal(err)
	}
	replacementRkey := syntax.RecordKey(syntax.NewTID(now.UnixMicro(), 0))
	var keyAllocations int
	newRecordKey := func() (syntax.RecordKey, error) {
		keyAllocations++
		return replacementRkey, nil
	}
	boundaryUnavailable := errors.New("PDS boundary unavailable")
	allowDispatch := false
	service, err := NewAppendCommandService(AppendCommandServiceConfig{
		Store: store, Lifecycles: addressed.lifecycles, NewRecordKey: newRecordKey,
		Now: func() time.Time { return now },
		NewBoundary: func(context.Context, syntax.DID, string) (auth.ActiveEffectPDSBoundary, error) {
			if !allowDispatch {
				return nil, boundaryUnavailable
			}
			return instagramImportTestBoundary{lifecycles: addressed.lifecycles, client: pds}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	request := AppendCommandRequest{
		Owner: base.Owner, OwnerGeneration: base.OwnerGeneration, SessionID: base.SessionID,
		OperationKind: "post.create", OperationKey: uuid.New(), Collection: "social.craftsky.feed.post",
		SelectedRkey: originalRkey, Intent: json.RawMessage(`{"text":"historical caption"}`),
		SelectRecordKey: func(ctx context.Context, tx pgx.Tx) (syntax.RecordKey, error) {
			return instagram.ReservePostImport(ctx, tx, base.Owner, sourceKey, originalRkey, newRecordKey)
		},
		CheckPrepared: func(ctx context.Context, tx pgx.Tx, command PreparedCommand) (bool, error) {
			return instagram.PostImportReservationActive(ctx, tx, base.Owner, sourceKey, command.SelectedURI)
		},
		AuthorizeDispatch: func(ctx context.Context, tx pgx.Tx) error {
			return instagram.RequireVerifiedImportAccount(ctx, tx, base.Owner)
		},
		BuildRecord: func(time.Time) (json.RawMessage, error) {
			return json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"historical caption","createdAt":"` + historicalDate + `","externalImport":{"source":"instagram"},"sponsored":false}`), nil
		},
		Accepted: func(record AuthoritativeRecord) (TerminalResult, error) {
			return TerminalResult{State: CommandAccepted, HTTPStatus: 201, ResponseBody: record.Record}, nil
		},
		FinalizeAccepted: func(ctx context.Context, tx pgx.Tx, command PreparedCommand, record AuthoritativeRecord) error {
			if err := postpublication.SeedAcceptedCreate(ctx, tx, record.URI, record.Record); err != nil {
				return err
			}
			return instagram.FinalizePostImport(ctx, tx, base.Owner, instagram.PostImport{
				SourceKey: sourceKey, URI: record.URI, Kind: "standard", OperationKey: command.ScopedKey.OperationKey,
			})
		},
		Rejected: setCommandRejectedResult,
	}

	// Two composers could both pin the original URI before one publishes it.
	stale := request
	stale.OperationKey = uuid.New()
	if _, err := service.Execute(ctx, stale); !errors.Is(err, boundaryUnavailable) {
		t.Fatalf("could not stage the interrupted create: %v", err)
	}
	prepared, err := store.LookupCommand(ctx, base.Owner, stale.OperationKind, stale.OperationKey)
	if err != nil || prepared == nil || prepared.Result.State != CommandPrepared || prepared.SelectedRkey != originalRkey {
		t.Fatalf("interrupted create did not pin the original rkey: %+v %v", prepared, err)
	}
	allowDispatch = true
	pds.beforeHead = func() {
		if _, err := pool.Exec(ctx, `UPDATE instagram_account_links SET conflict_pending=true WHERE owner_did=$1`, base.Owner); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := service.Execute(ctx, request); !errors.Is(err, instagram.ErrInstagramVerificationRequired) || pds.applyCalls != 0 {
		t.Fatalf("conflicting verification permitted dispatch: err=%v writes=%d", err, pds.applyCalls)
	}
	if _, err := pool.Exec(ctx, `UPDATE instagram_account_links SET conflict_pending=false WHERE owner_did=$1`, base.Owner); err != nil {
		t.Fatal(err)
	}
	pds.loseNextResponse = loseResponse
	if result, err := service.Execute(ctx, request); err != nil || result.State != originalState || pds.applyCalls != 1 || keyAllocations != 0 {
		t.Fatalf("original create=%+v err=%v writes=%d newKeys=%d", result, err, pds.applyCalls, keyAllocations)
	}
	originalURI := pds.lastWrites[0].URI
	history := instagram.NewPostImportStore(pool)
	var indexed bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM craftsky_posts WHERE uri=$1)`, originalURI).Scan(&indexed); err != nil || indexed {
		t.Fatalf("fixture unexpectedly indexed the original: %v %v", indexed, err)
	}
	items, err := history.Lookup(ctx, base.Owner, []string{sourceKey})
	if err != nil {
		t.Fatal(err)
	}
	duplicate := request
	duplicate.OperationKey = uuid.New()
	if loseResponse {
		if len(items) != 0 {
			t.Fatalf("unconfirmed import marked posted: %+v", items)
		}
	} else {
		if len(items) != 1 || items[0].URI != originalURI {
			t.Fatalf("accepted but unindexed import must remain posted: %+v", items)
		}
		if _, err := service.Execute(ctx, duplicate); !errors.Is(err, instagram.ErrPostAlreadyImported) || pds.applyCalls != 1 {
			t.Fatalf("missing indexed post allowed another publication: err=%v writes=%d", err, pds.applyCalls)
		}
	}

	deletePublication := func(uri syntax.ATURI) {
		t.Helper()
		delete(pds.records, uri)
		if err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
			if err := postfence.LockTx(ctx, tx, uri); err != nil {
				return err
			}
			return postpublication.Delete(ctx, tx, uri, now)
		}); err != nil {
			t.Fatal(err)
		}
	}
	deletePublication(originalURI)
	items, err = history.Lookup(ctx, base.Owner, []string{sourceKey})
	if err != nil || len(items) != 0 {
		t.Fatalf("confirmed deletion still displayed as posted: %+v %v", items, err)
	}
	if result, err := service.Execute(ctx, stale); err != nil || result.State != CommandRejected || pds.applyCalls != 1 {
		t.Fatalf("old prepared create crossed the deletion fence: %+v err=%v writes=%d", result, err, pds.applyCalls)
	}

	replacement := request
	replacement.OperationKey = uuid.New()
	result, err := service.Execute(ctx, replacement)
	if err != nil || result.State != CommandAccepted || pds.applyCalls != 2 || keyAllocations != 1 {
		t.Fatalf("replacement=%+v err=%v writes=%d newKeys=%d", result, err, pds.applyCalls, keyAllocations)
	}
	replacementURI := pds.lastWrites[0].URI
	if replacementURI == originalURI || replacementURI.RecordKey() != replacementRkey {
		t.Fatalf("reimport did not use the fresh rkey: old=%v new=%v", originalURI, replacementURI)
	}
	var published struct {
		CreatedAt string `json:"createdAt"`
	}
	if err := json.Unmarshal(result.ResponseBody, &published); err != nil || published.CreatedAt != historicalDate {
		t.Fatalf("reimport changed the historical date: %+v %v", published, err)
	}

	// Repeated delivery of the old deletion can only retire the old URI.
	deletePublication(originalURI)
	items, err = history.Lookup(ctx, base.Owner, []string{sourceKey})
	if err != nil || len(items) != 1 || items[0].URI != replacementURI || items[0].OperationKey != replacement.OperationKey {
		t.Fatalf("old deletion affected the replacement: %+v %v", items, err)
	}
	duplicate.OperationKey = uuid.New()
	if _, err := service.Execute(ctx, duplicate); !errors.Is(err, instagram.ErrPostAlreadyImported) || pds.applyCalls != 2 {
		t.Fatalf("old deletion unlocked a duplicate replacement: err=%v writes=%d", err, pds.applyCalls)
	}
}

type instagramImportTestBoundary struct {
	lifecycles *ownerlifecycle.Store
	client     auth.PDSClient
}

func (boundary instagramImportTestBoundary) WithActiveEffects(ctx context.Context, owners []ownerlifecycle.ExpectedOwner, operation auth.ActiveEffectPDSOperation) error {
	return boundary.lifecycles.WithActiveEffects(ctx, owners, func(ctx context.Context) error {
		return operation(ctx, boundary.client)
	})
}
