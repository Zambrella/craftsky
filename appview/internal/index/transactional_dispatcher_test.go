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
		SourceEventID: 7, Revision: "3m7", CID: "bafy-like", Action: "create",
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
		SourceEventID: 8, Revision: "3m8", CID: "bafy-profile", Action: "create",
		Record: json.RawMessage(`{"displayName":"Actor"}`),
	})
	if err != nil || outcome.Kind != tap.OutcomeApplied {
		t.Fatalf("outcome=%+v err=%v", outcome, err)
	}
}

func TestProjectionLifecycleReadyRequiresCleanupForTerminalRelationTarget(t *testing.T) {
	generation := int64(3)
	actor := syntax.DID("did:plc:lifecycle-decision-actor")
	target := syntax.DID("did:plc:lifecycle-decision-target")
	outcome, ready, cleanup := projectionLifecycleReady(
		ingestion.SourceRecord{DID: actor, Action: "update", OwnerGeneration: &generation},
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
