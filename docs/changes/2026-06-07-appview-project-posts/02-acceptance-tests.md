# Acceptance Test Specification: AppView Project Posts

## 1. Test Strategy

This specification verifies AppView support for project posts from persistence through indexing, write API behavior, read/list hydration, and profile project counts/lists. The requirements are high risk because they change AppView schema, Tap indexing, PDS write payloads, and public `/v1/*` response contracts; document review is required before implementation continues.

Test design follows the implementation sequence requested in `01-requirements.md`:

1. Persistence and migration coverage for minimal base project flags, one-to-one `craftsky_project_posts`, cascade, and supporting indexes. Historical backfill is explicitly out of scope for this stage.
2. Indexer/store coverage for create, update, delete, tag merging, idempotency, known craft details, and future unknown detail variants.
3. API coverage for `POST /v1/posts`, `PostResponse` hydration across existing post-shaped surfaces, profile `projectCount`, and the new profile project list route.

Most checks should be automated in Go tests under `appview/internal/index`, `appview/internal/api`, `appview/internal/routes`, and migration/CLI or DB test suites. Manual checks are limited to query-plan/index inspection and final review of API shape where automated assertions may not prove performance intent.

Discovered commands:

- `just test` from repo root after `just dev-d` is running; runs `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test -race ./...`.
- `just fmt` from repo root for Go formatting and vetting.

Existing relevant test conventions:

- Indexer integration-style tests use `appview/internal/testdb.WithSchema` in `appview/internal/index/craftsky_post_test.go`.
- API store tests use inline DDL fixtures in `appview/internal/api/post_store_test.go`, `timeline_store_test.go`, and `profile_store_test.go`.
- Handler tests use fake stores/PDS clients in `appview/internal/api/post_test.go` and `profile_test.go`.
- Route auth/device wrapping tests live in `appview/internal/routes/routes_test.go`.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-005, AC-009 | AT-001, AT-005, AT-009, IT-001, IT-010 | Acceptance / Integration | Yes |
| BR-002 | AC-005, AC-006 | AT-005, AT-006, UT-004, IT-006 | Acceptance / Unit / Integration | Yes |
| FR-001 | AC-001, AC-002 | AT-001, AT-002, IT-001, IT-002 | Acceptance / Integration | Yes |
| FR-002 | AC-001, AC-003, AC-010 | AT-001, AT-003, AT-010, IT-001, IT-003, IT-011 | Acceptance / Integration | Yes |
| FR-003 | AC-003, AC-010 | AT-003, AT-010, IT-003, IT-011, UT-002 | Acceptance / Unit / Integration | Yes |
| FR-004 | AC-002, AC-004 | AT-002, AT-004, IT-002, IT-004, IT-005 | Acceptance / Integration | Yes |
| FR-005 | AC-004, EC-003 | AT-004, IT-005, UT-003, REG-004 | Acceptance / Unit / Integration / Regression | Yes |
| FR-006 | AC-003, AC-011 | AT-003, AT-011, UT-001, IT-003, REG-001 | Acceptance / Unit / Integration / Regression | Yes |
| FR-007 | AC-005, AC-006, EC-004 | AT-005, AT-006, UT-004, UT-005, IT-006 | Acceptance / Unit / Integration | Yes |
| FR-008 | AC-006, EC-004 | AT-006, UT-004, UT-006, IT-007, REG-003 | Acceptance / Unit / Integration / Regression | Yes |
| FR-009 | AC-007, AC-008 | AT-007, AT-008, UT-007, UT-008, IT-008, IT-009 | Acceptance / Unit / Integration | Yes |
| FR-010 | AC-007, AC-008 | AT-007, AT-008, IT-008, IT-009, IT-012, REG-005 | Acceptance / Integration / Regression | Yes |
| FR-011 | AC-009 | AT-009, IT-010, UT-009 | Acceptance / Unit / Integration | Yes |
| FR-012 | AC-010 | AT-010, IT-011, IT-013, UT-010 | Acceptance / Unit / Integration | Yes |
| RULE-001 | AC-001, AC-002 | AT-001, AT-002, UT-011, IT-002 | Acceptance / Unit / Integration | Yes |
| RULE-002 | AC-007, AC-012 | AT-007, AT-012, IT-014, REG-006 | Acceptance / Integration / Regression | Yes |
| RULE-003 | AC-009, AC-010 | AT-009, AT-010, IT-010, IT-011, UT-009 | Acceptance / Unit / Integration | Yes |
| NFR-001 | AC-007, AC-010 | AT-007, AT-010, IT-008, IT-011, MAN-001 | Acceptance / Integration / Manual | Mixed |
| NFR-002 | AC-004 | AT-004, IT-004, IT-005 | Acceptance / Integration | Yes |
| NFR-003 | AC-013 | AT-013, IT-001, MAN-002, GAP-001 | Acceptance / Integration / Manual | Mixed |
| NFR-004 | AC-006, AC-007, AC-010 | AT-006, AT-007, AT-010, UT-004, UT-007, IT-006, IT-011, IT-013 | Acceptance / Unit / Integration | Yes |

