# Requirements: AppView Project Posts

## 1. Initial Request

The project lexicon has been updated and the AppView should now support project posts. The work should first settle the AppView persistence schema, then update indexing/storage for project posts, and finally update post-related endpoints so project posts can be created, read, counted, and listed through the AppView.

## 2. Current Codebase Findings

- Relevant files:
  - Project lexicons: `lexicon/social/craftsky/feed/post.json`, `lexicon/social/craftsky/project/defs.json`, `lexicon/social/craftsky/project/{knitting,crochet,quilting,sewing}.json`.
  - Generated Go lexicon types: `appview/internal/lexicon/craftsky/feedpost.go`, `appview/internal/lexicon/craftsky/projectdefs.go`, `appview/internal/lexicon/craftsky/project{knitting,crochet,quilting,sewing}.go`.
  - Current post indexer: `appview/internal/index/craftsky_post.go`.
  - Current post storage/API: `appview/migrations/000010_craftsky_posts.up.sql`, `appview/internal/api/post_store.go`, `appview/internal/api/post_request.go`, `appview/internal/api/post_response.go`, `appview/internal/api/post.go`, `appview/internal/api/timeline_store.go`, `appview/internal/routes/routes.go`.
  - Profile count surface: `appview/internal/api/profile_store.go`, `appview/internal/api/profile_response.go`.
  - Existing design references: `adr/001-post-lexicon-project-extensibility.md`, `docs/superpowers/specs/2026-04-23-post-lexicon-fields-design.md`, `docs/superpowers/specs/2026-05-04-feed-post-indexing-design.md`, `docs/superpowers/specs/2026-05-04-feed-post-crud-endpoints-design.md`, `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md`.
- Existing patterns:
  - Craftsky keeps one post record type, `social.craftsky.feed.post`; project posts are posts with an optional `project` sub-object, not separate top-level collections.
  - The AppView writes to user PDSes through the server-side PDS client and reads indexed craft data from Postgres.
  - `CraftskyPost` is registered for `social.craftsky.feed.post`, unmarshals generated lexicon types, gates on `craftsky_profiles`, upserts idempotently on URI/CID, and preserves the full record as `record JSONB`.
  - Post responses are built from `PostRow` plus handle resolution and are reused by create, read, profile lists, comments, timeline, notifications, interactions, and reports.
  - `/v1/*` API bodies use camelCase and standard AppView error envelopes.
- Current behavior:
  - The AppView indexes general post fields only: text, facets, images, reply/quote refs, facet-derived tags, full record JSON, `created_at`, and `indexed_at`.
  - `POST /v1/posts` rejects `project` as not writable.
  - `PostResponse` has no project metadata field.
  - `ProfileStore.Read` currently hardcodes `project_count` to zero.
  - Profile and timeline post lists cannot filter to project posts.
- Constraints discovered:
  - ADR 001 commits to materializing at least `is_project` and `project.common.craftType` as queryable AppView data.
  - The post lexicon fields spec commits to materializing status, pattern difficulty, materials, project tags, and per-craft details as queryable dimensions.
  - The updated lexicon now includes project common fields and craft details for knitting, crochet, quilting, and sewing.
  - Lexicon changes are not in scope for this requirements slice; the user confirmed the AppView scope using the recommended Option B schema direction.
  - Public post/project data belongs on PDS records; AppView Postgres stores indexed/read-model copies.
- Test/build commands discovered:
  - Go/AppView tests: `just test` from the repo root after the compose database is running.
  - Go formatting: `just fmt`.
  - Relevant tests are expected under `appview/internal/index`, `appview/internal/api`, and `appview/internal/routes`.

## 3. Clarifying Questions And Decisions

### Q1: For this requirements document, should the work be scoped to AppView project-post support using the recommended schema direction?

Answer: Option B AppView.

Decision / implication: Scope this requirements document to AppView database schema, indexer/storage, and post/profile endpoint behavior for project posts. Use a dedicated one-to-one project materialization table plus minimal base post flags. Do not include Flutter UI work or lexicon changes.

## 4. Candidate Approaches

### Option A: Wide `craftsky_posts` Table

Summary: Add nullable project columns directly to `craftsky_posts` for common project fields and craft-specific detail fields.

Pros:
- Simplest read queries; no join needed to return project metadata.
- Fits the ADR shorthand that project filters are the same post query plus `is_project = true`.
- Keeps one storage row per post.

