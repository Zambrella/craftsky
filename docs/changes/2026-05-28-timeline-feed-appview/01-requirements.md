# Requirements: Timeline Feed AppView

## 1. Initial Request

Implement the timeline/feed piece of work as an AppView-only chunk, including endpoints. For this slice, the feed should be a basic chronological feed of posts from accounts the signed-in user follows. The requirements should also note that the API and code should not become too rigid for later feed variants such as craft-specific project feeds, custom feeds from lists, search, and related discovery surfaces.

## 2. Current Codebase Findings

- Relevant files:
  - `docs/roadmap.md` lists `GET /v1/feed/timeline` as open v1 AppView/API work and the Flutter feed screen as a separate open client task.
  - `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md` already reserves `GET /v1/feed/timeline` for a chronological feed of followed accounts, and defines `/v1/`, authenticated requests, error envelopes, and opaque cursor pagination.
  - `atproto-craft-social-app-reference.md` describes the basic feed query as a join between posts and follows, ordered newest first, with a later optimisation path for materialised feed tables and caching.
  - `docs/superpowers/specs/2026-05-04-feed-post-indexing-design.md` defines `indexed_at` as the canonical chronological-feed ordering key for `craftsky_posts`.
  - `appview/internal/routes/routes.go` has authenticated `/v1/` post/profile routes but no `/v1/feed/timeline` route today.
  - `appview/internal/api/post.go`, `post_store.go`, and `post_response.go` implement existing post read/list handlers, opaque seek cursors, `PostResponse`, author hydration, and engagement summary hydration.
  - `appview/internal/api/follow_store.go` and `appview/migrations/000012_atproto_follows.up.sql` provide active follow graph storage and followed-DID lookup support.
  - `appview/migrations/000010_craftsky_posts.up.sql` stores `craftsky_posts` with `reply_*`, `quote_*`, `tags`, `created_at`, and `indexed_at` columns plus feed-related indexes.
  - `appview/migrations/000011_craftsky_interactions.up.sql` stores likes and reposts used by existing engagement summary helpers.
- Existing patterns:
  - Flutter reads social data from the AppView; it does not read Craftsky feed data directly from PDSes.
  - AppView `/v1/*` endpoints require `Authorization: Bearer <craftsky-token>` and `X-Craftsky-Device-Id` unless explicitly unauthenticated.
  - Error responses use the shared camelCase envelope `{error, message, requestId}`.
  - List endpoints use `{items, cursor}` with opaque cursors; invalid cursors map to `400 invalid_cursor`.
  - Post-shaped responses use `PostResponse` with embedded author identity/display fields and viewer engagement fields.
  - Existing post list handlers resolve handles for returned authors and use `EngagementSummaries` to batch counts and viewer state.
- Current behavior:
  - Users can create, read, delete, like, repost, and comment/reply on posts through existing AppView endpoints.
  - Profile post and comment lists exist, but there is no authenticated home timeline endpoint.
  - The AppView already indexes posts and follows needed for a basic followed-account timeline.
  - Repost records are indexed as interactions, not as post rows or feed-item reasons.
- Constraints discovered:
  - This chunk is AppView-only; no Flutter UI/data-layer work is in scope.
  - No lexicon changes are required for this timeline read endpoint.
  - Reads must come from the AppView database; PDS calls are not part of the happy-path timeline read.
  - Timeline ordering should use AppView indexing chronology (`indexed_at`) rather than user-declared `created_at`, matching the existing feed-indexing design.
  - Blocks, mutes, reports, moderation filtering, recommendations, ranking, search, custom feeds, and materialised feed tables are not implemented yet.
- Test/build commands discovered:
  - AppView Go tests: `just test` from the repository root after `just dev-d` is running.
  - Focused Go tests can be run from `appview/` with `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./...` when the compose database is available.

## 3. Clarifying Questions And Decisions

### Q1: For this AppView timeline chunk, what should “posts of all types” include?

Answer: Top-level posts, project posts, quote posts, and comments, but not replies.

Decision / implication: Timeline inclusion is defined over `social.craftsky.feed.post` rows from followed accounts. It includes root/top-level posts, project posts because they are the same record type with `project` data, quote posts because they are post rows with `quote_*` fields, and top-level comments. It excludes nested replies/replies-to-comments. Repost records are not separate feed items in this chunk.

### Q2: Should requirements use the direct joined timeline-query approach?

Answer: Confirm recommended.

