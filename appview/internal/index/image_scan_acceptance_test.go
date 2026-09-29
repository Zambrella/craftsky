package index_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/imagesafety"
	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/ingestion"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

const imageScanTapPreStateDDL = `
CREATE TABLE tap_source_records (uri TEXT PRIMARY KEY);
CREATE TABLE tap_projection_jobs (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    source_uri TEXT NOT NULL REFERENCES tap_source_records(uri) ON DELETE CASCADE,
    projection_kind TEXT NOT NULL,
    source_event_id BIGINT NOT NULL CHECK (source_event_id > 0),
    state TEXT NOT NULL CHECK (state IN ('pending', 'blocked', 'processing', 'complete', 'permanent_denied')),
    dependency_kind TEXT,
    dependency_key TEXT,
    CONSTRAINT tap_projection_jobs_dependency_check CHECK (
        (state = 'blocked'
            AND dependency_kind IN ('member_did', 'subject_uri', 'repository_did')
            AND dependency_key IS NOT NULL
            AND btrim(dependency_key) <> '' AND char_length(dependency_key) <= 2048)
        OR
        (state <> 'blocked' AND dependency_kind IS NULL AND dependency_key IS NULL)
    )
);
`

func TestCraftskyImagePostVisibilityWaitsForEveryCurrentScan(t *testing.T) {
	pool := testdb.WithSchema(t, craftskyPostsDDL+imageScanTapPreStateDDL)
	migration, err := os.ReadFile("../../migrations/000076_image_safety.up.sql")
	if err != nil {
		t.Fatalf("read image safety migration: %v", err)
	}
	if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
		t.Fatalf("apply image safety migration: %v", err)
	}

	ctx := context.Background()
	actor := syntax.DID("did:plc:image-safety-post")
	seedCraftskyMember(t, pool, actor.String())
	const (
		firstBlob  = "bafkreigxxxkul4e5rjz4fomqgn6ieeoxbcqeztmxjbrhnbpe7r44ya4ahe"
		secondBlob = "bafkreidjq52a7nre4puzipwf3gwfkgnxftvbwnp3jppfogo7her2g3ai64"
	)
	key := imagesafety.ScanKey{
		ScannerID:     "fixture",
		PolicyVersion: "policy-1",
		CorpusVersion: "corpus-1",
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO image_scan_results(
			id, blob_cid, scanner_id, policy_version, corpus_version, state, completed_at
		) VALUES
			('10000000-0000-4000-8000-000000000001',$1,$3,$4,$5,'clear',now()),
			('10000000-0000-4000-8000-000000000002',$2,$3,$4,$5,'pending',NULL)
	`, firstBlob, secondBlob, key.ScannerID, key.PolicyVersion, key.CorpusVersion); err != nil {
		t.Fatalf("seed scan results: %v", err)
	}

	event := tap.Event{
		URI:        "at://did:plc:image-safety-post/social.craftsky.feed.post/images",
		CID:        "bafy-post-images-1",
		DID:        actor,
		Rkey:       "images",
		Collection: "social.craftsky.feed.post",
		Action:     "create",
		Record: json.RawMessage(`{
			"$type":"social.craftsky.feed.post",
			"text":"two images",
			"createdAt":"` + fixedCreatedAt + `",
			"images":[
				{"image":{"$type":"blob","ref":{"$link":"` + firstBlob + `"},"mimeType":"image/jpeg","size":12345},"alt":"first"},
				{"image":{"$type":"blob","ref":{"$link":"` + secondBlob + `"},"mimeType":"image/png","size":54321},"alt":"second"}
			]
		}`),
	}
	projector := index.NewImageSafetyCraftskyPost(index.NewCraftskyPost(pool, testLogger()), key)

	project := func(want tap.OutcomeKind) tap.Outcome {
		t.Helper()
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatalf("begin projection: %v", err)
		}
		defer func() { _ = tx.Rollback(context.Background()) }()
		outcome, err := projector.Project(ctx, tx, imageSourceFromEvent(event))
		if err != nil {
			t.Fatalf("project image post: %v", err)
		}
		if outcome.Kind != want {
			t.Fatalf("outcome=%+v, want kind %s", outcome, want)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("commit projection: %v", err)
		}
		return outcome
	}

	outcome := project(tap.OutcomeBlocked)
	if outcome.Reason != tap.ReasonImageScanPending || outcome.Dependency.Kind != "image_subject_uri" || outcome.Dependency.Key != event.URI.String() {
		t.Fatalf("blocked outcome=%+v", outcome)
	}
	assertImagePostEligibility(t, pool, event.URI, "blocked", 2, 0)

	if _, err := pool.Exec(ctx, `
		UPDATE image_scan_results
		SET state='clear', completed_at=now(), updated_at=now()
		WHERE blob_cid=$1
	`, secondBlob); err != nil {
		t.Fatalf("clear final image: %v", err)
	}
	project(tap.OutcomeApplied)
	assertImagePostEligibility(t, pool, event.URI, "clear", 2, 1)

	if _, err := pool.Exec(ctx, `
		UPDATE image_scan_results
		SET state='error', completed_at=NULL, updated_at=now()
		WHERE blob_cid=$1
	`, secondBlob); err != nil {
		t.Fatalf("regress final image: %v", err)
	}
	project(tap.OutcomeBlocked)
	assertImagePostEligibility(t, pool, event.URI, "blocked", 2, 0)
}

func assertImagePostEligibility(
	t *testing.T,
	pool interface {
		QueryRow(context.Context, string, ...any) pgx.Row
	},
	uri syntax.ATURI,
	wantState string,
	wantRequirements int,
	wantEligible int,
) {
	t.Helper()
	var state string
	var requirements int
	if err := pool.QueryRow(context.Background(), `
		SELECT state.visibility_state, count(requirement.image_slot)
		FROM image_subject_states state
		LEFT JOIN image_subject_requirements requirement
		  ON requirement.subject_uri=state.subject_uri
		WHERE state.subject_uri=$1
		GROUP BY state.visibility_state
	`, uri).Scan(&state, &requirements); err != nil {
		t.Fatalf("read image subject state: %v", err)
	}
	if state != wantState || requirements != wantRequirements {
		t.Fatalf("state=%q requirements=%d, want %q/%d", state, requirements, wantState, wantRequirements)
	}
	var eligible int
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*)
		FROM craftsky_posts post
		JOIN image_subject_states state
		  ON state.subject_uri=post.uri
		 AND state.source_cid=post.cid
		 AND state.visibility_state='clear'
		WHERE post.uri=$1
	`, uri).Scan(&eligible); err != nil {
		t.Fatalf("query eligible post: %v", err)
	}
	if eligible != wantEligible {
		t.Fatalf("eligible posts=%d, want %d", eligible, wantEligible)
	}
}

