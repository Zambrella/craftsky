# Coding Plan: Pinned Profile Posts

## 1. Inputs

- Requirements: `docs/changes/2026-08-04-pinned-profile-posts/01-requirements.md` — status `Reviewed`, no open questions.
- Tests: `docs/changes/2026-08-04-pinned-profile-posts/02-acceptance-tests.md` — 12 acceptance, 9 unit, 16 integration, 8 regression, and 2 manual checks.
- Document review: `docs/changes/2026-08-04-pinned-profile-posts/03-document-review.md` — status `Approved`, no findings or blocking gaps.
- API conventions read before route design:
  - `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md`
  - `docs/superpowers/specs/2026-04-22-api-wire-alignment-design.md`
- Repository context inspected read-only:
  - Private persistence precedent: `appview/migrations/000024_saved_posts.*.sql`, `appview/internal/api/saved_post*.go`, and their database/store/handler/lifecycle/query-plan/observability tests.
  - Profile listing: `appview/internal/api/post.go`, `post_store.go`, `post_response.go`, `envelope/cursor.go`, `appview/internal/routes/{policy,routes}.go`, and profile/language/pagination tests.
  - Indexed lifecycle: `appview/internal/index/craftsky_post.go`, `craftsky_profile.go`, and their tests.
  - Flutter data/state/UI: `PostPage`, `PostApiClient`, `PostRepository`, `ApiPostRepository`, `FakePostRepository`, `UserPosts`, `UserProjects`, `account_operation_guard.dart`, `PostCard`, profile tab slivers, feed, and post-thread root rendering.
  - Flutter non-profile surfaces: search, tag search, general project discovery, saved-post rows, and notification presentation.
- Highest migration found during planning: `000034_scheduled_posts`; the planned migration is `000035_profile_pins`, but implementation must re-check immediately before creating it.

## 2. Implementation Strategy

Implement the approved dedicated AppView state approach as vertical TDD slices:

1. Add a reversible private `profile_pins` table with one occupied row per owner and slot, referential cleanup, a target lookup index, and an opaque state token used only to bind profile cursors to the selected pin state.
2. Add pure slot classification and a transactional `ProfilePinStore` that validates new pins, serializes owner mutations, atomically replaces a slot, preserves same-target idempotency, makes unpin target-specific, and returns the authoritative two-slot state.
3. Register the exact bodyless `GET`/`PUT`/`DELETE` contracts under existing `/v1/` middleware. Preserve the deliberate feature-specific `200 OK` authoritative-body exception; do not convert it to generic full-body `PUT` or empty `DELETE 204` behavior.
4. Extend only the two profile-list read paths with pin-aware page results. Select and policy-filter the visible pin before the limit, exclude it from the chronological remainder, and bind later-page cursors to an opaque state token rather than a post URI or timestamp.
5. Clear pins through foreign-key lifecycle for permanent post/profile deletion and through the post indexer transaction when an update changes the target's slot/top-level classification. Retain rows through temporary moderation, block, or language filtering.
6. Add Flutter page and two-slot models, exact client/repository calls, and an account-family Riverpod controller whose confirmed state and pending set are scoped by account and slot. Mutations remain server-confirmed and refresh only the affected live profile-list families.
7. Add explicit surface opt-in for the owner menu action and explicit page-level opt-in for the profile annotation. This keeps the shared canonical `Post` model and every non-profile surface free of pin presentation state.

This fits the current codebase: AppView uses numbered migrations, parameterized pgx stores, narrow handler interfaces, stdlib routing, centralized route policies, typed atproto boundary parsing, and opaque cursor helpers. Flutter already uses Dio clients behind repository interfaces, generated `dart_mappable` models, generated Riverpod notifiers, active-account leases, family caches, explicit `PostCard` callbacks, and localized context-menu copy.

## 3. Affected Areas

| Area | Existing Pattern | Planned Change | Requirement IDs | Test IDs |
|---|---|---|---|---|
| AppView persistence | Reversible numbered migrations; private saved state references `craftsky_profiles` and `craftsky_posts` | Add private `(owner_did, slot)` pin rows, structural constraints, lifecycle FKs, opaque state token, timestamps, and target index | FR-002, FR-005, FR-012, NFR-001, NFR-006, RULE-001 | IT-001–IT-003, IT-010–IT-013, REG-008 |
| Pin policy/store | Inline parameterized pgx stores with explicit transactions and typed identifiers | Add pure slot classifier plus serialized pin/replace/conditional-unpin/read methods returning both slots | FR-002–FR-004, FR-010, NFR-001, RULE-001, RULE-002 | UT-008, IT-002–IT-005, IT-011 |
| AppView API/routes | Narrow handlers, `/v1/` route policy registry, current-member middleware, standard envelopes | Add private read and bodyless target-specific mutations, all returning authoritative `200` bodies | FR-003, FR-004, FR-010, FR-013 | AT-001–AT-004, AT-006, AT-010, IT-004, IT-005, IT-009 |
| Profile pagination | Profile Posts/Projects query by `(profile_sort_at DESC, uri DESC)` with opaque cursors and policy/language predicates before `LIMIT` | Return pin-aware page rows, first-page `pinnedPostUri`, exact omission, pin-consuming limits, exclusion from remainder, and state-bound cursor validation | FR-006, FR-007, FR-011, NFR-002 | AT-005–AT-007, IT-006–IT-008, IT-013, REG-001, REG-003, REG-007 |
| Indexed lifecycle | Post/profile deletes are transactional; post upsert owns final structural fields | Cascade permanent deletes and explicitly remove structurally invalid pins after post upsert/materialization in the same indexer transaction | FR-005, FR-012 | AT-011, IT-010, REG-008 |
| Privacy/observability | Redacted structured logs, HTTP/DB metrics, bounded domain metrics | Add bounded pin operation metrics/log fields only; add no PDS, Tap, lexicon, notification, or ranking dependency | BR-003, FR-005, NFR-003, NFR-005, RULE-003 | AT-012, IT-012, IT-014, REG-004–REG-006 |
| Flutter wire/repository | `PostPage` plus `PostApiClient`/`PostRepository`/`ApiPostRepository` | Add page metadata and authoritative two-slot model; add exact GET/bodyless PUT/bodyless DELETE calls | FR-006, FR-013 | UT-001, UT-002, UT-009, IT-015 |
| Flutter state | Generated Riverpod providers and active-account operation leases | Add account-family confirmed pin state with per-slot pending ownership, stale-completion fencing, affected-list refresh, and `invalid_cursor` restart | FR-007, FR-008, FR-012, NFR-001 | AT-003, AT-004, AT-007–AT-010, UT-003, UT-004, UT-006, IT-011, IT-016 |
| Flutter menus/presentation | Shared `PostCard`, optional action callbacks, repost attribution slot, surface-level callback wiring | Add explicit eligible-surface pin action and explicit profile-only annotation; preserve default-off behavior everywhere else | BR-001, BR-002, FR-001, FR-009, NFR-004, RULE-004 | AT-001–AT-006, AT-008, AT-012, UT-005, UT-007, REG-002 |
| Localization/codegen | English ARB plus committed generated l10n, mapper, and Riverpod files | Add exact action/feedback/annotation copy and regenerate only related outputs | FR-001, FR-008, FR-009, NFR-004 | UT-003, UT-005, UT-007, MAN-001, MAN-002 |

