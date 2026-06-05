# Requirements: AppView Facet Endpoints And Plain Profile Bios

## 1. Initial Request

Replace the Flutter app's fake mention and hashtag facet endpoints with real AppView endpoints, and change profile descriptions to match Bluesky behavior: profile descriptions are plain text with no stored facets, while the frontend still renders hashtags, profiles, and links as clickable elements.

## 2. Current Codebase Findings

- Relevant files:
  - Flutter mock facet seams: `app/lib/shared/rich_text/data/facet_suggestion_repository.dart`, `app/lib/shared/rich_text/data/mock_facet_suggestion_repository.dart`, `app/lib/shared/rich_text/providers/facet_suggestion_providers.dart`.
  - Flutter facet generation/rendering: `app/lib/shared/rich_text/facet_generator.dart`, `app/lib/shared/rich_text/widgets/faceted_text.dart`, `app/lib/shared/rich_text/faceted_text_model.dart`, `app/lib/shared/rich_text/facet_action_handler.dart`.
  - Flutter profile bio edit/render: `app/lib/profile/pages/edit_profile_dialog.dart`, `app/lib/profile/models/profile.dart`, `app/lib/profile/widgets/profile_bio.dart`, `app/lib/profile/data/profile_api_client.dart`.
  - AppView routes/API/profile/post storage: `appview/internal/routes/routes.go`, `appview/internal/api/profile_request.go`, `appview/internal/api/profile_response.go`, `appview/internal/api/profile_store.go`, `appview/internal/api/post_store.go`, `appview/internal/api/handle_resolver.go`.
  - AppView profile initialization path: `appview/internal/auth/initialize_profile.go`, `appview/internal/auth/handlers_oauth.go`.
  - AppView data model: `appview/migrations/000008_craftsky_profiles.up.sql`, `appview/migrations/000009_bluesky_profiles.up.sql`, `appview/migrations/000010_craftsky_posts.up.sql`, `appview/migrations/000013_profile_social_summary_indexes.up.sql`.
  - API conventions: `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md`.
- Existing patterns:
  - Flutter AppView calls use Dio from `dioProvider`, JSON/camelCase bodies, and repository/provider seams.
  - AppView `/v1/*` endpoints require Craftsky session auth plus `X-Craftsky-Device-Id`, except explicitly public auth/ops endpoints.
  - Post text already uses real `app.bsky.richtext.facet` JSON for post mentions, tags, and links.
  - Profile writes already target Bluesky `app.bsky.actor.profile.description`, which is plain text and has no facets.
- Current behavior:
  - Mention and hashtag suggestions are mock-backed in Flutter.
  - Current mock mention suggestions are Craftsky-profile-only and sort followed accounts before others.
  - Current mock hashtag suggestions expose `tag` and `postsLast28Days`.
  - Profile edit currently generates `descriptionFacets` client-side, but AppView `PUT /v1/profiles/me` does not accept `descriptionFacets`.
  - Profile bio rendering currently depends on explicit facets to make ranges clickable.
  - AppView stores DIDs and profile fields but does not store handles; profile endpoints resolve handles on demand for known DIDs.
  - OAuth callback/profile initialization ensures Craftsky profile records exist for signed-in users, but this requirements slice must also ensure the new identity cache is populated for newly created/initialized Craftsky users.
- Constraints discovered:
  - Partial handle autocomplete cannot be implemented reliably through the atproto identity resolver alone; it needs a local searchable handle cache because identity resolution is exact-handle/DID oriented, not prefix search.
  - Lexicon changes are not required for this scope; profile description should remain the Bluesky plain `description` field.
  - Hashtag source data already exists in `craftsky_posts.tags` and is indexed with GIN.
- Test/build commands discovered:
  - Go/AppView tests: `just test` from repo root after the compose database is running.
  - Go format/vet: `just fmt`.
  - Flutter tests are under `app/test`; expected implementation should run relevant `flutter test` suites from `app/`.

## 3. Clarifying Questions And Decisions

### Q1: For mention autocomplete, which account set should AppView return?

Answer: Craftsky only.

Decision / implication: Mention suggestions and exact mention resolution must only return accounts that have Craftsky profiles.

### Q2: For hashtag autocomplete, what should AppView return?

Answer: Indexed 28-day counts.

Decision / implication: Hashtag suggestions must be derived from indexed `craftsky_posts.tags`, returning `postsLast28Days` counts for recent usage.

### Q3: How should the edit-profile bio editor behave after removing stored description facets?

Answer: Plain textbox.

Decision / implication: Remove facet autocomplete from profile bio editing. Rendering, not saving, is responsible for clickable bio elements.

### Q4: Should manually typed `@handle` mentions still become real mention facets on post submit?