type appliedProjector struct{}

func (appliedProjector) Project(context.Context, pgx.Tx, ingestion.SourceRecord) (tap.Outcome, error) {
	return tap.Applied(), nil
}

func imageSourceFromEvent(event tap.Event) ingestion.SourceRecord {
	return ingestion.SourceRecord{
		URI: event.URI, CID: event.CID, DID: event.DID, Rkey: event.Rkey,
		Collection: event.Collection, Action: event.Action, Record: event.Record,
	}
}

func TestImageSafetyExtractsEveryRenderedBlobIntoOneScanWorkflow(t *testing.T) {
	pool := testdb.WithSchema(t, craftskyProfilesDDL+imageScanTapPreStateDDL)
	migration, err := os.ReadFile("../../migrations/000076_image_safety.up.sql")
	if err != nil {
		t.Fatalf("read image safety migration: %v", err)
	}
	if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
		t.Fatalf("apply image safety migration: %v", err)
	}

	const (
		sharedBlob  = "bafkreigxxxkul4e5rjz4fomqgn6ieeoxbcqeztmxjbrhnbpe7r44ya4ahe"
		thumbBlob   = "bafkreidjq52a7nre4puzipwf3gwfkgnxftvbwnp3jppfogo7her2g3ai64"
		eventBlob   = "bafkreibm6jgql3m7ta4szj3q5wo7fkiny2hzqoqs4dbc65d5u3r3itkqae"
		productBlob = "bafkreic5jbn7z4xkzqh4mmyv5x5xjhzlb2y6h5b56tn3m76kt4qaxwji5u"
		avatarBlob  = "bafkreih6wflq3gq7ie2g4fbyv3a6xufbwmjzf4v6o7fzg7r3jkmf4v7fsa"
		bannerBlob  = "bafkreiax2trv4v7nq6oy3vk7ucjz5d4du4s2yx6iy6uuq2q3i4wl5hznje"
	)
	key := imagesafety.ScanKey{ScannerID: "fixture", PolicyVersion: "policy-1", CorpusVersion: "corpus-1"}
	actor := syntax.DID("did:plc:third-party-writer")
	seedMember(t, pool, actor.String())
	events := []struct {
		projector index.TransactionalIndexer
		event     tap.Event
	}{
		{
			projector: index.NewImageSafetyCraftskyPost(appliedProjector{}, key),
			event: tap.Event{
				URI: "at://did:plc:third-party-writer/social.craftsky.feed.post/images", CID: "post-cid", DID: actor,
				Rkey: "images", Collection: "social.craftsky.feed.post", Action: "create",
				Record: json.RawMessage(`{
					"$type":"social.craftsky.feed.post","text":"images","createdAt":"` + fixedCreatedAt + `",
					"images":[{"image":{"$type":"blob","ref":{"$link":"` + sharedBlob + `"},"mimeType":"image/jpeg","size":10}}],
					"embed":{"$type":"app.bsky.embed.external","external":{"uri":"https://example.com","title":"Card","description":"Summary","thumb":{"$type":"blob","ref":{"$link":"` + thumbBlob + `"},"mimeType":"image/png","size":20}}}
				}`),
			},
		},
		{
			projector: index.NewImageSafetyCraftskyBusinessEvent(appliedProjector{}, key),
			event: tap.Event{
				URI: "at://did:plc:third-party-writer/social.craftsky.business.event/event", CID: "event-cid", DID: actor,
				Rkey: "event", Collection: "social.craftsky.business.event", Action: "create",
				Record: json.RawMessage(`{"$type":"social.craftsky.business.event","name":"Event","roles":[],"startsAt":"` + fixedCreatedAt + `","endsAt":"` + fixedCreatedAt + `","createdAt":"` + fixedCreatedAt + `","image":{"image":{"$type":"blob","ref":{"$link":"` + eventBlob + `"},"mimeType":"image/webp","size":30}}}`),
			},
		},
		{
			projector: index.NewImageSafetyCraftskyBusinessProfile(appliedProjector{}, key),
			event: tap.Event{
				URI: "at://did:plc:third-party-writer/social.craftsky.business.profile/self", CID: "business-cid", DID: actor,
				Rkey: "self", Collection: "social.craftsky.business.profile", Action: "create",
				Record: json.RawMessage(`{"$type":"social.craftsky.business.profile","products":[
					{"title":"One","uri":"https://example.com/one","image":{"image":{"$type":"blob","ref":{"$link":"` + productBlob + `"},"mimeType":"image/jpeg","size":40}}},
					{"title":"Two","uri":"https://example.com/two","image":{"image":{"$type":"blob","ref":{"$link":"` + sharedBlob + `"},"mimeType":"image/jpeg","size":10}}}
				]}`),
			},
		},
	}

	ctx := context.Background()
	for _, fixture := range events {
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		outcome, err := fixture.projector.Project(ctx, tx, imageSourceFromEvent(fixture.event))
		if err != nil {
			_ = tx.Rollback(ctx)
			t.Fatalf("project %s: %v", fixture.event.Collection, err)
		}
		if outcome.Kind != tap.OutcomeBlocked || outcome.Reason != tap.ReasonImageScanPending {
			_ = tx.Rollback(ctx)
			t.Fatalf("%s outcome=%+v, want image-scan blocked", fixture.event.Collection, outcome)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}
	}

	profile := index.NewImageSafetyBlueskyProfile(pool, key)
	profileEvent := tap.Event{
		URI: "at://did:plc:third-party-writer/app.bsky.actor.profile/self", CID: "profile-cid", DID: actor,
		Rkey: "self", Collection: "app.bsky.actor.profile", Action: "create",
		Record: json.RawMessage(`{
			"avatar":{"$type":"blob","ref":{"$link":"` + avatarBlob + `"},"mimeType":"image/jpeg","size":50},
			"banner":{"$type":"blob","ref":{"$link":"` + bannerBlob + `"},"mimeType":"image/png","size":60}
		}`),
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	profileOutcome, err := profile.Project(ctx, tx, imageSourceFromEvent(profileEvent))
	if err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("project Bluesky profile: %v", err)
	}
	if profileOutcome.Kind != tap.OutcomeBlocked || profileOutcome.Dependency.Key != profileEvent.URI.String() {
		_ = tx.Rollback(ctx)
		t.Fatalf("profile outcome=%+v, want image-scan blocked", profileOutcome)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	var results, jobs, requirements, candidates, sources int
	if err := pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM image_scan_results),
			(SELECT count(*) FROM image_scan_jobs),
			(SELECT count(*) FROM image_subject_requirements),
			(SELECT count(*) FROM profile_image_candidates),
			(SELECT count(*) FROM image_blob_sources)
	`).Scan(&results, &jobs, &requirements, &candidates, &sources); err != nil {
		t.Fatal(err)
	}
	if results != 6 || jobs != 6 {
		t.Fatalf("scan results/jobs=%d/%d, want 6/6 unique blobs", results, jobs)
	}
	if requirements != 5 || candidates != 2 || sources != 7 {
		t.Fatalf("requirements/candidates/sources=%d/%d/%d, want 5/2/7", requirements, candidates, sources)
	}
}
