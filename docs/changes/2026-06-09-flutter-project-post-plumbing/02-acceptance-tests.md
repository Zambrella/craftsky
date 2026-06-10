# Acceptance Test Specification: Flutter Project Post Models And Providers

## 1. Test Strategy

This stage designs tests for the Flutter data/model/provider slice only. It intentionally does not create or edit test files. The implementation stage should add failing tests first, then implement the smallest code needed to pass them.

Primary strategy:

- **Model/unit coverage** for `Project`, `ProjectCommon`, `ProjectPattern`, gauge-like sub-objects, sealed known details variants, sparse fields, unknown/raw details, `Post.project`, `dart_mappable` value behavior, and constructor non-validation.
- **API/client coverage** for `POST /v1/posts` project payload serialization, project-plus-reply rejection, and `GET /v1/profiles/@{handleOrDid}/projects` path/query parsing.
- **Repository/fake coverage** for new method signatures and exact pass-through of optional `Project`, cursor, and limit arguments.
- **Riverpod provider coverage** for `userProjectsProvider` pagination and project-aware cache updates across create, delete, like, unlike, repost, and unrepost flows.
- **Regression coverage** proving general post parsing/create/profile Posts behavior remains unchanged when `project` is absent.
- **Manual/review checks** limited to generated-file/dependency consistency and package organization where a human review is useful after code generation.

Risk level carried forward from requirements: **Medium**. Review recommendation before implementation: **document review is recommended but may be skipped by explicit user choice** because this slice changes model wire contracts, generated code, method signatures, and live cache behavior.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-004, AC-006 | AT-001, AT-002, AT-003, UT-001, UT-008, IT-001, IT-004 | Acceptance / Unit / Integration | Yes |
| BR-002 | AC-004, AC-005, AC-008 | AT-002, AT-004, AT-005, IT-002, IT-007, IT-008, REG-002 | Acceptance / Integration / Regression | Yes |
| FR-001 | AC-001, AC-002 | AT-001, UT-001, UT-002, UT-003, UT-005, UT-006 | Acceptance / Unit | Yes |
| FR-002 | AC-002, AC-003 | AT-001, AT-006, UT-003, UT-004, UT-005, UT-006, UT-007 | Acceptance / Unit | Yes |
| FR-003 | AC-003, AC-009 | AT-006, UT-007, UT-009, UT-010, REG-001 | Acceptance / Unit / Regression | Yes |
| FR-004 | AC-001, AC-009 | AT-001, UT-008, UT-009, REG-001 | Acceptance / Unit / Regression | Yes |
| FR-005 | AC-004, AC-005, AC-012 | AT-002, AT-004, UT-011, IT-001, IT-002, IT-003, REG-002 | Acceptance / Unit / Integration / Regression | Yes |
| FR-006 | AC-005, AC-011 | AT-004, UT-012, IT-002, IT-003 | Acceptance / Unit / Integration | Yes |
| FR-007 | AC-006 | AT-003, IT-004, IT-005, IT-006 | Acceptance / Integration | Yes |
| FR-008 | AC-007 | AT-007, UT-013, UT-014, UT-015, IT-006 | Acceptance / Unit / Integration | Yes |
| FR-009 | AC-008, AC-013 | AT-005, AT-008, UT-016, UT-017, UT-018, IT-007, IT-008, IT-009, REG-004 | Acceptance / Unit / Integration / Regression | Yes |
| FR-010 | AC-001, AC-007 | MAN-001, UT-001, UT-013 | Manual / Unit | Partial |
| FR-011 | AC-014 | AT-009, UT-019, IT-010 | Acceptance / Unit / Integration | Yes |
| FR-012 | AC-015 | AT-002, UT-011, IT-001 | Acceptance / Unit / Integration | Yes |
| RULE-001 | AC-004, AC-005 | AT-002, AT-004, IT-001, IT-002, REG-002 | Acceptance / Integration / Regression | Yes |
| RULE-002 | AC-006, AC-008, AC-016 | AT-003, AT-005, AT-007, UT-015, IT-004, IT-007, REG-003, REG-004 | Acceptance / Unit / Integration / Regression | Yes |
| RULE-003 | AC-017 | UT-020, REG-005 | Unit / Regression | Yes |
| NFR-001 | AC-001, AC-004 | UT-001, UT-002, UT-003, UT-005, IT-001, REG-002 | Unit / Integration / Regression | Yes |
| NFR-002 | AC-003, AC-009 | AT-006, UT-007, UT-009, UT-010, REG-001 | Acceptance / Unit / Regression | Yes |
| NFR-003 | AC-007 | AT-007, UT-013, UT-014, IT-006 | Acceptance / Unit / Integration | Yes |
| NFR-004 | AC-010 | MAN-002, REG-006, REG-007 | Manual / Regression | Partial |