Answer: Yes, resolve exact handles.

Decision / implication: The AppView must expose or support exact Craftsky handle resolution so the Flutter post facet generator can resolve manually typed handles to DIDs.

### Q5: Are local searchable handles acceptable for mention autocomplete?

Answer: Yes, add handle cache.

Decision / implication: The AppView should include a searchable local handle cache/index to support partial mention autocomplete.

### Q6: Should the cache live in `bluesky_profiles` or a separate table?

Answer: Separate cache table.

Decision / implication: Requirements should call for a separate identity/handle cache table, because handles are identity metadata, not `app.bsky.actor.profile` record metadata.

### Q7: Should the requirements lock the concrete AppView endpoint paths?

Answer: Yes, lock `/v1/facets/*` paths.

Decision / implication: The requirements specify `GET /v1/facets/mentions`, `GET /v1/facets/mentions/resolve`, and `GET /v1/facets/hashtags`.

### Q8: What response shape should facet endpoints use?

Answer: Suggestion endpoints return object-wrapped `items`; exact resolve returns one object or a standard 404.

Decision / implication: Mention and hashtag suggestions return `{ "items": [...] }`. Exact mention resolve returns a minimal resolve object on success or `404 mention_not_found` in the standard AppView error envelope on failure.

### Q9: What identity-cache freshness window should be used?

Answer: 24 hours.

Decision / implication: Cached identities are fresh for autocomplete for 24 hours. Exact resolve refreshes missing or stale cache entries.

### Q10: What mention suggestion ranking and limits should be used?

Answer: Lock ranking with default limit 10 and max 25.

Decision / implication: Mention suggestions are ordered by followed-first, stronger prefix matches before weaker substring matches, and handle ascending as the final tie-breaker.

### Q11: What should hashtag counts include?

Answer: Root posts only.

Decision / implication: `postsLast28Days` counts indexed root posts only, excluding comments/replies.

### Q12: What hashtag casing should suggestions return?

Answer: Lowercase canonical.

Decision / implication: Hashtag suggestions return normalized lowercase tags from `craftsky_posts.tags`, without leading `#`.

### Q13: What plaintext bio parsing rules should be used?

Answer: Mirror the existing post facet generator token rules.

Decision / implication: Profile bio rendering detects dotted `@handle` mentions, Unicode hashtag tokens, and HTTP/S or bare-domain links consistently with the post facet generator; malformed tokens remain plain text.

### Q14: Should `descriptionFacets` remain in Flutter profile APIs?

Answer: Remove entirely.

Decision / implication: `descriptionFacets` should be removed from Flutter profile model, profile update body, save flow, and bio widget API rather than merely ignored.

### Q15: What exact mention resolve error should be used?

Answer: `404 mention_not_found`.

Decision / implication: Exact resolve failures use the endpoint-specific `mention_not_found` error code in the standard AppView error envelope.

### Q16: What exact mention resolve success shape should be used?

Answer: Minimal resolve object.

Decision / implication: Success returns `did`, canonical `handle`, and `isCraftskyProfile`; display/avatar/follow fields remain suggestion-only.

### Q17: Should existing Craftsky profiles get a handle-cache population path?

Answer: Yes, require a bounded backfill path.

Decision / implication: The migration creates cache schema, while a CLI/ops/bootstrap task or equivalent bounded process populates handles for existing Craftsky profiles without doing network work inside SQL migrations.

### Q18: What query bounds should suggestion endpoints enforce?

Answer: Require at least 1 non-whitespace character, max 64 characters.

Decision / implication: Empty or whitespace-only query returns an empty `items` list. Over-64-character queries fail validation.

### Q19: What hashtag ranking and limits should be used?

Answer: Lock ranking with default limit 10 and max 25.

Decision / implication: Hashtag suggestions sort by `postsLast28Days` descending, then tag ascending.

### Q20: How should profile bio mention taps work?

Answer: Navigate by visible handle.

Decision / implication: Profile bio mentions do not pre-resolve before navigation; the profile route handles missing or invalid targets.

### Q21: What link schemes should plaintext bio detection allow?

Answer: HTTP/S only.

Decision / implication: Bio rendering links explicit `http://` and `https://` URLs plus bare domains normalized to `https://`; other schemes remain plain text.

### Q22: Should mention suggestions include accounts missing display/avatar metadata?

Answer: Include with omitted optional fields.

Decision / implication: A Craftsky profile with a cached handle remains suggestible even if `displayName` or `avatar` is unavailable; unknown optional fields are omitted following existing AppView `omitempty` style.

### Q23: When should exact mention resolve be used?

Answer: For final post facet generation, not per-keystroke autocomplete or profile bio rendering.