Cons:
- Makes `craftsky_posts` wide and mostly NULL for general posts.
- Future craft details keep expanding the base table.
- Blurs the boundary between universal post fields and project-specific materialization.

Risks:
- Schema churn as more crafts and project-specific filters are added.

### Option B: Dedicated Project Materialization Table Plus Minimal Base Flags

Summary: Add minimal project flags to `craftsky_posts` and store project-specific indexed fields in a one-to-one `craftsky_project_posts` table keyed by post URI.

Pros:
- Preserves the single post record model while keeping general posts lean.
- Isolates project-specific fields, indexes, and future craft additions.
- Supports project-only profile lists/counts and future discovery/search without overloading the base post table.
- Allows post-shaped API responses to join in project metadata only when needed.

Cons:
- Requires joins for project-bearing post responses and project-only lists.
- Indexer upsert/delete behavior is more complex because it must maintain two tables consistently.

Risks:
- Query authors may forget to join or hydrate project metadata on one of the existing post-shaped endpoints.

### Option C: Read Project Data From `record JSONB` Only

Summary: Do not materialize project fields beyond the existing full-record JSONB; derive project metadata at read time or leave it unsurfaced.

Pros:
- Smallest migration and implementation footprint.
- Preserves all data already stored in `record JSONB`.

Cons:
- Does not satisfy prior AppView indexer commitments.
- Makes project filters/search/counts harder or slower.
- Pushes project parsing into every read path instead of the indexer.

Risks:
- Future project feeds/search would require another migration and backfill anyway.

## 5. Recommended Direction

Recommended approach: Option B — dedicated `craftsky_project_posts` materialization table plus minimal base flags on `craftsky_posts`.

Why: This keeps Craftsky's one-post-record architecture intact, satisfies prior commitments to materialize project query dimensions, avoids a permanently wide general-post table, and gives test design clear seams for schema, indexer, and API behavior.

## 6. Problem / Opportunity

Craftsky has lexicon support for project posts, but the AppView currently treats every `social.craftsky.feed.post` as a basic social post. Users and clients cannot create project posts through AppView, cannot read project metadata from AppView responses, and cannot list/count a profile's projects. Supporting project posts in AppView unlocks the product's core crafting use case while preserving PDS ownership and AppView read performance.

## 7. Goals

- G-001: Persist project-post query dimensions in Postgres so project posts can be filtered, counted, and returned without PDS read-through.
- G-002: Index existing and future `social.craftsky.feed.post` events with `project` payloads into an explicit project read model.
- G-003: Allow authenticated clients to create project posts through `POST /v1/posts` using the updated lexicon shape.
- G-004: Return project metadata on post-shaped read responses for project posts while preserving general-post response compatibility.
- G-005: Provide profile-level project counts and a profile project-post list suitable for the existing Projects tab and future clients.

## 8. Non-Goals

- NG-001: Do not change files under `lexicon/` in this stage.
- NG-002: Do not implement Flutter composer, feed card, profile Projects tab UI, or Dart model changes in this AppView slice.
- NG-003: Do not add project search, recommendation, ranking, algorithmic feeds, or global discovery endpoints.
- NG-004: Do not add cross-post project identity or mutable project lifecycle records; each project post remains a standalone snapshot.
- NG-005: Do not add unauthenticated access or direct PDS read-through from Flutter.
- NG-006: Do not introduce a separate top-level project post collection; records remain `social.craftsky.feed.post`.
- NG-007: Do not add post update/edit endpoints unless they already exist independently; this slice covers create and read/list/count surfaces.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Craftsky member | Authenticated user with a Craftsky profile | Create project posts and see project metadata in their profile/feed responses. |
| AppView indexer | Firehose/Tap event consumer | Materialize project metadata idempotently from `social.craftsky.feed.post` records. |
| Flutter client | First-party app consuming `/v1/*` | Send project metadata on create and render/list project posts from AppView responses. |
| Test designer / implementer | Workflow stages after this document | Stable requirements and IDs for schema, indexing, and endpoint tests. |

## 10. Current Behavior

The AppView stores all `social.craftsky.feed.post` records in `craftsky_posts` as basic posts. The full record is preserved as JSONB, but project fields are not materialized into queryable columns. The post create request rejects `project`, post responses omit project metadata, and profile project counts are always zero. Existing timeline/profile/comment/notification endpoints can return rows whose raw record has a project payload, but callers cannot tell from the AppView response.

## 11. Desired Behavior

