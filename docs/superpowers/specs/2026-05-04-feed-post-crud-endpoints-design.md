# Feed Post CRUD Endpoints Design

- **Status:** Draft
- **Date:** 2026-05-04
- **Related:**
  - [appview-api-architecture](2026-04-21-appview-api-architecture-design.md) — fixes the URL conventions, auth headers, error envelope, and pagination shape this spec slots into.
  - [api-wire-alignment](2026-04-22-api-wire-alignment-design.md) — codifies camelCase across the entire `/v1/*` surface.
  - [feed-post-indexing](2026-05-04-feed-post-indexing-design.md) — the storage half. This spec is the read/write half on top of the table and indexer it built.
  - [post-lexicon-fields](2026-04-23-post-lexicon-fields-design.md) — locks the `social.craftsky.feed.post` lexicon shape this surface mediates.
  - [appview-oauth-bff](2026-04-18-appview-oauth-bff-design.md) — owns the OAuth/DPoP layer; the new `PDSClient` methods sit on top of the same client factory.
  - [profile-onboarding](2026-04-23-profile-onboarding-design.md) — the `PUT /v1/profiles/me` write-proxy pattern this spec mirrors.

## Summary

Add the read and write endpoints the Flutter client needs to deal with `social.craftsky.feed.post` records: create a post, fetch a single post, list a user's own posts, and delete a post. Author hydration (display name, avatar) is embedded in every post response so the client can render a feed without N+1 lookups.

This spec covers a deliberately narrow CRUD slice — no thread fetch, no global feed, no likes, no images. Each of those has its own forcing function (image blob upload, like indexer, follow graph, thread query) and is appropriately a separate spec.

## Goals

1. Get the Flutter client a working text-post round-trip end-to-end: compose → create → see it on a profile → delete it.
2. Match the existing API conventions (`/v1/`, camelCase, error envelope, opaque cursors) and the existing handler shape ([profile.go](../../../appview/internal/api/profile.go)) so a contributor reading `post.go` next to `profile.go` sees the same structure.
3. Extend the `PDSClient` interface once with `CreateRecord` and `DeleteRecord` so this surface — and every future record-write endpoint (likes, follows, reposts) — uses the same primitive.
4. Keep author hydration cheap: one join, fetched once per request and zipped into list items.

## Non-goals

- **Thread fetch (`GET /v1/posts/{did}/{rkey}/thread`)** — needs a recursive `reply_*` traversal and is its own spec.
- **Global / follow feed (`GET /v1/feed/timeline`)** — the API spec defines it as "followed accounts" and the follow graph isn't indexed yet. Separate spec.
- **Likes (`POST/DELETE /v1/posts/{did}/{rkey}/likes`)** — needs a `social.craftsky.feed.like` indexer first.
- **Images on posts** — blob upload is deferred per [appview-api-architecture §non-goals](2026-04-21-appview-api-architecture-design.md). The post lexicon's `images` field stays unwritable from this surface; the response shape omits an `images` field entirely until blob upload lands.
- **Project-field round-trip on response** — the indexer doesn't materialise project fields yet ([feed-post-indexing §non-goals](2026-05-04-feed-post-indexing-design.md)). We accept project fields on neither the request body (Section 1) nor the response shape (Section 3); a project post would round-trip via `record JSONB` only.
- **Reply/quote target validation** — pass-through, no existence check. The indexer spec already commits to "store URI/CID verbatim and let the read endpoint handle 'target not indexed' rendering"; this spec is the write side of that posture.
- **Pre-emptive postgres mutation on write/delete** — the firehose tombstone or insert is the source of truth for the local row. We rely on the same eventually-consistent model the existing `PUT /v1/profiles/me` already uses.

## Context

### What's already in place