## 3. Acceptance Scenarios

### AT-001: Project Persistence Schema Exists

Requirement IDs: BR-001, FR-001, FR-002, RULE-001  
Acceptance Criteria: AC-001  
Priority: Must  
Level: Acceptance  
Automation Target: migration/schema test near `appview/cmd/cli/migrate_test.go` or `appview/internal/db/*_test.go`

```gherkin
Feature: AppView project post persistence
  Scenario: Migration creates project post read model
    Given the AppView migration chain has run
    When the database schema is inspected
    Then craftsky_posts has minimal project indicators such as is_project and project_craft_type
    And craftsky_project_posts exists keyed one-to-one by post URI
    And craftsky_project_posts references craftsky_posts(uri) with cascade delete behavior
    And craftsky_project_posts stores raw project JSON plus common project materialization fields
```

### AT-002: General And Project Creates Index Differently

Requirement IDs: FR-001, FR-004, RULE-001  
Acceptance Criteria: AC-002  
Priority: Must  
Level: Acceptance  
Automation Target: `appview/internal/index/craftsky_post_test.go`

```gherkin
Feature: Project post indexing
  Scenario: Indexer distinguishes general posts from project posts
    Given a Craftsky member profile exists
    When the indexer handles a social.craftsky.feed.post create without project.common
    Then craftsky_posts records the post as non-project
    And no craftsky_project_posts row exists for that URI
    When the indexer handles a social.craftsky.feed.post create with project.common
    Then craftsky_posts records the post as a project
    And a matching craftsky_project_posts row exists for that URI
```

### AT-003: Project Fields And Tags Are Materialized

Requirement IDs: FR-002, FR-003, FR-006  
Acceptance Criteria: AC-003  
Priority: Must  
Level: Acceptance  
Automation Target: `appview/internal/index/craftsky_post_test.go`

```gherkin
Feature: Project metadata materialization
  Scenario: Known craft details and searchable tags are stored
    Given a project post has common fields, project tags, inline hashtag facets, and known knitting details
    When the indexer handles the post
    Then craftsky_project_posts stores craft type, status, title, duration, pattern fields, materials, colors, design tags, project tags, raw project JSON, details type, raw details JSON, and knitting detail fields
    And craftsky_posts.tags contains the normalized union of facet tags and project.common.tags
```

### AT-004: Indexing Converges Under Replays Updates Deletes And Unknown Details

Requirement IDs: FR-004, FR-005, NFR-002  
Acceptance Criteria: AC-004  
Priority: Must  
Level: Acceptance  
Automation Target: `appview/internal/index/craftsky_post_test.go`

```gherkin
Feature: Idempotent project indexing
  Scenario: Project materialization converges across Tap delivery patterns
    Given a project post URI is delivered more than once with the same CID
    And later delivered with a changed CID
    And later delivered without a project object
    And later deleted
    When the indexer handles each event
    Then storage has no duplicate base or project rows
    And stale project materialization is removed after the general-post update
    And delete removes both base and project materialization
    And records with project.common plus a future unknown details variant are not rejected solely because of the unknown variant
```

