package index

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/ingestion"
	"social.craftsky/appview/internal/notifications"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

func TestReplaceSetSourceMaintainsAggregateAcrossRepresentativeChurn(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	store, err := ingestion.NewStore(pool, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	actor := syntax.DID("did:plc:aggregate-actor")
	subject := syntax.ATURI("at://did:plc:subject/social.craftsky.feed.post/3aaaaaaaaaaa2")
	scope := SetScope{Kind: "like", Actor: actor, Key: subject.String()}
	sources := []SetSource{
		{URI: "at://did:plc:aggregate-actor/social.craftsky.feed.like/3aaaaaaaaaaa3", Scope: scope, SubjectURI: subject, ActivityAt: now.Add(-2 * time.Hour), Eligible: true},
		{URI: "at://did:plc:aggregate-actor/social.craftsky.feed.like/3aaaaaaaaaaa4", Scope: scope, SubjectURI: subject, ActivityAt: now.Add(-time.Hour), Eligible: true},
	}
	for i, source := range sources {
		record := json.RawMessage(`{"subject":{"uri":"` + subject.String() + `","cid":"bafysubject"},"createdAt":"2026-09-23T10:00:00Z"}`)
		if _, err := store.IngestRecord(ctx, tap.Event{
			ID: uint64(i + 1), URI: source.URI, DID: actor, Collection: craftskyLikeNSID,
			Rkey: source.URI.RecordKey(), Rev: syntax.TID("3aaaaaaaaaaa" + string(rune('2'+i))),
			CID: syntax.CID("bafy-source"), Action: "create", Record: record,
		}); err != nil {
			t.Fatalf("ingest source %d: %v", i, err)
		}
		if err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
			_, err := replaceSetSourceTx(ctx, tx, source.URI, &source, now)
			return err
		}); err != nil {
			t.Fatalf("replace source %d: %v", i, err)
		}
	}

	aggregate := readSetAggregate(t, pool, scope)
	if aggregate.EligibleSourceCount != 2 || aggregate.RepresentativeURI != sources[0].URI {
		t.Fatalf("initial aggregate = %+v", aggregate)
	}
	activatedAt := aggregate.ActivatedAt

	if err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		changes, err := replaceSetSourceTx(ctx, tx, sources[0].URI, nil, now.Add(time.Hour))
		if err == nil && (len(changes) != 1 || changes[0].Transition != SetUnchanged) {
			t.Fatalf("representative deletion changes = %+v", changes)
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	aggregate = readSetAggregate(t, pool, scope)
	if aggregate.EligibleSourceCount != 1 || aggregate.RepresentativeURI != sources[1].URI || !aggregate.ActivatedAt.Equal(activatedAt) {
		t.Fatalf("aggregate after representative deletion = %+v", aggregate)
	}

	if err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		changes, err := replaceSetSourceTx(ctx, tx, sources[1].URI, nil, now.Add(2*time.Hour))
		if err == nil && (len(changes) != 1 || changes[0].Transition != SetDeactivated) {
			t.Fatalf("final deletion changes = %+v", changes)
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM pds_set_aggregates WHERE kind=$1 AND actor_did=$2 AND scope_key=$3`, scope.Kind, scope.Actor, scope.Key).Scan(&count); err != nil || count != 0 {
		t.Fatalf("aggregate count after final deletion = %d, err=%v", count, err)
	}
}

func TestCraftskyLikeProjectMaintainsLogicalAggregateAcrossDuplicateSources(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 24, 9, 0, 0, 0, time.UTC)
	store, err := ingestion.NewStore(pool, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	actor := syntax.DID("did:plc:aggregate-projector-actor")
	subject := syntax.ATURI("at://did:plc:aggregate-projector-subject/social.craftsky.feed.post/3aaaaaaaaaaa2")
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at
		) VALUES($1,'active',1,1,'test',$2,$2,$2)
	`, actor, now); err != nil {
		t.Fatalf("seed projector lifecycle: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles(did,record_cid) VALUES($1,'bafy-profile')`, actor); err != nil {
		t.Fatalf("seed projector profile: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_posts(uri,did,rkey,cid,text,record,created_at)
		VALUES($1,'did:plc:aggregate-projector-subject','3aaaaaaaaaaa2','bafy-subject','subject','{}'::jsonb,$2)
	`, subject, now); err != nil {
		t.Fatalf("seed projector subject: %v", err)
	}
	installationID := uuid.New()
	subscriptionID := uuid.New()
	if _, err := pool.Exec(ctx, `
		INSERT INTO push_installations(id,device_id,platform,fcm_token)
		VALUES($1,'aggregate-like-device','ios','aggregate-like-token')
	`, installationID); err != nil {
		t.Fatalf("seed projector push installation: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO push_account_subscriptions(id,installation_id,account_did,routing_id)
		VALUES($2,$1,'did:plc:aggregate-projector-subject',$3)
	`, installationID, subscriptionID, uuid.New()); err != nil {
		t.Fatalf("seed projector push subscription: %v", err)
	}

	dispatcher := NewTransactionalDispatcher()
	dispatcher.Register(craftskyLikeNSID, NewCraftskyLike(pool, nil, notifications.NewService()))
	var sourceURIs []syntax.ATURI
	var notificationID, notificationState string
	var notificationRevision int64
	var notificationActivity time.Time
	for index, rkey := range []syntax.RecordKey{"3aaaaaaaaaaa3", "3aaaaaaaaaaa4"} {
		uri := syntax.ATURI("at://" + actor.String() + "/social.craftsky.feed.like/" + rkey.String())
		sourceURIs = append(sourceURIs, uri)
		event := tap.Event{
			ID: uint64(index + 1), URI: uri, DID: actor, Collection: craftskyLikeNSID,
			Rkey: rkey, Rev: syntax.TID(rkey), CID: syntax.CID(fmt.Sprintf("bafy-like-%d", index)), Action: "create",
			Record: json.RawMessage(`{"$type":"social.craftsky.feed.like","subject":{"uri":"` + subject.String() + `","cid":"bafy-subject"},"createdAt":"2026-09-24T08:00:00Z"}`),
		}
		if _, err := store.IngestRecord(ctx, event); err != nil {
			t.Fatalf("ingest like %d: %v", index, err)
		}
		if _, err := pool.Exec(ctx, `UPDATE tap_source_records SET projection_generation=1 WHERE uri=$1`, uri); err != nil {
			t.Fatalf("bind like source %d to lifecycle: %v", index, err)
		}
		source, err := store.Source(ctx, uri)
		if err != nil {
			t.Fatalf("read like source %d: %v", index, err)
		}
		if err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
			outcome, err := dispatcher.Project(ctx, tx, source)
			if err == nil && outcome.Kind != tap.OutcomeApplied {
				t.Fatalf("project like %d outcome=%+v", index, outcome)
			}
			return err
		}); err != nil {
			t.Fatalf("project like %d: %v", index, err)
		}
		var gotID, gotState string
		var gotRevision int64
		var gotActivity time.Time
		if err := pool.QueryRow(ctx, `
			SELECT id::text,state,newness_revision,activity_at
			FROM notification_events
			WHERE recipient_did='did:plc:aggregate-projector-subject'
			  AND actor_did=$1 AND category='like' AND subject_key=$2
		`, actor, subject).Scan(&gotID, &gotState, &gotRevision, &gotActivity); err != nil {
			t.Fatalf("read notification after like %d: %v", index, err)
		}
		if index == 0 {
			notificationID, notificationState = gotID, gotState
			notificationRevision, notificationActivity = gotRevision, gotActivity
		} else if gotID != notificationID || gotState != notificationState || gotRevision != notificationRevision || !gotActivity.Equal(notificationActivity) {
			t.Fatalf("notification changed across duplicate source: id/state/revision/activity = %s/%s/%d/%s, want %s/%s/%d/%s", gotID, gotState, gotRevision, gotActivity, notificationID, notificationState, notificationRevision, notificationActivity)
		}
		var deliveries int
		var deliveryStatus string
		if err := pool.QueryRow(ctx, `SELECT count(*),COALESCE(max(status),'') FROM push_deliveries WHERE notification_id=$1`, gotID).Scan(&deliveries, &deliveryStatus); err != nil {
			t.Fatalf("read delivery after like %d: %v", index, err)
		}
		if deliveries != 1 || deliveryStatus != "pending" {
			t.Fatalf("delivery after like %d = %d/%s, want 1/pending", index, deliveries, deliveryStatus)
		}
	}

	var facts, aggregates, sourceCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM pds_set_sources WHERE kind='like' AND actor_did=$1 AND scope_key=$2`, actor, subject).Scan(&facts); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*), COALESCE(max(eligible_source_count),0) FROM pds_set_aggregates WHERE kind='like' AND actor_did=$1 AND scope_key=$2`, actor, subject).Scan(&aggregates, &sourceCount); err != nil {
		t.Fatal(err)
	}
	if facts != 2 || aggregates != 1 || sourceCount != 2 {
		t.Fatalf("normalized like facts/aggregates/source count = %d/%d/%d, want 2/1/2", facts, aggregates, sourceCount)
	}

	for index, uri := range sourceURIs {
		event := tap.Event{
			ID: uint64(index + 3), URI: uri, DID: actor, Collection: craftskyLikeNSID,
			Rkey: uri.RecordKey(), Rev: syntax.TID(fmt.Sprintf("3aaaaaaaaaaa%d", index+5)), Action: "delete",
		}
		if _, err := store.IngestRecord(ctx, event); err != nil {
			t.Fatalf("ingest like deletion %d: %v", index, err)
		}
		if _, err := pool.Exec(ctx, `UPDATE tap_source_records SET projection_generation=1 WHERE uri=$1`, uri); err != nil {
			t.Fatalf("bind like deletion %d to lifecycle: %v", index, err)
		}
		source, err := store.Source(ctx, uri)
		if err != nil {
			t.Fatalf("read deleted like source %d: %v", index, err)
		}
		if err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
			outcome, err := dispatcher.Project(ctx, tx, source)
			if err == nil && outcome.Kind != tap.OutcomeApplied {
				t.Fatalf("project like deletion %d outcome=%+v", index, outcome)
			}
			return err
		}); err != nil {
			t.Fatalf("project like deletion %d: %v", index, err)
		}

		wantFacts, wantAggregates, wantSources := 1-index, 1-index, 1-index
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM pds_set_sources WHERE kind='like' AND actor_did=$1 AND scope_key=$2`, actor, subject).Scan(&facts); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(ctx, `SELECT count(*), COALESCE(max(eligible_source_count),0) FROM pds_set_aggregates WHERE kind='like' AND actor_did=$1 AND scope_key=$2`, actor, subject).Scan(&aggregates, &sourceCount); err != nil {
			t.Fatal(err)
		}
		if facts != wantFacts || aggregates != wantAggregates || sourceCount != wantSources {
			t.Fatalf("after deletion %d facts/aggregates/source count = %d/%d/%d, want %d/%d/%d", index, facts, aggregates, sourceCount, wantFacts, wantAggregates, wantSources)
		}
		var gotID, gotState string
		var gotRevision int64
		var gotActivity time.Time
		if err := pool.QueryRow(ctx, `
			SELECT id::text,state,newness_revision,activity_at
			FROM notification_events
			WHERE recipient_did='did:plc:aggregate-projector-subject'
			  AND actor_did=$1 AND category='like' AND subject_key=$2
		`, actor, subject).Scan(&gotID, &gotState, &gotRevision, &gotActivity); err != nil {
			t.Fatalf("read notification after deletion %d: %v", index, err)
		}
		wantState := "active"
		if index == len(sourceURIs)-1 {
			wantState = "retracted"
		}
		if gotID != notificationID || gotState != wantState || gotRevision != notificationRevision || !gotActivity.Equal(notificationActivity) {
			t.Fatalf("notification after deletion %d = %s/%s/%d/%s, want %s/%s/%d/%s", index, gotID, gotState, gotRevision, gotActivity, notificationID, wantState, notificationRevision, notificationActivity)
		}
		var deliveries int
		var deliveryStatus string
		if err := pool.QueryRow(ctx, `SELECT count(*),COALESCE(max(status),'') FROM push_deliveries WHERE notification_id=$1`, gotID).Scan(&deliveries, &deliveryStatus); err != nil {
			t.Fatalf("read delivery after deletion %d: %v", index, err)
		}
		wantDeliveryStatus := "pending"
		if index == len(sourceURIs)-1 {
			wantDeliveryStatus = "cancelled"
		}
		if deliveries != 1 || deliveryStatus != wantDeliveryStatus {
			t.Fatalf("delivery after deletion %d = %d/%s, want 1/%s", index, deliveries, deliveryStatus, wantDeliveryStatus)
		}
	}
}

func TestCraftskyRepostProjectMaintainsLogicalAggregateAcrossDuplicateSources(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 24, 9, 0, 0, 0, time.UTC)
	store, err := ingestion.NewStore(pool, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	actor := syntax.DID("did:plc:aggregate-repost-actor")
	subject := syntax.ATURI("at://did:plc:aggregate-repost-subject/social.craftsky.feed.post/3aaaaaaaaaaa2")
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at
		) VALUES($1,'active',1,1,'test',$2,$2,$2)
	`, actor, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles(did,record_cid) VALUES($1,'bafy-profile')`, actor); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_posts(uri,did,rkey,cid,text,record,created_at)
		VALUES($1,'did:plc:aggregate-repost-subject','3aaaaaaaaaaa2','bafy-subject','subject','{}'::jsonb,$2)
	`, subject, now); err != nil {
		t.Fatal(err)
	}

	dispatcher := NewTransactionalDispatcher()
	dispatcher.Register(craftskyRepostNSID, NewCraftskyRepost(pool, nil, notifications.NewService()))
	sourceURIs := []syntax.ATURI{
		syntax.ATURI("at://" + actor.String() + "/social.craftsky.feed.repost/3aaaaaaaaaaa3"),
		syntax.ATURI("at://" + actor.String() + "/social.craftsky.feed.repost/3aaaaaaaaaaa4"),
	}
	for index, uri := range sourceURIs {
		event := tap.Event{
			ID: uint64(index + 1), URI: uri, DID: actor, Collection: craftskyRepostNSID,
			Rkey: uri.RecordKey(), Rev: syntax.TID(uri.RecordKey()), CID: syntax.CID(fmt.Sprintf("bafy-repost-%d", index)), Action: "create",
			Record: json.RawMessage(`{"$type":"social.craftsky.feed.repost","subject":{"uri":"` + subject.String() + `","cid":"bafy-subject"},"createdAt":"2026-09-24T08:00:00Z"}`),
		}
		if _, err := store.IngestRecord(ctx, event); err != nil {
			t.Fatalf("ingest repost %d: %v", index, err)
		}
		if _, err := pool.Exec(ctx, `UPDATE tap_source_records SET projection_generation=1 WHERE uri=$1`, uri); err != nil {
			t.Fatal(err)
		}
		source, err := store.Source(ctx, uri)
		if err != nil {
			t.Fatal(err)
		}
		if err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
			outcome, err := dispatcher.Project(ctx, tx, source)
			if err == nil && outcome.Kind != tap.OutcomeApplied {
				t.Fatalf("project repost %d outcome=%+v", index, outcome)
			}
			return err
		}); err != nil {
			t.Fatalf("project repost %d: %v", index, err)
		}
	}

	assertRepostSetCounts := func(wantFacts, wantAggregates, wantSources int) {
		t.Helper()
		var facts, aggregates, sourceCount int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM pds_set_sources WHERE kind='repost' AND actor_did=$1 AND scope_key=$2`, actor, subject).Scan(&facts); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(ctx, `SELECT count(*),COALESCE(max(eligible_source_count),0) FROM pds_set_aggregates WHERE kind='repost' AND actor_did=$1 AND scope_key=$2`, actor, subject).Scan(&aggregates, &sourceCount); err != nil {
			t.Fatal(err)
		}
		if facts != wantFacts || aggregates != wantAggregates || sourceCount != wantSources {
			t.Fatalf("repost facts/aggregates/source count = %d/%d/%d, want %d/%d/%d", facts, aggregates, sourceCount, wantFacts, wantAggregates, wantSources)
		}
	}
	assertRepostSetCounts(2, 1, 2)
	var notificationID string
	var notificationActivity time.Time
	if err := pool.QueryRow(ctx, `
		SELECT id::text,activity_at FROM notification_events
		WHERE recipient_did='did:plc:aggregate-repost-subject'
		  AND actor_did=$1 AND category='repost' AND subject_key=$2 AND state='active'
	`, actor, subject).Scan(&notificationID, &notificationActivity); err != nil {
		t.Fatalf("read logical repost notification: %v", err)
	}

	for index, uri := range sourceURIs {
		event := tap.Event{
			ID: uint64(index + 3), URI: uri, DID: actor, Collection: craftskyRepostNSID,
			Rkey: uri.RecordKey(), Rev: syntax.TID(fmt.Sprintf("3aaaaaaaaaaa%d", index+5)), Action: "delete",
		}
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
			t.Fatalf("project repost deletion %d: %v", index, err)
		}
		assertRepostSetCounts(1-index, 1-index, 1-index)
		var gotID, state string
		var activityAt time.Time
		if err := pool.QueryRow(ctx, `
			SELECT id::text,state,activity_at FROM notification_events
			WHERE recipient_did='did:plc:aggregate-repost-subject'
			  AND actor_did=$1 AND category='repost' AND subject_key=$2
		`, actor, subject).Scan(&gotID, &state, &activityAt); err != nil {
			t.Fatal(err)
		}
		wantState := "active"
		if index == len(sourceURIs)-1 {
			wantState = "retracted"
		}
		if gotID != notificationID || state != wantState || !activityAt.Equal(notificationActivity) {
			t.Fatalf("repost notification after deletion %d = %s/%s/%s, want %s/%s/%s", index, gotID, state, activityAt, notificationID, wantState, notificationActivity)
		}
	}
}

func TestBlueskyFollowProjectMaintainsLogicalAggregateAcrossDuplicateSources(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 24, 11, 0, 0, 0, time.UTC)
	store, err := ingestion.NewStore(pool, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	actor := syntax.DID("did:plc:aggregate-follow-actor")
	subject := syntax.DID("did:plc:aggregate-follow-subject")
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at
		) VALUES($1,'active',1,1,'test',$3,$3,$3),($2,'active',1,1,'test',$3,$3,$3)
	`, actor, subject, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_profiles(did,record_cid) VALUES($1,'bafy-actor'),($2,'bafy-subject')
	`, actor, subject); err != nil {
		t.Fatal(err)
	}

	dispatcher := NewTransactionalDispatcher()
	dispatcher.Register(blueskyFollowNSID, NewBlueskyFollow(pool, notifications.NewService()))
	sourceURIs := []syntax.ATURI{
		syntax.ATURI("at://" + actor.String() + "/app.bsky.graph.follow/3aaaaaaaaaaa5"),
		syntax.ATURI("at://" + actor.String() + "/app.bsky.graph.follow/3aaaaaaaaaaa6"),
	}
	for index, uri := range sourceURIs {
		event := tap.Event{
			ID: uint64(index + 1), URI: uri, DID: actor, Collection: blueskyFollowNSID,
			Rkey: uri.RecordKey(), Rev: syntax.TID(uri.RecordKey()), CID: syntax.CID(fmt.Sprintf("bafy-follow-%d", index)), Action: "create",
			Record: json.RawMessage(`{"$type":"app.bsky.graph.follow","subject":"` + subject.String() + `","createdAt":"2026-09-24T10:00:00Z"}`),
		}
		if _, err := store.IngestRecord(ctx, event); err != nil {
			t.Fatalf("ingest follow %d: %v", index, err)
		}
		if _, err := pool.Exec(ctx, `UPDATE tap_source_records SET projection_generation=1 WHERE uri=$1`, uri); err != nil {
			t.Fatal(err)
		}
		source, err := store.Source(ctx, uri)
		if err != nil {
			t.Fatal(err)
		}
		if err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
			outcome, err := dispatcher.Project(ctx, tx, source)
			if err == nil && outcome.Kind != tap.OutcomeApplied {
				t.Fatalf("project follow %d outcome=%+v", index, outcome)
			}
			return err
		}); err != nil {
			t.Fatalf("project follow %d: %v", index, err)
		}
	}

	assertFollowSetCounts := func(wantFacts, wantAggregates, wantSources int) {
		t.Helper()
		var facts, aggregates, sourceCount int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM pds_set_sources WHERE kind='follow' AND actor_did=$1 AND scope_key=$2`, actor, subject).Scan(&facts); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(ctx, `SELECT count(*),COALESCE(max(eligible_source_count),0) FROM pds_set_aggregates WHERE kind='follow' AND actor_did=$1 AND scope_key=$2`, actor, subject).Scan(&aggregates, &sourceCount); err != nil {
			t.Fatal(err)
		}
		if facts != wantFacts || aggregates != wantAggregates || sourceCount != wantSources {
			t.Fatalf("follow facts/aggregates/source count = %d/%d/%d, want %d/%d/%d", facts, aggregates, sourceCount, wantFacts, wantAggregates, wantSources)
		}
	}
	assertFollowSetCounts(2, 1, 2)
	var notificationID string
	var notificationActivity time.Time
	if err := pool.QueryRow(ctx, `
		SELECT id::text,activity_at FROM notification_events
		WHERE recipient_did=$1 AND actor_did=$2 AND category='follow'
		  AND subject_key=$1 AND state='active'
	`, subject, actor).Scan(&notificationID, &notificationActivity); err != nil {
		t.Fatalf("read logical follow notification: %v", err)
	}

	for index, uri := range sourceURIs {
		deleteRevisions := []syntax.TID{"3aaaaaaaaaaab", "3aaaaaaaaaaac"}
		event := tap.Event{
			ID: uint64(index + 3), URI: uri, DID: actor, Collection: blueskyFollowNSID,
			Rkey: uri.RecordKey(), Rev: deleteRevisions[index], Action: "delete",
		}
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
			t.Fatalf("project follow deletion %d: %v", index, err)
		}
		assertFollowSetCounts(1-index, 1-index, 1-index)
		var gotID, state string
		var activityAt time.Time
		if err := pool.QueryRow(ctx, `
			SELECT id::text,state,activity_at FROM notification_events
			WHERE recipient_did=$1 AND actor_did=$2 AND category='follow' AND subject_key=$1
		`, subject, actor).Scan(&gotID, &state, &activityAt); err != nil {
			t.Fatal(err)
		}
		wantState := "active"
		if index == len(sourceURIs)-1 {
			wantState = "retracted"
		}
		if gotID != notificationID || state != wantState || !activityAt.Equal(notificationActivity) {
			t.Fatalf("follow notification after deletion %d = %s/%s/%s, want %s/%s/%s", index, gotID, state, activityAt, notificationID, wantState, notificationActivity)
		}
	}
}