## 3. Acceptance Scenarios

### AT-001: Project-bearing posts parse into typed project models

Requirement IDs: BR-001, FR-001, FR-002, FR-004  
Acceptance Criteria: AC-001, AC-002  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/feed/models/post_test.dart` plus `app/test/projects/models/project_test.dart`

```gherkin
Feature: Project post model parsing
  Scenario: Parse AppView project-post JSON with typed common fields and known details
    Given AppView returns a post JSON object with a camelCase project.common payload
    And the details.$type is social.craftsky.project.knitting#details
    When Flutter parses the JSON through PostMapper.fromMap
    Then Post.project is non-null
    And project.common, project.common.pattern, and gauge-like fields are typed Dart objects
    And project.details is the knitting sealed details variant
    And a general post JSON without project still parses with Post.project equal to null
```

### AT-002: Create plumbing submits top-level project posts through AppView

Requirement IDs: BR-001, BR-002, FR-005, FR-012, RULE-001  
Acceptance Criteria: AC-004, AC-015  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/feed/providers/create_post_provider_test.dart` and `app/test/feed/data/post_api_client_test.dart`

```gherkin
Feature: Create project post plumbing
  Scenario: Submit a common-only project post
    Given a typed Project payload with project.common.craftType social.craftsky.feed.defs#embroidery
    And no project.details payload
    When CreatePost.create submits a top-level post
    Then the request goes to POST /v1/posts through the existing AppView-backed repository
    And the request body contains camelCase project.common JSON
    And no details field is required
    And the project-bearing response parses into Post.project
```

### AT-003: Profile projects are listed from the project endpoint

Requirement IDs: BR-001, FR-007, RULE-002  
Acceptance Criteria: AC-006  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/feed/data/post_api_client_test.dart`, `app/test/feed/data/post_repository_test.dart`, `app/test/projects/providers/user_projects_provider_test.dart`

```gherkin
Feature: Profile project listing
  Scenario: Fetch a profile's project list
    Given a profile handle or DID and optional cursor and limit
    When Flutter requests profile projects through the API client and repository
    Then it calls GET /v1/profiles/@{handleOrDid}/projects
    And it sends cursor and limit using existing string query parameter conventions
    And it parses the AppView PostPage response without client-side filtering
```

### AT-004: Project replies are rejected before submission

Requirement IDs: BR-002, FR-005, FR-006, RULE-001  
Acceptance Criteria: AC-005, AC-011  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/feed/providers/create_post_provider_test.dart`, `app/test/feed/data/post_api_client_test.dart`, `app/test/feed/data/post_repository_test.dart`

```gherkin
Feature: Standalone project post rule
  Scenario: Prevent project-plus-reply payload submission
    Given CreatePost.create receives both a typed Project and a reply reference
    When the provider handles the create request
    Then it transitions to AsyncError
    And it does not call the repository
    When a lower-level API or repository create method is directly called with project and reply
    Then it fails fast rather than sending POST /v1/posts
```

### AT-005: Project creates update live timeline and project caches only

Requirement IDs: BR-002, FR-009, RULE-002  
Acceptance Criteria: AC-008  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/feed/providers/create_post_provider_test.dart` and `app/test/projects/providers/user_projects_provider_test.dart`

```gherkin
Feature: Project create cache updates
  Scenario: Successful project create updates live project-aware caches
    Given live timelineProvider and userProjectsProvider entries exist for the author's DID and handle
    And live userPostsProvider entries also exist for the same author
    When CreatePost.create succeeds with a project post
    Then timelineProvider receives the created post
    And the matching userProjectsProvider entries receive the created post
    And userPostsProvider entries are not prepended with the project post
```

### AT-006: Unknown and sparse project details remain readable

Requirement IDs: FR-002, FR-003, NFR-002  
Acceptance Criteria: AC-003, AC-009  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/projects/models/project_test.dart` and `app/test/feed/models/post_test.dart`