Decision / implication: Requirements target a direct read query over indexed posts joined to active follows. Materialised feed tables and a generic feed/search framework are deferred, but the design must avoid hard-coding assumptions that would block later project feeds, custom feeds from lists, search, or other feed sources.

## 4. Candidate Approaches

### Option A: Direct Joined Timeline Query (Recommended)

Summary: Add `GET /v1/feed/timeline` backed by a bounded query joining `craftsky_posts` to `atproto_follows` for the authenticated viewer, ordered by `(indexed_at DESC, uri DESC)`, returning existing `PostResponse` items with pagination and engagement hydration.

Pros:
- Delivers the requested AppView endpoint with the smallest coherent scope.
- Reuses existing post response, cursor, auth, and engagement patterns.
- Matches the reference architecture's basic feed query and the current data model.
- Keeps materialised feed tables available as an optimisation later.

Cons:
- Query cost grows with follow graph and post volume until materialisation/caching is introduced.
- Does not provide feed-item reasons for reposts or future custom feeds.
- Follow graph and post indexing lag can make the feed temporarily stale.

Risks:
- If implemented directly in the handler rather than behind a small store/query boundary, future feed variants could require avoidable rewrites.

### Option B: Materialised Home Feed Table Now

Summary: Add per-viewer timeline rows and populate them at write/index time or through backfill, so reads fetch directly from a precomputed feed table.

Pros:
- Better scaling path for high-volume accounts and large follow graphs.
- Could naturally attach feed reasons and source metadata later.
- Decouples feed-read latency from join complexity.

Cons:
- Requires extra migrations, indexer/fan-out logic, backfill semantics, and deletion handling.
- Premature for the current v1 volume and requested basic chronological feed.
- More failure modes around fan-out lag and duplicate feed rows.

Risks:
- Overbuilding this early could slow delivery and create rigidity before custom-feed semantics are understood.

### Option C: Generic Feed/Search Framework Now

Summary: Build an extensible query abstraction for timeline, project feeds, list feeds, search, and future discovery surfaces in one pass.

Pros:
- Makes future feed variants a first-class design concern immediately.
- Could share filtering, pagination, and ordering across multiple surfaces.
- Reduces risk of single-purpose timeline assumptions.

Cons:
- Larger design and test surface than the requested AppView timeline slice.
- Search and project-specific feed requirements are not yet defined enough to bake into a general framework.
- Risk of speculative abstractions that do not match future product decisions.

Risks:
- Over-generalisation may make the simple timeline harder to reason about and test.

## 5. Recommended Direction

Recommended approach: Option A — Direct Joined Timeline Query.

Why: The AppView already has the two required read-side substrates: indexed Craftsky posts and indexed active follows. A direct joined query is the simplest path to the requested chronological followed-account feed, aligns with existing API/post-list conventions, and keeps the larger feed architecture reversible. Requirements include explicit extensibility constraints so future craft-specific feeds, list feeds, search, and materialised-feed optimisations are not blocked by this basic endpoint.

## 6. Problem / Opportunity

Craftsky has post creation, profile post lists, comments, likes, reposts, and follow graph state, but no home timeline for a signed-in user. A basic chronological followed-account feed is necessary before the Flutter feed screen can consume AppView data. This AppView-only slice should expose the endpoint and contract while preserving room for richer feed types later.

## 7. Goals

- G-001: Provide an authenticated AppView endpoint for a signed-in user's basic followed-account timeline.
- G-002: Return timeline items in deterministic reverse chronological AppView index order.
- G-003: Reuse existing post response and engagement conventions so the Flutter client can later render timeline rows without new PDS reads.
- G-004: Keep the implementation shape flexible enough for later project-specific feeds, custom/list feeds, search results, and materialised-feed optimisations.
- G-005: Keep this chunk limited to AppView API/storage/query behavior.

## 8. Non-Goals