Decision / implication: Autocomplete uses the suggestion endpoint while typing; exact resolve is used by the post facet generator when final submitted text is converted into mention facets.

### Q24: How should suggestion endpoints handle `limit > 25`?

Answer: Reject with validation error.

Decision / implication: `limit` values greater than 25 return a standard AppView `400 validation_error` envelope. Empty or whitespace-only queries still return `{items:[]}` rather than an error.

### Q25: What concrete identity-cache backfill command should be planned?

Answer: `cli identity-cache backfill`.

Decision / implication: The bounded AppView CLI path for existing Craftsky profiles is `cli identity-cache backfill`, run in dev via the appview container as `docker compose exec appview /app/cli identity-cache backfill`. It defaults to a batch limit of 100 and supports `--limit <n>` for bounded runs.

### Q26: Should profile bio parsing target full Bluesky parity?

Answer: No, not for this change.

Decision / implication: Plain bio rendering should use Craftsky's supported token behavior by centralizing or mirroring the existing post facet generator rules. Tests must lock explicit fixtures for dotted handles, Unicode hashtags, HTTP/S URLs, bare domains normalized to HTTPS, unsupported schemes, malformed handles/URLs, and overlapping URL fragment/hashtag cases. Full Bluesky parser parity is deferred out of scope.

### Q27: How should exact mention resolution test contracts be separated?

Answer: Keep AppView endpoint behavior and Flutter facet-generation fallback behavior separate.

Decision / implication: AppView handler/store tests assert success objects or `404 mention_not_found` envelopes. Flutter facet-generator tests assert that `mention_not_found` maps to no emitted mention facet while post submission continues for the rest of the text.

### Q28: Should a newly created/initialized Craftsky user be inserted into the identity cache immediately?

Answer: Yes.

Decision / implication: The AppView profile creation/initialization path for an authenticated Craftsky user must resolve and upsert that user's current DID/handle into the separate identity cache. The bounded backfill command remains necessary for pre-existing Craftsky profiles, but new users should not wait for backfill before appearing in mention autocomplete. This cache write must not store handles on `bluesky_profiles` and must not perform network work inside SQL migrations.

## 4. Candidate Approaches

### Option A: Dedicated facet endpoints plus separate identity cache

Summary: Add dedicated `/v1` facet suggestion endpoints, a separate DID/handle identity cache for searchable Craftsky handles, real Flutter repositories, and render-time plaintext bio detection.

Pros:
- Best matches current composer UX while replacing fake data.
- Supports partial handle autocomplete and exact manually typed mention resolution.
- Keeps profile descriptions Bluesky-compatible plain text.
- Keeps endpoint scope specific to facet autocomplete instead of broad account search.

Cons:
- Requires a migration and cache refresh/staleness strategy.
- Crosses AppView API, persistence, and Flutter UI/data layers.

Risks:
- Cached handles can become stale if not refreshed or validated carefully.
- Poor query limits/ranking could make autocomplete slow or noisy.

### Option B: Exact resolver only, no handle cache

Summary: Add exact handle resolution and hashtag suggestions, but avoid searchable handle storage.

Pros:
- Smaller AppView change.
- Avoids identity cache migration.

Cons:
- Does not fully replace mention autocomplete because partial handle search remains unreliable.
- Degrades current `@ali` suggestion behavior.

Risks:
- Users may see no useful mention suggestions while typing, despite real exact resolution at submit time.

### Option C: General account search API reused by facets

Summary: Add a broader account/profile search endpoint and have facet autocomplete call it.

Pros:
- Potentially reusable for future profile discovery.

Cons:
- Larger product/API scope than replacing fake facet endpoints.
- Blurs ranking and filtering requirements for autocomplete versus general search.

Risks:
- Delays the focused facets/profile-bio change by pulling in broader discovery UX decisions.

## 5. Recommended Direction

Recommended approach: Option A, with a separate AppView identity/handle cache table and an immediate identity-cache upsert from the profile creation/initialization path.

Why: It is the smallest approach that preserves the current user-facing composer autocomplete behavior, supports exact mention facets for manually typed handles, uses indexed post tags for hashtag suggestions, makes profile descriptions plain text without expanding into general profile search, and prevents newly onboarded Craftsky users from being absent from autocomplete until an operator backfill runs.

## 6. Problem / Opportunity

The Flutter app currently demonstrates mention and hashtag facet UX with mock data. This blocks end-to-end composer behavior against real Craftsky accounts and indexed tags. Separately, profile descriptions are drifting from Bluesky semantics by carrying client-generated `descriptionFacets`, even though the AppView and Bluesky profile record model treat descriptions as plain text. This change makes facet autocomplete real while simplifying profile descriptions and preserving clickable bio rendering.