```gherkin
Feature: Forward-compatible project details
  Scenario: Parse sparse or future project detail payloads
    Given project JSON has only common.craftType
    Or project JSON has details without $type
    Or project JSON has details with an unrecognized $type
    When Flutter parses and re-encodes the project post
    Then parsing succeeds
    And unknown details preserve raw fields and discriminator when present
    And Flutter does not infer a known details variant from common.craftType
    And omitted optional fields remain absent or null as appropriate
```

### AT-007: Profile projects provider paginates like existing profile lists

Requirement IDs: FR-008, FR-010, RULE-002, NFR-003  
Acceptance Criteria: AC-007, AC-016  
Priority: Must / Should  
Level: Acceptance  
Automation Target: `app/test/projects/providers/user_projects_provider_test.dart`

```gherkin
Feature: Profile projects provider pagination
  Scenario: Build, load more, retry, and preserve AppView results
    Given userProjectsProvider is watched for alice.craftsky.social
    When the provider builds
    Then it requests projects with userProjectsPageLimit equal to 10
    And it exposes UserProjectsState items, cursor, and hasMore
    When loadMore succeeds
    Then it appends the next page and advances the cursor
    When loadMore fails
    Then previously visible items and cursor remain available for retry
    And if AppView returned an item with project unexpectedly null, the item is still preserved
```

### AT-008: Project interactions update project-list caches

Requirement IDs: FR-009  
Acceptance Criteria: AC-013  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/feed/providers/delete_post_provider_test.dart`, `app/test/feed/providers/toggle_post_interactions_provider_test.dart`, `app/test/projects/providers/user_projects_provider_test.dart`

```gherkin
Feature: Project post mutation cache updates
  Scenario Outline: Mutating a project post updates live project caches consistently
    Given a project post is visible in timelineProvider and userProjectsProvider
    When the user successfully <mutation> the project post through the existing mutation provider
    Then the timeline cache is updated or removed consistently with existing post behavior
    And live userProjectsProvider entries keyed by author handle or DID are updated or removed
    And live userPostsProvider entries are not polluted with project-only updates

    Examples:
      | mutation   |
      | deletes    |
      | likes      |
      | unlikes    |
      | reposts    |
      | unreposts  |
```

### AT-009: CreatePost patches omitted project in synthetic create response

Requirement IDs: FR-011  
Acceptance Criteria: AC-014  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/feed/providers/create_post_provider_test.dart`