- NG-001: Do not build Flutter feed UI, Flutter data layer, or client pagination behavior in this chunk.
- NG-002: Do not add algorithmic ranking, recommendations, popularity ordering, or non-chronological sorting.
- NG-003: Do not implement craft-specific project feeds, custom feeds from lists, search endpoints, hashtag feeds, or discovery feeds in this chunk.
- NG-004: Do not implement a materialised feed table, fan-out-on-write, Redis cache, or background feed generation unless a narrow supporting index is proven necessary during implementation.
- NG-005: Do not change atproto lexicon schemas or generated lexicon types.
- NG-006: Do not include repost records as separate timeline items or feed reasons in this chunk.
- NG-007: Do not implement blocks, mutes, reports, moderation labels, or per-viewer content filtering beyond the followed-account filter in this chunk.
- NG-008: Do not fetch timeline posts from PDSes on request; the AppView indexed database is the read source.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Signed-in viewer | Authenticated Craftsky user requesting their home timeline. | See recent relevant posts/comments from accounts they follow. |
| Followed author | Craftsky account followed by the viewer and authoring indexed posts/comments. | Have eligible content appear in followers' timelines after indexing. |
| AppView | Go API and Postgres read model. | Serve a bounded, chronological, paginated feed from indexed public data. |
| Future Flutter client | Later consumer of this endpoint. | Receive a stable, existing post-shaped API contract it can render without PDS reads. |
| Future feed/search work | Later AppView features such as project feeds, list feeds, search, and custom feeds. | Avoid being blocked by timeline-only abstractions introduced now. |

## 10. Current Behavior

The AppView exposes post CRUD/read endpoints and profile-scoped post/comment lists, but it does not expose `GET /v1/feed/timeline`. A signed-in user has no AppView API surface for a home timeline assembled from the accounts they follow. The database already contains `craftsky_posts` and `atproto_follows` tables that can support the basic join.

## 11. Desired Behavior

