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
  - The AppView already indexes posts and follows needed for a basic home timeline.
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

Answer: Top-level posts, project posts, and quote posts. Comments and replies should not be included in the timeline.

Decision / implication: Timeline inclusion is defined over eligible `social.craftsky.feed.post` rows from the viewer and followed accounts. It includes root/top-level posts, project posts because they are the same record type with `project` data, and quote posts because they are post rows with `quote_*` fields. It excludes comments, nested replies/replies-to-comments, and repost records as separate feed items in this chunk.

### Q2: Should requirements use the direct joined timeline-query approach?

Answer: Confirm recommended.

Decision / implication: Requirements target a direct read query over indexed posts joined to active follows. Materialised feed tables and a generic feed/search framework are deferred, but the design must avoid hard-coding assumptions that would block later project feeds, custom feeds from lists, search, or other feed sources.

### Q3: When blocks, mutes, reports, and moderation labels land, which filters should apply to home timeline reads?

Answer: All of them.

Decision / implication: This AppView timeline chunk does not implement those moderation/privacy filters, but future moderation work should apply blocks, mutes, reports, and moderation labels to home timeline reads.

### Q4: Should the home timeline include the signed-in viewer's own eligible posts even if they do not follow themselves?

Answer: Include own posts.

Decision / implication: Timeline eligibility includes the authenticated viewer's own top-level/project/quote posts in addition to posts by followed authors. Self-follow is not required, and duplicate rows must not appear if a self-follow row exists.

### Q5: If handle resolution fails for one author in a timeline page, should the whole request fail or return fallback author data?

Answer: Fail the whole request.

Decision / implication: Timeline should use the existing `identity_unavailable` behavior for author handle-resolution failures. Fallback or cached-handle behavior is deferred to future reliability/performance work.

### Q6: For quote posts in the timeline, should the response expand the quoted post or keep the existing strong-reference-only shape?

Answer: Strong reference only.

Decision / implication: Quote posts remain eligible timeline rows, but this chunk does not hydrate nested quoted-post cards. Responses use the existing `PostResponse.quote` `{uri, cid}` field only.

### Q7: Should home timeline ordering be based on AppView index time or record creation time?

Answer: Use AppView index time.

Decision / implication: `indexed_at DESC, uri DESC` remains the timeline ordering rule. Record `created_at` is display data, not the feed ordering key, in this chunk.

### Q8: Should timeline pagination use the current follow graph on every page request, or preserve a page-1 follow snapshot in the cursor?

Answer: Current graph.

Decision / implication: Each page request evaluates eligibility against the currently indexed active follow graph. The cursor does not snapshot the followed-author set; minor drift after follow/unfollow changes between page requests is acceptable.

### Q9: If the viewer follows a non-Craftsky atproto account, should that account contribute anything to this timeline?

Answer: No contribution.

Decision / implication: Timeline returns indexed Craftsky post rows only. Following a non-Craftsky account does not cause `app.bsky.feed.post` or other non-Craftsky content to appear in this endpoint.

### Q10: Should an empty timeline response include onboarding/discovery suggestions, or just an empty `items` list?

Answer: Empty list only.

Decision / implication: Empty feed discovery/onboarding content is out of scope. The endpoint returns `200` with `items: []` and no cursor for an empty result set.

### Q11: After the viewer creates a post, should the timeline show it immediately via synthetic/write-through logic, or only after firehose indexing?

Answer: After indexing.

Decision / implication: Timeline reads are backed only by indexed AppView rows. AppView does not add synthetic just-created rows to timeline responses; optimistic client insertion can be considered in the later Flutter feed slice.

### Q12: If a post qualifies through both self-inclusion and a follow row, should duplicate rows be possible?

Answer: Deduplicate by URI.

Decision / implication: Timeline results are a set of post rows keyed by `uri`. Each post appears at most once, even if multiple eligibility paths exist.

### Q13: Should `GET /v1/feed/timeline` return a total count of all eligible timeline items?

Answer: No total count.

Decision / implication: The response remains `{items, cursor}` only. Timeline total counts are unstable and expensive, and are not required for this infinite-scroll style endpoint.

### Q14: Should `GET /v1/feed/timeline` accept filter query params now, such as `craftType`, `tag`, `hasImages`, or `authorList`?

Answer: No filters now.

Decision / implication: The only defined query parameters are `limit` and `cursor`. Future project feeds, list feeds, and search should receive their own specified contracts rather than hidden filter parameters on the basic timeline endpoint.