```gherkin
Feature: Synthetic project create response resilience
  Scenario: Patch created post state when AppView omits project
    Given CreatePost.create is called with a typed Project payload
    And the repository returns a synthetic Post without project metadata
    When the provider completes successfully
    Then the provider state contains a Post patched with the input project
    And live cache updates use the patched Post
    And PostApiClient tests still prove the API client honestly parses the response it receives
```

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-001, FR-010, NFR-001 | AC-001 | `Project` and `ProjectCommon` parse and serialize camelCase AppView JSON with value semantics. | TD-001 full common project map with title/summary/craftType/pattern/materials/tools/status/dates. | Typed objects expose common fields; `toMap()` emits camelCase keys only; equality/copyWith behave as generated mapper objects. | `app/test/projects/models/project_test.dart` |
| UT-002 | FR-001, NFR-001 | AC-001, AC-002 | `ProjectPattern` parses and round-trips pattern URL/name/source fields. | TD-001 nested pattern map. | Pattern fields are typed and re-encoded without snake_case keys. | `app/test/projects/models/project_test.dart` |
| UT-003 | FR-001, FR-002, NFR-001 | AC-001, AC-002 | Gauge-like sub-objects parse and round-trip stitch/row/unit fields where present. | TD-002 known details maps containing gauge-like data. | Gauge values remain typed and serialize to AppView-compatible camelCase JSON. | `app/test/projects/models/project_test.dart` |
| UT-004 | FR-002 | AC-002 | Knitting details discriminator maps to the knitting sealed variant. | TD-002 details with `$type: social.craftsky.project.knitting#details`. | Parsed details is `KnittingProjectDetails` with craft-specific fields accessible. | `app/test/projects/models/project_details_test.dart` |
| UT-005 | FR-001, FR-002, NFR-001 | AC-002 | Crochet details discriminator maps to the crochet sealed variant and preserves common sub-objects. | TD-002 crochet details map. | Parsed details is `CrochetProjectDetails`; serialization matches input fields. | `app/test/projects/models/project_details_test.dart` |
| UT-006 | FR-001, FR-002 | AC-002 | Sewing and quilting discriminators map to their sealed variants. | TD-002 sewing and quilting detail maps. | Parsed details variants match `$type` exactly; craft-specific nullable fields remain accessible. | `app/test/projects/models/project_details_test.dart` |
| UT-007 | FR-002, FR-003, FR-005, NFR-002 | AC-003, AC-012 | Unknown or missing details discriminator uses `UnknownProjectDetails` without inference from `common.craftType`. | TD-003 unknown `$type`, TD-004 missing `$type`, with `common.craftType` set to a known craft. | Parser does not throw; raw data and discriminator when present are preserved; known craft type does not force a known variant. | `app/test/projects/models/project_details_test.dart` |
| UT-008 | FR-004, BR-001 | AC-001 | `PostMapper.fromMap` exposes optional `project` for project posts. | TD-001 wrapped in a full post map. | `Post.project` is non-null and typed. | `app/test/feed/models/post_test.dart` |
| UT-009 | FR-003, FR-004, NFR-002 | AC-009 | General and sparse project posts parse without Flutter-only defaults. | TD-005 general post without `project`; TD-006 common-only project post. | General post has `project == null`; sparse project has absent/null optional fields and round-trips appropriately. | `app/test/feed/models/post_test.dart` |
| UT-010 | FR-003, NFR-002 | AC-003, AC-009 | Re-encoding unknown/raw details preserves raw fields for pass-through. | TD-003 unknown future details with nested arrays/maps. | `toMap()` includes raw detail fields and `$type` if originally present. | `app/test/projects/models/project_details_test.dart` |
| UT-011 | FR-005, FR-012 | AC-004, AC-012, AC-015 | Create serialization includes typed project only when provided and permits common-only open craft tokens. | Create input with TD-006 embroidery common-only project; create input with `project == null`. | Project create map contains `project.common` and no required `details`; general create map omits `project`. Tests do not promote arbitrary new unknown raw details construction as composer API. | `app/test/projects/models/project_test.dart`, `app/test/feed/data/post_api_client_test.dart` |
| UT-012 | FR-006 | AC-005, AC-011 | Project-plus-reply guard is enforced before repository submission in provider-level input handling. | `CreatePost.create(text, project, reply)` with fake repository tracking calls. | Provider enters `AsyncError`; fake repository callback is not called. | `app/test/feed/providers/create_post_provider_test.dart` |
| UT-013 | FR-008, FR-010, NFR-003 | AC-007 | `UserProjectsState` exposes items/cursor/hasMore with generated value semantics. | Empty page, cursor page, terminal page. | `hasMore` follows cursor presence; `copyWith` and equality behave like existing `UserPostsState`. | `app/test/projects/models/user_projects_state_test.dart` |
| UT-014 | FR-008, NFR-003 | AC-007 | `userProjectsProvider.loadMore` appends pages, preserves prior data on failure, and no-ops when exhausted or loading. | Fake repository with paged results, thrown error, and pending completer. | Behavior mirrors `userPostsProvider` pagination semantics with `userProjectsPageLimit = 10`. | `app/test/projects/providers/user_projects_provider_test.dart` |
| UT-015 | FR-008, RULE-002 | AC-007, AC-016 | Profile projects provider preserves items returned by AppView, including unexpected `project == null`. | PostPage containing one project post and one post with null project. | State items preserve both entries in order; no client-side filtering occurs. | `app/test/projects/providers/user_projects_provider_test.dart` |
| UT-016 | FR-009 | AC-008, AC-013 | User projects cache helper prepends project posts and dedupes by URI. | Live `userProjectsProvider` state containing old item; prepend same/new project. | New project appears at head; duplicate URI is not inserted twice. | `app/test/projects/providers/user_projects_provider_test.dart` |
| UT-017 | FR-009 | AC-013 | User projects cache helper removes by rkey and replaces by URI/rkey. | Live state with multiple project posts. | Delete removes matching rkey; like/repost replacement updates the matching item only. | `app/test/projects/providers/user_projects_provider_test.dart` |
| UT-018 | FR-009 | AC-008, AC-013 | Project-aware mutation helpers update timeline/project caches and avoid profile Posts caches for project posts. | Live timeline, live userProjects, live userPosts, project post. | Timeline and projects cache mutate; profile Posts cache remains unchanged for project-specific cache paths. | `app/test/feed/providers/create_post_provider_test.dart`, `app/test/feed/providers/toggle_post_interactions_provider_test.dart` |
| UT-019 | FR-011 | AC-014 | `CreatePost` patches missing `created.project` with input project after repository response. | Fake repository returns a Post without project despite project input. | Provider state and cache updates use a copy with `project` set to the input; API client behavior remains unpatched. | `app/test/feed/providers/create_post_provider_test.dart` |
| UT-020 | RULE-003 | AC-017 | Project model constructors do not enforce lexicon validation hints. | TD-007 overlong strings, empty/large arrays, unknown craft token, non-URI pattern URL, zero/negative numbers. | Constructing/parsing structurally valid maps does not throw solely due to validation-hint values. | `app/test/projects/models/project_test.dart` |

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | BR-001, BR-002, FR-005, FR-012, RULE-001, NFR-001 | AC-004, AC-015 | `PostApiClient.createPost` sends project JSON to AppView for top-level project creates. | Dio mock for `/v1/posts`; typed Project from TD-001 or TD-006. | Call `createPost(text: ..., project: project)`. | Mock receives `{'text': ..., 'project': project.toMap()}` with camelCase keys; response parses into `Post.project`. | `app/test/feed/data/post_api_client_test.dart` |
| IT-002 | BR-002, FR-005, FR-006, RULE-001 | AC-005, AC-011 | Lower-level create methods fail fast for project-plus-reply. | Dio mock with no matching POST expectation; typed Project and PostReply. | Call API client and repository create directly with both arguments. | Calls throw before HTTP submission; no request is observed. | `app/test/feed/data/post_api_client_test.dart`, `app/test/feed/data/post_repository_test.dart` |
| IT-003 | FR-005, FR-006 | AC-004, AC-011, AC-012 | Repository and fake repository signatures pass through optional project, facets, images, and reply as expected. | `ApiPostRepository` with Dio mock; `FakePostRepository` capturing callback arguments. | Call `PostRepository.create` through interface. | Project argument is forwarded unchanged when valid; invalid project-plus-reply guard is observable; existing facets/images continue passing through. | `app/test/feed/data/post_repository_test.dart`, `app/test/feed/fakes/fake_post_repository.dart` consumer tests |
| IT-004 | BR-001, FR-007, RULE-002 | AC-006, AC-016 | `PostApiClient.listProjectsByAuthor` calls the profile projects endpoint and parses `PostPage`. | Dio mock for `/v1/profiles/@alice.craftsky.social/projects` returning TD-008 page. | Call client with handle and no cursor. | Path matches exactly; returned page items and cursor parse. | `app/test/feed/data/post_api_client_test.dart` |
| IT-005 | FR-007 | AC-006 | Project list cursor and limit use existing query conventions. | Dio mock for projects route with query parameters. | Call client with `cursor: 'c1', limit: 50`. | Query parameters are `{'cursor': 'c1', 'limit': '50'}`; empty page parses with null cursor. | `app/test/feed/data/post_api_client_test.dart` |
| IT-006 | FR-007, FR-008 | AC-006, AC-007 | `ApiPostRepository` and fake repository expose profile projects for provider use. | Fake repository with `onListProjectsByAuthor`; provider container override. | Build `userProjectsProvider('alice.craftsky.social')`. | Repository receives handle and `limit: userProjectsPageLimit`; provider state reflects returned page. | `app/test/feed/data/post_repository_test.dart`, `app/test/projects/providers/user_projects_provider_test.dart` |
| IT-007 | BR-002, FR-009, RULE-002 | AC-008 | Successful project create updates timeline and userProjects live caches only. | Live timeline, live userProjects for DID/handle, live userPosts for DID/handle; fake create returns project post. | Call `CreatePost.create(project: ...)`. | Timeline and projects entries prepend; userPosts entries remain unchanged; non-live family entries are not instantiated. | `app/test/feed/providers/create_post_provider_test.dart` |
| IT-008 | FR-009 | AC-013 | Delete removes project posts from live userProjects and timeline caches without mutating profile Posts caches. | Live cache providers with same project post. | Call `deletePostProvider.delete(post: projectPost)`. | Project removed from userProjects and timeline; profile Posts cache remains unchanged. | `app/test/feed/providers/delete_post_provider_test.dart` |
| IT-009 | FR-009 | AC-013 | Like/repost providers update and roll back live project caches consistently. | Live timeline and userProjects caches; fake repository success and failure branches. | Toggle like/unlike/repost/unrepost on project post. | Optimistic updates patch project caches and timeline; failures roll back; profile Posts cache is not polluted by project-specific updates. | `app/test/feed/providers/toggle_post_interactions_provider_test.dart` |
| IT-010 | FR-011 | AC-014 | Patched project create result is used for provider state and all cache update targets. | Fake create returns Post without project; live timeline and userProjects exist. | Call `CreatePost.create(project: inputProject)`. | Provider result and inserted cache rows have `project == inputProject`; API client tests separately prove no response patching occurs at client layer. | `app/test/feed/providers/create_post_provider_test.dart` |

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Acceptance Criteria | Test |
|---|---|---|---|---|
| REG-001 | General post parsing remains unchanged when `project` is absent. | FR-003, FR-004, NFR-002 | AC-001, AC-009 | Keep/extend `app/test/feed/models/post_test.dart` minimal and fully populated general post round-trip tests; expected map does not gain a `project: null` wire key. |
| REG-002 | General `POST /v1/posts` create bodies remain unchanged without a project. | BR-002, FR-005, RULE-001, NFR-001 | AC-005 | Existing `PostApiClient.createPost` tests for text-only, facets, images, and replies continue to expect the same payloads with no `project` key. |
| REG-003 | Profile Posts route remains `/v1/profiles/@{handleOrDid}/posts` and does not switch to projects. | RULE-002 | AC-006 | Existing `listPostsByAuthor` API client tests remain unchanged; new project tests are separate. |
| REG-004 | General top-level creates still prepend profile Posts caches, while project creates do not. | FR-009, RULE-002 | AC-008, AC-013 | Existing `CreatePost` cache tests for general posts continue to pass; add paired project create tests proving split behavior. |
| REG-005 | Models remain wire/data objects rather than form validators. | RULE-003 | AC-017 | Add constructor/parsing tests with validation-hint-violating values and ensure no regression adds strict lexicon validation to model constructors. |
| REG-006 | Mapper bootstrap remains complete after adding project mappers. | NFR-004 | AC-010 | Existing tests that call `setUpAll(initializeMappers)` continue to parse all new project models; focused mapper tests fail if bootstrap omits project mappers. |
| REG-007 | No new dependencies are introduced for project plumbing. | NFR-004 | AC-010 | Implementation review and `flutter pub get`/lockfile diff should show existing `dart_mappable`, Riverpod, Dio, and test dependencies only. |

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Full AppView project post fixture | Full post map with `project.common` containing `craftType`, title/name-like fields, summary/notes-like optional strings, pattern object, materials/tools arrays, image-independent status/dates, plus standard post author/count fields. Use camelCase wire keys only. | AT-001, UT-001, UT-002, UT-008, IT-001 |
| TD-002 | Known details fixtures | Four maps with `$type` values: `social.craftsky.project.knitting#details`, `social.craftsky.project.crochet#details`, `social.craftsky.project.sewing#details`, `social.craftsky.project.quilting#details`; include representative craft-specific fields and gauge-like sub-objects where supported. | AT-001, UT-003, UT-004, UT-005, UT-006 |
| TD-003 | Unknown future details fixture | Details map with `$type: social.craftsky.project.weaving#details`, scalar fields, nested map, and array fields. | AT-006, UT-007, UT-010 |
| TD-004 | Missing discriminator details fixture | Details map with no `$type`, while `common.craftType` is a known craft such as knitting. | AT-006, UT-007 |
| TD-005 | General post fixture | Existing minimal and fully populated post maps without `project`. | UT-009, REG-001, REG-002 |
| TD-006 | Common-only embroidery project fixture | Project map with `common.craftType: social.craftsky.feed.defs#embroidery` and no `details`. | AT-002, UT-009, UT-011, IT-001 |
| TD-007 | Constructor non-validation fixture | Structurally parseable project maps with overlong strings, many array items, unknown token strings, malformed URI-like pattern fields, zero/negative numeric fields, and mismatched `common.craftType` vs `details.$type`. | UT-020, REG-005 |
| TD-008 | Profile projects page fixture | PostPage map with `items` containing project posts, an optional cursor, and one intentionally unexpected post with `project` omitted for AC-016. | AT-003, AT-007, IT-004, IT-006 |
| TD-009 | Mutation cache fixture | Project post and general post with same author DID/handle but distinct rkeys/URIs for create/delete/like/repost cache assertions. | AT-005, AT-008, UT-016, UT-017, UT-018, IT-007, IT-008, IT-009 |

