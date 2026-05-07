# Feed Post CRUD Endpoints Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add the four `/v1/posts` and `/v1/profiles/{handleOrDid}/posts` endpoints (create, read, delete, list-by-author) to the AppView so the Flutter client can do basic CRUD on `social.craftsky.feed.post` records.

**Architecture:** Mirror the existing profile-handler pattern in `appview/internal/api/`: typed request struct + `Decode`/`Validate` split, `PostStore` for read queries, synthetic response on create (no waiting for the indexer), pass-through to the user's PDS for writes. Add `CreateRecord` and `DeleteRecord` to the `PDSClient` interface. Hydrate author profile (`{did, handle, displayName, avatarCid}`) into every response.

**Tech Stack:** Go 1.22+, stdlib `net/http` mux, pgx v5, indigo `atproto/syntax` and `atproto/atclient`, generated craftsky lexicon types from `appview/internal/lexicon/craftsky/`.

**Spec:** [`docs/superpowers/specs/2026-05-04-feed-post-crud-endpoints-design.md`](../specs/2026-05-04-feed-post-crud-endpoints-design.md)

---

## File Structure

**New packages and files:**

| Path | Responsibility |
|---|---|
| `appview/internal/postutil/tags.go` | Shared `ExtractTags(facets)` helper used by both the indexer and the create handler so they produce identical `tags` arrays. |
| `appview/internal/postutil/tags_test.go` | Unit tests for tag extraction. |
| `appview/internal/api/post_request.go` | `PostCreateRequest` struct, `DecodePostCreate`, `ValidatePostCreate`. |
| `appview/internal/api/post_request_test.go` | Decode/validate tests. |
| `appview/internal/api/post_response.go` | `PostResponse` shape, `BuildPostResponse(row, handle)`. |
| `appview/internal/api/post_response_test.go` | Response-builder tests. |
| `appview/internal/api/post_store.go` | `PostReader` interface, `*PostStore` Postgres impl, `PostRow`, `ErrPostNotFound`. |
| `appview/internal/api/post_store_test.go` | Real-Postgres store tests via `testdb`. |
| `appview/internal/api/post.go` | The four handlers: `CreatePostHandler`, `GetPostHandler`, `DeletePostHandler`, `ListPostsByAuthorHandler`. |
| `appview/internal/api/post_test.go` | Handler tests with fakes. |

**Modified files:**

| Path | Change |
|---|---|
| `appview/internal/auth/pds_client.go` | Add `CreateRecord` and `DeleteRecord` to the `PDSClient` interface. |
| `appview/internal/auth/pds_client_indigo.go` | Implement the two new methods against indigo. |
| `appview/internal/auth/pds_client_indigo_test.go` | Tests for the two new methods. |
| `appview/internal/auth/anonymous_pds_client.go` | Return `ErrReadOnlyPDSClient` from the two new methods. |
| `appview/internal/auth/anonymous_pds_client_test.go` | Tests for the read-only behaviour. |
| `appview/internal/auth/handlers_test.go` | Extend `noopPDSClient` and `erroringGetPDSClient` mocks with the two new methods. |
| `appview/internal/auth/initialize_profile_test.go` | Extend `mockPDS` mock with the two new methods. |
| `appview/internal/index/craftsky_post.go` | Replace local `extractTags` with `postutil.ExtractTags`. |
| `appview/internal/routes/routes.go` | Register the four new routes. |

---

## Task 1: Extract `extractTags` into a shared `postutil` package

The existing `extractTags` lives in `appview/internal/index/craftsky_post.go`. The create handler needs identical logic to compute `tags` for the synthetic response. Pull the function out of the indexer into a new neutral package both `index` and `api` can import. Behaviour-preserving refactor.

**Files:**
- Create: `appview/internal/postutil/tags.go`
- Create: `appview/internal/postutil/tags_test.go`
- Modify: `appview/internal/index/craftsky_post.go`

- [ ] **Step 1.1: Write the failing test**

Create `appview/internal/postutil/tags_test.go`:

```go
// appview/internal/postutil/tags_test.go
package postutil_test

import (
	"reflect"
	"testing"

	appbsky "github.com/bluesky-social/indigo/api/bsky"

	"social.craftsky/appview/internal/postutil"
)

func TestExtractTags_NilFacets(t *testing.T) {
	got := postutil.ExtractTags(nil)
	if !reflect.DeepEqual(got, []string{}) {
		t.Fatalf("want empty slice, got %#v", got)
	}
}

func TestExtractTags_LowercasesTrimsAndDedupes(t *testing.T) {
	facets := []*appbsky.RichtextFacet{
		{Features: []*appbsky.RichtextFacet_Features_Elem{
			{RichtextFacet_Tag: &appbsky.RichtextFacet_Tag{Tag: "  Knitting  "}},
		}},
		{Features: []*appbsky.RichtextFacet_Features_Elem{
			{RichtextFacet_Tag: &appbsky.RichtextFacet_Tag{Tag: "knitting"}},
			{RichtextFacet_Tag: &appbsky.RichtextFacet_Tag{Tag: "Shawl"}},
		}},
	}
	got := postutil.ExtractTags(facets)
	want := []string{"knitting", "shawl"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("want %#v, got %#v", want, got)
	}
}

func TestExtractTags_IgnoresNonTagFeaturesAndEmpty(t *testing.T) {
	facets := []*appbsky.RichtextFacet{
		{Features: []*appbsky.RichtextFacet_Features_Elem{
			{RichtextFacet_Mention: &appbsky.RichtextFacet_Mention{Did: "did:plc:abc"}},
			{RichtextFacet_Tag: &appbsky.RichtextFacet_Tag{Tag: ""}},
			{RichtextFacet_Tag: &appbsky.RichtextFacet_Tag{Tag: "   "}},
		}},
		nil,
	}
	got := postutil.ExtractTags(facets)
	if !reflect.DeepEqual(got, []string{}) {
		t.Fatalf("want empty slice, got %#v", got)
	}
}
```

- [ ] **Step 1.2: Run test to verify it fails**

```
just test ./appview/internal/postutil/...
```

Expected: build failure (`package postutil does not exist`).

- [ ] **Step 1.3: Implement `postutil.ExtractTags`**

Create `appview/internal/postutil/tags.go`:

```go
// Package postutil holds small lexicon-derived helpers shared between
// the firehose indexer and the API handlers. Keeping them in a neutral
// package avoids an api → index import cycle and ensures both surfaces
// produce identical materialised values.
package postutil

import (
	"strings"

	appbsky "github.com/bluesky-social/indigo/api/bsky"
)

// ExtractTags walks facets and pulls hashtag-feature tags. Lowercase,
// trim, drop empties, dedupe (preserve first-seen order). Always
// returns a non-nil slice — callers store this in a NOT NULL column.
func ExtractTags(facets []*appbsky.RichtextFacet) []string {
	if len(facets) == 0 {
		return []string{}
	}
	out := []string{}
	seen := map[string]struct{}{}
	for _, facet := range facets {
		if facet == nil {
			continue
		}
		for _, feat := range facet.Features {
			if feat == nil || feat.RichtextFacet_Tag == nil {
				continue
			}
			t := strings.ToLower(strings.TrimSpace(feat.RichtextFacet_Tag.Tag))
			if t == "" {
				continue
			}
			if _, dup := seen[t]; dup {
				continue
			}
			seen[t] = struct{}{}
			out = append(out, t)
		}
	}
	return out
}
```

- [ ] **Step 1.4: Run new test, verify pass**

```
just test ./appview/internal/postutil/...
```

Expected: `ok  social.craftsky/appview/internal/postutil`.

- [ ] **Step 1.5: Replace usage in the indexer**

In `appview/internal/index/craftsky_post.go`:

1. Add import: `"social.craftsky/appview/internal/postutil"`.
2. Replace `tags := extractTags(rec.Facets)` with `tags := postutil.ExtractTags(rec.Facets)`.
3. Delete the local `extractTags` function (the entire `func extractTags(...)` block including its leading doc comment).

- [ ] **Step 1.6: Run indexer tests, verify still pass**

```
just test ./appview/internal/index/...
```

Expected: all existing tests pass unchanged.

- [ ] **Step 1.7: Commit**

```bash
git add appview/internal/postutil/ appview/internal/index/craftsky_post.go
git commit -m "refactor(appview): move ExtractTags to shared postutil package"
```

---

## Task 2: Extend `PDSClient` with `CreateRecord` and `DeleteRecord`

Add the two methods to the interface and every existing implementation/mock so the rest of the codebase keeps compiling. Production behaviour landed by `IndigoPDSClient`; `AnonymousPDSClient` returns the read-only sentinel; test mocks default to no-op.

**Files:**
- Modify: `appview/internal/auth/pds_client.go`
- Modify: `appview/internal/auth/pds_client_indigo.go`
- Modify: `appview/internal/auth/anonymous_pds_client.go`
- Modify: `appview/internal/auth/handlers_test.go`
- Modify: `appview/internal/auth/initialize_profile_test.go`
- Test: `appview/internal/auth/pds_client_indigo_test.go`
- Test: `appview/internal/auth/anonymous_pds_client_test.go`

- [ ] **Step 2.1: Write the failing tests for `IndigoPDSClient`**

Append to `appview/internal/auth/pds_client_indigo_test.go`:

```go
func TestIndigoPDSClient_CreateRecord_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/xrpc/com.atproto.repo.createRecord" || r.Method != http.MethodPost {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["repo"] != "did:plc:xyz" || body["collection"] != "social.craftsky.feed.post" {
			t.Fatalf("unexpected body: %+v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"uri":"at://did:plc:xyz/social.craftsky.feed.post/3lf2abc","cid":"bafyabc"}`))
	}))
	defer srv.Close()
	cli := &auth.IndigoPDSClient{Client: atclient.NewAPIClient(srv.URL)}

	uri, cid, err := cli.CreateRecord(
		context.Background(),
		syntax.DID("did:plc:xyz"),
		"social.craftsky.feed.post",
		map[string]any{"$type": "social.craftsky.feed.post", "text": "hi"},
	)
	if err != nil {
		t.Fatalf("CreateRecord: %v", err)
	}
	if string(uri) != "at://did:plc:xyz/social.craftsky.feed.post/3lf2abc" {
		t.Fatalf("uri = %q", uri)
	}
	if string(cid) != "bafyabc" {
		t.Fatalf("cid = %q", cid)
	}
}