The AppView exposes `GET /v1/feed/timeline` as an authenticated `/v1/` JSON endpoint. When called by a signed-in viewer, it returns a paginated page of post-shaped items from accounts the viewer actively follows. Items are ordered newest first by AppView index order. Eligible items include followed authors' top-level posts, project posts, quote posts, and top-level comments; nested replies and repost records are excluded. Each item uses the existing `PostResponse` shape with author display data and viewer engagement state. The response uses existing opaque cursor pagination and standard error handling. The implementation remains intentionally simple while using boundaries/naming that can support later feed sources and filters.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | Craftsky must provide a basic chronological followed-account timeline through the AppView. | A home feed is core social app behavior and the next AppView piece needed before Flutter feed work. | Prompt, roadmap, API architecture spec | AC-001, AC-002, AC-005 |
| BR-002 | Business | Must | The timeline design must preserve room for future feed variants such as craft-specific project feeds, custom/list feeds, and search results. | The user explicitly requested avoiding rigid code/API design that would block likely future feed surfaces. | Prompt, Q2 | AC-010 |
| FR-001 | Functional | Must | The AppView shall expose `GET /v1/feed/timeline` as an authenticated endpoint that also requires `X-Craftsky-Device-Id`. | Matches existing `/v1/` endpoint conventions and the API spec's reserved route. | API architecture spec, discovery | AC-001 |
| FR-002 | Functional | Must | The timeline shall select content only from accounts actively followed by the authenticated viewer according to the AppView's indexed `atproto_follows` state. | Defines the basic followed-account feed and keeps reads inside AppView. | Prompt, reference doc, codebase | AC-002, AC-003 |
| FR-003 | Functional | Must | The timeline shall return eligible indexed `social.craftsky.feed.post` rows from followed accounts: top-level posts, project posts, quote posts, and top-level comments. | Captures the clarified “posts of all types” scope. | Q1 | AC-004 |
| FR-004 | Functional | Must | The timeline shall exclude nested replies/replies-to-comments and shall exclude repost records as separate timeline items. | Keeps the chunk bounded and avoids expanding the response shape to feed reasons. | Q1, non-goals | AC-004 |
| FR-005 | Functional | Must | Timeline items shall be ordered by `indexed_at DESC` with a deterministic `uri DESC` tie-breaker. | Matches existing feed-indexing chronology rationale and prevents unstable pagination. | Feed indexing spec, discovery | AC-005, AC-006 |
| FR-006 | Functional | Must | The timeline response shall use the existing list shape `{items, cursor}` with `cursor` omitted when there are no more results. | Maintains API consistency and gives the later Flutter client a familiar contract. | API architecture spec, existing handlers | AC-006, AC-008 |
| FR-007 | Functional | Must | The timeline shall support `limit` and `cursor` query parameters with the existing default/max limit behavior and opaque cursor semantics. | Bounded pagination is required for list endpoints. | API architecture spec, existing post handlers | AC-006, AC-009 |
| FR-008 | Functional | Must | Each timeline item shall use the existing `PostResponse` wire shape, including author identity/display fields, timestamps, reply/quote fields as applicable, image views as applicable, tags, and viewer engagement summary fields. | Reuse avoids creating a competing post response contract and lets Flutter render without PDS reads. | Existing post API, discovery | AC-007 |
| FR-009 | Functional | Must | Timeline reads shall use AppView-indexed Postgres data for posts, follows, profile display fields, and engagement summaries; the happy path shall not fetch posts from PDSes. | Preserves the AppView read architecture and avoids exposing PDS-token concerns to timeline reads. | AGENTS.md, reference doc | AC-011 |
| FR-010 | Functional | Must | Invalid timeline cursors shall return `400` with error code `invalid_cursor` using the standard error envelope. | Matches existing list endpoint error behavior. | Existing handlers, API architecture spec | AC-009 |
| FR-011 | Functional | Should | Empty timelines, including users who follow no one or whose followed accounts have no eligible posts, should return `200` with `items: []` and no `cursor`. | Empty feed states should be normal responses, not errors. | Discovery | AC-008 |
| FR-012 | Functional | Should | Handle-resolution failures for returned authors should use the same `identity_unavailable` error behavior as existing post/profile endpoints. | Keeps identity failures consistent across post-shaped responses. | Existing handlers | AC-012 |
| NFR-001 | Non-functional | Must | The endpoint must follow existing `/v1/` API conventions: camelCase JSON, authenticated-device middleware, shared error envelope, and opaque cursor pagination. | Maintains API consistency. | AGENTS.md, API specs | AC-001, AC-006, AC-009, AC-012 |
| NFR-002 | Non-functional | Should | The timeline query should remain bounded and should use existing or narrow supporting indexes for followed-author filtering and reverse chronological ordering. | Reduces risk of unbounded scans while avoiding premature materialised-feed work. | Discovery, reference doc | AC-013 |
| NFR-003 | Non-functional | Should | Timeline code should introduce a clear feed store/query boundary rather than embedding timeline-specific SQL and response assembly directly in route registration. | Makes later feed variants and tests easier without building the whole future feed framework now. | Prompt, Q2, recommended approach | AC-010 |
| RULE-001 | Business rule | Must | A followed account for this endpoint is an active `atproto_follows` row where `did` is the authenticated viewer and `subject_did` is the candidate author. | Defines follow semantics for the feed. | Existing follow schema, prompt | AC-002, AC-003 |
| RULE-002 | Business rule | Must | A top-level post has no `reply_root_uri` and no `reply_parent_uri`; a top-level comment has both reply fields present and `reply_root_uri = reply_parent_uri`; a nested reply has both reply fields present and `reply_parent_uri <> reply_root_uri`. | Makes the user's “comments but not replies” decision testable against the existing schema. | Q1, codebase | AC-004 |
| RULE-003 | Business rule | Must | Project posts and quote posts are not special feed item types in this chunk; they are eligible when their `craftsky_posts` row otherwise matches the followed-account and inclusion rules. | Keeps the response on existing `PostResponse` and avoids project/feed-reason expansion now. | Q1, lexicon, codebase | AC-004, AC-007 |
| RULE-004 | Business rule | Must | Repost records in `craftsky_reposts` do not cause separate timeline items in this chunk. | Reposts need feed reasons/attribution that are out of scope for this basic feed. | Q1, non-goals | AC-004 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-001, NFR-001 | Given a request to `GET /v1/feed/timeline`, when the request lacks valid authentication or the required device ID header, then AppView returns the existing authenticated-device error behavior; given both are valid, the request reaches the timeline handler. |
| AC-002 | BR-001, FR-002, RULE-001 | Given the viewer follows author A but not author B, and both authors have eligible indexed posts, when the viewer requests the timeline, then items authored by A may appear and items authored by B do not appear. |
| AC-003 | FR-002, RULE-001 | Given an `atproto_follows` row exists for a different follower or an inactive/deleted follow is absent from active follow storage, when the viewer requests the timeline, then that relationship does not make the candidate author's posts eligible. |
| AC-004 | FR-003, FR-004, RULE-002, RULE-003, RULE-004 | Given a followed author has a root post, project post, quote post, top-level comment, nested reply, and repost record, when the viewer requests the timeline, then the root/project/quote/comment rows are included according to ordering and pagination, while the nested reply and repost record are excluded. |
| AC-005 | BR-001, FR-005 | Given multiple eligible timeline rows with different `indexed_at` values and at least two rows sharing the same `indexed_at`, when the timeline is returned, then rows are ordered by `indexed_at` descending and ties by `uri` descending. |
| AC-006 | FR-005, FR-006, FR-007, NFR-001 | Given more eligible rows exist than the requested `limit`, when the viewer requests the first page and then requests the returned `cursor`, then the second page continues after the last item of the first page without duplicates or skipped eligible rows under the same dataset. |
| AC-007 | FR-008, RULE-003 | Given eligible rows include author display data, images, tags, reply/quote fields, and engagement state, when the timeline response is built, then each item uses the existing `PostResponse` field names and values consistent with other post endpoints. |
| AC-008 | FR-006, FR-011 | Given the viewer follows no one or followed accounts have no eligible rows, when the viewer requests the timeline, then AppView returns `200` with `items: []` and omits `cursor`. |
| AC-009 | FR-007, FR-010, NFR-001 | Given a malformed or otherwise invalid `cursor`, when the viewer requests the timeline, then AppView returns `400` with the standard error envelope and `error: "invalid_cursor"`. |
| AC-010 | BR-002, NFR-003 | Given the code/design is reviewed for this chunk, when future feed variants are considered, then timeline selection, pagination, and response assembly are separated enough that adding a project filter, list-author source, or search-backed source does not require changing the public timeline response contract. |
| AC-011 | FR-009 | Given the timeline happy path is exercised in tests with AppView store fakes or database fixtures, when posts are returned, then the endpoint does not require PDS record fetches to assemble timeline items. |
| AC-012 | FR-012, NFR-001 | Given handle resolution fails for an author that would be returned, when the timeline handler builds the response, then it returns a standard error envelope consistent with existing post/profile identity failure behavior. |
| AC-013 | NFR-002 | Given representative followed-author and post fixtures larger than one page, when the timeline store query is tested or reviewed, then it uses bounded `LIMIT` pagination and indexed predicates/order keys rather than loading unbounded rows for client-side filtering. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | Viewer follows no accounts. | Return `200` with `items: []` and no `cursor`. | FR-011 |
| EC-002 | Viewer follows accounts with only nested replies. | Return no items for those nested replies. | FR-004, RULE-002 |
| EC-003 | Followed author has a top-level comment on an indexed post. | Include the comment as a normal `PostResponse` item. | FR-003, RULE-002 |
| EC-004 | Followed author has a project post whose project fields are not materialised in response columns. | Include the row as a normal `PostResponse`; project-specific response fields are not added in this chunk. | FR-003, RULE-003 |
| EC-005 | Followed author reposts someone else's post. | Do not create a separate timeline item for the repost record. | FR-004, RULE-004 |
| EC-006 | Cursor is syntactically invalid or has the wrong payload shape. | Return `400 invalid_cursor`. | FR-010 |
| EC-007 | A followed author's handle cannot be resolved during response hydration. | Return identity failure behavior consistent with existing post list endpoints. | FR-012 |
| EC-008 | New posts arrive between page requests. | Opaque seek cursor semantics apply; no offset/page-number contract is promised. | FR-005, FR-007 |

