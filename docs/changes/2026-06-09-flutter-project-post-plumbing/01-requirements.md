# Requirements: Flutter Project Post Models And Providers

## 1. Initial Request

The AppView now supports project posts. The next Flutter slice should add everything needed before UI: Dart classes that model project posts and project craft details, plus providers/API/repository plumbing. The user prefers sealed classes for the different project craft details.

## 2. Current Codebase Findings

- Relevant files:
  - Flutter post models: `app/lib/feed/models/post.dart`, `app/lib/feed/models/post_page.dart`, generated `*.mapper.dart` files.
  - Flutter post data layer: `app/lib/feed/data/post_api_client.dart`, `app/lib/feed/data/post_repository.dart`, `app/lib/feed/data/api_post_repository.dart`.
  - Flutter post providers: `app/lib/feed/providers/create_post_provider.dart`, `timeline_provider.dart`, `user_posts_provider.dart`, `user_comments_provider.dart`, `post_repository_provider.dart`, `post_api_client_provider.dart`.
  - Test fakes and provider/API tests: `app/test/feed/fakes/fake_post_repository.dart`, `app/test/feed/data/post_api_client_test.dart`, `app/test/feed/providers/*_test.dart`.
  - Mapper bootstrap: `app/lib/bootstrap.dart`.
  - Profile project tab placeholder/count surface: `app/lib/profile/widgets/profile_tab_bar.dart`, `app/lib/profile/pages/profile_page.dart`, `app/lib/profile/models/profile.dart`.
  - AppView project-post requirements and implementation references: `docs/changes/2026-06-07-appview-project-posts/01-requirements.md`, `appview/internal/api/post_project.go`, `appview/internal/api/post_response.go`, `appview/internal/routes/routes.go`.
  - Project lexicons: `lexicon/social/craftsky/feed/post.json`, `lexicon/social/craftsky/project/defs.json`, `lexicon/social/craftsky/project/{knitting,crochet,quilting,sewing}.json`.
- Existing patterns:
  - Flutter reads Craftsky data from AppView JSON/HTTP and writes via AppView; it does not read from or hold PDS tokens.
  - Domain models use `dart_mappable`; Riverpod providers use `riverpod_annotation` and generated `*.g.dart` files.
  - Existing sealed classes with `dart_mappable` exist in `composer_image_state.dart` using discriminator keys.
  - Cursor-accumulating profile post/comment providers preserve previous data on load-more failures and avoid concurrent pagination calls.
  - `CreatePost` prepends successful top-level creates into live timeline/profile-post caches to bridge AppView read-after-write lag.
- Current behavior:
  - `Post` has no `project` field and cannot parse project-bearing AppView responses.
  - `PostApiClient.createPost`, `PostRepository.create`, `ApiPostRepository.create`, and `CreatePost.create` cannot send a `project` payload.
  - `PostApiClient` has no client method for `GET /v1/profiles/@{handleOrDid}/projects`.
  - There is no `userProjectsProvider`; the profile Projects tab is UI-placeholder-only.
  - `Profile.projectCount` already exists and can parse AppView profile responses.
- Constraints discovered:
  - This slice must not implement UI, routes, localization text, source lexicon changes, AppView changes, database migrations, or dependency changes.
  - AppView project posts are still `social.craftsky.feed.post` records with optional `project`; general post responses omit `project`.
  - AppView treats profile Posts and Projects as separate lists: profile post lists exclude projects, profile project lists include standalone project posts only, timeline/feed still include projects.
  - AppView supports known create-time craft types including embroidery, but only knitting, crochet, sewing, and quilting have current detail lexicons.
  - AppView's Go API model preserves `details` as raw JSON for forward compatibility; Flutter may add typed known variants while preserving unknown details.
- Test/build commands discovered:
  - Flutter tests: `cd app && flutter test`.
  - Flutter analysis: `cd app && flutter analyze`.
  - Code generation after model/provider changes: `cd app && dart run build_runner build --delete-conflicting-outputs`.

## 3. Clarifying Questions And Decisions

### Q1: Should this Flutter slice include project create/write plumbing as well as read/list models/providers?

Answer: Option 2 — read/list + create plumbing.