### Q15: How should unknown query parameters on `GET /v1/feed/timeline` behave?

Answer: Ignore unknowns.

Decision / implication: Unknown query parameters have no effect in this chunk. Only `limit` and `cursor` are defined.

## 4. Candidate Approaches

### Option A: Direct Joined Timeline Query (Recommended)

Summary: Add `GET /v1/feed/timeline` backed by a bounded query over `craftsky_posts` eligible through either the authenticated viewer's own DID or an active `atproto_follows` relationship, ordered by `(indexed_at DESC, uri DESC)`, returning existing `PostResponse` items with pagination and engagement hydration.

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

Why: The AppView already has the two required read-side substrates: indexed Craftsky posts and indexed active follows. A direct joined query is the simplest path to the requested chronological home feed, aligns with existing API/post-list conventions, and keeps the larger feed architecture reversible. Requirements include explicit extensibility constraints so future craft-specific feeds, list feeds, search, and materialised-feed optimisations are not blocked by this basic endpoint.

## 6. Problem / Opportunity

Craftsky has post creation, profile post lists, comments, likes, reposts, and follow graph state, but no home timeline for a signed-in user. A basic chronological home feed is necessary before the Flutter feed screen can consume AppView data. This AppView-only slice should expose the endpoint and contract while preserving room for richer feed types later.

## 7. Goals

- G-001: Provide an authenticated AppView endpoint for a signed-in user's basic home timeline.
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
- NG-007: Do not implement blocks, mutes, reports, moderation labels, or per-viewer content filtering beyond viewer/self and followed-account eligibility in this chunk.
- NG-008: Do not fetch timeline posts from PDSes on request; the AppView indexed database is the read source.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Signed-in viewer | Authenticated Craftsky user requesting their home timeline. | See their own recent eligible posts and recent eligible posts from accounts they follow. |
| Followed author | Craftsky account followed by the viewer and authoring indexed posts. | Have eligible content appear in followers' timelines after indexing. |
| AppView | Go API and Postgres read model. | Serve a bounded, chronological, paginated feed from indexed public data. |
| Future Flutter client | Later consumer of this endpoint. | Receive a stable, existing post-shaped API contract it can render without PDS reads. |
| Future feed/search work | Later AppView features such as project feeds, list feeds, search, and custom feeds. | Avoid being blocked by timeline-only abstractions introduced now. |

## 10. Current Behavior

The AppView exposes post CRUD/read endpoints and profile-scoped post/comment lists, but it does not expose `GET /v1/feed/timeline`. A signed-in user has no AppView API surface for a home timeline assembled from their own posts and accounts they follow. The database already contains `craftsky_posts` and `atproto_follows` tables that can support the basic query.

## 11. Desired Behavior

