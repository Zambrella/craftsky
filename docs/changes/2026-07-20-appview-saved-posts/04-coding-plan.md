# Coding Plan: AppView Saved Posts

## 1. Inputs

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` — Approved, no open findings
- Governing API contracts:
  - `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md`
  - `docs/superpowers/specs/2026-04-22-api-wire-alignment-design.md`
- Repository constraints: `AGENTS.md`, including private AppView storage, `/v1/` JSON conventions, typed atproto identifiers, pgx persistence, and no lexicon change for this slice. The user explicitly approved the server's established raw-query pattern for this feature rather than introducing sqlc as part of the slice.

## 2. Implementation Strategy

Implement saved posts as one private AppView feature with three cooperating boundaries:

1. A reversible `000024` migration creates owner-scoped folders and saves. PostgreSQL constraints enforce one save per owner/post, one optional same-owner folder, owner-membership cleanup, exact-post cleanup, and non-destructive folder deletion.
2. A dedicated `SavedPostStore` owns saved/folder mutations and pagination using parameterized raw SQL through pgx, matching the current AppView store pattern. Its methods keep `syntax.DID` and `syntax.ATURI` at the Go boundary, use explicit transactions where atomicity is required, and map database failures to feature errors.
3. Existing canonical post hydration remains authoritative. `PostStore.EngagementSummaries` gains one set-based viewer-saved query and `PostResponse` gains the two additive viewer fields. Saved-list handlers load saved references, hydrate canonical posts in batches, apply current content/relationship policy, and attach quote views through the same paths used elsewhere.

Permanent reply-context cleanup belongs in the existing `CraftskyPost.handleDelete` transaction. It must delete exact-target saves and saves for still-indexed descendants before deleting the event URI, while leaving descendant public post rows intact. Temporary content, membership, moderation, and relationship ineligibility only suppresses list output; it never deletes private rows.

The implementation is backend-only. It adds no Flutter providers, screens, routes, lexicons, PDS calls, Tap collection, notification, or public count.

## 3. Affected Areas

| Area | Existing Pattern | Planned Change | Requirement IDs | Test IDs |
|---|---|---|---|---|
| Migration | Numbered golang-migrate up/down pairs; highest current version is `000023` | Add `000024_saved_posts` tables, constraints, and indexes; verify current maximum again immediately before implementation | FR-004, FR-006, FR-013, FR-014, FR-018, FR-019, NFR-006, RULE-001–RULE-007 | IT-001, IT-004, IT-009, IT-011, REG-007 |
| Query implementation | AppView stores currently keep parameterized raw SQL and explicit row scanning beside narrow pgx-backed methods | Keep saved/folder SQL localized in `SavedPostStore`; keep the recursive deletion CTE beside the existing indexer transaction; add no query generator or dependency | FR-002–FR-009, FR-013, FR-014, FR-017–FR-019, NFR-002, NFR-004 | IT-001–IT-005, IT-009–IT-011, IT-014 |
| Saved-state storage | API stores wrap pgx and expose narrow handler interfaces | Add `SavedPostStore` for folder CRUD, tri-state save upsert, idempotent unsave, scoped pagination, and bounded context checks | FR-001–FR-010, FR-017–FR-020, NFR-002–NFR-004 | AT-001–AT-004, AT-010, UT-002, UT-007, UT-010, IT-002–IT-005, IT-011 |
| Request/response contract | Strict decoding, `FieldError`, camelCase DTOs, standard envelope | Add saved request parsing, name validation, list filters, folder/save DTOs, cursor codecs, and feature error mapping | FR-002, FR-004–FR-010, FR-015, FR-017, FR-020, NFR-003 | AT-001–AT-004, AT-009, UT-001–UT-004, UT-006, UT-009–UT-010, IT-006 |
| Canonical post response | `PostResponse` plus `EngagementSummary` and `applyEngagementSummary` | Add `viewerHasSaved` and nullable `viewerSavedFolderId`; load viewer save state once per URI batch inside `EngagementSummaries` | FR-010–FR-012, NFR-002, NFR-004, NFR-006 | AT-005–AT-007, UT-008–UT-009, IT-007–IT-010, IT-014–IT-015, REG-001–REG-004 |
| Saved-list hydration | Existing post, relationship, moderation, handle, and quote-view batch helpers | Add bounded batch reads for canonical rows and required reply context; preserve saved ordering while omitting ineligible payloads | FR-008–FR-013, RULE-006, RULE-008 | AT-004–AT-008, IT-005, IT-007–IT-010, REG-002, REG-004 |
| Post deletion lifecycle | `CraftskyPost.handleDelete` already owns one transaction for indexed delete and notification retraction | Add a recursive descendant-save deletion query before the indexed post delete in the same transaction | FR-013, FR-014, RULE-006, RULE-008 | AT-005, AT-008, UT-005, IT-009 |
| Membership lifecycle | `craftsky_profiles` deletion is transactional; private owner rows use cascade | Let owner foreign keys remove saved rows/folders; do not connect session/device lifecycle to saved storage | FR-014, NFR-004 | AT-008, UT-005, IT-009, REG-005 |
| Route registry | `routes.go` plus exhaustive `RoutePolicy` table and middleware wrapping | Register seven authenticated routes with read/write rates and optional-JSON/no-body policies | FR-015, FR-020 | AT-009, IT-006, IT-012, REG-006 |
| Privacy/observability | Bounded API/DB operation labels; no private identifiers in log attributes | Instrument only fixed operation/result/stage/error-class values and prove no PDS/Tap/notification side effect | BR-004, FR-016, NFR-001, NFR-005, RULE-005 | AT-007, AT-009, UT-011, IT-013, REG-003 |

## 4. Files And Modules

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `appview/migrations/000024_saved_posts.up.sql` | Create | Create `saved_post_folders` and `saved_posts`, foreign keys, checks, and ordering indexes | FR-004, FR-006, FR-013, FR-014, FR-018, FR-019 | IT-001, IT-004, IT-009, IT-011, REG-007 |
| `appview/migrations/000024_saved_posts.down.sql` | Create | Drop only the two saved-state tables in dependency order | FR-019, NFR-006 | IT-001, REG-007 |
| `appview/internal/db/saved_posts_migration_test.go` | Create | Up/down/up schema, constraint, delete-action, duplicate-name, and index contract | FR-019, NFR-006 | IT-001, REG-007 |
| `appview/internal/api/saved_post_request.go` | Create | Strict optional-body decoding, tri-state folder assignment, folder-name validation, list query parsing | FR-002, FR-004, FR-008–FR-009, FR-015, FR-017 | UT-001–UT-002, UT-010, IT-006 |
| `appview/internal/api/saved_post_cursor.go` | Create | Scope/sort-bound saved and folder keyset cursor codecs using `envelope` | FR-007–FR-009, NFR-003 | UT-003–UT-004, IT-003, IT-005 |
| `appview/internal/api/saved_post_store.go` | Create | pgx-backed raw queries, explicit transactions, row scanning, error translation, pagination, and batch context availability | FR-001–FR-009, FR-013–FR-020, NFR-002, NFR-004 | IT-002–IT-005, IT-008–IT-011, IT-014 |
| `appview/internal/api/saved_post.go` | Create | DTOs, narrow store/hydrator interfaces, seven handlers, response building, and policy shaping | BR-001–BR-004, FR-001–FR-020 | AT-001–AT-010, IT-006–IT-010, IT-012–IT-013 |
| `appview/internal/api/saved_post_folder_request_test.go` | Create | Folder-name boundary table | FR-004, RULE-007 | UT-001 |
| `appview/internal/api/saved_post_request_test.go` | Create | Absent/omitted/null/value body states and strict JSON failures | FR-002, RULE-002 | UT-002 |
| `appview/internal/api/saved_post_cursor_test.go` | Create | Cursor round-trip, shape, compatibility, and opaque-ID behavior | FR-007, FR-009, NFR-003 | UT-003–UT-004 |
| `appview/internal/api/saved_post_lifecycle_test.go` | Create | Table-driven retention/destructive-event and timestamp-effects coverage | FR-005, FR-013–FR-014, RULE-003, RULE-006 | UT-005, UT-007 |
| `appview/internal/api/saved_post_response_test.go` | Create | Status choice, DTO casing, exact reply metadata, and central engagement application | FR-010–FR-011, FR-020 | UT-006, UT-009, IT-010 |
| `appview/internal/api/saved_post_policy_test.go` | Create | Muted direct-access allowance and strict-state payload suppression without row deletion | FR-012–FR-013 | UT-008, IT-008 |
| `appview/internal/api/saved_post_error_test.go` | Create | Missing/foreign folder indistinguishability and operation-specific 404/204 mapping | FR-017 | UT-010 |
| `appview/internal/api/saved_post_observability_test.go` | Create | Private sentinel redaction and no external collaborator calls | FR-016, NFR-001, NFR-005, RULE-005 | UT-011, IT-013 |
| `appview/internal/api/saved_post_store_test.go` | Create | Real-Postgres CRUD, pagination, policy retention, ownership, atomicity, and concurrency | FR-001–FR-020, NFR-002–NFR-004 | IT-002–IT-005, IT-007–IT-011 |
| `appview/internal/api/saved_post_test.go` | Create | End-to-end handler contracts with fakes plus authenticated request context | FR-001–FR-020 | AT-001–AT-010, IT-002, IT-005–IT-007, IT-013 |
| `appview/internal/api/saved_post_folder_test.go` | Create | Folder handler contracts, duplicate names, timestamps, and idempotent delete | FR-004–FR-007, FR-017, FR-020 | AT-003, IT-003–IT-004, IT-006 |
| `appview/internal/api/saved_post_query_plan_test.go` | Create | `EXPLAIN (FORMAT JSON)` and bounded-call regression for saved indexes and viewer-state query | FR-011, FR-019, NFR-002 | IT-014 |
| `appview/internal/api/post_store.go` | Change | Add batch eligible-post/context reads and one viewer-saved query inside `EngagementSummaries` | FR-010–FR-013, NFR-002, NFR-004 | IT-007–IT-010, IT-014–IT-015 |
| `appview/internal/api/post_response.go` | Change | Add and apply `ViewerHasSaved` and nullable `ViewerSavedFolderID` | FR-011 | UT-009, IT-010, IT-015, REG-001 |
| `appview/internal/api/post_response_test.go` | Change | Assert additive camelCase fields and unchanged existing shapes | FR-010–FR-011, NFR-006 | UT-009, REG-001 |
| Existing `post_test.go`, `timeline_test.go`, `notifications_test.go`, `search_*_test.go` | Change | Assert every canonical consumer receives the same saved viewer fields through `EngagementSummaries` | FR-011, NFR-002, NFR-006 | IT-010, IT-015, REG-001–REG-004 |
| `appview/internal/index/craftsky_post.go` | Change | Execute a parameterized recursive descendant-save cleanup CTE inside `handleDelete` before the post-row delete | FR-013, RULE-006, RULE-008 | AT-005, IT-009 |
| `appview/internal/index/craftsky_post_test.go` and shared index DDL fixtures | Change | Add saved tables and exact/root/intermediate deletion cases; retain descendant posts | FR-013–FR-014 | AT-005, AT-008, IT-009 |
| `appview/internal/index/craftsky_profile_test.go` or equivalent membership-deletion integration fixture | Change | Prove owner cascade and preservation of retained public content | FR-014 | AT-008, IT-009, REG-005 |
| `appview/internal/routes/routes.go` | Change | Construct the two stores and register all seven handlers | FR-015 | IT-012, REG-006 |
| `appview/internal/routes/policy.go` | Change | Add authenticated route policies with correct rate/body classes | FR-015 | IT-012, REG-006 |
| `appview/internal/routes/routes_test.go` | Change | Cover registration, auth/device/body/rate/error-envelope behavior | FR-015 | AT-009, IT-012, REG-006 |

No `lexicon/`, Flutter, PDS client, Tap filter, notification schema, or public interaction module changes are planned.

## 5. Services, Interfaces, And Data Flow

### 5.1 Persistence model

Use UUID columns internally for folder IDs because the runtime already depends on `github.com/google/uuid`; serialize them only as opaque strings at API boundaries.

```text
saved_post_folders
  id          UUID primary key
  owner_did   TEXT not null -> craftsky_profiles(did) on delete cascade
  name        TEXT not null
  created_at  TIMESTAMPTZ not null
  updated_at  TIMESTAMPTZ not null
  unique (owner_did, id)
  check char_length(name) between 1 and 100