When a `social.craftsky.feed.post` contains a valid `project` object, the AppView indexes it as both a post and a project post. The base post row records that it is a project and carries cheap top-level project filtering data. A dedicated project table stores the project payload and materialized query dimensions. Post create accepts the lexicon-shaped project object, writes it to the user's PDS, and returns a synthetic project-bearing post response. Post-shaped read/list endpoints include project metadata for project posts. Profile reads return real project counts, and a profile project list endpoint returns only top-level project posts.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | Craftsky shall support project posts as first-class AppView-readable posts without creating a separate top-level project collection. | Project posts are central to the product and ADR 001 chose one post type with optional project metadata. | Prompt, ADR 001 | AC-001, AC-005, AC-009 |
| BR-002 | Business | Must | Project-post support shall preserve PDS ownership for public project data and AppView Postgres as the indexed read model. | Maintains architectural rules: writes through PDS, reads through AppView. | AGENTS.md, discovery | AC-005, AC-006 |
| FR-001 | Functional | Must | The system shall add AppView persistence for project posts using minimal base flags on `craftsky_posts` and a one-to-one project materialization table keyed by post URI. | Confirmed Option B and keeps general posts lean while supporting project query dimensions. | Q1, discovery | AC-001, AC-002 |
| FR-002 | Functional | Must | The project materialization table shall store the raw project JSON payload and materialized common fields: craft type, status, title, duration, pattern fields, materials, colors, design tags, and project tags. | These fields are needed for API responses and future project filtering/search. | Lexicon findings, post lexicon fields spec | AC-001, AC-003, AC-010 |
| FR-003 | Functional | Must | The project materialization table shall store the details `$type`, raw details JSON, and materialized known craft detail fields from sewing, quilting, crochet, and knitting. | The updated lexicon includes first-class craft details that should be queryable/readable. | Lexicon findings | AC-003, AC-010 |
| FR-004 | Functional | Must | The `CraftskyPost` indexer shall upsert project materialization when a post record contains `project`, clear project materialization when an updated record no longer contains `project`, and delete project materialization when the post is deleted. | Indexing must converge under create/update/delete and preserve idempotency. | Existing indexer pattern | AC-002, AC-004 |
| FR-005 | Functional | Must | The indexer shall continue to store and update the full post `record JSONB` and shall not drop valid project posts only because `details` contains an unknown open-union variant. | Supports future craft additions and debugging/backfill without PDS fetches. | ADR 001, generated open-union behavior | AC-004, EC-003 |
| FR-006 | Functional | Must | The indexed `craftsky_posts.tags` search column shall merge existing facet-derived tags with `project.common.tags`, lowercased, trimmed, deduplicated, and non-null. | Existing specs require belt-and-suspenders tag merging for project posts. | Post lexicon fields spec, feed post indexing spec | AC-003, AC-011 |
| FR-007 | Functional | Must | `POST /v1/posts` shall accept an optional `project` object matching the lexicon-shaped `social.craftsky.project.defs#project` JSON structure and include it in the PDS createRecord body. | Clients need to create project posts through the AppView write path. | Prompt, current API behavior | AC-005, AC-006, EC-004 |
| FR-008 | Functional | Must | `POST /v1/posts` shall reject malformed project request bodies with standard AppView validation or bad-request errors, without allowing clients to set `createdAt`. | Maintains current request validation posture and server-stamped timestamps. | Existing post_request pattern | AC-006, EC-004 |
| FR-009 | Functional | Must | Every post-shaped response shall include a lexicon-shaped `project` object for project posts and omit the `project` field for general posts. | Makes project metadata available while preserving general-post response compatibility. | Desired behavior, API compatibility | AC-007, AC-008 |
| FR-010 | Functional | Must | Existing post-shaped read/list endpoints shall hydrate project metadata consistently for single post reads, profile post lists, timeline, comments/replies, notifications, and create responses where those endpoints return `PostResponse`. | Prevents missing project data on one AppView surface. | Existing response reuse | AC-007, AC-008 |
| FR-011 | Functional | Must | Profile reads shall calculate `projectCount` from indexed top-level project posts instead of returning a hardcoded zero. | Existing profile response exposes projectCount and the Projects tab needs real counts. | Codebase finding | AC-009 |
| FR-012 | Functional | Must | The AppView shall provide an authenticated, paginated profile project list endpoint that returns only top-level project posts for a resolved profile DID, ordered by the existing profile post ordering key. | Enables clients to list a profile's projects without mixing comments or non-project posts. | Prompt, profile Projects tab finding | AC-010 |
| RULE-001 | Business rule | Must | A project post is any `social.craftsky.feed.post` whose record contains `project.common`; absence of `project` means a general post. | Mirrors lexicon semantics. | Lexicon findings | AC-001, AC-002 |
| RULE-002 | Business rule | Must | Project posts shall remain eligible for existing post behaviors including likes, reposts, replies, quotes, moderation filtering, reporting, timeline inclusion, and deletion because they are still posts. | Preserves one-record-type architecture and avoids special interaction semantics. | ADR 001, existing routes | AC-007, AC-012 |
| RULE-003 | Business rule | Must | Project-only profile lists and project counts shall include top-level posts only and exclude replies/comments. | Matches existing profile post count behavior and avoids counting project comments as projects. | Existing profile count predicates | AC-009, AC-010 |
| NFR-001 | Non-functional | Must | Project-post reads shall not fetch project records from PDSes in the happy path. | Flutter/AppView architecture requires reads from indexed AppView data. | AGENTS.md | AC-007, AC-010 |
| NFR-002 | Non-functional | Must | Project indexing shall remain idempotent under Tap at-least-once delivery and CID replays. | Existing indexer invariant must continue to hold. | Existing indexer pattern | AC-004 |
| NFR-003 | Non-functional | Should | Project schema and read queries should use indexes that support project profile lists, project counts, craft-type filtering, and array membership filters without requiring full table scans at expected v1 scale. | Protects future project discovery paths and current profile surfaces. | Candidate approach analysis | AC-013 |
| NFR-004 | Non-functional | Must | All new or changed `/v1/*` request and response JSON fields shall use camelCase and existing AppView error envelopes. | Maintains API consistency. | API architecture spec | AC-006, AC-007, AC-010 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-001, FR-002, RULE-001 | Given the schema migration has run, when the database is inspected, then `craftsky_posts` contains minimal project indicators and `craftsky_project_posts` exists as a one-to-one project materialization table keyed by post URI with common project fields and FK/cascade behavior tied to `craftsky_posts`. |
| AC-002 | FR-001, FR-004, RULE-001 | Given an indexed create event for a general post, when the indexer handles it, then `craftsky_posts` records it as non-project and no `craftsky_project_posts` row exists; given a create event with `project.common`, then both the base post row and project materialization row exist. |
| AC-003 | FR-002, FR-003, FR-006 | Given a project post with common fields, project tags, facet tags, and one known craft details variant, when indexed, then the project table stores common fields, raw project/details JSON, materialized craft fields, and merged non-null tags in `craftsky_posts.tags`. |
| AC-004 | FR-004, FR-005, NFR-002 | Given repeated delivery of the same URI/CID, CID-changing updates, updates that remove `project`, deletes, and unknown `details` variants, when handled by the indexer, then storage converges without duplicate rows, stale project rows, or rejected valid common project records. |
| AC-005 | BR-001, BR-002, FR-007 | Given an authenticated request to `POST /v1/posts` with valid text and valid `project`, when the AppView writes the record, then the PDS createRecord body contains the lexicon-shaped project object and the response is HTTP 201 with project metadata. |
| AC-006 | BR-002, FR-007, FR-008, NFR-004 | Given malformed project JSON, missing required `project.common.craftType`, wrong field types, or disallowed `createdAt`, when creating a post, then the AppView returns the appropriate standard error response and does not write a PDS record. |
| AC-007 | FR-009, FR-010, RULE-002, NFR-001, NFR-004 | Given indexed project posts are returned by single-post, timeline, profile posts, comments/replies, or notification post hydration paths, when a client reads those endpoints, then each project post response includes `project` and no endpoint performs PDS read-through to hydrate it. |
| AC-008 | FR-009, FR-010 | Given indexed general posts are returned by post-shaped endpoints, when serialized, then the `project` field is omitted and existing general post response fields remain compatible. |
| AC-009 | FR-011, RULE-003 | Given a profile has a mix of top-level general posts, top-level project posts, and replies with project payloads, when the profile is read, then `projectCount` equals only top-level project posts visible under moderation rules. |
| AC-010 | FR-002, FR-003, FR-012, RULE-003, NFR-001, NFR-004 | Given a profile has indexed project and non-project posts, when `GET /v1/profiles/{handleOrDid}/projects` is requested with pagination, then the response contains only top-level project `PostResponse` items with project metadata and an opaque cursor when more rows remain. |
| AC-011 | FR-006 | Given a project post has duplicate tags across facets and `project.common.tags` with mixed casing/spacing, when indexed, then `craftsky_posts.tags` contains a lowercased, trimmed, deduplicated array suitable for existing hashtag queries. |
| AC-012 | RULE-002 | Given a project post is indexed, when existing like, repost, reply/comment, report, moderation visibility, and delete flows operate on it, then they treat it as an ordinary post with no project-specific special case required beyond response hydration. |
| AC-013 | NFR-003 | Given implementation completes, when schema and query plans are reviewed in tests or manual inspection, then project profile list/count and key project filters have explicit supporting indexes or a documented reason an existing index is sufficient. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | Record has `project` object but missing `common` or `common.craftType`. | Indexing rejects the invalid record according to generated lexicon/PDS-validity behavior; create endpoint rejects before PDS write if detected in request validation. | FR-007, FR-008, RULE-001 |
| EC-002 | Update changes a project post into a general post. | Base row becomes non-project and the project materialization row is removed. | FR-004 |
| EC-003 | Project details uses a future open-union `$type` unknown to current generated Go code. | AppView indexes `project.common`, stores raw project/details JSON where available, records the details type if extractable, and does not poison-pill solely because the details variant is unknown. | FR-005 |
| EC-004 | Create request includes valid general post fields plus invalid project field types. | Request fails with standard error response and no PDS write; valid general post creation remains unaffected when `project` is absent. | FR-008 |
| EC-005 | Project post is a reply/comment. | It can be read as a post/comment and carries project metadata, but it is excluded from profile project count and profile project list. | RULE-002, RULE-003 |
| EC-006 | Moderation hides a project post or author. | Project post is excluded anywhere the corresponding general post would be excluded. | RULE-002 |

