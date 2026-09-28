package index

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/ingestion"
	"social.craftsky/appview/internal/notifications"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

func TestProjectSetSourceReplacesValidFactWithLatestValidationState(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 23, 15, 0, 0, 0, time.UTC)
	store, err := ingestion.NewStore(pool, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	actor := syntax.DID("did:plc:fact-version-actor")
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at
		) VALUES($1,'active',1,1,'test',$2,$2,$2)
	`, actor, now); err != nil {
		t.Fatalf("seed active actor: %v", err)
	}
	uri := syntax.ATURI("at://did:plc:fact-version-actor/app.bsky.graph.follow/3aaaaaaaaaaa2")
	event := tap.Event{
		ID: 1, URI: uri, DID: actor, Collection: blueskyFollowNSID, Rkey: "3aaaaaaaaaaa2",
		Rev: "3aaaaaaaaaaa2", CID: "bafy-follow-valid", Action: "create",
		Record: json.RawMessage(`{"subject":"did:plc:target","createdAt":"2026-09-23T14:00:00Z"}`),
	}
	ingestAndProjectSetSource(t, store, pool, event, now)
	assertSetFactCounts(t, pool, uri, 1, 1)

	event.ID = 2
	event.Rev = "3aaaaaaaaaaa3"
	event.CID = "bafy-follow-invalid"
	event.Action = "update"
	event.Record = json.RawMessage(`{"subject":"not-a-did","createdAt":"2026-09-23T14:00:00Z"}`)
	ingestAndProjectSetSource(t, store, pool, event, now.Add(time.Minute))
	assertSetFactCounts(t, pool, uri, 0, 0)

	event.ID = 3
	event.Rev = "3aaaaaaaaaaa4"
	event.CID = "bafy-follow-restored"
	event.Record = json.RawMessage(`{"subject":"did:plc:target","createdAt":"2026-09-23T14:02:00Z"}`)
	ingestAndProjectSetSource(t, store, pool, event, now.Add(2*time.Minute))
	assertSetFactCounts(t, pool, uri, 1, 1)
}

func TestInvalidLatestFollowRemovesPreviousFactThroughProductionDispatcher(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 23, 15, 0, 0, 0, time.UTC)
	store, err := ingestion.NewStore(pool, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	actor := syntax.DID("did:plc:invalid-follow-actor")
	if _, err := pool.Exec(ctx, `INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at) VALUES($1,'active',1,1,'test',$2,$2,$2)`, actor, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles(did,record_cid) VALUES($1,'bafy-profile')`, actor); err != nil {
		t.Fatal(err)
	}
	uri := syntax.ATURI("at://" + actor.String() + "/app.bsky.graph.follow/3aaaaaaaaaaa2")
	event := tap.Event{ID: 1, URI: uri, DID: actor, Collection: blueskyFollowNSID, Rkey: "3aaaaaaaaaaa2", Rev: "3aaaaaaaaaaa2", CID: "bafy-valid", Action: "create", Record: json.RawMessage(`{"$type":"app.bsky.graph.follow","subject":"did:plc:target","createdAt":"2026-09-23T14:00:00Z"}`)}
	dispatcher := NewTransactionalDispatcher()
	dispatcher.Register(blueskyFollowNSID, NewBlueskyFollow(pool, notifications.NewService()))
	project := func() {
		t.Helper()
		if _, err := store.IngestRecord(ctx, event); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `UPDATE tap_source_records SET projection_generation=1 WHERE uri=$1`, uri); err != nil {
			t.Fatal(err)
		}
		source, err := store.Source(ctx, uri)
		if err != nil {
			t.Fatal(err)
		}
		if err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
			_, err := dispatcher.Project(ctx, tx, source)
			return err
		}); err != nil {
			t.Fatal(err)
		}
	}
	project()
	assertSetFactCounts(t, pool, uri, 1, 1)
	event.ID, event.Rev, event.CID, event.Action = 2, "3aaaaaaaaaaa3", "bafy-invalid", "update"
	event.Record = json.RawMessage(`{"$type":"app.bsky.graph.follow","subject":"not-a-did","createdAt":"2026-09-23T14:00:00Z"}`)
	project()
	assertSetFactCounts(t, pool, uri, 0, 0)
}