saved_posts
  owner_did   TEXT not null -> craftsky_profiles(did) on delete cascade
  post_uri    TEXT not null -> craftsky_posts(uri) on delete cascade
  folder_id   UUID null
  saved_at    TIMESTAMPTZ not null
  primary key (owner_did, post_uri)
  foreign key (owner_did, folder_id)
    -> saved_post_folders(owner_did, id)
    on delete set null (folder_id)
```

PostgreSQL 16's column-specific `SET NULL` keeps `owner_did` intact while unfiling saves. `IT-001` must prove this exact behavior; if the migration or driver exposes an incompatibility, the allowed fallback is an explicit update-then-delete transaction with a non-cascading composite foreign key.

Indexes:

- `(owner_did, lower(name), id)` for deterministic case-insensitive folder pages.
- `(owner_did, saved_at DESC, post_uri DESC)` for all saves; the same btree supports reverse scans for oldest-first.
- `(owner_did, folder_id, saved_at DESC, post_uri DESC) WHERE folder_id IS NOT NULL`.
- `(owner_did, saved_at DESC, post_uri DESC) WHERE folder_id IS NULL`.

Names are normalized and validated in Go before persistence. The SQL length check is a defensive invariant, not the source of slash/control validation. There is deliberately no name uniqueness index.

### 5.2 Raw pgx query boundary

`SavedPostStore` holds `*pgxpool.Pool`, following existing AppView stores. Parameterized SQL remains private to the implementation file, with small typed scan helpers for folder, save-state, and saved-reference rows. Public methods return feature types rather than `pgx.Row` or database-specific structs and translate `pgx.ErrNoRows`, UUID parse failures, and named constraint violations into stable errors.

Planned query groups:

- Folder: create, read-owned, rename-owned, delete-owned, and keyset list after `(lower(name), id)`.
- Save: atomic upsert with `assignmentPresent`, delete by owner/URI, read state, and batch viewer states.
- Saved list: newest/oldest variants for all, folder, and unfiled scopes, each selecting `limit + 1` references under an owner predicate.
- Hydration: batch eligible canonical rows and batch required reply-context chains for at most one page of URIs.
- Lifecycle: one parameterized recursive CTE in `CraftskyPost.handleDelete`, executed on its existing `pgx.Tx` before the indexed post row is removed.

Do not create a generic query wrapper or repository-wide database abstraction. Methods that require multiple statements accept or open an explicit pgx transaction; single-statement reads and mutations use the pool directly. Real-Postgres tests, query-plan assertions, owner-isolation cases, and concurrency barriers verify the raw queries at their actual execution boundary.

### 5.3 API-facing types and interfaces

```text
type FolderAssignment struct {
    Present bool      // body contained folderId
    ID      *string   // nil when explicit JSON null
}

