# Coding Plan: AppView Project Posts

## 1. Inputs

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` (`Approved with notes`)
- Repository references inspected:
  - `appview/internal/index/craftsky_post.go`
  - `appview/migrations/000010_craftsky_posts.up.sql`
  - `appview/internal/api/post_store.go`, `post_request.go`, `post_response.go`, `post.go`
  - `appview/internal/api/profile_store.go`, `profile_response.go`
  - `appview/internal/api/timeline_store.go`, `timeline.go`
  - `appview/internal/api/notification_store.go`, `notifications.go`
  - `appview/internal/routes/routes.go`
  - generated project lexicon types under `appview/internal/lexicon/craftsky/`
  - relevant tests under `appview/internal/index`, `appview/internal/api`, `appview/internal/routes`
  - API/indexing references under `docs/superpowers/specs/` and `adr/001-post-lexicon-project-extensibility.md`

## 2. Implementation Strategy

Implement the AppView slice in the same order as the approved test specification, adjusted by review feedback: migration/schema first, then indexing/materialization, then create API, shared response hydration, and finally profile project count/list routing. This matches the existing Go/AppView architecture: raw `pgx` SQL, inline test DDL fixtures, `PostStore` as the read-side boundary, handler factories in `appview/internal/api`, route registration in `appview/internal/routes/routes.go`, and Tap indexer registration already present in `appview/internal/app/deps.go`.

Use Option B from `01-requirements.md`: keep `social.craftsky.feed.post` as the only post collection, add minimal project indicators to `craftsky_posts`, and create a one-to-one `craftsky_project_posts` table keyed by post URI. Hydrate `PostResponse.project` from typed project materialization with raw JSON preservation for unknown details, not from normalized search columns and not by PDS read-through. After the 2026-06-09 grill-me clarification, only standalone records can be project posts: no reply pointer and no quote embed. Profile Posts and Projects are split tabs, so profile post counts/lists exclude project posts while the main timeline/feed still includes them.

Risk sequencing from `03-document-review.md` is mandatory:

1. Make `IT-001` the first failing test.
2. Choose the migration-chain strategy first. Add a small migration test/helper that applies real migration SQL into an isolated `testdb.WithSchema` schema; if that becomes impractical because existing migrations assume `public` or external migration state, fall back to an isolated SQL test that executes `000016_project_posts.up.sql` after a minimal pre-state and records the fallback in test comments.
3. Front-load `UT-003`/`IT-005` unknown open-union details before broad API work.
4. Introduce typed project DTOs for API/indexer boundaries instead of passing project payloads as unstructured `json.RawMessage` everywhere. Preserve raw JSON in storage for forward compatibility and exact authored response reconstruction when needed.
5. Centralize post response hydration so project-eligible surfaces such as single post, profile projects, timeline/feed, notifications, and create responses do not drift.

## 3. Affected Areas

| Area | Existing Pattern | Planned Change | Requirement IDs | Test IDs |
|---|---|---|---|---|
| Migrations / persistence | Numbered SQL migrations in `appview/migrations`; tests often use inline DDL with `testdb.WithSchema` | Add `000016_project_posts.up/down.sql` with base flags, one-to-one project table, no historical backfill, and indexes | FR-001, FR-002, FR-003, NFR-003, RULE-001 | AT-001, AT-013, IT-001, MAN-002 |
| Post indexer | `CraftskyPost` unmarshals generated `FeedPost`, upserts `craftsky_posts`, idempotent on URI/CID | Extract project raw/common/details before typed loss, merge project tags, maintain project child row on upsert/delete/removal | FR-004, FR-005, FR-006, NFR-002 | AT-002, AT-003, AT-004, AT-011, UT-001, UT-002, UT-003, UT-011, IT-002, IT-003, IT-004, IT-005 |
| Post utilities | `postutil.ExtractTags` handles facet tags only | Add merge/normalize helper for facet tags + `project.common.tags`, preserving non-null output | FR-006 | AT-011, UT-001, REG-001 |
| Create post API | `POST /v1/posts` rejects `project`; `lexiconRecordBody` writes allowed fields to PDS | Allow optional typed standalone `project`, validate minimal shape plus supported create-time craft type and no reply/quote, include in PDS createRecord body and synthetic response | FR-007, FR-008, BR-002, NFR-004 | AT-005, AT-006, UT-004, UT-005, UT-006, UT-012, IT-006, IT-007, REG-002, REG-003 |
| Post read store / row model | `PostRow` contains base post fields and author display fields; `postSelectColumns` reused by many queries | Extend `PostRow` with typed project DTO plus raw storage JSON and base flags; join project table in all post-shaped stores | FR-009, FR-010, NFR-001 | AT-007, AT-008, IT-008, IT-009, IT-012, IT-015, REG-005 |
| Post response builder | `BuildPostResponse` centralizes post wire shape | Add optional lexicon-shaped typed `project` field sourced from project materialization with raw authored data preserved for unknown details | FR-009, NFR-004 | AT-007, AT-008, UT-007, UT-008, UT-013, MAN-003 |
| Profile summary | `ProfileStore.Read` hardcodes `project_count` to zero | Count visible standalone project posts for `projectCount`; count visible non-project root posts for `postCount`/recent post counts | FR-011, RULE-003 | AT-009, UT-009, IT-010, REG-007 |
| Profile project list | Existing profile posts/comments handlers share `listAuthorPostsHandler` | Add authenticated `GET /v1/profiles/{handleOrDid}/projects`, store method filters visible standalone project posts; profile post lists filter projects out | FR-012, RULE-003, NFR-001, NFR-004 | AT-010, UT-010, IT-011, IT-013 |
| Routes | `routes.go` registers authenticated + device wrapped `/v1/*` routes | Register profile projects route with same auth/device stack as posts/comments | FR-012, NFR-004 | AT-010, IT-013 |
| Regression flows | Likes/reposts/replies/reports/delete operate by post URI | Avoid project-specific branches except response hydration; preserve ordinary post semantics | RULE-002 | AT-012, IT-014, REG-006 |

## 4. Files And Modules

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `appview/migrations/000016_project_posts.up.sql` | Create | Add `is_project`, `project_craft_type`, `craftsky_project_posts`, no historical backfill, and indexes | FR-001, FR-002, FR-003, NFR-003 | AT-001, AT-013, IT-001 |
| `appview/migrations/000016_project_posts.down.sql` | Create | Drop project indexes/table/base flags in reverse order | FR-001 | IT-001 |
| `appview/internal/index/craftsky_post.go` | Change | Maintain project materialization and merged tags on create/update/delete | FR-004, FR-005, FR-006, RULE-001 | AT-002, AT-003, AT-004, AT-011, IT-002, IT-003, IT-004, IT-005 |
| `appview/internal/postutil/tags.go` | Change | Add reusable normalized merge helper for facet tags + project tags | FR-006 | UT-001, AT-011, REG-001 |
| `appview/internal/api/post_project.go` | Create | Define typed `Project`, `ProjectCommon`, `Pattern`, and craft detail DTOs/interfaces used by request/response/indexing helpers | FR-002, FR-003, FR-007, FR-009 | UT-002, UT-003, UT-005, UT-007 |
| `appview/internal/api/post_request.go` | Change | Permit optional typed project payload and validate minimal `project.common.craftType`; keep `createdAt` rejected | FR-007, FR-008 | UT-004, UT-005, UT-006, REG-003 |
| `appview/internal/api/post.go` | Change | Include project in PDS createRecord body, synthetic `PostRow`, and add profile projects handler if local style keeps handlers here | FR-007, FR-012 | AT-005, AT-010, IT-006, IT-007, IT-013 |
| `appview/internal/api/post_store.go` | Change | Extend `PostRow`, select/join project data, add `ListProjectsByAuthor` | FR-009, FR-010, FR-012 | IT-008, IT-009, IT-011 |
| `appview/internal/api/post_response.go` | Change | Add `Project *Project` or equivalent lexicon-shaped typed optional field to `PostResponse` | FR-009, NFR-004 | UT-007, UT-008, UT-013 |
| `appview/internal/api/profile_store.go` | Change | Replace `0 AS project_count` with top-level visible project count | FR-011, RULE-003 | AT-009, UT-009, IT-010 |
| `appview/internal/api/notification_store.go` | Change | Hydrate notification subject posts with project metadata | FR-010 | AT-007, IT-012 |
| `appview/internal/api/timeline_store.go` | Change | Ensure timeline `postSelectColumns` includes joined project data | FR-010 | AT-007, IT-012 |
| `appview/internal/routes/routes.go` | Change | Register `GET /v1/profiles/{handleOrDid}/projects` | FR-012, NFR-004 | IT-013 |
| `appview/internal/index/craftsky_post_test.go` | Change | Update fixture DDL and add project indexing/convergence tests | FR-001, FR-004, FR-005, FR-006 | AT-002, AT-003, AT-004, AT-011, IT-002, IT-003, IT-004, IT-005 |
| `appview/internal/postutil/tags_test.go` | Change | Add facet/project tag merge tests | FR-006 | UT-001 |
| `appview/internal/api/post_request_test.go` | Change | Replace project rejection test with allow/validate/reject malformed cases | FR-007, FR-008 | UT-004, UT-005, UT-006 |
| `appview/internal/api/post_response_test.go` | Change | Assert project field included/omitted and camelCase | FR-009, NFR-004 | UT-007, UT-008, UT-013 |
| `appview/internal/api/post_store_test.go` | Change | Update fixture DDL; add read/list/project-list hydration tests | FR-009, FR-010, FR-012 | IT-008, IT-009, IT-011, IT-014, IT-015 |
| `appview/internal/api/profile_store_test.go` | Change | Update fixture DDL and add projectCount cases | FR-011, RULE-003 | UT-009, IT-010 |
| `appview/internal/api/timeline_store_test.go`, `notification_store_test.go` | Change | Update fixture DDL and add hydration assertions | FR-010 | IT-012 |
| `appview/internal/routes/routes_test.go` | Change | Assert new route uses auth/device wrapping and valid request path | FR-012, NFR-004 | IT-013 |

## 5. Services, Interfaces, And Data Flow

### Persistence shape

Use the next migration number after current `000015`: `000016_project_posts`.

Planned base-table additions:

```text
ALTER TABLE craftsky_posts
  ADD COLUMN is_project BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN project_craft_type TEXT;
```

Planned one-to-one materialization table:

```text
craftsky_project_posts
  uri TEXT PRIMARY KEY REFERENCES craftsky_posts(uri) ON DELETE CASCADE
  raw_project JSONB NOT NULL
  common_craft_type TEXT NOT NULL
  common_status TEXT NULL
  common_title TEXT NULL
  common_duration TEXT NULL
  pattern_url TEXT NULL
  pattern_name TEXT NULL
  pattern_difficulty TEXT NULL
  pattern_designer TEXT NULL
  pattern_publisher TEXT NULL
  materials TEXT[] NOT NULL DEFAULT '{}'
  colors TEXT[] NOT NULL DEFAULT '{}'
  design_tags TEXT[] NOT NULL DEFAULT '{}'
  project_tags TEXT[] NOT NULL DEFAULT '{}'
  details_type TEXT NULL
  raw_details JSONB NULL

  knitting_project_type TEXT NULL
  knitting_project_subtype TEXT NULL
  knitting_yarn_weight TEXT NULL
  knitting_needle_size_mm TEXT NULL
  knitting_gauge JSONB NULL
  knitting_finished_size TEXT NULL

  crochet_project_type TEXT NULL
  crochet_project_subtype TEXT NULL
  crochet_yarn_weight TEXT NULL
  crochet_hook_size_mm TEXT NULL
  crochet_gauge JSONB NULL
  crochet_finished_size TEXT NULL

  quilting_project_type TEXT NULL
  quilting_project_subtype TEXT NULL
  quilting_piecing_technique TEXT NULL
  quilting_quilting_method TEXT NULL
  quilting_size TEXT NULL

  sewing_project_type TEXT NULL
  sewing_project_subtype TEXT NULL
  sewing_size_made TEXT NULL
  sewing_fit_notes TEXT NULL

  indexed_at TIMESTAMPTZ NOT NULL DEFAULT now()
```

Historical backfill is explicitly not required for this stage. The migration should add schema only; project materialization is populated by new create/update events handled after the migration. Existing rows whose `record` already contains `project` may remain unmaterialized until they are reindexed or updated by a future explicit backfill task.

```text
-- No UPDATE/INSERT backfill pass in 000016.
-- New/updated Tap events populate craftsky_project_posts through CraftskyPost.
```

Do not split knitting, crochet, quilting, and sewing details into separate tables in this slice. A single one-to-one `craftsky_project_posts` table keeps the read path simple for `PostResponse` hydration, avoids one join per possible craft, and matches the requirement for a dedicated project materialization table keyed by post URI. The table may include nullable craft-specific columns and raw details JSON. Separate per-craft tables can be revisited later if detail fields grow enough to justify the extra join complexity.

Index guardrails:

- Add a partial profile-project ordering index on `craftsky_posts(did, indexed_at DESC, uri DESC)` where `is_project`, root-post predicates, and `quote_uri IS NULL` are true.
- Add a btree index for `craftsky_posts(project_craft_type)` or a compound index where useful for root project filters.
- Add GIN indexes for project arrays that are committed query dimensions: `materials`, `colors`, `design_tags`, `project_tags`.
- Add btree indexes for `common_craft_type`, `common_status`, and `pattern_difficulty` in `craftsky_project_posts`.
- For craft detail columns, add only indexes that support plausible v1/future filters; document deferred fields in the migration test or review note for MAN-002.

### Indexer project extraction

The current generated `FeedPost` type is still useful for text/facets/images/reply/quote, but unknown `project.details` variants are implementation-sensitive. Preserve raw project data by extracting `project` from `ev.Record` before or alongside typed unmarshalling.

Partial signatures/sketch:

```text
type Project struct {
  Common ProjectCommon `json:"common"`
  Details ProjectDetails `json:"details,omitempty"`
}

type ProjectCommon struct {
  CraftType string `json:"craftType"`
  Status *string `json:"status,omitempty"`
  Title *string `json:"title,omitempty"`
  Duration *string `json:"duration,omitempty"`
  Pattern *ProjectPattern `json:"pattern,omitempty"`
  Materials []string `json:"materials,omitempty"`
  Colors []string `json:"colors,omitempty"`
  DesignTags []string `json:"designTags,omitempty"`
  Tags []string `json:"tags,omitempty"`
}

type ProjectDetails interface {
  DetailsType() string
}

type KnittingDetails struct { ... }
type CrochetDetails struct { ... }
type QuiltingDetails struct { ... }
type SewingDetails struct { ... }
type UnknownProjectDetails struct {
  Type string
  Raw json.RawMessage
}

type indexedProject struct {
  Project Project
  RawProject json.RawMessage
  RawDetails json.RawMessage
}

func extractProjectForIndex(raw json.RawMessage, typed *craftskylex.FeedPost) (*indexedProject, error)
func projectTagsForSearch(project *indexedProject) []string
func upsertProjectMaterialization(ctx context.Context, tx pgx.Tx, ev tap.Event, project *indexedProject) error
func deleteProjectMaterialization(ctx context.Context, tx pgx.Tx, uri syntax.ATURI) error
```

Important extraction rules:

- A project post is detected by `project.common` with non-empty `craftType` on a standalone record only. If the record has `reply` or a quote embed, preserve the raw post record but do not set `is_project`, write project materialization, merge project tags, or return `project` (RULE-001).
- Unknown details are not an error if `project.common.craftType` exists. Store `raw_project`, `raw_details`, and `details_type`; known detail typed columns may remain NULL.
- The typed `Project` DTO should live in AppView API/indexing code, not in generated lexicon code. Generated lexicon types remain consumed as validation/decode inputs, while the DTO gives handlers/tests a stable, typed JSON wire shape.
- Use a transaction for base `craftsky_posts` upsert plus child row upsert/delete so updates that remove `project` cannot leave stale child rows.
- Preserve replay behavior: if the base upsert is a same-CID no-op, project maintenance must also be a no-op or idempotent and must not advance timestamps unexpectedly. If implementation cannot cheaply detect row count from the base upsert, child `ON CONFLICT` updates must be guarded by `WHERE ... IS DISTINCT FROM` checks on raw/materialized values.

### Tag flow

Add a helper that can be reused by indexer and create synthetic response:

```text
func MergeTags(tagSets ...[]string) []string

facetTags := postutil.ExtractTags(rec.Facets)
tags := postutil.MergeTags(facetTags, project.Common.Tags)
```

Normalize by trim + lowercase + dedupe while preserving first-seen order. Do not mutate or normalize `raw_project` for responses; normalized tags are for `craftsky_posts.tags` search only.

### API create flow

Planned request shape:

```text
type PostCreateRequest struct {
  Text string
  Facets json.RawMessage
  Reply *ReplyRef
  Embed *EmbedRequest
  Images []PostImage
  Project *Project `json:"project,omitempty"`
}
```

Validation sketch:

```text
DecodePostCreate:
  reject createdAt
  allow project
  still disallow unknown fields

ValidatePostCreateWithLimits:
  validate existing text/images/reply/quote
  if project present:
    unmarshal to map/object
    require project.common object
    require non-empty string project.common.craftType
    require craftType to be one of the current lexicon knownValues for first-party create
    reject when reply is present
    reject when embed.quote is present
    fail wrong JSON types as validation_failed / malformed_body per existing FieldError conventions
```

PDS body sketch:

```text
body := lexiconRecordBody(req)
if req.Project != nil:
  body["project"] = req.Project
```

Synthetic response row should carry the typed project payload and best-effort merged tags. `createdAt` remains server-stamped by the AppView; client-supplied `createdAt` remains rejected.

### Read/hydration flow

Preferred read model:

```text
type PostRow struct {
  ...existing fields...
  IsProject bool
  ProjectCraftType *string
  Project *Project // nil for general posts
  RawProject json.RawMessage // preserved authored project JSON for unknown details/debugging
}

type PostResponse struct {
  ...existing fields...
  Project *Project `json:"project,omitempty"`
}
```

`BuildPostResponse` should set `Project` only when `row.Project` is non-nil. This keeps all post-shaped endpoints consistent because most handlers already call `BuildPostResponse`. For unknown future details, `UnknownProjectDetails` should marshal back to the original raw details object so response hydration remains lexicon-shaped and forward-compatible.

SQL/hydration guardrail:

- Replace the fixed `postSelectColumns` constant with either a function that takes aliases or constants that include a `LEFT JOIN craftsky_project_posts pp ON pp.uri = p.uri` convention.
- Every query returning `PostRow` must select the same project columns.
- `notification_store.go` currently hand-selects subject post columns with aliases `sp`/`sbp`; update its subject scan explicitly or refactor to reuse the same select builder so notifications do not miss `project`.

## 6. State, Providers, Controllers, Or DI

No Flutter/Riverpod state, providers, widgets, or app routes are in scope for this AppView-only slice (NG-002).

Existing server-side DI remains unchanged:

```text
routes.AddRoutes(ctx, mux, deps)
  postStore := api.NewPostStore(deps.DB)
  api.CreatePostHandler(postStore, deps.NewPDSClient, deps.HandleResolver, ...)
  api.ListPostsByAuthorHandler(postStore, deps.HandleResolver, ...)
  api.ListProjectsByAuthorHandler(postStore, deps.HandleResolver, ...) // new
```

No new long-lived service or dependency in `app.Deps` is expected. `CraftskyPost` remains registered for `social.craftsky.feed.post`; no new NSID/indexer registration is needed because project posts are the same collection.

## 7. UI, Widgets, Routes, Or User-Facing Surfaces

No Flutter UI implementation is in scope.

User-facing HTTP changes:

- `POST /v1/posts`
  - Accept optional `project` object.
  - Include the same object in the PDS `createRecord` body.
  - Return HTTP 201 with `project` in the `PostResponse` for project posts.
- Existing post-shaped responses
  - Include `project` for project posts on project-eligible surfaces such as single read, timeline/feed, profile projects, notifications, and create response.
  - Omit `project` for general posts.
- `GET /v1/profiles/{handleOrDid}`
  - Return real `projectCount` for Craftsky profiles.
  - Return `postCount`/recent post counts for non-project root posts only.
- New `GET /v1/profiles/{handleOrDid}/projects`
  - Authenticated + device-id required.
  - Resolve handle/DID like existing profile posts/comments routes.
  - Use `limit` and opaque `cursor` with existing parsing/cursor conventions.
  - Return bare JSON page shape matching existing list endpoints:

```text
{
  "items": [PostResponse, ...],
  "cursor": "..." // omitted when absent
}
```

No CLI changes are expected beyond the new migration files.

## 8. Error, Loading, Empty, And Edge States

| State / Case | Planned Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| General post has no `project` | `is_project=false`, `project_craft_type=NULL`, no child row, response omits `project` | FR-001, FR-009, RULE-001 | AT-002, AT-008, IT-002, IT-009, UT-008 |
| Project post has `project.common.craftType` | Base flags set, child row upserted, response includes typed lexicon-shaped project | FR-001, FR-002, FR-009 | AT-002, AT-003, AT-007, UT-007, IT-003, IT-008 |
| Project object missing `common` or `craftType` on create | Reject before PDS write with existing `FieldError`/standard envelope | FR-008, RULE-001 | AT-006, UT-006, IT-007 |
| Project create uses unsupported craft type | Reject before PDS write; the indexer remains permissive for external future craft types | FR-008, RULE-001 | AT-006, UT-006, IT-007 |
| Project create includes reply or quote | Reject before PDS write because first-party project posts must be standalone | FR-008, RULE-001, RULE-003 | AT-006, UT-006, IT-007 |
| Indexed invalid project event lacks common/craftType or has project plus reply/quote | Treat as ordinary post indexing; do not create child materialization or expose `project` in AppView responses | FR-004, RULE-001 | AT-002, UT-011 |
| Unknown future details variant | Store common/raw project/raw details/details type; known details columns NULL; do not poison-pill solely because details type is unknown | FR-005 | AT-004, UT-003, IT-005, REG-004 |
| Update removes `project` | Base flags reset and child row deleted in same transaction | FR-004 | AT-004, IT-005, EC-002 |
| Delete project post | Delete base row; child row removed by FK cascade or explicit delete | FR-004 | AT-004, IT-005 |
| Duplicate same URI/CID replay | No duplicate rows; no unintended `indexed_at` churn | NFR-002 | AT-004, IT-004 |
| Firehose record has project plus reply/comment/quote | Preserve raw record but treat as non-project everywhere in AppView | RULE-001, RULE-002, RULE-003 | AT-002, AT-009, AT-010, EC-005 |
| Moderation hides project post or author | Same visibility predicates as ordinary posts; project count/list exclude hidden/takedown rows | RULE-002, RULE-003 | AT-009, AT-010, IT-010, IT-011 |
| Empty profile projects page | Return `items: []`, omit cursor, no PDS calls | FR-012, NFR-001 | AT-010, IT-011, IT-015 |
| Invalid profile projects cursor | Return 400 `invalid_cursor` using standard error envelope | FR-012, NFR-004 | UT-010, IT-013 |

## 9. Test Implementation Plan

| Order | Test ID | Target | Setup / Fixture | Initial Expected Failure |
|---|---|---|---|---|
| 1 | IT-001 | Migration test near `appview/cmd/cli/migrate_test.go` or new DB migration test | Apply real migrations into `testdb.WithSchema` where possible; fallback to isolated SQL execution if migration-chain harness is impractical | Missing `000016` schema/table/indexes |
| 2 | UT-001 | `appview/internal/postutil/tags_test.go` | Facet tags plus project tags with duplicates/casing/spacing | No merge helper; project tags ignored |
| 3 | IT-002 / UT-011 | `appview/internal/index/craftsky_post_test.go` | Updated DDL with project columns/table; general and minimal project records | Indexer only writes base post; no flags/child rows |
| 4 | UT-002 / IT-003 | `craftsky_post_internal_test.go` or `craftsky_post_test.go` | Full known knitting/crochet/quilting/sewing payloads | Known detail columns not extracted |
| 5 | UT-003 / IT-005 | Indexer helper and integration tests | Project common with unknown future details `$type`; update removes project; delete | Typed parsing loses/blocks unknown details or leaves stale rows |
| 6 | IT-004 | `craftsky_post_test.go` | Same URI/CID delivered twice | Duplicate child row or timestamp churn |
| 7 | UT-004 / UT-005 / UT-006 | `appview/internal/api/post_request_test.go` | Valid project, missing common/craftType, wrong types, createdAt | Decoder rejects `project` or accepts invalid shape |
| 8 | UT-012 / IT-006 / IT-007 | `appview/internal/api/post_test.go` | Fake PDS client and authenticated request context | PDS body lacks project; invalid requests still call PDS |
| 9 | UT-007 / UT-008 / UT-013 | `appview/internal/api/post_response_test.go` | Project-bearing `PostRow`, general `PostRow`, JSON marshal checks | `PostResponse` has no project field / wrong omission behavior |
| 10 | IT-008 / IT-009 | `appview/internal/api/post_store_test.go` | DB rows with/without `craftsky_project_posts` | Store rows do not hydrate project or general posts leak empty field |
| 11 | IT-012 | `timeline_store_test.go`, `notification_store_test.go`, comment/read tests | Project posts returned through timeline/comments/notifications | One surface misses project due to custom select scan |
| 12 | UT-009 / IT-010 | `profile_store_test.go` | Standalone project, project reply, project quote, general root, hidden posts | `projectCount` hardcoded 0, includes non-standalone projects, or `postCount` includes projects |
| 13 | UT-010 / IT-011 / IT-013 | `post_store_test.go`, `post_test.go`, `routes_test.go` | Mixed profile posts and route auth/device variants | No profile projects store method/handler/route |
| 14 | IT-014 / REG-* | Existing interaction/report/moderation/delete suites | Project post rows as targets | Special-case regressions or missing hydration in existing flows |
| 15 | MAN-001 through MAN-004 | Manual implementation review | Search for PDS read-through, inspect schema indexes/query plans, sample JSON | Any undocumented index deferral, non-camelCase shape, or hydration drift |

Focused commands:

```text
just test
just fmt
```

For faster TDD loops, run targeted Go tests from `appview/` with `TEST_DATABASE_URL` set, e.g. `go test ./internal/index -run TestCraftskyPost` and `go test ./internal/api -run TestPost` after the compose Postgres is available.

## 10. Sequencing And Guardrails

- First TDD step: write `IT-001` for migration/schema/index coverage using `000016_project_posts.up.sql`.
- Dependencies between work items:
  1. Migration before fixture updates and store/indexer implementation.
  2. Tag merge helper before indexer and synthetic create response.
  3. Unknown-details extraction before broad create/read API work.
  4. `PostRow`/`BuildPostResponse` hydration before endpoint-specific hydration tests.
  5. `ListProjectsByAuthor` store method before handler and route registration.
- Guardrails:
  - Do not change `lexicon/` or generated lexicon types in this slice unless a blocking mismatch is discovered; pause for design review if that happens.
  - Do not add Flutter/Dart changes.
  - Do not add direct PDS reads on read/list paths; project data must come from AppView Postgres.
  - Do not introduce a separate project collection or route that treats projects as non-post records.
  - Keep project response JSON lexicon-shaped and sourced from typed project data with raw unknown details preserved; normalized search tags remain storage/search-only.
  - Preserve ordinary post behavior for likes, reposts, replies, reports, moderation, timeline inclusion, and delete.
  - Keep `/v1/*` request/response JSON camelCase and error envelopes standard.
  - Keep source/test changes TDD-focused; update inline DDL fixtures when schema changes land.
- Out of scope:
  - Flutter composer/cards/profile Projects tab.
  - Lexicon or ADR changes.
  - Project search/global discovery/recommendations/ranking.
  - Cross-post project identity or mutable lifecycle records.
  - Post edit endpoints.

## 11. Risks And Open Questions

| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Blocking if encountered | Generated open-union parsing may not preserve unknown details through typed structs | Could violate FR-005 / poison-pill Tap | Extract raw project/details from `ev.Record` independently. If common/raw cannot be preserved, pause for requirements/design review per DR-001. |
| CPQ-002 | Non-blocking | Migration-chain test harness may be limited | IT-001 could be hard to implement against all existing migrations | Try real migration-chain first; fallback to isolated SQL test executing `000016` against minimal pre-state and document fallback. Backfill is intentionally out of scope. |
| CPQ-003 | Non-blocking | Indexing every craft-detail field could over-index v1 | Extra migration/index maintenance | Add required profile/count/craft/array indexes; document any intentionally deferred detail-field indexes for MAN-002/DR-003. |
| CPQ-004 | Non-blocking | Notification subject post hydration uses custom select/scanner | Easy to miss `project` on notifications | Add explicit IT-012 coverage and prefer shared select/scanning helper where feasible. |
| CPQ-005 | Non-blocking | AppView validation could diverge from PDS lexicon validation | Valid records could be rejected or invalid records forwarded | Keep validation minimal: JSON object, `common`, `common.craftType`; rely on PDS for deeper lexicon validation. |
| CPQ-006 | Non-blocking | `project.common.tags` authored display vs normalized search tags can be confused | API could leak normalized search-only tags as authored project tags | Hydrate `PostResponse.project` from typed project data backed by `raw_project`; use merged normalized tags only for `craftsky_posts.tags`. |
| CPQ-007 | Non-blocking | Per-craft detail tables might become attractive as detail schemas grow | A future schema may need another migration if nullable craft columns become too wide | Keep one project materialization table in this slice for simpler hydration and profile project queries; revisit per-craft tables only with evidence of schema/query pressure. |

## 12. Handoff To TDD Builder

- Coding plan: `docs/changes/2026-06-07-appview-project-posts/04-coding-plan.md`
- TDD execution plan: create `05-implementation-plan.md` during the next workflow stage if that stage requires one.
- Start with test: `IT-001` migration/schema/index coverage.
- Focused command: `just test` after `just dev-d`/compose Postgres is running; use package-level `go test` commands for red/green loops.
- Notes:
  - Carry forward all `03-document-review.md` findings, especially DR-001 through DR-004.
  - Every implementation step should cite the linked requirement/test IDs from this plan or the acceptance test spec.
  - Do not accept implementation completion until MAN-001 through MAN-004 or equivalent review notes are addressed.