### AT-005: Create Project Post Writes Lexicon-Shaped Project To PDS

Requirement IDs: BR-001, BR-002, FR-007  
Acceptance Criteria: AC-005  
Priority: Must  
Level: Acceptance  
Automation Target: `appview/internal/api/post_test.go`

```gherkin
Feature: Create project posts through AppView
  Scenario: Authenticated create forwards project metadata to the PDS
    Given an authenticated Craftsky member request body has valid text and a valid project object
    When the client posts to POST /v1/posts
    Then the AppView calls PDS createRecord for social.craftsky.feed.post
    And the createRecord body contains the lexicon-shaped project object unchanged except for server-stamped createdAt handling
    And the HTTP response status is 201
    And the response body includes project metadata
```

### AT-006: Malformed Project Create Requests Fail Before PDS Write

Requirement IDs: BR-002, FR-007, FR-008, NFR-004  
Acceptance Criteria: AC-006  
Priority: Must  
Level: Acceptance  
Automation Target: `appview/internal/api/post_request_test.go`, `appview/internal/api/post_test.go`

```gherkin
Feature: Validate project post create requests
  Scenario Outline: Invalid project create request is rejected
    Given an authenticated request to POST /v1/posts contains <invalid_project_body>
    When the AppView decodes and validates the request
    Then the response uses the standard AppView error envelope
    And the response JSON fields use camelCase
    And no PDS createRecord call is made

    Examples:
      | invalid_project_body |
      | malformed project JSON |
      | project.common missing |
      | project.common.craftType missing |
      | wrong field types inside project |
      | client-supplied createdAt |
```

### AT-007: Project Metadata Is Hydrated On Post-Shaped Read Surfaces

Requirement IDs: FR-009, FR-010, RULE-002, NFR-001, NFR-004  
Acceptance Criteria: AC-007  
Priority: Must  
Level: Acceptance  
Automation Target: `appview/internal/api/post_store_test.go`, `timeline_store_test.go`, `notifications_test.go`, `post_response_test.go`

```gherkin
Feature: Read project posts from AppView
  Scenario Outline: Post-shaped endpoints include project metadata without PDS read-through
    Given an indexed project post exists in Postgres
    When a client reads it through <endpoint_surface>
    Then each returned project post includes a lexicon-shaped project field
    And response fields use camelCase
    And the AppView does not fetch the project record from the PDS to hydrate it

    Examples:
      | endpoint_surface |
      | single post read |
      | timeline |
      | profile posts |
      | comments or replies |
      | notifications |
      | create response |
```

### AT-008: General Post Responses Stay Compatible

Requirement IDs: FR-009, FR-010  
Acceptance Criteria: AC-008  
Priority: Must  
Level: Acceptance  
Automation Target: `appview/internal/api/post_response_test.go`, `appview/internal/api/post_store_test.go`

```gherkin
Feature: General post response compatibility
  Scenario: General posts omit project metadata
    Given an indexed general post has no project object
    When a post-shaped endpoint serializes the post
    Then the JSON response omits the project field
    And existing general post fields, engagement fields, images, reply, quote, tags, and moderation fields remain compatible
```

### AT-009: Profile Project Count Uses Top-Level Visible Project Posts

Requirement IDs: BR-001, FR-011, RULE-003  
Acceptance Criteria: AC-009  
Priority: Must  
Level: Acceptance  
Automation Target: `appview/internal/api/profile_store_test.go`, `appview/internal/api/profile_response_test.go`

```gherkin
Feature: Profile project summary
  Scenario: Profile read reports visible top-level project count
    Given a profile has top-level general posts, top-level project posts, project replies, and moderated hidden project posts
    When the profile is read
    Then projectCount equals only visible top-level project posts
    And project replies are excluded from projectCount
    And general posts are excluded from projectCount
```

### AT-010: Profile Projects Endpoint Lists Only Top-Level Project Posts