## 4. Files And Modules

### 4.1 AppView

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `appview/migrations/000035_profile_pins.up.sql` | Create | Create private pin relation, constraints, FKs, opaque state-token column, timestamps, and post-target index. Re-check migration number first. | FR-005, FR-012, NFR-001, NFR-006, RULE-001 | IT-001, IT-010, IT-013, REG-008 |
| `appview/migrations/000035_profile_pins.down.sql` | Create | Drop the pin relation cleanly without changing posts, chronology, or saved-post state. | FR-005, NFR-006 | IT-001, REG-008 |
| `appview/internal/db/profile_pins_migration_test.go` | Create | Prove up/down/up, constraints, indexes, cascade cleanup, state-token/timestamp shape, and unrelated-state preservation. | FR-005, FR-012, NFR-006 | IT-001, REG-008 |
| `appview/internal/api/profile_pin_policy.go` | Create | Define `standard`/`project` slot vocabulary and pure target classification/validity decisions. | FR-004, RULE-001, RULE-002 | UT-008, IT-004 |
| `appview/internal/api/profile_pin_store.go` | Create | Own transactional read, pin/replace, target-specific unpin, owner serialization, authoritative state, and mutation classification. | FR-002–FR-005, FR-010, NFR-001 | IT-002–IT-004, IT-011, IT-012 |
| `appview/internal/api/profile_pin_cursor.go` | Create | Encode/decode profile traversal cursor with slot, chronological boundary, and opaque pin state token; reject malformed/mismatched/currently stale state. | FR-007, NFR-002, NFR-003 | IT-007, IT-013, REG-001 |
| `appview/internal/api/profile_pin.go` | Create | Define response DTO/interfaces and implement private read plus pin/unpin handlers and exact error mappings. | FR-003, FR-004, FR-013 | AT-004, AT-006, IT-004, IT-005 |
| `appview/internal/api/profile_pin_store_test.go` | Create | Real-Postgres independent slots, idempotency, atomic replacement, concurrency barriers, stale unpin, owner isolation, and authoritative response tests. | BR-001, FR-002, FR-003, FR-010, NFR-001, RULE-001 | IT-002, IT-003, IT-011 |
| `appview/internal/api/profile_pin_policy_test.go` | Create | Pure classifier plus viewer-policy and slot-mismatch tests. | FR-004, FR-006, FR-011, RULE-002 | UT-008, IT-008 |
| `appview/internal/api/profile_pin_test.go` | Create | Handler authorization, exact status/code/body, universal current-member behavior, hidden-target pin rejection, retained-pin removal, and no-body contract tests. | FR-003, FR-004, FR-010, FR-013 | AT-006, AT-010, IT-004, IT-005 |
| `appview/internal/api/post_store.go` | Change | Add pin-aware standard/project page queries/results while retaining current chronological helpers for non-profile callers/tests where useful. | FR-006, FR-007, FR-011, NFR-002 | IT-006–IT-008, IT-013, REG-001, REG-003, REG-007 |
| `appview/internal/api/post.go` | Change | Make profile Posts/Projects handlers consume the pin-aware result and emit optional page-level `pinnedPostUri`; keep `PostResponse` unchanged. | FR-006, FR-007, FR-011, FR-013 | AT-005–AT-007, IT-006–IT-008, REG-004 |
| `appview/internal/api/profile_pin_pagination_test.go` | Create | Limits 1/2/10, older pin, tied timestamps, full traversal, exact omission, replacement/clear invalidation, and no duplicates/gaps. | FR-006, FR-007, NFR-002 | AT-005, AT-007, IT-006, IT-007, REG-001 |
| `appview/internal/api/profile_pin_lifecycle_test.go` | Create | Permanent target/member deletion and read-time structural defence, with unaffected-slot preservation. | FR-005, FR-012 | AT-011, IT-010 |
| `appview/internal/api/profile_pin_privacy_test.go` | Create | Assert database-only side effects and no private selection in unrelated/public response surfaces. | BR-003, FR-005, NFR-003, RULE-003 | IT-012, REG-004, REG-006 |
| `appview/internal/api/profile_pin_query_plan_test.go` | Create | EXPLAIN owner/slot and target indexes; fixed query count and set-based list hydration. | NFR-002 | IT-013 |
| `appview/internal/api/profile_pin_observability_test.go` | Create | Assert bounded operation/slot/result/error-class dimensions and identifier/timestamp/state redaction. | NFR-003, NFR-005 | IT-014 |
| `appview/internal/index/craftsky_post.go` | Change | After a post upsert/materialization, delete only a pin whose stored slot no longer matches the final top-level/type structure; do not move it. | FR-012 | AT-011, IT-010 |
| `appview/internal/index/craftsky_post_test.go` | Change | Cover standard/project/reply transitions and unaffected slot behavior. | FR-012, RULE-002 | AT-011, IT-010 |
| `appview/internal/routes/policy.go` | Change | Add all three exact route policies as authenticated, current-member, no-body read/write classes. | FR-010, FR-013 | IT-009 |
| `appview/internal/routes/routes.go` | Change | Construct the pin store and register `GET /v1/profiles/me/pins`, `PUT /v1/posts/{did}/{rkey}/pin`, and `DELETE /v1/posts/{did}/{rkey}/pin`; inject it into profile list handlers. | FR-013 | IT-005, IT-009 |
| `appview/internal/routes/routes_test.go` | Change | Protect registration, middleware, current membership, rate class, no-body rejection, and standard envelope behavior. | FR-010, FR-013 | IT-009 |
| `appview/internal/observability/profile_pin.go` | Create | Expose one bounded domain observation method accepting operation, slot, result, error class, and duration only. | NFR-003, NFR-005 | IT-014 |
| `appview/internal/observability/metric_recorder.go` and validation/tests | Change | Record/validate the bounded profile-pin counter/distribution for noop, in-memory, and Sentry recorders. | NFR-003, NFR-005 | IT-014 |
| Existing `post_response.go` and response tests | Test-only guard | Do not add `isPinned`, pin timestamp, opaque token, or two-slot state to canonical post JSON. | FR-006, FR-013, NFR-003 | REG-004 |