type SavedPostState struct {
    SavedAt time.Time
    FolderID *string
}

type SaveMutationResult struct {
    State SavedPostState
    Created bool
}

type SavedPostFolder struct {
    ID string
    Name string
    CreatedAt time.Time
    UpdatedAt time.Time
}

type SavedPostRef struct {
    PostURI syntax.ATURI
    SavedAt time.Time
    FolderID *string
}

type SavedPostStoreWriter interface {
    Save(ctx, owner, postURI, assignment) (SaveMutationResult, error)
    Unsave(ctx, owner, postURI) error
    CreateFolder(ctx, owner, name) (SavedPostFolder, error)
    RenameFolder(ctx, owner, folderID, name) (SavedPostFolder, error)
    DeleteFolder(ctx, owner, folderID) error
}

type SavedPostStoreReader interface {
    ListFolders(ctx, owner, limit, cursor) ([]SavedPostFolder, string, error)
    ListSavedRefs(ctx, owner, filter) ([]SavedPostRef, string, error)
}

type SavedPostHydrator interface {
    ResolveSaveTarget(ctx, did, rkey) (PostTargetRef, error)
    ReadEligiblePostsByURI(ctx, viewer, uris) (map[uri]*PostRow, error)
    RequiredContextStates(ctx, viewer, uris) (map[uri]ContextState, error)
    RelationshipStates(ctx, viewer, subjects) (...)
    EngagementSummaries(ctx, viewer, uris) (...)
    QuoteViewRows(ctx, refs) (...)
}
```

These are partial signatures. Production routes pass `*SavedPostStore` and `*PostStore`; focused handler tests use fakes implementing only the required interface.

### 5.4 Save mutation flow

```text
authenticated DID
  -> parse path DID and record key once
  -> decode absent / omitted / null / value folderId
  -> PostStore.ResolveSaveTarget
       requires indexed post, current author membership, visible moderation,
       navigable required reply context
  -> batch/single relationship state
       muted: eligible
       either block direction: post_not_found (no protected payload)
  -> SavedPostStore.Save transaction
       parse optional folder storage ID
       require same-owner folder when non-null
       INSERT ... ON CONFLICT (owner_did, post_uri)
       preserve existing folder on omission
       replace on value; clear on explicit null
       preserve saved_at for every existing row
       return Created atomically
  -> 201 when Created, otherwise 200
  -> body {savedAt, folderId}
