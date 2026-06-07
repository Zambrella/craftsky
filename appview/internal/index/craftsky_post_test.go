// appview/internal/index/craftsky_post_test.go
package index_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

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
    did              TEXT        NOT NULL REFERENCES craftsky_profiles(did) ON DELETE CASCADE,
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
    pattern_difficulty TEXT,
    pattern_designer TEXT,
    pattern_publisher TEXT,
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
				"materials": ["merino"],
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
		projectTags     []string
		rawProject      string
	)
	if err := pool.QueryRow(context.Background(), `
		SELECT common_craft_type, common_status, common_title, project_tags, raw_project::text
		FROM craftsky_project_posts WHERE uri = $1`, ev.URI).
		Scan(&commonCraftType, &commonStatus, &commonTitle, &projectTags, &rawProject); err != nil {
		t.Fatalf("select project: %v", err)
	}
	if commonCraftType != "social.craftsky.feed.defs#knitting" || commonStatus == nil || *commonStatus != "social.craftsky.feed.defs#finished" || commonTitle == nil || *commonTitle != "Hitchhiker Shawl" {
		t.Fatalf("project common = craft=%q status=%v title=%v", commonCraftType, commonStatus, commonTitle)
	}
	if len(projectTags) != 1 || projectTags[0] != "fair-isle" {
		t.Fatalf("project_tags = %v, want [fair-isle]", projectTags)
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

func TestCraftskyPost_CascadeOnProfileDelete(t *testing.T) {
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
		Record:     json.RawMessage(`{"text":"will cascade","createdAt":"` + fixedCreatedAt + `"}`),
	}
	if err := idx.Handle(ctx, create); err != nil {
		t.Fatal(err)
	}

	// Delete the parent profile row directly. The FK cascade should fire.
	if _, err := pool.Exec(ctx,
		`DELETE FROM craftsky_profiles WHERE did = $1`, create.DID); err != nil {
		t.Fatalf("delete profile: %v", err)
	}

	var count int
	_ = pool.QueryRow(ctx,
		`SELECT count(*) FROM craftsky_posts WHERE did = $1`, create.DID).Scan(&count)
	if count != 0 {
		t.Errorf("post count = %d after profile delete, want 0 (cascade missing?)", count)
	}
}