## 15. Data / Persistence Impact

- New fields: None required by requirements.
- Changed fields: None.
- Migration required: Not expected for functional scope because `craftsky_posts` and `atproto_follows` already exist. A narrow supporting index may be added during implementation if query-plan/test evidence shows the existing indexes are insufficient.
- Backwards compatibility: Additive API endpoint only. Existing post/profile/follow endpoints and response shapes should not change.
- Data source: Existing `craftsky_posts`, `atproto_follows`, `bluesky_profiles`, `craftsky_likes`, `craftsky_reposts`, and related indexed tables.

## 16. UI / API / CLI Impact

- UI: None in this chunk. Flutter feed screen and client pagination are separate future work.
- API: Adds `GET /v1/feed/timeline?limit=<n>&cursor=<opaque>` returning `{items: PostResponse[], cursor?: string}`.
- CLI: No required CLI change. Existing request tooling may be able to hit the endpoint once route exists.
- Background jobs: No new background job, fan-out, cache warmer, or backfill required in this chunk.

## 17. Security / Privacy / Permissions

- Authentication: Required through existing AppView auth middleware.
- Authorization: The timeline is scoped to the authenticated viewer's active indexed follow graph. Callers cannot request another user's timeline by parameter in this chunk.
- Sensitive data: The endpoint serves public indexed post/follow/profile data only; it must not expose PDS access/refresh tokens or private AppView state.
- Abuse cases: This chunk does not implement blocks, mutes, reports, moderation labels, rate limits, or content takedown filtering. Those remain future moderation/rate-limit work and should be called out before public launch.

