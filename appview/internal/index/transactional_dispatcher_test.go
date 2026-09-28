package index

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/ingestion"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/sourcevalidation"
	"social.craftsky/appview/internal/tap"
)

type transactionalIndexerFunc func(context.Context, pgx.Tx, tap.Event) (tap.Outcome, error)

func (fn transactionalIndexerFunc) Project(ctx context.Context, tx pgx.Tx, source ingestion.SourceRecord) (tap.Outcome, error) {
	return fn(ctx, tx, eventFromSource(source))
}

type sourceCapturingIndexer struct {
	source ingestion.SourceRecord
}

func (indexer *sourceCapturingIndexer) Project(_ context.Context, _ pgx.Tx, source ingestion.SourceRecord) (tap.Outcome, error) {
	indexer.source = source
	return tap.Applied(), nil
}

func TestTransactionalDispatcherRegisterRejectsNilIndexer(t *testing.T) {
	dispatcher := NewTransactionalDispatcher()
	assertPanicsWith(t, "indexer must not be nil", func() {
		dispatcher.Register("social.craftsky.test.nil", nil)
	})
	if collections := dispatcher.Collections(); len(collections) != 0 {
		t.Fatalf("Collections() = %v after rejected registration", collections)
	}
}

func TestTransactionalDispatcherRegisterRejectsDuplicateWithoutReplacingOriginal(t *testing.T) {
	dispatcher := NewTransactionalDispatcher()
	collection := syntax.NSID("app.bsky.actor.profile")
	original := transactionalIndexerFunc(func(context.Context, pgx.Tx, tap.Event) (tap.Outcome, error) {
		return tap.Applied(), nil
	})
	replacement := transactionalIndexerFunc(func(context.Context, pgx.Tx, tap.Event) (tap.Outcome, error) {
		return tap.PermanentInvalid(tap.ReasonMalformedRecord), nil
	})
	dispatcher.Register(collection, original)

	assertPanicsWith(t, collection.String(), func() {
		dispatcher.Register(collection, replacement)
	})
	outcome, err := dispatcher.Project(context.Background(), nil, ingestion.SourceRecord{
		URI: "at://did:plc:actor/app.bsky.actor.profile/self",
		DID: "did:plc:actor", Collection: collection, Rkey: "self",
		Action: "create", Record: json.RawMessage(`{"displayName":"Actor"}`),
	})
	if err != nil || outcome.Kind != tap.OutcomeApplied {
		t.Fatalf("original registration was replaced: outcome=%+v err=%v", outcome, err)
	}
}

func TestTransactionalDispatcherCollectionsAreSortedSnapshotOfRegistrations(t *testing.T) {
	dispatcher := NewTransactionalDispatcher()
	registered := []syntax.NSID{
		"social.craftsky.feed.repost",
		"app.bsky.actor.profile",
		"social.craftsky.feed.like",
	}
	for _, collection := range registered {
		dispatcher.Register(collection, transactionalIndexerFunc(func(context.Context, pgx.Tx, tap.Event) (tap.Outcome, error) {
			return tap.Applied(), nil
		}))
	}

	want := []syntax.NSID{
		"app.bsky.actor.profile",
		"social.craftsky.feed.like",
		"social.craftsky.feed.repost",
	}
	got := dispatcher.Collections()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Collections() = %v, want %v", got, want)
	}
	got[0] = "social.craftsky.test.mutated"
	if next := dispatcher.Collections(); !reflect.DeepEqual(next, want) {
		t.Fatalf("Collections() did not return a snapshot: %v", next)
	}
}

func assertPanicsWith(t *testing.T, want string, fn func()) {
	t.Helper()
	defer func() {
		value := recover()
		if value == nil {
			t.Fatal("call did not panic")
		}
		if message := value.(string); !strings.Contains(message, want) {
			t.Fatalf("panic = %q, want substring %q", message, want)
		}
	}()
	fn()
}

func TestTransactionalDispatcherRejectsMalformedSupportedRecordBeforeMutation(t *testing.T) {
	dispatcher := NewTransactionalDispatcher()
	calledAction := ""
	dispatcher.Register(craftskyLikeNSID, transactionalIndexerFunc(func(_ context.Context, _ pgx.Tx, event tap.Event) (tap.Outcome, error) {
		calledAction = event.Action
		return tap.Applied(), nil
	}))

	outcome, err := dispatcher.Project(context.Background(), nil, ingestion.SourceRecord{
		URI: "at://did:plc:actor/social.craftsky.feed.like/one",
		DID: "did:plc:actor", Collection: craftskyLikeNSID, Rkey: "one",
		SourceEventID: 7, Revision: "3aaaaaaaaaaa2", CID: "bafy-like", Action: "create",
		Record: json.RawMessage(`{"createdAt":"2026-08-14T10:00:00Z"}`),
	})
	if err != nil {
		t.Fatalf("project malformed source: %v", err)
	}
	if outcome.Kind != tap.OutcomePermanentInvalid || outcome.Reason != tap.ReasonMalformedRecord {
		t.Fatalf("outcome=%+v", outcome)
	}
	if calledAction != "" {
		t.Fatalf("invalid record key should not reach serving projector, action=%s", calledAction)
	}
}