## 7. Goals

- G-001: Replace mock mention and hashtag suggestions with authenticated AppView-backed data.
- G-002: Preserve real post facet generation for mentions, links, and tags.
- G-003: Make profile descriptions plain text only, matching Bluesky `app.bsky.actor.profile.description` behavior.
- G-004: Preserve clickable rendering for profile bio links, hashtags, and profile mentions through render-time parsing.
- G-005: Avoid broad profile discovery/search scope beyond what facet autocomplete requires.

## 8. Non-Goals

- NG-001: Do not change Craftsky or Bluesky lexicon schemas.
- NG-002: Do not add facets to `app.bsky.actor.profile.description`.
- NG-003: Do not build a full general-purpose account search or hashtag search product surface.
- NG-004: Do not include non-Craftsky accounts in mention autocomplete or mention resolution for this change.
- NG-005: Do not change post record rich-text facet semantics.
- NG-006: Do not introduce PDS tokens or atproto credentials into the Flutter app.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Craftsky user composing a post | Authenticated Flutter app user writing a post | See real Craftsky account and hashtag suggestions; submit valid post facets. |
| Craftsky user editing profile | Authenticated Flutter app user editing their profile | Edit a plain bio without facet metadata being generated or saved. |
| Craftsky profile viewer | User viewing profile bios | Tap detected links, hashtags, and profile mentions in plain bio text. |
| AppView | Go service serving Flutter and indexing records | Serve bounded suggestion endpoints from indexed/cached data without direct PDS token exposure to the app. |

## 10. Current Behavior

The post composer uses mock account and hashtag repositories. Account suggestions are filtered to mock Craftsky profiles. Hashtag suggestions return mock tags and `postsLast28Days` values. The post facet generator scans submitted text and produces AT Protocol rich-text facets. Profile edit uses the same facet generator to produce `descriptionFacets`, but AppView profile writes do not accept or persist that field. Profile bio display renders with `FacetedText`, so clickable bio elements depend on explicit facets rather than plain-text detection.

## 11. Desired Behavior