func TestIndigoPDSClient_CreateRecord_EmptyURIErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"uri":"","cid":"bafyabc"}`))
	}))
	defer srv.Close()
	cli := &auth.IndigoPDSClient{Client: atclient.NewAPIClient(srv.URL)}
	_, _, err := cli.CreateRecord(context.Background(),
		syntax.DID("did:plc:xyz"), "social.craftsky.feed.post", map[string]any{})
	if err == nil {
		t.Fatal("want error on empty uri, got nil")
	}
}

func TestIndigoPDSClient_DeleteRecord_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/xrpc/com.atproto.repo.deleteRecord" || r.Method != http.MethodPost {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["repo"] != "did:plc:xyz" || body["collection"] != "social.craftsky.feed.post" || body["rkey"] != "3lf2abc" {
			t.Fatalf("unexpected body: %+v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	cli := &auth.IndigoPDSClient{Client: atclient.NewAPIClient(srv.URL)}
	if err := cli.DeleteRecord(context.Background(),
		syntax.DID("did:plc:xyz"), "social.craftsky.feed.post", "3lf2abc"); err != nil {
		t.Fatalf("DeleteRecord: %v", err)
	}
}

func TestIndigoPDSClient_DeleteRecord_NotFound_TranslatesToErrRecordNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"RecordNotFound","message":"no such record"}`))
	}))
	defer srv.Close()
	cli := &auth.IndigoPDSClient{Client: atclient.NewAPIClient(srv.URL)}
	err := cli.DeleteRecord(context.Background(),
		syntax.DID("did:plc:xyz"), "social.craftsky.feed.post", "3lf2abc")
	if !errors.Is(err, auth.ErrRecordNotFound) {
		t.Fatalf("want ErrRecordNotFound, got %v", err)
	}
}
```

If the existing test file does not import any of `encoding/json`, `errors`, `httptest`, `http`, `atclient`, or `syntax`, add them. Open the file and add only the missing imports.

- [ ] **Step 2.2: Write the failing tests for `AnonymousPDSClient`**

Append to `appview/internal/auth/anonymous_pds_client_test.go`:

```go
func TestAnonymousPDSClient_CreateRecord_ReadOnly(t *testing.T) {
	cli := auth.NewAnonymousPDSClient(nil, time.Second)
	_, _, err := cli.CreateRecord(context.Background(),
		syntax.DID("did:plc:xyz"), "social.craftsky.feed.post", map[string]any{})
	if !errors.Is(err, auth.ErrReadOnlyPDSClient) {
		t.Fatalf("want ErrReadOnlyPDSClient, got %v", err)
	}
}

func TestAnonymousPDSClient_DeleteRecord_ReadOnly(t *testing.T) {
	cli := auth.NewAnonymousPDSClient(nil, time.Second)
	err := cli.DeleteRecord(context.Background(),
		syntax.DID("did:plc:xyz"), "social.craftsky.feed.post", "3lf2abc")
	if !errors.Is(err, auth.ErrReadOnlyPDSClient) {
		t.Fatalf("want ErrReadOnlyPDSClient, got %v", err)
	}
}
```

- [ ] **Step 2.3: Run tests to verify they fail**

```
just test ./appview/internal/auth/...
```

Expected: build failure — `cli.CreateRecord` and `cli.DeleteRecord` undefined.

- [ ] **Step 2.4: Add the methods to the interface**

In `appview/internal/auth/pds_client.go`, replace the `PDSClient` interface with:

```go
type PDSClient interface {
	GetRecord(ctx context.Context, repo syntax.DID, collection string, rkey string, out any) (cid string, err error)
	PutRecord(ctx context.Context, repo syntax.DID, collection string, rkey string, record any) error
	CreateRecord(ctx context.Context, repo syntax.DID, collection string, record any) (uri syntax.ATURI, cid syntax.CID, err error)
	DeleteRecord(ctx context.Context, repo syntax.DID, collection string, rkey string) error
}
```

- [ ] **Step 2.5: Implement on `IndigoPDSClient`**

Append to `appview/internal/auth/pds_client_indigo.go`:

```go
// CreateRecord calls com.atproto.repo.createRecord on the user's PDS.
// Returns the AT-URI and CID assigned by the PDS. The PDS stamps the
// rkey on TID-keyed collections.
func (i *IndigoPDSClient) CreateRecord(
	ctx context.Context,
	repo syntax.DID,
	collection string,
	record any,
) (syntax.ATURI, syntax.CID, error) {
	nsid, err := syntax.ParseNSID("com.atproto.repo.createRecord")
	if err != nil {
		return "", "", fmt.Errorf("parse nsid: %w", err)
	}
	body := map[string]any{
		"repo":       repo.String(),
		"collection": collection,
		"record":     record,
	}
	var resp struct {
		URI string `json:"uri"`
		CID string `json:"cid"`
	}
	if err := i.Client.Post(ctx, nsid, body, &resp); err != nil {
		return "", "", err
	}
	if resp.URI == "" || resp.CID == "" {
		return "", "", fmt.Errorf("createRecord: PDS returned empty uri or cid")
	}
	return syntax.ATURI(resp.URI), syntax.CID(resp.CID), nil
}

// DeleteRecord calls com.atproto.repo.deleteRecord on the user's PDS.
// "Record not found" responses are translated to ErrRecordNotFound so
// callers can treat delete-of-already-deleted as idempotent success.
func (i *IndigoPDSClient) DeleteRecord(
	ctx context.Context,
	repo syntax.DID,
	collection string,
	rkey string,
) error {
	nsid, err := syntax.ParseNSID("com.atproto.repo.deleteRecord")
	if err != nil {
		return fmt.Errorf("parse nsid: %w", err)
	}
	body := map[string]any{
		"repo":       repo.String(),
		"collection": collection,
		"rkey":       rkey,
	}
	var resp any
	if err := i.Client.Post(ctx, nsid, body, &resp); err != nil {
		return translateGetRecordError(err)
	}
	return nil
}
```

`translateGetRecordError` already maps both `RecordNotFound` (XRPC error name) and HTTP 404; reusing it for delete is correct because the PDS uses the same error shape.

- [ ] **Step 2.6: Implement on `AnonymousPDSClient`**

Append to `appview/internal/auth/anonymous_pds_client.go`:

```go
// CreateRecord is not supported by the anonymous client.
func (c *AnonymousPDSClient) CreateRecord(_ context.Context, _ syntax.DID, _ string, _ any) (syntax.ATURI, syntax.CID, error) {
	return "", "", ErrReadOnlyPDSClient
}

// DeleteRecord is not supported by the anonymous client.
func (c *AnonymousPDSClient) DeleteRecord(_ context.Context, _ syntax.DID, _, _ string) error {
	return ErrReadOnlyPDSClient
}
```

- [ ] **Step 2.7: Extend test mocks**

In `appview/internal/auth/handlers_test.go`, find `noopPDSClient` and `erroringGetPDSClient` and add the two new methods to each. Append immediately after the existing methods:

```go
// On noopPDSClient:
func (noopPDSClient) CreateRecord(_ context.Context, _ syntax.DID, _ string, _ any) (syntax.ATURI, syntax.CID, error) {
	return "", "", nil
}
func (noopPDSClient) DeleteRecord(_ context.Context, _ syntax.DID, _, _ string) error {
	return nil
}

// On erroringGetPDSClient:
func (erroringGetPDSClient) CreateRecord(_ context.Context, _ syntax.DID, _ string, _ any) (syntax.ATURI, syntax.CID, error) {
	return "", "", nil
}
func (erroringGetPDSClient) DeleteRecord(_ context.Context, _ syntax.DID, _, _ string) error {
	return nil
}
```

In `appview/internal/auth/initialize_profile_test.go`, find `mockPDS` and add:

```go
func (m *mockPDS) CreateRecord(_ context.Context, _ syntax.DID, _ string, _ any) (syntax.ATURI, syntax.CID, error) {
	return "", "", nil
}
func (m *mockPDS) DeleteRecord(_ context.Context, _ syntax.DID, _, _ string) error {
	return nil
}
```

- [ ] **Step 2.8: Run all auth tests**

```
just test ./appview/internal/auth/...
```

Expected: all tests pass, including the four new tests added in 2.1 and 2.2.

- [ ] **Step 2.9: Commit**

```bash
git add appview/internal/auth/
git commit -m "feat(appview): add CreateRecord and DeleteRecord to PDSClient"
```

---

## Task 3: Build `PostStore` (read side)

The store wraps two queries: `ReadOne(did, rkey)` and `ListByAuthor(did, limit, cursor)`. Tests run against a real Postgres via `testdb.WithSchema`.

**Files:**
- Create: `appview/internal/api/post_store.go`
- Create: `appview/internal/api/post_store_test.go`

- [ ] **Step 3.1: Write the failing tests**

Create `appview/internal/api/post_store_test.go`:

```go
// appview/internal/api/post_store_test.go
package api_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/testdb"
)

const postStoreDDL = `
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
`

func seedMember(t *testing.T, pool *pgxpool.Pool, did string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO craftsky_profiles (did, record_cid) VALUES ($1, 'seed')`, did); err != nil {
		t.Fatalf("seed member: %v", err)
	}
}

func seedBskyProfile(t *testing.T, pool *pgxpool.Pool, did, displayName, avatarCID string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO bluesky_profiles (did, display_name, avatar_cid, record_cid)
		 VALUES ($1, $2, $3, 'seed')`, did, displayName, avatarCID); err != nil {
		t.Fatalf("seed bsky profile: %v", err)
	}
}

