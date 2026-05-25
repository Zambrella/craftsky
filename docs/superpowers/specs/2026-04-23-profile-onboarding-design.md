# Profile Onboarding & Endpoints — design

**Date:** 2026-04-23
**Status:** proposed
**Scope:** End user completes onboarding and creates/updates their Craftsky profile. Covers the `social.craftsky.actor.profile` indexing path, onboarding-on-login semantics in the OAuth callback, and the three `/v1/profiles/*` HTTP endpoints. Flutter app changes are explicitly out of scope.

## Summary

A Craftsky profile is split across two atproto records on the user's PDS: `app.bsky.actor.profile` (display name, bio, avatar, banner) and `social.craftsky.actor.profile` (craft list, with room to grow later). Onboarding is not a separate flow: during the OAuth callback, the AppView initialises both records if they don't exist and reads them if they do, so every logged-in user has both a Bluesky-profile row and a Craftsky-profile row in Postgres by the time the callback returns. The Flutter app then treats "onboarded" as a client-side UX state (`crafts.length > 0`) rather than a server-side flag.

Reads and writes flow through three authenticated v1 endpoints: `GET /v1/profiles/@{handleOrDid}`, `GET /v1/profiles/me`, and `PUT /v1/profiles/me`. The AppView writes both records in parallel on PUT, returns a combined representation on GET, and drops the throwaway `bluesky_posts_sample` indexer now that the first real `social.craftsky.*` indexer is landing.

## Goals

1. Ship the minimum server surface required to onboard a user and let them manage their profile, without touching the Flutter app.
2. Confirm the existing `social.craftsky.actor.profile` lexicon is sufficient for v1 — no lexicon changes.
3. Establish the pattern for Craftsky indexers beyond the throwaway `bluesky_posts_sample`: one indexer per NSID, registered on the dispatcher, idempotent on `(URI, CID)`.
4. Make "logged in → profile exists" an invariant from the server's perspective, so the client UX doesn't need to handle a "no profile yet" case between OAuth and the first save.
5. Fix the `GET /v1/profiles/...` response shape for v1 so future additive work doesn't break clients.

## Non-goals

- **Flutter app changes.** Out of scope per user direction. This spec stops at the HTTP wire.
- **Historical note: avatar/banner blob upload.** This was deferred when the profile endpoints first landed. The follow-up image-upload implementation keeps avatar/banner on `app.bsky.actor.profile`, reuses `POST /v1/blobs/images`, and allows `PUT /v1/profiles/me` to carry uploaded `avatar` and `banner` blob objects.
- **`PATCH /v1/profiles/me`.** v1 is PUT-only. Adding PATCH later is additive.
- **`*.craftsky.social` handles.** Would require operating a handle service — separate project.
- **Counts** (followers, following, posts) on profile responses. Require indexers that don't exist yet.
- **Viewer-relationship fields** (is-viewer-following, is-blocked) on profile responses. Require graph indexing.
- **Skip optimization on PUT.** Each PUT writes both PDS records even if one side is unchanged. Optimisation is future work.
- **App-layer blob proxying.** Avatar/banner URLs point at Bluesky's public CDN. Own-CDN/proxying is future work.
- **CLI profile commands.** HTTP endpoints only.

## 1. Lexicon

No lexicon changes. The existing `social.craftsky.actor.profile` — literal `self` record key, `crafts: array<string>` with open `knownValues` sourced from `social.craftsky.feed.defs` — is the v1 shape. All Bluesky-side fields (display name, bio, avatar, banner) stay on `app.bsky.actor.profile`; we do not mirror them.

This spec therefore does not need an ADR under AGENTS.md rule #4 (ADRs gate *changes* to lexicon).

Future additions to `social.craftsky.actor.profile` — per-craft skill levels, external links (Ravelry, Etsy), pronouns, location — are all additive under atproto evolution rules and can be deferred until real usage motivates them.

## 2. Data model

Two new tables. Both live in the `public` schema. DID is the primary key on both — a user has exactly one of each record on their PDS.