### 4.2 Flutter

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `app/lib/feed/models/profile_pin_state.dart` + generated mapper | Create | Model only nullable `standardPostUri`/`projectPostUri`, `ProfilePinSlot`, slot selection, and pure target classification helpers. | FR-004, FR-010, FR-013, RULE-002 | UT-002, UT-005 |
| `app/lib/feed/models/post_page.dart` + generated mapper | Change | Add optional page-level `pinnedPostUri` with absent/null-tolerant decode and omission on encode. | FR-006, FR-013 | UT-001, REG-004 |
| `app/lib/feed/data/post_api_client.dart` | Change | Add exact private read and bodyless target-specific PUT/DELETE calls returning `ProfilePinState`. | FR-003, FR-013 | UT-009, IT-015 |
| `app/lib/feed/data/post_repository.dart`, `api_post_repository.dart` | Change | Expose `profilePins`, `pin`, and `unpin` through the existing repository boundary. | FR-003, FR-013 | IT-015, IT-016 |
| `app/test/feed/fakes/fake_post_repository.dart` | Change | Add callbacks/counters/deferred futures for pin state and mutations; retain loud failure for unstubbed calls. | FR-008, NFR-001 | UT-003, UT-004, UT-006, IT-016 |
| `app/lib/feed/providers/profile_pins_provider.dart` + generated provider | Create | Account-family authoritative pin state, per-slot pending set, same-slot suppression, active-account lease fencing, affected-profile refresh, and mutation outcomes. | FR-008, FR-012, NFR-001 | AT-003, AT-004, AT-008–AT-010, UT-003, UT-006, IT-011, IT-016 |
| `app/lib/feed/providers/user_posts_provider.dart` | Change | Retain first-page pin metadata in accumulated state and restart from page one exactly once on `ApiBadRequest(code: invalid_cursor)`. | FR-006, FR-007, FR-008 | AT-007, UT-004, IT-016 |
| `app/lib/projects/providers/user_projects_provider.dart` | Change | Mirror standard-post metadata/restart behavior for project profiles. | FR-006, FR-007, FR-008 | AT-007, UT-004, IT-016 |
| `app/lib/feed/models/user_posts_state.dart`, `app/lib/projects/models/user_projects_state.dart` + generated mappers | Change | Carry the visible first-page `pinnedPostUri` independently of canonical post items across later-page appends. | FR-006 | AT-005, UT-001 |
| `app/lib/auth/providers/account_boundary_provider.dart` | Change | Invalidate pin repository/controller state on account boundary so pending state cannot survive as another active account. | FR-008, FR-012, NFR-001 | AT-009, UT-006, IT-011 |
| `app/lib/feed/widgets/post_card.dart` | Change | Add explicit `allowProfilePinAction` and `showPinnedProfileAttribution` inputs; derive correct owner action from account pin state; render exact non-interactive pinned row before card header. | FR-001, FR-008, FR-009, NFR-004 | AT-004–AT-006, AT-008, UT-005, UT-007, REG-002 |
| `app/lib/profile/widgets/profile_tabs/profile_post_feed_slivers.dart` | Change | Opt into pin actions for qualifying owner cards and pass annotation only when `post.uri == pinnedPostUri`. | FR-001, FR-006, FR-009 | AT-001, AT-005 |
| `app/lib/profile/widgets/profile_tabs/profile_posts_tab.dart` | Change | Pass standard page pin metadata and listen/show exact pin mutation feedback. | BR-001, FR-008 | AT-001, AT-003, AT-005, AT-008 |
| `app/lib/profile/widgets/profile_tabs/profile_projects_tab.dart` | Change | Pass project page pin metadata and listen/show exact pin mutation feedback. | BR-001, FR-008 | AT-002, AT-005, AT-008 |
| `app/lib/feed/pages/feed_page.dart` | Change | Opt timeline cards into pin menu actions; do not opt into pinned attribution or ordering. | FR-001, FR-009, RULE-003 | AT-002, AT-012, REG-002, REG-006 |
| `app/lib/feed/pages/post_thread_page.dart` | Change | Opt only the root/top-level card into pin menu actions; leave comment and nested-reply cards default-off; never show pinned attribution. | FR-001, FR-009, RULE-002, RULE-003 | AT-002, AT-012, REG-002 |
| Search, tag-search, project-discovery, saved-post, and notification widgets/tests | Test-only/default guard | Keep the new `PostCard` inputs default-off and do not add pin-state reads or presentation. | FR-001, FR-009, RULE-003 | AT-012, REG-002, REG-006 |
| `app/lib/l10n/app_en.arb`, `app/lib/l10n/generated/*` | Change/regenerate | Add exact `Pin post`, `Unpin post`, `Pinned post`, success, and retry copy plus accessibility descriptions. | FR-001, FR-008, FR-009, NFR-004 | UT-003, UT-005, UT-007, MAN-001, MAN-002 |
| Flutter test targets named in `02-acceptance-tests.md` | Create/change | Add model, client, provider, profile-tab, `PostCard`, feed, thread, search, projects, saved-post, notification, and account-boundary coverage. | All Flutter-linked requirements | AT-001–AT-012, UT-001–UT-007, UT-009, IT-011, IT-015, IT-016, REG-001–REG-007 |

Generated files are committed in this repository. Implementation must run `flutter gen-l10n` after ARB edits and `dart run build_runner build --delete-conflicting-outputs` after mapper/provider edits, then inspect the generated diff for unrelated churn.

## 5. Services, Interfaces, And Data Flow

### 5.1 Private persistence shape

Use AppView Postgres only. Do not add or change a PDS record, lexicon, Tap indexer registration, public post column, or public interaction.

```text
profile_pins
- owner_did   TEXT NOT NULL
    -> craftsky_profiles(did) ON DELETE CASCADE
- slot        TEXT NOT NULL CHECK IN ('standard', 'project')
- post_uri    TEXT NOT NULL
    -> craftsky_posts(uri) ON DELETE CASCADE
- state_token UUID NOT NULL
- created_at  TIMESTAMPTZ NOT NULL
- updated_at  TIMESTAMPTZ NOT NULL

PRIMARY KEY (owner_did, slot)
INDEX (post_uri) for target-FK delete/update lifecycle work
```