Decision / implication: Requirements include project models, parsing project-bearing `Post` responses, profile project list API/repository/provider support, and create plumbing through the existing create stack. UI remains out of scope.

### Q2: Which modeling approach should be used for craft-specific project details?

Answer: Option A — typed project model with sealed details variants.

Decision / implication: Requirements use typed `Project` models and a sealed details hierarchy for known craft variants, with an unknown/raw fallback for forward-compatible open-union details.

### Q3: What decisions came out of the requirements grill-me review?

Answer: The user confirmed the following refinements:

- Unknown/raw details are readable and re-encodable when already parsed, but are not an intentional new-create API for constructing future unsupported details.
- `details.$type` is the authoritative sealed-variant discriminator; missing `$type` means unknown/raw details rather than craft-type inference.
- Flutter shall parse AppView responses without rejecting a mismatch between `common.craftType` and `details.$type`.
- Profile project pagination should use a distinct `UserProjectsState` and `userProjectsPageLimit = 10`.
- Project-aware cache helpers should update live project-list caches for create, delete, like, and repost flows.
- New project models/providers should live under `app/lib/projects/...`, while existing post API/repository code remains in `feed/data`.
- `CreatePost` should patch a missing `created.project` with the input project, mirroring the existing reply patch behavior.
- Project models should use `dart_mappable` value semantics, `copyWith`, and `toMap` support.
- Common-only project creates are allowed for any supported/open craft token string, including embroidery.
- Model constructors should not enforce lexicon validation rules such as max lengths, array limits, known values, URI format, or positive integers; AppView/PDS and later composer form validation own those checks. This slice enforces only the cross-field `project` plus `reply` guard.
- `userProjectsProvider` should trust AppView's project endpoint and not client-side filter out posts whose `project` is unexpectedly null.
- Known details discriminator strings are: `social.craftsky.project.knitting#details`, `social.craftsky.project.crochet#details`, `social.craftsky.project.sewing#details`, and `social.craftsky.project.quilting#details`.

Decision / implication: Requirements, acceptance criteria, edge cases, and open questions are updated to make these decisions test-design source of truth.

## 4. Candidate Approaches

### Option A: Typed Project Model With Sealed Details Variants

Summary: Add Dart project model classes for common fields and patterns, then model craft-specific details as a sealed hierarchy with known variants for knitting, crochet, sewing, and quilting plus an unknown/raw fallback.

Pros:
- Matches the user's sealed-class preference.
- Gives later UI and form code type-safe access to craft-specific fields.
- Keeps common project fields shared while allowing craft details to evolve independently.
- Unknown/raw fallback preserves forward compatibility with atproto open unions.

Cons:
- Requires more model classes, mapper setup, and tests than raw JSON.
- Requires careful discriminator handling for lexicon `$type` values.

Risks:
- Known detail parsing could accidentally drop unknown fields or fail on future variants if the fallback is incomplete.

### Option B: Raw Details Map Only

Summary: Model `Project` and `ProjectCommon` but keep `details` as `Map<String, dynamic>?` or raw JSON without typed subclasses.

Pros:
- Smallest implementation footprint.
- Naturally tolerant of future details variants.

Cons:
- Pushes type checks and stringly-typed field access into later UI.
- Does not satisfy the user's preference for sealed craft detail classes.

Risks:
- Later composer/detail UI work may need a second refactor before it can be tested cleanly.

### Option C: One Wide Details Class

Summary: Put all known detail fields across crafts into one nullable-field-heavy model.

Pros:
- Simpler mapper shape than sealed variants.
- Avoids open-union discriminator complexity for known fields.

Cons:
- Loses craft-specific type safety.
- Becomes confusing as future crafts add fields with overlapping names or meanings.
- Encourages UI code to infer craft variant from nullable fields.

Risks:
- High chance of model churn and ambiguous behavior when new details variants are added.

## 5. Recommended Direction

Recommended approach: Option A — typed project model with sealed known craft details and an unknown/raw fallback.

Why: It matches the requested sealed-class direction, aligns with existing Dart sealed-class patterns, gives later UI/test stages a strongly typed contract, and still respects atproto open-union forward compatibility.