```

`Created` must be determined by the atomic upsert result, not a preflight existence query that races. The store query may use a PostgreSQL returning expression or a transaction-local CTE, but `IT-002` and `IT-011` define the result rather than a particular SQL trick.

Unsave parses the typed path and constructs the canonical Craftsky post AT-URI without reading `craftsky_posts`. It deletes only `(authenticated owner, URI)` and always returns 204, including after target deletion.

### 5.5 Folder flow

- Generate UUID in Go, but expose it as an opaque string.
- Trim name, count Unicode code points, reject slash/backslash/control characters, then persist accepted casing.
- Rename updates only `name` and `updated_at` under the owner predicate.
- Save mutations never update folder metadata.
- Delete uses `DELETE ... WHERE owner_did = $owner AND id = $id`; the column-specific foreign key unfiles contained saves atomically. Missing, malformed-storage, and foreign IDs all become the same successful no-op.
- Rename/list/assignment parse failures or non-owned IDs map to `ErrSavedPostFolderNotFound`, producing the indistinguishable 404.

### 5.6 Saved-list flow

```text
parse filter + cursor
  -> list limit+1 owner-scoped SavedPostRefs in savedAt/URI order
  -> batch ReadEligiblePostsByURI for at most 101 URIs
  -> batch RequiredContextStates for the same URIs
  -> batch RelationshipStates for target/context authors
  -> omit target or context that is missing, non-member, moderated,
     unavailable, or blocked; retain its saved row
  -> allow otherwise-eligible muted target/context under direct-access semantics
  -> one EngagementSummaries call for the remaining URI set
  -> resolve handles, BuildPostResponse, apply summary, attach quote views
  -> restore original savedAt/URI ordering
  -> emit {post, savedAt, folderId} items and the candidate-page cursor
