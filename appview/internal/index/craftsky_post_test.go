// appview/internal/index/craftsky_post_test.go
package index_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

// craftskyPostsDDL mirrors appview/migrations/000010_craftsky_posts.up.sql.
// craftsky_profiles is needed because craftsky_posts has a FK into it.
const craftskyPostsDDL = `
CREATE TABLE craftsky_profiles (
    did         TEXT        NOT NULL PRIMARY KEY,
    crafts      TEXT[]      NOT NULL DEFAULT '{}',
    record_cid  TEXT        NOT NULL,
    indexed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE craftsky_posts (
    uri              TEXT        NOT NULL PRIMARY KEY,
    did              TEXT        NOT NULL,
    rkey             TEXT        NOT NULL,
    cid              TEXT        NOT NULL,

    text             TEXT        NOT NULL,
    facets           JSONB,
    images           JSONB,

    reply_root_uri   TEXT,
    reply_root_cid   TEXT,
    reply_parent_uri TEXT,
    reply_parent_cid TEXT,

    quote_uri        TEXT,
    quote_cid        TEXT,

    tags             TEXT[]      NOT NULL DEFAULT '{}',

    is_project       BOOLEAN     NOT NULL DEFAULT false,
    project_craft_type TEXT,

    record           JSONB       NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL,
    indexed_at       TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (did, rkey)
);
CREATE TABLE craftsky_project_posts (
    uri TEXT PRIMARY KEY REFERENCES craftsky_posts(uri) ON DELETE CASCADE,
    raw_project JSONB NOT NULL,
    common_craft_type TEXT NOT NULL,
    common_status TEXT,
    common_title TEXT,
    common_duration TEXT,
    pattern_url TEXT,
    pattern_name TEXT,
    pattern_name_facets JSONB,
    pattern_difficulty TEXT,
    pattern_designer TEXT,
    pattern_designer_facets JSONB,
    pattern_publisher TEXT,
    pattern_publisher_facets JSONB,
    materials TEXT[] NOT NULL DEFAULT '{}',
    colors TEXT[] NOT NULL DEFAULT '{}',
    design_tags TEXT[] NOT NULL DEFAULT '{}',
    project_tags TEXT[] NOT NULL DEFAULT '{}',
    details_type TEXT,
    raw_details JSONB,
    knitting_project_type TEXT,
    knitting_project_subtype TEXT,
    knitting_yarn_weight TEXT,
    knitting_needle_size_mm TEXT,
    knitting_gauge JSONB,
    knitting_finished_size TEXT,
    crochet_project_type TEXT,
    crochet_project_subtype TEXT,
    crochet_yarn_weight TEXT,
    crochet_hook_size_mm TEXT,
    crochet_gauge JSONB,
    crochet_finished_size TEXT,
    quilting_project_type TEXT,
    quilting_project_subtype TEXT,
    quilting_piecing_technique TEXT,
    quilting_quilting_method TEXT,
    quilting_size TEXT,
    sewing_project_type TEXT,
    sewing_project_subtype TEXT,
    sewing_size_made TEXT,
    sewing_fit_notes TEXT,
    indexed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE craftsky_post_mentions (
    post_uri TEXT NOT NULL REFERENCES craftsky_posts(uri) ON DELETE CASCADE,
    mentioned_did TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    indexed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (post_uri, mentioned_did)
);
` + relationshipNotificationPolicyDDL + `
CREATE TABLE saved_posts (
    owner_did TEXT NOT NULL REFERENCES craftsky_profiles(did) ON DELETE CASCADE,
    post_uri  TEXT NOT NULL REFERENCES craftsky_posts(uri) ON DELETE CASCADE,
    folder_id UUID,
    saved_at  TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (owner_did, post_uri)
);
`