func seedPost(t *testing.T, pool *pgxpool.Pool, did, rkey, text string, indexedAt time.Time) string {
	t.Helper()
	uri := "at://" + did + "/social.craftsky.feed.post/" + rkey
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO craftsky_posts (uri, did, rkey, cid, text, record, created_at, indexed_at)
		VALUES ($1, $2, $3, 'bafycid', $4, '{}'::jsonb, $5, $5)`,
		uri, did, rkey, text, indexedAt); err != nil {
		t.Fatalf("seed post: %v", err)
	}
	return uri
}

func TestPostStore_ReadOne_HappyPath(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, postStoreDDL)
	seedMember(t, pool, "did:plc:alice")
	seedBskyProfile(t, pool, "did:plc:alice", "Alice", "bafyavatar")
	seedPost(t, pool, "did:plc:alice", "rk1", "hello", time.Now())

	store := api.NewPostStore(pool)
	row, err := store.ReadOne(context.Background(), "did:plc:alice", "rk1")
	if err != nil {
		t.Fatalf("ReadOne: %v", err)
	}
	if row.Text != "hello" || row.DID != "did:plc:alice" || row.Rkey != "rk1" {
		t.Errorf("row mismatch: %+v", row)
	}
	if row.AuthorDisplayName == nil || *row.AuthorDisplayName != "Alice" {
		t.Errorf("displayName = %v", row.AuthorDisplayName)
	}
	if row.AuthorAvatarCID == nil || *row.AuthorAvatarCID != "bafyavatar" {
		t.Errorf("avatarCID = %v", row.AuthorAvatarCID)
	}
}

func TestPostStore_ReadOne_NoBlueskyMirror(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, postStoreDDL)
	seedMember(t, pool, "did:plc:alice")
	seedPost(t, pool, "did:plc:alice", "rk1", "hello", time.Now())

	store := api.NewPostStore(pool)
	row, err := store.ReadOne(context.Background(), "did:plc:alice", "rk1")
	if err != nil {
		t.Fatalf("ReadOne: %v", err)
	}
	if row.AuthorDisplayName != nil {
		t.Errorf("expected nil displayName, got %v", *row.AuthorDisplayName)
	}
	if row.AuthorAvatarCID != nil {
		t.Errorf("expected nil avatarCID, got %v", *row.AuthorAvatarCID)
	}
}

func TestPostStore_ReadOne_NotFound(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, postStoreDDL)
	seedMember(t, pool, "did:plc:alice")

	store := api.NewPostStore(pool)
	_, err := store.ReadOne(context.Background(), "did:plc:alice", "missing")
	if !errors.Is(err, api.ErrPostNotFound) {
		t.Fatalf("want ErrPostNotFound, got %v", err)
	}
}

func TestPostStore_ListByAuthor_OrdersByIndexedAtDesc(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, postStoreDDL)
	seedMember(t, pool, "did:plc:alice")
	t1 := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	seedPost(t, pool, "did:plc:alice", "rk1", "first", t1)
	seedPost(t, pool, "did:plc:alice", "rk2", "second", t2)
	seedPost(t, pool, "did:plc:alice", "rk3", "third", t3)

	store := api.NewPostStore(pool)
	rows, cursor, err := store.ListByAuthor(context.Background(), "did:plc:alice", 50, "")
	if err != nil {
		t.Fatalf("ListByAuthor: %v", err)
	}
	if cursor != "" {
		t.Errorf("want empty cursor on final page, got %q", cursor)
	}
	if len(rows) != 3 {
		t.Fatalf("want 3 rows, got %d", len(rows))
	}
	if rows[0].Rkey != "rk3" || rows[1].Rkey != "rk2" || rows[2].Rkey != "rk1" {
		t.Errorf("ordering wrong: %s,%s,%s", rows[0].Rkey, rows[1].Rkey, rows[2].Rkey)
	}
}

func TestPostStore_ListByAuthor_RespectsLimitAndPaginates(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, postStoreDDL)
	seedMember(t, pool, "did:plc:alice")
	for i := 0; i < 5; i++ {
		seedPost(t, pool, "did:plc:alice",
			"rk"+string(rune('0'+i)),
			"p", time.Date(2026, 5, 1+i, 12, 0, 0, 0, time.UTC))
	}

	store := api.NewPostStore(pool)
	page1, cursor, err := store.ListByAuthor(context.Background(), "did:plc:alice", 2, "")
	if err != nil || len(page1) != 2 {
		t.Fatalf("page1 err=%v len=%d", err, len(page1))
	}
	if cursor == "" {
		t.Fatal("want non-empty cursor on partial page")
	}
	page2, cursor2, err := store.ListByAuthor(context.Background(), "did:plc:alice", 2, cursor)
	if err != nil || len(page2) != 2 {
		t.Fatalf("page2 err=%v len=%d", err, len(page2))
	}
	if cursor2 == "" {
		t.Fatal("want non-empty cursor after page2")
	}
	page3, cursor3, err := store.ListByAuthor(context.Background(), "did:plc:alice", 2, cursor2)
	if err != nil || len(page3) != 1 {
		t.Fatalf("page3 err=%v len=%d", err, len(page3))
	}
	if cursor3 != "" {
		t.Errorf("want empty cursor on final page, got %q", cursor3)
	}
}

func TestPostStore_ReadAuthor_HydratesFromBlueskyProfile(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, postStoreDDL)
	seedMember(t, pool, "did:plc:alice")
	seedBskyProfile(t, pool, "did:plc:alice", "Alice", "bafyAvatar")

	store := api.NewPostStore(pool)
	got, err := store.ReadAuthor(context.Background(), "did:plc:alice")
	if err != nil {
		t.Fatalf("ReadAuthor: %v", err)
	}
	if got.DisplayName == nil || *got.DisplayName != "Alice" {
		t.Errorf("displayName = %v", got.DisplayName)
	}
	if got.AvatarCID == nil || *got.AvatarCID != "bafyAvatar" {
		t.Errorf("avatarCID = %v", got.AvatarCID)
	}
}

func TestPostStore_ReadAuthor_NoBlueskyMirror_ReturnsNils(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, postStoreDDL)
	seedMember(t, pool, "did:plc:alice")

	store := api.NewPostStore(pool)
	got, err := store.ReadAuthor(context.Background(), "did:plc:alice")
	if err != nil {
		t.Fatalf("ReadAuthor: %v", err)
	}
	if got.DisplayName != nil || got.AvatarCID != nil {
		t.Errorf("expected nils, got %+v", got)
	}
}

func TestPostStore_ListByAuthor_InvalidCursor(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, postStoreDDL)
	store := api.NewPostStore(pool)
	_, _, err := store.ListByAuthor(context.Background(), "did:plc:alice", 50, "!!!not-base64!!!")
	if !errors.Is(err, envelope.ErrInvalidCursor) {
		t.Fatalf("want ErrInvalidCursor, got %v", err)
	}
}
```

- [ ] **Step 3.2: Run tests to verify they fail**

```
just test ./appview/internal/api/... -run TestPostStore
```

Expected: build failure (`api.NewPostStore`, `api.PostStore`, `api.ErrPostNotFound` undefined).

- [ ] **Step 3.3: Implement `post_store.go`**

Create `appview/internal/api/post_store.go`:

```go
// appview/internal/api/post_store.go
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/api/envelope"
)

// ErrPostNotFound is returned by PostStore.ReadOne when no row matches.
var ErrPostNotFound = errors.New("post: not found")

// PostRow is the joined view of craftsky_posts plus author display fields
// from bluesky_profiles. Reply/quote pointers are kept as separate
// pointers so handlers can decide nesting at the JSON layer.
type PostRow struct {
	URI            string
	DID            string
	Rkey           string
	CID            string
	Text           string
	Facets         json.RawMessage
	ReplyRootURI   *string
	ReplyRootCID   *string
	ReplyParentURI *string
	ReplyParentCID *string
	QuoteURI       *string
	QuoteCID       *string
	Tags           []string
	CreatedAt      time.Time
	IndexedAt      time.Time

	AuthorDisplayName *string
	AuthorAvatarCID   *string
}

// PostAuthorRow is the slim author-hydration view used when we need to
// build a synthetic response for a freshly-created post (the post row
// itself doesn't exist yet at that moment, but the author's bsky
// profile may).
type PostAuthorRow struct {
	DisplayName *string
	AvatarCID   *string
}

// PostReader is the read-side interface handlers depend on. Tests inject
// fakes; production uses *PostStore.
type PostReader interface {
	ReadOne(ctx context.Context, did, rkey string) (*PostRow, error)
	ListByAuthor(ctx context.Context, did string, limit int, cursor string) (rows []*PostRow, nextCursor string, err error)
	ReadAuthor(ctx context.Context, did string) (*PostAuthorRow, error)
}

// PostStore is the Postgres-backed implementation.
type PostStore struct {
	pool *pgxpool.Pool
}

func NewPostStore(pool *pgxpool.Pool) *PostStore {
	return &PostStore{pool: pool}
}

const postSelectColumns = `
	p.uri, p.did, p.rkey, p.cid, p.text, p.facets,
	p.reply_root_uri, p.reply_root_cid, p.reply_parent_uri, p.reply_parent_cid,
	p.quote_uri, p.quote_cid, p.tags, p.created_at, p.indexed_at,
	bp.display_name, bp.avatar_cid
`

func scanPostRow(scanner pgx.Row) (*PostRow, error) {
	out := &PostRow{}
	err := scanner.Scan(
		&out.URI, &out.DID, &out.Rkey, &out.CID, &out.Text, &out.Facets,
		&out.ReplyRootURI, &out.ReplyRootCID, &out.ReplyParentURI, &out.ReplyParentCID,
		&out.QuoteURI, &out.QuoteCID, &out.Tags, &out.CreatedAt, &out.IndexedAt,
		&out.AuthorDisplayName, &out.AuthorAvatarCID,
	)
	return out, err
}

// ReadOne returns the post identified by (did, rkey). Returns
// ErrPostNotFound when no row exists.
func (s *PostStore) ReadOne(ctx context.Context, did, rkey string) (*PostRow, error) {
	q := `
		SELECT ` + postSelectColumns + `
		FROM craftsky_posts p
		LEFT JOIN bluesky_profiles bp ON bp.did = p.did
		WHERE p.did = $1 AND p.rkey = $2
	`
	row, err := scanPostRow(s.pool.QueryRow(ctx, q, did, rkey))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPostNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("post read %s/%s: %w", did, rkey, err)
	}
	return row, nil
}

// ListByAuthor returns up to limit posts authored by did, ordered by
// (indexed_at DESC, uri DESC), starting after the cursor if non-empty.
// Returns the encoded next-page cursor when the result is full; empty
// string when this is the final page.
func (s *PostStore) ListByAuthor(ctx context.Context, did string, limit int, cursor string) ([]*PostRow, string, error) {
	cur, err := envelope.DecodeCursor(cursor)
	if err != nil {
		return nil, "", err
	}
	var (
		curIndexedAt any
		curURI       any
	)
	if v, ok := cur["indexedAt"].(string); ok && v != "" {
		t, perr := time.Parse(time.RFC3339Nano, v)
		if perr != nil {
			return nil, "", envelope.ErrInvalidCursor
		}
		curIndexedAt = t
		uri, _ := cur["uri"].(string)
		curURI = uri
	}

	q := `
		SELECT ` + postSelectColumns + `
		FROM craftsky_posts p
		LEFT JOIN bluesky_profiles bp ON bp.did = p.did
		WHERE p.did = $1
		  AND ($2::timestamptz IS NULL
		       OR (p.indexed_at, p.uri) < ($2::timestamptz, $3::text))
		ORDER BY p.indexed_at DESC, p.uri DESC
		LIMIT $4
	`
	rows, err := s.pool.Query(ctx, q, did, curIndexedAt, curURI, limit)
	if err != nil {
		return nil, "", fmt.Errorf("post list %s: %w", did, err)
	}
	defer rows.Close()

	out := make([]*PostRow, 0, limit)
	for rows.Next() {
		row, scanErr := scanPostRow(rows)
		if scanErr != nil {
			return nil, "", fmt.Errorf("post list scan: %w", scanErr)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("post list iter: %w", err)
	}

	if len(out) < limit {
		return out, "", nil
	}
	last := out[len(out)-1]
	next, err := envelope.EncodeCursor(map[string]any{
		"indexedAt": last.IndexedAt.UTC().Format(time.RFC3339Nano),
		"uri":       last.URI,
	})
	if err != nil {
		return nil, "", fmt.Errorf("encode cursor: %w", err)
	}
	return out, next, nil
}

// ReadAuthor returns the bluesky_profiles display fields for did.
// Returns (&PostAuthorRow{nil, nil}, nil) — not an error — when the
// user has no bluesky_profiles row yet. The post-create path uses this
// to hydrate authors whose row hasn't been indexed yet.
func (s *PostStore) ReadAuthor(ctx context.Context, did string) (*PostAuthorRow, error) {
	const q = `
		SELECT display_name, avatar_cid
		FROM bluesky_profiles
		WHERE did = $1
	`
	out := &PostAuthorRow{}
	err := s.pool.QueryRow(ctx, q, did).Scan(&out.DisplayName, &out.AvatarCID)
	if errors.Is(err, pgx.ErrNoRows) {
		return &PostAuthorRow{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("post read author %s: %w", did, err)
	}
	return out, nil
}
```

- [ ] **Step 3.4: Run tests to verify they pass**

```
just test ./appview/internal/api/... -run TestPostStore
```

Expected: all six TestPostStore_* pass.

- [ ] **Step 3.5: Commit**

```bash
git add appview/internal/api/post_store.go appview/internal/api/post_store_test.go
git commit -m "feat(appview): add PostStore for craftsky_posts read side"
```

---

## Task 4: Build `PostCreateRequest` decode + validate

Mirror `profile_request.go`'s pattern: `Decode*` returns `*FieldError` on parse problems; `Validate*` returns `*FieldError` with code `validation_failed` on lexicon-rule failures.

**Files:**
- Create: `appview/internal/api/post_request.go`
- Create: `appview/internal/api/post_request_test.go`

- [ ] **Step 4.1: Write the failing tests**

Create `appview/internal/api/post_request_test.go`:

```go
// appview/internal/api/post_request_test.go
package api_test

import (
	"errors"
	"strings"
	"testing"

	"social.craftsky/appview/internal/api"
)

func TestDecodePostCreate_HappyPathTextOnly(t *testing.T) {
	t.Parallel()
	body := strings.NewReader(`{"text":"hello"}`)
	req, err := api.DecodePostCreate(body)
	if err != nil {
		t.Fatalf("DecodePostCreate: %v", err)
	}
	if req.Text != "hello" {
		t.Errorf("text = %q", req.Text)
	}
}

func TestDecodePostCreate_RejectsImagesField(t *testing.T) {
	t.Parallel()
	body := strings.NewReader(`{"text":"hi","images":[]}`)
	_, err := api.DecodePostCreate(body)
	var fe *api.FieldError
	if !errors.As(err, &fe) {
		t.Fatalf("want *FieldError, got %v", err)
	}
	if fe.Code != "unexpected_field" {
		t.Errorf("code = %q", fe.Code)
	}
	if _, ok := fe.Fields["images"]; !ok {
		t.Errorf("expected images in fields, got %v", fe.Fields)
	}
}

func TestDecodePostCreate_RejectsProjectField(t *testing.T) {
	t.Parallel()
	body := strings.NewReader(`{"text":"hi","project":{}}`)
	_, err := api.DecodePostCreate(body)
	var fe *api.FieldError
	if !errors.As(err, &fe) || fe.Code != "unexpected_field" {
		t.Fatalf("want unexpected_field, got %v", err)
	}
	if _, ok := fe.Fields["project"]; !ok {
		t.Errorf("expected project in fields")
	}
}

func TestDecodePostCreate_RejectsCreatedAtField(t *testing.T) {
	t.Parallel()
	body := strings.NewReader(`{"text":"hi","createdAt":"2026-05-04T12:00:00Z"}`)
	_, err := api.DecodePostCreate(body)
	var fe *api.FieldError
	if !errors.As(err, &fe) || fe.Code != "unexpected_field" {
		t.Fatalf("want unexpected_field, got %v", err)
	}
}

func TestDecodePostCreate_MalformedJSON(t *testing.T) {
	t.Parallel()
	body := strings.NewReader(`{not json`)
	_, err := api.DecodePostCreate(body)
	var fe *api.FieldError
	if !errors.As(err, &fe) || fe.Code != "malformed_body" {
		t.Fatalf("want malformed_body, got %v", err)
	}
}

func TestValidatePostCreate_RejectsEmptyText(t *testing.T) {
	t.Parallel()
	err := api.ValidatePostCreate(api.PostCreateRequest{Text: ""})
	var fe *api.FieldError
	if !errors.As(err, &fe) || fe.Code != "validation_failed" {
		t.Fatalf("want validation_failed, got %v", err)
	}
	if _, ok := fe.Fields["text"]; !ok {
		t.Errorf("expected text in fields")
	}
}

func TestValidatePostCreate_RejectsTextOver2000Graphemes(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("a", 2001)
	err := api.ValidatePostCreate(api.PostCreateRequest{Text: long})
	var fe *api.FieldError
	if !errors.As(err, &fe) || fe.Code != "validation_failed" {
		t.Fatalf("want validation_failed, got %v", err)
	}
}

func TestValidatePostCreate_AcceptsValidReply(t *testing.T) {
	t.Parallel()
	err := api.ValidatePostCreate(api.PostCreateRequest{
		Text: "hi",
		Reply: &api.ReplyRef{
			Root:   api.StrongRef{URI: "at://did:plc:abc/social.craftsky.feed.post/rk1", CID: "bafy1"},
			Parent: api.StrongRef{URI: "at://did:plc:abc/social.craftsky.feed.post/rk2", CID: "bafy2"},
		},
	})
	if err != nil {
		t.Fatalf("want nil, got %v", err)
	}
}

func TestValidatePostCreate_RejectsReplyWithBadURI(t *testing.T) {
	t.Parallel()
	err := api.ValidatePostCreate(api.PostCreateRequest{
		Text: "hi",
		Reply: &api.ReplyRef{
			Root:   api.StrongRef{URI: "not-a-uri", CID: "bafy1"},
			Parent: api.StrongRef{URI: "at://did:plc:abc/social.craftsky.feed.post/rk2", CID: "bafy2"},
		},
	})
	var fe *api.FieldError
	if !errors.As(err, &fe) {
		t.Fatalf("want *FieldError, got %v", err)
	}
	if _, ok := fe.Fields["reply.root.uri"]; !ok {
		t.Errorf("expected reply.root.uri in fields, got %v", fe.Fields)
	}
}
```

- [ ] **Step 4.2: Run tests to verify they fail**

```
just test ./appview/internal/api/... -run TestDecodePostCreate
just test ./appview/internal/api/... -run TestValidatePostCreate
```

Expected: build failure (types undefined).

- [ ] **Step 4.3: Implement `post_request.go`**

Create `appview/internal/api/post_request.go`:

```go
// appview/internal/api/post_request.go
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"unicode/utf8"

	appbsky "github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

// StrongRef is the wire shape of a strongRef ({uri, cid}). Used for
// reply pointers and quote embeds. cid uses a string rather than
// syntax.CID so unmarshalling never fails on the "informal helper"
// type — the validator runs as a separate step.
type StrongRef struct {
	URI string `json:"uri"`
	CID string `json:"cid"`
}

// ReplyRef mirrors the lexicon's #replyRef.
type ReplyRef struct {
	Root   StrongRef `json:"root"`
	Parent StrongRef `json:"parent"`
}

// EmbedRequest mirrors what the wire accepts on a create request. Today
// only quote embeds are supported. The wire shape uses {embed: {quote:
// {uri, cid}}}; the AppView translates it to the lexicon's
// {embed: {$type: ..#quoteEmbed, record: {uri, cid}}} before writing.
type EmbedRequest struct {
	Quote *StrongRef `json:"quote,omitempty"`
}

// PostCreateRequest is the decoded body of POST /v1/posts.
// createdAt is server-stamped; project, images are not writable in this
// pass and are explicitly rejected.
type PostCreateRequest struct {
	Text   string                    `json:"text"`
	Facets []*appbsky.RichtextFacet  `json:"facets,omitempty"`
	Reply  *ReplyRef                 `json:"reply,omitempty"`
	Embed  *EmbedRequest             `json:"embed,omitempty"`
}

// rejectedPostFields enumerates wire fields that are NOT writable here.
var rejectedPostFields = []string{"images", "project", "createdAt"}

// DecodePostCreate reads a JSON body into PostCreateRequest. Rejects
// any of rejectedPostFields and any unknown keys with code
// "unexpected_field"; malformed JSON with "malformed_body".
func DecodePostCreate(body io.Reader) (PostCreateRequest, error) {
	raw, err := io.ReadAll(body)
	if err != nil {
		return PostCreateRequest{}, &FieldError{
			Code:   "malformed_body",
			Fields: map[string]string{"_": err.Error()},
		}
	}
	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(raw, &rawMap); err != nil {
		return PostCreateRequest{}, &FieldError{
			Code:   "malformed_body",
			Fields: map[string]string{"_": err.Error()},
		}
	}
	rejected := map[string]string{}
	for _, k := range rejectedPostFields {
		if _, present := rawMap[k]; present {
			rejected[k] = "not writable in v1"
		}
	}
	if len(rejected) > 0 {
		return PostCreateRequest{}, &FieldError{
			Code:   "unexpected_field",
			Fields: rejected,
		}
	}
	out := PostCreateRequest{}
	strict := json.NewDecoder(bytes.NewReader(raw))
	strict.DisallowUnknownFields()
	if err := strict.Decode(&out); err != nil {
		return PostCreateRequest{}, &FieldError{
			Code:   "unexpected_field",
			Fields: map[string]string{"_": err.Error()},
		}
	}
	return out, nil
}

// ValidatePostCreate enforces lexicon rules: non-empty text, ≤ 2000
// graphemes (approximated by rune count, matching profile_request),
// and AT-URI parseability on reply/quote pointers.
func ValidatePostCreate(req PostCreateRequest) error {
	fields := map[string]string{}
	if req.Text == "" {
		fields["text"] = "must not be empty"
	} else if utf8.RuneCountInString(req.Text) > 2000 {
		fields["text"] = "exceeds 2000 graphemes"
	}
	if req.Reply != nil {
		validateStrongRef(fields, "reply.root", req.Reply.Root)
		validateStrongRef(fields, "reply.parent", req.Reply.Parent)
	}
	if req.Embed != nil && req.Embed.Quote != nil {
		validateStrongRef(fields, "embed.quote", *req.Embed.Quote)
	}
	if len(fields) > 0 {
		return &FieldError{Code: "validation_failed", Fields: fields}
	}
	return nil
}

func validateStrongRef(fields map[string]string, prefix string, ref StrongRef) {
	if _, err := syntax.ParseATURI(ref.URI); err != nil {
		fields[prefix+".uri"] = fmt.Sprintf("not a valid AT-URI: %s", err)
	}
	if ref.CID == "" {
		fields[prefix+".cid"] = "must not be empty"
	}
}
```

- [ ] **Step 4.4: Run tests to verify they pass**

```
just test ./appview/internal/api/... -run TestDecodePostCreate
just test ./appview/internal/api/... -run TestValidatePostCreate
```

Expected: all nine tests pass.

- [ ] **Step 4.5: Commit**

```bash
git add appview/internal/api/post_request.go appview/internal/api/post_request_test.go
git commit -m "feat(appview): add PostCreateRequest decode and validate"
```

---

## Task 5: Build `PostResponse` and `BuildPostResponse`

Pure transformation: `*PostRow` + `syntax.Handle` → `*PostResponse`. No business logic; just shape conversion.

**Files:**
- Create: `appview/internal/api/post_response.go`
- Create: `appview/internal/api/post_response_test.go`

- [ ] **Step 5.1: Write the failing tests**

Create `appview/internal/api/post_response_test.go`:

```go
// appview/internal/api/post_response_test.go
package api_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
)

func ptrStr(s string) *string { return &s }

func baseRow() *api.PostRow {
	return &api.PostRow{
		URI:       "at://did:plc:alice/social.craftsky.feed.post/rk1",
		DID:       "did:plc:alice",
		Rkey:      "rk1",
		CID:       "bafycid",
		Text:      "hello",
		Tags:      []string{},
		CreatedAt: time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC),
		IndexedAt: time.Date(2026, 5, 4, 12, 0, 1, 0, time.UTC),
	}
}

func TestBuildPostResponse_MinimalPost(t *testing.T) {
	t.Parallel()
	resp := api.BuildPostResponse(baseRow(), syntax.Handle("alice.example"))
	if resp.URI != "at://did:plc:alice/social.craftsky.feed.post/rk1" {
		t.Errorf("uri = %q", resp.URI)
	}
	if resp.Author.DID != "did:plc:alice" || resp.Author.Handle != "alice.example" {
		t.Errorf("author = %+v", resp.Author)
	}
	if resp.Reply != nil {
		t.Errorf("expected nil reply, got %+v", resp.Reply)
	}
	if resp.Quote != nil {
		t.Errorf("expected nil quote, got %+v", resp.Quote)
	}
	if resp.Author.DisplayName != nil {
		t.Errorf("expected nil displayName")
	}
	// Tags must serialise as []
	b, _ := json.Marshal(resp.Tags)
	if string(b) != "[]" {
		t.Errorf("tags = %s", b)
	}
}

func TestBuildPostResponse_WithReplyAndQuote(t *testing.T) {
	t.Parallel()
	row := baseRow()
	row.ReplyRootURI = ptrStr("at://did:plc:bob/social.craftsky.feed.post/r1")
	row.ReplyRootCID = ptrStr("bafyR1")
	row.ReplyParentURI = ptrStr("at://did:plc:bob/social.craftsky.feed.post/r2")
	row.ReplyParentCID = ptrStr("bafyR2")
	row.QuoteURI = ptrStr("at://did:plc:carol/social.craftsky.feed.post/q1")
	row.QuoteCID = ptrStr("bafyQ1")

	resp := api.BuildPostResponse(row, syntax.Handle("alice.example"))
	if resp.Reply == nil || resp.Reply.Root.URI != *row.ReplyRootURI {
		t.Errorf("reply: %+v", resp.Reply)
	}
	if resp.Reply.Parent.URI != *row.ReplyParentURI {
		t.Errorf("reply.parent: %+v", resp.Reply.Parent)
	}
	if resp.Quote == nil || resp.Quote.URI != *row.QuoteURI {
		t.Errorf("quote: %+v", resp.Quote)
	}
}

func TestBuildPostResponse_WithAuthorDisplayFields(t *testing.T) {
	t.Parallel()
	row := baseRow()
	row.AuthorDisplayName = ptrStr("Alice")
	row.AuthorAvatarCID = ptrStr("bafyAvatar")

	resp := api.BuildPostResponse(row, syntax.Handle("alice.example"))
	if resp.Author.DisplayName == nil || *resp.Author.DisplayName != "Alice" {
		t.Errorf("displayName = %v", resp.Author.DisplayName)
	}
	if resp.Author.AvatarCID == nil || *resp.Author.AvatarCID != "bafyAvatar" {
		t.Errorf("avatarCID = %v", resp.Author.AvatarCID)
	}
}

func TestBuildPostResponse_FacetsPassThrough(t *testing.T) {
	t.Parallel()
	row := baseRow()
	row.Facets = json.RawMessage(`[{"index":{"byteStart":0,"byteEnd":5}}]`)
	resp := api.BuildPostResponse(row, syntax.Handle("alice.example"))
	if string(resp.Facets) != string(row.Facets) {
		t.Errorf("facets = %s", resp.Facets)
	}
}
```

- [ ] **Step 5.2: Run tests to verify they fail**

```
just test ./appview/internal/api/... -run TestBuildPostResponse
```

Expected: build failure.

- [ ] **Step 5.3: Implement `post_response.go`**

Create `appview/internal/api/post_response.go`:

```go
// appview/internal/api/post_response.go
package api

import (
	"encoding/json"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// PostAuthor is the embedded author shape on every post-shaped response.
// Display name and avatar may be null when the user has no Bluesky
// profile mirror.
type PostAuthor struct {
	DID         string  `json:"did"`
	Handle      string  `json:"handle"`
	DisplayName *string `json:"displayName"`
	AvatarCID   *string `json:"avatarCid"`
}

// ResponseStrongRef is the JSON wire shape of a strongRef on a response
// body. Same fields as request StrongRef but kept distinct so request
// and response shapes can evolve independently.
type ResponseStrongRef struct {
	URI string `json:"uri"`
	CID string `json:"cid"`
}

// ResponseReply mirrors the lexicon's #replyRef on response bodies.
type ResponseReply struct {
	Root   ResponseStrongRef `json:"root"`
	Parent ResponseStrongRef `json:"parent"`
}

// PostResponse is the canonical wire shape returned by every
// post-shaped endpoint (POST, GET single, list items).
type PostResponse struct {
	URI       string             `json:"uri"`
	CID       string             `json:"cid"`
	Rkey      string             `json:"rkey"`
	Text      string             `json:"text"`
	Facets    json.RawMessage    `json:"facets"`
	Tags      []string           `json:"tags"`
	Reply     *ResponseReply     `json:"reply"`
	Quote     *ResponseStrongRef `json:"quote"`
	CreatedAt time.Time          `json:"createdAt"`
	IndexedAt time.Time          `json:"indexedAt"`
	Author    PostAuthor         `json:"author"`
}

// BuildPostResponse converts a PostRow + resolved handle into the wire
// response. Reply and quote pointers are flattened from the row's
// pointer columns into the lexicon-shaped nested objects.
func BuildPostResponse(row *PostRow, handle syntax.Handle) *PostResponse {
	tags := row.Tags
	if tags == nil {
		tags = []string{}
	}
	resp := &PostResponse{
		URI:       row.URI,
		CID:       row.CID,
		Rkey:      row.Rkey,
		Text:      row.Text,
		Facets:    row.Facets,
		Tags:      tags,
		CreatedAt: row.CreatedAt.UTC(),
		IndexedAt: row.IndexedAt.UTC(),
		Author: PostAuthor{
			DID:         row.DID,
			Handle:      handle.String(),
			DisplayName: row.AuthorDisplayName,
			AvatarCID:   row.AuthorAvatarCID,
		},
	}
	if row.ReplyRootURI != nil && row.ReplyParentURI != nil {
		resp.Reply = &ResponseReply{
			Root: ResponseStrongRef{
				URI: *row.ReplyRootURI,
				CID: derefOrEmpty(row.ReplyRootCID),
			},
			Parent: ResponseStrongRef{
				URI: *row.ReplyParentURI,
				CID: derefOrEmpty(row.ReplyParentCID),
			},
		}
	}
	if row.QuoteURI != nil {
		resp.Quote = &ResponseStrongRef{
			URI: *row.QuoteURI,
			CID: derefOrEmpty(row.QuoteCID),
		}
	}
	return resp
}

func derefOrEmpty(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
```

- [ ] **Step 5.4: Run tests to verify they pass**

```
just test ./appview/internal/api/... -run TestBuildPostResponse
```

Expected: all four tests pass.

- [ ] **Step 5.5: Commit**

```bash
git add appview/internal/api/post_response.go appview/internal/api/post_response_test.go
git commit -m "feat(appview): add PostResponse builder"
```

---

## Task 6: Implement `CreatePostHandler`

Server-stamps `createdAt`, calls `pds.CreateRecord`, builds a synthetic response. Tests use a fake PDS to assert the lexicon-shaped body that gets passed to the PDS.

**Files:**
- Create: `appview/internal/api/post.go` (handler #1; rest land in tasks 7–9)
- Create: `appview/internal/api/post_test.go` (handler #1 tests)

- [ ] **Step 6.1: Write the failing tests**

Create `appview/internal/api/post_test.go`:

```go
// appview/internal/api/post_test.go
package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/middleware"
)

// fakePDS records the last call. zero-value methods succeed; populate
// errors to simulate failures.
type fakePDS struct {
	mu             sync.Mutex
	lastCreateColl string
	lastCreateRec  any
	createURI      syntax.ATURI
	createCID      syntax.CID
	createErr      error

	lastDeleteRkey string
	deleteErr      error
}

func (f *fakePDS) GetRecord(_ context.Context, _ syntax.DID, _, _ string, _ any) (string, error) {
	return "", nil
}
func (f *fakePDS) PutRecord(_ context.Context, _ syntax.DID, _, _ string, _ any) error { return nil }
func (f *fakePDS) CreateRecord(_ context.Context, _ syntax.DID, coll string, rec any) (syntax.ATURI, syntax.CID, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lastCreateColl = coll
	f.lastCreateRec = rec
	if f.createErr != nil {
		return "", "", f.createErr
	}
	if f.createURI == "" {
		f.createURI = syntax.ATURI("at://did:plc:alice/social.craftsky.feed.post/rkSrv")
		f.createCID = syntax.CID("bafySrv")
	}
	return f.createURI, f.createCID, nil
}
func (f *fakePDS) DeleteRecord(_ context.Context, _ syntax.DID, _, rkey string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lastDeleteRkey = rkey
	return f.deleteErr
}

func newPDSFactory(p *fakePDS) auth.PDSClientFactory {
	return func(_ context.Context, _ syntax.DID, _ string) (auth.PDSClient, error) {
		return p, nil
	}
}

// fakePostStore implements api.PostReader for handler tests.
type fakePostStore struct {
	one        *api.PostRow
	oneErr     error
	listRows   []*api.PostRow
	listCursor string
	listErr    error
	author     *api.PostAuthorRow
	authorErr  error
	lastDID    string
	lastRkey   string
}

func (f *fakePostStore) ReadOne(_ context.Context, did, rkey string) (*api.PostRow, error) {
	f.lastDID = did
	f.lastRkey = rkey
	return f.one, f.oneErr
}
func (f *fakePostStore) ListByAuthor(_ context.Context, _ string, _ int, _ string) ([]*api.PostRow, string, error) {
	return f.listRows, f.listCursor, f.listErr
}
func (f *fakePostStore) ReadAuthor(_ context.Context, _ string) (*api.PostAuthorRow, error) {
	if f.author == nil && f.authorErr == nil {
		return &api.PostAuthorRow{}, nil
	}
	return f.author, f.authorErr
}

func authedReq(method, path, body string, did string) *http.Request {
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, path, nil)
	} else {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
	}
	ctx := middleware.WithDID(r.Context(), syntax.DID(did))
	return r.WithContext(ctx)
}

func TestCreatePost_HappyPath(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{}
	store := &fakePostStore{
		one: &api.PostRow{DID: "did:plc:alice", AuthorAvatarCID: nil, AuthorDisplayName: nil},
	}
	resolver := fakeResolver{handleFor: "alice.example"}
	h := api.CreatePostHandler(store, newPDSFactory(pds), resolver, nilLogger())
	req := authedReq(http.MethodPost, "/v1/posts", `{"text":"hello"}`, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var resp api.PostResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Text != "hello" || resp.URI == "" || resp.CID == "" {
		t.Errorf("resp = %+v", resp)
	}
	if resp.Rkey != "rkSrv" {
		t.Errorf("rkey not derived from PDS uri: %q", resp.Rkey)
	}
	if resp.Author.Handle != "alice.example" {
		t.Errorf("author.handle = %q", resp.Author.Handle)
	}

	body, _ := pds.lastCreateRec.(map[string]any)
	if body["$type"] != "social.craftsky.feed.post" {
		t.Errorf("missing/wrong $type: %v", body["$type"])
	}
	if _, ok := body["createdAt"].(string); !ok {
		t.Errorf("createdAt missing or non-string: %v", body["createdAt"])
	}
}

func TestCreatePost_MalformedBody_400(t *testing.T) {
	t.Parallel()
	h := api.CreatePostHandler(&fakePostStore{}, newPDSFactory(&fakePDS{}), fakeResolver{}, nilLogger())
	req := authedReq(http.MethodPost, "/v1/posts", `{not json`, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestCreatePost_TextEmpty_422(t *testing.T) {
	t.Parallel()
	h := api.CreatePostHandler(&fakePostStore{}, newPDSFactory(&fakePDS{}), fakeResolver{}, nilLogger())
	req := authedReq(http.MethodPost, "/v1/posts", `{"text":""}`, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestCreatePost_PDSDown_502(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{createErr: errors.New("pds down")}
	store := &fakePostStore{one: &api.PostRow{DID: "did:plc:alice"}}
	h := api.CreatePostHandler(store, newPDSFactory(pds), fakeResolver{handleFor: "a.example"}, nilLogger())
	req := authedReq(http.MethodPost, "/v1/posts", `{"text":"hi"}`, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
}

func TestCreatePost_QuoteEmbed_TranslatedToLexiconShape(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{}
	store := &fakePostStore{one: &api.PostRow{DID: "did:plc:alice"}}
	h := api.CreatePostHandler(store, newPDSFactory(pds), fakeResolver{handleFor: "a.example"}, nilLogger())
	body := `{"text":"hi","embed":{"quote":{"uri":"at://did:plc:bob/social.craftsky.feed.post/r1","cid":"bafyB"}}}`
	req := authedReq(http.MethodPost, "/v1/posts", body, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	rec, _ := pds.lastCreateRec.(map[string]any)
	embed, _ := rec["embed"].(map[string]any)
	if embed["$type"] != "social.craftsky.feed.post#quoteEmbed" {
		t.Errorf("embed $type: %v", embed["$type"])
	}
	r, _ := embed["record"].(map[string]any)
	if r["uri"] != "at://did:plc:bob/social.craftsky.feed.post/r1" {
		t.Errorf("embed.record.uri = %v", r["uri"])
	}
}

```

- [ ] **Step 6.2: Run tests to verify they fail**

```
just test ./appview/internal/api/... -run TestCreatePost
```

Expected: build failure (`api.CreatePostHandler` undefined).

- [ ] **Step 6.3: Implement `post.go` with `CreatePostHandler`**

Create `appview/internal/api/post.go`:

```go
// appview/internal/api/post.go
package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/postutil"
)

const craftskyPostNSID = "social.craftsky.feed.post"

// CreatePostHandler serves POST /v1/posts.
func CreatePostHandler(
	store PostReader,
	newPDS auth.PDSClientFactory,
	resolver HandleResolver,
	logger *slog.Logger,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		did, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "no did in context", runID, nil)
			return
		}
		sessionID, _ := middleware.GetOAuthSessionID(r.Context())

		req, err := DecodePostCreate(r.Body)
		if err != nil {
			fe, isFE := err.(*FieldError)
			switch {
			case isFE && fe.Code == "malformed_body":
				envelope.WriteError(w, http.StatusBadRequest,
					"malformed_body", "could not parse body", runID, fe.Fields)
			case isFE:
				envelope.WriteError(w, http.StatusBadRequest,
					fe.Code, "request body rejected", runID, fe.Fields)
			default:
				envelope.WriteError(w, http.StatusBadRequest,
					"malformed_body", "could not parse body", runID, nil)
			}
			return
		}
		if err := ValidatePostCreate(req); err != nil {
			fe, isFE := err.(*FieldError)
			if isFE {
				envelope.WriteError(w, http.StatusUnprocessableEntity,
					fe.Code, "validation failed", runID, fe.Fields)
				return
			}
			envelope.WriteError(w, http.StatusUnprocessableEntity,
				"validation_failed", "validation failed", runID, nil)
			return
		}

		body := lexiconRecordBody(req)

		pds, err := newPDS(r.Context(), did, sessionID)
		if err != nil {
			logger.Error("post: newPDS failed", slog.String("err", err.Error()))
			envelope.WriteError(w, http.StatusBadGateway,
				"pds_unavailable", "could not contact PDS", runID, nil)
			return
		}
		uri, cid, err := pds.CreateRecord(r.Context(), did, craftskyPostNSID, body)
		if err != nil {
			logger.Warn("post: CreateRecord failed",
				slog.String("did", did.String()), slog.String("err", err.Error()))
			envelope.WriteError(w, http.StatusBadGateway,
				"pds_write_failed", "PDS rejected the post", runID, nil)
			return
		}

		row, err := syntheticPostRow(r, store, did, uri, cid, req)
		if err != nil {
			logger.Error("post: hydrate author failed", slog.String("err", err.Error()))
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "post created but hydrate failed", runID, nil)
			return
		}
		handle, err := resolver.ResolveHandle(r.Context(), did)
		if err != nil {
			logger.Warn("post: ResolveHandle failed", slog.String("err", err.Error()))
			envelope.WriteError(w, http.StatusBadGateway,
				"identity_unavailable", "could not resolve handle", runID, nil)
			return
		}
		resp := BuildPostResponse(row, handle)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(resp)
	})
}

// lexiconRecordBody translates the wire request into the lexicon-shaped
// record body that goes to the PDS.
func lexiconRecordBody(req PostCreateRequest) map[string]any {
	body := map[string]any{
		"$type":     craftskyPostNSID,
		"text":      req.Text,
		"createdAt": time.Now().UTC().Format(time.RFC3339),
	}
	if len(req.Facets) > 0 {
		body["facets"] = req.Facets
	}
	if req.Reply != nil {
		body["reply"] = map[string]any{
			"root":   map[string]any{"uri": req.Reply.Root.URI, "cid": req.Reply.Root.CID},
			"parent": map[string]any{"uri": req.Reply.Parent.URI, "cid": req.Reply.Parent.CID},
		}
	}
	if req.Embed != nil && req.Embed.Quote != nil {
		body["embed"] = map[string]any{
			"$type": craftskyPostNSID + "#quoteEmbed",
			"record": map[string]any{
				"uri": req.Embed.Quote.URI,
				"cid": req.Embed.Quote.CID,
			},
		}
	}
	return body
}

// syntheticPostRow assembles the PostRow that BuildPostResponse needs
// from the request body, the PDS-assigned (uri, cid), and a single
// author lookup against the store. We don't wait for the firehose to
// land the row.
func syntheticPostRow(
	r *http.Request,
	store PostReader,
	did syntax.DID,
	uri syntax.ATURI,
	cid syntax.CID,
	req PostCreateRequest,
) (*PostRow, error) {
	now := time.Now().UTC()
	row := &PostRow{
		URI:       string(uri),
		DID:       did.String(),
		Rkey:      path.Base(string(uri)),
		CID:       string(cid),
		Text:      req.Text,
		Tags:      postutil.ExtractTags(req.Facets),
		CreatedAt: now,
		IndexedAt: now,
	}
	if len(req.Facets) > 0 {
		raw, err := json.Marshal(req.Facets)
		if err != nil {
			return nil, err
		}
		row.Facets = raw
	}
	if req.Reply != nil {
		row.ReplyRootURI = strPtr(req.Reply.Root.URI)
		row.ReplyRootCID = strPtr(req.Reply.Root.CID)
		row.ReplyParentURI = strPtr(req.Reply.Parent.URI)
		row.ReplyParentCID = strPtr(req.Reply.Parent.CID)
	}
	if req.Embed != nil && req.Embed.Quote != nil {
		row.QuoteURI = strPtr(req.Embed.Quote.URI)
		row.QuoteCID = strPtr(req.Embed.Quote.CID)
	}

	author, err := store.ReadAuthor(r.Context(), did.String())
	if err != nil {
		return nil, err
	}
	if author != nil {
		row.AuthorDisplayName = author.DisplayName
		row.AuthorAvatarCID = author.AvatarCID
	}
	return row, nil
}

func strPtr(s string) *string { return &s }

// trimAt strips a leading "@" from a path segment (used by handle-or-DID
// inputs). Defined here so it's callable from the post list handler in
// task 9 without re-importing strings everywhere.
func trimAt(s string) string { return strings.TrimPrefix(s, "@") }
```

Note: the synthetic-author lookup is intentionally best-effort — for a brand-new author who has just created their first post, `ReadOne` will return `ErrPostNotFound` because the firehose hasn't landed the row yet. In that case we leave display fields nil. A subsequent `GET /v1/posts/...` after the firehose arrives will return the populated fields.

- [ ] **Step 6.4: Run tests to verify they pass**

```
just test ./appview/internal/api/... -run TestCreatePost
```

Expected: all five tests pass.

- [ ] **Step 6.5: Commit**

```bash
git add appview/internal/api/post.go appview/internal/api/post_test.go
git commit -m "feat(appview): add POST /v1/posts handler"
```

---

## Task 7: Implement `GetPostHandler`

Single-row read; 404 on miss; 502 on handle-resolution failure.

**Files:**
- Modify: `appview/internal/api/post.go`
- Modify: `appview/internal/api/post_test.go`

- [ ] **Step 7.1: Append failing tests to `post_test.go`**

```go
func TestGetPost_HappyPath(t *testing.T) {
	t.Parallel()
	row := &api.PostRow{
		URI: "at://did:plc:alice/social.craftsky.feed.post/rk1",
		DID: "did:plc:alice", Rkey: "rk1", CID: "bafy", Text: "hi",
	}
	store := &fakePostStore{one: row}
	h := api.GetPostHandler(store, fakeResolver{handleFor: "alice.example"}, nilLogger())
	req := authedReq(http.MethodGet, "/v1/posts/did:plc:alice/rk1", "", "did:plc:alice")
	req.SetPathValue("did", "did:plc:alice")
	req.SetPathValue("rkey", "rk1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var resp api.PostResponse
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Text != "hi" || resp.Author.Handle != "alice.example" {
		t.Errorf("resp = %+v", resp)
	}
}

func TestGetPost_NotFound_404(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{oneErr: api.ErrPostNotFound}
	h := api.GetPostHandler(store, fakeResolver{handleFor: "alice.example"}, nilLogger())
	req := authedReq(http.MethodGet, "/v1/posts/did:plc:alice/rk1", "", "did:plc:alice")
	req.SetPathValue("did", "did:plc:alice")
	req.SetPathValue("rkey", "rk1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestGetPost_BadDID_400(t *testing.T) {
	t.Parallel()
	h := api.GetPostHandler(&fakePostStore{}, fakeResolver{}, nilLogger())
	req := authedReq(http.MethodGet, "/v1/posts/not-a-did/rk1", "", "did:plc:alice")
	req.SetPathValue("did", "not-a-did")
	req.SetPathValue("rkey", "rk1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestGetPost_HandleResolutionFailure_502(t *testing.T) {
	t.Parallel()
	row := &api.PostRow{DID: "did:plc:alice", Rkey: "rk1"}
	store := &fakePostStore{one: row}
	h := api.GetPostHandler(store, fakeResolver{err: errors.New("plc down")}, nilLogger())
	req := authedReq(http.MethodGet, "/v1/posts/did:plc:alice/rk1", "", "did:plc:alice")
	req.SetPathValue("did", "did:plc:alice")
	req.SetPathValue("rkey", "rk1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d", rr.Code)
	}
}
```

- [ ] **Step 7.2: Run tests to verify they fail**

```
just test ./appview/internal/api/... -run TestGetPost
```

Expected: build failure (`api.GetPostHandler` undefined).

- [ ] **Step 7.3: Append `GetPostHandler` to `post.go`**

```go
// GetPostHandler serves GET /v1/posts/{did}/{rkey}.
func GetPostHandler(store PostReader, resolver HandleResolver, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		did, err := syntax.ParseDID(r.PathValue("did"))
		if err != nil {
			envelope.WriteError(w, http.StatusBadRequest,
				"invalid_identifier", "did path segment is not a valid DID", runID, nil)
			return
		}
		rkey := r.PathValue("rkey")
		row, err := store.ReadOne(r.Context(), did.String(), rkey)
		if errors.Is(err, ErrPostNotFound) {
			envelope.WriteError(w, http.StatusNotFound,
				"post_not_found", "post not found", runID, nil)
			return
		}
		if err != nil {
			logger.Error("post: ReadOne failed",
				slog.String("did", did.String()),
				slog.String("rkey", rkey),
				slog.String("err", err.Error()))
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "post read failed", runID, nil)
			return
		}
		handle, err := resolver.ResolveHandle(r.Context(), did)
		if err != nil {
			logger.Warn("post: ResolveHandle failed",
				slog.String("did", did.String()), slog.String("err", err.Error()))
			envelope.WriteError(w, http.StatusBadGateway,
				"identity_unavailable", "could not resolve handle", runID, nil)
			return
		}
		resp := BuildPostResponse(row, handle)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	})
}
```

- [ ] **Step 7.4: Run tests to verify they pass**

```
just test ./appview/internal/api/... -run TestGetPost
```

Expected: all four tests pass.

- [ ] **Step 7.5: Commit**

```bash
git add appview/internal/api/post.go appview/internal/api/post_test.go
git commit -m "feat(appview): add GET /v1/posts/{did}/{rkey} handler"
```

---

## Task 8: Implement `DeletePostHandler`

Authorise (caller DID must equal path DID), call `pds.DeleteRecord`, swallow record-not-found, return 204.

**Files:**
- Modify: `appview/internal/api/post.go`
- Modify: `appview/internal/api/post_test.go`

- [ ] **Step 8.1: Append failing tests to `post_test.go`**

```go
func TestDeletePost_Self_204_CallsPDS(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{}
	h := api.DeletePostHandler(newPDSFactory(pds), nilLogger())
	req := authedReq(http.MethodDelete, "/v1/posts/did:plc:alice/rk1", "", "did:plc:alice")
	req.SetPathValue("did", "did:plc:alice")
	req.SetPathValue("rkey", "rk1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if pds.lastDeleteRkey != "rk1" {
		t.Errorf("PDS not called: %q", pds.lastDeleteRkey)
	}
}

func TestDeletePost_OtherUser_403_NoPDSCall(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{}
	h := api.DeletePostHandler(newPDSFactory(pds), nilLogger())
	req := authedReq(http.MethodDelete, "/v1/posts/did:plc:bob/rk1", "", "did:plc:alice")
	req.SetPathValue("did", "did:plc:bob")
	req.SetPathValue("rkey", "rk1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d", rr.Code)
	}
	if pds.lastDeleteRkey != "" {
		t.Errorf("PDS should not have been called")
	}
}

func TestDeletePost_RecordAlreadyGone_204_Idempotent(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{deleteErr: auth.ErrRecordNotFound}
	h := api.DeletePostHandler(newPDSFactory(pds), nilLogger())
	req := authedReq(http.MethodDelete, "/v1/posts/did:plc:alice/rk1", "", "did:plc:alice")
	req.SetPathValue("did", "did:plc:alice")
	req.SetPathValue("rkey", "rk1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestDeletePost_PDSDown_502(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{deleteErr: errors.New("pds down")}
	h := api.DeletePostHandler(newPDSFactory(pds), nilLogger())
	req := authedReq(http.MethodDelete, "/v1/posts/did:plc:alice/rk1", "", "did:plc:alice")
	req.SetPathValue("did", "did:plc:alice")
	req.SetPathValue("rkey", "rk1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestDeletePost_BadDID_400(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{}
	h := api.DeletePostHandler(newPDSFactory(pds), nilLogger())
	req := authedReq(http.MethodDelete, "/v1/posts/not-a-did/rk1", "", "did:plc:alice")
	req.SetPathValue("did", "not-a-did")
	req.SetPathValue("rkey", "rk1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rr.Code)
	}
}
```

- [ ] **Step 8.2: Run tests to verify they fail**

```
just test ./appview/internal/api/... -run TestDeletePost
```

Expected: build failure.

- [ ] **Step 8.3: Append `DeletePostHandler` to `post.go`**

```go
// DeletePostHandler serves DELETE /v1/posts/{did}/{rkey}. Idempotent —
// returns 204 even if the underlying record was already gone.
func DeletePostHandler(newPDS auth.PDSClientFactory, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		caller, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "no did in context", runID, nil)
			return
		}
		did, err := syntax.ParseDID(r.PathValue("did"))
		if err != nil {
			envelope.WriteError(w, http.StatusBadRequest,
				"invalid_identifier", "did path segment is not a valid DID", runID, nil)
			return
		}
		if did != caller {
			envelope.WriteError(w, http.StatusForbidden,
				"forbidden", "cannot delete another user's post", runID, nil)
			return
		}
		rkey := r.PathValue("rkey")
		sessionID, _ := middleware.GetOAuthSessionID(r.Context())
		pds, err := newPDS(r.Context(), did, sessionID)
		if err != nil {
			logger.Error("post: newPDS failed", slog.String("err", err.Error()))
			envelope.WriteError(w, http.StatusBadGateway,
				"pds_unavailable", "could not contact PDS", runID, nil)
			return
		}
		if err := pds.DeleteRecord(r.Context(), did, craftskyPostNSID, rkey); err != nil {
			if errors.Is(err, auth.ErrRecordNotFound) {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			logger.Warn("post: DeleteRecord failed",
				slog.String("did", did.String()), slog.String("rkey", rkey),
				slog.String("err", err.Error()))
			envelope.WriteError(w, http.StatusBadGateway,
				"pds_unavailable", "PDS delete failed", runID, nil)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}
```

- [ ] **Step 8.4: Run tests to verify they pass**

```
just test ./appview/internal/api/... -run TestDeletePost
```

Expected: all five tests pass.

- [ ] **Step 8.5: Commit**

```bash
git add appview/internal/api/post.go appview/internal/api/post_test.go
git commit -m "feat(appview): add DELETE /v1/posts/{did}/{rkey} handler"
```

---

## Task 9: Implement `ListPostsByAuthorHandler`

Resolve `{handleOrDid}`, parse `limit`/`cursor`, list posts, hydrate handle once, build per-item responses.

**Files:**
- Modify: `appview/internal/api/post.go`
- Modify: `appview/internal/api/post_test.go`

- [ ] **Step 9.1: Append failing tests to `post_test.go`**

```go
func TestListPosts_HappyPath_PaginatesCorrectly(t *testing.T) {
	t.Parallel()
	rows := []*api.PostRow{
		{URI: "at://did:plc:alice/social.craftsky.feed.post/rk2", DID: "did:plc:alice", Rkey: "rk2", Text: "second"},
		{URI: "at://did:plc:alice/social.craftsky.feed.post/rk1", DID: "did:plc:alice", Rkey: "rk1", Text: "first"},
	}
	store := &fakePostStore{listRows: rows, listCursor: "next-cursor-opaque"}
	h := api.ListPostsByAuthorHandler(store, fakeResolver{handleFor: "alice.example"}, nilLogger())
	req := authedReq(http.MethodGet, "/v1/profiles/@did:plc:alice/posts?limit=2", "", "did:plc:alice")
	req.SetPathValue("handleOrDid", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Items  []api.PostResponse `json:"items"`
		Cursor string             `json:"cursor,omitempty"`
	}
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if len(resp.Items) != 2 {
		t.Fatalf("items len = %d", len(resp.Items))
	}
	if resp.Items[0].Rkey != "rk2" {
		t.Errorf("ordering wrong: %q", resp.Items[0].Rkey)
	}
	if resp.Cursor != "next-cursor-opaque" {
		t.Errorf("cursor = %q", resp.Cursor)
	}
}

func TestListPosts_ResolvesHandle(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{listRows: []*api.PostRow{}}
	resolver := fakeResolver{
		didFor:    syntax.DID("did:plc:alice"),
		handleFor: syntax.Handle("alice.example"),
	}
	h := api.ListPostsByAuthorHandler(store, resolver, nilLogger())
	req := authedReq(http.MethodGet, "/v1/profiles/@alice.example/posts", "", "did:plc:bob")
	req.SetPathValue("handleOrDid", "alice.example")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
}

func TestListPosts_BadCursor_400(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{listErr: envelope.ErrInvalidCursor}
	h := api.ListPostsByAuthorHandler(store, fakeResolver{handleFor: "alice.example"}, nilLogger())
	req := authedReq(http.MethodGet, "/v1/profiles/@did:plc:alice/posts?cursor=garbage", "", "did:plc:alice")
	req.SetPathValue("handleOrDid", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestListPosts_FinalPage_OmitsCursorField(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{listRows: []*api.PostRow{}, listCursor: ""}
	h := api.ListPostsByAuthorHandler(store, fakeResolver{handleFor: "alice.example"}, nilLogger())
	req := authedReq(http.MethodGet, "/v1/profiles/@did:plc:alice/posts", "", "did:plc:alice")
	req.SetPathValue("handleOrDid", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	if strings.Contains(rr.Body.String(), `"cursor"`) {
		t.Errorf("cursor field should be omitted, got body: %s", rr.Body.String())
	}
}

func TestListPosts_LimitDefaultAndCap(t *testing.T) {
	t.Parallel()
	captured := struct {
		limit int
	}{}
	store := &fakePostStoreCapturing{
		fakePostStore: fakePostStore{listRows: []*api.PostRow{}},
		captured:      &captured,
	}
	h := api.ListPostsByAuthorHandler(store, fakeResolver{handleFor: "alice.example"}, nilLogger())
	req := authedReq(http.MethodGet, "/v1/profiles/@did:plc:alice/posts?limit=500", "", "did:plc:alice")
	req.SetPathValue("handleOrDid", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	if captured.limit != 100 {
		t.Errorf("limit = %d, want capped at 100", captured.limit)
	}
}
```

Add the capturing-variant store helper near the existing `fakePostStore`:

```go
type fakePostStoreCapturing struct {
	fakePostStore
	captured *struct{ limit int }
}

func (f *fakePostStoreCapturing) ListByAuthor(_ context.Context, _ string, limit int, _ string) ([]*api.PostRow, string, error) {
	f.captured.limit = limit
	return f.listRows, f.listCursor, f.listErr
}
```

- [ ] **Step 9.2: Run tests to verify they fail**

```
just test ./appview/internal/api/... -run TestListPosts
```

Expected: build failure (`api.ListPostsByAuthorHandler` undefined).

- [ ] **Step 9.3: Append `ListPostsByAuthorHandler` to `post.go`**

```go
// ListPostsByAuthorHandler serves GET /v1/profiles/{handleOrDid}/posts.
func ListPostsByAuthorHandler(
	store PostReader,
	resolver HandleResolver,
	logger *slog.Logger,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		raw := trimAt(r.PathValue("handleOrDid"))
		did, err := resolveToDID(r.Context(), raw, resolver)
		if err != nil {
			switch {
			case errors.Is(err, errInvalidIdentifier):
				envelope.WriteError(w, http.StatusBadRequest,
					"invalid_identifier", "not a valid handle or DID", runID, nil)
			default:
				logger.Warn("post list: ResolveDID failed",
					slog.String("input", raw), slog.String("err", err.Error()))
				envelope.WriteError(w, http.StatusBadGateway,
					"identity_unavailable", "could not resolve identity", runID, nil)
			}
			return
		}
		limit := parseLimit(r.URL.Query().Get("limit"))
		cursor := r.URL.Query().Get("cursor")

		rows, nextCursor, err := store.ListByAuthor(r.Context(), did.String(), limit, cursor)
		if err != nil {
			if errors.Is(err, envelope.ErrInvalidCursor) {
				envelope.WriteError(w, http.StatusBadRequest,
					"invalid_cursor", "cursor could not be decoded", runID, nil)
				return
			}
			logger.Error("post list: ListByAuthor failed",
				slog.String("did", did.String()), slog.String("err", err.Error()))
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "post list failed", runID, nil)
			return
		}

		items := make([]*PostResponse, 0, len(rows))
		if len(rows) > 0 {
			// Only pay handle-resolution cost when there are rows to render.
			handle, herr := resolver.ResolveHandle(r.Context(), did)
			if herr != nil {
				logger.Warn("post list: ResolveHandle failed",
					slog.String("did", did.String()), slog.String("err", herr.Error()))
				envelope.WriteError(w, http.StatusBadGateway,
					"identity_unavailable", "could not resolve handle", runID, nil)
				return
			}
			for _, row := range rows {
				items = append(items, BuildPostResponse(row, handle))
			}
		}
		body := struct {
			Items  []*PostResponse `json:"items"`
			Cursor string          `json:"cursor,omitempty"`
		}{Items: items, Cursor: nextCursor}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(body)
	})
}

// parseLimit returns the validated limit, defaulting to 50 and capping
// at 100. Per pagination spec §5: caps are silent (we don't 400 on
// overshoot, we cap).
func parseLimit(raw string) int {
	const defaultLimit, maxLimit = 50, 100
	if raw == "" {
		return defaultLimit
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return defaultLimit
	}
	if n > maxLimit {
		return maxLimit
	}
	return n
}
```

Add `"strconv"` to the imports at the top of `post.go`.

- [ ] **Step 9.4: Run tests to verify they pass**

```
just test ./appview/internal/api/... -run TestListPosts
```

Expected: all five tests pass.

- [ ] **Step 9.5: Run full api package suite**

```
just test ./appview/internal/api/...
```

Expected: every TestPost*, TestList*, TestGet*, TestDelete*, TestCreate*, TestBuildPostResponse*, TestDecodePostCreate*, TestValidatePostCreate*, plus all existing profile tests pass.

- [ ] **Step 9.6: Commit**

```bash
git add appview/internal/api/post.go appview/internal/api/post_test.go
git commit -m "feat(appview): add GET /v1/profiles/{handleOrDid}/posts handler"
```

---

## Task 10: Wire routes

Add four `mux.Handle` lines to `routes.go`. No new tests beyond a smoke verification that the router compiles and serves.

**Files:**
- Modify: `appview/internal/routes/routes.go`

- [ ] **Step 10.1: Add the routes**

In `appview/internal/routes/routes.go`, after the existing profile routes (after the line registering `PUT /v1/profiles/me`) and before the `mux.Handle("/", http.NotFoundHandler())` fallthrough, add:

```go
postStore := api.NewPostStore(deps.DB)
mux.Handle("POST /v1/posts",
	authN(deviceID(api.CreatePostHandler(postStore, deps.NewPDSClient, deps.HandleResolver, deps.Logger))))
mux.Handle("GET /v1/posts/{did}/{rkey}",
	authN(deviceID(api.GetPostHandler(postStore, deps.HandleResolver, deps.Logger))))
mux.Handle("DELETE /v1/posts/{did}/{rkey}",
	authN(deviceID(api.DeletePostHandler(deps.NewPDSClient, deps.Logger))))
mux.Handle("GET /v1/profiles/{handleOrDid}/posts",
	authN(deviceID(api.ListPostsByAuthorHandler(postStore, deps.HandleResolver, deps.Logger))))
```

- [ ] **Step 10.2: Build the package**

```
just test ./appview/internal/routes/...
```

Expected: existing routes tests pass; the four new routes compile and resolve their handlers.

- [ ] **Step 10.3: Run the full appview test suite**

```
just test ./appview/...
```

Expected: every package green.

- [ ] **Step 10.4: Commit**

```bash
git add appview/internal/routes/routes.go
git commit -m "feat(appview): wire post CRUD routes"
```

---

## Task 11: Smoke-test against the running stack (optional but recommended)

Verify the round-trip works end-to-end with the docker compose stack. This is a manual verification step; do not commit changes from this task unless the smoke test reveals a bug.

**Files:** none modified.

- [ ] **Step 11.1: Start the dev stack**

```
just dev
```

Wait until `appview` logs say it's listening on its port and Tap has connected.

- [ ] **Step 11.2: Authenticate and obtain a session token**

Follow the existing OAuth login flow in the goat-based smoke documentation referenced from the appview README (the project established this in commit `8ecd80e` "docs(appview): document goat-based smoke testing for the post indexer"). Set `TOKEN` and `DEVICE_ID` env vars from the resulting session.

- [ ] **Step 11.3: Create a post**

```
curl -sS -X POST http://localhost:<appview-port>/v1/posts \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Craftsky-Device-Id: $DEVICE_ID" \
  -H "Content-Type: application/json" \
  -d '{"text":"smoke test post"}'
```

Expected: HTTP 201 with `PostResponse` JSON. Capture `uri` and the `did/rkey` from it.

- [ ] **Step 11.4: Read the post back**

```
curl -sS "http://localhost:<port>/v1/posts/<did>/<rkey>" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Craftsky-Device-Id: $DEVICE_ID"
```

Expected: HTTP 200; the same post body. (Wait ~5s after step 3 if the firehose hasn't landed yet — first read may briefly return 404 until the indexer commits.)

- [ ] **Step 11.5: List your own posts**

```
curl -sS "http://localhost:<port>/v1/profiles/@<did>/posts" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Craftsky-Device-Id: $DEVICE_ID"
```

Expected: HTTP 200; the post appears in `items`.

- [ ] **Step 11.6: Delete the post**

```
curl -sS -X DELETE "http://localhost:<port>/v1/posts/<did>/<rkey>" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Craftsky-Device-Id: $DEVICE_ID" \
  -i
```

Expected: HTTP 204; empty body.

- [ ] **Step 11.7: Confirm delete propagated**

After ~5s, repeat step 11.4. Expected: HTTP 404 once the firehose tombstone reaches the indexer. (Per the spec, the delete is asynchronous from the local DB's perspective.)

If any step fails, file a follow-up task — the unit tests caught the contract-level concerns; the smoke test caught wiring or env-specific bugs.

---

## Done

After Task 10 (and optionally Task 11):

- Four new routes serve under `/v1/`.
- Two new methods on `PDSClient` (`CreateRecord`, `DeleteRecord`) are available for follow-up record types (likes, follows).
- `postutil.ExtractTags` is shared between the indexer and the create handler — no drift risk.
- All commits build and pass tests on their own; bisect stays useful.