```

Required reply context is checked with one bounded recursive CTE seeded by the page's reply URIs. The traversal tracks visited URIs and a maximum depth so malformed cycles cannot become an unbounded query. A valid reply must reach its indexed root through the parent chain; every required row must remain current-member and moderation-eligible. This query never archives or copies post content.

Policy suppression may produce a short or empty page with a continuation cursor because pagination advances over owner save references before private relationship shaping. This preserves bounded work and deterministic progress; it must never loop or expose a protected payload.

### 5.7 Shared viewer-state hydration

Extend `EngagementSummary`:

```text
ViewerHasSaved       bool
ViewerSavedFolderID  *string
```

`PostStore.EngagementSummaries` keeps its existing public method. Its observed implementation adds exactly one batch query for `(viewerDID, postURIs)` and merges the result with like/repost/quote/reply summaries. `applyEngagementSummary` copies the two fields into `PostResponse`.

Because timeline, search, profile content, posts, comments/replies, notifications, quotes-as-posts, and saved lists already call this method (directly or through `SearchStore` delegation), no parallel saved-state helper is added to those handlers. Tests update their fakes and assertions; source changes should be limited to callers that currently bypass the shared seam.

`QuotePreviewPost` remains intentionally compact and does not gain saved fields. A quote post's outer canonical `PostResponse` does.

### 5.8 Permanent deletion flow

Within `CraftskyPost.handleDelete`'s existing transaction:

```text
WITH RECURSIVE affected(uri, path) AS (
  exact event URI
  UNION descendants seeded by reply_parent_uri = event URI
                    or reply_root_uri = event URI
  UNION deeper descendants joined on reply_parent_uri
)
DELETE saved_posts whose post_uri is in affected;
DELETE craftsky_posts WHERE uri = event URI;
retract notification source;
COMMIT;
```

The recursive query must prevent cycles and include `reply_root_uri = eventURI` as a root-delete safety net when an intermediate indexed parent is already missing. Only save rows are deleted for descendants. Their `craftsky_posts` rows survive unless Tap supplies their own delete events. Exact-target foreign-key cascade remains a second integrity guard.

Owner membership deletion requires no new service call: both private tables cascade from `craftsky_profiles`. Sign-out, logout-all, device removal, expiry, reinstall, and account switching never call this store.

## 6. State, Providers, Controllers, Or DI

No Flutter/Riverpod state is part of this slice.

Server dependency wiring remains local to `routes.AddRoutes`, following the current `PostStore`/`SearchStore` pattern:

```text
postStore  := api.NewPostStore(deps.DB, observer)
savedStore := api.NewSavedPostStore(deps.DB, observer)

save/unsave handlers      <- savedStore + postStore
folder handlers           <- savedStore
saved-list handler        <- savedStore + postStore + HandleResolver
existing canonical routes <- postStore (unchanged dependency shape)
```

No new field is required on `app.Deps`; `routes.go` already has the pool and observer and constructs request-serving stores. `SavedPostStore` retains the shared pool like existing stores, while request-scoped transactions remain local to the methods that need them.

## 7. UI, Widgets, Routes, Or User-Facing Surfaces

### Routes

| Method and path | Handler | Policy | Success |
|---|---|---|---|
| `POST /v1/posts/{did}/{rkey}/saves` | `SavePostHandler` | Authenticated, device ID, write rate, `BodyDefaultJSON` permitting empty body | `201` new or `200` existing; `{savedAt, folderId}` |
| `DELETE /v1/posts/{did}/{rkey}/saves` | `UnsavePostHandler` | Authenticated, device ID, write rate, no body | `204` |
| `GET /v1/saved-posts` | `ListSavedPostsHandler` | Authenticated, device ID, read rate, no body | `{items, cursor?}` |
| `GET /v1/saved-post-folders` | `ListSavedPostFoldersHandler` | Authenticated, device ID, read rate, no body | `{items, cursor?}` |
| `POST /v1/saved-post-folders` | `CreateSavedPostFolderHandler` | Authenticated, device ID, write rate, default JSON | `201` folder |
| `PATCH /v1/saved-post-folders/{folderId}` | `RenameSavedPostFolderHandler` | Authenticated, device ID, write rate, default JSON | `200` folder |
| `DELETE /v1/saved-post-folders/{folderId}` | `DeleteSavedPostFolderHandler` | Authenticated, device ID, write rate, no body | `204` |

All errors use `envelope.WriteError`; all JSON keys are camelCase. Default list limit is 50, maximum 100. A terminal page omits `cursor` rather than returning null or an empty string.

### Cursor payload sketches

```text
saved list:
  kind, scope(all|folder|unfiled), folderId?, sort(newest|oldest), savedAt, uri