The post composer calls real AppView endpoints for Craftsky-only mention suggestions, final-submit exact handle resolution, and hashtag suggestions. AppView supports partial mention search through a separate identity/handle cache table and returns bounded, authenticated JSON responses. Existing Craftsky profiles can be populated through bounded backfill, while newly created/initialized Craftsky users are inserted into the identity cache as part of the AppView profile initialization path. Hashtag suggestions come from indexed root-post tags and include lowercase canonical tags with `postsLast28Days`. Profile bio editing is a plain text field and profile updates do not send or model `descriptionFacets`. Profile bio display parses plain text at render time, detects supported `@handle`, `#hashtag`, and HTTP/S link ranges using rules that mirror post facet generation, styles them as clickable, and dispatches the same destinations as existing facet rendering where applicable.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | The app shall replace mock mention and hashtag autocomplete data with real AppView-backed data. | Enables end-to-end composer behavior against real indexed Craftsky data. | Prompt, discovery | AC-001, AC-002, AC-003 |
| BR-002 | Business | Must | Profile descriptions shall be saved and served as plain text without stored facet metadata. | Matches Bluesky profile description behavior. | Prompt, Q3 | AC-008, AC-009 |
| BR-003 | Business | Must | Plain profile bios shall still render supported links, profile mentions, and hashtags as clickable elements. | Preserves rich readable bio UX without storing facets. | Prompt | AC-010, AC-011 |
| FR-001 | Functional | Must | The AppView shall provide `GET /v1/facets/mentions?q=<query>&limit=<n>` as an authenticated mention suggestion endpoint that searches Craftsky-profile-only accounts by cached handle and display name. | Replaces mock account suggestions and preserves current scope. | Q1, Q5, Q6, Q7, API conventions | AC-001, AC-004, AC-005, AC-014 |
| FR-002 | Functional | Must | Mention suggestion responses shall be `{items:[...]}` and each item shall include DID, handle, `isCraftskyProfile`, and `viewerIsFollowing`, with optional display name and avatar URL omitted when unknown. | Matches Flutter repository needs, current suggestion UI, and existing `omitempty` response style. | Codebase discovery, Q8, Q22 | AC-001, AC-005, AC-015 |
| FR-003 | Functional | Must | The AppView shall provide `GET /v1/facets/mentions/resolve?handle=<handle>` for exact Craftsky handle resolution used during final post facet generation. | Maintains current final-submit mention facet behavior without per-keystroke resolve calls. | Q4, Q7, Q23 | AC-002, AC-006, AC-016 |
| FR-004 | Functional | Must | Exact mention resolution shall reject or omit handles that do not resolve to a Craftsky profile. | Keeps mention facets Craftsky-only for this change. | Q1, Q4 | AC-006, EC-001 |
| FR-005 | Functional | Must | The AppView shall provide `GET /v1/facets/hashtags?q=<query>&limit=<n>` as an authenticated hashtag suggestion endpoint that searches indexed Craftsky post tags and returns `{items:[...]}` containing `tag` plus `postsLast28Days`. | Replaces mock hashtag suggestions with indexed 28-day counts. | Q2, Q7, Q8, codebase discovery | AC-003, AC-007, AC-014 |
| FR-006 | Functional | Must | Hashtag suggestions shall be based on lowercase canonical `craftsky_posts.tags` usage in root posts from the last 28 days and exclude empty tags. | Ensures counts reflect indexed root-post Craftsky content and existing normalized tag storage. | Q2, Q11, Q12, codebase discovery | AC-003, AC-007, EC-002 |
| FR-007 | Functional | Must | Flutter facet suggestion repositories shall call the new AppView endpoints through the existing authenticated Dio/provider pattern. | Integrates real endpoints without bypassing app auth conventions. | Codebase discovery | AC-001, AC-003, AC-012 |
| FR-008 | Functional | Must | Flutter post facet generation shall continue to generate AT Protocol facets for post text using exact AppView-backed mention resolution of the final submitted text, not hidden selected-mention state. | Preserves post record semantics and rich post rendering. | Prompt, Q4, Q23 | AC-002, AC-006, AC-012, AC-016 |
| FR-009 | Functional | Must | The edit-profile bio UI shall use a plain text input without mention or hashtag autocomplete. | User chose plain textbox for bios. | Q3 | AC-008 |
| FR-010 | Functional | Must | Flutter profile update requests shall not include `descriptionFacets`, and `descriptionFacets` shall be removed from Flutter profile model, save flow, and bio widget APIs. | Aligns client with AppView profile request contract and Bluesky profile semantics. | Prompt, Q14, codebase discovery | AC-009, AC-010 |
| FR-011 | Functional | Must | Profile bio rendering shall detect supported links, `@handle` mentions, and `#hashtag` tokens from plain text using token rules that mirror the existing post facet generator, and render them as clickable styled ranges. | Keeps clickable profile bios without stored facets while maintaining token consistency. | Prompt, Q13 | AC-010, AC-011, EC-003, EC-004 |
| FR-012 | Functional | Must | Click actions for detected profile bio mentions, hashtags, and links shall route consistently with existing facet click actions where the same destination exists, with mention taps navigating optimistically by visible handle. | Avoids duplicate navigation semantics and avoids network work during bio rendering/taps. | Q20, codebase discovery | AC-011, AC-017 |
| FR-013 | Functional | Must | The AppView shall maintain a separate identity/handle cache table rather than storing handles directly on `bluesky_profiles`. | Handles are identity metadata, not profile record metadata. | Q6 | AC-004, AC-013 |
| FR-014 | Functional | Must | The identity cache shall treat entries as fresh for autocomplete for 24 hours, and exact resolve shall refresh missing or stale entries. | Reduces stale-handle risk with a concrete testable freshness rule. | Q9 | AC-013, AC-018, EC-005 |
| FR-015 | Functional | Must | The AppView shall provide `cli identity-cache backfill` as a bounded identity-cache population path for existing Craftsky profiles, separate from SQL migration-time network work. | Ensures autocomplete works for existing accounts without broad per-query network fan-out. | Q17, Q25 | AC-019 |
| FR-016 | Functional | Must | The AppView shall upsert the authenticated user's current DID/handle into the separate identity cache when it creates or initializes that user's Craftsky profile. | Prevents newly created Craftsky users from being absent from real mention autocomplete until an operator backfill runs. | User feedback, Q28 | AC-021 |
| NFR-001 | Non-functional | Must | New AppView endpoints shall follow existing `/v1` API conventions for auth, device ID, camelCase JSON, and error envelopes. | Maintains API consistency. | AGENTS.md, API spec | AC-012 |
| NFR-002 | Non-functional | Should | Suggestion endpoints should apply default limit 10, maximum limit 25, query length bounds, and deterministic ordering suitable for autocomplete. | Prevents slow/noisy autocomplete responses. | Q10, Q18, Q19, discovery risk | AC-005, AC-007, AC-014 |
| NFR-003 | Non-functional | Should | Profile bio plaintext detection should be render-safe, allow only HTTP/S link targets, normalize bare domains to HTTPS, and drop malformed or unsafe ranges without crashing. | Matches existing `FacetedText` safety posture and avoids unsafe link schemes. | Q21, codebase discovery | AC-010, AC-020, EC-003, EC-004 |
| RULE-001 | Business rule | Must | Mention autocomplete and exact mention resolution shall be Craftsky-profile-only. | Confirmed product scope. | Q1 | AC-001, AC-006 |
| RULE-002 | Business rule | Must | Profile descriptions shall not use or persist facets, even when the text contains clickable-looking tokens. | Matches Bluesky bio behavior. | Prompt, Q3 | AC-008, AC-009, AC-010 |
| RULE-003 | Business rule | Must | Hashtag suggestion popularity shall be represented as `postsLast28Days`, counting root posts only and returning lowercase canonical tags. | Preserves existing Flutter display contract while clarifying count scope and casing. | Q2, Q11, Q12, codebase discovery | AC-003, AC-007 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-001, FR-002, RULE-001 | Given an authenticated user types a mention query in the post composer, when the Flutter repository searches accounts, then it calls the AppView mention suggestion endpoint and receives only Craftsky-profile accounts matching the query. |
| AC-002 | BR-001, FR-003, FR-008 | Given a post contains a manually typed `@handle`, when the user submits the post, then the facet generator resolves the exact handle through AppView-backed resolution and emits a mention facet when the handle resolves to a Craftsky DID. |
| AC-003 | BR-001, FR-005, FR-006, RULE-003 | Given an authenticated user types a hashtag query in the post composer, when the Flutter repository searches hashtags, then it calls the AppView hashtag endpoint and receives matching indexed tags with `postsLast28Days`. |
| AC-004 | FR-001, FR-013 | Given the AppView has cached identity data for Craftsky profiles, when the mention suggestion endpoint searches by query, then it can match cached handles without querying the atproto network for every candidate. |
| AC-005 | FR-001, FR-002, NFR-002 | Given more matching mention candidates exist than the requested limit, when suggestions are returned, then the response is bounded and ordered by followed-first, stronger prefix matches before weaker substring matches, and handle ascending as the final tie-breaker. |
| AC-006 | FR-003, FR-004, FR-008, RULE-001 | Given an exact handle resolves to a non-Craftsky account or cannot be resolved, when post facets are generated, then no mention facet is emitted for that handle and submission can continue for the rest of the text. |
| AC-007 | FR-005, FR-006, NFR-002, RULE-003 | Given indexed root posts contain repeated matching tags in the last 28 days, when hashtag suggestions are requested, then lowercase canonical tags are de-duplicated, counted by matching recent root posts, sorted by count descending then tag ascending, and returned with non-negative `postsLast28Days` values. |
| AC-008 | BR-002, FR-009, RULE-002 | Given a user opens the edit-profile bio field, when they type `@handle`, `#tag`, or a URL, then no mention/hashtag autocomplete appears in the bio editor. |
| AC-009 | BR-002, FR-010, RULE-002 | Given a user saves a profile bio containing clickable-looking tokens, when Flutter sends `PUT /v1/profiles/me`, then the request body contains `description` but not `descriptionFacets`. |
| AC-010 | BR-003, FR-010, FR-011, NFR-003, RULE-002 | Given a profile response contains plain `description` text and no facets, when the profile bio renders, then supported links, `@handles`, and `#hashtags` are detected, styled, and made clickable without requiring `descriptionFacets`. |
| AC-011 | BR-003, FR-011, FR-012 | Given a detected bio link, mention, or hashtag is tapped, when a destination is available, then links launch as URLs, mentions navigate to the profile route by visible handle, and hashtags navigate to the tag search route. |
| AC-012 | FR-007, FR-008, NFR-001 | Given Flutter calls the new AppView endpoints, when requests are sent, then they use the existing authenticated Dio configuration and AppView enforces session auth, device ID, camelCase JSON, and standard error envelopes. |
| AC-013 | FR-013, FR-014 | Given a handle is resolved through AppView, when the cache is missing or stale, then the identity cache can be inserted or refreshed without storing the handle on `bluesky_profiles`. |
| AC-014 | FR-001, FR-005, NFR-002 | Given a suggestion endpoint receives an empty or whitespace-only query, then it returns `{items:[]}`; given a query longer than 64 characters, then it returns a standard validation error. |
| AC-015 | FR-002 | Given a suggestible Craftsky profile has a cached handle but no display name or avatar metadata, when mention suggestions are returned, then the item is included with required fields and unknown optional fields omitted. |
| AC-016 | FR-003, FR-008 | Given a user types or edits final post text containing `@handle`, when the post submit flow generates facets, then exact resolve is used during final generation and not on each autocomplete keystroke. |
| AC-017 | FR-012 | Given a detected profile bio mention is tapped, when the visible handle is syntactically valid, then the app navigates to that handle's profile route without pre-resolving it. |
| AC-018 | FR-014 | Given a cached identity entry is less than or equal to 24 hours old, when autocomplete searches it, then the entry is considered fresh; given exact resolve sees a missing or stale entry, then it refreshes the cache on successful resolution. |
| AC-019 | FR-015 | Given existing Craftsky profile rows predate the identity cache, when `cli identity-cache backfill` runs with its default batch limit of 100 or an explicit `--limit <n>`, then it resolves and stores handles for existing Craftsky profiles within the bound without performing network work inside the SQL migration. |
| AC-020 | NFR-003 | Given a profile bio contains an explicit HTTP/S URL or bare domain, when rendered, then the link target is HTTP/S with bare domains normalized to HTTPS; given another URI scheme, then it remains plain text. |
| AC-021 | FR-016 | Given the AppView successfully creates or initializes a Craftsky profile for an authenticated user, when that flow completes, then the user's current DID and canonical handle are upserted into the separate identity cache so mention autocomplete can find the user without waiting for `cli identity-cache backfill`. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | Exact mention handle resolves to a DID with no `craftsky_profiles` row | AppView-backed resolver returns no usable mention target; Flutter emits no mention facet for that token. | FR-004, RULE-001 |
| EC-002 | Hashtag query has no matching indexed recent tags | AppView returns an empty suggestion list, not an error. | FR-006 |
| EC-003 | Profile bio contains malformed URL-like text or invalid handle-like text | Renderer treats invalid tokens as plain text and does not throw. | FR-011, NFR-003 |
| EC-004 | Plaintext bio token detection ranges overlap, such as a URL containing `#fragment` | Renderer chooses a deterministic non-overlapping interpretation and leaves skipped overlapping text safe/plain. | FR-011, NFR-003 |
| EC-005 | Cached handle is stale after a user changes handle | Exact resolution should refresh or invalidate cache for correctness; stale autocomplete display is acceptable only within the documented cache freshness window. | FR-014, RISK-001 |
| EC-006 | Network/API error during autocomplete | Composer remains usable; suggestions can fail closed without blocking typing or post submission except where exact mention resolution is needed for facets. | FR-007, FR-008 |
| EC-007 | Suggestion endpoint receives `limit` greater than 25 | AppView rejects the request with a standard `400 validation_error` envelope and never returns more than 25 items. | NFR-002 |
| EC-008 | Profile initialization succeeds but current handle resolution for identity-cache upsert fails transiently | The profile creation/initialization path should not store a partial cache row or store a handle on `bluesky_profiles`; the failure is logged, and exact resolve or bounded backfill can populate the cache later. | FR-016, FR-014, RISK-006 |