// seedCraftskyMember inserts a craftsky_profiles row so a post for did
// can pass the membership check.
func seedCraftskyMember(t *testing.T, pool *pgxpool.Pool, did string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO craftsky_profiles (did, record_cid) VALUES ($1, $2)`,
		did, "seed"); err != nil {
		t.Fatalf("seed craftsky_profiles: %v", err)
	}
}

func stringPointer(value string) *string {
	return &value
}

func assertStringQuery(t *testing.T, pool *pgxpool.Pool, query string, want []string) {
	t.Helper()
	rows, err := pool.Query(context.Background(), query)
	if err != nil {
		t.Fatalf("query strings: %v", err)
	}
	defer rows.Close()

	got := make([]string, 0, len(want))
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			t.Fatalf("scan string: %v", err)
		}
		got = append(got, value)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate strings: %v", err)
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("query strings = %q, want %q", got, want)
	}
}

// fixedCreatedAt is a constant timestamp used in test events so assertions
// on `created_at` are exact. Chosen arbitrarily; not load-bearing.
const fixedCreatedAt = "2026-05-04T12:00:00Z"

// testTime parses fixedCreatedAt for comparisons that need a time.Time.
func testTime(t *testing.T) time.Time {
	t.Helper()
	tt, err := time.Parse(time.RFC3339, fixedCreatedAt)
	if err != nil {
		t.Fatalf("parse fixed time: %v", err)
	}
	return tt
}

func TestCraftskyPost_OtherCollectionIgnored(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	idx := index.NewCraftskyPost(pool, testLogger())

	ev := tap.Event{
		URI:        "at://did:plc:b/app.bsky.feed.post/k",
		CID:        "c",
		DID:        "did:plc:b",
		Rkey:       "k",
		Collection: "app.bsky.feed.post",
		Action:     "create",
		Record:     json.RawMessage(`{}`),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Errorf("want nil for other collection; got %v", err)
	}
}

func TestCraftskyPost_UnknownAction(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	idx := index.NewCraftskyPost(pool, testLogger())

	ev := tap.Event{
		URI:        "at://did:plc:a/social.craftsky.feed.post/k",
		CID:        "c",
		DID:        "did:plc:a",
		Rkey:       "k",
		Collection: "social.craftsky.feed.post",
		Action:     "weird",
		Record:     json.RawMessage(`{"text":"hi","createdAt":"` + fixedCreatedAt + `"}`),
	}
	if err := idx.Handle(context.Background(), ev); err == nil {
		t.Error("want error for unknown action; got nil")
	}
}

func TestCraftskyPost_Create_NonMember_DroppedSilently(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	idx := index.NewCraftskyPost(pool, testLogger())

	ev := tap.Event{
		URI:        "at://did:plc:nm/social.craftsky.feed.post/k",
		CID:        "c",
		DID:        "did:plc:nm",
		Rkey:       "k",
		Collection: "social.craftsky.feed.post",
		Action:     "create",
		Record:     json.RawMessage(`{"text":"hi","createdAt":"` + fixedCreatedAt + `"}`),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Errorf("Handle should drop non-members without error; got %v", err)
	}

	var count int
	_ = pool.QueryRow(context.Background(),
		`SELECT count(*) FROM craftsky_posts WHERE did = $1`, ev.DID).Scan(&count)
	if count != 0 {
		t.Errorf("count = %d, want 0 (non-member must not be indexed)", count)
	}
}

func TestCraftskyPost_Create_PlainText(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	seedCraftskyMember(t, pool, "did:plc:m")
	idx := index.NewCraftskyPost(pool, testLogger())

	ev := tap.Event{
		URI:        "at://did:plc:m/social.craftsky.feed.post/r1",
		CID:        "bafy1",
		DID:        "did:plc:m",
		Rkey:       "r1",
		Collection: "social.craftsky.feed.post",
		Action:     "create",
		Record: json.RawMessage(`{
			"$type": "social.craftsky.feed.post",
			"text": "first post",
			"createdAt": "` + fixedCreatedAt + `"
		}`),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	var (
		uri, did, rkey, cid, text string
		facets, images            *string
		replyRoot, replyParent    *string
		quoteURI, quoteCID        *string
		tags                      []string
		createdAt                 time.Time
	)
	err := pool.QueryRow(context.Background(), `
		SELECT uri, did, rkey, cid, text,
		       facets::text, images::text,
		       reply_root_uri, reply_parent_uri,
		       quote_uri, quote_cid,
		       tags, created_at
		FROM craftsky_posts WHERE uri = $1`, ev.URI).
		Scan(&uri, &did, &rkey, &cid, &text,
			&facets, &images,
			&replyRoot, &replyParent,
			&quoteURI, &quoteCID,
			&tags, &createdAt)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if did != "did:plc:m" || rkey != "r1" || cid != "bafy1" {
		t.Errorf("ident = (%q,%q,%q)", did, rkey, cid)
	}
	if uri != string(ev.URI) {
		t.Errorf("uri = %q, want %q", uri, ev.URI)
	}
	if text != "first post" {
		t.Errorf("text = %q", text)
	}
	if facets != nil || images != nil {
		t.Errorf("facets/images should be NULL on plain text post; got facets=%v images=%v", facets, images)
	}
	if replyRoot != nil || replyParent != nil || quoteURI != nil || quoteCID != nil {
		t.Errorf("reply/quote columns should be NULL on plain text post")
	}
	if len(tags) != 0 {
		t.Errorf("tags = %v, want empty", tags)
	}
	if !createdAt.Equal(testTime(t)) {
		t.Errorf("created_at = %v, want %v", createdAt, testTime(t))
	}
}

func TestCraftskyPost_Create_WithProjectPayload_MaterializesProject(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	seedCraftskyMember(t, pool, "did:plc:p")
	idx := index.NewCraftskyPost(pool, testLogger())

	const projectJSON = `{
		"$type": "social.craftsky.feed.post",
		"text": "finished the shawl!",
		"createdAt": "` + fixedCreatedAt + `",
		"project": {
			"common": {
				"craftType": "social.craftsky.feed.defs#knitting",
				"status":    "social.craftsky.feed.defs#finished",
				"title":     "Hitchhiker Shawl",
				"materials": [{"text":"merino"}],
				"tags":      ["fair-isle"]
			}
		}
	}`
	ev := tap.Event{
		URI:        "at://did:plc:p/social.craftsky.feed.post/r",
		CID:        "bafyP",
		DID:        "did:plc:p",
		Rkey:       "r",
		Collection: "social.craftsky.feed.post",
		Action:     "create",
		Record:     json.RawMessage(projectJSON),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	var (
		isProject        bool
		projectCraftType *string
		tags             []string
		recRaw           string
	)
	if err := pool.QueryRow(context.Background(),
		`SELECT is_project, project_craft_type, tags, record::text FROM craftsky_posts WHERE uri = $1`, ev.URI).
		Scan(&isProject, &projectCraftType, &tags, &recRaw); err != nil {
		t.Fatalf("select: %v", err)
	}
	if !isProject {
		t.Fatalf("is_project = false, want true")
	}
	if projectCraftType == nil || *projectCraftType != "social.craftsky.feed.defs#knitting" {
		t.Fatalf("project_craft_type = %v", projectCraftType)
	}
	if len(tags) != 1 || tags[0] != "fair-isle" {
		t.Errorf("tags = %v, want [fair-isle]", tags)
	}

	var (
		commonCraftType string
		commonStatus    *string
		commonTitle     *string
		materials       []string
		projectTags     []string
		rawProject      string
	)
	if err := pool.QueryRow(context.Background(), `
		SELECT common_craft_type, common_status, common_title, materials, project_tags, raw_project::text
		FROM craftsky_project_posts WHERE uri = $1`, ev.URI).
		Scan(&commonCraftType, &commonStatus, &commonTitle, &materials, &projectTags, &rawProject); err != nil {
		t.Fatalf("select project: %v", err)
	}
	if commonCraftType != "social.craftsky.feed.defs#knitting" || commonStatus == nil || *commonStatus != "social.craftsky.feed.defs#finished" || commonTitle == nil || *commonTitle != "Hitchhiker Shawl" {
		t.Fatalf("project common = craft=%q status=%v title=%v", commonCraftType, commonStatus, commonTitle)
	}
	if len(projectTags) != 1 || projectTags[0] != "fair-isle" {
		t.Fatalf("project_tags = %v, want [fair-isle]", projectTags)
	}
	if len(materials) != 1 || materials[0] != "merino" {
		t.Fatalf("materials = %v, want [merino]", materials)
	}
	if rawProject == "" {
		t.Fatalf("raw_project empty")
	}

	// The raw record column must round-trip the project payload byte-for-meaning.
	var got map[string]any
	if err := json.Unmarshal([]byte(recRaw), &got); err != nil {
		t.Fatalf("decode record: %v", err)
	}
	project, ok := got["project"].(map[string]any)
	if !ok {
		t.Fatalf("record.project missing or not an object; got %T", got["project"])
	}
	common, ok := project["common"].(map[string]any)
	if !ok {
		t.Fatalf("record.project.common missing")
	}
	if common["craftType"] != "social.craftsky.feed.defs#knitting" {
		t.Errorf("craftType = %v", common["craftType"])
	}
	if common["title"] != "Hitchhiker Shawl" {
		t.Errorf("title = %v", common["title"])
	}
}

func TestCraftskyPost_Create_ProjectPatternFacetsMaterializeTagsAndMentions(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	seedCraftskyMember(t, pool, "did:plc:author")
	idx := index.NewCraftskyPost(pool, testLogger())

	const projectJSON = `{
		"$type": "social.craftsky.feed.post",
		"text": "caption #captiontag",
		"createdAt": "` + fixedCreatedAt + `",
		"facets": [{"index":{"byteStart":8,"byteEnd":19},"features":[{"$type":"app.bsky.richtext.facet#tag","tag":"CaptionTag"}]}],
		"project": {
			"common": {
				"craftType": "social.craftsky.feed.defs#knitting",
				"tags": ["structured-tag"],
				"materials": [{
					"text": "3m of @alice.craftsky.social #viscose fabric",
					"facets": [
						{"index":{"byteStart":6,"byteEnd":28},"features":[{"$type":"app.bsky.richtext.facet#mention","did":"did:plc:alice"}]},
						{"index":{"byteStart":29,"byteEnd":37},"features":[{"$type":"app.bsky.richtext.facet#tag","tag":"viscose"}]}
					]
				}],
				"pattern": {
					"name": "#hitchhiker",
					"nameFacets": [{"index":{"byteStart":0,"byteEnd":11},"features":[{"$type":"app.bsky.richtext.facet#tag","tag":"Hitchhiker"}]}],
					"designer": "@alice.craftsky.social",
					"designerFacets": [{"index":{"byteStart":0,"byteEnd":22},"features":[{"$type":"app.bsky.richtext.facet#mention","did":"did:plc:alice"}]}],
					"publisher": "@alice.craftsky.social",
					"publisherFacets": [{"index":{"byteStart":0,"byteEnd":22},"features":[{"$type":"app.bsky.richtext.facet#mention","did":"did:plc:alice"}]}]
				}
			}
		}
	}`
	ev := tap.Event{
		URI:        "at://did:plc:author/social.craftsky.feed.post/pattern-facets",
		CID:        "bafyPatternFacets",
		DID:        "did:plc:author",
		Rkey:       "pattern-facets",
		Collection: "social.craftsky.feed.post",
		Action:     "create",
		Record:     json.RawMessage(projectJSON),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	var postTags, projectTags []string
	if err := pool.QueryRow(context.Background(), `SELECT tags FROM craftsky_posts WHERE uri = $1`, ev.URI).Scan(&postTags); err != nil {
		t.Fatalf("select post tags: %v", err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT project_tags FROM craftsky_project_posts WHERE uri = $1`, ev.URI).Scan(&projectTags); err != nil {
		t.Fatalf("select project tags: %v", err)
	}
	wantTags := []string{"captiontag", "structured-tag", "hitchhiker", "viscose"}
	if strings.Join(postTags, ",") != strings.Join(wantTags, ",") {
		t.Fatalf("post tags = %v, want %v", postTags, wantTags)
	}
	if strings.Join(projectTags, ",") != strings.Join(wantTags, ",") {
		t.Fatalf("project tags = %v, want %v", projectTags, wantTags)
	}
	var nameFacetCount, designerFacetCount, publisherFacetCount int
	if err := pool.QueryRow(context.Background(), `
		SELECT
			COALESCE(jsonb_array_length(pattern_name_facets), 0),
			COALESCE(jsonb_array_length(pattern_designer_facets), 0),
			COALESCE(jsonb_array_length(pattern_publisher_facets), 0)
		FROM craftsky_project_posts WHERE uri = $1`, ev.URI).Scan(&nameFacetCount, &designerFacetCount, &publisherFacetCount); err != nil {
		t.Fatalf("select pattern facet counts: %v", err)
	}
	if nameFacetCount != 1 || designerFacetCount != 1 || publisherFacetCount != 1 {
		t.Fatalf("pattern facet counts = name:%d designer:%d publisher:%d, want 1 each", nameFacetCount, designerFacetCount, publisherFacetCount)
	}

	var mentions []string
	if err := pool.QueryRow(context.Background(), `
		SELECT array_agg(mentioned_did ORDER BY mentioned_did)
		FROM craftsky_post_mentions WHERE post_uri = $1`, ev.URI).Scan(&mentions); err != nil {
		t.Fatalf("select mentions: %v", err)
	}
	if len(mentions) != 1 || mentions[0] != "did:plc:alice" {
		t.Fatalf("mentions = %v, want [did:plc:alice]", mentions)
	}
}

