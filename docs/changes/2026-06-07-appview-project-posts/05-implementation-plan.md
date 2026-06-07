# TDD Implementation Plan: AppView Project Posts

## Inputs
- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md`
- Coding plan: `04-coding-plan.md`

## Implementation Rules
- Do not implement behavior without a linked requirement ID.
- Write or update a failing test before implementation.
- Run the smallest relevant test first.
- Refactor only after tests pass.
- Keep traceability updated.
- No historical backfill is required for `000016_project_posts`; the user clarified this on 2026-06-07 and the source documents were updated before code changes.

## Test Order
| Step | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| 1 | IT-001 | BR-001, FR-001, FR-002, NFR-003 | AC-001, AC-013 | Fails: missing project schema/indexes |
| 2 | UT-001 | FR-006 | AC-011 | Fails: project tags ignored |
| 3 | IT-002 / UT-011 | FR-001, FR-004, RULE-001 | AC-002 | Fails: indexer does not set project flags/materialization |
| 4 | UT-002 / IT-003 | FR-002, FR-003, FR-006 | AC-003, AC-010, AC-011 | Fails: project fields/details are not materialized |
| 5 | UT-003 / IT-005 | FR-004, FR-005, NFR-002 | AC-004 | Fails: unknown details/update/delete convergence unsupported |
| 6 | IT-004 | NFR-002 | AC-004 | Fails if replay duplicates/churns project rows |
| 7 | UT-004 / UT-005 / UT-006 | FR-007, FR-008, NFR-004 | AC-006 | Fails: create decoder rejects or under-validates project |
| 8 | UT-012 / IT-006 / IT-007 | BR-002, FR-007, FR-008 | AC-005, AC-006 | Fails: PDS body lacks project or invalid requests write |
| 9 | UT-007 / UT-008 / UT-013 | FR-009, NFR-004 | AC-007, AC-008 | Fails: response has no optional project field |
| 10 | IT-008 / IT-009 | FR-009, FR-010, NFR-001 | AC-007, AC-008 | Fails: store reads do not hydrate project |
| 11 | IT-012 | FR-010 | AC-007 | Fails: timeline/comments/notifications may miss project |
| 12 | UT-009 / IT-010 | FR-011, RULE-003 | AC-009 | Fails: projectCount hardcoded zero |
| 13 | UT-010 / IT-011 / IT-013 | FR-012, NFR-004 | AC-010 | Fails: profile projects endpoint absent |
| 14 | IT-014 / REG-* | RULE-002 | AC-012 | Fails if existing post flows regress |

## Implementation Steps

### Step 0: Documentation clarification
- Write failing test: N/A.
- Run command: N/A.
- Confirmed failure: Source documents conflicted on migration backfill scope.
- Implement: User clarified no backfill required; updated `01-requirements.md`, `02-acceptance-tests.md`, and `03-document-review.md` to align with `04-coding-plan.md` before code changes.
- Run command: N/A.
- Refactor: N/A.
- Notes: Completed before implementation code changes.

### Step 1: IT-001
- Write failing test: Added `appview/internal/db/project_posts_migration_test.go` to assert `000016_project_posts` base flags, child table, FK/cascade, and supporting indexes.
- Run command: `go test ./internal/db -run TestProjectPostsMigrationCreatesSchemaAndIndexes -count=1`
- Confirmed failure: Missing `../../migrations/000016_project_posts.up.sql`.
- Implement: Added `000016_project_posts.up/down.sql` with `is_project`, `project_craft_type`, `craftsky_project_posts`, cascade FK, profile-project/craft-type indexes, and project filter/array indexes. No historical backfill included.
- Run command: `go test ./internal/db -run TestProjectPostsMigrationCreatesSchemaAndIndexes -count=1`
- Refactor: None.
- Notes: Command passes in this environment; DB-backed assertions skip when `TEST_DATABASE_URL`/`DATABASE_URL` are unset after verifying the migration file exists.

### Step 2: UT-001
- Write failing test: Added `TestMergeTags_LowercasesTrimsDedupesAndPreservesFirstSeenOrder` and non-nil empty slice coverage.
- Run command: `go test ./internal/postutil -run 'TestMergeTags' -count=1`
- Confirmed failure: `undefined: postutil.MergeTags`.
- Implement: Added `postutil.MergeTags`.
- Run command: `go test ./internal/postutil -run 'TestMergeTags|TestExtractTags' -count=1`
- Refactor: None.
- Notes: Focused tag utility tests pass.

### Step 3: IT-002 / UT-011
- Write failing test: Updated indexer DB fixtures/tests for base `is_project`/`project_craft_type` and child-row materialization; added non-DB `TestExtractProjectForIndex_ProjectnessRequiresCommonCraftType`.
- Run command: `go test ./internal/index -run TestExtractProjectForIndex_ProjectnessRequiresCommonCraftType -count=1`
- Confirmed failure: `undefined: extractProjectForIndex`.
- Implement: Added raw project extraction, RULE-001 detection, transactional base upsert, project child upsert/delete, and project tag merging.
- Run command: `go test ./internal/index -run TestExtractProjectForIndex_ProjectnessRequiresCommonCraftType -count=1`; `go test ./internal/index -run 'TestCraftskyPost_Create_WithProjectPayload_MaterializesProject|TestCraftskyPost_Create_GeneralPostHasNoProjectRow' -count=1`
- Refactor: None.
- Notes: Non-DB focused test passes. DB-backed index tests pass/skip depending on test DB availability.

### Step 4: UT-002 / IT-003
- Write failing test: Added `TestExtractProjectForIndex_PreservesKnownAndUnknownDetails` for known detail type/raw gauge and unknown details preservation.
- Run command: `go test ./internal/index -run 'TestExtractProjectForIndex' -count=1`
- Confirmed failure: Covered by the implementation introduced in Step 3 before this focused test was added.
- Implement: Generic raw detail extraction plus materialized known-detail column mapping in `upsertProjectMaterialization`.
- Run command: `go test ./internal/index -run 'TestExtractProjectForIndex' -count=1`
- Refactor: None.
- Notes: Focused extraction tests pass; DB materialization tests compile/pass or skip based on DB availability.

### Step 5: UT-003 / IT-005
- Write failing test: Unknown future details covered by `TestExtractProjectForIndex_PreservesKnownAndUnknownDetails`; DB update/delete convergence remains in updated index integration fixtures.
- Run command: `go test ./internal/index -count=1`
- Confirmed failure: Covered by Step 3 implementation before the focused unknown-details test was added.
- Implement: Unknown details are parsed from raw JSON independently of generated open-union structs; project materialization is deleted when project is absent on update and cascades on post delete.
- Run command: `go test ./internal/index -count=1`
- Refactor: None.
- Notes: Index package tests pass in this environment; DB-backed portions skip without database configuration.

### Step 6: IT-004
- Write failing test: Existing idempotency test remains in `craftsky_post_test.go`; project child upsert uses `ON CONFLICT` with `raw_project IS DISTINCT FROM` guard.
- Run command: `go test ./internal/index -count=1`
- Confirmed failure: Existing DB-backed idempotency test is skipped when no DB is configured.
- Implement: Base upsert remains CID-guarded; project child upsert is idempotent and does not advance when `raw_project` is unchanged.
- Run command: `go test ./internal/index -count=1`
- Refactor: None.
- Notes: Covered structurally and by DB-backed test suite when a test database is available.

### Step 7: UT-004 / UT-005 / UT-006
- Write failing test: Replaced project rejection with `TestDecodePostCreate_AcceptsProjectField`; added minimal valid project and missing `craftType` validation tests.
- Run command: `go test ./internal/api -run 'TestDecodePostCreate_AcceptsProjectField|TestValidatePostCreate_AcceptsMinimalProject|TestValidatePostCreate_RejectsProjectWithoutCraftType' -count=1`
- Confirmed failure: `PostCreateRequest.Project`, `api.Project`, and `api.ProjectCommon` were undefined.
- Implement: Added `api.Project`, `ProjectCommon`, and `ProjectPattern`; allowed `project` in decode; retained `createdAt` rejection; validated non-empty `project.common.craftType`.
- Run command: `go test ./internal/api -run 'TestDecodePostCreate_AcceptsProjectField|TestValidatePostCreate_AcceptsMinimalProject|TestValidatePostCreate_RejectsProjectWithoutCraftType|TestDecodePostCreate_RejectsCreatedAtField' -count=1`
- Refactor: None.
- Notes: Focused request validation tests pass.

### Step 8: UT-012 / IT-006 / IT-007
- Write failing test: Added `TestCreatePost_WithProject_WritesProjectToPDSAndResponse` and `TestCreatePost_InvalidProjectDoesNotWritePDS`.
- Run command: `go test ./internal/api -run 'TestCreatePost_WithProject_WritesProjectToPDSAndResponse|TestCreatePost_InvalidProjectDoesNotWritePDS' -count=1`
- Confirmed failure: `PostResponse.Project` was undefined and synthetic create response had no project support.
- Implement: Included project in `lexiconRecordBody`, synthetic `PostRow.Project`, and merged facet/project tags for create responses.
- Run command: `go test ./internal/api -run 'TestCreatePost_WithProject_WritesProjectToPDSAndResponse|TestCreatePost_InvalidProjectDoesNotWritePDS' -count=1`
- Refactor: None.
- Notes: Focused create handler tests pass; invalid project validation occurs before PDS create calls.

### Step 9: UT-007 / UT-008 / UT-013
- Write failing test: Added response tests for project inclusion, general omission, and camelCase project keys.
- Run command: `go test ./internal/api -run 'TestBuildPostResponse_IncludesProjectForProjectRows|TestBuildPostResponse_OmitsProjectForGeneralRows' -count=1`
- Confirmed failure: Covered by Step 8 implementation before these response-specific tests were added.
- Implement: Added optional `PostResponse.Project` sourced from `PostRow.Project`.
- Run command: `go test ./internal/api -run 'TestBuildPostResponse_IncludesProjectForProjectRows|TestBuildPostResponse_OmitsProjectForGeneralRows' -count=1`
- Refactor: None.
- Notes: Focused response tests pass.

### Step 10: IT-008 / IT-009
- Write failing test: Added `PostStore` project/general read hydration tests and updated inline DDL fixtures for project materialization.
- Run command: `go test ./internal/api -run 'TestPostStore_ReadOne_HydratesProjectFromMaterialization|TestPostStore_ReadOne_GeneralPostOmitsProject' -count=1`
- Confirmed failure: DB-backed tests are skipped when no database is configured in this environment.
- Implement: Extended `PostRow` with project flags/raw project/typed project; updated shared `postSelectColumns`, scans, and joins to `craftsky_project_posts` across store reads/lists.
- Run command: `go test ./internal/api -run 'TestPostStore_ReadOne_HydratesProjectFromMaterialization|TestPostStore_ReadOne_GeneralPostOmitsProject' -count=1`; `go test ./internal/api -count=1`
- Refactor: Centralized project hydration through `scanPostRow` for all `postSelectColumns` callers.
- Notes: API package tests pass without DB; DB-backed tests require compose Postgres.

### Step 11: IT-012
- Write failing test: Cross-surface coverage relies on shared `postSelectColumns` users plus explicit notification subject hydration.
- Run command: `go test ./internal/api -run TestNotificationStore_ListNotifications_DerivesFollowNotificationsScopedToViewer -count=1`; `go test ./internal/api -count=1`
- Confirmed failure: Notification subject scan needed nullable project flag handling for no-subject follow rows.
- Implement: Added project joins for timeline and all shared post store read paths; updated notification subject select/scan/hydration with nullable project fields.
- Run command: `go test ./internal/api -run TestNotificationStore_ListNotifications_DerivesFollowNotificationsScopedToViewer -count=1`; `go test ./internal/api -count=1`
- Refactor: None beyond shared scan use.
- Notes: API package tests pass without DB; DB-backed notification/timeline/comment tests require compose Postgres.

### Step 12: UT-009 / IT-010
- Write failing test: Updated profile summary test to include top-level project, project reply, general roots, and hidden project post.
- Run command: `go test ./internal/api -run TestProfileStore_ReadByDID_ProfileSummaryCountsRootPosts -count=1`
- Confirmed failure: Existing implementation returned hardcoded `0 AS project_count` (DB-backed test requires compose Postgres).
- Implement: Replaced hardcoded zero with visible top-level `craftsky_posts.is_project = true` count using existing moderation hide/takedown predicate shape.
- Run command: `go test ./internal/api -run TestProfileStore_ReadByDID_ProfileSummaryCountsRootPosts -count=1`
- Refactor: None.
- Notes: Focused command passes/skips depending on DB availability.

### Step 13: UT-010 / IT-011 / IT-013
- Write failing test: Added profile projects handler test and `PostStore.ListProjectsByAuthor` top-level-project filtering test.
- Run command: `go test ./internal/api -run 'TestListProjectsByAuthor_HappyPath|TestPostStore_ListProjectsByAuthor_ReturnsOnlyTopLevelProjects' -count=1`
- Confirmed failure: `ListProjectsByAuthorHandler` and `PostStore.ListProjectsByAuthor` were absent before implementation.
- Implement: Added `ListProjectsByAuthor`, handler, fake-store method, and route `GET /v1/profiles/{handleOrDid}/projects` with the existing auth/device route stack.
- Run command: `go test ./internal/api -run 'TestListProjectsByAuthor_HappyPath|TestPostStore_ListProjectsByAuthor_ReturnsOnlyTopLevelProjects' -count=1`; `go test ./internal/routes -count=1`
- Refactor: Reused `listAuthorPostsHandler` and existing opaque cursor conventions.
- Notes: Handler and routes package tests pass; DB-backed store assertions require compose Postgres.

### Step 14: IT-014 / REG-*
- Write failing test: Existing interaction/report/moderation/delete regression suites remain in place; project support does not add special branches for these flows.
- Run command: `go test ./...`; `just test`
- Confirmed failure: `just test` is blocked because compose Postgres at `localhost:5433` is not running (`connect: connection refused`).
- Implement: Project posts remain rows in `craftsky_posts`; interactions/reports/deletes continue by post URI; project materialization uses FK cascade on post delete.
- Run command: `go test ./...`; `just fmt`; `just test`.
- Refactor: None.
- Notes: `go test ./...` passes without `TEST_DATABASE_URL`. `just test` is documented as environment-blocked until `just dev-d`/compose Postgres is running.

## Verification
- Focused commands:
  - `go test ./internal/db -run TestProjectPostsMigrationCreatesSchemaAndIndexes -count=1`
  - `go test ./internal/postutil -run 'TestMergeTags|TestExtractTags' -count=1`
  - `go test ./internal/index -run 'TestExtractProjectForIndex' -count=1`
  - `go test ./internal/index -count=1`
  - `go test ./internal/api -run 'TestDecodePostCreate_AcceptsProjectField|TestValidatePostCreate_AcceptsMinimalProject|TestValidatePostCreate_RejectsProjectWithoutCraftType|TestDecodePostCreate_RejectsCreatedAtField' -count=1`
  - `go test ./internal/api -run 'TestCreatePost_WithProject_WritesProjectToPDSAndResponse|TestCreatePost_InvalidProjectDoesNotWritePDS' -count=1`
  - `go test ./internal/api -run 'TestBuildPostResponse_IncludesProjectForProjectRows|TestBuildPostResponse_OmitsProjectForGeneralRows' -count=1`
  - `go test ./internal/api -run 'TestPostStore_ReadOne_HydratesProjectFromMaterialization|TestPostStore_ReadOne_GeneralPostOmitsProject' -count=1`
  - `go test ./internal/api -run TestProfileStore_ReadByDID_ProfileSummaryCountsRootPosts -count=1`
  - `go test ./internal/api -run 'TestListProjectsByAuthor_HappyPath|TestPostStore_ListProjectsByAuthor_ReturnsOnlyTopLevelProjects' -count=1`
  - `go test ./internal/routes -count=1`
- Broader commands:
  - `gofmt -w ...changed Go files`
  - `just fmt` — passed (`gofmt -w . && go vet ./...`).
  - `go test ./...` from `appview/` without `TEST_DATABASE_URL` — passed.
- Manual checks:
  - MAN-001: Reviewed read/list code paths; project hydration comes from `craftsky_project_posts` joins/raw JSON, not PDS clients.
  - MAN-002: Migration includes explicit profile-project, project craft type, common craft/status/difficulty, and array GIN indexes; no craft-detail-specific indexes beyond documented v1/common dimensions.
  - MAN-003: Response/create tests assert lexicon-shaped `project`, general omission, and camelCase keys.
  - MAN-004: Shared `scanPostRow` hydrates project for single read, profile posts/projects, timeline, comments/replies; notification subject hydration is explicit.
- Blocked commands:
  - `just test` — blocked because compose Postgres at `localhost:5433` was not running; failures were connection refused before test schemas could be created.

## Completion Checklist
- [x] All Must requirements covered by tests or documented gaps
- [x] All planned Must tests passing where runnable without compose Postgres; DB-backed tests documented as requiring compose Postgres
- [x] Relevant regression tests passing where runnable without compose Postgres
- [x] No unlinked behavior implemented
- [x] Docs updated
- [x] Review completed or explicitly skipped