## 15. Data / Persistence Impact

- New fields: None on existing profile/post response models required for profile bio facets.
- New tables: A separate identity/handle cache table is expected for DID/handle search, with freshness metadata such as resolved timestamp or stale window.
- Changed fields:
  - Flutter profile model/update/render flow should remove `descriptionFacets` entirely for profile bios.
  - AppView profile records should remain plain `description`; no `descriptionFacets` should be introduced.
- Migration required: Yes, for the identity/handle cache table and any supporting indexes.
- Data population required: Yes. The AppView profile creation/initialization path should upsert the identity cache for newly created/initialized Craftsky users. `cli identity-cache backfill` should populate the identity cache for existing Craftsky profiles with a default batch limit of 100 and configurable `--limit <n>`. SQL migrations must not perform network handle resolution.
- Backwards compatibility:
  - Additive AppView endpoints are compatible with existing `/v1` conventions.
  - Removing client-sent `descriptionFacets` aligns Flutter with the existing AppView profile request contract.
  - Existing post facet JSON remains unchanged.

## 16. UI / API / CLI Impact

- UI:
  - Post composer mention and hashtag autocomplete should look/behave like the existing mock-backed UI but use real data.
  - Edit-profile bio field should become plain text input with no facet autocomplete overlay.
  - Profile bio display should render detected links, mentions, and hashtags as clickable styled text.