## 15. Data / Persistence Impact

- New fields:
  - Minimal base post flags on `craftsky_posts`, expected to include at least `is_project` and `project_craft_type` or equivalent queryable fields.
  - New `craftsky_project_posts` one-to-one table keyed by `uri`, referencing `craftsky_posts(uri)` with cascade delete.
  - Project common materialization fields for craft type, status, title, duration, pattern URL/name/difficulty/designer/publisher, materials, colors, design tags, project tags, raw project JSON, details type, and raw details JSON.
  - Craft detail materialization fields for current sewing, quilting, crochet, and knitting detail schemas where they are intended to be filterable or returned without reparsing raw JSON.
- Changed fields:
  - `craftsky_posts.tags` population changes from facet-only to facet tags plus `project.common.tags`.
  - `profile.projectCount` changes from hardcoded zero to data-driven project-post count.
- Migration required:
  - Yes. Add a new numbered migration and corresponding down migration.
  - Migration/backfill should derive project materialization from existing `craftsky_posts.record JSONB` so previously indexed project records, if any, become queryable without PDS fetches.
- Backwards compatibility:
  - General posts remain `social.craftsky.feed.post` records.
  - General post responses should omit `project` to minimize response shape changes for existing clients.
  - Existing PDS records remain the source of truth; AppView materialization can be rebuilt from `record JSONB`.