- The lexicon `social.craftsky.feed.post` is locked at [lexicon/social/craftsky/feed/post.json](../../../lexicon/social/craftsky/feed/post.json). Generated Go types in [appview/internal/lexicon/craftsky/feedpost.go](../../../appview/internal/lexicon/craftsky/feedpost.go).
- The `craftsky_posts` table is populated by [`CraftskyPost`](../../../appview/internal/index/craftsky_post.go), gated on `craftsky_profiles` membership, ordered by server-side `indexed_at`.
- The `PDSClient` interface ([appview/internal/auth/pds_client.go](../../../appview/internal/auth/pds_client.go)) exposes `GetRecord` and `PutRecord`. No `CreateRecord` / `DeleteRecord` yet.
- The handler pattern is established by [profile.go](../../../appview/internal/api/profile.go): typed request struct, decode/validate split into separate functions returning `*FieldError`, dual-write to PDS, synthetic response built without waiting for the indexer.
- The error envelope ([envelope/](../../../appview/internal/api/envelope/)) is shared across handlers; new endpoints reuse `envelope.WriteError`.
- Auth + device-id middleware is already wired and applies to every `/v1/*` route ([routes.go:39-47](../../../appview/internal/routes/routes.go)).

### Scope decisions

Three resolved during brainstorming:

- **Endpoint set:** create, read-single, delete, list-by-author. No thread, no feed, no likes, no images.
- **Write fields:** `text`, `facets`, `reply`, `embed.quoteEmbed`. No `project`, no `images`. Symmetric with what the indexer materialises and the response shape returns.
- **Author hydration:** every post response embeds `{did, handle, displayName, avatarCid}`. One join; one fetch per request (re-zipped into list items) since list-by-author shares an author across all rows.

## Design

### 1. URL surface

| Method | Path | Auth | Purpose |
|---|---|---|---|
| `POST` | `/v1/posts` | yes | Create a post on the caller's PDS |
| `GET` | `/v1/posts/{did}/{rkey}` | yes | Fetch a single post |
| `DELETE` | `/v1/posts/{did}/{rkey}` | yes | Delete a post (must be caller's) |
| `GET` | `/v1/profiles/{handleOrDid}/posts` | yes | A user's own posts, newest first |

`{did}` on the post paths is a bare `syntax.DID`; no handle support there per [api architecture §2.2](2026-04-21-appview-api-architecture-design.md). `{handleOrDid}` on the profile-posts path uses the existing `@`-prefix convention and reuses the existing `resolveToDID` helper from `profile.go`.

### 2. Request/response shapes

#### `POST /v1/posts` request

```json
{
  "text": "Cast on for the Hitchhiker shawl tonight.",
  "facets": [ ... ] | omitted,
  "reply": {
    "root":   { "uri": "at://...", "cid": "bafy..." },
    "parent": { "uri": "at://...", "cid": "bafy..." }
  } | omitted,
  "embed": {
    "quote": { "uri": "at://...", "cid": "bafy..." }
  } | omitted
}
```

- `text` — required, ≤ 2000 graphemes (lexicon limit; not byte length).
- `facets` — pass-through array. Lexicon-validated by the receiving PDS; this surface does not parse the union internals.
- `reply.root` and `reply.parent` — both required when `reply` is present (lexicon requires both); `parent.uri` and `parent.cid` are validated as a parseable AT-URI / CID pair but not checked for existence anywhere.
- `embed.quote` — same: validated parseable, not checked for existence. Wire shape uses `embed.quote` (one nesting layer over `{uri, cid}`); the AppView translates this to the lexicon's `embed.quoteEmbed.record` shape before writing to the PDS.
- `createdAt` — **not on the request**. Server-stamped to `now()`. Switching to client-supplied later is additive (accept if present, stamp if absent).

#### `PostResponse` (used by every endpoint that returns a post)

```json
{
  "uri": "at://did:plc:xyz/social.craftsky.feed.post/3lf2abc",
  "cid": "bafy...",
  "rkey": "3lf2abc",
  "text": "Cast on for the Hitchhiker shawl tonight.",
  "facets": [ ... ] | null,
  "tags": ["knitting", "shawl"],
  "reply": {
    "root":   { "uri": "...", "cid": "..." },
    "parent": { "uri": "...", "cid": "..." }
  } | null,
  "quote": { "uri": "...", "cid": "..." } | null,
  "createdAt": "2026-05-04T18:23:45Z",
  "indexedAt": "2026-05-04T18:23:47Z",
  "author": {
    "did": "did:plc:xyz",
    "handle": "alice.craftsky.social",
    "displayName": "Alice",
    "avatarCid": "bafy..." | null
  }
}
```

- `facets` — pass-through of stored JSONB. `null` (not `[]`) when absent, matching what the indexer stores.
- `tags` — server-extracted from facets (already in the DB column); always an array.
- `reply` / `quote` — flattened from the four/two DB columns into the lexicon-shaped nested objects. `null` when absent. Note: the response uses `quote` not `embed.quote` — the response shape is post-centric, the request shape mirrors the lexicon's `embed` union. Asymmetric on purpose: clients reading a post want a single `quote` field, but writing one happens through the open `embed` union.
- `images` — **omitted entirely** in v1, since the write surface doesn't accept them. Adding it later is additive.
- `indexedAt` — surfaced because it's the canonical chronological ordering key. Clients showing "X minutes ago" should prefer it over `createdAt` (which is client-declared and game-able, per the indexer spec's anti-backdating rationale).
- `author.avatarCid` — bare CID, not a URL. Image proxying is its own future spec.