## 6. Problem / Opportunity

The AppView can now create and return project posts, but the Flutter app cannot parse, construct, list, or cache project post data. Adding the non-UI model/data/provider layer now lets the next UI slice focus on presentation and composer interactions instead of data contracts.

## 7. Goals

- G-001: Represent AppView project post JSON in typed Flutter models.
- G-002: Preserve type-safe known craft details while tolerating future unknown details variants.
- G-003: Add AppView client/repository/provider support for a profile's project list.
- G-004: Extend create plumbing so future composer UI can submit project posts without changing data-layer contracts again.
- G-005: Keep this slice UI-free and compatible with existing feed/profile behavior.

## 8. Non-Goals

- NG-001: Do not implement project composer UI, profile Projects tab UI, feed card project UI, detail pages, routes, or localization copy.
- NG-002: Do not change files under `lexicon/` or generated AppView lexicon code.
- NG-003: Do not change AppView routes, database schema, migrations, indexing, or API response contracts.
- NG-004: Do not add project search/filter/discovery providers beyond profile project listing.
- NG-005: Do not add drafts, private project metadata, wishlists, mutes, or local persistence.
- NG-006: Do not add dependencies or change Flutter authentication/session behavior.
- NG-007: Do not implement create/edit update flows beyond plumbing optional project data into the existing post create flow.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Craftsky member | Authenticated Flutter app user | Eventually create and view project posts through UI backed by reliable data plumbing. |
| Flutter UI implementer | Later workflow stage building composer/profile/feed UI | Typed project models, providers, and API methods ready to consume. |
| Test designer / implementer | Next workflow stages | Stable requirements and IDs for model, API, repository, provider, and cache behavior tests. |
| AppView API | Existing backend serving project-bearing post responses | Flutter client must match its JSON contract and endpoint conventions. |

## 10. Current Behavior

Flutter can parse and create general posts, list timeline/profile posts/comments, and update live caches after create. It ignores project metadata because `Post` has no `project` field. Profile project counts parse through `Profile.projectCount`, but no provider can fetch profile projects. The profile Projects tab is a placeholder, and create plumbing cannot include a project payload.

## 11. Desired Behavior

