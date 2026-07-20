package index_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/relationships"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

const atprotoBlocksDDL = `
CREATE TABLE atproto_blocks (
    uri         TEXT        NOT NULL PRIMARY KEY,
    blocker_did TEXT        NOT NULL,
    rkey        TEXT        NOT NULL,
    cid         TEXT        NOT NULL,
    subject_did TEXT        NOT NULL,
    record      JSONB       NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL,
    indexed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (blocker_did, rkey)
);
CREATE INDEX atproto_blocks_blocker_subject_idx ON atproto_blocks (blocker_did, subject_did);
CREATE INDEX atproto_blocks_subject_blocker_idx ON atproto_blocks (subject_did, blocker_did);
CREATE TABLE notification_events (
	id UUID PRIMARY KEY,
	recipient_did TEXT NOT NULL,
	actor_did TEXT NOT NULL
);
CREATE TABLE push_deliveries (
	notification_id UUID NOT NULL REFERENCES notification_events(id),
	status TEXT NOT NULL,
	lease_owner TEXT,
	lease_expires_at TIMESTAMPTZ,
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
`

func TestBlueskyBlockReconcilesOrderedLifecycleAndDuplicatePairs(t *testing.T) {
	pool := testdb.WithSchema(t, atprotoBlocksDDL)
	idx := index.NewBlueskyBlock(pool)
	ctx := context.Background()

	create := tap.Event{
		URI:        "at://did:plc:alice/app.bsky.graph.block/one",
		CID:        "bafyblock1",
		DID:        "did:plc:alice",
		Rkey:       "one",
		Collection: "app.bsky.graph.block",
		Action:     "create",
		Record: json.RawMessage(`{
			"$type": "app.bsky.graph.block",
			"subject": "did:plc:bob",
			"createdAt": "2026-07-19T12:00:00Z"
		}`),
	}
	if err := idx.Handle(ctx, create); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := idx.Handle(ctx, create); err != nil {
		t.Fatalf("duplicate replay: %v", err)
	}
	assertBlockProjection(t, pool, create.URI.String(), 1, "bafyblock1", "did:plc:bob")

	duplicatePair := create
	duplicatePair.URI = "at://did:plc:alice/app.bsky.graph.block/two"
	duplicatePair.Rkey = "two"
	duplicatePair.CID = "bafyblock2"
	if err := idx.Handle(ctx, duplicatePair); err != nil {
		t.Fatalf("compatible duplicate pair: %v", err)
	}
	var pairCount int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM atproto_blocks
		WHERE blocker_did = 'did:plc:alice' AND subject_did = 'did:plc:bob'
	`).Scan(&pairCount); err != nil {
		t.Fatalf("count duplicate pair: %v", err)
	}
	if pairCount != 2 {
		t.Fatalf("duplicate pair count = %d, want 2 retained public records", pairCount)
	}

	update := create
	update.Action = "update"
	update.CID = "bafyblock3"
	update.Record = json.RawMessage(`{
		"$type": "app.bsky.graph.block",
		"subject": "did:plc:carol",
		"createdAt": "2026-07-19T12:01:00Z"
	}`)
	if err := idx.Handle(ctx, update); err != nil {
		t.Fatalf("update: %v", err)
	}
	assertBlockProjection(t, pool, create.URI.String(), 1, "bafyblock3", "did:plc:carol")

	deleteEvent := tap.Event{
		URI:        create.URI,
		DID:        create.DID,
		Rkey:       create.Rkey,
		Collection: create.Collection,
		Action:     "delete",
	}
	if err := idx.Handle(ctx, deleteEvent); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := idx.Handle(ctx, deleteEvent); err != nil {
		t.Fatalf("duplicate delete replay: %v", err)
	}
	assertBlockProjection(t, pool, create.URI.String(), 0, "", "")
	assertBlockProjection(t, pool, duplicatePair.URI.String(), 1, "bafyblock2", "did:plc:bob")

	for name, record := range map[string]json.RawMessage{
		"invalid subject":   json.RawMessage(`{"subject":"not-a-did","createdAt":"2026-07-19T12:00:00Z"}`),
		"invalid createdAt": json.RawMessage(`{"subject":"did:plc:bob","createdAt":"yesterday"}`),
	} {
		t.Run(name, func(t *testing.T) {
			malformed := create
			malformed.URI = "at://did:plc:alice/app.bsky.graph.block/malformed"
			malformed.Rkey = "malformed"
			malformed.CID = "bafymalformed"
			malformed.Record = record
			if err := idx.Handle(ctx, malformed); err == nil {
				t.Fatal("malformed block succeeded")
			}
			assertBlockProjection(t, pool, malformed.URI.String(), 0, "", "")
		})
	}
}

func TestBlueskyBlockMutualDirectionsRemainIndependentUntilFinalDelete(t *testing.T) {
	pool := testdb.WithSchema(t, `
		CREATE TABLE actor_mutes (
			owner_did TEXT NOT NULL,
			subject_did TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			PRIMARY KEY (owner_did, subject_did)
		);
	`+atprotoBlocksDDL)
	idx := index.NewBlueskyBlock(pool)
	ctx := context.Background()
	alice := syntax.DID("did:plc:alice")
	bob := syntax.DID("did:plc:bob")

	aliceBlocksBob := tap.Event{
		URI:        "at://did:plc:alice/app.bsky.graph.block/alice-block",
		CID:        "bafy-alice",
		DID:        alice,
		Rkey:       "alice-block",
		Collection: "app.bsky.graph.block",
		Action:     "create",
		Record:     json.RawMessage(`{"subject":"did:plc:bob","createdAt":"2026-07-19T12:00:00Z"}`),
	}
	bobBlocksAlice := tap.Event{
		URI:        "at://did:plc:bob/app.bsky.graph.block/bob-block",
		CID:        "bafy-bob",
		DID:        bob,
		Rkey:       "bob-block",
		Collection: "app.bsky.graph.block",
		Action:     "create",
		Record:     json.RawMessage(`{"subject":"did:plc:alice","createdAt":"2026-07-19T12:00:01Z"}`),
	}
	if err := idx.Handle(ctx, aliceBlocksBob); err != nil {
		t.Fatalf("index Alice block: %v", err)
	}
	if err := idx.Handle(ctx, bobBlocksAlice); err != nil {
		t.Fatalf("index Bob block: %v", err)
	}

	deleteAlice := aliceBlocksBob
	deleteAlice.Action = "delete"
	deleteAlice.Record = nil
	if err := idx.Handle(ctx, deleteAlice); err != nil {
		t.Fatalf("delete Alice block: %v", err)
	}
	store := relationships.NewStore(pool)
	aliceState, err := store.State(ctx, alice, bob)
	if err != nil {
		t.Fatalf("Alice state after own delete: %v", err)
	}
	bobState, err := store.State(ctx, bob, alice)
	if err != nil {
		t.Fatalf("Bob state after Alice delete: %v", err)
	}
	if aliceState.Blocking || !aliceState.BlockedBy || !bobState.Blocking || bobState.BlockedBy {
		t.Fatalf("states after one direction deleted = Alice %+v, Bob %+v", aliceState, bobState)
	}

	deleteBob := bobBlocksAlice
	deleteBob.Action = "delete"
	deleteBob.Record = nil
	if err := idx.Handle(ctx, deleteBob); err != nil {
		t.Fatalf("delete Bob block: %v", err)
	}
	aliceState, err = store.State(ctx, alice, bob)
	if err != nil {
		t.Fatalf("Alice final state: %v", err)
	}
	bobState, err = store.State(ctx, bob, alice)
	if err != nil {
		t.Fatalf("Bob final state: %v", err)
	}
	if aliceState.HasBlock() || bobState.HasBlock() {
		t.Fatalf("final states = Alice %+v, Bob %+v; want no block", aliceState, bobState)
	}
}

func TestBlueskyBlockRetainsCurrentOwnerRecordForAbsentSubject(t *testing.T) {
	pool := testdb.WithSchema(t, `
		CREATE TABLE craftsky_profiles (
			did TEXT PRIMARY KEY,
			record_cid TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			indexed_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
	`+atprotoBlocksDDL)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_profiles (did, record_cid)
		VALUES ('did:plc:alice', 'alice-cid')
	`); err != nil {
		t.Fatalf("insert owner profile: %v", err)
	}

	ev := tap.Event{
		URI:        "at://did:plc:alice/app.bsky.graph.block/absent",
		CID:        "bafyabsent",
		DID:        "did:plc:alice",
		Rkey:       "absent",
		Collection: "app.bsky.graph.block",
		Action:     "create",
		Record:     json.RawMessage(`{"subject":"did:plc:bob","createdAt":"2026-07-19T12:00:00Z"}`),
	}
	if err := index.NewBlueskyBlock(pool).Handle(ctx, ev); err != nil {
		t.Fatalf("index absent-subject block: %v", err)
	}
	assertBlockProjection(t, pool, ev.URI.String(), 1, "bafyabsent", "did:plc:bob")

	store := relationships.NewStore(pool)
	items, more, err := store.ListBlocks(ctx, "did:plc:alice", 100, time.Time{}, "")
	if err != nil {
		t.Fatalf("list blocks while subject absent: %v", err)
	}
	if len(items) != 0 || more {
		t.Fatalf("absent subject surfaced in block list: %+v more=%v", items, more)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_profiles (did, record_cid)
		VALUES ('did:plc:bob', 'bob-cid')
	`); err != nil {
		t.Fatalf("insert joining subject: %v", err)
	}
	items, more, err = store.ListBlocks(ctx, "did:plc:alice", 100, time.Time{}, "")
	if err != nil {
		t.Fatalf("list blocks after subject joins: %v", err)
	}
	if len(items) != 1 || items[0].SubjectDID != "did:plc:bob" || more {
		t.Fatalf("retained block did not reactivate with membership: %+v more=%v", items, more)
	}
}

func TestBlueskyBlockEmitsBoundedFailureAndLagOutcomes(t *testing.T) {
	pool := testdb.WithSchema(t, atprotoBlocksDDL)
	observer := &recordingRelationshipObserver{}
	idx := index.NewBlueskyBlock(pool, observer)
	ctx := context.Background()
	sentinelURI := "at://did:plc:private-sentinel/app.bsky.graph.block/private-rkey"
	malformed := tap.Event{
		URI: syntax.ATURI(sentinelURI), DID: "did:plc:private-sentinel", Rkey: "private-rkey",
		CID: "private-cid", Collection: "app.bsky.graph.block", Action: "create",
		Record: json.RawMessage(`{"subject":"not-a-did","createdAt":"2026-07-19T12:00:00Z"}`),
	}
	if err := idx.Handle(ctx, malformed); err == nil {
		t.Fatal("malformed block succeeded")
	}
	valid := malformed
	valid.URI = "at://did:plc:alice/app.bsky.graph.block/valid"
	valid.DID = "did:plc:alice"
	valid.Rkey = "valid"
	valid.CID = "valid-cid"
	valid.Record = json.RawMessage(`{"subject":"did:plc:bob","createdAt":"2026-07-19T12:00:00Z"}`)
	if _, err := pool.Exec(ctx, `
		INSERT INTO notification_events(id,recipient_did,actor_did)
		VALUES('00000000-0000-0000-0000-000000000001','did:plc:alice','did:plc:bob');
		INSERT INTO push_deliveries(notification_id,status)
		VALUES('00000000-0000-0000-0000-000000000001','pending');
	`); err != nil {
		t.Fatal(err)
	}
	if err := idx.Handle(ctx, valid); err != nil {
		t.Fatalf("valid block: %v", err)
	}

	if !observer.has("index_create", "validate", "error", "validation") {
		t.Fatalf("missing validation outcome: %+v", observer.calls)
	}
	if !observer.has("index_create", "lag", "success", "none") {
		t.Fatalf("missing lag outcome: %+v", observer.calls)
	}
	if !observer.has("push_cancellation", "delivery", "some", "none") {
		t.Fatalf("missing push cancellation outcome: %+v", observer.calls)
	}
	for _, call := range observer.calls {
		if strings.Contains(call.operation+call.stage+call.result+call.errorClass, "private-sentinel") ||
			strings.Contains(call.operation+call.stage+call.result+call.errorClass, "private-rkey") {
			t.Fatalf("identifier leaked into observation: %+v", call)
		}
	}
}

type relationshipObservation struct {
	operation, stage, result, errorClass string
	duration                             time.Duration
}

type recordingRelationshipObserver struct {
	calls []relationshipObservation
}

func (o *recordingRelationshipObserver) ObserveRelationship(operation, result string, duration time.Duration) {
	o.calls = append(o.calls, relationshipObservation{operation: operation, stage: "complete", result: result, duration: duration})
}

func (o *recordingRelationshipObserver) ObserveRelationshipOutcome(operation, stage, result, errorClass string, duration time.Duration) {
	o.calls = append(o.calls, relationshipObservation{operation: operation, stage: stage, result: result, errorClass: errorClass, duration: duration})
}

func (o *recordingRelationshipObserver) has(operation, stage, result, errorClass string) bool {
	for _, call := range o.calls {
		if call.operation == operation && call.stage == stage && call.result == result && call.errorClass == errorClass {
			return true
		}
	}
	return false
}

func assertBlockProjection(t *testing.T, pool *pgxpool.Pool, uri string, wantCount int, wantCID, wantSubject string) {
	t.Helper()
	var count int
	var cid, subject *string
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*), max(cid), max(subject_did)
		FROM atproto_blocks WHERE uri = $1
	`, uri).Scan(&count, &cid, &subject); err != nil {
		t.Fatalf("read projection %s: %v", uri, err)
	}
	if count != wantCount {
		t.Fatalf("projection %s count = %d, want %d", uri, count, wantCount)
	}
	if wantCount > 0 && (cid == nil || subject == nil || *cid != wantCID || *subject != wantSubject) {
		t.Fatalf("projection %s cid/subject = %v/%v, want %s/%s", uri, cid, subject, wantCID, wantSubject)
	}
}