#### List response

```json
{
  "items": [ PostResponse, ... ],
  "cursor": "eyJpbmRleGVkQXQiOiIyMDI2LTA1LTA0VDE4OjIzOjQ3WiIsInVyaSI6ImF0Oi8vLi4uIn0"
}
```

`cursor` is **omitted entirely** when there are no more pages — not `null`, not `""` — per the pagination spec.

#### Cursor encoding

```
base64url(JSON({ "indexedAt": "<RFC3339Nano>", "uri": "at://..." }))
```

Opaque to clients. Encoded/decoded inside `appview/internal/api/envelope/` so every paginated endpoint shares one implementation:

```go
envelope.EncodeCursor(v any) (string, error)
envelope.DecodeCursor(s string, out any) error
```

Bad input → `envelope.ErrInvalidCursor`, mapped by handlers to 400 `invalid_cursor`. Stability window: cursors stay valid as long as the referenced row exists; no artificial expiry in v1.

### 3. Data flow per endpoint

#### `POST /v1/posts`

1. Pull caller `did` from `middleware.GetDID(ctx)`.
2. `DecodePostCreate(r.Body)` → `*PostCreateRequest`.
   - Bad JSON → 400 `malformed_body`.
   - Wrong field types → 422 `validation_failed` with field map.
3. `ValidatePostCreate(req)`:
   - `text` non-empty, ≤ 2000 graphemes.
   - `reply.root.uri`, `reply.parent.uri`, `embed.quote.uri` — parse as `syntax.ATURI` if present.
   - `*.cid` — non-empty if present (no canonical-form check; CIDs use the same "informal helper" posture as elsewhere in the codebase).
   - Failure → 422 `validation_failed`, field map.
4. Build the lexicon record body:
   ```go
   body := map[string]any{
       "$type":     "social.craftsky.feed.post",
       "text":      req.Text,
       "createdAt": time.Now().UTC().Format(time.RFC3339),
   }
   if len(req.Facets) > 0 { body["facets"] = req.Facets }
   if req.Reply != nil   { body["reply"] = lexiconReply(req.Reply) }
   if req.Embed != nil && req.Embed.Quote != nil {
       body["embed"] = map[string]any{
           "$type": "social.craftsky.feed.post#quoteEmbed",
           "record": map[string]any{
               "uri": req.Embed.Quote.URI,
               "cid": req.Embed.Quote.CID,
           },
       }
   }
   ```