Flutter shall model project metadata returned by AppView as an optional `Post.project`. Project common fields and pattern/gauge sub-objects shall be typed. Craft-specific details shall use a sealed details hierarchy with known variants for knitting, crochet, sewing, and quilting, and a raw unknown fallback for unsupported/future variants. The post API/repository shall list profile projects from `GET /v1/profiles/@{handleOrDid}/projects`, and a new cursor-accumulating provider shall expose those project posts without adding UI. Create plumbing shall accept a typed project payload for standalone top-level project posts and serialize it to AppView's existing `POST /v1/posts` request. Successful project creates shall update live timeline and project-list caches without polluting the profile Posts cache.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | Flutter shall support project posts as first-class AppView-backed post data before project UI is implemented. | Enables UI work to build on a stable model/data/provider contract. | Prompt, Q1 | AC-001, AC-004, AC-006 |
| BR-002 | Business | Must | Flutter shall preserve Craftsky's architecture where public project post reads and writes go through the AppView, not direct PDS access. | Maintains project architectural rules and token boundaries. | AGENTS.md, discovery | AC-004, AC-005, AC-008 |
| FR-001 | Functional | Must | The system shall add typed Flutter models for `Project`, `ProjectCommon`, `ProjectPattern`, and project gauge-like detail sub-objects matching AppView camelCase JSON. | Common project fields are shared across all crafts and should not be stringly typed in UI code. | Lexicon findings, AppView contract | AC-001, AC-002 |
| FR-002 | Functional | Must | The system shall model project details as a sealed hierarchy with known variants for knitting, crochet, sewing, and quilting, using `details.$type` as the authoritative discriminator. | Satisfies the confirmed sealed-class approach and supports craft-specific UI later without guessing from craft type. | Q2, Q3, lexicon findings | AC-002, AC-003 |
| FR-003 | Functional | Must | The details model shall preserve unsupported, future, or missing-discriminator details variants in an unknown/raw fallback without throwing away raw fields or the discriminator when possible. | atproto details are an open union; future variants and malformed-but-backend-returned details must not break reading project posts. | Q3, AppView contract, atproto lexicon shape | AC-003, AC-009 |
| FR-004 | Functional | Must | The `Post` model shall expose optional `project` metadata for project posts and continue to parse general posts where `project` is absent. | AppView omits `project` for general posts and includes it for project posts. | AppView implementation, discovery | AC-001, AC-009 |
| FR-005 | Functional | Must | The post create API, repository, fake repository, and `CreatePost` provider shall accept an optional typed `Project` payload and serialize it as `project` only when provided; unknown/raw details may be re-encoded when already present on a parsed project but are not an intentional new-create surface for unsupported future details. | Later composer UI needs write plumbing without revisiting data-layer signatures while avoiding an unsupported raw-details authoring API. | Q1, Q3, existing create stack | AC-004, AC-005, AC-012 |
| FR-006 | Functional | Must | Project create plumbing shall preserve the standalone project rule: `CreatePost` shall surface project-plus-reply input as provider `AsyncError` without calling the repository, and lower-level direct API/repository calls shall fail fast rather than submit the invalid payload. | AppView rejects project replies; Flutter should not silently generate invalid project-post writes. | Q3, AppView requirements, RULE-001 | AC-005, AC-011 |
| FR-007 | Functional | Must | `PostApiClient`, `PostRepository`, and `ApiPostRepository` shall support listing a profile's project posts from `GET /v1/profiles/@{handleOrDid}/projects` using existing cursor/limit conventions. | Flutter needs data-layer access to AppView's profile project list route. | AppView route discovery | AC-006 |
| FR-008 | Functional | Must | The system shall add a cursor-accumulating profile projects provider keyed by `handleOrDid`, with distinct `UserProjectsState` and `userProjectsPageLimit = 10`, modeled after existing profile list providers. | Future profile Projects UI needs provider state with pagination and retry behavior while keeping Posts and Projects semantics explicit. | Q3, existing provider patterns | AC-007 |
| FR-009 | Functional | Must | Project-aware cache helpers shall update live timeline/project caches for successful project creates and update/remove live project-list caches for delete, like, and repost flows involving project posts, while avoiding profile Posts cache pollution. | Timeline includes projects; profile Posts and Projects are separate AppView lists; project interactions should update future Projects UI state. | Q3, AppView split-tab rule, existing cache pattern | AC-008, AC-013 |
| FR-010 | Functional | Should | Project models and providers should live under `app/lib/projects/...`, while existing post-shaped API/repository code may remain under `app/lib/feed/data`, so later UI code can discover project concepts without importing AppView-specific internals. | Keeps future UI implementation straightforward and avoids bloating feed/profile feature areas with craft-specific models. | Q3, existing code organization | AC-001, AC-007 |
| FR-011 | Functional | Must | If `CreatePost` receives a project input but AppView's synthetic create response omits `project`, `CreatePost` shall patch the returned `Post` used for provider state and cache updates with the input project. | Mirrors existing reply create resilience and prevents immediate project creates from appearing as general posts due to response-shape lag. | Q3, existing create stack | AC-014 |
| FR-012 | Functional | Must | Common-only project creates shall be valid for any supported/open craft token string, including `social.craftsky.feed.defs#embroidery`, and shall not require details. | Lexicon details are optional and embroidery currently has no details schema. | Q3, lexicon findings | AC-015 |
| RULE-001 | Business rule | Must | A Flutter project post payload represents a standalone `social.craftsky.feed.post` with optional `project`; it is not a separate collection or direct PDS write. | Aligns Flutter with AppView and lexicon architecture. | AppView project-post requirements | AC-004, AC-005 |
| RULE-002 | Business rule | Must | Profile project lists shall trust and preserve the posts returned by AppView's project endpoint; profile post lists remain separate and must not be client-side mixed with projects by this slice. | Prevents split-tab behavior from drifting in Flutter while avoiding silent client-side filtering of backend contract bugs. | Q3, AppView project-post requirements | AC-006, AC-008, AC-016 |
| RULE-003 | Business rule | Must | Project model constructors shall behave as wire/data objects and shall not enforce lexicon validation rules such as max lengths, array limits, known token values, URI format, or positive numeric constraints in this slice. | AppView/PDS and later composer form validation own create validation; strict constructors could reject readable backend data. | Q3 | AC-017 |
| NFR-001 | Non-functional | Must | New/changed Flutter models shall round-trip AppView `/v1/*` camelCase JSON without introducing snake_case wire keys. | Maintains API contract consistency. | AGENTS.md API casing rule | AC-001, AC-004 |
| NFR-002 | Non-functional | Must | Project model parsing shall be forward-compatible with absent optional fields, absent details, and unknown details variants. | Prevents old clients from failing on sparse or future project records. | Lexicon open-union behavior | AC-003, AC-009 |
| NFR-003 | Non-functional | Should | Provider pagination behavior should match existing user post/comment providers, including preserving previous data on load-more failures and avoiding concurrent load-more calls. | Keeps UI behavior consistent once projects are rendered. | Existing provider pattern | AC-007 |
| NFR-004 | Non-functional | Must | The implementation shall use existing dependencies, code generation tools, mapper initialization patterns, and `dart_mappable` value semantics/copy/toMap support without adding packages. | This is data plumbing, not dependency/platform work, and project models need normal generated model behavior. | Q3, scope boundaries | AC-010 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-001, FR-004, FR-010, NFR-001 | Given AppView project-post JSON with `project.common`, when `PostMapper.fromMap` parses it, then `Post.project` is non-null with typed common/pattern fields; given general post JSON without `project`, then parsing succeeds with `Post.project == null`. |
| AC-002 | FR-001, FR-002 | Given project JSON has `details.$type` equal to `social.craftsky.project.knitting#details`, `social.craftsky.project.crochet#details`, `social.craftsky.project.sewing#details`, or `social.craftsky.project.quilting#details`, when parsed, then each details payload becomes the matching sealed variant with its craft-specific fields accessible in typed Dart code. |
| AC-003 | FR-002, FR-003, NFR-002 | Given project JSON has no details, has details with no `$type`, or has details with an unrecognized `$type`, when parsed, then the post still parses successfully and unknown details preserve raw data/discriminator for later inspection or pass-through without inferring a known variant from `common.craftType`. |
| AC-004 | BR-001, BR-002, FR-005, RULE-001, NFR-001 | Given `CreatePost.create` or `PostApiClient.createPost` receives a valid typed project for a top-level post, when the request is sent, then the `POST /v1/posts` body includes camelCase `project` JSON matching the typed model and the project-bearing response parses into `Post.project`. |
| AC-005 | BR-002, FR-005, FR-006, RULE-001 | Given create plumbing is invoked without a project, when it sends a post, then existing general post payloads remain unchanged; given it is invoked with both `project` and `reply`, then it does not silently submit an invalid project-reply payload. |
| AC-006 | BR-001, FR-007, RULE-002 | Given a handle or DID and optional cursor/limit, when profile projects are requested through the API/repository, then Flutter calls `GET /v1/profiles/@{handleOrDid}/projects`, passes cursor/limit using existing conventions, and parses the returned `PostPage`. |
| AC-007 | FR-008, FR-010, NFR-003 | Given the profile projects provider is watched, when it builds and paginates, then it uses `userProjectsPageLimit = 10`, exposes `UserProjectsState` items/cursor/hasMore state, appends next pages, preserves previous data on load-more errors, and no-ops when exhausted or already loading. |
| AC-008 | BR-002, FR-009, RULE-002 | Given a project post is successfully created through `CreatePost`, when live caches exist, then the timeline cache and matching profile project-list caches keyed by author handle or DID receive the post, while live profile Posts caches do not receive it as a normal post. |
| AC-009 | FR-003, FR-004, NFR-002 | Given sparse project JSON with only required common craft type and optional fields omitted, when parsed and re-encoded, then optional fields remain absent/null as appropriate and parsing does not require Flutter-only defaults. |
| AC-010 | NFR-004 | Given code generation has run, when Flutter analysis/tests are executed, then generated mapper/provider files are consistent and no new dependency is required. |
| AC-011 | FR-006 | Given `CreatePost.create` is called with both `project` and `reply`, when the provider handles the input, then it transitions to `AsyncError` without calling the repository; given a lower-level API/repository method is directly called with both, then it fails fast rather than sending the invalid request. |
| AC-012 | FR-005 | Given a parsed project contains `UnknownProjectDetails`, when it is re-encoded through model serialization, then raw details are preserved; given code constructs a new project for create, then tests do not require or advertise constructing arbitrary unknown raw details as a supported composer/create path. |
| AC-013 | FR-009 | Given a project post is deleted, liked, unliked, reposted, or unreposted through existing mutation providers, when live profile project-list caches exist, then those caches are removed or updated consistently with timeline caches and are not written into profile Posts caches. |
| AC-014 | FR-011 | Given `CreatePost` is called with a project and AppView returns a synthetic post response with `project` omitted, when the provider completes, then the provider state and cache updates use a `Post` patched with the input project while the API client remains an honest parser of the response. |
| AC-015 | FR-012 | Given a common-only project payload with a craft type such as `social.craftsky.feed.defs#embroidery`, when create plumbing serializes it, then `project.common` is sent without requiring `details`. |
| AC-016 | RULE-002 | Given AppView's profile projects endpoint returns an item whose `project` is unexpectedly null, when the API/repository/provider parses the page, then Flutter preserves the returned item rather than silently filtering it out. |
| AC-017 | RULE-003 | Given model constructors receive values that violate lexicon validation hints but are otherwise structurally parseable, when constructing or parsing project models, then constructors do not reject solely due to max length, array count, token known-value, URI format, or positive-number rules. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | General post response omits `project`. | `Post` parsing succeeds and `project` is null. | FR-004 |
| EC-002 | Minimal project has `common.craftType` only. | Project parses with common craft type and null/empty optional fields. | FR-001, NFR-002 |
| EC-003 | Known craft details omit optional fields. | Matching sealed details variant parses with optional fields absent/null. | FR-002, NFR-002 |
| EC-004 | Details has an unknown future `$type` or no `$type`. | Unknown/raw details fallback is used; post parsing does not fail and does not infer a known variant from `common.craftType`. | FR-002, FR-003, NFR-002 |
| EC-005 | AppView returns project with craft type `embroidery` and no known details schema. | Common craft type parses; absent details or unknown details remains tolerated. | FR-001, FR-003 |
| EC-006 | Create is called with both `reply` and `project`. | Flutter does not silently submit an invalid standalone-project payload. | FR-006, RULE-001 |
| EC-007 | Profile projects load-more fails after first page. | Provider exposes an error while preserving previous items/cursor for retry. | FR-008, NFR-003 |
| EC-008 | Profile project list returns duplicate project rows across pages. | Dedupe is not required unless the existing provider pattern already does it; duplicates may be preserved unless a later requirement changes profile list behavior. | FR-008 |
| EC-009 | `common.craftType` and `details.$type` describe different crafts. | Flutter parses without rejecting and exposes the mismatch naturally through typed fields. | FR-002, RULE-003 |
| EC-010 | Optional arrays are empty during create serialization. | Empty optional arrays are omitted from create JSON, while empty arrays returned by AppView still parse normally. | FR-001, NFR-001 |

