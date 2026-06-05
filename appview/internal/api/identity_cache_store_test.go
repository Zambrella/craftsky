package api_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/testdb"
)

type exactResolveFakeResolver struct {
	didByHandle    map[string]syntax.DID
	handleByDID    map[string]syntax.Handle
	errByHandle    map[string]error
	errHandleByDID map[string]error
}

func (f exactResolveFakeResolver) ResolveDID(_ context.Context, handle syntax.Handle) (syntax.DID, error) {
	if err := f.errByHandle[handle.String()]; err != nil {
		return "", err
	}
	did := f.didByHandle[handle.String()]
	if did == "" {
		return "", errors.New("not found")
	}
	return did, nil
}

func (f exactResolveFakeResolver) ResolveHandle(_ context.Context, did syntax.DID) (syntax.Handle, error) {
	if err := f.errHandleByDID[did.String()]; err != nil {
		return "", err
	}
	handle := f.handleByDID[did.String()]
	if handle == "" {
		return "", errors.New("not found")
	}
	return handle, nil
}

const facetStoreDDL = `
CREATE TABLE craftsky_profiles (
    did         TEXT        NOT NULL PRIMARY KEY,
    crafts      TEXT[]      NOT NULL DEFAULT '{}',
    record_cid  TEXT        NOT NULL,
    indexed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE bluesky_profiles (
    did          TEXT        NOT NULL PRIMARY KEY,
    display_name TEXT,
    description  TEXT,
    avatar_cid   TEXT,
    avatar_mime  TEXT,
    banner_cid   TEXT,
    banner_mime  TEXT,
    record_cid   TEXT        NOT NULL,
    indexed_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE atproto_follows (
    uri         TEXT        NOT NULL PRIMARY KEY,
    did         TEXT        NOT NULL,
    rkey        TEXT        NOT NULL,
    cid         TEXT        NOT NULL,
    subject_did TEXT        NOT NULL,
    record      JSONB       NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL,
    indexed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (did, rkey),
    UNIQUE (did, subject_did)
);
CREATE TABLE atproto_identity_cache (
    did          TEXT        NOT NULL PRIMARY KEY,
    handle       TEXT        NOT NULL,
    handle_lower TEXT        NOT NULL UNIQUE,
    resolved_at  TIMESTAMPTZ NOT NULL,
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
`

func TestFacetStoreSearchMentionSuggestionsUsesFreshSeparateIdentityCache(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, facetStoreDDL)
	ctx := context.Background()
	now := time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)

	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_profiles (did, crafts, record_cid) VALUES
			('did:plc:alice', '{}', 'cid-alice'),
			('did:plc:alina', '{}', 'cid-alina'),
			('did:plc:stale', '{}', 'cid-stale')
	`); err != nil {
		t.Fatalf("seed craftsky profiles: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO bluesky_profiles (did, display_name, record_cid) VALUES
			('did:plc:alice', 'Alice', 'cid-bsky-alice'),
			('did:plc:alina', NULL, 'cid-bsky-alina'),
			('did:plc:stale', 'Alison Stale', 'cid-bsky-stale'),
			('did:plc:mallory', 'Alice Elsewhere', 'cid-bsky-mallory')
	`); err != nil {
		t.Fatalf("seed bluesky profiles: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO atproto_identity_cache (did, handle, handle_lower, resolved_at) VALUES
			('did:plc:alice', 'alice.craftsky.social', 'alice.craftsky.social', $1),
			('did:plc:alina', 'alina.craftsky.social', 'alina.craftsky.social', $2),
			('did:plc:stale', 'alistale.craftsky.social', 'alistale.craftsky.social', $3),
			('did:plc:mallory', 'alice.elsewhere.example', 'alice.elsewhere.example', $1)
	`, now.Add(-23*time.Hour), now.Add(-24*time.Hour), now.Add(-24*time.Hour-time.Minute)); err != nil {
		t.Fatalf("seed identity cache: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO atproto_follows (uri, did, rkey, cid, subject_did, record, created_at) VALUES
			('at://did:plc:viewer/app.bsky.graph.follow/f1', 'did:plc:viewer', 'f1', 'cid-f1', 'did:plc:alice', '{}', $1)
	`, now); err != nil {
		t.Fatalf("seed follows: %v", err)
	}

	rows, err := api.NewFacetStore(pool).SearchMentionSuggestions(ctx, syntax.DID("did:plc:viewer"), "ali", 10, now)
	if err != nil {
		t.Fatalf("SearchMentionSuggestions: %v", err)
	}
	got := make([]string, 0, len(rows))
	for _, row := range rows {
		got = append(got, row.Handle)
		if !row.IsCraftskyProfile {
			t.Fatalf("row %#v is not Craftsky-profile-only", row)
		}
	}
	want := []string{"alice.craftsky.social", "alina.craftsky.social"}
	if len(got) != len(want) {
		t.Fatalf("handles = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("handles = %v, want %v", got, want)
		}
	}
}