Rules:

- An empty slot has no row.
- `state_token` is generated by the AppView on creation/replacement. It remains unchanged for same-target idempotent pinning and is never returned by the pin-state API or profile page metadata.
- `created_at`, `updated_at`, and `state_token` are internal only.
- The owner FK clears both slots when current Craftsky membership is permanently removed.
- The target FK clears the affected slot when the indexed post is permanently deleted.
- Do not add a unique `post_uri` constraint: structural validation already prevents a valid post occupying both slot kinds, and `(owner_did, slot)` is the capacity authority.
- Follow the repository's established localized, parameterized pgx query pattern for this feature. `appview/queries/` still contains no adopted sqlc query surface; do not introduce sqlc tooling as incidental scope.

### 5.2 Domain and store interfaces

Illustrative signatures only:

```text
enum ProfilePinSlot { standard, project }

ProfilePinState:
- StandardPostURI optional syntax.ATURI
- ProjectPostURI optional syntax.ATURI

ProfilePinMutationResult:
- State ProfilePinState
- Slot ProfilePinSlot
- Operation one of pin, replace, unpin, noop

ProfilePinStore:
- Read(ctx, ownerDID) -> ProfilePinState
- Pin(ctx, ownerDID, targetDID, targetRkey) -> ProfilePinMutationResult
- Unpin(ctx, ownerDID, targetURI) -> ProfilePinMutationResult
- ListProfilePage(ctx, viewerDID, authorDID, languages, slot, limit, cursor)
    -> rows, nextCursor, optional visiblePinnedPostURI
```

Pin flow:

1. The handler parses `{did}` and `{rkey}` once as typed atproto identifiers and obtains the authenticated owner DID from middleware context.
2. If target DID differs from owner DID, return `403 forbidden` before mutation.
3. In a transaction, lock/serialize pin mutations for the owner. Owner-wide serialization keeps the two-slot response stable through commit while still allowing Flutter to initiate an independent-slot request.
4. Read the current visible indexed target under row locking. Missing/deleted/moderation-hidden returns `404 post_not_found`.
5. Classify final target structure:
   - `standard`: top-level `is_project=false`, including quote posts.
   - `project`: top-level `is_project=true`, with no quote.
   - replies/comments or inconsistent structure: `422 pin_not_allowed`.
6. Insert the empty slot or replace its `post_uri`, `state_token`, and `updated_at` atomically. If it already targets the same URI, do not rewrite token or timestamps.
7. Read both owner slots in the same serialized transaction, commit, and return the authoritative state.

Unpin flow:

1. Parse the target path and construct the canonical post URI; do not require the target to remain visible or even indexed.
2. Serialize for the owner and delete only `WHERE owner_did = owner AND post_uri = targetURI`.
3. A stale/missing target is an idempotent no-op. It cannot clear a newer URI in the same slot.
4. Read both slots, commit, and return `200` authoritative state.

Concurrency tests use advisory-lock/transaction barriers or explicit connection locks, never sleeps. The serialized final mutation determines the last committed winner.

### 5.3 Pin-aware profile cursor and page query

The existing generic seek cursor contains `indexedAt` and `uri`. Profile Posts/Projects need a feature-specific cursor with these logical fields:

```text
kind: profilePosts or profileProjects
pinStateToken: opaque random UUID from occupied row, or fixed empty marker
afterIndexedAt: optional chronological boundary
afterUri: optional chronological boundary
```

Guardrails:

- Never encode `post_uri`, owner DID, pin timestamps, or both-slot state into the cursor.
- The random state token makes a hidden pin's identity non-derivable from the cursor.
- First page reads the current slot token even when the pin is not visible to this viewer. Later pages compare the cursor token with the current stored token before returning rows.
- A replacement, clear, target cascade, or structural cleanup changes occupied/empty token state and makes the prior cursor `400 invalid_cursor`.
- Same-target idempotent pin leaves the token unchanged and does not invalidate a valid traversal.
- Returning to the same empty value after an intervening pin is treated as the same current empty selection; the chronological result is again equivalent and remains safe to traverse.

Page-one query/result algorithm:

1. Read the owner/slot pin row and its token in a bounded query/CTE.
2. Join the pinned post through the same slot, top-level, moderation, block/membership, and language predicates used by the corresponding profile list.
3. If visible, return it first, set `pinnedPostUri`, and request `limit - 1` chronological rows. If not visible, omit metadata and request the full limit.
4. Exclude the stored pinned URI from the chronological remainder whether or not it would otherwise fall inside that page.
5. Keep the remainder ordered by `(profile_sort_at DESC, uri DESC)`.
6. Build the next cursor from the last chronological row. For `limit=1` with a visible pin and more chronology, encode an explicit “before first chronological row” boundary so page two begins at the chronological head while still excluding the pin.

Later-page algorithm:

1. Decode the correct list kind and chronological boundary.
2. Compare its token with current slot token; mismatch is `envelope.ErrInvalidCursor`.
3. Do not promote the pin and always omit `pinnedPostUri`.
4. Continue chronological selection while excluding the current pin URI, preserving limits and uniqueness.

Use fixed-count, set-based queries/CTEs. Engagement and quote hydration remains batch-based. Do not look up pin state per returned card.

### 5.4 API contracts and errors

All routes use existing authenticated/device/current-member middleware, camelCase JSON, standard request IDs, and route rate/body policy:

```text
GET /v1/profiles/me/pins
-> 200 { standardPostUri: string|null, projectPostUri: string|null }

PUT /v1/posts/{did}/{rkey}/pin
-> request body forbidden
-> 200 authoritative ProfilePinState

DELETE /v1/posts/{did}/{rkey}/pin
-> request body forbidden
-> 200 authoritative ProfilePinState
```

The `200` mutation body is a documented feature-specific exception to the API architecture's generic full-replace `PUT` and empty `DELETE 204` examples. Route-policy middleware rejects any body before handler work. Do not add a compatibility `POST`, a body DTO, or a `204` path.

Error mapping:

| Case | Status/code | Mutation effect |
|---|---|---|
| Invalid DID or rkey path | Existing `400 invalid_identifier` envelope | None |
| Another author's target | `403 forbidden` | None |
| Missing/deleted/currently hidden new-pin target | `404 post_not_found` | None |
| Reply/comment or structurally invalid target | `422 pin_not_allowed` | None |
| Malformed/stale profile cursor | `400 invalid_cursor` | None |
| Unexpected store failure | `500 internal_error` | Transaction rolled back |
| Stale target-specific unpin | `200` with current two-slot state | Newer target retained |