## 15. Data / Persistence Impact

- New fields:
  - `Post.project` as optional Flutter model data.
  - New Flutter model classes for project common, pattern, gauge/details, and sealed details variants.
  - New `UserProjectsState` for profile project pagination.
- Changed fields:
  - Existing create method signatures across API/repository/provider layers gain an optional project payload.
- Migration required:
  - No database or local-storage migration.
- Backwards compatibility:
  - General post parsing and create payloads remain compatible when `project` is absent.
  - Unknown details variants must remain readable for forward compatibility.
  - Generated `*.mapper.dart` and `*.g.dart` files are expected to change as normal codegen output in the implementation stage.
  - Optional empty arrays should be omitted from create JSON but parsed normally when returned by AppView.

## 16. UI / API / CLI Impact

- UI:
  - No UI implementation in this slice.
  - Existing profile Projects tab may remain a placeholder until a later UI slice.
- API:
  - Flutter client adds support for existing AppView `project` response/request field on `/v1/posts`.
  - Flutter client adds support for existing AppView `GET /v1/profiles/@{handleOrDid}/projects`.
  - No AppView API contract changes are introduced by this slice.
- CLI:
  - No CLI changes.
- Background jobs:
  - No background job changes.

## 17. Security / Privacy / Permissions

- Authentication:
  - New project list calls use the existing authenticated Dio/client stack and device/session headers.