func TestFacetStoreSearchMentionSuggestionsTreatsWildcardQueryLiterally(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, facetStoreDDL)
	ctx := context.Background()
	now := time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)

	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_profiles (did, crafts, record_cid) VALUES
			('did:plc:percent', '{}', 'cid-percent'),
			('did:plc:plain', '{}', 'cid-plain')
	`); err != nil {
		t.Fatalf("seed craftsky profiles: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO bluesky_profiles (did, display_name, record_cid) VALUES
			('did:plc:percent', '100% Wool', 'cid-bsky-percent'),
			('did:plc:plain', 'Plain Wool', 'cid-bsky-plain')
	`); err != nil {
		t.Fatalf("seed bluesky profiles: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO atproto_identity_cache (did, handle, handle_lower, resolved_at) VALUES
			('did:plc:percent', 'percent.craftsky.social', 'percent.craftsky.social', $1),
			('did:plc:plain', 'plain.craftsky.social', 'plain.craftsky.social', $1)
	`, now); err != nil {
		t.Fatalf("seed identity cache: %v", err)
	}

	rows, err := api.NewFacetStore(pool).SearchMentionSuggestions(ctx, syntax.DID("did:plc:viewer"), "%", 10, now)
	if err != nil {
		t.Fatalf("SearchMentionSuggestions wildcard: %v", err)
	}
	if len(rows) != 1 || rows[0].Handle != "percent.craftsky.social" {
		t.Fatalf("rows = %#v, want only literal percent display name", rows)
	}
}

func TestFacetStoreResolveMentionRefreshesCacheAndFiltersCraftskyProfiles(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, facetStoreDDL)
	ctx := context.Background()
	now := time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)

	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_profiles (did, crafts, record_cid) VALUES
			('did:plc:alice', '{}', 'cid-alice')
	`); err != nil {
		t.Fatalf("seed craftsky profile: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO atproto_identity_cache (did, handle, handle_lower, resolved_at) VALUES
			('did:plc:alice', 'alice.craftsky.social', 'alice.craftsky.social', $1)
	`, now.Add(-25*time.Hour)); err != nil {
		t.Fatalf("seed stale cache: %v", err)
	}
	resolver := exactResolveFakeResolver{
		didByHandle: map[string]syntax.DID{
			"alice.craftsky.social":   syntax.DID("did:plc:alice"),
			"mallory.example":         syntax.DID("did:plc:mallory"),
			"transient.craftsky.test": syntax.DID("did:plc:transient"),
		},
		handleByDID: map[string]syntax.Handle{
			"did:plc:alice":   syntax.Handle("alice.craftsky.social"),
			"did:plc:mallory": syntax.Handle("mallory.example"),
		},
	}
	store := api.NewFacetStore(pool, resolver)

	row, err := store.ResolveMention(ctx, syntax.Handle("alice.craftsky.social"), now)
	if err != nil {
		t.Fatalf("ResolveMention Alice: %v", err)
	}
	if row.DID.String() != "did:plc:alice" || row.Handle.String() != "alice.craftsky.social" {
		t.Fatalf("row = %+v", row)
	}
	var resolvedAt time.Time
	if err := pool.QueryRow(ctx, `SELECT resolved_at FROM atproto_identity_cache WHERE did = 'did:plc:alice'`).Scan(&resolvedAt); err != nil {
		t.Fatalf("read refreshed cache: %v", err)
	}
	if !resolvedAt.Equal(now) {
		t.Fatalf("resolved_at = %s, want %s", resolvedAt, now)
	}

	_, err = store.ResolveMention(ctx, syntax.Handle("mallory.example"), now)
	if !errors.Is(err, api.ErrMentionNotFound) {
		t.Fatalf("Mallory err = %v, want ErrMentionNotFound", err)
	}
	var malloryRows int
	if err := pool.QueryRow(ctx, `SELECT count(*)::int FROM atproto_identity_cache WHERE did = 'did:plc:mallory'`).Scan(&malloryRows); err != nil {
		t.Fatalf("count Mallory cache rows: %v", err)
	}
	if malloryRows != 0 {
		t.Fatalf("Mallory cache rows = %d, want 0", malloryRows)
	}
}

func TestFacetStoreResolveMentionRefreshesReassignedStaleHandle(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, facetStoreDDL)
	ctx := context.Background()
	now := time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)

	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_profiles (did, crafts, record_cid) VALUES
			('did:plc:oldalice', '{}', 'cid-old-alice'),
			('did:plc:newalice', '{}', 'cid-new-alice')
	`); err != nil {
		t.Fatalf("seed craftsky profiles: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO atproto_identity_cache (did, handle, handle_lower, resolved_at) VALUES
			('did:plc:oldalice', 'alice.craftsky.social', 'alice.craftsky.social', $1)
	`, now.Add(-25*time.Hour)); err != nil {
		t.Fatalf("seed stale reassigned cache: %v", err)
	}
	resolver := exactResolveFakeResolver{
		didByHandle: map[string]syntax.DID{
			"alice.craftsky.social": syntax.DID("did:plc:newalice"),
		},
		handleByDID: map[string]syntax.Handle{
			"did:plc:newalice": syntax.Handle("alice.craftsky.social"),
		},
	}
	store := api.NewFacetStore(pool, resolver)

	row, err := store.ResolveMention(ctx, syntax.Handle("alice.craftsky.social"), now)
	if err != nil {
		t.Fatalf("ResolveMention reassigned handle: %v", err)
	}
	if row.DID.String() != "did:plc:newalice" || row.Handle.String() != "alice.craftsky.social" {
		t.Fatalf("row = %+v", row)
	}

	var did string
	var count int
	if err := pool.QueryRow(ctx, `
		SELECT min(did), count(*)::int
		FROM atproto_identity_cache
		WHERE handle_lower = 'alice.craftsky.social'
	`).Scan(&did, &count); err != nil {
		t.Fatalf("read reassigned cache row: %v", err)
	}
	if count != 1 || did != "did:plc:newalice" {
		t.Fatalf("handle owner count=%d did=%q, want count=1 did=did:plc:newalice", count, did)
	}
}

func TestFacetStoreSearchHashtagSuggestionsCountsRecentRootPosts(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, facetStoreDDL+`
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
    record           JSONB       NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL,
    indexed_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (did, rkey)
);
`)
	ctx := context.Background()
	now := time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles (did, crafts, record_cid) VALUES ('did:plc:alice', '{}', 'cid')`); err != nil {
		t.Fatalf("seed profile: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_posts (uri, did, rkey, cid, text, reply_root_uri, reply_parent_uri, tags, record, created_at) VALUES
			('at://did:plc:alice/social.craftsky.feed.post/1', 'did:plc:alice', '1', 'cid1', 'root one', NULL, NULL, ARRAY['SockKAL', 'sockkal', 'sockmending'], '{}', $1),
			('at://did:plc:alice/social.craftsky.feed.post/2', 'did:plc:alice', '2', 'cid2', 'root two', NULL, NULL, ARRAY['sockkal'], '{}', $2),
			('at://did:plc:alice/social.craftsky.feed.post/old', 'did:plc:alice', 'old', 'cid3', 'old root', NULL, NULL, ARRAY['sockkal'], '{}', $3),
			('at://did:plc:alice/social.craftsky.feed.post/reply', 'did:plc:alice', 'reply', 'cid4', 'reply', 'root', 'parent', ARRAY['sockkal'], '{}', $2),
			('at://did:plc:alice/social.craftsky.feed.post/empty', 'did:plc:alice', 'empty', 'cid5', 'empty', NULL, NULL, ARRAY[''], '{}', $2)
	`, now.Add(-2*time.Hour), now.Add(-24*time.Hour), now.Add(-29*24*time.Hour)); err != nil {
		t.Fatalf("seed posts: %v", err)
	}

	rows, err := api.NewFacetStore(pool).SearchHashtagSuggestions(ctx, "sock", 10, now)
	if err != nil {
		t.Fatalf("SearchHashtagSuggestions: %v", err)
	}
	want := []api.HashtagSuggestionRow{
		{Tag: "sockkal", PostsLast28Days: 2},
		{Tag: "sockmending", PostsLast28Days: 1},
	}
	if len(rows) != len(want) {
		t.Fatalf("rows = %#v, want %#v", rows, want)
	}
	for i := range want {
		if rows[i] != want[i] {
			t.Fatalf("row %d = %#v, want %#v; all=%#v", i, rows[i], want[i], rows)
		}
	}
}

func TestFacetStoreSearchHashtagSuggestionsTreatsWildcardQueryLiterally(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, facetStoreDDL+`
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
    record           JSONB       NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL,
    indexed_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (did, rkey)
);
`)
	ctx := context.Background()
	now := time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles (did, crafts, record_cid) VALUES ('did:plc:alice', '{}', 'cid')`); err != nil {
		t.Fatalf("seed profile: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_posts (uri, did, rkey, cid, text, tags, record, created_at) VALUES
			('at://did:plc:alice/social.craftsky.feed.post/percent', 'did:plc:alice', 'percent', 'cid1', 'literal percent', ARRAY['wool%blend'], '{}', $1),
			('at://did:plc:alice/social.craftsky.feed.post/plain', 'did:plc:alice', 'plain', 'cid2', 'plain', ARRAY['woolblend'], '{}', $1)
	`, now); err != nil {
		t.Fatalf("seed posts: %v", err)
	}

	rows, err := api.NewFacetStore(pool).SearchHashtagSuggestions(ctx, "%", 10, now)
	if err != nil {
		t.Fatalf("SearchHashtagSuggestions wildcard: %v", err)
	}
	if len(rows) != 1 || rows[0].Tag != "wool%blend" {
		t.Fatalf("rows = %#v, want only literal percent tag", rows)
	}
}