func TestInvalidLatestBlueskyProfileRemovesPreviousServingRow(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 23, 16, 0, 0, 0, time.UTC)
	store, err := ingestion.NewStore(pool, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	actor := syntax.DID("did:plc:invalid-bluesky-profile")
	if _, err := pool.Exec(ctx, `INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at) VALUES($1,'active',1,1,'test',$2,$2,$2)`, actor, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles(did,record_cid) VALUES($1,'bafy-profile')`, actor); err != nil {
		t.Fatal(err)
	}
	uri := syntax.ATURI("at://" + actor.String() + "/app.bsky.actor.profile/self")
	event := tap.Event{ID: 1, URI: uri, DID: actor, Collection: blueskyProfileNSID, Rkey: "self", Rev: "3aaaaaaaaaaa2", CID: "bafy-valid", Action: "create", Record: json.RawMessage(`{"$type":"app.bsky.actor.profile","displayName":"valid"}`)}
	dispatcher := NewTransactionalDispatcher()
	dispatcher.Register(blueskyProfileNSID, NewBlueskyProfile(pool))
	project := func() {
		t.Helper()
		if _, err := store.IngestRecord(ctx, event); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `UPDATE tap_source_records SET projection_generation=1 WHERE uri=$1`, uri); err != nil {
			t.Fatal(err)
		}
		source, err := store.Source(ctx, uri)
		if err != nil {
			t.Fatal(err)
		}
		if err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
			_, err := dispatcher.Project(ctx, tx, source)
			return err
		}); err != nil {
			t.Fatal(err)
		}
	}
	project()
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM bluesky_profiles WHERE did=$1`, actor).Scan(&count); err != nil || count != 1 {
		t.Fatalf("before invalid: %d, %v", count, err)
	}
	event.ID, event.Rev, event.CID, event.Action = 2, "3aaaaaaaaaaa3", "bafy-invalid", "update"
	event.Record = json.RawMessage(`{"$type":"app.bsky.actor.profile","displayName":42}`)
	project()
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM bluesky_profiles WHERE did=$1`, actor).Scan(&count); err != nil || count != 0 {
		t.Fatalf("after invalid: %d, %v", count, err)
	}
}

func TestProjectSetSourceWakesBlockedDependencyWithoutTapReplay(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 23, 16, 0, 0, 0, time.UTC)
	store, err := ingestion.NewStore(pool, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	actor := syntax.DID("did:plc:blocked-like-actor")
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at
		) VALUES($1,'active',1,1,'test',$2,$2,$2)
	`, actor, now); err != nil {
		t.Fatalf("seed active lifecycle: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles(did,record_cid) VALUES($1,'bafy-profile')`, actor); err != nil {
		t.Fatalf("seed active profile: %v", err)
	}
	subject := syntax.ATURI("at://did:plc:post-owner/social.craftsky.feed.post/3aaaaaaaaaaa3")
	event := tap.Event{
		ID: 10, URI: "at://did:plc:blocked-like-actor/social.craftsky.feed.like/3aaaaaaaaaaa4",
		DID: actor, Collection: craftskyLikeNSID, Rkey: "3aaaaaaaaaaa4",
		Rev: "3aaaaaaaaaaa4", CID: "bafy-like", Action: "create",
		Record: json.RawMessage(`{"subject":{"uri":"` + subject.String() + `","cid":"bafysubject"},"createdAt":"2026-09-23T15:00:00Z"}`),
	}
	if _, err := store.IngestRecord(ctx, event); err != nil {
		t.Fatalf("ingest like: %v", err)
	}
	source, err := store.Source(ctx, event.URI)
	if err != nil {
		t.Fatal(err)
	}
	if err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		_, err := projectSetSourceTx(ctx, tx, source, now)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	var eligible bool
	var reason, dependencyKind, dependencyKey string
	if err := pool.QueryRow(ctx, `
		SELECT eligible,ineligibility_reason,dependency_kind,dependency_key
		FROM pds_set_sources WHERE source_uri=$1
	`, event.URI).Scan(&eligible, &reason, &dependencyKind, &dependencyKey); err != nil {
		t.Fatal(err)
	}
	if eligible || reason != "missing_subject" || dependencyKind != "subject_uri" || dependencyKey != subject.String() {
		t.Fatalf("blocked fact = eligible:%t reason:%s dependency:%s/%s", eligible, reason, dependencyKind, dependencyKey)
	}
	assertSetFactCounts(t, pool, event.URI, 1, 0)

	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_posts(uri,did,rkey,cid,text,record,created_at)
		VALUES($1,'did:plc:post-owner','3aaaaaaaaaaa3','bafy-post','post','{}'::jsonb,$2)
	`, subject, now); err != nil {
		t.Fatalf("create dependency: %v", err)
	}
	if err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		_, err := projectSetSourceTx(ctx, tx, source, now.Add(time.Minute))
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT eligible FROM pds_set_sources WHERE source_uri=$1`, event.URI).Scan(&eligible); err != nil || !eligible {
		t.Fatalf("woken fact eligible=%t err=%v", eligible, err)
	}
	assertSetFactCounts(t, pool, event.URI, 1, 1)
}

func ingestAndProjectSetSource(t *testing.T, store *ingestion.Store, pool interface {
	Begin(context.Context) (pgx.Tx, error)
}, event tap.Event, now time.Time) {
	t.Helper()
	if _, err := store.IngestRecord(context.Background(), event); err != nil {
		t.Fatalf("ingest %s: %v", event.Rev, err)
	}
	source, err := store.Source(context.Background(), event.URI)
	if err != nil {
		t.Fatalf("read %s: %v", event.Rev, err)
	}
	if err := pgx.BeginFunc(context.Background(), pool, func(tx pgx.Tx) error {
		_, err := projectSetSourceTx(context.Background(), tx, source, now)
		return err
	}); err != nil {
		t.Fatalf("project %s: %v", event.Rev, err)
	}
}

func assertSetFactCounts(t *testing.T, pool interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, uri syntax.ATURI, wantFacts, wantAggregates int) {
	t.Helper()
	var facts, aggregates int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM pds_set_sources WHERE source_uri=$1`, uri).Scan(&facts); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM pds_set_aggregates`).Scan(&aggregates); err != nil {
		t.Fatal(err)
	}
	if facts != wantFacts || aggregates != wantAggregates {
		t.Fatalf("fact/aggregate counts = %d/%d, want %d/%d", facts, aggregates, wantFacts, wantAggregates)
	}
}