func TestTransactionalDispatcherRoutesValidatedSource(t *testing.T) {
	dispatcher := NewTransactionalDispatcher()
	dispatcher.Register(blueskyProfileNSID, transactionalIndexerFunc(func(_ context.Context, _ pgx.Tx, event tap.Event) (tap.Outcome, error) {
		if event.URI != syntax.ATURI("at://did:plc:actor/app.bsky.actor.profile/self") || event.ID != 8 {
			t.Fatalf("event=%+v", event)
		}
		return tap.Applied(), nil
	}))

	outcome, err := dispatcher.Project(context.Background(), nil, ingestion.SourceRecord{
		URI: "at://did:plc:actor/app.bsky.actor.profile/self",
		DID: "did:plc:actor", Collection: blueskyProfileNSID, Rkey: "self",
		SourceEventID: 8, Revision: "3aaaaaaaaaaa3", CID: "bafy-profile", Action: "create",
		Record: json.RawMessage(`{"displayName":"Actor"}`),
	})
	if err != nil || outcome.Kind != tap.OutcomeApplied {
		t.Fatalf("outcome=%+v err=%v", outcome, err)
	}
}

func TestTransactionalDispatcherPreservesDurableSourceMetadata(t *testing.T) {
	dispatcher := NewTransactionalDispatcher()
	indexer := &sourceCapturingIndexer{}
	dispatcher.Register(blueskyProfileNSID, indexer)
	updatedAt := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	source := ingestion.SourceRecord{
		URI: "at://did:plc:actor/app.bsky.actor.profile/self",
		DID: "did:plc:actor", Collection: blueskyProfileNSID, Rkey: "self",
		SourceEventID: 8, Revision: "3aaaaaaaaaaa3", CID: "bafy-profile", Action: "create",
		Record: json.RawMessage(`{"displayName":"Actor"}`), OrderingStatus: "authoritative",
		StructuralValidationStatus: sourcevalidation.Valid, SemanticValidationStatus: sourcevalidation.Valid,
		UpdatedAt: updatedAt,
	}

	outcome, err := dispatcher.Project(context.Background(), nil, source)
	if err != nil || outcome.Kind != tap.OutcomeApplied {
		t.Fatalf("outcome=%+v err=%v", outcome, err)
	}
	if !reflect.DeepEqual(indexer.source, source) {
		t.Fatalf("projected source = %+v, want %+v", indexer.source, source)
	}
}

func TestTransactionalDispatcherRoutesBusinessRecordsWithSourceRevision(t *testing.T) {
	dispatcher := NewTransactionalDispatcher()
	wantRevision := syntax.TID("3mbusinessrev01")
	for _, collection := range []syntax.NSID{businessProfileCollection, businessEventCollection} {
		collection := collection
		dispatcher.Register(collection, transactionalIndexerFunc(func(_ context.Context, _ pgx.Tx, event tap.Event) (tap.Outcome, error) {
			if event.Collection != collection || event.Rev != wantRevision {
				t.Fatalf("event=%+v", event)
			}
			return tap.Applied(), nil
		}))
	}
	records := map[syntax.NSID]struct {
		rkey   syntax.RecordKey
		record json.RawMessage
	}{
		businessProfileCollection: {rkey: "self", record: json.RawMessage(`{"tagline":"Independent","products":[{"title":"Yarn","uri":"https://shop.example/yarn","image":{"image":{"$type":"blob","ref":{"$link":"bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq"},"mimeType":"image/jpeg","size":1024},"aspectRatio":{"width":4,"height":3}}}],"futureExtension":true}`)},
		businessEventCollection:   {rkey: "3meventrecord", record: json.RawMessage(`{"name":"Market","startsAt":"2026-09-10T10:00:00Z","endsAt":"2026-09-10T12:00:00Z","roles":["vendor"],"createdAt":"2026-09-01T00:00:00Z","futureExtension":true}`)},
	}
	for collection, fixture := range records {
		outcome, err := dispatcher.Project(context.Background(), nil, ingestion.SourceRecord{
			URI: syntax.ATURI("at://did:plc:actor/" + collection.String() + "/" + fixture.rkey.String()),
			DID: "did:plc:actor", Collection: collection, Rkey: fixture.rkey,
			Revision: wantRevision, CID: "bafy-business", Action: "create", Record: fixture.record,
		})
		if err != nil || outcome.Kind != tap.OutcomeApplied {
			t.Fatalf("collection=%s outcome=%+v err=%v", collection, outcome, err)
		}
	}
}