### 5.5 Lifecycle and privacy boundary

- Post deletion: `ON DELETE CASCADE` clears only the target row.
- Membership removal: the owner FK to `craftsky_profiles` clears both owner slots. This matches current membership representation; there is no separate membership table.
- Structural post update: after the indexer has written the final `craftsky_posts`/project materialization state, delete a matching pin when top-level or slot classification no longer matches. Keep this in the same transaction as the index update.
- Read-time defence: pin reads and profile promotion join against current target structure so a racing stale row is never surfaced even before cleanup completes.
- Temporary moderation/block/language filtering: do not mutate `profile_pins`; simply omit promotion/metadata for that viewer.
- No PDS client, OAuth/PDS token, lexicon, Tap event, notification service, engagement counter, search rank, timeline rank, or project discovery query participates in a mutation.
- No response to another owner exposes the private two-slot selection. Allowed profile visitors intentionally observe only the visible ordering and `Pinned post` annotation.

### 5.6 Observability

Add one bounded domain observation path for mutations:

```text
operation: pin | replace | unpin
slot: standard | project
result: success | noop | rejected | error
errorClass: none | forbidden | not_found | policy | store
duration: bounded distribution
```

Do not accept or record DID, handle, rkey, URI, post text, state token, pin timestamp, or two-slot state. Existing HTTP metrics continue to cover the read endpoint and status classes. Structured handler logs use existing run/request ID plus the same bounded values.

## 6. State, Providers, Controllers, Or DI

### 6.1 AppView DI

- Construct `ProfilePinStore` beside `PostStore` in `routes.go` using the shared pool and observer.
- Keep handler dependencies narrow (`ProfilePinReader`, `ProfilePinMutator`, pin-aware page reader); never pass the full `app.Deps` into a handler.
- `PostStore` may own the pin-aware page SQL because it already owns moderation/language predicates and batch post hydration. `ProfilePinStore` remains the mutation/private-state authority.
- Route registration injects the same pin state reader into standard/project profile handlers so cursor validation and mutation state have one source of truth.

### 6.2 Flutter provider graph

Use generated Riverpod and the existing `AccountKey`/active lease boundary:

```text
dioProvider
  -> postApiClientProvider
      -> postRepositoryProvider
          -> profilePinsProvider(AccountKey)
              -> PostCard menu selectors/actions
              -> targeted userPostsProvider/userProjectsProvider invalidation

postRepositoryProvider
  -> userPostsProvider(handleOrDid)
      -> UserPostsState(items, cursor, pinnedPostUri)
  -> userProjectsProvider(handleOrDid)
      -> UserProjectsState(items, cursor, pinnedPostUri)
```

Suggested state shape:

```text
ProfilePinsPresentation:
- confirmed ProfilePinState
- pendingSlots Set<ProfilePinSlot>

ProfilePinMutationOutcome:
- pinned | unpinned | pinFailed | unpinFailed | staleCompletion
```

Controller behavior:

1. `profilePinsProvider(account)` loads `GET /v1/profiles/me/pins` and exposes no guessed current-pin state before that read succeeds.
2. A qualifying `PostCard` derives its slot from `Post` structure and compares the target URI with confirmed state to choose `Pin post` or `Unpin post`.
3. Before the first await, capture the active-account lease. If the slot is already in `pendingSlots`, suppress the duplicate request. The other slot remains enabled.
4. Add only that slot to `pendingSlots`; do not alter confirmed URIs or profile-list order.
5. Call repository pin/unpin. On a current completion, replace the complete confirmed two-slot state and clear the slot. On failure, clear the slot and preserve confirmed state.
6. On success, refresh only live standard or project profile families for the post author's DID/handle IDs using existing `authorPostCacheIds`/`ref.exists` patterns. Do not refresh timeline, search, discovery, notifications, or the independent profile list.
7. Return a bounded mutation outcome to the calling widget. The widget shows the exact localized success/error text only if the initiating lease is still current and its context remains mounted.
8. If the lease is stale, publish no state, cache invalidation, or feedback into the new active account. Account-boundary invalidation removes pending controller state; account A reloads authoritative server state when active again.

Profile pagination behavior:

- Initial build copies page `pinnedPostUri` into provider state.
- Normal load-more appends items, advances cursor, and retains the first-page `pinnedPostUri` even though later pages omit it.
- On `ApiBadRequest.code == 'invalid_cursor'`, discard all accumulated items/cursor/metadata, make one request without a cursor, and publish only the restarted page after the active-account lease check.
- Other load-more errors retain existing items/cursor/metadata and use the current retry UI.

## 7. UI, Widgets, Routes, Or User-Facing Surfaces

### 7.1 Context-menu action

`PostCard` keeps the new surface permission explicit and default-off.

Eligibility requires all of:

- authenticated viewer owns `post.author.did`;
- caller opted the card into owner pin actions;
- target is top-level (`reply == null`);
- target classifies as standard/quote or project;
- authoritative pin state has loaded for the active account.

Opt-in matrix:

| Surface | Pin menu | Pinned annotation/ranking |
|---|---|---|
| Own profile Posts tab | Yes | Yes when page metadata identifies card |
| Own profile Projects tab | Yes | Yes when page metadata identifies card |
| Timeline owner card | Yes | No |
| Top-level thread root owner card | Yes | No |
| Thread comment or nested reply | No | No |
| Search/tag search | No | No |
| General project discovery | No | No |
| Notifications | No | No |
| Saved posts | No | No |

Menu behavior:

- Current target: `Unpin post` with an appropriate outlined pin/removal icon.
- Other eligible target: `Pin post` with pin icon.
- No confirmation dialog for replacement.
- While a slot is pending, every visible action for that active account/slot has a null `onPressed`/disabled semantics; the other slot remains enabled.
- Relationship/report/delete menu behavior and grouping remain unchanged.

### 7.2 Profile-only pinned attribution

Keep presentation explicit instead of adding `isPinned` to `Post`:

```text
ProfilePostFeedSlivers
  compares post.uri with page-level pinnedPostUri
    -> PostCard(showPinnedProfileAttribution: true)
```

In `PostCard`, render one attribution slot above the normal author row:

- If `showPinnedProfileAttribution`, render the pin row.
- Otherwise preserve existing repost attribution behavior.
- Use the repost row's 16px icon scale, 8px gap, label typography, outline/subdued colour, spacing, and width behavior.
- Visible and semantic text is exactly `Pinned post`.
- Use no gesture detector, button semantics, tooltip action, or author navigation around the pinned row.
- Let the text use available width/wrapping/ellipsis behavior that remains readable at supported large text scales without hiding the card menu.