- API:
  - New authenticated `GET /v1/facets/mentions?q=<query>&limit=<n>` endpoint for mention suggestions. Response shape: `{items:[...]}`.
  - New authenticated `GET /v1/facets/mentions/resolve?handle=<handle>` endpoint for exact mention handle resolution during final post facet generation. Success response shape: `{did, handle, isCraftskyProfile}`. Failure: `404 mention_not_found` using the standard error envelope.
  - New authenticated `GET /v1/facets/hashtags?q=<query>&limit=<n>` endpoint for hashtag suggestions. Response shape: `{items:[...]}`.
  - Endpoints must follow camelCase JSON and existing error-envelope conventions.
- Profile initialization: when the AppView creates or initializes an authenticated user's Craftsky profile, it must resolve and upsert that user's current DID/handle into the identity cache.
- CLI: `cli identity-cache backfill` is expected for existing Craftsky profiles, with default batch limit 100 and configurable `--limit <n>`.
- Background jobs: None required by the current direction; profile-initialization cache upsert, opportunistic exact-resolve refresh, and CLI/bootstrap backfill are preferred. A future continuous background refresh job is not in scope.

## 17. Security / Privacy / Permissions

- Authentication: New `/v1` endpoints require the same Craftsky session token and device ID as other authenticated app endpoints.
- Authorization: Mention and hashtag suggestions are visible only to authenticated users in this scope.
- Sensitive data: The Flutter app must not receive or store PDS tokens; AppView continues to mediate reads/writes.
- Abuse cases:
  - Suggestion endpoints should be bounded by limit and query validation to reduce enumeration and load risk.
  - Mention suggestions should not expose non-Craftsky accounts.
  - Link rendering in profile bios must only launch parseable/safe URI values and must not crash on malformed text.

