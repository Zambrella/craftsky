# Coding Plan: AppView Facet Endpoints And Plain Profile Bios

## 1. Inputs
- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` (`Approved with notes`)
- Review notes carried forward:
  - DR-001: `limit > 25` must reject with `400 validation_error`; empty or whitespace-only `q` returns `{items:[]}`.
  - DR-002: bounded backfill command is `cli identity-cache backfill`, default limit `100`, with `--limit <n>`.
  - DR-003: profile-bio parsing targets Craftsky-supported token rules, not full Bluesky parity; fixture coverage must be explicit.
  - DR-004: AppView exact-resolve endpoint tests and Flutter “no facet emitted but continue” tests remain separate.
  - Manual post-review update: newly created/initialized Craftsky users must be upserted into the identity cache immediately; backfill is only for existing profiles.

## 2. Implementation Strategy

Implement this as an additive AppView API/persistence change plus a Flutter data/UI cleanup.

On the AppView, add authenticated `/v1/facets/*` read endpoints under the existing `authN(deviceID(...))` route stack. Back mention suggestions with a separate identity/handle cache table, not `bluesky_profiles`, and join that cache to existing `craftsky_profiles`, `bluesky_profiles`, and `atproto_follows` tables for Craftsky-only suggestion rows. Exact mention resolution should use the existing `api.HandleResolver` seam, refresh missing/stale cache rows, and return `404 mention_not_found` for non-Craftsky or unresolved handles. Also wire the AppView profile creation/initialization path so a newly created/initialized Craftsky user gets their current DID/handle upserted into the identity cache immediately, without waiting for the bounded backfill command. Hashtag suggestions should use indexed `craftsky_posts.tags` data from recent root posts.

On Flutter, replace the production mock facet repositories with AppView-backed Dio repositories while keeping mock implementations as test fixtures. Use exact AppView-backed mention resolution only during final post facet generation via the existing `MentionResolver` seam. Remove `descriptionFacets` from the profile model, repository/API update signatures, save flow, and `ProfileBio` API. Replace the profile-bio editor’s autocomplete field with a plain branded text field. Preserve clickable bio rendering by parsing plain description text at render time with token rules centralized from, or explicitly shared with, the post facet generator.

The first TDD slice should be AppView route/API contract coverage (`IT-001` and `UT-001`/`UT-002`) because Flutter repository work depends on stable endpoint paths, response shapes, auth/device behavior, and validation semantics.

## 3. Affected Areas

| Area | Existing Pattern | Planned Change | Requirement IDs | Test IDs |
|---|---|---|---|---|
| AppView facet routes | `routes.go` registers `/v1/*` handlers with `authN(deviceID(...))`; handlers live in `internal/api` | Add `GET /v1/facets/mentions`, `GET /v1/facets/mentions/resolve`, and `GET /v1/facets/hashtags` | FR-001, FR-003, FR-005, NFR-001 | IT-001, UT-001, UT-004, UT-013, REG-003 |
| AppView request/response contracts | Small DTO files such as `profile_request.go` / `profile_response.go`; camelCase JSON; `envelope.WriteError` | Add facet request validation and response DTOs with wrapped `items` for suggestions and minimal resolve object | FR-002, FR-003, FR-005, NFR-001, NFR-002 | UT-001, UT-002, UT-004, UT-013, IT-001 |
| AppView identity cache | Handles currently resolved on demand through `HandleResolver`; no searchable handle cache | Add separate identity cache migration/store with 24h freshness; no handle column on `bluesky_profiles` | FR-001, FR-013, FR-014, FR-015, FR-016 | IT-002, IT-003, IT-005, IT-009, REG-005, REG-006 |
| AppView profile initialization | OAuth callback calls `auth.InitializeProfile` to create/ensure Craftsky profile records; no identity-cache side effect exists | After successful profile creation/initialization, resolve current handle and upsert identity cache for the authenticated DID | FR-016 | IT-009, REG-006, MAN-004 |
| AppView mention suggestions/resolution | Profile reads resolve handles per DID; no account autocomplete endpoint | Search fresh cached Craftsky handles/display names; exact resolve refreshes cache and filters to Craftsky profiles | FR-001, FR-002, FR-003, FR-004, RULE-001 | AT-001, AT-002, UT-003, UT-004, IT-001, IT-002, IT-003 |
| AppView hashtag suggestions | `craftsky_posts.tags` has normalized tag array and GIN index | Count matching lowercase tags on recent root posts only, sorted by count desc then tag asc | FR-005, FR-006, RULE-003 | AT-003, UT-005, IT-004 |
| AppView CLI | Cobra root command with subcommands in `appview/cmd/cli`; DB-using commands call `loadDeps` | Add `identity-cache backfill` command with default limit 100 and explicit `--limit` | FR-015 | IT-005, MAN-002 |
| Flutter facet data repositories | `AccountSuggestionRepository` / `HashtagSuggestionRepository` are mock-backed providers; account repo implements `MentionResolver` | Add Dio-backed repositories and switch production providers to AppView implementations | FR-007, FR-008, NFR-001 | AT-001, AT-002, AT-003, AT-006, UT-006, UT-007, UT-008 |
| Flutter post facet generation | `FacetGenerator` scans final text and uses injected `MentionResolver` | Keep final-text scanning; resolve handles through AppView-backed repository; preserve byte offsets and link/tag semantics | FR-008 | AT-002, UT-007, UT-012, REG-002 |
| Flutter profile edit/save | Bio editor uses `FacetAutocompleteEditor`; save generates/sends `descriptionFacets` | Use plain text input; stop generating/sending facets; remove facets from profile repository/API signatures | FR-009, FR-010, RULE-002 | AT-004, UT-011, IT-007, REG-001, REG-004 |
| Flutter profile bio rendering/actions | `ProfileBio` passes stored `descriptionFacets` into `FacetedText`; actions live in `FacetActionHandler` | Parse plain text at render time; style/tap detected links, mentions, and hashtags; mention taps navigate by visible handle | FR-011, FR-012, NFR-003 | AT-005, UT-009, UT-010, IT-008 |

## 4. Files And Modules

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `appview/migrations/000015_identity_handle_cache.up.sql` / `.down.sql` | Create | Separate identity/handle cache schema and indexes; no network work | FR-013, FR-014, FR-015, FR-016 | IT-002, IT-005, IT-009, REG-005, REG-006 |
| `appview/internal/api/facet_request.go` | Create | Parse/validate `q`, `limit`, and `handle` query params; enforce default 10, max 25, max query length 64 | FR-001, FR-003, FR-005, NFR-002 | UT-001, UT-004, UT-013 |
| `appview/internal/api/facet_response.go` | Create | DTOs for mention suggestions, exact mention resolve, hashtag suggestions, and `{items:[...]}` wrappers | FR-002, FR-003, FR-005, NFR-001 | UT-002, UT-004, IT-001 |
| `appview/internal/api/identity_cache_store.go` | Create | Read/write identity cache rows; freshness checks; upsert by DID/canonical handle; expose a narrow current-handle upsert service usable by auth wiring without package cycles | FR-013, FR-014, FR-015, FR-016 | IT-002, IT-003, IT-005, IT-009 |
| `appview/internal/api/facet_store.go` | Create | Query mention suggestions and hashtag suggestions; check Craftsky membership for exact resolve | FR-001, FR-002, FR-004, FR-005, FR-006 | UT-003, UT-005, IT-002, IT-003, IT-004 |
| `appview/internal/api/facet.go` | Create | HTTP handlers for `/v1/facets/*`; error mapping and response encoding | FR-001, FR-003, FR-005, NFR-001, RULE-001 | IT-001, UT-004, UT-013 |
| `appview/internal/routes/routes.go` | Change | Register new facet endpoints behind auth/device middleware | NFR-001 | IT-001, REG-003 |
| `appview/internal/app/deps.go` | Change | Wire `IdentityCacheStore` / `FacetStore` dependencies if stored on `Deps`, or construct in `routes.go` from `deps.DB`; make profile initialization cache updater available to OAuth handlers | FR-001, FR-005, FR-014, FR-016 | IT-001, IT-002, IT-004, IT-009 |
| `appview/internal/auth/handlers_oauth.go` | Change | After `InitializeProfile` succeeds, call injected identity-cache updater for the authenticated DID | FR-016 | IT-009, REG-006 |
| `appview/internal/auth/initialize_profile.go` | Change if needed | Keep PDS profile initialization behavior; avoid direct DB coupling unless using a narrow injected interface | FR-016 | IT-009 |
| `appview/internal/auth/*_test.go` | Change | Assert profile initialization/callback path upserts identity cache for new users and handles resolver failure without partial cache rows | FR-016 | IT-009, REG-006 |
| `appview/cmd/cli/identity_cache.go` | Create | Add `identity-cache backfill` Cobra command using `loadDeps` and `HandleResolver` | FR-015 | IT-005, MAN-002 |
| `appview/cmd/cli/main.go` | Change | Add the identity-cache command to the root command tree | FR-015 | IT-005 |
| `appview/internal/api/*facet*_test.go` | Create | Handler, request, response, ranking, and error-envelope tests | FR-001-FR-006, NFR-001, NFR-002, RULE-001, RULE-003 | UT-001-UT-005, UT-013, IT-001 |
| `appview/internal/api/identity_cache_store_test.go` | Create | Cache freshness/search/refresh/backfill/profile-initialization upsert tests with fake resolver inputs | FR-013, FR-014, FR-015, FR-016 | IT-002, IT-003, IT-005, IT-009, REG-005, REG-006 |
| `appview/internal/routes/routes_test.go` | Change | Auth/device regression coverage for new `/v1/facets/*` routes | NFR-001 | IT-001, REG-003 |
| `app/lib/shared/rich_text/data/appview_facet_suggestion_repository.dart` | Create | Dio-backed account suggestion, exact mention resolve, and hashtag suggestion implementation | FR-007, FR-008, NFR-001 | UT-006, UT-007, UT-008, AT-001-AT-003 |
| `app/lib/shared/rich_text/data/facet_suggestion_repository.dart` | Change | Keep interfaces; add optional `limit` parameters only if tests need explicit bounds, otherwise repositories use default 10 internally | FR-007, FR-008 | UT-006, UT-008, IT-006 |
| `app/lib/shared/rich_text/providers/facet_suggestion_providers.dart` | Change | Switch production providers from mocks to AppView-backed repositories using `dioProvider`; keep debounce provider | FR-007, FR-008 | AT-001, AT-003, AT-006 |
| `app/lib/shared/rich_text/facet_token_parser.dart` | Create | Central plain-token detection shared by `FacetGenerator` and profile-bio rendering | FR-011, NFR-003 | UT-009, UT-012, REG-002 |
| `app/lib/shared/rich_text/facet_generator.dart` | Change | Reuse shared token parser; call exact mention resolver for final text only | FR-008, FR-011 | UT-007, UT-012, REG-002 |
| `app/lib/shared/rich_text/faceted_text_model.dart` / `faceted_text.dart` | Change | Allow render-time plain ranges or an internal mention-by-handle feature without requiring stored AT facets | FR-011, FR-012, NFR-003 | UT-009, UT-010, IT-008 |
| `app/lib/shared/rich_text/facet_action_handler.dart` | Change | Ensure link actions allow only HTTP/S, hashtags route to `/search?tag=`, and profile-bio mentions navigate by visible handle without pre-resolve | FR-012, NFR-003 | UT-010, IT-008, AC-017, AC-020 |
| `app/lib/profile/models/profile.dart` + `profile.mapper.dart` | Change | Remove `descriptionFacets` from model and generated mapper | FR-010, RULE-002 | UT-011, REG-004 |
| `app/lib/profile/data/profile_api_client.dart` | Change | Remove `descriptionFacets` argument and request body field | FR-010, RULE-002 | UT-011, REG-001 |
| `app/lib/profile/data/profile_repository.dart` / `api_profile_repository.dart` | Change | Remove `descriptionFacets` from repository signatures | FR-010 | UT-011, IT-007 |
| `app/lib/profile/providers/save_profile_provider.dart` + generated provider if needed | Change | Remove facet argument and forwarding from save mutation | FR-010 | AT-004, IT-007, REG-001 |
| `app/lib/profile/pages/edit_profile_dialog.dart` | Change | Replace bio `FacetAutocompleteEditor`/`FacetTextEditingController` with plain `BrandTextField`/`TextEditingController`; remove facet generation | FR-009, FR-010 | AT-004, IT-007 |
| `app/lib/profile/widgets/profile_bio.dart` | Change | Remove `descriptionFacets` parameter; parse plain text into clickable ranges at render time | FR-010, FR-011, FR-012 | AT-005, UT-009, UT-010, IT-008 |
| `app/lib/profile/widgets/profile_meta_section.dart` / `profile_about_tab.dart` | Change | Stop passing `descriptionFacets` to `ProfileBio` | FR-010 | REG-004 |
| `app/test/profile/fakes/fake_profile_repository.dart` | Change | Remove facet-aware update callback once production interface changes | FR-010 | IT-007, REG-004 |
| `app/test/shared/rich_text/*`, `app/test/profile/*` | Change/Create | Flutter repository, generator, bio parser/action, model/API/save-flow tests | FR-007-FR-012, NFR-003, RULE-002 | UT-006-UT-012, AT-001-AT-006, IT-006-IT-008, REG-001, REG-002, REG-004 |

## 5. Services, Interfaces, And Data Flow

### Final AppView endpoint contracts

```text
GET /v1/facets/mentions?q=<query>&limit=<n>
Authorization: Bearer <craftsky-session-token>
X-Craftsky-Device-Id: <device-id>

200 { "items": [
  {
    "did": "did:plc:alice",
    "handle": "alice.craftsky.social",
    "displayName": "Alice",        // omitted when unknown
    "avatar": "https://...",        // omitted when unknown
    "isCraftskyProfile": true,
    "viewerIsFollowing": true
  }
] }

GET /v1/facets/mentions/resolve?handle=alice.craftsky.social
200 { "did": "did:plc:alice", "handle": "alice.craftsky.social", "isCraftskyProfile": true }
404 { "error": "mention_not_found", "message": "mention not found", "requestId": "..." }

GET /v1/facets/hashtags?q=sock&limit=10
200 { "items": [ { "tag": "sockkal", "postsLast28Days": 12 } ] }
```

Validation rules:
- Missing, empty, or whitespace-only `q` on suggestion endpoints returns `200 {"items":[]}`.
- `q` after trimming must be at most 64 characters; longer queries return `400 validation_error`.
- Missing `limit` defaults to `10`; accepted explicit bounds are `1..25`; `limit > 25` returns `400 validation_error`.
- Invalid or empty exact `handle` returns `404 mention_not_found` rather than leaking resolver details.
- All JSON fields are camelCase and all non-2xx responses use `envelope.WriteError`.

### Identity cache schema sketch

Use a separate table name that makes the boundary clear. Recommended final name: `atproto_identity_cache`.

```text
atproto_identity_cache
  did TEXT PRIMARY KEY
  handle TEXT NOT NULL
  handle_lower TEXT NOT NULL
  resolved_at TIMESTAMPTZ NOT NULL
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()

indexes:
  UNIQUE(handle_lower)
  handle_lower text_pattern_ops index for prefix autocomplete
  resolved_at index if stale scans/backfill need it
```

Guardrails:
- Do not add handles to `bluesky_profiles`.
- SQL migrations create schema/indexes only; no resolver/network calls in migrations.
- Cache freshness is `resolved_at >= now - interval '24 hours'` for autocomplete inclusion.
- Exact resolve may refresh stale/missing rows; autocomplete must not fan out to the atproto network per candidate.

### AppView store/interface sketches

```text
type FacetStore struct { pool *pgxpool.Pool }
type IdentityCacheStore struct { pool *pgxpool.Pool }

type MentionSuggestionRow struct {
  DID string
  Handle string
  DisplayName *string
  AvatarCID *string
  AvatarMime *string
  ViewerIsFollowing bool
}

func (s *FacetStore) SearchMentionSuggestions(ctx, viewerDID, query string, limit int, now time.Time) ([]MentionSuggestionRow, error)
func (s *FacetStore) SearchHashtagSuggestions(ctx, query string, limit int, now time.Time) ([]HashtagSuggestionRow, error)
func (s *FacetStore) IsCraftskyProfile(ctx context.Context, did syntax.DID) (bool, error)

func (s *IdentityCacheStore) FreshByHandle(ctx, handle syntax.Handle, now time.Time) (*IdentityCacheRow, error)
func (s *IdentityCacheStore) Upsert(ctx, did syntax.DID, handle syntax.Handle, resolvedAt time.Time) error
func (s *IdentityCacheStore) BackfillCandidateDIDs(ctx, limit int) ([]syntax.DID, error)
func (s *IdentityCacheService) UpsertCurrentHandle(ctx context.Context, did syntax.DID, now time.Time) error
```

Mention suggestion query shape:

```text
FROM atproto_identity_cache ic
JOIN craftsky_profiles cp ON cp.did = ic.did
LEFT JOIN bluesky_profiles bp ON bp.did = ic.did
WHERE ic.resolved_at >= $fresh_cutoff
  AND (
    ic.handle_lower LIKE '%' || $query_lower || '%'
    OR lower(coalesce(bp.display_name, '')) LIKE '%' || $query_lower || '%'
  )
ORDER BY
  EXISTS(SELECT 1 FROM atproto_follows f WHERE f.did=$viewer AND f.subject_did=ic.did) DESC,
  CASE WHEN ic.handle_lower LIKE $query_lower || '%' THEN 0 ELSE 1 END ASC,
  ic.handle_lower ASC
LIMIT $limit
```

Hashtag query shape:

```text
SELECT lower(tag) AS tag, COUNT(DISTINCT p.uri) AS posts_last_28_days
FROM craftsky_posts p
CROSS JOIN LATERAL unnest(p.tags) AS tag
WHERE p.reply_root_uri IS NULL
  AND p.reply_parent_uri IS NULL
  AND p.created_at >= $now - interval '28 days'
  AND trim(tag) <> ''
  AND lower(tag) LIKE '%' || $query_lower || '%'
GROUP BY lower(tag)
ORDER BY posts_last_28_days DESC, tag ASC
LIMIT $limit
```

Exact resolve flow:

```text
parse handle with syntax.ParseHandle(strings.TrimPrefix(raw, "@"))
if invalid: 404 mention_not_found

if cache row is fresh and Craftsky profile exists:
  return cached DID + handle

did := HandleResolver.ResolveDID(ctx, handle)
canonicalHandle := HandleResolver.ResolveHandle(ctx, did) // for returned/cached handle
if resolver fails or did has no craftsky_profiles row:
  return 404 mention_not_found

IdentityCacheStore.Upsert(ctx, did, canonicalHandle, now)
return {did, handle: canonicalHandle, isCraftskyProfile: true}
```

### Profile initialization cache flow

New Craftsky users must not depend on an operator running backfill before they can appear in mention autocomplete. Add a narrow identity-cache updater to the OAuth/profile initialization wiring.

```text
OAuth callback / profile initialization
  -> InitializeProfile(ctx, pdsClient, did) succeeds
  -> IdentityCacheUpdater.UpsertCurrentHandle(ctx, did)
       ResolveHandle(ctx, did) -> canonical handle
       Upsert(ctx, did, canonical handle, now)
  -> continue creating Craftsky session / returning token
```

Implementation guardrail for package boundaries:
- Avoid making `internal/auth` import `internal/api`, because `internal/api` already imports `internal/auth` in several files.
- Prefer an interface defined in `internal/auth`, implemented by an AppView/api identity-cache service and injected from `routes.go`/`deps.go`:

```text
// in internal/auth or a neutral package
type IdentityCacheUpdater interface {
  UpsertCurrentHandle(ctx context.Context, did syntax.DID) error
}
```

If the upsert fails because handle resolution is transiently unavailable, do not create a partial row and do not store anything on `bluesky_profiles`. Log the failure; exact resolve and `cli identity-cache backfill` remain recovery paths. Tests should lock this behavior without using live identity network calls.

### CLI backfill flow

```text
cli identity-cache backfill [--limit <n>]
default limit = 100

candidate DIDs = Craftsky profile DIDs missing cache or stale by 24h, bounded by limit
for each DID:
  handle := HandleResolver.ResolveHandle(ctx, did)
  if success: upsert cache row
  if failure: log and continue; command exits non-zero only for setup/store failures
```

Manual dev command: `docker compose exec appview /app/cli identity-cache backfill`.

### Flutter repository/data flow

```text
dioProvider
  -> AppViewAccountSuggestionRepository
       searchAccounts(q) -> GET /v1/facets/mentions?q=q&limit=10
       didForHandle(handle) -> GET /v1/facets/mentions/resolve?handle=handle
           200 -> did
           404 mention_not_found -> null
  -> AppViewHashtagSuggestionRepository
       searchHashtags(q) -> GET /v1/facets/hashtags?q=q&limit=10

FacetAutocompleteEditor
  -> accountSuggestionRepositoryProvider.searchAccounts(token.query) while typing
  -> hashtagSuggestionRepositoryProvider.searchHashtags(token.query) while typing

Create post / final submit
  -> facetGeneratorProvider
    -> FacetGenerator.generate(finalText)
      -> MentionResolver.didForHandle(handle) only during final generation
```

Do not add hidden selected-mention state to the composer; final text remains the source of truth.

### Plain bio token parser sketch

Centralize the current `FacetGenerator` regex/helper behavior into a shared parser so posts and profile bios use the same supported token semantics.

```text
enum FacetTokenKind { mention, link, tag }

class FacetToken {
  FacetTokenKind kind;
  int charStart;
  int charEnd;
  String visibleText;
  String? handle; // mention without leading @
  String? uri;    // explicit http(s), or bare domain normalized to https://
  String? tag;    // without leading #
}

List<FacetToken> detectSupportedFacetTokens(String text) {
  // Current mention/link/hashtag patterns.
  // Sort by charStart, longer ranges first for same start.
  // Drop overlaps deterministically.
  // Trim trailing link punctuation as current generator does.
}
```

`FacetGenerator.generate` uses this parser, resolving only mention tokens before emitting AT Protocol mention facets. `ProfileBio` uses the same parser to build render ranges directly, without storing or sending facets.

## 6. State, Providers, Controllers, Or DI

### AppView DI

- Keep handler constructors narrow and testable rather than passing all of `app.Deps` into handlers.
- Preferred route wiring:

```text
facetStore := api.NewFacetStore(deps.DB)
identityCache := api.NewIdentityCacheStore(deps.DB)

mux.Handle("GET /v1/facets/mentions",
  authN(deviceID(api.ListFacetMentionSuggestionsHandler(facetStore, deps.Logger))))
mux.Handle("GET /v1/facets/mentions/resolve",
  authN(deviceID(api.ResolveFacetMentionHandler(facetStore, identityCache, deps.HandleResolver, deps.Logger))))
mux.Handle("GET /v1/facets/hashtags",
  authN(deviceID(api.ListFacetHashtagSuggestionsHandler(facetStore, deps.Logger))))
```

- If stores are placed on `app.Deps`, wire them in `newDeps` after `deps.ProfileStore` and keep the fields specific (`FacetStore`, `IdentityCacheStore` / `IdentityCacheUpdater`).
- Pass the identity-cache updater into OAuth handlers so `CallbackHandler` can upsert the authenticated user's current handle after `InitializeProfile` succeeds.
- CLI `identity-cache backfill` should call `loadDeps(ctx)` and reuse `deps.DB`, `deps.HandleResolver`, and `deps.Logger`.

### Flutter Riverpod provider graph

```text
dioProvider
  -> accountSuggestionRepositoryProvider = AppViewAccountSuggestionRepository(dio)
       -> facetGeneratorProvider = FacetGenerator(mentionResolver: accountSuggestionRepositoryProvider)
  -> hashtagSuggestionRepositoryProvider = AppViewHashtagSuggestionRepository(dio)

facetAutocompleteDebounceProvider
  -> FacetAutocompleteEditor typing UX

profileRepositoryProvider
  -> saveProfileProvider
      -> EditProfileDialog plain text save
```

Provider guardrails:
- Keep `MockAccountSuggestionRepository` and `MockHashtagSuggestionRepository` for tests and explicit provider overrides.
- Suggestion repository failures should fail closed for autocomplete. Catch mapped `ApiException`/Dio errors inside `searchAccounts` and `searchHashtags` and return `[]`, unless existing app-wide error policy strongly prefers surfacing exceptions. Exact `didForHandle` should map only `mention_not_found` to `null`; other API failures may also return `null` to keep post submission continuing for other text per EC-006.
- Do not introduce PDS tokens or atproto credentials into Flutter.

## 7. UI, Widgets, Routes, Or User-Facing Surfaces

### Composer autocomplete

- Keep `FacetAutocompleteEditor` UI composition and insertion behavior.
- Suggestion endpoint results are already ordered by AppView; the widget should preserve result order and continue taking at most `_maxDisplayedSuggestions` for display.
- Mention rows must render with required handle/DID/isCraftskyProfile/viewerIsFollowing data and tolerate omitted `displayName`/`avatar`.
- Hashtag selection inserts `#${tag} `; AppView returns `tag` without a leading `#` and in lowercase canonical form.

### Edit-profile bio

- Replace the bio `FacetAutocompleteEditor` with `BrandTextField`.
- Replace `FacetTextEditingController` with `TextEditingController` for the bio.
- Remove `facetGeneratorProvider` reads from the save path.
- Save only `description` plus the existing display name/crafts/avatar/banner fields.
- No mention/hashtag autocomplete overlay should appear for bio typing.

### Profile bio rendering/taps

- `ProfileBio` should accept only `description` and render nothing for null/empty descriptions.
- For non-empty plain descriptions, parse supported tokens at render time and style them with the same primary-color convention as `FacetedText`.
- Links: launch only `http://` or `https://`; bare domains normalize to `https://`; unsupported schemes remain plain text.
- Mentions: navigate to `/profile/<visible-handle>` using the visible handle without pre-resolving.
- Hashtags: route to `/search?tag=<tag>` using existing `FacetActionHandler` semantics.
- No new routes are required.

## 8. Error, Loading, Empty, And Edge States

| State / Case | Planned Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| Empty or whitespace suggestion query | Return `200 {"items":[]}` from AppView; Flutter shows no hashtag surface and mention “No results” only if widget behavior already does so | FR-001, FR-005, NFR-002 | UT-001, IT-001, AC-014 |
| Query length > 64 | Return `400 validation_error` envelope | NFR-002 | UT-001, UT-013, AC-014 |
| `limit > 25` | Return `400 validation_error`; do not clamp | NFR-002 | UT-001, EC-007 |
| Missing auth or device ID | Existing middleware returns `401`/`400 missing_device_id` envelopes | NFR-001 | IT-001, REG-003 |
| Mention suggestion account lacks display/avatar | Include row with required fields; omit optional fields | FR-002 | AT-001, UT-002, IT-001, AC-015 |
| Mention cache stale/missing in autocomplete | Autocomplete only uses fresh cache; no network fan-out while typing | FR-014 | IT-002, AC-018 |
| Mention cache stale/missing in exact resolve | Resolve through `HandleResolver`; upsert cache on successful Craftsky resolve | FR-014 | IT-003, AC-018 |
| Newly initialized Craftsky user | Profile initialization/callback path resolves the current handle and upserts identity cache without waiting for backfill | FR-016 | IT-009, AC-021, REG-006 |
| Profile initialization handle resolution failure | Do not store a partial cache row or write handles to `bluesky_profiles`; log and allow later exact resolve/backfill recovery | FR-016 | IT-009, EC-008 |
| Exact handle invalid, non-Craftsky, unknown, or resolver failure | AppView returns `404 mention_not_found`; Flutter maps to `null`/no mention facet and continues | FR-003, FR-004, FR-008, RULE-001 | UT-004, UT-007, IT-003, AT-002 |
| Hashtag no matches | Return `200 {"items":[]}` | FR-006 | UT-005, IT-004, EC-002 |
| Repeated tags in one post | Count each lowercase canonical tag once per root post via `COUNT(DISTINCT p.uri)` | FR-006, RULE-003 | IT-004, AC-007 |
| Autocomplete endpoint error in Flutter | Fail closed with empty/hidden suggestions; keep text entry usable | FR-007, FR-008 | AT-006, EC-006 |
| Profile save with rich-looking bio text | Request contains `description` only and no `descriptionFacets` | FR-010, RULE-002 | AT-004, UT-011, REG-001 |
| Plain bio malformed token or unsupported scheme | Leave plain, do not throw, do not launch unsafe URI | FR-011, NFR-003 | UT-009, AC-020, EC-003 |
| Overlapping bio ranges such as URL fragment with `#tag` | Parser chooses deterministic non-overlapping range order; skipped overlap remains safe/plain | FR-011, NFR-003 | UT-009, EC-004 |

## 9. Test Implementation Plan

| Order | Test ID | Target | Setup / Fixture | Initial Expected Failure |
|---|---|---|---|---|
| 1 | IT-001 | `appview/internal/routes/routes_test.go`, `appview/internal/api/facet_test.go` | Register new routes with fake auth/device; seed minimal store/fakes | `/v1/facets/*` routes are missing / return 404 |
| 2 | UT-001 | `appview/internal/api/facet_request_test.go` | Table tests for empty/whitespace/64/65-char query and limit 10/25/26 | No request parser exists; validation behavior undefined |
| 3 | UT-002 | `appview/internal/api/facet_response_test.go` | Marshal suggestion with omitted display/avatar | DTOs do not exist |
| 4 | UT-003 | `appview/internal/api/facet_suggestion_test.go` | Fake rows for followed prefix, unfollowed prefix, followed substring, handle tie | Ranking helper/store method missing |
| 5 | UT-004 | `appview/internal/api/facet_test.go` | Handler fake resolver/store for valid Craftsky, non-Craftsky, invalid/missing handle | Exact resolve handler missing |
| 6 | IT-002 | `appview/internal/api/identity_cache_store_test.go` or `facet_store_test.go` | Test DB with fresh/stale cache, `craftsky_profiles`, `bluesky_profiles`, follows | Cache schema/store missing |
| 7 | IT-003 | `appview/internal/api/facet_store_test.go`, `facet_test.go` | Fake resolver for Alice/Mallory/unknown and DB Craftsky membership | Refresh/filter exact resolve flow missing |
| 8 | IT-004 / UT-005 | `appview/internal/api/facet_store_test.go`, `facet_suggestion_test.go` | Seed root/reply/old/duplicate/empty tag posts | Hashtag query/count method missing |
| 9 | IT-009 | `appview/internal/auth/*_test.go`, `appview/internal/api/identity_cache_store_test.go` | Fake authenticated DID/handle resolver and cache writer/store; resolver failure variant | Profile initialization path does not update identity cache |
| 10 | IT-005 | `appview/cmd/cli/identity_cache*_test.go` | Fake bounded resolver; existing Craftsky rows; default and explicit limits | CLI command missing |
| 11 | UT-006 / UT-008 | `app/test/shared/rich_text/facet_suggestion_repository_test.dart` | Dio mock endpoints with wrapped `items` | AppView repository classes missing |
| 12 | UT-007 / AT-002 | `app/test/shared/rich_text/facet_generator_test.dart`, `app/test/feed/providers/create_post_provider_test.dart` | Dio success for Alice, 404 `mention_not_found` for unknown | Exact AppView resolver not wired to generator |
| 13 | IT-006 / AT-001 / AT-003 / AT-006 | `app/test/shared/rich_text/facet_autocomplete_editor_test.dart` | Provider overrides with AppView repo fakes and zero debounce | Production providers still mock-backed / error handling not locked |
| 14 | UT-011 / IT-007 / REG-001 / REG-004 / AT-004 | `app/test/profile/models/profile_test.dart`, `profile_api_client_test.dart`, `edit_profile_dialog_test.dart` | Profile JSON with description only; save rich-looking text | `descriptionFacets` still in model/API/save flow |
| 15 | UT-009 / UT-010 / IT-008 / AT-005 | `app/test/profile/widgets/profile_bio_test.dart`, `app/test/shared/rich_text/faceted_text_actions_test.dart` | TD-005 token fixtures, fake router/launcher | Bio depends on stored facets; parser absent |
| 16 | REG-002 | `app/test/shared/rich_text/facet_generator_test.dart` | Existing emoji/link/tag/overlap fixtures | Refactor can break byte offsets or AT facet JSON |
| 17 | MAN-001-MAN-004 | Manual dev smoke | `just dev-d`, backfill command, new-user initialization, composer/profile flows | Manual only after automated tests pass |

Focused commands:
- AppView: `just test` from repo root after compose Postgres is running via `just dev-d`.
- AppView formatting/vet after implementation: `just fmt`.
- Flutter focused tests from `app/`: `flutter test test/shared/rich_text test/profile test/feed/providers/create_post_provider_test.dart`.
- Flutter API/client focus from `app/`: `flutter test test/profile/data/profile_api_client_test.dart test/shared/rich_text/facet_generator_test.dart`.

## 10. Sequencing And Guardrails

- First TDD step: write `IT-001` proving authenticated `GET /v1/facets/mentions` is registered under `/v1`, enforces auth/device headers, and returns `{items:[...]}` for a valid empty/fake result.
- Suggested implementation sequence:
  1. AppView route skeleton, request validation, DTOs, and handler contracts.
  2. Identity cache migration/store and mention suggestion/exact resolve store logic.
  3. Hashtag suggestion query logic.
  4. Profile initialization identity-cache upsert for newly created/initialized Craftsky users.
  5. CLI identity-cache backfill command for existing users.
  6. Flutter AppView-backed repositories and provider wiring.
  7. Flutter final post mention-resolution fallback tests and facet-generator parser refactor.
  8. Remove profile `descriptionFacets` from model/API/save/editor.
  9. Plain profile-bio render-time parser/tap behavior.
  10. Run focused and full relevant test commands; perform manual smoke checks.
- Dependencies between work items:
  - Flutter repository endpoint tests depend on final AppView URL/JSON shapes in §5.
  - Profile-bio parser should land before refactoring `FacetGenerator` if the implementation centralizes token detection.
  - Removing `descriptionFacets` from generated model/repository APIs will require updating fakes and tests in the same TDD slice.
- Guardrails:
  - Do not modify `lexicon/`; no ADR or lexgen is required for this change.
  - Do not store handles on `bluesky_profiles`.
  - Do not add SQL migration-time network work.
  - Do not rely on `cli identity-cache backfill` for newly created/initialized users; profile initialization must upsert their identity cache row.
  - Do not broaden to general account search, non-Craftsky mention targets, or background refresh jobs.
  - Do not put PDS tokens or atproto credentials in Flutter.
  - Do not reintroduce `descriptionFacets` in AppView profile request/response contracts or Flutter profile models.
  - Keep AppView endpoint `404 mention_not_found` tests separate from Flutter generator “no facet emitted” tests.
  - Preserve AT Protocol post facet UTF-8 byte offsets during parser refactors.
- Out of scope:
  - Full Bluesky parser parity for bios.
  - Non-Craftsky mention suggestions/resolution.
  - Hashtag search pages or profile discovery.
  - Continuous background identity refresh.
  - Live atproto network tests in automated CI.

## 11. Risks And Open Questions

| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Non-blocking | Exact SQL index strategy for substring display-name search may need tuning as data grows | Autocomplete could become slower on large profile sets | Start with bounded results, handle prefix index, and Craftsky-only joins; defer trigram/full-text extension decisions unless performance evidence demands it |
| CPQ-002 | Non-blocking | Resolver transient failures are intentionally collapsed to `mention_not_found` for exact mention endpoint contract | Users may miss a mention facet during transient identity outage rather than seeing a hard post failure | Matches AC-006/EC-006; log server-side without exposing tokens or resolver internals |
| CPQ-003 | Non-blocking | Refactoring `FacetGenerator` to share parser logic may change overlap ordering accidentally | Could break post facets while fixing bios | Keep existing `facet_generator_test.dart` fixtures and add REG-002 before refactor |
| CPQ-004 | Non-blocking | `descriptionFacets` removal touches generated Dart mapper/provider code and several fakes | Partial removal could leave stale generated fields or test-only APIs | Update generated files and fakes in the same TDD slice; REG-004 fails if any profile model/render API still accepts facets |
| CPQ-005 | Non-blocking | Adding identity-cache upsert to OAuth/profile initialization crosses package boundaries between `auth` and AppView API/cache services | A careless implementation could create an import cycle or make auth depend directly on API internals | Use a narrow injected interface/closure for cache upsert; keep `auth` independent of `api` concrete types |

Blocking open questions: None identified.

## 12. Handoff To TDD Builder

- Coding plan: `04-coding-plan.md`
- TDD execution plan: `05-implementation-plan.md`
- Start with test: `IT-001` for authenticated `GET /v1/facets/mentions` route/response/auth-device contract, followed by `UT-001` request validation.
- Focused first command: `just test` after `just dev-d` is running, or narrower Go package execution from `appview/` if the TDD builder chooses a focused loop.
- Notes:
  - Treat `01-requirements.md`, `02-acceptance-tests.md`, `03-document-review.md`, and this file as source of truth.
  - Preserve resolved decisions: over-limit rejects with `400 validation_error`; backfill command is `cli identity-cache backfill`; newly created/initialized Craftsky users upsert identity-cache rows as part of profile initialization; bios are plain text with render-time clickable parsing; exact resolve endpoint and Flutter no-facet fallback are distinct contracts.