### 7.3 Feedback and navigation

- No new Flutter route, page, or management screen.
- Successful pin: `Post pinned`.
- Successful unpin: `Post unpinned`.
- Failed pin: `Couldn’t pin post. Try again.`
- Failed unpin: `Couldn’t unpin post. Try again.`
- Use the existing context messenger/snackbar pattern after the provider returns its fenced outcome.
- Do not reorder a visible list until the refreshed AppView page confirms the new order.

## 8. Error, Loading, Empty, And Edge States

| State / Case | Planned Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| Both slots empty | Private read returns both nullable fields as null; profile page is chronological and omits `pinnedPostUri`. | RULE-001, FR-013 | UT-002, IT-002, REG-001 |
| Pin state initial load pending/error | Do not guess Pin versus Unpin; expose no enabled pin mutation until authoritative state is available. Existing card/menu remains otherwise usable. | FR-001, FR-008 | UT-003, UT-005 |
| Same target pinned again | Return `200`, preserve one row/token/timestamps, reconcile same state, and show normal pin success. | FR-002, FR-008 | IT-002, IT-005 |
| Occupied slot replacement | Serialize and atomically replace row/token; no empty/two-row committed state and no confirmation UI. | FR-002, NFR-001 | AT-003, IT-002, IT-003 |
| Stale unpin after replacement | Exact URI delete matches nothing; return current state and retain winner. | FR-003, NFR-001 | AT-004, IT-003, IT-005 |
| Retained pin currently hidden | New pin lookup rejects hidden targets, but target-specific unpin bypasses visibility and succeeds. | FR-004, FR-011 | AT-006, IT-004, IT-008 |
| Another author's target | Reject `403 forbidden` before mutation; do not disclose private state beyond the authenticated owner's normal response path. | FR-004, NFR-003 | AT-006, IT-004 |
| Reply/comment/mismatched target | Hide action in Flutter and reject direct call as `422 pin_not_allowed`. | FR-001, FR-004, RULE-002 | AT-006, UT-005, UT-008, IT-004 |
| Visible pin with limit 1 | Return only the pin plus a cursor capable of starting chronology on page two; never exceed limit. | FR-006, FR-007 | IT-006, IT-007 |
| Pin older than original page one | Promote once and exclude it from its chronological position on later pages. | FR-006, FR-007 | AT-005, AT-007, IT-006, IT-007 |
| Pin hidden for one viewer | Omit promotion/metadata, fill page chronologically, retain private row/token. | FR-011 | AT-006, IT-008, REG-003 |
| Pin changes during traversal | Server returns `400 invalid_cursor`; Flutter discards stale traversal and restarts once. | FR-007 | AT-007, UT-004, IT-007, IT-016 |
| Mutation pending | Disable all same-account/same-slot actions; leave independent slot usable; preserve confirmed menus/order. | FR-008 | AT-003, AT-004, AT-008, UT-003, UT-005 |
| Mutation failure | Restore affected slot actions, retain confirmed state/order, and show exact retry copy. | FR-008 | AT-008, UT-003 |
| Active account changes in flight | Lease check suppresses state, cache, and feedback completion for new account; boundary invalidates pending provider. | FR-008, FR-012, NFR-001 | AT-009, UT-006, IT-011 |
| Target/member permanent deletion | FK clears only affected target or both owner rows as applicable; no placeholder/move. | FR-012 | AT-011, IT-010 |
| Structural type/top-level update | Indexer deletes affected pin in same transaction; read path fails closed; other slot unchanged. | FR-012 | AT-011, IT-010 |
| Temporary moderation/language change | Keep pin row and token; promotion reappears when policy allows. | FR-011, FR-012 | AT-011, IT-008 |
| Narrow/large-text card | Reuse flexible attribution row and verify readable semantics/layout; physical device checks remain manual. | NFR-004 | UT-007, MAN-001, MAN-002 |

## 9. Test Implementation Plan

The order mirrors the approved handoff: persistence, policy/store, wire contracts, profile traversal, lifecycle/privacy/telemetry, Flutter data/state, then UI/workflow/regressions. Each automated ID remains a separate red-green-refactor loop even when it shares a target file.