Requirement IDs: FR-002, FR-003, FR-012, RULE-003, NFR-001, NFR-004  
Acceptance Criteria: AC-010  
Priority: Must  
Level: Acceptance  
Automation Target: `appview/internal/api/post_store_test.go`, `appview/internal/api/post_test.go`, `appview/internal/routes/routes_test.go`

```gherkin
Feature: Profile project post lists
  Scenario: Client pages through a profile's project posts
    Given a profile has visible project posts, general posts, project replies, and hidden project posts
    When an authenticated client requests GET /v1/profiles/{handleOrDid}/projects with a page limit
    Then the response contains only visible top-level project PostResponse items
    And each item includes project metadata
    And an opaque cursor is returned when more project rows remain
    And the AppView does not read through to the PDS to hydrate project metadata
```

### AT-011: Facet And Project Tags Merge Deterministically

Requirement IDs: FR-006  
Acceptance Criteria: AC-011  
Priority: Must  
Level: Acceptance  
Automation Target: `appview/internal/postutil/tags_test.go`, `appview/internal/index/craftsky_post_test.go`

```gherkin
Feature: Searchable project tags
  Scenario: Duplicate mixed-case tags normalize into searchable post tags
    Given a project post contains hashtag facets and project.common.tags with duplicates, whitespace, and mixed casing
    When the indexer stores the post
    Then craftsky_posts.tags is non-null
    And each tag is trimmed and lowercased
    And duplicates are removed
    And the result is suitable for existing hashtag queries
```

### AT-012: Project Posts Behave As Ordinary Posts For Interactions And Moderation

Requirement IDs: RULE-002  
Acceptance Criteria: AC-012  
Priority: Must  
Level: Acceptance  
Automation Target: `appview/internal/api/post_store_test.go`, `report_test.go`, `timeline_store_test.go`, `appview/internal/index/craftsky_interaction_test.go`

```gherkin
Feature: Project posts keep ordinary post semantics
  Scenario: Existing post flows operate on a project post
    Given an indexed project post exists
    When users like, repost, reply to, report, moderate, read, and delete it through existing post flows
    Then those flows treat it as a normal social.craftsky.feed.post
    And no project-specific special case is required beyond response hydration
```

### AT-013: Project Query Paths Have Supporting Indexes

Requirement IDs: NFR-003  
Acceptance Criteria: AC-013  
Priority: Should  
Level: Acceptance / Manual Review  
Automation Target: migration/schema assertions plus manual `EXPLAIN` review if no stable automated plan test exists