folder list:
  kind, foldedName, folderId
```

No owner DID is encoded. Folder IDs, folded names, and post URIs are owner-visible keyset values under the existing non-confidential base64url-JSON contract.

### User interface

None identified. Flutter saved-post controls, screens, folder pickers, navigation, local caching, and optimistic behavior remain out of scope.

## 8. Error, Loading, Empty, And Edge States

| State / Case | Planned Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| Missing authenticated DID | Standard internal error; middleware normally prevents it | FR-015, NFR-004 | IT-006, IT-012 |
| Invalid path DID or record key | Parse once at HTTP boundary; return 400 without store access | FR-001, FR-015 | IT-006 |
| Target missing, moderated, non-member, invalid-context, or blocked | Return indistinguishable `404 post_not_found`; create no save | FR-001, FR-012, FR-017 | AT-001, AT-006, IT-006, IT-008 |
| Muted but otherwise eligible target | Allow save/list and apply current viewer mute state | FR-012 | AT-006, UT-008, IT-008 |
| No POST body or omitted `folderId` | New save unfiled; existing save preserves folder | FR-002, RULE-002 | AT-002, UT-002, IT-002 |
| Explicit `folderId: null` | Atomically unfile and preserve `savedAt` | FR-002, RULE-003 | AT-002, UT-002, IT-002 |
| Malformed, trailing, or unknown JSON | Return standard 400 error without mutation | FR-015 | AT-009, UT-002, IT-006 |
| Invalid folder name | Return 422 `validation_failed` with bounded field detail | FR-004–FR-005, RULE-007 | AT-003, UT-001, IT-003, IT-006 |
| Duplicate/case-variant folder name | Succeed with a distinct opaque ID | FR-004, RULE-007 | AT-003, IT-003 |
| Missing/foreign/malformed-storage folder for assignment, rename, or scope | Same 404 `saved_post_folder_not_found` and no mutation | FR-017 | AT-002–AT-004, UT-010, IT-002–IT-005 |
| Missing/foreign/malformed-storage folder delete | Return 204 and change nothing | FR-006, FR-017 | AT-003, UT-010, IT-004 |
| Invalid sort/filter/limit | Return 400 `validation_failed`; do not silently substitute defaults except when the parameter is absent | FR-008–FR-009, FR-015 | AT-004, IT-005–IT-006 |
| Malformed or incompatible cursor | Return 400 `invalid_cursor` | NFR-003 | AT-004, UT-003–UT-004, IT-005 |
| Empty folder/saved page | Return `items: []`; omit cursor | FR-007–FR-008 | IT-003, IT-005 |
| Temporarily hidden target/context | Omit protected payload, retain save/folder/timestamp, permit later reappearance | FR-012–FR-013, RULE-006 | AT-005–AT-006, IT-008 |
| Permanent exact/root/ancestor delete | Remove affected saves transactionally; retain unrelated and descendant public rows | FR-013, RULE-006, RULE-008 | AT-005, AT-008, IT-009 |
| Folder delete racing with move | Composite FK and transaction produce one valid ordering; map stale assignment to not found | FR-006, FR-018 | AT-010, IT-011 |
| Concurrent duplicate save | Atomic upsert leaves one row and one stable initial `savedAt` | FR-002, FR-018 | AT-010, IT-011 |
| Identity resolution failure during list hydration | Return 502 `identity_unavailable` without leaking identifiers | FR-010, NFR-001 | IT-006, IT-013 |
| Database/internal failure | Sanitized 500 envelope; logs carry bounded operation/stage/error class only | NFR-001, NFR-005 | UT-011, IT-013 |
| Immediate read after mutation | Read committed AppView state; no Tap wait | FR-016, FR-020 | AT-010, IT-002, IT-004, IT-006 |

## 9. Test Implementation Plan

| Order | Test ID | Target | Setup / Fixture | Initial Expected Failure |
|---|---|---|---|---|
| 1 | IT-001 | `internal/db/saved_posts_migration_test.go` | Current pre-feature profile/post schema; load `000024` up/down | Migration files/tables do not exist |
| 2 | UT-001 | `internal/api/saved_post_folder_request_test.go` | Unicode/name boundary table | Validator/type missing |
| 3 | UT-002 | `internal/api/saved_post_request_test.go` | Empty/omitted/null/value/malformed bodies | Tri-state decoder missing |
| 4 | UT-003–UT-004 | `internal/api/saved_post_cursor_test.go` | Fixed timestamps, scopes, duplicate names, opaque IDs | Cursor codecs missing |
| 5 | UT-006–UT-007 | `saved_post_response_test.go`, `saved_post_lifecycle_test.go` | Mutation outcomes and fixed clock | Status/timestamp helpers or store result types missing |
| 6 | IT-002 | `internal/api/saved_post_store_test.go` | Real Postgres, two owners, two folders, fixed clock | Save store/upsert missing |
| 7 | IT-003–IT-004 | Store and folder handler tests | Duplicate names, pagination, non-empty folder delete | Folder queries/handlers missing |
| 8 | IT-005 | Store/cursor/handler tests | 103 mixed-scope saves and both sorts | Scoped keyset list missing |
| 9 | UT-009–UT-010 | Response/error tests | Saved/unsaved viewers and foreign IDs | Viewer fields/error translation missing |
| 10 | IT-006 | Saved handler suites | Fake stores, auth context, full request matrix | HTTP handlers/contracts missing |
| 11 | IT-007 | Saved list handler/store | Ordinary/project/quote/comment/reply fixtures | Canonical batch hydration missing |
| 12 | UT-008, IT-008 | `saved_post_policy_test.go`, store test | Mute/block/hide/takedown/member restoration | Policy/context retention missing |
| 13 | UT-005, IT-009 | Lifecycle and index tests | Exact/root/intermediate graph plus membership/session events | Descendant cleanup/owner cascade not wired |
| 14 | IT-010 | Response/store tests | Full pages for Alice/Bob; recording engagement fake | Shared engagement saved state missing |
| 15 | IT-011, AT-010 | Store concurrency test | Controlled transaction barriers, `-race` | Atomic outcomes not implemented/proven |
| 16 | IT-012 | `internal/routes/routes_test.go` | Real mux and route policies | Seven routes/policies missing |
| 17 | UT-011, IT-013 | Observability/privacy tests | Sentinel identifiers and fail-on-call external fakes | Bounded logging/no-side-effect proof missing |
| 18 | IT-014 | Query-plan test | Representative cardinality plus `ANALYZE` | Index/query shape not proven |
| 19 | IT-015 | Existing canonical surface tests | Post/timeline/profile/project/search/thread/notification/quote fixtures | Additive fields absent/inconsistent |
| 20 | AT-001–AT-009 | Handler/store/index acceptance suites | Compose the preceding fixtures into business scenarios | One or more vertical contracts incomplete |
| 21 | REG-001–REG-007 | Existing API/index/lifecycle/migration suites | Pre-feature behavior and unrelated schema snapshots | Regression introduced or not yet asserted |

Focused commands by wave:

```text
# Migration first red/green loop
cd appview && TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' \
  go test ./internal/db -run TestSavedPostsMigration -count=1