| Order | Test ID | Target | Setup / Fixture | Initial Expected Failure |
|---:|---|---|---|---|
| 1 | IT-001 | `appview/internal/db/profile_pins_migration_test.go` | Pre-`000035` membership/posts plus unrelated saved/chronology rows; up/down/up. | Migration files/table/constraints/indexes absent. |
| 2 | UT-008 | `appview/internal/api/profile_pin_policy_test.go` | Standard, quote, project, reply, nested reply, mismatched structures. | Slot classifier absent. |
| 3 | IT-002 | `appview/internal/api/profile_pin_store_test.go` | Empty slots, two targets per slot, same-target retry. | Store and atomic upsert absent. |
| 4 | IT-003 | `appview/internal/api/profile_pin_store_test.go` | Explicit transaction barriers for A/B replacement and stale A delete. | Serialization/conditional delete absent. |
| 5 | IT-004 | `appview/internal/api/profile_pin_test.go` | Current/non-member, other author, missing/hidden/invalid/valid targets and retained hidden pin. | Authorization/error policy absent. |
| 6 | IT-005 | `appview/internal/api/profile_pin_test.go` | GET/bodyless PUT/DELETE, body rejection, empty/occupied/noop/error state. | Handlers and exact `200` body absent. |
| 7 | IT-009 | `appview/internal/routes/routes_test.go` | Auth/device/current-member/body/rate policy matrix for three paths. | Route policies/registration absent. |
| 8 | UT-002 | `app/test/feed/models/profile_pin_state_test.dart` | Authoritative states with both, one, and no URI. | Flutter state model absent. |
| 9 | IT-015 | `app/test/feed/data/post_api_client_test.dart` | Mock Dio exact methods/paths/no-body/`200` states/errors. | Client/repository methods absent. |
| 10 | UT-009 | `app/test/feed/data/post_api_client_test.dart` | 400/403/404/422/500 envelopes including `invalid_cursor`. | Pin endpoint mapping assertions absent. |
| 11 | IT-006 | `appview/internal/api/profile_pin_pagination_test.go` | Older visible pin, no/hidden pin, limits 1/2/10 for both slots. | Promotion/metadata/limit integration absent. |
| 12 | IT-007 | `appview/internal/api/profile_pin_pagination_test.go` | 23 posts, tied timestamps, fixed token, replace/clear before later page. | Pin-aware cursor/traversal absent. |
| 13 | IT-008 | `appview/internal/api/profile_pin_policy_test.go` | Allowed/blocked/moderated/membership/language viewers plus reversal. | Policy-before-promotion behavior absent. |
| 14 | IT-013 | `appview/internal/api/profile_pin_query_plan_test.go` | Representative profile size, installed indexes, EXPLAIN/call counts. | Indexed bounded query proof absent. |
| 15 | IT-010 | `appview/internal/api/profile_pin_lifecycle_test.go`, index test | Both slots plus target/member delete and structural updates. | Lifecycle cleanup absent. |
| 16 | IT-011 | AppView store + Flutter provider tests | Two owners, session/device/account reload boundaries. | Cross-account persistence/isolation coverage absent. |
| 17 | IT-012 | `appview/internal/api/profile_pin_privacy_test.go` | PDS/Tap/notification/ranking sentinels around mutations. | Privacy side-effect sentinel absent. |
| 18 | IT-014 | `appview/internal/api/profile_pin_observability_test.go` | Success/noop/rejection/error with identifier canaries. | Bounded/redacted pin telemetry absent. |
| 19 | UT-001 | `app/test/feed/models/post_page_test.dart` | Present/absent/null `pinnedPostUri` and re-encode; canonical Post JSON. | Page field/omission behavior absent. |
| 20 | UT-003 | `app/test/feed/providers/profile_pins_provider_test.dart` | Deferred success/failure across standard/project actions. | Confirmed/pending/outcome controller absent. |
| 21 | UT-006 | `app/test/feed/providers/profile_pins_provider_test.dart` | Begin under A, activate B, complete A. | Lease fence/invalidation absent. |
| 22 | UT-004 | Standard/project provider tests | Existing page then `ApiBadRequest('invalid_cursor')`, followed by page one. | Restart behavior absent. |
| 23 | IT-016 | Three Flutter provider suites | Deferred mutations, targeted refresh, same/other slot, multiple accounts, stale cursor. | Cross-provider coordination absent. |
| 24 | UT-005 | `app/test/feed/widgets/post_card_test.dart` | Owner/surface/type/current-pin/pending matrix. | Explicit action eligibility and labels absent. |
| 25 | UT-007 | `app/test/feed/widgets/post_card_test.dart` | Pin/repost baseline, semantics, narrow width, large text. | Pinned attribution absent. |
| 26 | AT-001 | `app/test/profile/widgets/profile_posts_tab_test.dart` | Own standard profile page and successful mutation/refresh. | End-user standard flow absent. |
| 27 | AT-002 | Feed, thread, and Projects-tab widget tests | Quote/project owner cards on approved surfaces. | Multi-surface owner actions absent. |
| 28 | AT-003 | Provider + Posts-tab tests | Occupied A, deferred B replacement, independent project action. | Server-confirmed replacement UI absent. |
| 29 | AT-004 | Provider + `PostCard` tests | Current unpin and stale target response containing newer B. | Target-specific unpin UI absent. |
| 30 | AT-005 | Profile tabs + `PostCard` tests | Older pin and page metadata at limit 10. | Profile-first annotation workflow absent. |
| 31 | AT-006 | AppView handler + `PostCard` tests | Invalid target matrix and viewer-hidden stored pin. | Cross-layer rejection/menu protection absent. |
| 32 | AT-007 | AppView pagination + Flutter provider tests | Complete traversal then pin replacement. | Server invalidation/client restart workflow absent. |
| 33 | AT-008 | Provider + `PostCard` tests | Deferred pin/unpin failures and exact messages. | Failure preservation/feedback absent. |
| 34 | AT-009 | Provider + account-boundary test | Switch A to B during deferred request. | End-to-end account fence absent. |
| 35 | AT-010 | Handler + provider tests | Ordinary current members use both slots. | Universal-member flow absent. |
| 36 | AT-011 | Lifecycle + indexer tests | Permanent/structural and temporary policy cases. | Full lifecycle workflow absent. |
| 37 | AT-012 | Feed/thread/search/projects/saved/notification tests | Same post data outside profile context. | Default-off surface regression assertions absent. |
| 38 | REG-001 | Existing standard/project ordering tests | Both slots empty and no visible pin. | Exact unchanged chronology/metadata guard absent. |
| 39 | REG-002 | Shared-card and actual surface widget tests | Timeline, thread, search, notification, saved, discovery contexts. | Profile-only presentation guard absent. |
| 40 | REG-003 | Existing policy/language profile-list tests | Stored pin plus excluded rows. | Policy-before-limit regression guard absent. |
| 41 | REG-004 | AppView response + Flutter model tests | Inspect canonical post and non-profile JSON keys. | No-leak contract guard absent. |
| 42 | REG-005 | Handler authorization tests | Multiple ordinary current-member fixtures using the existing authorization path only. | Universal current-member authorization regression guard absent. |
| 43 | REG-006 | Feed/search/project/notification response tests | Compare before/after database pin mutation. | No-ranking/count/notification guard absent. |
| 44 | REG-007 | Existing profile membership tests | Quote/standard/project/reply shapes with pin feature installed. | Existing membership boundary guard absent. |
| 45 | REG-008 | Migration test | Up/down around chronology and saved-post sentinels. | Reversal preservation guard absent. |
| 46 | MAN-001 | Physical iOS/Android assistive technology | VoiceOver/TalkBack plus keyboard/focus, both slot states. | Manual verification pending after implementation. |
| 47 | MAN-002 | Physical/emulator visual review | Light/dark, narrow width, max supported text scale versus repost row. | Manual verification pending after implementation. |

Focused commands after the corresponding tests exist:

- First loop, from `appview/` with compose Postgres available:
  - `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/db -run TestProfilePinsMigration -count=1`
- Focused AppView feature packages:
  - `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/db ./internal/api ./internal/index ./internal/routes -run 'ProfilePin|ProfilePins|Pinned' -count=1`
- Focused Flutter model/client/provider/widget/profile tests, from `app/`:
  - `flutter test test/feed/models/post_page_test.dart test/feed/models/profile_pin_state_test.dart test/feed/data/post_api_client_test.dart test/feed/providers/profile_pins_provider_test.dart test/feed/providers/user_posts_provider_test.dart test/projects/providers/user_projects_provider_test.dart test/feed/widgets/post_card_test.dart test/profile/widgets/profile_posts_tab_test.dart test/profile/widgets/profile_projects_tab_test.dart`
- Generation during implementation:
  - `flutter gen-l10n`
  - `dart run build_runner build --delete-conflicting-outputs`