```gherkin
Feature: Project query performance
  Scenario: Project list and filter paths are index-supported
    Given implementation has completed
    When schema indexes and representative project profile list/count/filter query plans are reviewed
    Then project profile list and count queries have explicit supporting indexes or reuse an existing suitable index
    And craft type and array membership filters have explicit supporting indexes or a documented reason for deferral
```

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-006 | AC-011 | Normalize and merge facet-derived tags with `project.common.tags`. | Facet tags `#FairIsle`, `#wip`; project tags ` fairisle `, `WIP`, `linen`. | Non-null array contains lowercased, trimmed, deduped tags, e.g. `fairisle`, `wip`, `linen`. | `appview/internal/postutil/tags_test.go` or index helper tests |
| UT-002 | FR-003 | AC-003, AC-010 | Map known craft detail variants into raw/details-type/materialized fields. | Knitting, crochet, quilting, sewing detail JSON with representative fields. | Details `$type`, raw JSON, and known detail columns are populated consistently. | `appview/internal/index/craftsky_post_internal_test.go` |
| UT-003 | FR-005 | AC-004, EC-003 | Extract common fields without rejecting unknown open-union details. | Project JSON with `common.craftType` and `details.$type` for a future NSID. | Common fields and raw JSON can be retained; no parser poison-pill solely from unknown details. | `appview/internal/index/craftsky_post_internal_test.go` |
| UT-004 | FR-007, FR-008, NFR-004 | AC-006 | Decode post create allows `project` but still rejects `createdAt` and malformed bodies. | JSON bodies with valid project, malformed project, and `createdAt`. | Valid project decodes; malformed body and `createdAt` return standard `FieldError` codes. | `appview/internal/api/post_request_test.go` |
| UT-005 | FR-007 | AC-005, AC-006 | Validate minimal valid project create shape. | `project.common.craftType` with optional common/detail fields. | Request passes validation without requiring client-supplied `createdAt`. | `appview/internal/api/post_request_test.go` |
| UT-006 | FR-008 | AC-006, EC-004 | Reject invalid project request structures. | Missing `project.common`, missing `craftType`, wrong field types. | Validation fails with field-specific standard error and no write path is invoked. | `appview/internal/api/post_request_test.go` |
| UT-007 | FR-009, NFR-004 | AC-007 | Build `PostResponse` with lexicon-shaped project object for project rows. | `PostRow` or project-bearing row with raw/materialized project metadata. | Marshaled JSON includes camelCase `project` with `common` and optional `details`. | `appview/internal/api/post_response_test.go` |
| UT-008 | FR-009, FR-010 | AC-008 | Omit `project` from general post responses. | `PostRow` for a non-project post. | Marshaled JSON omits `project` while preserving existing fields. | `appview/internal/api/post_response_test.go` |
| UT-009 | FR-011, RULE-003 | AC-009 | Count predicate includes only top-level project posts. | Rows for top-level project, project reply, general post, hidden project. | Count logic includes visible top-level project posts only. | `appview/internal/api/profile_store_test.go` helper/query assertions |
| UT-010 | FR-012, NFR-004 | AC-010 | Decode profile project list pagination parameters consistently with existing list endpoints. | `limit`, absent cursor, valid opaque cursor, invalid cursor. | Limit caps and cursor errors match existing envelope conventions. | `appview/internal/api/post_test.go` or pagination helper tests |
| UT-011 | RULE-001 | AC-001, AC-002 | Determine project-ness from presence of `project.common`. | Records with no `project`, empty `project`, and `project.common.craftType`. | Only records containing valid `project.common` are treated as project posts. | `appview/internal/index/craftsky_post_internal_test.go` |
| UT-012 | BR-002, FR-007 | AC-005 | Build PDS createRecord body with server-stamped `createdAt` and client project payload. | Authenticated post create request with text, images, reply/quote, project. | PDS body includes all allowed fields and project; client `createdAt` cannot override server timestamp. | `appview/internal/api/post_test.go` |
| UT-013 | NFR-004 | AC-006, AC-007, AC-010 | Verify new/changed JSON fields use camelCase. | Serialized create errors, project post response, profile projects page. | No snake_case keys in `/v1/*` JSON bodies. | `appview/internal/api/*_test.go` |

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | BR-001, FR-001, FR-002, NFR-003 | AC-001, AC-013 | Apply migration and assert project schema, constraints, cascade, and indexes. | Real test DB or migration harness with existing `craftsky_posts` schema. | Run migration chain/up migration and inspect catalogs/data. | Base flags/table exist; FK/cascade works; no historical backfill is required; supporting indexes exist. | `appview/cmd/cli/migrate_test.go` or DB migration test suite |
| IT-002 | FR-001, FR-004, RULE-001 | AC-002 | Index create events for general and project posts. | `testdb.WithSchema` updated with project tables and seeded member profile. | Handle create events with and without `project.common`. | General post has no project row; project post has base flags plus project row. | `appview/internal/index/craftsky_post_test.go` |
| IT-003 | FR-002, FR-003, FR-006 | AC-003, AC-011 | Index full known craft project payload. | Project post JSON with common fields, pattern, materials/colors/designTags/tags, hashtag facets, knitting details. | Handle create event. | Project table has raw/materialized fields and `craftsky_posts.tags` has merged normalized tags. | `appview/internal/index/craftsky_post_test.go` |
| IT-004 | FR-004, NFR-002 | AC-004 | Same URI/CID replay is idempotent. | Seed member and project create event. | Handle event twice. | Exactly one `craftsky_posts` row and one `craftsky_project_posts` row exist with stable values. | `appview/internal/index/craftsky_post_test.go` |
| IT-005 | FR-004, FR-005, NFR-002 | AC-004, EC-003 | Updates, project removal, delete, and unknown details converge. | Seed member and create/update/delete events for same URI. | Handle project create, CID-changing update, general-post update, unknown-details project, delete. | Updated values replace old ones; project row removed when project removed; unknown details do not poison-pill; delete cascades/removes rows. | `appview/internal/index/craftsky_post_test.go` |
| IT-006 | BR-002, FR-007, NFR-004 | AC-005, AC-006 | Create handler sends project-bearing record to PDS and returns 201. | Fake authenticated request context and fake PDS client. | POST valid project request. | Fake PDS receives project payload in `social.craftsky.feed.post`; response is 201 with `project`. | `appview/internal/api/post_test.go` |
| IT-007 | FR-008 | AC-006, EC-004 | Create handler rejects invalid project without PDS write. | Fake authenticated request and fake PDS recording call count. | POST malformed or invalid project body. | Standard error envelope; PDS create call count remains zero. | `appview/internal/api/post_test.go` |
| IT-008 | FR-009, FR-010, NFR-001 | AC-007 | Store/read hydrates project metadata on single read and author list. | DB with project materialization joined to `craftsky_posts`. | Read single post and profile posts list. | Rows/responses include project metadata from Postgres only. | `appview/internal/api/post_store_test.go` |
| IT-009 | FR-009, FR-010 | AC-008 | General post read/list responses omit project. | DB with general post and no project row. | Read single/list endpoints. | Project field absent; existing response compatibility preserved. | `appview/internal/api/post_store_test.go`, `post_response_test.go` |
| IT-010 | FR-011, RULE-003 | AC-009 | Profile read computes projectCount with moderation and top-level predicates. | DB with visible project roots, project replies, general roots, hidden project roots. | `ProfileStore.Read`. | `ProjectCount` equals visible top-level project roots only; `PostCount` still includes project posts as posts. | `appview/internal/api/profile_store_test.go` |
| IT-011 | FR-002, FR-003, FR-012, RULE-003, NFR-001, NFR-004 | AC-010 | Profile projects store method returns paginated project `PostResponse` rows. | DB with mixed posts and project rows for one profile, plus moderation output. | List profile projects with page size/cursor. | Only visible top-level project posts returned in existing order with project metadata and opaque cursor. | `appview/internal/api/post_store_test.go` or new project store tests |
| IT-012 | FR-010 | AC-007 | Timeline, comments/replies, and notifications hydrate project rows consistently. | DB/fake stores seeded with project posts in each surface. | Call `ListTimeline`, comments/replies list methods, notification hydration paths. | Every returned project post carries project metadata; general posts still omit it. | `timeline_store_test.go`, `post_store_test.go`, `notification_store_test.go` |
| IT-013 | FR-012, NFR-004 | AC-010 | New route is authenticated, requires device ID, resolves handle/DID, and uses existing envelope conventions. | Routes mux with fake deps. | Request `GET /v1/profiles/{handleOrDid}/projects` with missing auth, missing device, invalid identifier, valid request. | 401/400 errors match route stack; valid request calls handler and returns paginated JSON. | `appview/internal/routes/routes_test.go`, `appview/internal/api/post_test.go` |
| IT-014 | RULE-002 | AC-012 | Existing interaction/report/moderation/delete flows work on project posts. | Seed project post row plus interaction/report/moderation fixtures. | Like, repost, reply, report, moderate, delete/read via existing flows. | Existing flows treat URI as ordinary post; response hydration is the only project-specific behavior. | Existing interaction, report, post, timeline, moderation tests |
| IT-015 | BR-002, NFR-001 | AC-007, AC-010 | Read paths do not invoke PDS clients for project hydration. | Fake PDS or hydrator that fails if called; DB has project rows. | Exercise read/list/project list handlers. | Requests succeed without any PDS hydration call. | Handler/store tests with fake PDS/hydrator |

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Test |
|---|---|---|---|
| REG-001 | Existing facet-derived hashtag indexing remains supported. | FR-006 | Existing general post facet tags still populate `craftsky_posts.tags`; project tags are additive, not a replacement. |
| REG-002 | General post create remains valid when `project` is absent. | FR-007, FR-008 | `POST /v1/posts` with text-only or image post succeeds exactly as before and omits project in response. |
| REG-003 | Client-supplied `createdAt` remains disallowed. | FR-008 | Existing `DecodePostCreate_RejectsCreatedAtField` behavior remains, including when a project object is also present. |
| REG-004 | AppView preserves full `record JSONB` for debugging/backfill. | FR-005 | Indexed general and project posts store the complete source record JSON including unknown or future project fields. |
| REG-005 | Existing post-shaped surfaces keep pagination/order/moderation behavior. | FR-010 | Profile posts, timeline, comments/replies, and notifications return the same rows/order as before, with project hydration added only where applicable. |
| REG-006 | Like/repost/reply/report/delete flows operate on all posts by URI. | RULE-002 | Existing interaction and moderation tests pass when the target post has project materialization. |
| REG-007 | Profile `postCount` still counts root project posts as posts. | RULE-003 | Profile summary keeps `postCount` semantics while adding independent `projectCount`. |

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Minimal general post | `social.craftsky.feed.post` with text and server/indexed `createdAt`, no `project`. | AT-002, AT-008, IT-002, IT-009, REG-002 |
| TD-002 | Minimal project post | Text plus `project.common.craftType = social.craftsky.feed.defs#knitting`. | AT-002, IT-002, UT-011 |
| TD-003 | Full knitting project | Common fields: craftType, status, title, duration, pattern URL/name/difficulty/designer/publisher, materials, colors, designTags, tags; details `$type = social.craftsky.project.knitting#details`, projectType, projectSubtype, yarnWeight, needleSizeMm, gauge, finishedSize. | AT-003, AT-005, AT-007, IT-003, IT-006 |
| TD-004 | Other known craft detail variants | Sewing, quilting, and crochet detail payloads with one representative materialized field per variant. | UT-002, IT-003, IT-011 |
| TD-005 | Unknown future details variant | `project.common.craftType` plus `details.$type = social.craftsky.project.future#details` and arbitrary detail fields. | AT-004, UT-003, IT-005, REG-004 |
| TD-006 | Tag normalization set | Facet tags and `project.common.tags` containing mixed case, whitespace, duplicates, and unique tags. | AT-011, UT-001, IT-003 |
| TD-007 | Profile mixed post set | Top-level general post, top-level project posts, project reply/comment, quote project root, hidden project post. | AT-009, AT-010, IT-010, IT-011, REG-007 |
| TD-008 | Invalid create bodies | Malformed JSON; project without common; common without craftType; wrong field types; client `createdAt`. | AT-006, UT-004, UT-006, IT-007 |
| TD-009 | Tap convergence event stream | Same URI/CID replay, CID-changing project update, update removing project, delete event. | AT-004, IT-004, IT-005 |
| TD-010 | Auth and route variants | Valid bearer/device headers; missing auth; missing `X-Craftsky-Device-Id`; invalid handle/DID. | AT-010, IT-013 |