### 2.1 `craftsky_profiles`

```sql
CREATE TABLE craftsky_profiles (
    did         TEXT        NOT NULL PRIMARY KEY,
    crafts      TEXT[]      NOT NULL DEFAULT '{}',
    record_cid  TEXT        NOT NULL,
    indexed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

- `crafts` — array from the record. Empty for new users (the login initialiser writes an empty record). A Postgres array (rather than JSON) so future filter/search queries are straightforward.
- `record_cid` — CID of the last-indexed record. Used as an idempotency guard against Tap redelivery: an event with an already-indexed CID is a no-op.
- `indexed_at` — updated on every upsert that changes `record_cid`.
- `created_at` — set on initial insert only. Used as the `createdAt` field in GET responses ("when the user onboarded"). Unlike `indexed_at`, it is not touched by later upserts.

### 2.2 `bluesky_profiles`

```sql
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
```

- All fields except `did`, `record_cid`, and `indexed_at` are nullable — each is optional on the Bluesky side.
- Avatar/banner blob refs are split into `_cid`/`_mime` pairs so queries don't need to parse JSON to filter on presence.
- No `created_at` — a row here has no stable "first seen by Craftsky" meaning because it may be written by login backfill or by firehose, depending on which arrives first.

### 2.3 What is deliberately **not** in the schema

- **No explicit `craftsky_members` table.** Membership is implicit: "DID exists in `craftsky_profiles`." The `bluesky_profiles` indexer checks membership via a PK lookup before writing.
- **No history/audit tables.** The PDS repo is history; we index the current state.
- **No follower/post counts.** Future specs.
- **No computed-state columns** (e.g. `is_onboarded`). The client derives onboarding state from `crafts.length > 0`.

### 2.4 Migrations

Numbered relative to the current head (`000006`). The implementer verifies the head at implementation time and renumbers if new migrations landed in between.

- `000007_drop_bluesky_posts_sample.up.sql` / `.down.sql` — drops the `bluesky_posts_sample` table; up symmetrical with migration `000001`.
- `000008_craftsky_profiles.up.sql` / `.down.sql` — creates `craftsky_profiles`.
- `000009_bluesky_profiles.up.sql` / `.down.sql` — creates `bluesky_profiles`.

No indexes beyond the primary key on either table in v1. Handle-based reads resolve to a DID first (via the identity directory), then hit the PK; no handle index needed.

## 3. Indexers

Two new indexers replace the sample. Both register on the existing `index.Dispatcher` in `deps.go`, keyed by their NSID.

### 3.1 `CraftskyProfile` (handles `social.craftsky.actor.profile`)

File: `appview/internal/index/craftsky_profile.go`.

- **`create` / `update`** — unmarshal the record, read `crafts` (default `[]` if absent), upsert by DID:

  ```sql
  INSERT INTO craftsky_profiles (did, crafts, record_cid)
  VALUES ($1, $2, $3)
  ON CONFLICT (did) DO UPDATE SET
      crafts = EXCLUDED.crafts,
      record_cid = EXCLUDED.record_cid,
      indexed_at = now()
  WHERE craftsky_profiles.record_cid != EXCLUDED.record_cid;
  ```

  The `WHERE` clause makes replayed events with the same CID true no-ops — `indexed_at` is preserved. `created_at` is not touched by the `DO UPDATE` branch, so it keeps its insert-time value.

- **`delete`** — in one transaction, `DELETE FROM craftsky_profiles WHERE did = $1` and `DELETE FROM bluesky_profiles WHERE did = $1`. The user has left Craftsky (by deleting their `social.craftsky.actor.profile`) and so is no longer a member; the Bluesky-side row must not linger, otherwise they come back silently on the next Bluesky profile event.

- **Malformed record** — return a wrapped error so Tap retries. The OAuth-callback write path validates records at write time (§4.2), so this fires only for malformed writes originating elsewhere — a pathological case. The indexer logs and errors rather than guessing.

### 3.2 `BlueskyProfile` (handles `app.bsky.actor.profile`)

File: `appview/internal/index/bluesky_profile.go`.

- **`create` / `update`** — check membership first: `SELECT 1 FROM craftsky_profiles WHERE did = $1`. If the user is not a member, return `nil` (successfully processed by dropping). This is how we keep `bluesky_profiles` proportional to the Craftsky userbase rather than the whole atproto network. If they are a member, upsert:

  ```sql
  INSERT INTO bluesky_profiles
      (did, display_name, description, avatar_cid, avatar_mime,
       banner_cid, banner_mime, record_cid)
  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
  ON CONFLICT (did) DO UPDATE SET
      display_name = EXCLUDED.display_name,
      description = EXCLUDED.description,
      avatar_cid = EXCLUDED.avatar_cid,
      avatar_mime = EXCLUDED.avatar_mime,
      banner_cid = EXCLUDED.banner_cid,
      banner_mime = EXCLUDED.banner_mime,
      record_cid = EXCLUDED.record_cid,
      indexed_at = now()
  WHERE bluesky_profiles.record_cid != EXCLUDED.record_cid;
  ```

- **`delete`** — `DELETE FROM bluesky_profiles WHERE did = $1`. The user deleted their Bluesky profile record; the two records are independent on the atproto side, so this does not touch `craftsky_profiles`. Run the membership check first and drop if not a member, to avoid paying a write for replays from ex-members.

### 3.3 Wiring

In `appview/internal/app/deps.go`:

```go
// before
blueskySample := index.NewBlueskyPostsSample(pool)
dispatcher := index.NewDispatcher(index.NotImplemented{})
dispatcher.Register("app.bsky.feed.post", blueskySample)