- Final gates from repository root:
  - `just fmt`
  - `just test`
  - `just app-analyze`
  - `just app-test`

## 10. Sequencing And Guardrails

- First TDD step: `IT-001` — add only the failing `TestProfilePinsMigration`, run it to prove `000035` is missing, then add the minimum reversible migration.
- Dependencies between work items:
  1. Migration/schema before store behavior.
  2. Pure slot classification before transactional pin validation.
  3. Store semantics before handlers and route contracts.
  4. Exact AppView contract before Flutter client methods.
  5. Pin-aware cursor/page query before Flutter restart and profile annotation.
  6. Lifecycle/privacy/observability before considering the backend slice complete.
  7. Flutter model/API before controller; controller before menu/profile workflows.
  8. Regression and generation/final gates after every selected behavior is green.
- Guardrails:
  - Do not modify `lexicon/`; no atproto lexicon skill, ADR, lexgen, PDS record, or Tap dispatcher change is needed.
  - Do not add payment, free/paid-user, plan, tier, entitlement, access-gating model, dependency, seam, fixture, telemetry dimension, or test.
  - Do not add a public pin field to `craftsky_posts`, `PostResponse`, `Post`, timeline items, search results, notification records, or PDS data.
  - Do not change `profile_sort_at`; promotion is response composition only.
  - Do not optimistically reorder Flutter lists or change confirmed pin menu state before the authoritative response.
  - Do not use one global pending flag; key by account family and slot, with the other slot usable.
  - Do not expose raw pin URI/timestamp/state in cursor contents, logs, traces, metrics, or another owner's private response.
  - Do not clear stored pins because one viewer blocks, mutes, moderates, or language-filters the target.
  - Do not post-filter after `LIMIT`; pin visibility and all existing policy/language predicates participate before capacity is allocated.
  - Do not use timing sleeps for concurrency tests; use deterministic transaction barriers.
  - Keep route PUT/DELETE bodyless and preserve authoritative `200`; protect the exception at policy, handler, and Flutter layers.
  - Parse typed DIDs/rkeys at the HTTP boundary and use typed `syntax.ATURI`/`syntax.DID` values internally where the existing interface permits.
  - Use parameterized raw pgx consistent with the current AppView stores; do not add an ORM or activate sqlc as unrelated tooling scope.
  - Run generators deliberately and inspect their changed-file set before formatting/staging; avoid unrelated generated churn.
- Explicit implementation approval gate: because implementation adds a migration and private-state behavior, the TDD stage must begin only after the user chooses to continue to `implement-tdd` from this completed plan.
- Out of scope:
  - Any commercial/access policy.
  - PDS portability/publication or independent-AppView convergence.
  - Timeline/search/discovery ranking, notifications, engagement changes, or analytics beyond bounded operational diagnostics.
  - More than two fixed slots, manual reordering, scheduling, collections, or a management page.
  - A new system/device E2E harness; use the approved split Go/Flutter automation and two manual checks.

## 11. Risks And Open Questions

| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Resolved design risk | A raw post URI or timestamp in a base64 cursor could reveal private hidden-pin state. | Violates AppView-private selection intent. | Store a random internal `state_token` per occupied selection; cursor carries only that opaque token or the empty marker. Same-target pin preserves it; replacement rotates it. |
| CPQ-002 | Resolved design risk | A visible pin with `limit=1` leaves no chronological row from which to derive the next seek boundary. | Page two could skip the chronological head or end early. | Cursor supports an explicit pre-chronology boundary; later query starts at the head while excluding the fixed pin. Cover limits 1/2/10. |
| CPQ-003 | Resolved design risk | Reading the two slots after only a slot-scoped lock can race an independent-slot commit and make the mutation response ambiguous. | Response may not represent one commit-time owner state. | Serialize pin mutations by owner, then read both slots before commit. Client pending remains per slot; server serialization is short and does not change UI availability. |
| CPQ-004 | Non-blocking | Existing AppView SQL is inline pgx despite the aspirational sqlc convention. | Introducing sqlc here would add config/generated churn unrelated to pin behavior. | Follow the current saved-post/post-store pattern with localized parameterized SQL and real-Postgres/query-plan tests. Reassess only if sqlc is adopted before implementation starts. |
| CPQ-005 | Non-blocking | `PostReader` is broad and many handler tests use fakes, so changing existing method signatures can cause large unrelated fake churn. | Compile noise could obscure TDD behavior. | Add a narrow pin-aware page interface/result and adapt only profile-list handler fakes; keep unrelated post-reader methods unchanged where possible. |
| CPQ-006 | Non-blocking | Initial private pin read can fail while cards otherwise render. | Showing guessed `Pin post` could be wrong for the current target. | Keep the new action unavailable until authoritative pin state loads; do not block other card actions or add speculative local state. |
| CPQ-007 | Non-blocking | Generated mapper/provider/l10n output may touch unrelated files. | Review and staging noise. | Run generators after source tests define the behavior, inspect `git diff --name-only`, and retain only legitimate generated updates. |
| CPQ-008 | Non-blocking | Physical screen-reader announcements and exact visual parity cannot be proven entirely in widget tests. | UI verification remains incomplete after automated gates. | Run MAN-001 and MAN-002 after implementation; record their outcome in `05-implementation-plan.md` or as explicit remaining release gates. |

No blocking implementation questions remain.

## 12. Handoff To TDD Builder

- Coding plan: `docs/changes/2026-08-04-pinned-profile-posts/04-coding-plan.md`
- TDD execution plan: `docs/changes/2026-08-04-pinned-profile-posts/05-implementation-plan.md`
- Start with test: `IT-001`, named `TestProfilePinsMigration`, in `appview/internal/db/profile_pins_migration_test.go`.
- Focused initial command:
  - `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/db -run TestProfilePinsMigration -count=1`
- Source of truth before coding: re-read `01-requirements.md`, `02-acceptance-tests.md`, `03-document-review.md`, and this file from disk.
- First expected red: test cannot read/apply `appview/migrations/000035_profile_pins.up.sql` because the migration does not exist.
- TDD notes:
  - Execute one test ID at a time in the order in Section 9.
  - Record the exact red failure, minimum green implementation, focused command, refactor, and nearby regression command in `05-implementation-plan.md`.
  - Re-check the highest migration immediately before IT-001 implementation.
  - Treat DR-001–DR-003 as fixed contracts: bodyless authoritative mutations, exact metadata omission, and pending state by account plus slot.
  - MAN-001 and MAN-002 remain post-automation manual gates and must not be reported as passed unless actually run.