5. `pds := newPDS(ctx, did, sessionID); uri, cid, err := pds.CreateRecord(ctx, did, "social.craftsky.feed.post", body)`.
   - Connection error → 502 `pds_unavailable`.
   - PDS rejects (lexicon validation, etc.) → 502 `pds_write_failed`, message includes the PDS error name.
6. Hydrate author: `displayName` / `avatarCid` from a single SELECT against `bluesky_profiles` (LEFT JOIN-safe — both default to `null` if the row is missing); `handle` from `HandleResolver.ResolveHandle(ctx, did)`. Failure to resolve the handle → 502 `identity_unavailable`, matching the existing profile handler.
7. Build `PostResponse`:
   - `uri`, `cid` from PDS response.
   - `rkey` = last URL-segment of `uri` (TID stamped by PDS).
   - `tags` extracted from `req.Facets` using a shared helper `extractTagsFromFacets` (same logic as the indexer's `extractTags`, in a package both packages can import).
   - `indexedAt` = `now()` — small white lie (the row hasn't landed yet) but stable for the client's optimistic cache.
   - `author` from step 6.
8. 201 + JSON.

#### `GET /v1/posts/{did}/{rkey}`

1. Parse `{did}` as `syntax.DID`. Bad → 400 `invalid_identifier`.
2. `row, err := postStore.ReadOne(ctx, did, rkey)`. Returns the post's columns plus `display_name` / `avatar_cid` from the LEFT JOIN.
   - `ErrPostNotFound` → 404 `post_not_found`.
   - Other → 500 `internal_error`.
3. `handle, err := resolver.ResolveHandle(ctx, did)`. Failure → 502 `identity_unavailable`.
4. Build `PostResponse` from row + handle. 200 + JSON.

#### `DELETE /v1/posts/{did}/{rkey}`

1. Parse `{did}`. Bad → 400.
2. `caller, _ := middleware.GetDID(ctx)`. If `caller != did` → 403 `forbidden`.
3. `pds.DeleteRecord(ctx, did, "social.craftsky.feed.post", rkey)`.
   - Connection error → 502 `pds_unavailable`.
   - PDS returns "record not found" → swallow, treat as success (idempotent per [api architecture §2.3](2026-04-21-appview-api-architecture-design.md)).
4. 204, no body.

The local postgres row is **not** pre-emptively deleted. The firehose tombstone arrives within seconds and the indexer drops it. Brief read-after-delete inconsistency on the same instance for the same client; acceptable for v1, matches the asymmetry on the create side.

#### `GET /v1/profiles/{handleOrDid}/posts`

1. Strip `@` prefix from path; resolve handle → DID via the existing `resolveToDID(ctx, raw, resolver)` helper from `profile.go` (move to a shared file if test ergonomics suggest it; otherwise import).
2. Parse `limit` (default 50, max 100; `min(parsed, 100)` not 400-on-overshoot, per pagination spec) and `cursor` (opaque).
3. `rows, nextCursor, err := postStore.ListByAuthor(ctx, did, limit, cursor)`. Bad cursor → 400 `invalid_cursor`. Each row carries `display_name` and `avatar_cid` from the LEFT JOIN against `bluesky_profiles`; in this query they're identical across rows but joining per row is cheap and keeps the query simple.
4. `handle, err := resolver.ResolveHandle(ctx, did)`. Resolved once and zipped into every item's `author.handle` field. Failure → 502 `identity_unavailable`.
5. `{ items: [PostResponse...], cursor: "..." }`. Omit `cursor` field when there are no more pages.

### 4. Postgres queries

`craftsky_posts` columns are listed in [feed-post-indexing §Schema](2026-05-04-feed-post-indexing-design.md#schema). Two new queries belong in this spec, both in `post_store.go`:

`ReadOne` — single row by `(did, rkey)`:

```sql
SELECT p.uri, p.did, p.rkey, p.cid, p.text, p.facets,
       p.reply_root_uri, p.reply_root_cid, p.reply_parent_uri, p.reply_parent_cid,
       p.quote_uri, p.quote_cid, p.tags, p.created_at, p.indexed_at,
       bp.display_name, bp.avatar_cid
FROM craftsky_posts p
LEFT JOIN bluesky_profiles bp ON bp.did = p.did
WHERE p.did = $1 AND p.rkey = $2
```

`LEFT JOIN` on `bluesky_profiles` — a craftsky member without a Bluesky-profile mirror should still return their post (display name and avatar default to NULL, surfaced as `null` in the response). The post's existence implies craftsky membership (the indexer's FK enforces it), so no separate craftsky_profiles join is needed for filtering.

The author handle is **not** stored in any local table. Handlers fetch it via the existing `HandleResolver` (the same mechanism `GetProfileHandler` uses) after `ReadOne` returns. This keeps the store concerned only with what's actually in postgres and matches the pattern already in `profile.go`.

`ListByAuthor` — keyset pagination over `(indexed_at, uri)`:

```sql
SELECT p.uri, p.did, p.rkey, p.cid, p.text, p.facets,
       p.reply_root_uri, p.reply_root_cid, p.reply_parent_uri, p.reply_parent_cid,
       p.quote_uri, p.quote_cid, p.tags, p.created_at, p.indexed_at,
       bp.display_name, bp.avatar_cid
FROM craftsky_posts p
LEFT JOIN bluesky_profiles bp ON bp.did = p.did
WHERE p.did = $1
  AND ($2::timestamptz IS NULL OR (p.indexed_at, p.uri) < ($2, $3))
ORDER BY p.indexed_at DESC, p.uri DESC
LIMIT $4
```

The existing `craftsky_posts_did_indexed_at_desc` index covers the WHERE/ORDER BY.

`nextCursor` is the `(indexed_at, uri)` of the **last** row returned, encoded via `envelope.EncodeCursor`. Omit when fewer than `limit` rows came back.

### 5. `PDSClient` interface additions

[appview/internal/auth/pds_client.go](../../../appview/internal/auth/pds_client.go) gains:

```go
type PDSClient interface {
    GetRecord(ctx context.Context, repo syntax.DID, collection, rkey string, out any) (cid string, err error)
    PutRecord(ctx context.Context, repo syntax.DID, collection, rkey string, record any) error
    CreateRecord(ctx context.Context, repo syntax.DID, collection string, record any) (uri syntax.ATURI, cid syntax.CID, err error)
    DeleteRecord(ctx context.Context, repo syntax.DID, collection, rkey string) error
}
```

Implementations:

- **`IndigoPDSClient`** ([pds_client_indigo.go](../../../appview/internal/auth/pds_client_indigo.go)):
  - `CreateRecord` → POST `com.atproto.repo.createRecord` body `{repo, collection, record}`. Response carries `{uri, cid}`. Empty `uri` or `cid` → loud error (matches the existing posture in `GetRecord`).
  - `DeleteRecord` → POST `com.atproto.repo.deleteRecord` body `{repo, collection, rkey}`. "Record not found" gets translated via a sibling of `translateGetRecordError` so the handler can swallow it.
- **`AnonymousPDSClient`** ([anonymous_pds_client.go](../../../appview/internal/auth/anonymous_pds_client.go)) — both methods return a "not supported" sentinel, matching the existing `PutRecord` posture.
- **Test mocks** ([handlers_test.go](../../../appview/internal/auth/handlers_test.go), [initialize_profile_test.go](../../../appview/internal/auth/initialize_profile_test.go)) gain default no-op or pass-through implementations; existing tests keep passing.

### 6. File layout

Mirror the profile pattern.

```
appview/internal/api/
├── post.go                # handlers
├── post_request.go        # PostCreateRequest, DecodePostCreate, ValidatePostCreate
├── post_response.go       # PostResponse, BuildPostResponse
├── post_store.go          # PostReader interface + *PostStore impl, ErrPostNotFound
├── post_test.go
├── post_request_test.go
├── post_response_test.go
└── post_store_test.go
```

A new shared helper for tag extraction lives somewhere both `index/craftsky_post.go` and `api/post_request.go` can import. Two viable homes:

- New tiny package `appview/internal/lexicon/craftsky/facets/` (or similar) — exports `ExtractTags(facets []*appbsky.RichtextFacet) []string`. Indexer and handler both call it.
- Keep one canonical implementation in `index/` and re-import it from `api/`. The `api → index` direction risks a cycle if anything in `index/` ever imports `api/`; today nothing does, but the package boundary suggests the helper doesn't belong in `index/` long-term.

Recommend the new shared package. The implementer picks the exact name and confirms it doesn't collide with the existing `lexicon/craftsky/` codegen output.

### 7. Route registration

In [routes.go](../../../appview/internal/routes/routes.go), grouped after the existing profile routes:

```go
postStore := api.NewPostStore(deps.DB, deps.Logger)
mux.Handle("POST   /v1/posts",
    authN(deviceID(api.CreatePostHandler(postStore, deps.NewPDSClient, deps.Logger))))
mux.Handle("GET    /v1/posts/{did}/{rkey}",
    authN(deviceID(api.GetPostHandler(postStore, deps.Logger))))
mux.Handle("DELETE /v1/posts/{did}/{rkey}",
    authN(deviceID(api.DeletePostHandler(deps.NewPDSClient, deps.Logger))))
mux.Handle("GET    /v1/profiles/{handleOrDid}/posts",
    authN(deviceID(api.ListPostsByAuthorHandler(postStore, deps.HandleResolver, deps.Logger))))
```

`PostStore` is constructed once on AppView startup and closed over by all four handlers — same pattern as `ProfileStore`.

### 8. Error envelope

Every error response uses the project envelope ([api architecture §6](2026-04-21-appview-api-architecture-design.md)). Codes specific to this surface:

| Code | Status | When |
|---|---|---|
| `malformed_body` | 400 | JSON parse fail or wrong field types |
| `invalid_identifier` | 400 | `{did}` path segment doesn't parse |
| `invalid_cursor` | 400 | `cursor` query param doesn't decode |
| `validation_failed` | 422 | text length, AT-URI parse, etc. — includes `fields` map |
| `forbidden` | 403 | `DELETE` on someone else's post |
| `post_not_found` | 404 | `GET /v1/posts/{did}/{rkey}` and no row |
| `pds_unavailable` | 502 | can't reach the user's PDS |
| `pds_write_failed` | 502 | PDS rejected the create (e.g. lexicon validation) |
| `internal_error` | 500 | unexpected store failure |

`requestId` is the existing `runID` from `middleware.GetRunID(ctx)`.

### 9. Tests

Real Postgres via `testdb.WithSchema(t, ddl)`; PDS mocked via the existing `mockPDS` pattern.

**`post_store_test.go`** — store layer:

- `ReadOne_Found_HydratesAuthor` — happy path, all fields populated from join.
- `ReadOne_FoundNoBlueskyMirror_NullDisplayFields` — `LEFT JOIN` keeps the row.
- `ReadOne_NotFound_ReturnsErrPostNotFound`.
- `ListByAuthor_OrdersByIndexedAtDesc`.
- `ListByAuthor_RespectsLimit`.
- `ListByAuthor_CursorContinues` — second page picks up where first left off; tied `indexed_at` resolved by `uri DESC`.
- `ListByAuthor_EmptyForUnknownAuthor` — returns `([], "", nil)`.
- `ListByAuthor_InvalidCursor_ReturnsError`.

**`post_test.go`** — handlers:

- `Create_HappyPath_201_SyntheticPost`.
- `Create_MalformedBody_400`.
- `Create_TextEmpty_422`.
- `Create_TextTooLongGraphemes_422_FieldMap`.
- `Create_PDSDown_502_PdsUnavailable`.
- `Create_PDSReject_502_PdsWriteFailed`.
- `Create_WithReply_PassesThroughToPDS` — assert PDS receives the lexicon-shaped body.
- `Create_WithQuote_PassesThroughToPDS`.
- `Create_TagsExtractedFromFacets_ReturnedInResponse`.
- `Get_Found_ReturnsHydratedPost`.
- `Get_NotFound_404`.
- `Get_BadDID_400`.
- `Delete_Self_204_CallsPDS`.
- `Delete_OtherUser_403_NoPDSCall`.
- `Delete_RecordAlreadyGone_204_Idempotent`.
- `Delete_PDSDown_502`.
- `List_HappyPath_PaginatesCorrectly`.
- `List_HandleResolution_Works`.
- `List_BadCursor_400`.

**`post_request_test.go`** / **`post_response_test.go`** — pure functions:

- Request decode field-error map shape on bad bodies.
- `BuildPostResponse` correctness on every column-presence combination (reply present/absent, quote present/absent, facets present/absent, tags empty).

Coverage target: every branch in every handler, every field of `PostResponse`, every error code in §8.

### 10. Wiring changes

- New file: `appview/internal/api/post.go` and siblings.
- Modified: `appview/internal/auth/pds_client.go` — interface gains two methods.
- Modified: `appview/internal/auth/pds_client_indigo.go` — implements them.
- Modified: `appview/internal/auth/anonymous_pds_client.go` — returns "not supported" for both.
- Modified: test mocks in `appview/internal/auth/*_test.go` — implement the two new methods (default no-op).
- Modified: `appview/internal/routes/routes.go` — registers four new routes.
- New (probably): a small shared helper package for facet → tag extraction so `index/` and `api/` share one implementation.

No schema changes. No changes to Tap, dispatcher, the consumer, the firehose ingestion side, or auth/OAuth flow.

## Alternatives considered

### Server-generated TID + existing `PutRecord` instead of adding `CreateRecord`

The post lexicon's `key: tid` could be satisfied by generating a TID server-side and calling the existing `PutRecord(did, nsid, rkey, body)`. Zero new interface surface.

**Rejected** because:

- Two AppView replicas could in principle produce the same TID at the microsecond. Vanishingly unlikely but the failure mode is a silent overwrite of someone else's post.
- We need `DeleteRecord` regardless. Adding both at once for symmetry is cleaner than mixing patterns (create-via-PutRecord, delete-via-new-method).
- Likes, follows, reposts, project posts will all use `CreateRecord`. Paying the interface cost once now is cheaper than retrofitting every record-write later.

### Wait for the indexer instead of a synthetic create response

`POST /v1/posts` could poll postgres for ~5s after the PDS write and return the indexed row.

**Rejected** because:

- Adds latency on the happy path.
- Blocks a goroutine on a sync primitive for every create.
- Any timeout becomes a "did it work?" question for the client; the synthetic response sidesteps this by always returning the URI/CID the PDS confirmed.
- The existing `PUT /v1/profiles/me` already uses the synthetic-row pattern ([profile.go:225](../../../appview/internal/api/profile.go:225)); same lesson applies here.

### Pre-emptive postgres delete on `DELETE /v1/posts/...`

The handler could `DELETE FROM craftsky_posts WHERE uri = $1` immediately after the PDS delete succeeds, instead of waiting for the firehose tombstone.

**Rejected** for v1: it'd save a few seconds of read-after-delete inconsistency on the same replica, but introduces a divergence between the firehose (source of truth) and the local DB if the PDS delete succeeded in this request but the firehose event drops on the floor. Easier to keep the AppView's local state strictly downstream of the firehose. If real users complain about the lag, revisit.

### Validating reply/quote target existence at write time

The handler could `SELECT 1 FROM craftsky_posts WHERE uri = $1` before accepting a reply or quote.

**Rejected** because:

- A user can validly reply to a Bluesky post (lives in `bluesky_profiles`/no local table), a deleted post, a post from a non-Craftsky member (not gated into our DB), or a future-PDS post we haven't seen yet. Rejecting these is wrong.
- The indexer spec explicitly takes the "store URI/CID verbatim, let the read endpoint render 'not indexed' how it likes" posture. The write side adopts the same posture for symmetry.

### Author hydration via per-row JOIN on list-by-author

`ListByAuthor` could `JOIN craftsky_profiles JOIN bluesky_profiles` per row.

**Rejected** because every row in this query has the same `did` — the join would replicate identical author columns across every result. Fetching once and zipping is cheaper and clearer.

### Using `app.bsky.feed.post` instead of `social.craftsky.feed.post`

We could write Bluesky-shaped posts and inherit Bluesky's existing ecosystem.

**Rejected** by prior decision — the project has its own lexicon for project-shaped posts. Moot for this spec; flagged here only because a contributor may wonder.

## Consequences

### Code changes required

- New: `appview/internal/api/post.go`, `post_request.go`, `post_response.go`, `post_store.go` and tests.
- New: small shared helper for facet → tag extraction.
- Modified: `appview/internal/auth/pds_client.go` (interface), `pds_client_indigo.go` (impl), `anonymous_pds_client.go` (read-only stub), test mocks.
- Modified: `appview/internal/routes/routes.go`.

No lexicon, generated-types, dispatcher, Tap, or migration changes.

### Migration path

None — the `craftsky_posts` table already exists.

### Performance and storage

- `POST /v1/posts`: one PDS round-trip + one DB read for author. Tag extraction is a stack-allocated walk; not on a hot path.
- `GET /v1/posts/{did}/{rkey}`: one DB query (single-row PK lookup with two joins). Sub-millisecond.
- `DELETE /v1/posts/{did}/{rkey}`: one PDS round-trip; no DB touch. PDS latency dominates.
- `GET /v1/profiles/{handleOrDid}/posts`: one cursor query (covered by `craftsky_posts_did_indexed_at_desc`) + one author lookup (covered by PK on `craftsky_profiles`). At limit 100 with a 25 KB-worst-case post and full author hydration, response payload stays under ~3 MB worst case.
- Cursor encoding/decoding: trivial JSON round-trip.

### Risks

- **Read-after-write delay.** Window between `POST /v1/posts` returning and the indexed row appearing is bounded by Tap latency (~seconds). The synthetic response covers the optimistic-cache case for the same client; cross-client visibility of a brand-new post is delayed by that window. Acceptable for v1.
- **Read-after-delete delay.** Same window in the other direction — a deleted post stays visible to other clients until the firehose tombstone arrives. Same acceptance.
- **Project posts round-trip via `record JSONB` only.** Until the project-fields materialisation pass lands, a project post written through this surface (which doesn't accept `project` in the request) cannot exist. A project post arriving on the firehose from a different client is preserved in `record` but the `PostResponse` does not surface project fields — clients see it as a plain post. Documented above.
- **`CreateRecord` failure modes.** PDS-side lexicon validation errors come back as XRPC errors; we map them generically to `pds_write_failed`. A finer mapping (per-error-name → per-code) is possible later; the current grouping is good enough for v1.
- **Tag extraction divergence.** The handler extracts tags from `req.Facets` for the synthetic response; the indexer extracts tags from `record.facets` when the firehose event arrives. Both should produce the same array. The shared helper enforces this; the test suite explicitly checks that the synthetic response's `tags` matches what the indexer would produce for the same body.

## Open questions

None at time of writing. Resolved during brainstorming:

- **Endpoint scope:** create + read-single + delete + list-by-author. No thread, no global feed, no likes, no images.
- **Write surface:** `text`, `facets`, `reply`, `embed.quoteEmbed` only. No `project`, no `images`.
- **Author hydration:** `{did, handle, displayName, avatarCid}` on every post response.
- **rkey generation:** add `CreateRecord` to `PDSClient`.
- **Read-after-write semantics:** synthetic response on create; firehose-driven row on subsequent reads.
- **Reply/quote target validation:** pass-through, no existence check.
- **`createdAt`:** server-stamped; client override is additive future work.