// after
dispatcher := index.NewDispatcher(index.NotImplemented{})
dispatcher.Register("social.craftsky.actor.profile", index.NewCraftskyProfile(pool))
dispatcher.Register("app.bsky.actor.profile", index.NewBlueskyProfile(pool))
```

### 3.4 Dropping the sample indexer

As part of this spec:

- Delete `appview/internal/index/bluesky_posts_sample.go` and its test.
- Remove the registration in `deps.go`.
- Add migration `000007` to drop the `bluesky_posts_sample` table.

The code comment on `BlueskyPostsSample` explicitly called out deletion as a condition of the first real `social.craftsky.*` indexer landing. This spec is that moment.

### 3.5 Login ↔ firehose race

The login initialiser (§4) writes via `com.atproto.repo.putRecord` on the user's PDS. The firehose then delivers the resulting event back to our indexer. Login does **not** write to Postgres directly — the firehose is the sole path from PDS state to our tables.

For a brand-new user, there is a small window between `/oauth/callback` returning and the firehose event being indexed, during which `GET /v1/profiles/me` would 404. The OAuth callback is already slow (PAR + token exchange + multiple PDS round-trips), so the firehose event typically arrives first in practice, but we make no guarantee. Clients should tolerate a transient 404 on the very first profile read after onboarding. Flagged as a known behaviour, not a bug.

For returning users, both rows were written by the firehose when those records were first created on the PDS, so they already exist by the time the user logs in again.

## 4. Onboarding-on-login

Modifies the existing OAuth callback path. Runs after `ProcessCallback` + `SaveSession` succeed and before the `craftsky_sessions` row is returned to the client.

### 4.1 Flow

```
/oauth/callback arrives
├── ProcessCallback(code, state)                               [existing]
├── SaveSession(...)                                           [existing]
├── NEW: initializeProfile(ctx, did, oauthSessionID)
│   ├── client := oauthApp.ResumeSession(ctx, did, oauthSessionID).APIClient()
│   ├── getRecord("app.bsky.actor.profile", did, "self")
│   │     ├── 404: continue with no Bluesky profile data
│   │     ├── other error: fail login (profile_init_failed)
│   │     └── ok: retain fetched record
│   ├── getRecord("social.craftsky.actor.profile", did, "self")
│   │     ├── 404: putRecord("social.craftsky.actor.profile", did, "self", {crafts: []})
│   │     │       on PDS error: fail login (profile_init_failed)
│   │     ├── other error: fail login (profile_init_failed)
│   │     └── ok + lexicon-invalid: fail login (profile_data_invalid)
│   │     └── ok + lexicon-valid: proceed
│   └── return nil
├── create craftsky_sessions row                               [existing]
└── hand token to client                                       [existing]
```

### 4.2 Missing vs malformed records

- **Missing `app.bsky.actor.profile`** is normal — a DID can exist without one. Proceed; the user's future `GET /v1/profiles/me` will return nulls for the Bluesky-side fields until they set them.
- **Missing `social.craftsky.actor.profile`** is expected for new users. The initialiser writes an empty `{crafts: []}` record.
- **Malformed `social.craftsky.actor.profile`** (present but fails lexicon validation) fails the login. This is a legitimate data-integrity issue; papering over it is worse than surfacing it. In practice this only fires in the dev/ecosystem period when another client might write invalid data against our lexicon.

### 4.3 Write for the empty Craftsky profile

```
com.atproto.repo.putRecord
  repo:       {did}
  collection: social.craftsky.actor.profile
  rkey:       self
  record:     {"$type": "social.craftsky.actor.profile", "crafts": []}