func TestCraftskyPost_Create_ProjectFacetsStoreRawFutureShapes(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	seedCraftskyMember(t, pool, "did:plc:futurefacet")
	idx := index.NewCraftskyPost(pool, testLogger())

	const projectJSON = `{
		"$type": "social.craftsky.feed.post",
		"text": "caption",
		"createdAt": "` + fixedCreatedAt + `",
		"project": {
			"common": {
				"craftType": "social.craftsky.feed.defs#knitting",
				"tags": ["structured"],
				"materials": [{
					"text": "future #fiber",
					"facets": [{"index":{"byteStart":7,"byteEnd":13},"features":[{"$type":"app.bsky.richtext.facet#futureTag","tag":"fiber","future":true}]}]
				}],
				"pattern": {
					"name": "#futurepattern",
					"nameFacets": [{"index":{"byteStart":0,"byteEnd":14},"features":[{"$type":"app.bsky.richtext.facet#futureTag","tag":"futurepattern","future":true}]}]
				}
			}
		}
	}`
	ev := tap.Event{
		URI:        "at://did:plc:futurefacet/social.craftsky.feed.post/future-facets",
		CID:        "bafyFutureFacets",
		DID:        "did:plc:futurefacet",
		Rkey:       "future-facets",
		Collection: "social.craftsky.feed.post",
		Action:     "create",
		Record:     json.RawMessage(projectJSON),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	var rawProject, patternNameFacets string
	var projectTags []string
	if err := pool.QueryRow(context.Background(), `
		SELECT raw_project::text, pattern_name_facets::text, project_tags
		FROM craftsky_project_posts WHERE uri = $1`, ev.URI).Scan(&rawProject, &patternNameFacets, &projectTags); err != nil {
		t.Fatalf("select project: %v", err)
	}
	if !strings.Contains(rawProject, `"future": true`) && !strings.Contains(rawProject, `"future":true`) {
		t.Fatalf("raw_project lost future facet field: %s", rawProject)
	}
	if !strings.Contains(patternNameFacets, `futurepattern`) {
		t.Fatalf("pattern_name_facets not stored raw: %s", patternNameFacets)
	}
	if len(projectTags) != 1 || projectTags[0] != "structured" {
		t.Fatalf("project_tags = %v, want [structured]", projectTags)
	}
}

func TestCraftskyPost_Create_ProjectPatternFacetsIgnoreInvalidByteRanges(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	seedCraftskyMember(t, pool, "did:plc:invalidfacet")
	idx := index.NewCraftskyPost(pool, testLogger())

	ev := tap.Event{
		URI:        "at://did:plc:invalidfacet/social.craftsky.feed.post/r",
		CID:        "bafyInvalidPatternFacets",
		DID:        "did:plc:invalidfacet",
		Rkey:       "r",
		Collection: "social.craftsky.feed.post",
		Action:     "create",
		Record: json.RawMessage(`{
			"$type": "social.craftsky.feed.post",
			"text": "caption",
			"createdAt": "` + fixedCreatedAt + `",
			"project": {
				"common": {
					"craftType": "social.craftsky.feed.defs#knitting",
					"tags": ["structured-tag"],
					"pattern": {
						"name": "#hat",
						"nameFacets": [{"index":{"byteStart":0,"byteEnd":99},"features":[{"$type":"app.bsky.richtext.facet#tag","tag":"invalid-name-tag"}]}],
						"designerFacets": [{"index":{"byteStart":0,"byteEnd":8},"features":[{"$type":"app.bsky.richtext.facet#mention","did":"did:plc:ghost"}]}]
					}
				}
			}
		}`),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	var tags []string
	if err := pool.QueryRow(context.Background(), `SELECT tags FROM craftsky_posts WHERE uri = $1`, ev.URI).Scan(&tags); err != nil {
		t.Fatalf("select tags: %v", err)
	}
	if len(tags) != 1 || tags[0] != "structured-tag" {
		t.Fatalf("tags = %v, want [structured-tag]", tags)
	}

	var mentionCount int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM craftsky_post_mentions WHERE post_uri = $1`, ev.URI).Scan(&mentionCount); err != nil {
		t.Fatalf("select mention count: %v", err)
	}
	if mentionCount != 0 {
		t.Fatalf("mention count = %d, want 0", mentionCount)
	}
}

func TestCraftskyPost_Create_KnittingDetailsPopulatesOnlyKnittingColumns(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	seedCraftskyMember(t, pool, "did:plc:knit")
	idx := index.NewCraftskyPost(pool, testLogger())

	ev := tap.Event{
		URI:        "at://did:plc:knit/social.craftsky.feed.post/r",
		CID:        "bafyKnit",
		DID:        "did:plc:knit",
		Rkey:       "r",
		Collection: "social.craftsky.feed.post",
		Action:     "create",
		Record: json.RawMessage(`{
			"$type": "social.craftsky.feed.post",
			"text": "knitting details",
			"createdAt": "` + fixedCreatedAt + `",
			"project": {
				"common": {"craftType": "social.craftsky.feed.defs#knitting"},
				"details": {
					"$type": "social.craftsky.project.knitting#details",
					"projectType": "sweater",
					"projectSubtype": "cardigan",
					"yarnWeight": "dk",
					"needleSizeMm": "4.5",
					"gauge": {"stitches": 22, "measurement": 10, "unit": "cm"},
					"finishedSize": "M",
					"hookSizeMm": "5.0",
					"piecingTechnique": "strip",
					"quiltingMethod": "hand",
					"size": "throw",
					"sizeMade": "12",
					"fitNotes": "snug"
				}
			}
		}`),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	var (
		detailsType                                      *string
		rawDetails                                       *string
		knittingProjectType, knittingProjectSubtype      *string
		knittingYarnWeight, knittingNeedleSizeMM         *string
		knittingGauge, knittingFinishedSize              *string
		crochetProjectType, crochetProjectSubtype        *string
		crochetYarnWeight, crochetHookSizeMM             *string
		crochetGauge, crochetFinishedSize                *string
		quiltingProjectType, quiltingProjectSubtype      *string
		quiltingPiecingTechnique, quiltingQuiltingMethod *string
		quiltingSize                                     *string
		sewingProjectType, sewingProjectSubtype          *string
		sewingSizeMade, sewingFitNotes                   *string
	)
	if err := pool.QueryRow(context.Background(), `
		SELECT details_type, raw_details::text,
		       knitting_project_type, knitting_project_subtype, knitting_yarn_weight, knitting_needle_size_mm, knitting_gauge::text, knitting_finished_size,
		       crochet_project_type, crochet_project_subtype, crochet_yarn_weight, crochet_hook_size_mm, crochet_gauge::text, crochet_finished_size,
		       quilting_project_type, quilting_project_subtype, quilting_piecing_technique, quilting_quilting_method, quilting_size,
		       sewing_project_type, sewing_project_subtype, sewing_size_made, sewing_fit_notes
		FROM craftsky_project_posts WHERE uri = $1`, ev.URI).Scan(
		&detailsType, &rawDetails,
		&knittingProjectType, &knittingProjectSubtype, &knittingYarnWeight, &knittingNeedleSizeMM, &knittingGauge, &knittingFinishedSize,
		&crochetProjectType, &crochetProjectSubtype, &crochetYarnWeight, &crochetHookSizeMM, &crochetGauge, &crochetFinishedSize,
		&quiltingProjectType, &quiltingProjectSubtype, &quiltingPiecingTechnique, &quiltingQuiltingMethod, &quiltingSize,
		&sewingProjectType, &sewingProjectSubtype, &sewingSizeMade, &sewingFitNotes,
	); err != nil {
		t.Fatalf("select project details: %v", err)
	}
	if detailsType == nil || *detailsType != "social.craftsky.project.knitting#details" || rawDetails == nil || *rawDetails == "" {
		t.Fatalf("details type/raw = %v/%v", detailsType, rawDetails)
	}
	if knittingProjectType == nil || *knittingProjectType != "sweater" || knittingProjectSubtype == nil || *knittingProjectSubtype != "cardigan" || knittingYarnWeight == nil || *knittingYarnWeight != "dk" || knittingNeedleSizeMM == nil || *knittingNeedleSizeMM != "4.5" || knittingGauge == nil || knittingFinishedSize == nil || *knittingFinishedSize != "M" {
		t.Fatalf("knitting columns not populated as expected")
	}
	for name, got := range map[string]*string{
		"crochet_project_type": crochetProjectType, "crochet_project_subtype": crochetProjectSubtype, "crochet_yarn_weight": crochetYarnWeight, "crochet_hook_size_mm": crochetHookSizeMM, "crochet_gauge": crochetGauge, "crochet_finished_size": crochetFinishedSize,
		"quilting_project_type": quiltingProjectType, "quilting_project_subtype": quiltingProjectSubtype, "quilting_piecing_technique": quiltingPiecingTechnique, "quilting_quilting_method": quiltingQuiltingMethod, "quilting_size": quiltingSize,
		"sewing_project_type": sewingProjectType, "sewing_project_subtype": sewingProjectSubtype, "sewing_size_made": sewingSizeMade, "sewing_fit_notes": sewingFitNotes,
	} {
		if got != nil {
			t.Fatalf("%s = %q, want NULL for knitting details", name, *got)
		}
	}
}

func TestCraftskyPost_Create_ProjectReplyOrQuoteIsOrdinaryPost(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	seedCraftskyMember(t, pool, "did:plc:standalone")
	idx := index.NewCraftskyPost(pool, testLogger())

	cases := []tap.Event{
		{
			URI:        "at://did:plc:standalone/social.craftsky.feed.post/reply-project",
			CID:        "bafyReplyProject",
			DID:        "did:plc:standalone",
			Rkey:       "reply-project",
			Collection: "social.craftsky.feed.post",
			Action:     "create",
			Record: json.RawMessage(`{
				"$type":"social.craftsky.feed.post",
				"text":"reply project",
				"createdAt":"` + fixedCreatedAt + `",
				"reply": {
					"root": {"uri":"at://did:plc:other/social.craftsky.feed.post/root", "cid":"bafyRoot"},
					"parent": {"uri":"at://did:plc:other/social.craftsky.feed.post/root", "cid":"bafyRoot"}
				},
				"project":{"common":{"craftType":"social.craftsky.feed.defs#knitting","tags":["project-tag"]}}
			}`),
		},
		{
			URI:        "at://did:plc:standalone/social.craftsky.feed.post/quote-project",
			CID:        "bafyQuoteProject",
			DID:        "did:plc:standalone",
			Rkey:       "quote-project",
			Collection: "social.craftsky.feed.post",
			Action:     "create",
			Record: json.RawMessage(`{
				"$type":"social.craftsky.feed.post",
				"text":"quote project",
				"createdAt":"` + fixedCreatedAt + `",
				"embed": {
					"$type":"social.craftsky.feed.post#quoteEmbed",
					"record": {"uri":"at://did:plc:other/social.craftsky.feed.post/orig", "cid":"bafyOrig"}
				},
				"project":{"common":{"craftType":"social.craftsky.feed.defs#knitting","tags":["project-tag"]}}
			}`),
		},
	}

	for _, ev := range cases {
		if err := idx.Handle(context.Background(), ev); err != nil {
			t.Fatalf("Handle %s: %v", ev.Rkey, err)
		}
		var isProject bool
		var projectCraftType *string
		var tags []string
		if err := pool.QueryRow(context.Background(), `SELECT is_project, project_craft_type, tags FROM craftsky_posts WHERE uri = $1`, ev.URI).Scan(&isProject, &projectCraftType, &tags); err != nil {
			t.Fatalf("select base %s: %v", ev.Rkey, err)
		}
		if isProject || projectCraftType != nil || len(tags) != 0 {
			t.Fatalf("%s project state = isProject=%v craft=%v tags=%v, want ordinary post", ev.Rkey, isProject, projectCraftType, tags)
		}
		assertProjectChildCount(t, pool, ev.URI.String(), 0)
	}
}

func TestCraftskyPost_Create_GeneralPostHasNoProjectRow(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	seedCraftskyMember(t, pool, "did:plc:g")
	idx := index.NewCraftskyPost(pool, testLogger())

	ev := tap.Event{
		URI:        "at://did:plc:g/social.craftsky.feed.post/r",
		CID:        "bafyG",
		DID:        "did:plc:g",
		Rkey:       "r",
		Collection: "social.craftsky.feed.post",
		Action:     "create",
		Record:     json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"general","createdAt":"` + fixedCreatedAt + `"}`),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	var isProject bool
	var projectCraftType *string
	if err := pool.QueryRow(context.Background(), `SELECT is_project, project_craft_type FROM craftsky_posts WHERE uri = $1`, ev.URI).Scan(&isProject, &projectCraftType); err != nil {
		t.Fatalf("select base flags: %v", err)
	}
	if isProject || projectCraftType != nil {
		t.Fatalf("general post flags = (%v, %v), want false nil", isProject, projectCraftType)
	}
	var childCount int
	if err := pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM craftsky_project_posts WHERE uri = $1`, ev.URI).Scan(&childCount); err != nil {
		t.Fatalf("count child rows: %v", err)
	}
	if childCount != 0 {
		t.Fatalf("project child rows = %d, want 0", childCount)
	}
}

func TestCraftskyPost_Create_WithTagsFromFacets(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	seedCraftskyMember(t, pool, "did:plc:t")
	idx := index.NewCraftskyPost(pool, testLogger())

	// Two #tag features (one duplicate after lowercasing/trimming) and
	// one #link feature that must NOT contribute a tag.
	ev := tap.Event{
		URI:        "at://did:plc:t/social.craftsky.feed.post/r",
		CID:        "bafyT",
		DID:        "did:plc:t",
		Rkey:       "r",
		Collection: "social.craftsky.feed.post",
		Action:     "create",
		Record: json.RawMessage(`{
			"$type": "social.craftsky.feed.post",
			"text": "with tags #FairIsle #fairisle and a link",
			"createdAt": "` + fixedCreatedAt + `",
			"facets": [
				{
					"index": {"byteStart": 11, "byteEnd": 20},
					"features": [{"$type": "app.bsky.richtext.facet#tag", "tag": "FairIsle"}]
				},
				{
					"index": {"byteStart": 21, "byteEnd": 30},
					"features": [{"$type": "app.bsky.richtext.facet#tag", "tag": "  fairisle "}]
				},
				{
					"index": {"byteStart": 35, "byteEnd": 39},
					"features": [{"$type": "app.bsky.richtext.facet#link", "uri": "https://example.com"}]
				}
			]
		}`),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	var (
		tags   []string
		facets *string
	)
	if err := pool.QueryRow(context.Background(),
		`SELECT tags, facets::text FROM craftsky_posts WHERE uri = $1`, ev.URI).
		Scan(&tags, &facets); err != nil {
		t.Fatalf("select: %v", err)
	}
	if len(tags) != 1 || tags[0] != "fairisle" {
		t.Errorf("tags = %v, want [fairisle]", tags)
	}
	if facets == nil {
		t.Errorf("facets column should be populated; got NULL")
	}

	// Forward-compat: the stored facets must carry $type discriminators on
	// each feature so a future read endpoint can route renders. The generated
	// MarshalJSON adds these — this test guards against silently dropping them.
	var facetsParsed []map[string]any
	if err := json.Unmarshal([]byte(*facets), &facetsParsed); err != nil {
		t.Fatalf("parse facets JSON: %v", err)
	}
	if len(facetsParsed) != 3 {
		t.Fatalf("facets count = %d, want 3", len(facetsParsed))
	}
	features, _ := facetsParsed[0]["features"].([]any)
	if len(features) == 0 {
		t.Fatalf("facets[0].features missing")
	}
	feat0, _ := features[0].(map[string]any)
	if feat0["$type"] != "app.bsky.richtext.facet#tag" {
		t.Errorf("facets[0].features[0].$type = %v, want app.bsky.richtext.facet#tag", feat0["$type"])
	}
}

func TestCraftskyPost_MalformedCreatedAt_Errors(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	seedCraftskyMember(t, pool, "did:plc:bad")
	idx := index.NewCraftskyPost(pool, testLogger())

	ev := tap.Event{
		URI:        "at://did:plc:bad/social.craftsky.feed.post/r",
		CID:        "bafyBAD",
		DID:        "did:plc:bad",
		Rkey:       "r",
		Collection: "social.craftsky.feed.post",
		Action:     "create",
		Record:     json.RawMessage(`{"text":"x","createdAt":"not-a-timestamp"}`),
	}
	if err := idx.Handle(context.Background(), ev); err == nil {
		t.Error("want error for unparseable createdAt; got nil")
	}

	var count int
	_ = pool.QueryRow(context.Background(),
		`SELECT count(*) FROM craftsky_posts WHERE uri = $1`, ev.URI).Scan(&count)
	if count != 0 {
		t.Errorf("count = %d, want 0 (malformed event must not insert a row)", count)
	}
}

func TestCraftskyPost_Create_WithImages(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	seedCraftskyMember(t, pool, "did:plc:i")
	idx := index.NewCraftskyPost(pool, testLogger())

	ev := tap.Event{
		URI:        "at://did:plc:i/social.craftsky.feed.post/r",
		CID:        "bafyI",
		DID:        "did:plc:i",
		Rkey:       "r",
		Collection: "social.craftsky.feed.post",
		Action:     "create",
		Record: json.RawMessage(`{
			"$type": "social.craftsky.feed.post",
			"text": "post with images",
			"createdAt": "` + fixedCreatedAt + `",
			"images": [
				{
					"image": {"$type":"blob","ref":{"$link":"bafkreigxxxkul4e5rjz4fomqgn6ieeoxbcqeztmxjbrhnbpe7r44ya4ahe"},"mimeType":"image/jpeg","size":12345},
					"alt": "first photo"
				},
				{
					"image": {"$type":"blob","ref":{"$link":"bafkreidjq52a7nre4puzipwf3gwfkgnxftvbwnp3jppfogo7her2g3ai64"},"mimeType":"image/png","size":54321},
					"alt": "second photo"
				}
			]
		}`),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	var imagesJSON string
	if err := pool.QueryRow(context.Background(),
		`SELECT images::text FROM craftsky_posts WHERE uri = $1`, ev.URI).
		Scan(&imagesJSON); err != nil {
		t.Fatalf("select: %v", err)
	}
	var images []map[string]any
	if err := json.Unmarshal([]byte(imagesJSON), &images); err != nil {
		t.Fatalf("decode images: %v (raw=%s)", err, imagesJSON)
	}
	if len(images) != 2 {
		t.Fatalf("len(images) = %d, want 2", len(images))
	}
	if images[0]["cid"] != "bafkreigxxxkul4e5rjz4fomqgn6ieeoxbcqeztmxjbrhnbpe7r44ya4ahe" || images[0]["mime"] != "image/jpeg" || images[0]["alt"] != "first photo" {
		t.Errorf("images[0] = %v", images[0])
	}
	if images[1]["cid"] != "bafkreidjq52a7nre4puzipwf3gwfkgnxftvbwnp3jppfogo7her2g3ai64" || images[1]["mime"] != "image/png" || images[1]["alt"] != "second photo" {
		t.Errorf("images[1] = %v", images[1])
	}
}

func TestCraftskyPost_Create_WithImages_StoresSizeAndAspectRatio(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	seedCraftskyMember(t, pool, "did:plc:is")
	idx := index.NewCraftskyPost(pool, testLogger())

	ev := tap.Event{
		URI:        "at://did:plc:is/social.craftsky.feed.post/r",
		CID:        "bafyIS",
		DID:        "did:plc:is",
		Rkey:       "r",
		Collection: "social.craftsky.feed.post",
		Action:     "create",
		Record: json.RawMessage(`{
			"$type": "social.craftsky.feed.post",
			"text": "post with image metadata",
			"createdAt": "` + fixedCreatedAt + `",
			"images": [
				{
					"image": {"$type":"blob","ref":{"$link":"bafkreigxxxkul4e5rjz4fomqgn6ieeoxbcqeztmxjbrhnbpe7r44ya4ahe"},"mimeType":"image/jpeg","size":253496},
					"alt": "project photo",
					"aspectRatio": {"width":919,"height":2000}
				}
			]
		}`),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	var imagesJSON string
	if err := pool.QueryRow(context.Background(),
		`SELECT images::text FROM craftsky_posts WHERE uri = $1`, ev.URI).
		Scan(&imagesJSON); err != nil {
		t.Fatalf("select: %v", err)
	}
	var images []map[string]any
	if err := json.Unmarshal([]byte(imagesJSON), &images); err != nil {
		t.Fatalf("decode images: %v (raw=%s)", err, imagesJSON)
	}
	if len(images) != 1 {
		t.Fatalf("len(images) = %d, want 1", len(images))
	}
	if images[0]["cid"] != "bafkreigxxxkul4e5rjz4fomqgn6ieeoxbcqeztmxjbrhnbpe7r44ya4ahe" || images[0]["mime"] != "image/jpeg" || images[0]["alt"] != "project photo" {
		t.Fatalf("images[0] core fields = %v", images[0])
	}
	if images[0]["size"] != float64(253496) {
		t.Fatalf("images[0].size = %v, want 253496", images[0]["size"])
	}
	aspect, ok := images[0]["aspectRatio"].(map[string]any)
	if !ok {
		t.Fatalf("images[0].aspectRatio = %T %v", images[0]["aspectRatio"], images[0]["aspectRatio"])
	}
	if aspect["width"] != float64(919) || aspect["height"] != float64(2000) {
		t.Fatalf("aspectRatio = %v", aspect)
	}
}

func TestCraftskyPost_Create_WithReply(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	seedCraftskyMember(t, pool, "did:plc:r")
	idx := index.NewCraftskyPost(pool, testLogger())

	ev := tap.Event{
		URI:        "at://did:plc:r/social.craftsky.feed.post/reply",
		CID:        "bafyR",
		DID:        "did:plc:r",
		Rkey:       "reply",
		Collection: "social.craftsky.feed.post",
		Action:     "create",
		Record: json.RawMessage(`{
			"$type": "social.craftsky.feed.post",
			"text": "reply text",
			"createdAt": "` + fixedCreatedAt + `",
			"reply": {
				"root":   {"uri": "at://did:plc:author/social.craftsky.feed.post/root",   "cid": "bafyRoot"},
				"parent": {"uri": "at://did:plc:author/social.craftsky.feed.post/parent", "cid": "bafyParent"}
			}
		}`),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	var rootURI, rootCID, parentURI, parentCID string
	if err := pool.QueryRow(context.Background(), `
		SELECT reply_root_uri, reply_root_cid, reply_parent_uri, reply_parent_cid
		FROM craftsky_posts WHERE uri = $1`, ev.URI).
		Scan(&rootURI, &rootCID, &parentURI, &parentCID); err != nil {
		t.Fatalf("select: %v", err)
	}
	if rootURI != "at://did:plc:author/social.craftsky.feed.post/root" || rootCID != "bafyRoot" {
		t.Errorf("root = (%q, %q)", rootURI, rootCID)
	}
	if parentURI != "at://did:plc:author/social.craftsky.feed.post/parent" || parentCID != "bafyParent" {
		t.Errorf("parent = (%q, %q)", parentURI, parentCID)
	}
}

func TestCraftskyPost_Create_WithQuote(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	seedCraftskyMember(t, pool, "did:plc:q")
	idx := index.NewCraftskyPost(pool, testLogger())

	ev := tap.Event{
		URI:        "at://did:plc:q/social.craftsky.feed.post/quote",
		CID:        "bafyQ",
		DID:        "did:plc:q",
		Rkey:       "quote",
		Collection: "social.craftsky.feed.post",
		Action:     "create",
		Record: json.RawMessage(`{
			"$type": "social.craftsky.feed.post",
			"text": "quoting another post",
			"createdAt": "` + fixedCreatedAt + `",
			"embed": {
				"$type": "social.craftsky.feed.post#quoteEmbed",
				"record": {"uri": "at://did:plc:other/social.craftsky.feed.post/orig", "cid": "bafyOrig"}
			}
		}`),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	var quoteURI, quoteCID string
	if err := pool.QueryRow(context.Background(), `
		SELECT quote_uri, quote_cid FROM craftsky_posts WHERE uri = $1`, ev.URI).
		Scan(&quoteURI, &quoteCID); err != nil {
		t.Fatalf("select: %v", err)
	}
	if quoteURI != "at://did:plc:other/social.craftsky.feed.post/orig" || quoteCID != "bafyOrig" {
		t.Errorf("quote = (%q, %q)", quoteURI, quoteCID)
	}
}

func TestCraftskyPost_Replay_PreservesIndexedAt(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	seedCraftskyMember(t, pool, "did:plc:rp")
	idx := index.NewCraftskyPost(pool, testLogger())

	ev := tap.Event{
		URI:        "at://did:plc:rp/social.craftsky.feed.post/r",
		CID:        "bafyRP",
		DID:        "did:plc:rp",
		Rkey:       "r",
		Collection: "social.craftsky.feed.post",
		Action:     "create",
		Record:     json.RawMessage(`{"text":"once","createdAt":"` + fixedCreatedAt + `"}`),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatal(err)
	}

	var firstIndexedAt string
	if err := pool.QueryRow(context.Background(),
		`SELECT indexed_at::text FROM craftsky_posts WHERE uri = $1`, ev.URI).
		Scan(&firstIndexedAt); err != nil {
		t.Fatalf("select first indexed_at: %v", err)
	}

	// Replay identical event.
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatal(err)
	}

	var secondIndexedAt string
	if err := pool.QueryRow(context.Background(),
		`SELECT indexed_at::text FROM craftsky_posts WHERE uri = $1`, ev.URI).
		Scan(&secondIndexedAt); err != nil {
		t.Fatalf("select second indexed_at: %v", err)
	}

	if firstIndexedAt != secondIndexedAt {
		t.Errorf("indexed_at changed on replay: %q -> %q", firstIndexedAt, secondIndexedAt)
	}
}

func TestCraftskyPost_Replay_PreservesMentionIndexedAt(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	seedCraftskyMember(t, pool, "did:plc:mentionreplay")
	idx := index.NewCraftskyPost(pool, testLogger())
	ctx := context.Background()

	ev := tap.Event{
		URI:        "at://did:plc:mentionreplay/social.craftsky.feed.post/r",
		CID:        "bafyMentionReplay",
		DID:        "did:plc:mentionreplay",
		Rkey:       "r",
		Collection: "social.craftsky.feed.post",
		Action:     "create",
		Record: json.RawMessage(`{
			"$type": "social.craftsky.feed.post",
			"text": "hi @alice",
			"createdAt": "` + fixedCreatedAt + `",
			"facets": [{"index":{"byteStart":3,"byteEnd":9},"features":[{"$type":"app.bsky.richtext.facet#mention","did":"did:plc:alice"}]}]
		}`),
	}
	if err := idx.Handle(ctx, ev); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	sentinel := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	if _, err := pool.Exec(ctx, `UPDATE craftsky_post_mentions SET indexed_at = $1 WHERE post_uri = $2 AND mentioned_did = $3`, sentinel, ev.URI, "did:plc:alice"); err != nil {
		t.Fatalf("set sentinel indexed_at: %v", err)
	}
	if err := idx.Handle(ctx, ev); err != nil {
		t.Fatalf("replay Handle: %v", err)
	}

	var indexedAt time.Time
	if err := pool.QueryRow(ctx, `SELECT indexed_at FROM craftsky_post_mentions WHERE post_uri = $1 AND mentioned_did = $2`, ev.URI, "did:plc:alice").Scan(&indexedAt); err != nil {
		t.Fatalf("select mention indexed_at: %v", err)
	}
	if !indexedAt.Equal(sentinel) {
		t.Fatalf("mention indexed_at = %s, want %s", indexedAt, sentinel)
	}
}

func TestCraftskyPost_Update_NewCID_ReplacesRow(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	seedCraftskyMember(t, pool, "did:plc:u")
	idx := index.NewCraftskyPost(pool, testLogger())
	ctx := context.Background()

	create := tap.Event{
		URI:        "at://did:plc:u/social.craftsky.feed.post/r",
		CID:        "bafy1",
		DID:        "did:plc:u",
		Rkey:       "r",
		Collection: "social.craftsky.feed.post",
		Action:     "create",
		Record:     json.RawMessage(`{"text":"original","createdAt":"` + fixedCreatedAt + `"}`),
	}
	if err := idx.Handle(ctx, create); err != nil {
		t.Fatal(err)
	}

	var firstIndexedAt string
	if err := pool.QueryRow(ctx,
		`SELECT indexed_at::text FROM craftsky_posts WHERE uri = $1`, create.URI).
		Scan(&firstIndexedAt); err != nil {
		t.Fatalf("select first indexed_at: %v", err)
	}

	update := create
	update.CID = "bafy2"
	update.Action = "update"
	update.Record = json.RawMessage(`{"text":"edited","createdAt":"` + fixedCreatedAt + `"}`)
	if err := idx.Handle(ctx, update); err != nil {
		t.Fatal(err)
	}

	var (
		text, cid       string
		secondIndexedAt string
	)
	if err := pool.QueryRow(ctx,
		`SELECT text, cid, indexed_at::text FROM craftsky_posts WHERE uri = $1`, create.URI).
		Scan(&text, &cid, &secondIndexedAt); err != nil {
		t.Fatalf("select after update: %v", err)
	}
	if text != "edited" {
		t.Errorf("text = %q, want edited", text)
	}
	if cid != "bafy2" {
		t.Errorf("cid = %q, want bafy2", cid)
	}
	if secondIndexedAt == firstIndexedAt {
		t.Errorf("indexed_at did not advance: %q stayed", firstIndexedAt)
	}
}

func TestCraftskyPost_Update_ProjectTagsRefreshWhenOnlyCaptionFacetsChange(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	seedCraftskyMember(t, pool, "did:plc:projecttags")
	idx := index.NewCraftskyPost(pool, testLogger())
	ctx := context.Background()

	create := tap.Event{
		URI:        "at://did:plc:projecttags/social.craftsky.feed.post/r",
		CID:        "bafyProjectTags1",
		DID:        "did:plc:projecttags",
		Rkey:       "r",
		Collection: "social.craftsky.feed.post",
		Action:     "create",
		Record: json.RawMessage(`{
			"$type": "social.craftsky.feed.post",
			"text": "caption #one",
			"createdAt": "` + fixedCreatedAt + `",
			"facets": [{"index":{"byteStart":8,"byteEnd":12},"features":[{"$type":"app.bsky.richtext.facet#tag","tag":"one"}]}],
			"project": {"common":{"craftType":"social.craftsky.feed.defs#knitting"}}
		}`),
	}
	if err := idx.Handle(ctx, create); err != nil {
		t.Fatalf("create Handle: %v", err)
	}

	update := create
	update.Action = "update"
	update.CID = "bafyProjectTags2"
	update.Record = json.RawMessage(`{
		"$type": "social.craftsky.feed.post",
		"text": "caption #two",
		"createdAt": "` + fixedCreatedAt + `",
		"facets": [{"index":{"byteStart":8,"byteEnd":12},"features":[{"$type":"app.bsky.richtext.facet#tag","tag":"two"}]}],
		"project": {"common":{"craftType":"social.craftsky.feed.defs#knitting"}}
	}`)
	if err := idx.Handle(ctx, update); err != nil {
		t.Fatalf("update Handle: %v", err)
	}

	var projectTags []string
	if err := pool.QueryRow(ctx, `SELECT project_tags FROM craftsky_project_posts WHERE uri = $1`, create.URI).Scan(&projectTags); err != nil {
		t.Fatalf("select project tags: %v", err)
	}
	if len(projectTags) != 1 || projectTags[0] != "two" {
		t.Fatalf("project_tags = %v, want [two]", projectTags)
	}
}

func TestCraftskyPost_ProjectUpdateRemovalUnknownDetailsAndDeleteConverge(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	seedCraftskyMember(t, pool, "did:plc:conv")
	idx := index.NewCraftskyPost(pool, testLogger())
	ctx := context.Background()

	create := tap.Event{
		URI:        "at://did:plc:conv/social.craftsky.feed.post/r",
		CID:        "bafyCreate",
		DID:        "did:plc:conv",
		Rkey:       "r",
		Collection: "social.craftsky.feed.post",
		Action:     "create",
		Record: json.RawMessage(`{
			"$type":"social.craftsky.feed.post",
			"text":"project",
			"createdAt":"` + fixedCreatedAt + `",
			"project":{"common":{"craftType":"social.craftsky.feed.defs#knitting"}}
		}`),
	}
	if err := idx.Handle(ctx, create); err != nil {
		t.Fatalf("create Handle: %v", err)
	}
	assertProjectChildCount(t, pool, create.URI.String(), 1)

	removeProject := create
	removeProject.Action = "update"
	removeProject.CID = "bafyGeneral"
	removeProject.Record = json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"now general","createdAt":"` + fixedCreatedAt + `"}`)
	if err := idx.Handle(ctx, removeProject); err != nil {
		t.Fatalf("remove project Handle: %v", err)
	}
	var isProject bool
	var projectCraftType *string
	if err := pool.QueryRow(ctx, `SELECT is_project, project_craft_type FROM craftsky_posts WHERE uri = $1`, create.URI).Scan(&isProject, &projectCraftType); err != nil {
		t.Fatalf("select base after removal: %v", err)
	}
	if isProject || projectCraftType != nil {
		t.Fatalf("after project removal flags = (%v, %v), want false nil", isProject, projectCraftType)
	}
	assertProjectChildCount(t, pool, create.URI.String(), 0)

	unknownProject := create
	unknownProject.Action = "update"
	unknownProject.CID = "bafyUnknown"
	unknownProject.Record = json.RawMessage(`{
		"$type":"social.craftsky.feed.post",
		"text":"future project",
		"createdAt":"` + fixedCreatedAt + `",
		"project":{
			"common":{"craftType":"social.craftsky.feed.defs#future"},
			"details":{"$type":"social.craftsky.project.future#details","newField":"kept","projectType":"future-type"}
		}
	}`)
	if err := idx.Handle(ctx, unknownProject); err != nil {
		t.Fatalf("unknown project Handle: %v", err)
	}
	var detailsType, rawDetails, knittingProjectType, crochetProjectType, quiltingProjectType, sewingProjectType *string
	if err := pool.QueryRow(ctx, `
		SELECT details_type, raw_details::text, knitting_project_type, crochet_project_type, quilting_project_type, sewing_project_type
		FROM craftsky_project_posts WHERE uri = $1`, create.URI).
		Scan(&detailsType, &rawDetails, &knittingProjectType, &crochetProjectType, &quiltingProjectType, &sewingProjectType); err != nil {
		t.Fatalf("select unknown details: %v", err)
	}
	if detailsType == nil || *detailsType != "social.craftsky.project.future#details" || rawDetails == nil || !strings.Contains(*rawDetails, "newField") {
		t.Fatalf("unknown details not preserved: type=%v raw=%v", detailsType, rawDetails)
	}
	if knittingProjectType != nil || crochetProjectType != nil || quiltingProjectType != nil || sewingProjectType != nil {
		t.Fatalf("known craft columns populated for unknown details: knitting=%v crochet=%v quilting=%v sewing=%v", knittingProjectType, crochetProjectType, quiltingProjectType, sewingProjectType)
	}

	del := tap.Event{URI: create.URI, DID: create.DID, Rkey: create.Rkey, Collection: "social.craftsky.feed.post", Action: "delete"}
	if err := idx.Handle(ctx, del); err != nil {
		t.Fatalf("delete Handle: %v", err)
	}
	var baseCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM craftsky_posts WHERE uri = $1`, create.URI).Scan(&baseCount); err != nil {
		t.Fatalf("count base after delete: %v", err)
	}
	if baseCount != 0 {
		t.Fatalf("base rows after delete = %d, want 0", baseCount)
	}
	assertProjectChildCount(t, pool, create.URI.String(), 0)
}

func assertProjectChildCount(t *testing.T, pool *pgxpool.Pool, uri string, want int) {
	t.Helper()
	var got int
	if err := pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM craftsky_project_posts WHERE uri = $1`, uri).Scan(&got); err != nil {
		t.Fatalf("count project rows: %v", err)
	}
	if got != want {
		t.Fatalf("project child rows for %s = %d, want %d", uri, got, want)
	}
}

