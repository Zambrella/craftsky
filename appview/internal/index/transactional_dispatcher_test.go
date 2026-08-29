package index

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/ingestion"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/tap"
)

type transactionalIndexerFunc func(context.Context, pgx.Tx, tap.Event) (tap.Outcome, error)

func (fn transactionalIndexerFunc) Project(ctx context.Context, tx pgx.Tx, event tap.Event) (tap.Outcome, error) {
	return fn(ctx, tx, event)
}

func TestTransactionalDispatcherRejectsMalformedSupportedRecordBeforeMutation(t *testing.T) {
	dispatcher := NewTransactionalDispatcher()
	called := false
	dispatcher.Register(craftskyLikeNSID, transactionalIndexerFunc(func(context.Context, pgx.Tx, tap.Event) (tap.Outcome, error) {
		called = true
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
	if called {
		t.Fatal("malformed source reached the serving-table projector")
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
	called := false
	for _, collection := range []syntax.NSID{businessProfileCollection, businessEventCollection} {
		dispatcher.Register(collection, transactionalIndexerFunc(func(context.Context, pgx.Tx, tap.Event) (tap.Outcome, error) {
			called = true
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
			called = false
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
			if called {
				t.Fatal("invalid business record reached projector")
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