```

indigo's DPoP-signing `APIClient` handles auth, nonce rotation, and retry. The resulting firehose event is picked up by the `CraftskyProfile` indexer and inserted into the DB normally.

### 4.4 Error responses from the callback

All existing callback error paths from the OAuth BFF spec §3.6 stay. New error modes:

| Failure | Response |
|---|---|
| getRecord `app.bsky.actor.profile` returns non-404 error | HTML error page: `profile_init_failed` |
| getRecord `social.craftsky.actor.profile` returns non-404 error | HTML error page: `profile_init_failed` |
| putRecord for the empty Craftsky profile fails | HTML error page: `profile_init_failed` |
| Fetched `social.craftsky.actor.profile` fails lexicon validation | HTML error page: `profile_data_invalid` |

The callback renders HTML, not JSON, because the user's system browser lands on it — matching the existing callback error-handling convention.

### 4.5 Idempotency

- **Callback retry.** A user who interrupts and re-runs the OAuth dance hits a fresh callback. `getRecord` finds the Craftsky record written on a prior attempt and skips the write. All reads are side-effect-free; the one write is guarded by 404.
- **Concurrent logins on multiple devices.** Two logins hitting `getRecord` simultaneously can both see 404 and both call `putRecord`. The second overwrites the first with identical content (same `$type`, same `crafts: []`). The firehose delivers both events; the indexer's CID-guard upsert is a no-op for the second.

### 4.6 Code layout

- **New file:** `appview/internal/auth/initialize_profile.go` with `initializeProfile(ctx, did, oauthSessionID)` and its helpers. Pure PDS-client work; no Postgres access.
- **Modified:** `appview/internal/auth/handlers_oauth.go` to call the initialiser from the callback handler.

## 5. HTTP endpoints

Three endpoints, all under `/v1/profiles/`, all authenticated.

### 5.1 `GET /v1/profiles/@{handleOrDid}`

**Auth:** required (`Authorization: Bearer`, `X-Craftsky-Device-Id`).

**Path param:** `{handleOrDid}` — either a handle (e.g. `alice.bsky.social`) or a DID (e.g. `did:plc:xyz`). The `@` prefix is part of the URL shape per API architecture spec §2.2. DIDs are identified by their `did:` prefix.

**Flow:**

1. Parse the path segment into `syntax.Handle` or `syntax.DID`.
2. If a handle, resolve to DID via `HandleResolver.ResolveDID`. On error → `502 identity_unavailable`.
3. `SELECT` from `craftsky_profiles` by DID. If no row → `404 profile_not_found`. This is the membership gate: non-Craftsky users are invisible to this endpoint.
4. `SELECT` from `bluesky_profiles` by DID. A missing row is fine — the user has no Bluesky profile record. Nullable fields remain null.
5. Resolve the current handle via `HandleResolver.ResolveHandle` — we always return the current handle in responses, not a cached one. On error → `502 identity_unavailable`.
6. Compose the response per §5.4.

**Errors:**

| Code | Status | Meaning |
|---|---|---|
| `invalid_identifier` | 400 | Path segment parsed as neither handle nor DID. |
| `profile_not_found` | 404 | DID is not a Craftsky member. |
| `identity_unavailable` | 502 | Directory lookup failed (either direction). |

### 5.2 `GET /v1/profiles/me`

**Auth:** required.

Same as §5.1, with the DID pulled from context (via `middleware.GetDID`) rather than the path. The handle-resolution step still runs so the response contains the current handle.

Per API architecture spec §2.2, `me` is the preferred self-referential form; a client that calls `GET /v1/profiles/@{did}` with its own DID gets the same result.

### 5.3 `PUT /v1/profiles/me`

**Auth:** required.

**Request body** (camelCase per the wire-alignment spec):

```json
{
  "displayName": "Alice",
  "description": "textile person",
  "crafts": ["knitting", "sewing"]
}
```

All three fields optional. Missing = cleared (per full-replace PUT semantics from the API architecture spec §2.3). Avatar and banner are **not** accepted in v1:

- If the body contains `avatar` or `banner`, return `400 unexpected_field` with a `fields` map pointing to the offending keys. Explicit rejection (rather than silent ignore) so clients don't believe they succeeded.

**Validation:**

| Field | Constraint |
|---|---|
| `displayName` | ≤ 64 graphemes / 640 bytes (Bluesky convention). |
| `description` | ≤ 256 graphemes / 2560 bytes (Bluesky convention). |
| `crafts` | `maxLength: 10`; each item ≤ 50 graphemes / 50 bytes (matches `social.craftsky.actor.profile`). |

Failures → `422 validation_failed` with a `fields` map.

**Write flow:**

1. Resume the user's OAuth session; get an indigo `APIClient`.
2. **Read-before-write on the Bluesky side.** Call `com.atproto.repo.getRecord` for `app.bsky.actor.profile`. We need current avatar/banner blob refs to preserve them on the full-replace `putRecord` (the client didn't send them in v1, but we must not blow them away). On PDS error → `502 pds_read_failed`. On 404 → treat as empty record; we're writing a fresh one.
3. Merge: take avatar + banner from the fetched record (if present), take displayName + description from the request (or clear to null if missing), produce the new Bluesky record body.
4. Parallel writes: `putRecord app.bsky.actor.profile` with the merged body and `putRecord social.craftsky.actor.profile` with `{crafts: <request value or []>}`.
5. Aggregate results per API architecture spec §4.2:

   | Both succeed | One succeeds | Both fail |
   |---|---|---|
   | `200 OK` with the updated profile (response shape = §5.4) | `502 pds_write_partial` with `fields: {"bsky": "ok"\|"failed", "craftsky": "ok"\|"failed"}` | `502 pds_write_failed` |

**Note on the response:** the AppView returns the profile composed from the record bodies it just wrote (merged with the client's input), not from the indexer's view. The firehose event may not have arrived yet. This makes the response immediately consistent with the write; a subsequent `GET` may briefly race the firehose, same as the onboarding-on-login window in §3.5.

**`createdAt` on the PUT response:** the AppView does not read `craftsky_profiles.created_at` for the PUT response (to avoid a DB round-trip whose result may not exist yet for brand-new users who just onboarded). Instead, the PUT response **omits `createdAt`**; the `GET` response is where clients read it. §5.4's "always present" rule applies to GET only; for PUT, `createdAt` is omitted from the JSON.

**Errors:**

| Code | Status | Meaning |
|---|---|---|
| `unexpected_field` | 400 | Body contains an unknown field. |
| `validation_failed` | 422 | Length / count violation. |
| `pds_read_failed` | 502 | Couldn't read current Bluesky record for merge. |
| `pds_write_partial` | 502 | One of the two writes failed. |
| `pds_write_failed` | 502 | Both writes failed. |

### 5.4 Response JSON shape

Used by all three endpoints' success responses:

```json
{
  "did": "did:plc:xyz",
  "handle": "alice.bsky.social",
  "displayName": "Alice",
  "description": "textile person",
  "avatar": "https://cdn.bsky.app/img/avatar/plain/did:plc:xyz/bafk...@jpeg",
  "banner": "https://cdn.bsky.app/img/banner/plain/did:plc:xyz/bafk...@jpeg",
  "crafts": ["knitting", "sewing"],
  "createdAt": "2026-04-23T10:00:00Z"
}
```

- `did`, `handle` — always present.
- `displayName`, `description`, `avatar`, `banner` — nullable; **omitted from the JSON when absent** (not emitted as `null`), matching REST convention.
- `crafts` — always an array, possibly empty.
- `createdAt` — `craftsky_profiles.created_at`, always present.

**Avatar/banner URL synthesis** (per Q9a): construct `https://cdn.bsky.app/img/{avatar|banner}/plain/{did}/{cid}@{ext}` where `{ext}` derives from the MIME type via this table:

| MIME type | `{ext}` |
|---|---|
| `image/jpeg` | `jpeg` |
| `image/png` | `png` |
| `image/gif` | `gif` |
| `image/webp` | `webp` |

If the MIME type is not in this table, **omit the field** rather than emit a broken URL. The list is intentionally the set that the Bluesky CDN is known to serve as of v1; additions require touching this spec.

**On `createdAt` for the PUT response:** omitted (see §5.3's "Note on the response"). The field is documented as `always present` for GET responses only.

**Avatar/banner writes:** `PUT /v1/profiles/me` accepts optional `avatar` and `banner` fields using the raw atproto blob object returned by `POST /v1/blobs/images`. Omitting either field preserves the existing value, sending `null` clears it, and sending a blob object replaces it. The AppView validates the blob shape, MIME type, and configured image byte limit before writing the merged `app.bsky.actor.profile` record.

### 5.5 Code layout

New files under `appview/internal/api/`:

- `profile.go` — handler factories: `GetProfileHandler`, `GetMeProfileHandler`, `PutMeProfileHandler`.
- `profile_request.go` — PUT request type + validation.
- `profile_response.go` — response type + URL synthesis.
- `profile_test.go` — handler unit tests.

Each handler takes explicit dependencies (pool, resolver, OAuth app), matching the existing `WhoAmIHandler` pattern. No handler takes `*Deps`.

### 5.6 Route registration

In `appview/internal/routes/routes.go`:

```go
mux.Handle("GET /v1/profiles/@{handleOrDid}", authN(deviceID(api.GetProfileHandler(...))))
mux.Handle("GET /v1/profiles/me",             authN(deviceID(api.GetMeProfileHandler(...))))
mux.Handle("PUT /v1/profiles/me",             authN(deviceID(api.PutMeProfileHandler(...))))
```

## 6. `HandleResolver` extension

Extend the existing interface to cover both directions, typed with indigo's `syntax.DID` / `syntax.Handle` rather than raw `string`.

### 6.1 Interface

```go
import "github.com/bluesky-social/indigo/atproto/syntax"

type HandleResolver interface {
    ResolveHandle(ctx context.Context, did syntax.DID) (syntax.Handle, error)
    ResolveDID(ctx context.Context, handle syntax.Handle) (syntax.DID, error)
}
```

### 6.2 `DirectoryHandleResolver`

Both methods lose their internal parse step — parsing moves to the caller, at the type boundary:

```go
func (r DirectoryHandleResolver) ResolveHandle(ctx context.Context, did syntax.DID) (syntax.Handle, error) {
    id, err := r.Directory.LookupDID(ctx, did)
    if err != nil {
        return "", fmt.Errorf("%w: lookup: %v", ErrHandleUnavailable, err)
    }
    if id.Handle == "" || id.Handle == syntax.HandleInvalid {
        return "", fmt.Errorf("%w: empty handle for %s", ErrHandleUnavailable, did)
    }
    return id.Handle, nil
}

func (r DirectoryHandleResolver) ResolveDID(ctx context.Context, handle syntax.Handle) (syntax.DID, error) {
    id, err := r.Directory.LookupHandle(ctx, handle)
    if err != nil {
        return "", fmt.Errorf("%w: lookup: %v", ErrHandleUnavailable, err)
    }
    return id.DID, nil
}
```

The existing `ErrHandleUnavailable` sentinel is reused — handlers already map it to `502 identity_unavailable`.

No Craftsky-level cache. indigo's `identity.DefaultDirectory()` provides internal caching for both directions.

### 6.3 Downstream impact

- `appview/internal/api/whoami.go` pulls the DID from context as a `string` (via `middleware.GetDID`) and now must call `syntax.ParseDID` at the call site.
- Existing test stubs for `HandleResolver` update their signatures.
- `middleware.GetDID` continues to return `(string, bool)`. Migrating it to `(syntax.DID, bool)` is a cross-cutting change touching every authenticated handler; explicitly deferred to avoid scope creep in this spec.

## 7. Testing

Tests follow the compose-Postgres-on-host pattern already established: unit tests via `go test ./...` on the host, hitting the dev compose Postgres (matching `just test`).

### 7.1 Indexer tests

`appview/internal/index/craftsky_profile_test.go`, `bluesky_profile_test.go`:

- Exercise each action (`create`, `update`, `delete`) against a fixture `tap.Event`.
- Verify idempotency: re-deliver the same event, confirm behaviour (second delivery with same CID does not touch `indexed_at`; delivery with new CID does).
- `bluesky_profile_test` exercises the membership gate: event for a DID not in `craftsky_profiles` → row dropped.
- `craftsky_profile_test` exercises the cascading delete: deleting the Craftsky row removes the Bluesky row in the same transaction.

### 7.2 Login initialiser tests

`appview/internal/auth/initialize_profile_test.go`, against a mocked PDS client:

- Returning user, both records present → both reads succeed, no writes, returns nil.
- New user, no Craftsky record → Bluesky read succeeds, Craftsky read 404s, putRecord fires.
- User with no Bluesky profile → Bluesky 404, Craftsky 404, putRecord fires, login succeeds.
- Bluesky read non-404 error → function returns error; callback renders `profile_init_failed`.
- Craftsky read returns malformed record → function returns error; callback renders `profile_data_invalid`.
- Craftsky putRecord fails → function returns error; callback renders `profile_init_failed`.

The callback handler's existing tests gain one case: "initialiser error → error page rendered with the expected code."

### 7.3 Handler tests

`appview/internal/api/profile_test.go`, following the `whoami_test.go` pattern — table-driven, `httptest.NewRequest` / `NewRecorder`, fake `HandleResolver`, direct DB access for fixture setup.

- `GET /v1/profiles/@{handleOrDid}`: by handle (resolver called), by DID (resolver skipped), non-member DID (404), non-member handle (404), invalid identifier (400), resolver error (502).
- `GET /v1/profiles/me`: happy path, missing DID in context (500 — routing bug).
- `PUT /v1/profiles/me`: happy path, `avatar`/`banner` in body (400), oversize `displayName` (422), oversize `crafts` array (422), each of the PDS error paths in §5.3.

PUT tests use a mocked PDS client. The read-before-write step returns a fixture Bluesky record; tests assert the merged write body.

### 7.4 Routing test

`appview/internal/routes/routes_test.go` — add assertions for the three new routes (method + path → correct middleware chain).

### 7.5 Coverage gaps explicitly accepted

- No end-to-end firehose → indexer integration test. Dispatcher routing and Tap consumer are tested separately.
- No cross-table concurrency test. Happy path covered; concurrent correctness relies on Postgres transactional guarantees.
- No latency benchmark on the OAuth callback. It grows by 2–3 PDS round-trips; no target set in v1.

## 8. Implementation map

### 8.1 New files

```
appview/internal/index/craftsky_profile.go
appview/internal/index/craftsky_profile_test.go
appview/internal/index/bluesky_profile.go
appview/internal/index/bluesky_profile_test.go

appview/internal/auth/initialize_profile.go
appview/internal/auth/initialize_profile_test.go

appview/internal/api/profile.go
appview/internal/api/profile_request.go
appview/internal/api/profile_response.go
appview/internal/api/profile_test.go

appview/migrations/000007_drop_bluesky_posts_sample.up.sql
appview/migrations/000007_drop_bluesky_posts_sample.down.sql
appview/migrations/000008_craftsky_profiles.up.sql
appview/migrations/000008_craftsky_profiles.down.sql
appview/migrations/000009_bluesky_profiles.up.sql
appview/migrations/000009_bluesky_profiles.down.sql
```

### 8.2 Modified files

```
appview/internal/api/handle_resolver.go      (add ResolveDID, switch to syntax types)
appview/internal/api/handle_resolver_test.go
appview/internal/api/whoami.go               (parse syntax.DID at call site)
appview/internal/api/whoami_test.go

appview/internal/auth/handlers_oauth.go      (call initializeProfile before token return)
appview/internal/auth/handlers_test.go

appview/internal/app/deps.go                 (register new indexers, drop sample)
appview/internal/app/deps_test.go

appview/internal/routes/routes.go            (register 3 new routes)
appview/internal/routes/routes_test.go
```

### 8.3 Deleted files

```
appview/internal/index/bluesky_posts_sample.go
appview/internal/index/bluesky_posts_sample_test.go
```

### 8.4 Dependencies

No new Go modules. indigo's `oauth`, `syntax`, `identity`, and `APIClient`, plus `pgx` and stdlib, are all already in use.

### 8.5 Migration renumbering

Implementer runs `ls appview/migrations/` at implementation time; if new migrations have landed since this spec, renumber `000007`/`000008`/`000009` accordingly.

### 8.6 AGENTS.md updates

Short addition to the Architectural Rules or Coding Conventions section: note that new indexers register via `dispatcher.Register(nsid, idx)` in `deps.go`. Rules #1–#5 stand unchanged.

## 9. Open questions & future work

### 9.1 Flagged, not resolved

1. **Avatar/banner write path.** Landing this requires the deferred blob-upload spec. Until then, avatar/banner are read-only.
2. **`PATCH /v1/profiles/me`.** Deferred. If/when added, pin the field-omission contract (omitted = unchanged) to avoid ambiguity.
3. **Skip optimisation on PUT.** Every PUT writes both records even if one side is unchanged. If latency becomes an issue, compare merged Bluesky body against the current CID and short-circuit if identical. No wire change required.
4. **`middleware.GetDID` returning `syntax.DID`.** Kept as `(string, bool)` for now; migrating is a cross-cutting refactor.
5. **Membership table.** Using "DID exists in `craftsky_profiles`" as the check. A dedicated `craftsky_members` table becomes interesting only if the PK lookup on every Bluesky profile firehose event becomes a hotspot.

### 9.2 Follow-up specs this unblocks

1. **Profile counts** (followers, following, posts). Require graph + post indexers.
2. **Viewer-relationship fields** on responses. Require graph indexing.
3. **Blob upload.** Required before PUT can accept avatar/banner. Already flagged in the API architecture spec.
4. **`DELETE /v1/profiles/me`.** The indexer handles the firehose `delete` path, but no HTTP endpoint exposes it. Users leave Craftsky via other atproto clients in v1.
5. **Handle-change reindexing.** Not needed — we key everything by DID, resolvers run live on every GET. Flagged for completeness.
6. **`*.craftsky.social` handle service.** Own project.
7. **Profile search** ("find users by craft / handle / displayName"). Own spec alongside post search.