# Pure request/cursor/response tests
cd appview && go test ./internal/api -run 'TestSaved(Post|Folder|Cursor)' -count=1

# Real persistence, index, and lifecycle tests
cd appview && TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' \
  go test -race ./internal/api ./internal/index ./internal/relationships ./internal/db \
  -run 'TestSaved|SavedPost|CraftskyPost.*Delete' -count=1

# Route contract
cd appview && go test ./internal/routes -run 'TestSaved|TestV1RoutePolicies|TestAddRoutes' -count=1

# Final gates
just fmt
just test
```

Database tests must fail or run; a skip caused by missing `TEST_DATABASE_URL` is not completion evidence.

## 10. Sequencing And Guardrails

- First TDD step: add `IT-001` only and run it red because `000024_saved_posts` does not exist; then add only the reversible migration needed to make it green.
- Dependencies between work items:
  1. Migration precedes pgx-backed store code.
  2. Request/cursor types precede handler implementations.
  3. Core save/folder store behavior precedes list hydration.
  4. Viewer saved state lands through `EngagementSummaries` before canonical-surface regression updates.
  5. Descendant cleanup lands only after saved tables/queries exist.
  6. Route registration lands after handlers compile, then privacy/query-plan/regression gates close the slice.
- Guardrails:
  - Parse DIDs, record keys, and AT-URIs at HTTP/Tap boundaries; carry typed identifiers internally where signatures semantically represent atproto IDs.
  - Derive owner only from middleware context. No request body, path, cursor, or query parameter may choose another owner.
  - Every saved/folder SQL statement includes an owner predicate or a database constraint that enforces the same boundary.
  - Never log or metric-label a DID, post URI, folder ID/name, owner-target pair, cursor payload, or save timestamp.
  - Never call PDS, Tap admin, push, notification creation, or public interaction code from saved handlers.
  - Never add a second viewer-saved hydration path; extend `EngagementSummaries` and its existing consumers.
  - Never rely on the exact-post foreign key for root/intermediate cleanup.
  - Never delete descendant `craftsky_posts` rows while cleaning saved replies.
  - Keep folder IDs opaque outside storage even if UUID is selected internally.
  - Keep raw SQL parameterized, localized to the owning store or lifecycle transaction, and covered by real-Postgres tests; do not build a generic query abstraction in this slice.
  - Preserve existing worktree changes, especially unrelated `docs/roadmap.md`; do not commit without explicit authorization.
- Out of scope:
  - Flutter UI/state/navigation and client models.
  - Lexicons, PDS records, Tap collection filters, Bluesky import/export, cross-AppView portability.
  - Nested/multiple folders, counts, quotas, notes, sharing, search, manual order, notifications, analytics events.
  - Cursor encryption, post snapshots, background cleanup jobs, load/soak testing, numeric latency SLA.

## 11. Risks And Open Questions

| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Resolved | `AGENTS.md` anticipates sqlc, but the working AppView stores use raw pgx and the user approved that established pattern for this slice. | Introducing code generation here would add unrelated tooling and generated-code churn. | Use localized parameterized raw SQL in `SavedPostStore` and the indexer lifecycle transaction; rely on typed scan helpers plus real-Postgres, query-plan, ownership, and concurrency tests. |
| CPQ-002 | Non-blocking | Composite same-owner folder integrity must coexist with non-destructive folder deletion. | A naive composite `ON DELETE SET NULL` could try to null the owner column. | Use PostgreSQL 16 column-specific `ON DELETE SET NULL (folder_id)` and prove it in IT-001; use an explicit update/delete transaction only if migration execution requires the approved equivalent. |
| CPQ-003 | Non-blocking | Post-policy/context filtering after keyset selection can yield short or empty pages. | Clients may need another request even when a page has no visible items. | Keep bounded `limit + 1` candidate pages and always advance the cursor; never scan unbounded hidden rows in one request. This preserves determinism and privacy. |
| CPQ-004 | Non-blocking | Recursive reply context can contain malformed cycles or missing ancestors in test/corrupt data. | Unbounded recursion or invalid navigation. | Bound traversal by page URI set, track visited URI paths, cap depth, and treat incomplete/cyclic context as temporarily unavailable without deleting the save. |
| CPQ-005 | Non-blocking | A root/ancestor delete must find descendants even if one intermediate row is already missing. | Orphaned saved replies remain. | Seed cleanup from both `reply_parent_uri` and `reply_root_uri`; recurse through present parents; test root and intermediate deletions separately. |
| CPQ-006 | Non-blocking | UUID storage is stricter than the opaque wire contract. | Arbitrary invalid strings could become database syntax errors or leak representation. | Parse only inside the store adapter and map invalid storage-form inputs to the same not-found/no-op outcomes; tests never assert UUID syntax. |
| CPQ-007 | Non-blocking | Existing `EngagementSummaries` already performs several bounded queries rather than one aggregate SQL statement. | “One shared seam” could be misread as “one total query.” | Add exactly one set-based viewer-saved query inside the existing seam; IT-010/IT-014 forbid per-item or per-surface saved queries, not existing bounded engagement queries. |

Blocking questions: None.

## 12. Handoff To TDD Builder

- Coding plan: `04-coding-plan.md`
- TDD execution plan: `05-implementation-plan.md`
- Start with test: `IT-001` in `appview/internal/db/saved_posts_migration_test.go`
- First focused command:

  ```text
  cd appview && TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/db -run TestSavedPostsMigration -count=1
  ```

- Initial red expectation: the migration files and saved tables do not exist.
- First green target: reversible `000024` schema with owner/post uniqueness, composite owner/folder integrity, duplicate-name allowance, exact-target cascade, non-destructive folder deletion, and supporting indexes.
- Store implementation note: use raw parameterized pgx queries and explicit transactions; add no sqlc configuration, generated models, dependencies, or Just recipes.
- Final verification: focused unit/real-Postgres/route tests, `just fmt`, then full `just test` with compose Postgres running.
- No implementation, test source, migration, dependency, commit, push, or PR change is authorized by this document alone.