func TestTransactionalDispatcherRejectsBusinessRecordsOutsideLexiconContract(t *testing.T) {
	dispatcher := NewTransactionalDispatcher()
	calledAction := ""
	for _, collection := range []syntax.NSID{businessProfileCollection, businessEventCollection} {
		dispatcher.Register(collection, transactionalIndexerFunc(func(_ context.Context, _ pgx.Tx, event tap.Event) (tap.Outcome, error) {
			calledAction = event.Action
			return tap.Applied(), nil
		}))
	}

	cases := []struct {
		name       string
		collection syntax.NSID
		rkey       syntax.RecordKey
		record     json.RawMessage
	}{
		{
			name:       "profile key is not self",
			collection: businessProfileCollection,
			rkey:       "other",
			record:     json.RawMessage(`{"$type":"social.craftsky.business.profile","tagline":"Studio"}`),
		},
		{
			name:       "event key is not a TID",
			collection: businessEventCollection,
			rkey:       "market",
			record:     json.RawMessage(`{"$type":"social.craftsky.business.event","name":"Market","startsAt":"2026-09-10T10:00:00Z","endsAt":"2026-09-10T12:00:00Z","roles":["vendor"],"createdAt":"2026-09-01T00:00:00Z"}`),
		},
		{
			name:       "event omits required roles",
			collection: businessEventCollection,
			rkey:       "3meventrecord",
			record:     json.RawMessage(`{"$type":"social.craftsky.business.event","name":"Market","startsAt":"2026-09-10T10:00:00Z","endsAt":"2026-09-10T12:00:00Z","createdAt":"2026-09-01T00:00:00Z"}`),
		},
		{
			name:       "profile exceeds product maximum",
			collection: businessProfileCollection,
			rkey:       "self",
			record:     profileWithProductCount(t, 21),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calledAction = ""
			outcome, err := dispatcher.Project(context.Background(), nil, ingestion.SourceRecord{
				URI: syntax.ATURI("at://did:plc:actor/" + tc.collection.String() + "/" + tc.rkey.String()),
				DID: "did:plc:actor", Collection: tc.collection, Rkey: tc.rkey,
				Revision: "3mbusinessrev01", CID: "bafy-business", Action: "create", Record: tc.record,
			})
			if err != nil {
				t.Fatalf("project invalid business record: %v", err)
			}
			if outcome.Kind != tap.OutcomePermanentInvalid || outcome.Reason != tap.ReasonMalformedRecord {
				t.Fatalf("outcome=%+v", outcome)
			}
			wantAction := "delete"
			if tc.name == "profile key is not self" || tc.name == "event key is not a TID" {
				wantAction = ""
			}
			if calledAction != wantAction {
				t.Fatalf("invalid business record projection action=%q, want cleanup=%q", calledAction, wantAction)
			}
		})
	}
}

func profileWithProductCount(t *testing.T, count int) json.RawMessage {
	t.Helper()
	products := make([]map[string]any, count)
	for index := range products {
		products[index] = map[string]any{"title": "Kit", "uri": "https://example.com/kit"}
	}
	record, err := json.Marshal(map[string]any{
		"$type":    "social.craftsky.business.profile",
		"products": products,
	})
	if err != nil {
		t.Fatalf("marshal profile fixture: %v", err)
	}
	return record
}

func TestBusinessProjectionLifecycleIsIndependentOfMembership(t *testing.T) {
	actor := syntax.DID("did:plc:business-owner")
	source := ingestion.SourceRecord{DID: actor, Collection: businessProfileCollection, Action: "create"}
	outcome, ready, cleanup := projectionLifecycleReady(source, map[syntax.DID]projectionOwnerRole{actor: projectionActorRole}, nil)
	if !ready || cleanup || outcome.Kind != "" {
		t.Fatalf("missing membership outcome=%+v ready=%v cleanup=%v", outcome, ready, cleanup)
	}
	outcome, ready, cleanup = projectionLifecycleReady(source, map[syntax.DID]projectionOwnerRole{actor: projectionActorRole}, map[syntax.DID]ownerlifecycle.Lifecycle{
		actor: {Owner: actor, State: ownerlifecycle.StateTerminal},
	})
	if ready || cleanup || outcome.Kind != tap.OutcomePermanentInvalid || outcome.Reason != tap.ReasonOwnerTerminal {
		t.Fatalf("terminal owner outcome=%+v ready=%v cleanup=%v", outcome, ready, cleanup)
	}
}

func TestProjectionLifecycleReadyRequiresCleanupForTerminalRelationTarget(t *testing.T) {
	generation := int64(3)
	actor := syntax.DID("did:plc:lifecycle-decision-actor")
	target := syntax.DID("did:plc:lifecycle-decision-target")
	outcome, ready, cleanup := projectionLifecycleReady(
		ingestion.SourceRecord{DID: actor, Action: "update", ProjectionGeneration: &generation},
		map[syntax.DID]projectionOwnerRole{
			actor:  projectionActorRole,
			target: projectionRelationTargetRole,
		},
		map[syntax.DID]ownerlifecycle.Lifecycle{
			actor:  {Owner: actor, State: ownerlifecycle.StateActive, Generation: generation},
			target: {Owner: target, State: ownerlifecycle.StateTerminal, Generation: 8},
		},
	)
	if !ready || !cleanup || outcome.Kind != "" {
		t.Fatalf("outcome=%+v ready=%t cleanup=%t, want authorized cleanup", outcome, ready, cleanup)
	}
}