The AppView exposes `GET /v1/feed/timeline` as an authenticated `/v1/` JSON endpoint. When called by a signed-in viewer, it returns a paginated page of post-shaped items from the viewer and accounts the viewer actively follows. Items are ordered newest first by AppView index order. Eligible items include top-level posts, project posts, and quote posts; comments, nested replies, and repost records are excluded. Each item appears at most once, uses the existing `PostResponse` shape with author display data and viewer engagement state, and quote posts expose only the existing strong-reference `quote` field. The response uses existing opaque cursor pagination and standard error handling. The endpoint defines only `limit` and `cursor` query parameters, returns no total count, and does not include empty-state suggestions. The implementation remains intentionally simple while using boundaries/naming that can support later feed sources and filters.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | Craftsky must provide a basic chronological home timeline through the AppView. | A home feed is core social app behavior and the next AppView piece needed before Flutter feed work. | Prompt, roadmap, API architecture spec | AC-001, AC-002, AC-005 |
| BR-002 | Business | Must | The timeline design must preserve room for future feed variants such as craft-specific project feeds, custom/list feeds, and search results. | The user explicitly requested avoiding rigid code/API design that would block likely future feed surfaces. | Prompt, Q2 | AC-010 |
| FR-001 | Functional | Must | The AppView shall expose `GET /v1/feed/timeline` as an authenticated endpoint that also requires `X-Craftsky-Device-Id`. | Matches existing `/v1/` endpoint conventions and the API spec's reserved route. | API architecture spec, discovery | AC-001 |
| FR-002 | Functional | Must | The timeline shall select content only from the authenticated viewer and accounts actively followed by the viewer according to the AppView's current indexed `atproto_follows` state. | Defines the basic home feed, includes the viewer's own posts without requiring self-follow, and keeps reads inside AppView. | Prompt, reference doc, codebase, Q4, Q8 | AC-002, AC-003, AC-014 |
| FR-003 | Functional | Must | The timeline shall return eligible indexed `social.craftsky.feed.post` rows from eligible authors: top-level posts, project posts, and quote posts. | Captures the clarified “posts of all types” scope without comments. | Q1, review feedback, Q4 | AC-004 |
| FR-004 | Functional | Must | The timeline shall exclude comments, nested replies/replies-to-comments, and repost records as separate timeline items. | Keeps the chunk bounded and avoids expanding the response shape to feed reasons or conversation activity. | Q1, review feedback, non-goals | AC-004 |
| FR-005 | Functional | Must | Timeline items shall be ordered by `indexed_at DESC` with a deterministic `uri DESC` tie-breaker. | Matches existing feed-indexing chronology rationale and prevents unstable pagination. | Feed indexing spec, discovery | AC-005, AC-006 |
| FR-006 | Functional | Must | The timeline response shall use the existing list shape `{items, cursor}` with `cursor` omitted when there are no more results and no total-count field. | Maintains API consistency and gives the later Flutter client a familiar contract without an unstable/expensive total count. | API architecture spec, existing handlers, Q13 | AC-006, AC-008, AC-016 |
| FR-007 | Functional | Must | The timeline shall support only `limit` and `cursor` query parameters with the existing default/max limit behavior and opaque cursor semantics; unknown query parameters shall have no effect. | Bounded pagination is required for list endpoints, while future filters need separate explicit contracts. | API architecture spec, existing post handlers, Q14, Q15 | AC-006, AC-009, AC-017 |
| FR-008 | Functional | Must | Each timeline item shall use the existing `PostResponse` wire shape, including author identity/display fields, timestamps, reply/quote strong-reference fields as applicable, image views as applicable, tags, and viewer engagement summary fields. Quote posts shall not expand nested quoted-post content in this chunk. | Reuse avoids creating a competing post response contract and lets Flutter render without PDS reads; quote-card hydration is deferred. | Existing post API, discovery, Q6 | AC-007 |
| FR-009 | Functional | Must | Timeline reads shall use AppView-indexed Postgres data for posts, follows, profile display fields, and engagement summaries; the happy path shall not fetch posts from PDSes or add synthetic just-created rows before indexing. | Preserves the AppView read architecture and avoids exposing PDS-token concerns or duplicate/consistency problems to timeline reads. | AGENTS.md, reference doc, Q11 | AC-011, AC-018 |
| FR-010 | Functional | Must | Invalid timeline cursors shall return `400` with error code `invalid_cursor` using the standard error envelope. | Matches existing list endpoint error behavior. | Existing handlers, API architecture spec | AC-009 |
| FR-011 | Functional | Should | Empty timelines, including users who follow no one and have no own eligible posts, or whose followed accounts have no eligible posts, should return `200` with `items: []`, no `cursor`, and no embedded onboarding/discovery suggestions. | Empty feed states should be normal responses, and discovery suggestions are a separate future surface. | Discovery, Q10 | AC-008 |
| FR-012 | Functional | Should | Handle-resolution failures for returned authors should fail the request with the same `identity_unavailable` error behavior as existing post/profile endpoints. | Keeps identity failures consistent across post-shaped responses. | Existing handlers, Q5 | AC-012 |
| FR-013 | Functional | Must | Following a non-Craftsky atproto account shall not cause that account's non-Craftsky posts, such as `app.bsky.feed.post`, to appear in this timeline endpoint. | The timeline returns indexed Craftsky posts only. | Q9 | AC-015 |
| NFR-001 | Non-functional | Must | The endpoint must follow existing `/v1/` API conventions: camelCase JSON, authenticated-device middleware, shared error envelope, and opaque cursor pagination. | Maintains API consistency. | AGENTS.md, API specs | AC-001, AC-006, AC-009, AC-012 |
| NFR-002 | Non-functional | Should | The timeline query should remain bounded and should use existing or narrow supporting indexes for followed-author filtering and reverse chronological ordering. | Reduces risk of unbounded scans while avoiding premature materialised-feed work. | Discovery, reference doc | AC-013 |
| NFR-003 | Non-functional | Should | Timeline code should introduce a clear feed store/query boundary rather than embedding timeline-specific SQL and response assembly directly in route registration. | Makes later feed variants and tests easier without building the whole future feed framework now. | Prompt, Q2, recommended approach | AC-010 |
| RULE-001 | Business rule | Must | An eligible author for this endpoint is either the authenticated viewer's DID or an active `atproto_follows` row where `did` is the authenticated viewer and `subject_did` is the candidate author. Each page evaluates this against current indexed state. | Defines self-inclusion and follow semantics for the feed. | Existing follow schema, prompt, Q4, Q8 | AC-002, AC-003, AC-014 |
| RULE-002 | Business rule | Must | An eligible top-level timeline post has no `reply_root_uri` and no `reply_parent_uri`; any row with either reply field present is excluded from this timeline chunk. | Makes the “no comments or replies” decision testable against the existing schema. | Q1, review feedback, codebase | AC-004 |
| RULE-003 | Business rule | Must | Project posts and quote posts are not special feed item types in this chunk; they are eligible when their `craftsky_posts` row otherwise matches the author-eligibility and inclusion rules. | Keeps the response on existing `PostResponse` and avoids project/feed-reason expansion now. | Q1, lexicon, codebase | AC-004, AC-007 |
| RULE-004 | Business rule | Must | Repost records in `craftsky_reposts` do not cause separate timeline items in this chunk. | Reposts need feed reasons/attribution that are out of scope for this basic feed. | Q1, non-goals | AC-004 |
| RULE-005 | Business rule | Must | Timeline results are deduplicated by post `uri`; a post appears at most once even if multiple eligibility paths exist, such as self-inclusion plus a self-follow row. | Prevents duplicate feed rows. | Q12 | AC-014 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-001, NFR-001 | Given a request to `GET /v1/feed/timeline`, when the request lacks valid authentication or the required device ID header, then AppView returns the existing authenticated-device error behavior; given both are valid, the request reaches the timeline handler. |
| AC-002 | BR-001, FR-002, RULE-001 | Given the viewer follows author A but not author B, and both authors have eligible indexed posts, when the viewer requests the timeline, then items authored by the viewer and author A may appear and items authored by author B do not appear. |
| AC-003 | FR-002, RULE-001 | Given an `atproto_follows` row exists for a different follower or an inactive/deleted follow is absent from active follow storage, when the viewer requests the timeline, then that relationship does not make the candidate author's posts eligible. |
| AC-004 | FR-003, FR-004, RULE-002, RULE-003, RULE-004 | Given a followed author has a root post, project post, quote post, top-level comment, nested reply, and repost record, when the viewer requests the timeline, then the root/project/quote rows are included according to ordering and pagination, while the top-level comment, nested reply, and repost record are excluded. |
| AC-005 | BR-001, FR-005 | Given multiple eligible timeline rows with different `indexed_at` values and at least two rows sharing the same `indexed_at`, when the timeline is returned, then rows are ordered by `indexed_at` descending and ties by `uri` descending. |
| AC-006 | FR-005, FR-006, FR-007, NFR-001 | Given more eligible rows exist than the requested `limit`, when the viewer requests the first page and then requests the returned `cursor`, then the second page continues after the last item of the first page without duplicates or skipped eligible rows under the same dataset. |
| AC-007 | FR-008, RULE-003 | Given eligible rows include author display data, images, tags, reply/quote fields, and engagement state, when the timeline response is built, then each item uses the existing `PostResponse` field names and values consistent with other post endpoints. |
| AC-008 | FR-006, FR-011 | Given the viewer has no own eligible rows and either follows no one or followed accounts have no eligible rows, when the viewer requests the timeline, then AppView returns `200` with `items: []` and omits `cursor`. |
| AC-009 | FR-007, FR-010, NFR-001 | Given a malformed or otherwise invalid `cursor`, when the viewer requests the timeline, then AppView returns `400` with the standard error envelope and `error: "invalid_cursor"`. |
| AC-010 | BR-002, NFR-003 | Given the code/design is reviewed for this chunk, when future feed variants are considered, then timeline selection, pagination, and response assembly are separated enough that adding a project filter, list-author source, or search-backed source does not require changing the public timeline response contract. |
| AC-011 | FR-009 | Given the timeline happy path is exercised in tests with AppView store fakes or database fixtures, when posts are returned, then the endpoint does not require PDS record fetches to assemble timeline items. |
| AC-012 | FR-012, NFR-001 | Given handle resolution fails for an author that would be returned, when the timeline handler builds the response, then it returns a standard error envelope consistent with existing post/profile identity failure behavior. |
| AC-013 | NFR-002 | Given representative followed-author and post fixtures larger than one page, when the timeline store query is tested or reviewed, then it uses bounded `LIMIT` pagination and indexed predicates/order keys rather than loading unbounded rows for client-side filtering. |
| AC-014 | FR-002, RULE-001, RULE-005 | Given the viewer has an eligible own post and also has a self-follow row, when the viewer requests the timeline, then the own post appears at most once; given the viewer has an eligible own post and no self-follow row, then the own post remains eligible. |
| AC-015 | FR-013 | Given the viewer follows a non-Craftsky atproto account that has non-Craftsky posts, when the viewer requests the timeline, then those non-Craftsky posts are not returned by this endpoint. |
| AC-016 | FR-006 | Given the viewer requests any timeline page, when AppView returns a success response, then the body contains `items` and optionally `cursor`, and does not contain a total-count field. |
| AC-017 | FR-007 | Given the viewer requests the timeline with unknown query parameters alongside valid `limit`/`cursor`, when AppView handles the request, then unknown parameters do not change the defined timeline behavior. |
| AC-018 | FR-009 | Given the viewer has just created a post that has not yet been indexed into `craftsky_posts`, when the viewer requests the timeline, then AppView does not synthesize that post into the response before indexing. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | Viewer follows no accounts. | Return `200` with `items: []` and no `cursor`. | FR-011 |
| EC-002 | Viewer follows accounts with only comments or nested replies. | Return no items for those comment/reply rows. | FR-004, RULE-002 |
| EC-003 | Followed author has a top-level comment on an indexed post. | Exclude the comment from the timeline. | FR-004, RULE-002 |
| EC-004 | Followed author has a project post whose project fields are not materialised in response columns. | Include the row as a normal `PostResponse`; project-specific response fields are not added in this chunk. | FR-003, RULE-003 |
| EC-005 | Followed author reposts someone else's post. | Do not create a separate timeline item for the repost record. | FR-004, RULE-004 |
| EC-006 | Cursor is syntactically invalid or has the wrong payload shape. | Return `400 invalid_cursor`. | FR-010 |
| EC-007 | A followed author's handle cannot be resolved during response hydration. | Return identity failure behavior consistent with existing post list endpoints. | FR-012 |
| EC-008 | New posts arrive between page requests. | Opaque seek cursor semantics apply; no offset/page-number contract is promised. | FR-005, FR-007 |
| EC-009 | Viewer has own eligible posts but does not follow themselves. | Include those own posts. | FR-002, RULE-001 |
| EC-010 | Viewer follows themselves and own posts qualify through both self-inclusion and follow relationship. | Return each own post at most once. | RULE-005 |
| EC-011 | Viewer follows non-Craftsky accounts. | Do not return non-Craftsky post records from those accounts. | FR-013 |
| EC-012 | Request includes unknown query parameters such as `craftType` or `tag`. | Ignore them; only `limit` and `cursor` affect this endpoint. | FR-007 |

