package index

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/ingestion"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

func TestTransactionalDispatcherConvertsTerminalRelationTargetsToCleanupDeletes(t *testing.T) {
	pool := testdb.WithSchema(t, projectionLifecycleDDL)
	ctx := context.Background()
	actor := syntax.DID("did:plc:projection-actor")
	target := syntax.DID("did:plc:projection-terminal-target")
	seedProjectionLifecycle(t, pool, actor, "active", 4)
	seedProjectionLifecycle(t, pool, target, "terminal", 9)

	tests := []struct {
		name       string
		collection syntax.NSID
		record     json.RawMessage
	}{
		{
			name:       "follow target",
			collection: blueskyFollowNSID,
			record:     json.RawMessage(`{"subject":"did:plc:projection-terminal-target","createdAt":"2026-08-14T10:00:00Z"}`),
		},
		{
			name:       "block target",
			collection: blueskyBlockNSID,
			record:     json.RawMessage(`{"subject":"did:plc:projection-terminal-target","createdAt":"2026-08-14T10:00:00Z"}`),
		},
		{
			name:       "like subject owner",
			collection: craftskyLikeNSID,
			record:     json.RawMessage(`{"subject":{"uri":"at://did:plc:projection-terminal-target/social.craftsky.feed.post/one","cid":"bafy-subject"},"createdAt":"2026-08-14T10:00:00Z"}`),
		},
		{
			name:       "repost subject owner",
			collection: craftskyRepostNSID,
			record:     json.RawMessage(`{"subject":{"uri":"at://did:plc:projection-terminal-target/social.craftsky.feed.post/one","cid":"bafy-subject"},"createdAt":"2026-08-14T10:00:00Z"}`),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dispatcher := NewTransactionalDispatcher()
			var projectedAction string
			dispatcher.Register(test.collection, transactionalIndexerFunc(func(_ context.Context, _ pgx.Tx, event tap.Event) (tap.Outcome, error) {
				projectedAction = event.Action
				return tap.Applied(), nil
			}))
			tx, err := pool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = tx.Rollback(context.Background()) }()
			outcome, err := dispatcher.Project(ctx, tx, projectionLifecycleSource(actor, 4, test.collection, "create", test.record))
			if err != nil {
				t.Fatalf("project: %v", err)
			}
			if outcome.Kind != tap.OutcomeApplied {
				t.Fatalf("outcome=%+v, want applied suppression", outcome)
			}
			if projectedAction != "delete" {
				t.Fatalf("terminal target projected action=%q, want cleanup delete", projectedAction)
			}
		})
	}
}

func TestTransactionalDispatcherAllowsUnknownAndDepartedRelationTargets(t *testing.T) {
	pool := testdb.WithSchema(t, projectionLifecycleDDL)
	ctx := context.Background()
	actor := syntax.DID("did:plc:projection-history-actor")
	departed := syntax.DID("did:plc:projection-departed-target")
	seedProjectionLifecycle(t, pool, actor, "active", 2)
	seedProjectionLifecycle(t, pool, departed, "departed", 6)

	for _, target := range []syntax.DID{departed, "did:plc:projection-unknown-target"} {
		t.Run(target.String(), func(t *testing.T) {
			dispatcher := NewTransactionalDispatcher()
			called := false
			dispatcher.Register(blueskyFollowNSID, transactionalIndexerFunc(func(context.Context, pgx.Tx, tap.Event) (tap.Outcome, error) {
				called = true
				return tap.Applied(), nil
			}))
			record := json.RawMessage(`{"subject":"` + target.String() + `","createdAt":"2026-08-14T10:00:00Z"}`)
			tx, err := pool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			outcome, err := dispatcher.Project(ctx, tx, projectionLifecycleSource(actor, 2, blueskyFollowNSID, "create", record))
			_ = tx.Rollback(context.Background())
			if err != nil || outcome.Kind != tap.OutcomeApplied || !called {
				t.Fatalf("outcome=%+v called=%t err=%v", outcome, called, err)
			}
		})
	}
}