- Authorization:
  - Project create uses existing post create authorization through AppView. Flutter does not handle PDS credentials.
- Sensitive data:
  - Project metadata in this slice is public post data from PDS/AppView. Do not add private project drafts or hidden metadata.
- Abuse cases:
  - Existing moderation metadata parsing on `Post` remains applicable to project posts; this slice does not add new moderation behavior.

## 18. Observability

- Events:
  - No analytics/event instrumentation required.
- Logs:
  - Existing provider logging should remain safe; avoid logging full project payloads if future logs are added.
- Metrics:
  - No metrics required.
- Alerts:
  - No alerts required.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Sealed details mapping may not handle `$type`/unknown variants correctly. | Project posts with future or malformed-but-backend-accepted details could fail to parse. | Require unknown/raw fallback tests and sparse/unknown fixture coverage. |
| RISK-002 | Create cache behavior may prepend projects into profile Posts lists. | Profile tabs/counts appear inconsistent with AppView split-list rule. | Require cache tests for project create vs general create. |
| RISK-003 | Model codegen/bootstrap may omit new mappers. | Runtime parsing failures or test failures. | Require mapper initialization and generated-file consistency checks. |
| RISK-004 | Provider implementation may diverge from existing pagination behavior. | Later UI gets inconsistent loading/error/retry behavior. | Mirror existing provider tests for initial load, loadMore, errors, and concurrency. |
| RISK-005 | Strict model validation could reject backend-readable project data. | Future/open token values or legacy records could crash parsing. | Treat models as wire/data objects and defer validation to AppView/PDS or future composer form state. |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | AppView project response/request JSON remains the shape implemented for `Project` in `appview/internal/api/post_project.go`: `common` plus optional raw `details`. | Flutter model fields or tests would need adjustment. |
| ASM-002 | Known details payloads use `$type` values matching the current lexicon strings for Flutter to choose a sealed variant. | Payloads without `$type` will intentionally become unknown/raw details rather than typed known variants. |
| ASM-003 | Profile project list endpoint uses the same `@{handleOrDid}` convention and `PostPage` envelope as profile post/comment lists. | API client route or parser requirements would need revision. |
| ASM-004 | UI stages will consume provider/model contracts but do not need UI-specific formatting helpers in this slice. | Later UI may request additional derived properties or presentation adapters. |