## 15. Data / Persistence Impact

- New fields: None required by requirements.
- Changed fields: None.
- Migration required: Not expected for functional scope because `craftsky_posts` and `atproto_follows` already exist. A narrow supporting index may be added during implementation if query-plan/test evidence shows the existing indexes are insufficient.
- Backwards compatibility: Additive API endpoint only. Existing post/profile/follow endpoints and response shapes should not change.
- Data source: Existing `craftsky_posts`, `atproto_follows`, `bluesky_profiles`, `craftsky_likes`, `craftsky_reposts`, and related indexed tables.

## 16. UI / API / CLI Impact

- UI: None in this chunk. Flutter feed screen and client pagination are separate future work.
- API: Adds `GET /v1/feed/timeline?limit=<n>&cursor=<opaque>` returning `{items: PostResponse[], cursor?: string}`. No total count, filter params, feed suggestions, or quoted-post expansion are included in this chunk.
- CLI: No required CLI change. Existing request tooling may be able to hit the endpoint once route exists.
- Background jobs: No new background job, fan-out, cache warmer, or backfill required in this chunk.

## 17. Security / Privacy / Permissions

- Authentication: Required through existing AppView auth middleware.
- Authorization: The timeline is scoped to the authenticated viewer's DID plus the viewer's current active indexed follow graph. Callers cannot request another user's timeline by parameter in this chunk.
- Sensitive data: The endpoint serves public indexed post/follow/profile data only; it must not expose PDS access/refresh tokens or private AppView state.
- Abuse cases: This chunk does not implement blocks, mutes, reports, moderation labels, rate limits, or content takedown filtering. When those moderation/privacy systems land, all of blocks, mutes, reports, and moderation labels should apply to home timeline reads. Those remain future moderation/rate-limit work and should be called out before public launch.