## 16. UI / API / CLI Impact

- UI:
  - No Flutter UI implementation in this slice.
  - This work supplies the AppView surface future Flutter project composer/cards/profile Projects tab can consume.
- API:
  - `POST /v1/posts` accepts optional `project`.
  - `PostResponse` includes `project` for project posts and omits it for general posts.
  - Existing post-shaped read/list endpoints hydrate project metadata consistently.
  - Add authenticated `GET /v1/profiles/{handleOrDid}/projects` with the same handle-or-DID convention, pagination style, auth/device requirements, and error envelope conventions as existing profile post lists.
  - Profile responses return data-driven `projectCount`.
- CLI:
  - No CLI changes expected beyond migrations already managed by existing migrate tooling.
- Background jobs:
  - Existing Tap consumer/indexer path changes. No new long-running job is required.

## 17. Security / Privacy / Permissions

- Authentication:
  - New and changed `/v1/*` routes remain authenticated and require `X-Craftsky-Device-Id` where existing route stacks do.
- Authorization:
  - Creating a project post uses the authenticated caller's DID and existing PDS client factory.
  - Deleting/interacting/reporting with a project post follows existing post authorization rules.
- Sensitive data:
  - Project metadata is public PDS record data. Do not store private-by-intent project drafts, mutes, wishlists, or hidden metadata in the PDS or project materialization table as part of this slice.