func TestCraftskyPost_Update_BeforeCreate_TreatedAsCreate(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	seedCraftskyMember(t, pool, "did:plc:ub")
	idx := index.NewCraftskyPost(pool, testLogger())

	ev := tap.Event{
		URI:        "at://did:plc:ub/social.craftsky.feed.post/r",
		CID:        "bafyUB",
		DID:        "did:plc:ub",
		Rkey:       "r",
		Collection: "social.craftsky.feed.post",
		Action:     "update",
		Record:     json.RawMessage(`{"text":"hi","createdAt":"` + fixedCreatedAt + `"}`),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	var count int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM craftsky_posts WHERE uri = $1`, ev.URI).
		Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1 (update-before-create must insert)", count)
	}
}

func TestCraftskyPost_Delete(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	seedCraftskyMember(t, pool, "did:plc:d")
	idx := index.NewCraftskyPost(pool, testLogger())
	ctx := context.Background()

	create := tap.Event{
		URI:        "at://did:plc:d/social.craftsky.feed.post/r",
		CID:        "bafyD",
		DID:        "did:plc:d",
		Rkey:       "r",
		Collection: "social.craftsky.feed.post",
		Action:     "create",
		Record:     json.RawMessage(`{"text":"to delete","createdAt":"` + fixedCreatedAt + `"}`),
	}
	if err := idx.Handle(ctx, create); err != nil {
		t.Fatal(err)
	}

	del := tap.Event{
		URI: create.URI, DID: create.DID, Rkey: create.Rkey,
		Collection: "social.craftsky.feed.post",
		Action:     "delete",
	}
	if err := idx.Handle(ctx, del); err != nil {
		t.Fatalf("delete Handle: %v", err)
	}

	var count int
	_ = pool.QueryRow(ctx,
		`SELECT count(*) FROM craftsky_posts WHERE uri = $1`, create.URI).Scan(&count)
	if count != 0 {
		t.Errorf("count = %d, want 0 after delete", count)
	}
}

func TestCraftskyPost_Delete_Nonexistent_NoOp(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	idx := index.NewCraftskyPost(pool, testLogger())

	del := tap.Event{
		URI:        "at://did:plc:none/social.craftsky.feed.post/r",
		DID:        "did:plc:none",
		Rkey:       "r",
		Collection: "social.craftsky.feed.post",
		Action:     "delete",
	}
	if err := idx.Handle(context.Background(), del); err != nil {
		t.Errorf("delete-of-nonexistent should be no-op; got %v", err)
	}
}

func TestCraftskyPost_DeleteCleanupErrorRedactsSavedURI(t *testing.T) {
	const privateURISentinel = "at://did:plc:private-owner/social.craftsky.feed.post/private-saved-uri"
	ddlWithoutSavedPosts := strings.Replace(craftskyPostsDDL, `
CREATE TABLE saved_posts (
    owner_did TEXT NOT NULL REFERENCES craftsky_profiles(did) ON DELETE CASCADE,
    post_uri  TEXT NOT NULL REFERENCES craftsky_posts(uri) ON DELETE CASCADE,
    folder_id UUID,
    saved_at  TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (owner_did, post_uri)
);
`, "", 1)
	pool := testdb.WithSchema(t, ddlWithoutSavedPosts)
	idx := index.NewCraftskyPost(pool, testLogger())

	err := idx.Handle(context.Background(), tap.Event{
		URI:        syntax.ATURI(privateURISentinel),
		DID:        "did:plc:private-owner",
		Rkey:       "private-saved-uri",
		Collection: "social.craftsky.feed.post",
		Action:     "delete",
	})
	if err == nil {
		t.Fatal("delete without saved_posts table succeeded, want cleanup error")
	}
	if strings.Contains(err.Error(), privateURISentinel) {
		t.Fatalf("cleanup error leaked private saved URI: %v", err)
	}
}

func TestCraftskyPost_Delete_CleansExactAndDescendantSavesOnly(t *testing.T) {
	t.Parallel()

	const (
		rootURI       = "at://did:plc:author/social.craftsky.feed.post/root"
		commentURI    = "at://did:plc:author/social.craftsky.feed.post/comment"
		parentURI     = "at://did:plc:author/social.craftsky.feed.post/parent"
		targetURI     = "at://did:plc:author/social.craftsky.feed.post/target"
		safetyURI     = "at://did:plc:author/social.craftsky.feed.post/safety"
		unrelatedURI  = "at://did:plc:author/social.craftsky.feed.post/unrelated"
		missingParent = "at://did:plc:author/social.craftsky.feed.post/missing"
	)

	tests := []struct {
		name              string
		deleteURI         string
		wantSaved         []string
		wantRetainedPosts []string
	}{
		{
			name:              "exact target",
			deleteURI:         targetURI,
			wantSaved:         []string{commentURI, parentURI, rootURI, safetyURI, unrelatedURI},
			wantRetainedPosts: []string{commentURI, parentURI, rootURI, safetyURI, unrelatedURI},
		},
		{
			name:              "intermediate ancestor",
			deleteURI:         parentURI,
			wantSaved:         []string{commentURI, rootURI, safetyURI, unrelatedURI},
			wantRetainedPosts: []string{commentURI, rootURI, safetyURI, targetURI, unrelatedURI},
		},
		{
			name:              "root including missing-parent safety-net descendant",
			deleteURI:         rootURI,
			wantSaved:         []string{unrelatedURI},
			wantRetainedPosts: []string{commentURI, parentURI, safetyURI, targetURI, unrelatedURI},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			pool := testdb.WithSchema(t, craftskyPostsDDL)
			ctx := context.Background()
			seedCraftskyMember(t, pool, "did:plc:alice")
			seedCraftskyMember(t, pool, "did:plc:author")

			posts := []struct {
				uri    string
				root   *string
				parent *string
			}{
				{uri: rootURI},
				{uri: commentURI, root: stringPointer(rootURI), parent: stringPointer(rootURI)},
				{uri: parentURI, root: stringPointer(rootURI), parent: stringPointer(commentURI)},
				{uri: targetURI, root: stringPointer(rootURI), parent: stringPointer(parentURI)},
				{uri: safetyURI, root: stringPointer(rootURI), parent: stringPointer(missingParent)},
				{uri: unrelatedURI},
			}
			for i, post := range posts {
				if _, err := pool.Exec(ctx, `
					INSERT INTO craftsky_posts (
						uri, did, rkey, cid, text, record, created_at,
						reply_root_uri, reply_parent_uri
					) VALUES ($1, 'did:plc:author', $2, $3, '', '{}', $4, $5, $6)
				`, post.uri, fmt.Sprintf("r%d", i), fmt.Sprintf("cid%d", i), testTime(t), post.root, post.parent); err != nil {
					t.Fatalf("seed post %s: %v", post.uri, err)
				}
				if _, err := pool.Exec(ctx, `
					INSERT INTO saved_posts (owner_did, post_uri, saved_at)
					VALUES ('did:plc:alice', $1, $2)
				`, post.uri, testTime(t)); err != nil {
					t.Fatalf("seed save %s: %v", post.uri, err)
				}
			}

			idx := index.NewCraftskyPost(pool, testLogger())
			if err := idx.Handle(ctx, tap.Event{
				URI:        syntax.ATURI(test.deleteURI),
				DID:        "did:plc:author",
				Collection: "social.craftsky.feed.post",
				Action:     "delete",
			}); err != nil {
				t.Fatalf("delete Handle: %v", err)
			}

			assertStringQuery(t, pool, `SELECT post_uri FROM saved_posts ORDER BY post_uri`, test.wantSaved)
			assertStringQuery(t, pool, `SELECT uri FROM craftsky_posts ORDER BY uri`, test.wantRetainedPosts)
		})
	}
}

func TestCraftskyPost_PreservesPublicRecordOnMembershipDelete(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	seedCraftskyMember(t, pool, "did:plc:cc")
	idx := index.NewCraftskyPost(pool, testLogger())
	ctx := context.Background()

	create := tap.Event{
		URI:        "at://did:plc:cc/social.craftsky.feed.post/r",
		CID:        "bafyCC",
		DID:        "did:plc:cc",
		Rkey:       "r",
		Collection: "social.craftsky.feed.post",
		Action:     "create",
		Record:     json.RawMessage(`{"text":"must survive membership loss","createdAt":"` + fixedCreatedAt + `"}`),
	}
	if err := idx.Handle(ctx, create); err != nil {
		t.Fatal(err)
	}

	// Membership loss hides public records at read time; it must not delete the
	// indexed source record that can become eligible again on rejoin.
	if _, err := pool.Exec(ctx,
		`DELETE FROM craftsky_profiles WHERE did = $1`, create.DID); err != nil {
		t.Fatalf("delete profile: %v", err)
	}

	var count int
	_ = pool.QueryRow(ctx,
		`SELECT count(*) FROM craftsky_posts WHERE did = $1`, create.DID).Scan(&count)
	if count != 1 {
		t.Errorf("post count = %d after profile delete, want 1 retained public record", count)
	}
}