## 21. Open Questions

- None identified.

## 22. Review Status

Status: Draft after grill-me review
Risk level: Medium
Review recommended: Yes
Reviewer: User grill-me review
Date: 2026-06-10
Notes: This is medium risk because it changes Flutter wire models, create request plumbing, generated code, and live cache behavior for a new post subtype. The user completed a grill-me requirements review and confirmed the refinements recorded in Q3. Additional document review remains recommended before test design, but not required if the user chooses to proceed.

## 23. Handoff To Test Design

- Requirements file: `docs/changes/2026-06-09-flutter-project-post-plumbing/01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover requirement IDs:
  - `BR-001`, `BR-002`
  - `FR-001` through `FR-012`
  - `RULE-001`, `RULE-002`, `RULE-003`
  - `NFR-001`, `NFR-002`, `NFR-004`
- Suggested test levels:
  - Model/unit tests for project common/pattern/gauge/details parsing, serialization, sparse fields, and unknown details fallback.
  - API client tests for create project payloads and profile projects route/query parsing.
  - Repository/fake tests for new method signatures and pass-through behavior.
  - Riverpod provider tests for profile project list pagination and create/delete/like/repost cache updates.
  - Regression tests proving general post parsing/create/profile Posts behavior remains unchanged.
- Blocking open questions: None.