func TestBlueskyBlockProjectMaintainsLogicalAggregateAcrossDuplicateSources(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	store, err := ingestion.NewStore(pool, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	actor := syntax.DID("did:plc:aggregate-block-actor")
	subject := syntax.DID("did:plc:aggregate-block-subject")
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at
		) VALUES($1,'active',1,1,'test',$2,$2,$2)
	`, actor, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles(did,record_cid) VALUES($1,'bafy-actor')`, actor); err != nil {
		t.Fatal(err)
	}
	installationID := uuid.New()
	subscriptionID := uuid.New()
	notificationID := uuid.New()
	deliveryID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO push_installations(id,device_id,platform,fcm_token) VALUES($1,'aggregate-block-device','ios','aggregate-block-token')`, installationID); err != nil {
		t.Fatalf("seed block push installation: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO push_account_subscriptions(id,installation_id,account_did,routing_id) VALUES($1,$2,$3,$4)`, subscriptionID, installationID, subject, uuid.New()); err != nil {
		t.Fatalf("seed block push subscription: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO notification_events(
			id,recipient_did,actor_did,category,subject_key,source_uri,source_cid,source_rkey,
			eligibility_scope,recipient_followed_actor,push_enabled_snapshot,state,
			first_activity_at,activity_at,initial_push_evaluated_at
		) VALUES($1,$2,$3,'everythingElse','block-cancellation','at://source','bafy-source','source',
			'everyone',false,true,'active',$4,$4,$4)
	`, notificationID, subject, actor, now); err != nil {
		t.Fatalf("seed block notification: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO push_deliveries(id,notification_id,account_subscription_id,status,next_attempt_at,deadline_at)
		VALUES($1,$2,$3,'pending',$4::timestamptz,$4::timestamptz + interval '6 hours')
	`, deliveryID, notificationID, subscriptionID, now); err != nil {
		t.Fatalf("seed block delivery: %v", err)
	}

	dispatcher := NewTransactionalDispatcher()
	dispatcher.Register(blueskyBlockNSID, NewBlueskyBlock(pool))
	sourceURIs := []syntax.ATURI{
		syntax.ATURI("at://" + actor.String() + "/app.bsky.graph.block/3aaaaaaaaaaad"),
		syntax.ATURI("at://" + actor.String() + "/app.bsky.graph.block/3aaaaaaaaaaae"),
	}
	for index, uri := range sourceURIs {
		event := tap.Event{
			ID: uint64(index + 1), URI: uri, DID: actor, Collection: blueskyBlockNSID,
			Rkey: uri.RecordKey(), Rev: syntax.TID(uri.RecordKey()), CID: syntax.CID(fmt.Sprintf("bafy-block-%d", index)), Action: "create",
			Record: json.RawMessage(`{"$type":"app.bsky.graph.block","subject":"` + subject.String() + `","createdAt":"2026-09-24T11:00:00Z"}`),
		}
		if _, err := store.IngestRecord(ctx, event); err != nil {
			t.Fatalf("ingest block %d: %v", index, err)
		}
		if _, err := pool.Exec(ctx, `UPDATE tap_source_records SET projection_generation=1 WHERE uri=$1`, uri); err != nil {
			t.Fatal(err)
		}
		source, err := store.Source(ctx, uri)
		if err != nil {
			t.Fatal(err)
		}
		if err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
			outcome, err := dispatcher.Project(ctx, tx, source)
			if err == nil && outcome.Kind != tap.OutcomeApplied {
				t.Fatalf("project block %d outcome=%+v", index, outcome)
			}
			return err
		}); err != nil {
			t.Fatalf("project block %d: %v", index, err)
		}
		var deliveryStatus string
		if err := pool.QueryRow(ctx, `SELECT status FROM push_deliveries WHERE id=$1`, deliveryID).Scan(&deliveryStatus); err != nil {
			t.Fatal(err)
		}
		wantStatus := "cancelled"
		if index == 1 {
			wantStatus = "pending"
		}
		if deliveryStatus != wantStatus {
			t.Fatalf("delivery after block %d = %s, want %s", index, deliveryStatus, wantStatus)
		}
		if index == 0 {
			if _, err := pool.Exec(ctx, `UPDATE push_deliveries SET status='pending' WHERE id=$1`, deliveryID); err != nil {
				t.Fatal(err)
			}
		}
	}

	assertBlockSetCounts := func(wantFacts, wantAggregates, wantSources int) {
		t.Helper()
		var facts, aggregates, sourceCount int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM pds_set_sources WHERE kind='block' AND actor_did=$1 AND scope_key=$2`, actor, subject).Scan(&facts); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(ctx, `SELECT count(*),COALESCE(max(eligible_source_count),0) FROM pds_set_aggregates WHERE kind='block' AND actor_did=$1 AND scope_key=$2`, actor, subject).Scan(&aggregates, &sourceCount); err != nil {
			t.Fatal(err)
		}
		if facts != wantFacts || aggregates != wantAggregates || sourceCount != wantSources {
			t.Fatalf("block facts/aggregates/source count = %d/%d/%d, want %d/%d/%d", facts, aggregates, sourceCount, wantFacts, wantAggregates, wantSources)
		}
	}
	assertBlockSetCounts(2, 1, 2)

	for index, uri := range sourceURIs {
		deleteRevisions := []syntax.TID{"3aaaaaaaaaaaf", "3aaaaaaaaaaag"}
		event := tap.Event{
			ID: uint64(index + 3), URI: uri, DID: actor, Collection: blueskyBlockNSID,
			Rkey: uri.RecordKey(), Rev: deleteRevisions[index], Action: "delete",
		}
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
			t.Fatalf("project block deletion %d: %v", index, err)
		}
		assertBlockSetCounts(1-index, 1-index, 1-index)
	}
}

func TestCraftskyLikeProjectPersistsMissingSubjectFactUntilDependencyWakeup(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)
	store, err := ingestion.NewStore(pool, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	actor := syntax.DID("did:plc:blocked-like-actor")
	subject := syntax.ATURI("at://did:plc:blocked-like-subject/social.craftsky.feed.post/3aaaaaaaaaaa2")
	uri := syntax.ATURI("at://" + actor.String() + "/social.craftsky.feed.like/3aaaaaaaaaaa3")
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at
		) VALUES($1,'active',1,1,'test',$2,$2,$2)
	`, actor, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles(did,record_cid) VALUES($1,'bafy-profile')`, actor); err != nil {
		t.Fatal(err)
	}
	event := tap.Event{
		ID: 1, URI: uri, DID: actor, Collection: craftskyLikeNSID,
		Rkey: uri.RecordKey(), Rev: syntax.TID("3aaaaaaaaaaa3"), CID: syntax.CID("bafy-like"), Action: "create",
		Record: json.RawMessage(`{"$type":"social.craftsky.feed.like","subject":{"uri":"` + subject.String() + `","cid":"bafy-subject"},"createdAt":"2026-09-24T09:00:00Z"}`),
	}
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
	dispatcher := NewTransactionalDispatcher()
	dispatcher.Register(craftskyLikeNSID, NewCraftskyLike(pool, nil))
	project := func() tap.Outcome {
		t.Helper()
		var outcome tap.Outcome
		if err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
			var err error
			outcome, err = dispatcher.Project(ctx, tx, source)
			return err
		}); err != nil {
			t.Fatal(err)
		}
		return outcome
	}
	if outcome := project(); outcome.Kind != tap.OutcomeBlocked || outcome.Reason != tap.ReasonMissingSubject {
		t.Fatalf("missing-subject outcome = %+v", outcome)
	}
	var facts, eligible, aggregates int
	if err := pool.QueryRow(ctx, `
		SELECT count(*),count(*) FILTER (WHERE eligible)
		FROM pds_set_sources WHERE source_uri=$1
	`, uri).Scan(&facts, &eligible); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM pds_set_aggregates WHERE kind='like' AND actor_did=$1 AND scope_key=$2`, actor, subject).Scan(&aggregates); err != nil {
		t.Fatal(err)
	}
	if facts != 1 || eligible != 0 || aggregates != 0 {
		t.Fatalf("blocked facts/eligible/aggregates = %d/%d/%d, want 1/0/0", facts, eligible, aggregates)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_posts(uri,did,rkey,cid,text,record,created_at)
		VALUES($1,'did:plc:blocked-like-subject','3aaaaaaaaaaa2','bafy-subject','subject','{}'::jsonb,$2)
	`, subject, now); err != nil {
		t.Fatal(err)
	}
	if outcome := project(); outcome.Kind != tap.OutcomeApplied {
		t.Fatalf("dependency wake-up outcome = %+v", outcome)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM pds_set_aggregates WHERE kind='like' AND actor_did=$1 AND scope_key=$2`, actor, subject).Scan(&aggregates); err != nil {
		t.Fatal(err)
	}
	if aggregates != 1 {
		t.Fatalf("aggregate count after dependency wake-up = %d, want 1", aggregates)
	}
}

func readSetAggregate(t *testing.T, pool interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, scope SetScope) SetAggregate {
	t.Helper()
	var aggregate SetAggregate
	if err := pool.QueryRow(context.Background(), `
		SELECT eligible_source_count,representative_source_uri,activated_at
		FROM pds_set_aggregates WHERE kind=$1 AND actor_did=$2 AND scope_key=$3
	`, scope.Kind, scope.Actor, scope.Key).Scan(&aggregate.EligibleSourceCount, &aggregate.RepresentativeURI, &aggregate.ActivatedAt); err != nil {
		t.Fatalf("read aggregate: %v", err)
	}
	return aggregate
}
