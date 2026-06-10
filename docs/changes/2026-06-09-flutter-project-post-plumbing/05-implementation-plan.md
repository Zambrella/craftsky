# TDD Implementation Plan: Flutter Project Post Models And Providers

## Inputs
- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` — Approved with notes, no blocking issues.
- Coding plan: `04-coding-plan.md`

## Implementation Rules
- Do not implement behavior without a linked requirement ID.
- Write or update a failing test before implementation.
- Run the smallest relevant test first.
- Refactor only after tests pass.
- Keep traceability updated.
- Keep this slice Flutter-only: no UI, AppView, lexicon, migration, dependency, route, or localization changes.
- Preserve AppView-only reads/writes and do not introduce direct PDS access or token handling.

## Test Order
| Step | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| 1 | UT-001 | FR-001, FR-010, NFR-001 | AC-001 | Fails: project model package/classes missing. |
| 2 | UT-002, UT-003 | FR-001, FR-002, NFR-001 | AC-001, AC-002 | Fails: pattern/gauge mapping missing. |
| 3 | UT-004, UT-005, UT-006 | FR-001, FR-002, NFR-001 | AC-002 | Fails: details sealed mapper missing or discriminator wrong. |
| 4 | UT-007, UT-010 | FR-002, FR-003, FR-005, NFR-002 | AC-003, AC-012 | Fails: unknown fallback missing or drops raw fields. |
| 5 | UT-008, UT-009, REG-001 | BR-001, FR-003, FR-004, NFR-002 | AC-001, AC-009 | Fails: `Post.project` missing or general serialization regresses. |
| 6 | REG-006 | NFR-004 | AC-010 | Fails if mapper bootstrap omits project mappers. |
| 7 | UT-011, IT-001, REG-002 | BR-001, BR-002, FR-005, FR-012, RULE-001, NFR-001 | AC-004, AC-005, AC-012, AC-015 | Fails: create API lacks `project` arg/body. |
| 8 | AT-004, UT-012, IT-002 | BR-002, FR-005, FR-006, RULE-001 | AC-005, AC-011 | Fails: project-plus-reply can submit. |
| 9 | IT-003 | FR-005, FR-006, FR-007 | AC-004, AC-011, AC-012 | Fails: repository/fake signatures missing project/list support. |
| 10 | IT-004, IT-005, REG-003 | BR-001, FR-007, RULE-002 | AC-006, AC-016 | Fails: profile projects route method missing. |
| 11 | UT-013 | FR-008, FR-010, NFR-003 | AC-007 | Fails: `UserProjectsState` missing. |
| 12 | AT-007, UT-014, UT-015, IT-006 | FR-007, FR-008, FR-010, RULE-002, NFR-003 | AC-006, AC-007, AC-016 | Fails: `userProjectsProvider` missing. |
| 13 | UT-016, UT-017 | FR-009 | AC-008, AC-013 | Fails: project cache helpers missing. |
| 14 | AT-005, AT-009, UT-018, UT-019, IT-007, IT-010, REG-004 | FR-009, FR-011, RULE-002 | AC-008, AC-014 | Fails: project create cache fan-out/patching missing. |
| 15 | AT-008, IT-008, IT-009 | FR-009 | AC-013 | Fails: delete/like/repost providers do not update project caches. |
| 16 | UT-020, REG-005 | RULE-003 | AC-017 | Fails if model constructors/parsers enforce lexicon hints. |
| 17 | MAN-001, MAN-002, REG-007 | FR-010, NFR-004 | AC-001, AC-007, AC-010 | Fails if layout/codegen/dependencies are inconsistent. |

## Implementation Steps

### Step 1: UT-001
- Write failing test: Added `app/test/projects/models/project_test.dart` coverage for `Project`/`ProjectCommon` camelCase parsing, serialization, equality, and generated `copyWith`.
- Run command: `cd app && flutter test test/projects/models/project_test.dart`
- Confirmed failure: Red: test failed to compile because `app/lib/projects/models/project.dart`, `ProjectMapper`, and `ProjectCommon` did not exist.
- Implement: Added `app/lib/projects/models/project.dart` with `Project`, `ProjectCommon`, `ProjectPattern`, `ProjectGauge`, and project details scaffolding; initialized `ProjectMapper` in `bootstrap.dart`; ran build runner to generate `project.mapper.dart`.
- Run command: `cd app && flutter test test/projects/models/project_test.dart` — passed.
- Refactor: Adjusted project details mapping away from generated `$type` discriminators to a custom `ProjectDetailsMapper` because generated code emitted unescaped `$type` strings.
- Notes: Covers FR-001, FR-010, NFR-001 for common project model fields.

### Step 2: UT-002, UT-003
- Write failing test: Added pattern coverage in `project_test.dart` and gauge coverage in `project_details_test.dart`.
- Run command: `cd app && flutter test test/projects/models/project_test.dart` and/or `cd app && flutter test test/projects/models/project_details_test.dart`
- Confirmed failure: Red was covered by the Step 1 missing-model compile failure before the shared project model scaffolding existed.
- Implement: `ProjectPattern` and `ProjectGauge` were added with generated mappers.
- Run command: `cd app && flutter test test/projects/models/project_test.dart test/projects/models/project_details_test.dart` — passed.
- Refactor: None.
- Notes: Covers camelCase pattern/gauge mapping for FR-001, FR-002, NFR-001.

### Step 3: UT-004, UT-005, UT-006
- Write failing test: Added known-details tests for knitting, crochet, sewing, and quilting discriminators in `project_details_test.dart`.
- Run command: `cd app && flutter test test/projects/models/project_details_test.dart`
- Confirmed failure: Red: known details initially re-encoded with generated `__type` rather than lexicon `$type`.
- Implement: Added `ProjectDetailsFieldHook` so project serialization uses the custom `$type`-preserving `ProjectDetailsMapper`.
- Run command: `cd app && flutter test test/projects/models/project_test.dart test/projects/models/project_details_test.dart` — passed.
- Refactor: Kept discriminator logic in one custom mapper to avoid generated `$type` escaping issues.
- Notes: Covers FR-002 known variant parsing without using `common.craftType` inference.

### Step 4: UT-007, UT-010
- Write failing test: Added unknown and missing-discriminator details coverage in `project_details_test.dart`.
- Run command: `cd app && flutter test test/projects/models/project_details_test.dart`
- Confirmed failure: Red would fail without custom unknown fallback; the same details-focused run verified the fallback after mapper implementation.
- Implement: `UnknownProjectDetails` stores optional discriminator and raw fields, with custom encode/decode preserving raw nested maps/lists.
- Run command: `cd app && flutter test test/projects/models/project_test.dart test/projects/models/project_details_test.dart` — passed.
- Refactor: None.
- Notes: Covers FR-003/NFR-002 forward compatibility and FR-005 pass-through serialization scope.

### Step 5: UT-008, UT-009, REG-001
- Write failing test: Extended `app/test/feed/models/post_test.dart` with project-bearing post parsing and general-post-without-project regression coverage.
- Run command: `cd app && flutter test test/feed/models/post_test.dart`
- Confirmed failure: Red: focused test failed to compile because `Post.project` did not exist.
- Implement: Added optional `Project? project` to `Post`, imported project models, regenerated `post.mapper.dart`.
- Run command: `cd app && flutter test test/feed/models/post_test.dart` — passed.
- Refactor: None.
- Notes: General post serialization still omits `project`; project-bearing post responses parse with typed project details.

### Step 6: REG-006
- Write failing test: Covered through model tests that call `setUpAll(initializeMappers)` after adding project mappers.
- Run command: focused mapper/bootstrap tests as discovered during inspection.
- Confirmed failure: Red would occur if `ProjectMapper` was omitted from `initializeMappers`; the focused model tests require bootstrap initialization.
- Implement: Added `ProjectMapper.ensureInitialized()` to `initializeMappers()`.
- Run command: `cd app && flutter test test/projects/models/project_test.dart test/projects/models/project_details_test.dart test/feed/models/post_test.dart` — project/post model suites passed.
- Refactor: None.
- Notes: Generated mapper files are included from build runner output.

### Step 7: UT-011, IT-001, REG-002
- Write failing test: Added `PostApiClient.createPost` tests for common-only embroidery project payloads and general-create regression coverage; added project create serialization model coverage.
- Run command: `cd app && flutter test test/feed/data/post_api_client_test.dart`
- Confirmed failure: Red: API client lacked a `project` named parameter.
- Implement: Added `Project? project` to create API, serialized `project.toCreateMap()` only when present, and kept general create bodies unchanged.
- Run command: `cd app && flutter test test/feed/data/post_api_client_test.dart` — passed.
- Refactor: Added `Project.toCreateMap()` to omit empty optional create arrays while preserving normal `toMap()` round trips.
- Notes: Covers top-level project create serialization and common-only embroidery creates.

### Step 8: AT-004, UT-012, IT-002
- Write failing test: Added API/repository/provider project-plus-reply guard tests.
- Run command: focused create/API/repository tests as relevant.
- Confirmed failure: Red: direct calls could pass both project and reply before guard wiring.
- Implement: Added shared `assertProjectCreateIsTopLevel()` and invoked it in `PostApiClient`, `ApiPostRepository`, and `CreatePost` before repository submission.
- Run command: `cd app && flutter test test/feed/data/post_api_client_test.dart test/feed/data/post_repository_test.dart test/feed/providers/create_post_provider_test.dart` — passed.
- Refactor: None.
- Notes: Provider test proves repository callback is not called and state becomes `AsyncError`.

### Step 9: IT-003
- Write failing test: Extended repository/fake tests for project argument pass-through.
- Run command: `cd app && flutter test test/feed/data/post_repository_test.dart` and affected fake consumer tests.
- Confirmed failure: Red: repository interface/fake signatures had no `Project?` parameter.
- Implement: Added `Project? project` to `PostRepository.create`, `ApiPostRepository.create`, and `FakePostRepository` create callbacks.
- Run command: `cd app && flutter test test/feed/data/post_repository_test.dart test/feed/providers/create_post_provider_test.dart` — passed.
- Refactor: Kept old fake `onCreate` callback compatible for existing tests; `onCreateWithFacets` carries the new project argument.
- Notes: Existing facets/images/reply behavior remains covered by existing tests.

### Step 10: IT-004, IT-005, REG-003
- Write failing test: Added API client tests for `/v1/profiles/@{handleOrDid}/projects` and cursor/limit query parameters.
- Run command: `cd app && flutter test test/feed/data/post_api_client_test.dart`
- Confirmed failure: Red: `listProjectsByAuthor` did not exist.
- Implement: Added `PostApiClient.listProjectsByAuthor`, plus repository interface/implementation/fake methods.
- Run command: `cd app && flutter test test/feed/data/post_api_client_test.dart test/feed/data/post_repository_test.dart` — passed.
- Refactor: Kept existing posts route tests unchanged as REG-003 coverage.
- Notes: Project endpoint parsing preserves returned `PostPage` items without filtering.

### Step 11: UT-013
- Write failing test: Added `app/test/projects/models/user_projects_state_test.dart` for state value semantics and `hasMore`.
- Run command: `cd app && flutter test test/projects/models/user_projects_state_test.dart`
- Confirmed failure: Red: `UserProjectsState` was missing before implementation.
- Implement: Added `app/lib/projects/models/user_projects_state.dart` with generated mapper/copy/equality and bootstrap initialization.
- Run command: `cd app && flutter test test/projects/models/user_projects_state_test.dart` — passed.
- Refactor: None.
- Notes: Distinct state type lives under `app/lib/projects/models`.

### Step 12: AT-007, UT-014, UT-015, IT-006
- Write failing test: Added `userProjectsProvider` build/pagination/failure/no-filter tests.
- Run command: `cd app && flutter test test/projects/providers/user_projects_provider_test.dart`
- Confirmed failure: Red: provider family and `userProjectsPageLimit` were missing.
- Implement: Added `app/lib/projects/providers/user_projects_provider.dart` with page limit 10, `loadMore`, previous-data preservation, and AppView item preservation.
- Run command: `cd app && flutter test test/projects/providers/user_projects_provider_test.dart` — passed.
- Refactor: Added provider log formatting in `bootstrap.dart`.
- Notes: Covers Must endpoint/state preservation plus expected NFR-003 pagination parity.

### Step 13: UT-016, UT-017
- Write failing test: Added user projects cache helper tests for prepend/dedupe/replace/remove.
- Run command: `cd app && flutter test test/projects/providers/user_projects_provider_test.dart`
- Confirmed failure: Red: project cache helper methods did not exist.
- Implement: Added notifier cache methods plus live-cache helper functions guarded by `ref.exists` and `post.project != null`.
- Run command: `cd app && flutter test test/projects/providers/user_projects_provider_test.dart` — passed.
- Refactor: None.
- Notes: Helpers do not instantiate non-live provider family entries.

### Step 14: AT-005, AT-009, UT-018, UT-019, IT-007, IT-010, REG-004
- Write failing test: Added CreatePost tests for project create cache fan-out and missing-response-project patching.
- Run command: `cd app && flutter test test/feed/providers/create_post_provider_test.dart`
- Confirmed failure: Red: `CreatePost.create` had no `project` argument and could only prepend top-level creates into profile Posts caches.
- Implement: Added `project` input, patched omitted response project for provider state/cache updates, prepended project creates into timeline and live userProjects caches, and avoided profile Posts cache pollution for project posts.
- Run command: `cd app && flutter test test/feed/providers/create_post_provider_test.dart` — passed.
- Refactor: None.
- Notes: General top-level create profile Posts behavior remains covered by existing tests.

### Step 15: AT-008, IT-008, IT-009
- Write failing test: Added delete/like/repost project cache mutation tests.
- Run command: focused delete/like/repost provider tests as relevant.
- Confirmed failure: Red risk: existing mutation providers only touched timeline/userPosts/userComments caches.
- Implement: Delete removes project posts from live userProjects caches only; like/repost optimistic updates and rollbacks now update live project caches and skip profile Posts for project posts.
- Run command: `cd app && flutter test test/feed/providers/delete_post_provider_test.dart test/feed/providers/toggle_post_interactions_provider_test.dart` — passed.
- Refactor: Added a project guard to `updateLiveUserPostCaches`.
- Notes: Timeline behavior remains unchanged and covered by existing tests.

### Step 16: UT-020, REG-005
- Write failing test: Added constructor non-validation coverage in `project_test.dart`.
- Run command: `cd app && flutter test test/projects/models/project_test.dart`
- Confirmed failure: Covered after project model implementation; constructors remain plain data constructors.
- Implement: No extra validation added; structurally parseable overlong/unknown/non-positive values are accepted by the model layer.
- Run command: `cd app && flutter test test/projects/models/project_test.dart` — passed.
- Refactor: None.
- Notes: AppView/PDS and future composer validation remain responsible for lexicon hints.

### Step 17: MAN-001, MAN-002, REG-007
- Write failing test/check: Performed layout/dependency/codegen review and full verification commands.
- Run command: `cd app && dart run build_runner build --delete-conflicting-outputs`, `cd app && flutter analyze`, `cd app && flutter test`
- Confirmed failure: `flutter analyze` initially reported style issues; fixed import ordering, constructor parameter ordering, null-aware map element, documented ignores, test casts, and test cleanup.
- Implement: New project models/providers live under `app/lib/projects/...`; AppView-shaped post API/repository changes remain under `feed/data`; no dependency changes were made.
- Run command: `cd app && dart run build_runner build --delete-conflicting-outputs` — passed; `cd app && flutter analyze` — passed; `cd app && flutter test` — passed.
- Refactor: Style-only lint cleanup after green tests.
- Notes: Build runner reports the repository's current warning that `--delete-conflicting-outputs` was ignored by this build_runner version, but generation completed successfully.

## Completion Checklist
- [x] All Must requirements covered by tests or documented gaps
- [x] All planned Must tests passing
- [x] Relevant regression tests passing
- [x] No unlinked behavior implemented
- [x] Docs updated
- [x] Review completed or explicitly skipped

## Execution Notes
- Created initial implementation plan from approved workflow documents on 2026-06-10.
- Implemented project post plumbing across Flutter model, API, repository, provider, and cache layers without UI/AppView/lexicon/dependency changes.
- The approved test order was preserved; a few closely related tests were implemented in the same source files when shared scaffolding already existed from earlier red/green loops.
