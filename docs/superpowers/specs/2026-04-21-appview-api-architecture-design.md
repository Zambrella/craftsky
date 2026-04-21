# AppView API Architecture (v1) — design

**Date:** 2026-04-21
**Status:** proposed
**Scope:** Shape of the HTTP API between the Flutter client and the AppView. URL conventions, versioning, auth headers, error envelope, pagination, and the v1 endpoint surface.

## Summary

The Flutter app talks to the AppView over HTTP + JSON using a **client-shaped REST API** (not atproto XRPC). Every Craftsky endpoint lives under a **single `/v1/` prefix** that gets bumped as a whole when a breaking change lands anywhere in the surface. Authentication is `Authorization: Bearer <craftsky-token>` plus an `X-Craftsky-Device-Id` header. Errors use a `{error, message, requestId}` envelope. Lists paginate via opaque cursors.

This spec fixes the *shape* of the API for v1. Individual endpoint bodies (request/response fields) are described here at summary level — their exact JSON schemas are defined alongside the handlers that implement them, in follow-up specs per feature area (feed, profiles, posts, notifications).

## Goals

1. Fix the URL conventions, method semantics, headers, error shape, and pagination style so every handler in v1 (and every follow-up spec) starts from the same baseline.
2. Enumerate the v1 endpoint surface — the minimum set the Flutter app needs to ship a first usable version.
3. Lock in a versioning strategy that's cheap now and won't paint us into a corner later.
4. Leave explicit follow-up items for things deliberately out of scope (rate limiting, CORS, observability, etc.) so nothing rots silently.

## Non-goals

- **XRPC surface.** We are not exposing `/xrpc/social.craftsky.*` endpoints for third-party atproto clients. If that becomes a goal later, it's additive — it doesn't change this spec.
- **Full request/response JSON schemas.** Each endpoint's body shape belongs to its feature spec. This one fixes the *envelope* (error shape, pagination shape, auth headers) and the URL surface.
- **Write-proxy implementation details.** `POST /v1/posts` etc. require a DPoP-signing path to the user's PDS; that mechanism was left as future work in the OAuth BFF spec and still is. This spec commits to the URL shape of those endpoints; the implementation lands alongside the first write-proxy spec.
- **Blob upload.** Deferred per user decision — v1 will eventually need it (for images on posts) but not for this spec. Own spec.
- **Rate limiting, CORS, observability, body-size limits, success-response envelope shape.** All tracked in the roadmap doc; each is its own spec if/when addressed.
- **atproto-layer concerns.** OAuth, DPoP, token storage, and the AppView→PDS leg are all owned by [`2026-04-18-appview-oauth-bff-design.md`](./2026-04-18-appview-oauth-bff-design.md) and its successors. This spec sits *above* that layer and assumes it.

## 1. Why REST and not XRPC

atproto's native HTTP convention is XRPC: every endpoint is `GET|POST /xrpc/<NSID>`, described by a lexicon schema. Bluesky's AppView exposes ~150 XRPC endpoints under `app.bsky.*`.

We considered XRPC and rejected it for v1:

- The Flutter app is the only client we plan to support in v1. The main XRPC argument — "third-party atproto clients can read your data out of the box" — doesn't pay off until such clients exist.
- REST is more idiomatic for the typical Flutter/web contributor. Lower onboarding cost for the open-source contributor pool the project is targeting (per AGENTS.md's stated rationale for picking Go over Serverpod).
- REST URL conventions (resourceful paths, caching, OpenAPI tooling, reverse-proxy friendliness) fit HTTP's grain better than XRPC's "everything is a verb" model.
- XRPC commits us to writing *query lexicons* alongside our record lexicons — same "load-bearing, painful to change" property, doubled. Record lexicons (`social.craftsky.feed.post`) stay atproto-native; the API surface does not need to.

**What stays XRPC:**

- `/oauth/*` endpoints from the BFF spec (those addresses are contracts with external systems — PDS Authorization Server, OAuth spec — not with our Flutter app). Unchanged.
- The AppView → PDS leg internally. PDSes only speak XRPC, so the write-proxy handlers call `com.atproto.repo.createRecord` etc. downstream regardless of what the Flutter-facing API looks like.
- Any future pass-through route we expose publicly for atproto interop (e.g. `POST /xrpc/com.atproto.repo.createRecord` as an explicit pass-through). Not in v1; noted for completeness.

**Reversibility:** adding an XRPC surface alongside REST later is additive — same handlers can be reached from two URL shapes. So "XRPC for atproto interop" is a future-work option, not a door we're closing.

## 2. URL conventions

### 2.1 Version prefix

Every Craftsky endpoint lives under `/v1/`. When a breaking change lands **anywhere** in the API, the whole surface bumps to `/v2/` at once. Individual endpoints are not versioned independently.

What's **not** under `/v1/`:

- `/oauth/client-metadata.json`, `/oauth/jwks.json`, `/oauth/callback` — contracts with the Authorization Server, governed by the atproto OAuth spec. Stay where they are.
- `/health`, `/healthz` — ops surface, not app surface. Stay where they are.

What **moves** under `/v1/` as part of this spec:

- `POST /auth/login` → `POST /v1/auth/login`
- `POST /auth/logout` → `POST /v1/auth/logout`
- `GET /whoami` → `GET /v1/whoami`

This is a breaking change to the OAuth BFF surface, but the BFF shipped recently and the Flutter app is still in scaffold — the migration cost is effectively zero. Better to pay it now than to have a mixed-versioning world.

**When do we bump to `/v2/`?** Only on *breaking* changes: removing a field, renaming a route, changing a type, changing error semantics. Additive changes (new optional field in response, new optional query param, new endpoint) stay at the current version. Deprecation window: both `/v1/` and `/v2/` served side-by-side until the oldest Flutter client in the wild is known to be off `/v1/`, then `/v1/` is deleted.

### 2.2 Resource identifiers

atproto records are identified by AT-URIs (`at://did:plc:xyz/social.craftsky.feed.post/abc`); accounts are identified by DIDs. Both contain characters (colons, slashes) that are fiddly in URL paths.

**Rule:** accept both handles and DIDs as inputs on URL paths for user-facing resources; always return DIDs (and AT-URIs) in response bodies as the canonical identifier. This matches what `bsky.app` does.

Concretely:

- **Handle-or-DID input:** `GET /v1/profiles/@alice.craftsky.social` and `GET /v1/profiles/@did:plc:xyz` both resolve the same profile. The `@` prefix is part of the path segment; it disambiguates from other path segment styles and matches Bluesky's URL patterns. Handle resolution happens server-side; DID is the canonical form internally.
- **Post paths:** `GET /v1/posts/{did}/{rkey}` — we don't accept handles here for two reasons: (a) AT-URIs inside post bodies already use DIDs so clients are passing DIDs through, not fresh handle lookups; (b) keeping handle resolution to the top-level profile routes limits the blast radius of handle-resolution latency/errors.
- **Response bodies always use DIDs.** A post object returns `{"author": {"did": "did:plc:xyz", "handle": "alice.craftsky.social", ...}, ...}`. The client stores the DID as the stable reference and uses the handle for display.
- **`me`** is accepted as a synonym for the authenticated user's DID on endpoints where it makes sense: `GET /v1/profiles/me`, `PATCH /v1/profiles/me`. The `me` form is preferred for self-referential routes because it doesn't embed an identity the client has to look up.

### 2.3 HTTP methods

Standard REST semantics:

- `GET` — read; idempotent; no body.
- `POST` — create; returns the created resource.
- `PATCH` — partial update; body contains only fields to change.
- `PUT` — full replace; body contains the complete resource; missing fields are cleared.
- `DELETE` — delete; idempotent (deleting an already-deleted resource returns 204, not 404).

`PATCH` and `PUT` both apply to `/v1/profiles/me` — see §4.

### 2.4 Query parameters

- Pagination: `limit` (integer) and `cursor` (opaque string). See §5.
- Filter/scope params are specific to each endpoint and defined in per-endpoint specs.
- No path params in the query string; no query params in the path. Clean separation.

## 3. Authentication

### 3.1 Headers on authenticated requests

Every authenticated request carries two headers:

```
Authorization: Bearer <craftsky-session-token>
X-Craftsky-Device-Id: <client-generated-device-id>
```

- **`Authorization: Bearer`** — the opaque Craftsky session token defined in the OAuth BFF spec. The existing `Authenticated` middleware already resolves this.
- **`X-Craftsky-Device-Id`** — a stable client-generated UUID. Generated once at first launch, persisted in the client's secure storage, sent on every authenticated request thereafter. The AppView logs it and stores the most recently seen value on the `craftsky_sessions` row (schema addition below). In v1 it has no behavioural effect — it's instrumentation and a hook for future features (active-sessions UI, per-device rate limits, push-token routing).

**Why require it even though v1 doesn't use it?** Retrofitting a required header later forces a coordinated Flutter + AppView release to avoid 400s. Requiring it from day one means the data is there whenever a future feature needs it, with zero coordination cost.

**Missing or malformed `X-Craftsky-Device-Id`:** 400 `{"error": "missing_device_id", ...}`. Not 401 — the token is valid; the request envelope isn't.

### 3.2 Unauthenticated endpoints

- `/health`, `/healthz` — ops.
- `/oauth/*` — per OAuth BFF spec.
- `POST /v1/auth/login` — by definition pre-auth.

Every other endpoint under `/v1/` requires authentication.

### 3.3 Schema addition

Add a column to `craftsky_sessions` (from the OAuth BFF spec):

```sql
ALTER TABLE craftsky_sessions
  ADD COLUMN last_device_id TEXT;
```

Updated opportunistically alongside `last_seen_at` (same throttle window — default 5m). No index in v1; add when the active-sessions UI lands.

## 4. The v1 endpoint surface

Grouped by resource. Each line is an endpoint; response/request body details belong to per-feature specs.

### 4.1 Feed

- `GET /v1/feed/timeline` — chronological feed of followed accounts, newest first.

### 4.2 Profiles

- `GET /v1/profiles/@{handleOrDid}` — profile summary (display name, avatar, bio, Craftsky-specific fields, counts).
- `GET /v1/profiles/@{handleOrDid}/posts` — a user's own posts, newest first.
- `PATCH /v1/profiles/me` — partial profile update. Body contains only the fields to change.
- `PUT /v1/profiles/me` — full profile replace. Body must contain the complete profile; missing fields are cleared.
- `POST /v1/profiles/@{handleOrDid}/follows` — follow this profile. Writes an `app.bsky.graph.follow` record to the caller's PDS.
- `DELETE /v1/profiles/@{handleOrDid}/follows` — unfollow.

**Why one profile endpoint for two lexicons:** the Flutter app thinks of "a profile" as a single unit. The AppView splits the body into `app.bsky.actor.profile` (display name, avatar, bio) and `social.craftsky.actor.profile` (craft-specific fields) writes to the caller's PDS. Client sends one body; AppView does two PDS writes.

**Partial-success handling on `PATCH`/`PUT`:**

- **Both writes succeed** → `200` with the updated profile.
- **Both fail** → `502 {"error": "pds_write_failed", "message": "..."}`. No records changed. Safe to retry.
- **Only one succeeds** → `502 {"error": "pds_write_partial", "message": "...", "fields": {"bsky": "ok", "craftsky": "failed"}}`. Client should retry — PDS `putRecord` is idempotent (same body → same CID → no-op on replay), so retry converges. The AppView does **not** attempt a rollback; the firehose has already emitted the successful write and there's no atomic "undo as if it never happened" available.

### 4.3 Posts

- `GET /v1/posts/{did}/{rkey}` — single post (for deep links).
- `GET /v1/posts/{did}/{rkey}/thread` — the post plus its replies tree, rooted at this post.
- `POST /v1/posts` — create a `social.craftsky.feed.post`. Body is the post fields; AppView writes to the caller's PDS.
- `DELETE /v1/posts/{rkey}` — delete your own post. `{did}` is implicit (the authenticated user); using `/{rkey}` alone prevents "delete someone else's post" from even being expressible in the URL.
- `POST /v1/posts/{did}/{rkey}/likes` — like this post. Writes a `social.craftsky.feed.like` record to the caller's PDS. Idempotent — liking an already-liked post returns the existing like.
- `DELETE /v1/posts/{did}/{rkey}/likes` — unlike. Idempotent.

### 4.4 Notifications

- `GET /v1/notifications` — likes, follows, replies, reposts directed at the authenticated user.

### 4.5 Auth (moved under `/v1/`)

- `POST /v1/auth/login` — per OAuth BFF spec.
- `POST /v1/auth/logout` — per OAuth BFF spec. Continues to accept `?all=true`.
- `GET /v1/whoami` — returns the authenticated user's DID + handle.

### 4.6 Out of scope for v1 (explicitly)

- Search
- Reposts (lexicon exists; UX not scoped)
- Blocks, mutes, reports
- Blob upload (deferred — needed for image posts but not in this spec)
- Push notification registration

## 5. Pagination

List endpoints (feed, profile posts, notifications, etc.) paginate with **opaque cursors**.

**Request:**

```
GET /v1/feed/timeline?limit=50&cursor=eyJhZnRlciI6MTcxMzUwMDAwMH0
```

- `limit` — integer, bounded. Default 50, max 100 (per-endpoint specs can tighten, not loosen).
- `cursor` — opaque base64url string. Absent/empty means "from the start."

**Response:**

```json
{
  "items": [ ... ],
  "cursor": "eyJhZnRlciI6MTcxMzQ5OTk5OX0"
}
```

- `items` — the page.
- `cursor` — next-page token. **Absent entirely** when there are no more items. (Not `null`, not `""` — omitted.)

**Cursor contents are implementation-defined and opaque.** Clients must not inspect, parse, or construct cursors; they round-trip whatever the server gave them. The AppView is free to change the encoding (base64-encoded JSON today, something else tomorrow) without breaking clients.

**Stability:** cursors remain valid indefinitely at the spec level. Implementations may invalidate cursors older than some window (e.g. 7 days); when that happens, the endpoint returns `400 {"error": "invalid_cursor", ...}` and the client restarts from no cursor. Per-endpoint specs define their stability window.

**Why not offset/page/timestamp:** insertions are common on social feeds; offset and page-based pagination both skip or re-show items when new posts arrive. Timestamp-based pagination works for strict reverse-chronological lists but leaks the ordering key and doesn't generalise. Opaque cursors let the server pick the right strategy per endpoint without the client knowing.

## 6. Error envelope

Every error response (any 4xx or 5xx) uses the same JSON body shape:

```json
{
  "error": "snake_case_code",
  "message": "Human-readable description.",
  "requestId": "01HQ8V9XK7F5Q0W8ZRXGJ6N1YT"
}
```

- **`error`** — machine-readable snake_case code. Enumerated per endpoint in its feature spec. Clients branch on this, not on the HTTP status.
- **`message`** — human-readable text. Not localised; not intended to be surfaced to end users verbatim. Useful for logs and developer debugging.
- **`requestId`** — server-generated ID for correlating to logs. ULID (or similar). Always present, even when the client didn't send one.

**Validation errors with per-field detail** add a `fields` sibling:

```json
{
  "error": "validation_failed",
  "message": "One or more fields are invalid.",
  "fields": {
    "text": "exceeds max length of 3000 characters",
    "project.materials": "unknown material code: flgg"
  },
  "requestId": "01HQ..."
}
```

**HTTP status codes** follow the obvious mapping:

- `400` — malformed request (bad JSON, missing required field, invalid params).
- `401` — authentication required or failed.
- `403` — authenticated but not permitted (e.g. delete someone else's post).
- `404` — resource doesn't exist or isn't visible to caller.
- `409` — conflict (e.g. duplicate handle).
- `422` — semantically invalid (`validation_failed` with `fields`).
- `429` — rate-limited. (Rate limiting itself is future work; status code is reserved.)
- `500` — unexpected server error.
- `502` — downstream dependency (PDS, Authorization Server) failed.
- `503` — temporary, retryable (migrations running, overloaded).

**Error codes** are strings, not integers — consistency with the OAuth BFF spec's existing errors (`handle_not_found`, `authorization_server_unavailable`, etc.). All codes for a given endpoint are enumerated in that endpoint's feature spec; there is no global registry but codes should be unique across the surface.

## 7. Implementation map

### 7.1 Code layout

No new top-level packages. Route registration stays in `appview/internal/routes/routes.go`. Handlers stay in `appview/internal/api/` grouped by feature area (new files as feature specs land: `feed.go`, `profile.go`, `post.go`, `notification.go`).

Cross-cutting concerns get small shared packages:

- `appview/internal/api/envelope/` — error-response builder (`WriteError(w, status, code, message, fields)`), cursor helpers (encode/decode opaque cursors), request-ID generation.
- `appview/internal/middleware/` — new middleware for `X-Craftsky-Device-Id` validation (rejects missing/malformed with 400; passes value into context for handlers to record on `craftsky_sessions`).

Existing `Authenticated` middleware is unchanged. The device-id middleware composes on top of it.

### 7.2 Route registration

In `routes.go`, the existing handlers move under `/v1/`:

```go
// before
mux.Handle("GET /whoami", authN(api.WhoAmIHandler()))

// after
mux.Handle("GET /v1/whoami", authN(deviceId(api.WhoAmIHandler())))
```

where `deviceId` is the new middleware. Health and OAuth routes stay at their existing paths.

New v1 routes are added route-by-route as feature specs land. This spec does **not** register them — it only fixes the shapes.

### 7.3 Migration

One migration for the schema addition:

```
appview/migrations/000003_craftsky_sessions_device_id.up.sql
appview/migrations/000003_craftsky_sessions_device_id.down.sql
```

Adds the `last_device_id` column.

### 7.4 AGENTS.md updates

Add a short "API Conventions" section pointing at this spec. Contributors shipping new endpoints should read it before picking a URL shape or error code.

### 7.5 Testing posture

- **Envelope helpers** unit-tested in `envelope/` package.
- **Device-id middleware** unit-tested against the standard middleware test pattern.
- **Endpoint-level tests** live in their feature specs — not this one.
- **Contract test:** one test that walks a small set of representative routes and confirms they all return the standard error envelope on failure. Prevents envelope drift as new handlers land.

## 8. Open questions flagged, not resolved

1. **Success response envelope.** Do successful responses get a `{"data": ...}` wrapper, or are they returned bare? This spec **does not decide** — v1 proceeds with bare bodies (matches what's already built, and REST convention leans bare). If we add a wrapper later it's a breaking change → `/v2/`. Flagged in the roadmap.
2. **Web client / CORS.** No web client in v1, so CORS stays disallowed. If a web client is ever scoped, CORS policy becomes its own spec. Flagged in the roadmap.
3. **OpenAPI.** A hand-written OpenAPI doc would let us generate Dart client code and formal API docs. Deferred — probably worth doing around the time the API stabilises. Flagged in the roadmap.
4. **Per-endpoint body schemas.** Each feature spec defines its own request/response JSON. This spec does not enumerate them. A future "API reference" doc (potentially generated from OpenAPI) would consolidate.

## 9. Future work

Explicitly out of scope for v1, listed here so it's discoverable. Also tracked in `docs/roadmap.md`.

1. **Rate limiting.** Per-token and per-device-id. Future spec.
2. **CORS policy.** If/when a web client is in scope.
3. **Request body size limits.** Per-endpoint; probably a middleware sitting in front of handlers.
4. **Success response envelope.** See §8.1.
5. **Observability.** Request logging format, request-ID propagation into downstream calls, metrics, tracing. Probably its own cross-cutting spec.
6. **Search endpoints.** `GET /v1/search/posts`, etc. Own spec.
7. **Repost endpoints.** Lexicon exists; UX and endpoint shape not scoped.
8. **Blocks, mutes, reports.** Moderation spec.
9. **Blob upload.** Needed for images on posts; likely triggers TMB upgrade per OAuth BFF spec §6.
10. **Push notification registration.** `POST /v1/notifications/devices`, `DELETE /v1/notifications/devices/{id}`. Own spec.
11. **Active-sessions UI endpoints.** `GET /v1/auth/sessions`, `DELETE /v1/auth/sessions/{id}`. Depends on Flutter profile/settings screen. Uses `last_device_id` / `device_label` data we're already collecting.
12. **OpenAPI / typed client generation.** See §8.3.
13. **XRPC interop surface.** `/xrpc/social.craftsky.*` for third-party atproto clients. Additive — doesn't break this spec.