## 18. Observability

- Events: None required.
- Logs: AppView should log endpoint failures consistently with existing handlers, including request/run IDs but not sensitive tokens.
- Metrics: Not required for this stage; future counters for suggestion latency/cache hit rate could be useful.
- Alerts: None required.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Handle cache staleness | Users may see stale handles in autocomplete or exact mention resolution may fail after handle changes. | Keep handles in a separate cache with freshness metadata; refresh on exact resolve and opportunistic reads. |
| RISK-002 | Cross-layer scope | Change touches AppView routes/storage and Flutter repositories/UI. | Keep requirements focused on facet autocomplete and profile bio behavior; avoid broad search features. |
| RISK-003 | Autocomplete performance | Unbounded searches over profiles/tags could degrade typing responsiveness. | Require limits, indexes, and deterministic ordering. |
| RISK-004 | Plaintext bio parsing differences from Bluesky | Rendered clickable ranges may not exactly match Bluesky edge cases. | Define supported token patterns in tests and safely treat ambiguous cases as plain text. |
| RISK-005 | Existing client/server mismatch for `descriptionFacets` | Current Flutter may send a field AppView rejects when non-empty. | Remove `descriptionFacets` from profile save path as part of this change. |
| RISK-006 | Sparse handle cache after migration or profile initialization | Mention autocomplete may miss existing or newly created Craftsky profiles until handles are populated. | Require immediate identity-cache upsert on profile creation/initialization plus bounded CLI/ops/bootstrap backfill for existing Craftsky profiles. |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | `craftsky_posts.tags` contains normalized indexed tags sufficient for hashtag suggestions. | Hashtag endpoint may require additional normalization or indexer changes. |
| ASM-002 | Craftsky-only mention suggestions are desirable for this release. | Requirements would need to expand to non-Craftsky profile hydration/search. |
| ASM-003 | The existing Flutter navigation destinations for facet taps remain valid for profile bio detected tokens. | Bio tap handling may need additional route/product decisions. |
| ASM-004 | A bounded opportunistic identity cache is acceptable without a background refresh job. | Requirements would need background job design and operational acceptance criteria. |
| ASM-005 | Default limit 10 and max limit 25 are sufficient for autocomplete UX. | Requirements would need different pagination/limit behavior. |

## 21. Open Questions

- None blocking.
- Non-blocking for later design: exact SQL table/column names should be finalized during technical design while preserving the requirements above.

## 22. Review Status

Status: Draft

Risk level: Medium

Review recommended: Yes

Reviewer: Pending

Date: 2026-06-02

Notes: Medium risk because this is user-visible, introduces new AppView API endpoints and a persistence migration, and changes profile edit/render behavior. Review is recommended before test design or implementation.

## 23. Handoff To Test Design

- Requirements file: `docs/changes/2026-06-02-appview-facet-endpoints-profile-bios/01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover requirement IDs:
  - Business: BR-001, BR-002, BR-003
  - Functional: FR-001 through FR-016
  - Non-functional: NFR-001
  - Rules: RULE-001, RULE-002, RULE-003
- Suggested test levels:
  - AppView handler tests for mention suggestion, exact mention resolution, hashtag suggestion, auth/device enforcement, error envelopes.
  - AppView store/integration tests for identity cache search/refresh, profile-initialization identity-cache upsert, and hashtag 28-day counts.
  - Flutter repository tests with Dio/mock adapter for real endpoint mapping and error handling.
  - Flutter widget tests for composer autocomplete retaining behavior with repository overrides.
  - Flutter widget/unit tests for plain bio editor and plaintext bio rendering/tap dispatch.
  - Regression tests ensuring profile save does not include `descriptionFacets` and post facet generation still emits valid post facets.
- Blocking open questions: None.