## 8. Manual Checks

| ID | Requirement IDs | Check | Steps | Expected Result |
|---|---|---|---|---|
| MAN-001 | NFR-001 | Confirm no PDS read-through on read/list paths. | Review changed read/list handlers and stores after automated tests; search for PDS/hydrator calls in project read paths. | Project metadata comes only from Postgres materialization in the happy path. |
| MAN-002 | NFR-003 | Query-plan/index review for project counts/lists/filters. | With representative data, inspect schema indexes and run `EXPLAIN` for profile project count/list and key project filters if practical. | Queries use explicit supporting indexes or the implementation documents why existing indexes/deferrals are sufficient. |
| MAN-003 | BR-001, FR-009, NFR-004 | API response shape review. | Inspect sample JSON for create response, single read, timeline, profile projects, and general post responses. | Project posts expose lexicon-shaped `project`; general posts omit `project`; all `/v1/*` keys are camelCase. |
| MAN-004 | RISK-002, FR-010 | Cross-surface hydration review. | Confirm all code paths returning `PostResponse` use shared hydration/building logic or have explicit project tests. | No post-shaped endpoint accidentally omits project metadata. |

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | Automated query-plan assertions may be brittle or unavailable. | NFR-003 | PostgreSQL planner choices can vary by data size/version, and the repo may not have an existing stable EXPLAIN assertion convention. | Assert indexes structurally in migration tests; perform MAN-002 before implementation approval; document any intentionally deferred filters. |
| GAP-002 | Unknown open-union detail behavior depends on generated lexicon unmarshalling and fallback implementation choices. | FR-005 | The current generated types may reject unknown variants unless the indexer preserves/parses raw JSON before typed decoding. | Add UT-003/IT-005 early; if implementation cannot preserve unknown details with current generated types, pause for requirements/design review rather than dropping records. |
| GAP-003 | Full migration-chain coverage depends on having a migration-chain test harness. | FR-001, FR-002 | Existing tests often use inline DDL fixtures rather than applying every migration. Historical backfill is out of scope. | Prefer adding migration-chain coverage; if not feasible, create an isolated migration SQL test plus MAN-002/manual schema inspection. |