## 8. Manual Checks

| ID | Requirement IDs | Acceptance Criteria | Check | Steps | Expected Result |
|---|---|---|---|---|---|
| MAN-001 | FR-010 | AC-001, AC-007 | Project concepts are discoverable under `app/lib/projects/...` while existing post API/repository code remains in `feed/data`. | Review implementation file layout after code is written. | New project models/providers are under `app/lib/projects/...`; `PostApiClient`, `PostRepository`, and `ApiPostRepository` remain in `app/lib/feed/data` for post-shaped AppView calls. |
| MAN-002 | NFR-004 | AC-010 | Code generation and dependency consistency. | After implementation, run `cd app && dart run build_runner build --delete-conflicting-outputs`, `cd app && flutter analyze`, and `cd app && flutter test`; inspect `pubspec.yaml`/lockfile diffs. | Generated mapper/provider files are consistent; analysis/tests pass; no new package dependency is added. |

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | No full end-to-end UI test for project creation or profile Projects tab rendering in this slice. | BR-001, FR-008, FR-009 | UI, routes, localization, project composer, and profile Projects tab rendering are explicitly out of scope. | Cover in the later UI/composer workflow once UI exists. |
| GAP-002 | Manual package-layout and dependency review is partial automation. | FR-010, NFR-004 | File organization and dependency intent are best verified by review plus analysis/codegen commands. | Keep MAN-001 and MAN-002 in document review checklist. |
| GAP-003 | Unknown/raw details create authoring is intentionally not advertised as a supported composer path. | FR-005, FR-003 | Requirements allow re-encoding parsed unknown details but do not intentionally expose arbitrary future raw details construction for new creates. | Tests should prove pass-through serialization, not require composer APIs for arbitrary unknown details authoring. |