## 18. Observability

- Events: No product analytics required in this AppView-only requirements chunk.
- Logs: Timeline handler/store should follow existing logging style with request/run ID, viewer DID, limit, cursor presence, row count, and error code where useful. Logs must not include bearer tokens.
- Metrics: No new metrics required, but row count/latency would be useful future observability for feed performance.
- Alerts: None required for this slice.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Direct joined query becomes slow for users with large follow graphs or high-volume followed accounts. | Timeline endpoint latency increases as data grows. | Keep pages bounded, use indexed predicates/order keys, and defer materialised feed tables until volume proves need. |
| RISK-002 | Post-vs-conversation semantics are misunderstood during implementation. | Timeline includes comments or replies that are out of scope for this chunk. | Use `RULE-002` and acceptance tests with root post, top-level comment, and nested reply fixtures. |
| RISK-003 | Handle resolution for many distinct authors causes latency or endpoint failure. | Timeline pages may fail with `identity_unavailable` or be slower than expected. | Batch unique author handling where possible; keep page sizes bounded; use existing resolver behavior and tests. |
| RISK-004 | Firehose/indexing lag makes follows or posts appear stale. | Recently followed accounts or newly-created posts may not appear immediately. | Document AppView indexed state as the source of truth; future UX can handle eventual consistency. |
| RISK-005 | Blocks/mutes/reports/moderation-label filtering are absent. | Timeline may show content a later product version should suppress. | Keep out of this scope but record that all of these filters should apply to home timeline reads when those systems land. |
| RISK-006 | Timeline-only implementation choices make future project/list/search feeds harder. | Later feed work requires rewrites or incompatible API changes. | Require a feed store/query boundary and preserve existing post response/cursor conventions. |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | `atproto_follows` represents the active follow graph needed for the basic home timeline. | Timeline would need a different graph source or active/deleted-state model. |
| ASM-002 | Any row with reply metadata is conversation activity rather than an eligible home-timeline post for this chunk. | Inclusion/exclusion SQL and tests would need to change. |
| ASM-003 | Project posts and quote posts can be rendered with the current `PostResponse` without new project-specific or quote-expanded fields. | The API response shape would need additional fields and possibly project-field materialisation. |
| ASM-004 | Repost activity should not appear as a separate timeline item until feed reasons/attribution are designed. | A larger feed-item envelope would be needed sooner. |
| ASM-005 | AppView-only requirements are sufficient for this chunk; Flutter consumption will be handled by a later workflow stage/change. | Scope would need to expand to include app models, API client, providers, and UI tests. |
| ASM-006 | Minor pagination drift caused by follow/unfollow changes between page requests is acceptable for this basic timeline. | Cursor payloads would need to snapshot followed-author sets, increasing complexity. |

## 21. Open Questions

- [ ] Non-blocking: Future repost timeline shape is undecided. Should repost support use a feed-item envelope with reasons, or continue returning only post-shaped items on timeline?
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
  - `FR-001` through `FR-013`
  - `NFR-001`
  - `RULE-001` through `RULE-005`
- Suggested test levels:
  - Unit tests for cursor handling, inclusion/exclusion classification, and response assembly.
  - Store/query integration tests against Postgres fixtures for own posts, followed/unfollowed authors, self-follow deduplication, non-Craftsky follows, root posts vs comments/replies, ordering, and pagination.
  - Handler/route tests for auth/device requirements, response envelope behavior, invalid cursors, unknown query parameters, empty timelines, no total-count field, and happy-path response shape.
  - Regression tests ensuring existing post/profile routes still return the same `PostResponse` shape.
- Blocking open questions: None.
- Review recommendation: Because risk is medium and feed semantics are foundational for later Flutter work, review of `01-requirements.md` is recommended before test design.