func TestTransactionalDispatcherRechecksActorGenerationAndState(t *testing.T) {
	pool := testdb.WithSchema(t, projectionLifecycleDDL)
	ctx := context.Background()
	active := syntax.DID("did:plc:projection-current-actor")
	departed := syntax.DID("did:plc:projection-departed-actor")
	terminal := syntax.DID("did:plc:projection-terminal-actor")
	seedProjectionLifecycle(t, pool, active, "active", 3)
	seedProjectionLifecycle(t, pool, departed, "departed", 5)
	seedProjectionLifecycle(t, pool, terminal, "terminal", 8)

	profileRecord := json.RawMessage(`{"displayName":"Projector"}`)
	tests := []struct {
		name       string
		actor      syntax.DID
		generation int64
		action     string
		wantKind   tap.OutcomeKind
		wantReason tap.ReasonCode
		wantCalled bool
	}{
		{name: "stale active generation", actor: active, generation: 2, action: "update", wantKind: tap.OutcomeBlocked, wantReason: tap.ReasonSourceOrderUncertain},
		{name: "departed create", actor: departed, generation: 5, action: "create", wantKind: tap.OutcomeBlocked, wantReason: tap.ReasonOwnerDeparted},
		{name: "current departed delete cleanup", actor: departed, generation: 5, action: "delete", wantKind: tap.OutcomeApplied, wantCalled: true},
		{name: "stale departed delete", actor: departed, generation: 4, action: "delete", wantKind: tap.OutcomeBlocked, wantReason: tap.ReasonSourceOrderUncertain},
		{name: "terminal actor", actor: terminal, generation: 8, action: "create", wantKind: tap.OutcomePermanentInvalid, wantReason: tap.ReasonOwnerTerminal},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dispatcher := NewTransactionalDispatcher()
			called := false
			dispatcher.Register(blueskyProfileNSID, transactionalIndexerFunc(func(context.Context, pgx.Tx, tap.Event) (tap.Outcome, error) {
				called = true
				return tap.Applied(), nil
			}))
			record := profileRecord
			if test.action == "delete" {
				record = nil
			}
			tx, err := pool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			outcome, err := dispatcher.Project(ctx, tx, projectionLifecycleSource(test.actor, test.generation, blueskyProfileNSID, test.action, record))
			_ = tx.Rollback(context.Background())
			if err != nil {
				t.Fatalf("project: %v", err)
			}
			if outcome.Kind != test.wantKind || outcome.Reason != test.wantReason || called != test.wantCalled {
				t.Fatalf("outcome=%+v called=%t, want %s/%s called=%t", outcome, called, test.wantKind, test.wantReason, test.wantCalled)
			}
		})
	}
}

func TestTransactionalDispatcherOmitsOnlyTerminalPostMentions(t *testing.T) {
	pool := testdb.WithSchema(t, projectionLifecycleDDL)
	ctx := context.Background()
	actor := syntax.DID("did:plc:projection-post-actor")
	activeMention := syntax.DID("did:plc:projection-active-mention")
	terminalMention := syntax.DID("did:plc:projection-terminal-mention")
	unknownMention := syntax.DID("did:plc:projection-unknown-mention")
	seedProjectionLifecycle(t, pool, actor, "active", 1)
	seedProjectionLifecycle(t, pool, activeMention, "active", 4)
	seedProjectionLifecycle(t, pool, terminalMention, "terminal", 7)

	dispatcher := NewTransactionalDispatcher()
	var got []string
	dispatcher.Register(craftskyPostNSID, transactionalIndexerFunc(func(ctx context.Context, _ pgx.Tx, _ tap.Event) (tap.Outcome, error) {
		got = filterTerminalProjectionMentions(ctx, []string{
			terminalMention.String(), unknownMention.String(), activeMention.String(),
		})
		return tap.Applied(), nil
	}))
	record := json.RawMessage(`{
		"text":"terminal active unknown",
		"createdAt":"2026-08-14T10:00:00Z",
		"facets":[
			{"index":{"byteStart":0,"byteEnd":8},"features":[{"$type":"app.bsky.richtext.facet#mention","did":"did:plc:projection-terminal-mention"}]},
			{"index":{"byteStart":9,"byteEnd":15},"features":[{"$type":"app.bsky.richtext.facet#mention","did":"did:plc:projection-active-mention"}]},
			{"index":{"byteStart":16,"byteEnd":23},"features":[{"$type":"app.bsky.richtext.facet#mention","did":"did:plc:projection-unknown-mention"}]}
		]
	}`)
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := dispatcher.Project(ctx, tx, projectionLifecycleSource(actor, 1, craftskyPostNSID, "create", record))
	_ = tx.Rollback(context.Background())
	if err != nil || outcome.Kind != tap.OutcomeApplied {
		t.Fatalf("outcome=%+v err=%v", outcome, err)
	}
	want := []string{unknownMention.String(), activeMention.String()}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("filtered mentions=%v, want %v", got, want)
	}
}

func projectionLifecycleSource(actor syntax.DID, generation int64, collection syntax.NSID, action string, record json.RawMessage) ingestion.SourceRecord {
	return ingestion.SourceRecord{
		URI:             syntax.ATURI("at://" + actor.String() + "/" + collection.String() + "/projection-test"),
		DID:             actor,
		Collection:      collection,
		Rkey:            "projection-test",
		SourceEventID:   1,
		CID:             "bafy-projection-test",
		Action:          action,
		Record:          record,
		OwnerGeneration: &generation,
	}
}

func seedProjectionLifecycle(t *testing.T, pool *pgxpool.Pool, owner syntax.DID, state string, generation int64) {
	t.Helper()
	terminalAt := "NULL"
	if state == "terminal" {
		terminalAt = "now()"
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,
			transitioned_at,terminal_at,created_at,updated_at
		) VALUES($1,$2,$3,1,'test',now(),`+terminalAt+`,now(),now())
	`, owner, state, generation); err != nil {
		t.Fatal(err)
	}
}

const projectionLifecycleDDL = `
CREATE TABLE owner_lifecycles (
	owner_did TEXT PRIMARY KEY,
	state TEXT NOT NULL,
	generation BIGINT NOT NULL,
	auth_epoch BIGINT NOT NULL,
	transition_reason TEXT NOT NULL,
	transitioned_at TIMESTAMPTZ NOT NULL,
	terminal_at TIMESTAMPTZ,
	purge_completed_at TIMESTAMPTZ,
	created_at TIMESTAMPTZ NOT NULL,
	updated_at TIMESTAMPTZ NOT NULL
);
`