## 10. Out Of Scope

- UI rendering tests for project cards, profile Projects tab contents, project detail pages, composer project forms, routes, or localization copy.
- AppView route, database, migration, indexer, or lexicon tests; this stage consumes the existing AppView contract only.
- Direct PDS read/write tests; Flutter must continue using AppView JSON/HTTP and Craftsky session behavior.
- Search/filter/discovery providers beyond profile project listing.
- Drafts, private metadata, wishlists, mutes, local persistence, analytics, metrics, and alerts.
- Strict lexicon validation in model constructors or Flutter form validation for field lengths, known token lists, URI format, positive numbers, or array limits.

## 11. Handoff To Document Review

- Requirements file: `docs/changes/2026-06-09-flutter-project-post-plumbing/01-requirements.md`
- Test specification: `docs/changes/2026-06-09-flutter-project-post-plumbing/02-acceptance-tests.md`
- Next review artifact: `03-document-review.md`
- External Plannotator review, if the user initiates it outside this agent: `docs/changes/2026-06-09-flutter-project-post-plumbing/`
- Recommended first failing test for implementation: **UT-001** in `app/test/projects/models/project_test.dart` for parsing/serializing `Project` + `ProjectCommon` camelCase JSON with mapper value semantics.
- Suggested test order for implementation:
  1. UT-001 through UT-010: project models, sealed details, unknown fallback, sparse/general post parsing.
  2. REG-001 and REG-006: ensure existing `Post` parsing and mapper bootstrap remain stable.
  3. IT-001 through IT-003 and UT-011/UT-012: create payload serialization, signatures, project-plus-reply guard.
  4. IT-004 through IT-006 and UT-013 through UT-015: profile projects API/repository/provider pagination.
  5. AT-005, AT-008, UT-016 through UT-019, IT-007 through IT-010: cache updates and synthetic response patching.
  6. REG-002 through REG-005 and REG-007: general behavior and non-validation/dependency regressions.
  7. MAN-001 and MAN-002 after code generation.
- Commands discovered:
  - `cd app && dart run build_runner build --delete-conflicting-outputs`
  - `cd app && flutter analyze`
  - `cd app && flutter test`
  - Focused tests may use `cd app && flutter test test/<path>` if the implementation agent wants tighter TDD loops, adjusting the path to Flutter's expected test path from the `app/` working directory.
- Blocking gaps: None. GAP-001 through GAP-003 are non-blocking and reflect scope boundaries or intentional design constraints.
- Risk-based review recommendation: **Medium risk; document review recommended before implementation, but the user may explicitly skip review.**