## 18. Observability

- Events: No product analytics required in this AppView-only requirements chunk.
- Logs: Timeline handler/store should follow existing logging style with request/run ID, viewer DID, limit, cursor presence, row count, and error code where useful. Logs must not include bearer tokens.
- Metrics: No new metrics required, but row count/latency would be useful future observability for feed performance.
- Alerts: None required for this slice.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Direct joined query becomes slow for users with large follow graphs or high-volume followed accounts. | Timeline endpoint latency increases as data grows. | Keep pages bounded, use indexed predicates/order keys, and defer materialised feed tables until volume proves need. |
| RISK-002 | Comment-vs-reply semantics are misunderstood during implementation. | Timeline includes too many nested replies or excludes desired top-level comments. | Use `RULE-002` and acceptance tests with root post, top-level comment, and nested reply fixtures. |
| RISK-003 | Handle resolution for many distinct authors causes latency or endpoint failure. | Timeline pages may fail with `identity_unavailable` or be slower than expected. | Batch unique author handling where possible; keep page sizes bounded; use existing resolver behavior and tests. |
| RISK-004 | Firehose/indexing lag makes follows or posts appear stale. | Recently followed accounts or newly-created posts may not appear immediately. | Document AppView indexed state as the source of truth; future UX can handle eventual consistency. |
| RISK-005 | Blocks/mutes/moderation filtering are absent. | Timeline may show content a later product version should suppress. | Keep out of this scope but record as future moderation work before public launch. |
| RISK-006 | Timeline-only implementation choices make future project/list/search feeds harder. | Later feed work requires rewrites or incompatible API changes. | Require a feed store/query boundary and preserve existing post response/cursor conventions. |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | `atproto_follows` represents the active follow graph needed for the basic home timeline. | Timeline would need a different graph source or active/deleted-state model. |
| ASM-002 | Top-level comments are represented by rows where `reply_root_uri = reply_parent_uri`; nested replies have a different parent than root. | Inclusion/exclusion SQL and tests would need to change. |
| ASM-003 | Project posts and quote posts can be rendered with the current `PostResponse` without new project-specific or quote-expanded fields. | The API response shape would need additional fields and possibly project-field materialisation. |
| ASM-004 | Repost activity should not appear as a separate timeline item until feed reasons/attribution are designed. | A larger feed-item envelope would be needed sooner. |
| ASM-005 | AppView-only requirements are sufficient for this chunk; Flutter consumption will be handled by a later workflow stage/change. | Scope would need to expand to include app models, API client, providers, and UI tests. |

## 21. Open Questions

- [ ] Non-blocking: When blocks, mutes, reports, and moderation labels land, which filters should apply to home timeline reads?
- [ ] Non-blocking: Should future repost support use a feed-item envelope with reasons, or continue returning only post-shaped items on timeline?
- [ ] Non-blocking: What exact query/API shape should craft-specific project feeds, custom/list feeds, and search use?
- [ ] Non-blocking: When project fields are materialised, should timeline support optional project filters directly, or should those live under separate feed/search endpoints?

## 22. Review Status

Status: Draft
Risk level: Medium
Review recommended: Yes
Reviewer:
Date: 2026-05-28
Notes: Medium risk because this adds a new user-visible API endpoint and feed semantics that later Flutter work will depend on. Review is recommended before test design, but not required to proceed if the user accepts the documented scope and risks.

## 23. Handoff To Test Design

- Requirements file: `docs/changes/2026-05-28-timeline-feed-appview/01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover requirement IDs:
  - `BR-001`, `BR-002`
  - `FR-001` through `FR-010`
  - `NFR-001`
  - `RULE-001` through `RULE-004`
- Suggested test levels:
  - Unit tests for cursor handling, inclusion/exclusion classification, and response assembly.
  - Store/query integration tests against Postgres fixtures for followed/unfollowed authors, comments vs replies, ordering, and pagination.
  - Handler/route tests for auth/device requirements, response envelope behavior, invalid cursors, empty timelines, and happy-path response shape.
  - Regression tests ensuring existing post/profile routes still return the same `PostResponse` shape.
- Blocking open questions: None.
- Review recommendation: Because risk is medium and feed semantics are foundational for later Flutter work, review of `01-requirements.md` is recommended before test design.