- Abuse cases:
  - Existing moderation hide/takedown/warn predicates apply to project posts wherever post-shaped rows are returned.

## 18. Observability

- Events:
  - No new analytics/event pipeline required.
- Logs:
  - Indexer errors should identify project materialization failures with URI/DID context without logging secrets.
  - Create endpoint logs should follow current post create logging posture and avoid dumping sensitive session data.
- Metrics:
  - No explicit metrics required in this slice.
- Alerts:
  - No explicit alerts required in this slice.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Project lexicon field set may continue changing as craft taxonomy evolves. | Materialized schema could need follow-up migrations. | Store raw project/details JSON and keep fields additive; do not change lexicon in this slice. |
| RISK-002 | Project hydration may be missed on one post-shaped endpoint. | Inconsistent API responses and client bugs. | Require tests across every endpoint that returns `PostResponse` or centralize response building/hydration. |
| RISK-003 | Unknown future detail union variants may be mishandled by generated types. | Tap poison-pill or dropped valid future project posts. | Index common fields independently where possible and preserve raw JSON; add tests for unknown detail `$type`. |
| RISK-004 | Schema/index choices may under-support future project discovery filters. | Future search/feed work may need more migrations. | Include explicit indexes for profile project lists/counts and common project filters; document deferred filters. |
| RISK-005 | Create endpoint validation may duplicate or diverge from PDS lexicon validation. | Valid records could be rejected or invalid records forwarded. | Keep AppView validation focused on request structure/required common fields; rely on PDS lexicon validation for deep schema validation where appropriate. |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | The updated project lexicon shape is accepted as the source of truth for this AppView slice. | Requirements would need revision and possibly lexicon-skill/ADR work before AppView implementation. |
| ASM-002 | A one-to-one project materialization table plus minimal base flags satisfies the prior ADR/spec commitment to queryable project dimensions. | If strict interpretation requires all columns on `craftsky_posts`, schema direction would need revisiting. |
| ASM-003 | Profile project lists should include top-level project posts only, matching existing profile post count semantics. | If replies with project metadata should count as projects, profile counts/lists and tests would change. |
| ASM-004 | Existing `record JSONB` contains enough data to backfill project materialization for any already-indexed project records. | Backfill would require PDS refetch or accepting only future project materialization. |
| ASM-005 | Returning a lexicon-shaped `project` object is preferable to inventing a separate flattened API-only project response shape in this slice. | Flutter/client model requirements may need adjustment if a flattened shape is desired. |

## 21. Open Questions

- [ ] Non-blocking: Should future project discovery endpoints filter by every materialized craft-detail field, or only a smaller public subset?
- [ ] Non-blocking: Should `project.common.tags` be displayed directly in API responses exactly as authored, while `craftsky_posts.tags` remains normalized for search?
- [ ] Non-blocking: Should profile `postCount` continue to include project posts, or should product copy later distinguish all posts vs projects more explicitly? This slice assumes project posts are still posts and remain included.

## 22. Review Status

Status: Draft
Risk level: High
Review recommended: Required
Reviewer:
Date:
Notes: High risk because this changes AppView persistence, indexing, PDS write request shape, and public `/v1/*` API response contracts for load-bearing project-post data. Explicit approval is required before test design or implementation continues.

## 23. Handoff To Test Design

- Requirements file: `docs/changes/2026-06-07-appview-project-posts/01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover requirement IDs:
  - Business: BR-001, BR-002
  - Functional: FR-001 through FR-012
  - Rules: RULE-001 through RULE-003
  - Non-functional: NFR-001, NFR-002, NFR-004
- Suggested test levels:
  - Migration/schema tests or migration review for base flags, child table, indexes, cascade, and backfill behavior.
  - Indexer integration tests using test DB fixtures for create/update/delete, known craft details, unknown details, tag merging, and idempotency.
  - API handler/store tests for create validation/write body, synthetic response, read/list hydration, profile project count, profile projects endpoint, general-post compatibility, and moderation filtering.
  - Route tests for the new profile projects route and auth/device wrapping.
  - Regression tests for existing general post create/read/list behavior and existing like/repost/reply/report/delete flows on project posts.
- Blocking open questions:
  - None recorded for test design, but risk review/approval is required before continuing.