## 10. Out Of Scope

- Flutter composer, card, profile Projects tab, and Dart model tests are out of scope for this AppView slice (NG-002).
- Lexicon schema tests or generated lexicon changes are out of scope unless implementation discovers a blocking mismatch (NG-001).
- Project search, recommendations, ranking, algorithmic feeds, and global discovery endpoint tests are out of scope (NG-003).
- Cross-post project identity, mutable project lifecycle records, or separate project collection tests are out of scope (NG-004, NG-006).
- Unauthenticated access and direct PDS read-through tests are out of scope except negative checks that new routes remain authenticated and read paths do not fetch PDS records (NG-005).
- Post edit/update endpoint tests are out of scope unless an existing endpoint independently supports updates; indexer update convergence is still in scope because Tap events can update records (NG-007).

## 11. Handoff To Document Review

- Requirements file: `docs/changes/2026-06-07-appview-project-posts/01-requirements.md`
- Test specification: `docs/changes/2026-06-07-appview-project-posts/02-acceptance-tests.md`
- Next review artifact: `03-document-review.md`
- External Plannotator review, if the user initiates it outside this agent: `docs/changes/2026-06-07-appview-project-posts/`
- Risk level carried forward: High.
- Risk-based review recommendation: Required before implementation. This test design exposes significant risks around schema migration, unknown details handling, and consistent response hydration across every `PostResponse` path.
- Recommended first failing test for implementation: `IT-001` — migration/schema test proving minimal base project flags, `craftsky_project_posts`, FK/cascade, no required historical backfill, and supporting indexes.
- Suggested test order for implementation:
  1. `IT-001` migration/schema/index coverage without historical backfill.
  2. `IT-002`, `IT-003`, `UT-001`, `UT-002`, `UT-011` for core project indexing and materialization.
  3. `IT-004`, `IT-005`, `UT-003` for idempotency, update/delete convergence, and unknown details.
  4. `UT-004`, `UT-005`, `UT-006`, `UT-012`, `IT-006`, `IT-007` for create request/PDS write behavior.
  5. `UT-007`, `UT-008`, `IT-008`, `IT-009`, `IT-012` for `PostResponse` hydration and compatibility.
  6. `IT-010`, `IT-011`, `UT-009`, `UT-010`, `IT-013` for profile counts and the profile projects endpoint.
  7. `IT-014` and all `REG-*` tests for interaction, moderation, delete, and existing behavior preservation.
  8. `MAN-001` through `MAN-004` before final approval.
- Commands discovered:
  - `just test`
  - `just fmt`
- Blocking gaps: None for test design. `GAP-001` through `GAP-003` are implementation/review risks that should be addressed before accepting the completed implementation.
