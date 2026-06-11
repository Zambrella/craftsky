# Coding Plan: Flutter Project Post Models And Providers

## 1. Inputs

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` — Approved with notes, no blocking issues.
- Inspected implementation context:
  - `app/lib/feed/models/post.dart`, `post_page.dart`, generated mapper patterns.
  - `app/lib/feed/data/post_api_client.dart`, `post_repository.dart`, `api_post_repository.dart`.
  - `app/lib/feed/providers/create_post_provider.dart`, `timeline_provider.dart`, `user_posts_provider.dart`, `user_comments_provider.dart`, `delete_post_provider.dart`, `toggle_like_post_provider.dart`, `toggle_repost_post_provider.dart`.
  - `app/lib/feed/providers/composer_image_state.dart` for existing `dart_mappable` sealed-class discriminator usage.
  - `app/lib/bootstrap.dart` for mapper initialization and provider log formatting.
  - `app/test/feed/**` for fake repository, Dio mock, provider-container, and cache-helper test patterns.
  - AppView/lexicon shape references: `appview/internal/api/post_project.go`, `appview/internal/api/post_response.go`, `lexicon/social/craftsky/project/*.json`.

## 2. Implementation Strategy

Implement this as a Flutter-only data/model/provider slice. Add typed project models under `app/lib/projects/...`, wire `Post.project` into the existing post model, extend the AppView-backed post create/list repository surface, and add a Riverpod profile projects provider that mirrors existing cursor-accumulating profile list providers.

The plan preserves existing architecture: Flutter continues to read/write via the AppView JSON/HTTP API and the Craftsky session-backed `Dio`; no PDS tokens, AppView changes, lexicon changes, migrations, dependencies, UI, routes, or localization are introduced.

TDD should start with project model parsing/serialization because `Post.project`, API payloads, provider cache updates, and repository fixtures all depend on the typed model contract.

## 3. Affected Areas

| Area | Existing Pattern | Planned Change | Requirement IDs | Test IDs |
|---|---|---|---|---|
| Project models | No Flutter project package exists; models use `dart_mappable`. | Create `app/lib/projects/models/project.dart` with typed common/pattern/gauge classes and sealed details variants with unknown/raw fallback. | BR-001, FR-001, FR-002, FR-003, FR-010, FR-012, RULE-003, NFR-001, NFR-002, NFR-004 | AT-001, AT-006, UT-001-UT-007, UT-010, UT-020, REG-005, MAN-001 |
| Post wire model | `Post` has no `project`; AppView may return `project`. | Add optional `Project? project` to `Post` with `ignoreNull` serialization. | FR-004, NFR-001, NFR-002 | AT-001, AT-006, UT-008, UT-009, REG-001, REG-006 |
| Mapper/bootstrap | All mappers explicitly initialized in `initializeMappers()`. | Initialize project mappers and include `UserProjectsState`; update provider log formatting for `UserProjectsProvider`. | NFR-004, FR-010 | AC-010, REG-006, MAN-002 |
| Post API client | `createPost`, profile posts/comments, timeline use AppView routes. | Add optional `project` create payload and `listProjectsByAuthor` route; fail fast on project+reply. | BR-002, FR-005, FR-006, FR-007, FR-012, RULE-001, RULE-002 | AT-002, AT-003, AT-004, IT-001, IT-002, IT-004, IT-005, REG-002, REG-003 |
| Repository and fake | `PostRepository` abstracts AppView post operations; tests use `FakePostRepository`. | Extend create signature with `Project?`; add `listProjectsByAuthor`; update fake callbacks/captures. | FR-005, FR-006, FR-007 | IT-003, IT-006, UT-012 |
| Create provider | `CreatePost` prepends top-level posts to timeline/profile-post caches and patches missing reply. | Reject project+reply; pass `project`; patch missing returned project; project creates update timeline/project caches only. | FR-005, FR-006, FR-009, FR-011, FR-012, RULE-001, RULE-002 | AT-002, AT-004, AT-005, AT-009, UT-012, UT-018, UT-019, IT-007, IT-010, REG-004 |
| Profile project provider | `userPostsProvider` and `userCommentsProvider` are cursor-accumulating families. | Create distinct `UserProjectsState`, `userProjectsProvider`, `userProjectsPageLimit = 10`, and cache helpers. | FR-008, FR-009, FR-010, RULE-002, NFR-003 | AT-007, UT-013-UT-017, IT-006, MAN-001 |
| Mutation cache fan-out | Delete/like/repost update timeline/userPosts/userComments caches. | For posts with `project != null`, also update/remove live `userProjectsProvider` entries and avoid project-specific writes to `userPostsProvider`. | FR-009, RULE-002 | AT-008, UT-017, UT-018, IT-008, IT-009, REG-004 |

## 4. Files And Modules

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `app/lib/projects/models/project.dart` | Create | `Project`, `ProjectCommon`, `ProjectPattern`, `ProjectGauge`, sealed `ProjectDetails` variants, discriminator constants, unknown/raw fallback. | FR-001, FR-002, FR-003, FR-012, RULE-003, NFR-001, NFR-002 | UT-001-UT-007, UT-010, UT-020, AT-001, AT-006 |
| `app/lib/projects/models/project.mapper.dart` | Generated | Generated mapping/copy/equality for project models. | NFR-004 | REG-006, MAN-002 |
| `app/lib/projects/models/user_projects_state.dart` | Create | Distinct profile projects pagination state with `items`, `cursor`, `hasMore`, `toString`. | FR-008, FR-010, NFR-003 | UT-013, AT-007 |
| `app/lib/projects/models/user_projects_state.mapper.dart` | Generated | Generated value semantics for provider state. | NFR-004 | UT-013, REG-006 |
| `app/lib/projects/providers/user_projects_provider.dart` | Create | Riverpod family provider and cache helpers for profile projects. | FR-008, FR-009, RULE-002, NFR-003 | AT-007, UT-014-UT-017, IT-006 |
| `app/lib/projects/providers/user_projects_provider.g.dart` | Generated | Riverpod generated provider family. | NFR-004 | MAN-002 |
| `app/lib/feed/models/post.dart` | Change | Import project model and add optional `Project? project`. | FR-004 | UT-008, UT-009, REG-001 |
| `app/lib/feed/models/post.mapper.dart` | Generated | Updated mapper includes `project`. | NFR-004 | REG-006 |
| `app/lib/feed/data/post_api_client.dart` | Change | Accept `Project? project`, include `project.toMap()` only when provided, add profile projects endpoint, guard project+reply. | FR-005, FR-006, FR-007, FR-012, RULE-001 | IT-001, IT-002, IT-004, IT-005, REG-002, REG-003 |
| `app/lib/feed/data/post_repository.dart` | Change | Add `Project? project` to `create`; add `listProjectsByAuthor`. | FR-005, FR-007 | IT-003, IT-006 |
| `app/lib/feed/data/api_post_repository.dart` | Change | Forward project create/list calls and guard invalid project+reply before delegating. | FR-005, FR-006, FR-007 | IT-002, IT-003, IT-006 |
| `app/test/feed/fakes/fake_post_repository.dart` | Change during implementation | Extend callback signatures and add project-list callback for tests. | FR-005, FR-007, FR-008 | UT-012, IT-003, IT-006, provider tests |
| `app/lib/feed/providers/create_post_provider.dart` | Change | Add project input, provider-level project+reply guard, patch omitted response project, update correct live caches. | FR-005, FR-006, FR-009, FR-011, FR-012 | AT-002, AT-004, AT-005, AT-009, UT-012, UT-018, UT-019, IT-007, IT-010 |
| `app/lib/feed/providers/delete_post_provider.dart` | Change | Remove project posts from live project caches on successful delete. | FR-009 | AT-008, IT-008 |
| `app/lib/feed/providers/toggle_like_post_provider.dart` | Change | Patch/rollback project caches for project post likes/unlikes. | FR-009 | AT-008, IT-009 |
| `app/lib/feed/providers/toggle_repost_post_provider.dart` | Change | Patch/rollback project caches for project post reposts/unreposts. | FR-009 | AT-008, IT-009 |
| `app/lib/bootstrap.dart` | Change | Initialize project mappers and summarize `UserProjectsProvider` logs. | NFR-004, FR-010 | REG-006, MAN-001, MAN-002 |
| `app/test/projects/models/project_test.dart` | Create in implementation | Project/common/pattern/sparse/create serialization/non-validation tests. | FR-001, FR-003, FR-012, RULE-003 | UT-001, UT-002, UT-009, UT-011, UT-020 |
| `app/test/projects/models/project_details_test.dart` | Create in implementation | Known details discriminator and unknown/raw fallback tests. | FR-002, FR-003 | UT-003-UT-007, UT-010 |
| `app/test/projects/models/user_projects_state_test.dart` | Create in implementation | State value semantics and `hasMore`. | FR-008 | UT-013 |
| `app/test/projects/providers/user_projects_provider_test.dart` | Create in implementation | Build/loadMore/preserve/no-op/cache helpers/project cache behavior. | FR-008, FR-009, RULE-002, NFR-003 | AT-007, UT-014-UT-017, IT-006 |
| Existing `app/test/feed/**` files | Change in implementation | Extend model/API/repository/create/delete/toggle regression coverage. | See requirements | AT-001-AT-005, AT-008-AT-009, IT-001-IT-010, REG-001-REG-004 |

Generated files are implementation-stage output after `cd app && dart run build_runner build --delete-conflicting-outputs`; do not hand-edit generated files.

## 5. Services, Interfaces, And Data Flow

### Project model shape

Model only fields currently present in the AppView/lexicon contract. Do not add Flutter-only validation or fields not returned/accepted by AppView. In particular, `ProjectCommon` should follow `appview/internal/api/post_project.go` and `lexicon/social/craftsky/project/defs.json`: `craftType`, `status`, `title`, `duration`, `pattern`, `materials`, `colors`, `designTags`, and `tags`.

```text
// Partial signatures only.
@MappableClass(ignoreNull: true)
class Project { Project({required ProjectCommon common, ProjectDetails? details}); }

@MappableClass(ignoreNull: true)
class ProjectCommon {
  ProjectCommon({
    required String craftType,
    String? status,
    String? title,
    String? duration,
    ProjectPattern? pattern,
    List<String>? materials,
    List<String>? colors,
    List<String>? designTags,
    List<String>? tags,
  });
}

@MappableClass(ignoreNull: true)
class ProjectPattern { ... url/name/difficulty/designer/publisher ... }

@MappableClass(ignoreNull: true)
class ProjectGauge { ... stitches/rows/measurement/unit ... }
```

Guardrail for `EC-010`: empty optional arrays should be omitted from create JSON while empty arrays returned by AppView still parse. Since `ignoreNull` does not remove empty lists, implement a small model-level encode hook or create-payload helper that removes empty optional arrays only on serialization. Tests should drive this narrowly for create serialization and round-trip expectations.

### Sealed project details

Use `details.$type` as the sole discriminator. Do not infer a known details class from `ProjectCommon.craftType` and do not reject craft-type/detail mismatches.

```text
const knittingDetailsType = 'social.craftsky.project.knitting#details';
const crochetDetailsType = 'social.craftsky.project.crochet#details';
const sewingDetailsType = 'social.craftsky.project.sewing#details';
const quiltingDetailsType = 'social.craftsky.project.quilting#details';

@MappableClass(discriminatorKey: r'$type', ignoreNull: true)
sealed class ProjectDetails { const ProjectDetails(); }

@MappableClass(discriminatorValue: knittingDetailsType, ignoreNull: true)
final class KnittingProjectDetails extends ProjectDetails {
  // projectType, projectSubtype, yarnWeight, needleSizeMm, gauge, finishedSize
}

@MappableClass(discriminatorValue: crochetDetailsType, ignoreNull: true)
final class CrochetProjectDetails extends ProjectDetails {
  // projectType, projectSubtype, yarnWeight, hookSizeMm, gauge, finishedSize
}

@MappableClass(discriminatorValue: sewingDetailsType, ignoreNull: true)
final class SewingProjectDetails extends ProjectDetails {
  // projectType, projectSubtype, sizeMade, fitNotes
}

@MappableClass(discriminatorValue: quiltingDetailsType, ignoreNull: true)
final class QuiltingProjectDetails extends ProjectDetails {
  // projectType, projectSubtype, size, piecingTechnique, quiltingMethod
}

@MappableClass(
  discriminatorValue: MappableClass.useAsDefault,
  ignoreNull: true,
  hook: UnmappedPropertiesHook('raw'),
)
final class UnknownProjectDetails extends ProjectDetails {
  UnknownProjectDetails({@MappableField(key: r'$type') this.type, required this.raw});
  final String? type;
  final Map<String, dynamic> raw;
}
```

If the generated default-subclass mapper cannot preserve raw details exactly, keep the public model shape above but add the smallest local mapping hook/helper needed to satisfy `UT-007` and `UT-010`. Do not broaden this into a new create-authoring API for arbitrary future details; it exists for read/pass-through preservation.

### Post create/list service flow

```text
CreatePost.create(text, project?, reply?, images?, facets?)
  -> guard project + reply before repository call
  -> PostRepository.create(..., project: project)
  -> ApiPostRepository.create(..., project: project)
  -> PostApiClient.createPost(..., project: project)
  -> POST /v1/posts { text, project?: project.toMap(), reply?, images?, facets? }
  -> PostMapper.fromMap(response)
  -> provider patches missing created.project only when input project != null
  -> cache fan-out

userProjectsProvider(handleOrDid)
  -> PostRepository.listProjectsByAuthor(handleOrDid, limit: 10, cursor?)
  -> ApiPostRepository.listProjectsByAuthor(...)
  -> PostApiClient.listProjectsByAuthor(...)
  -> GET /v1/profiles/@{handleOrDid}/projects?cursor=&limit=
  -> PostPageMapper.fromMap(response)
```

### Invalid project-plus-reply guard

Add a small shared helper or consistently duplicated guard in the create layers touched by tests:

```text
void assertProjectCreateIsTopLevel({Project? project, PostReply? reply}) {
  if (project != null && reply != null) {
    throw ArgumentError('Project posts cannot be replies');
  }
}
```

Expected placement:

- `CreatePost.create`: throws inside `AsyncValue.guard` before reading/calling the repository, producing `AsyncError` and zero fake-repository calls.
- `ApiPostRepository.create`: fail fast for direct repository calls.
- `PostApiClient.createPost`: fail fast for direct API client calls before `_dio.post`.

## 6. State, Providers, Controllers, Or DI

### Provider graph

```text
dioProvider
  -> postApiClientProvider
    -> postRepositoryProvider
      -> timelineProvider
      -> userPostsProvider(handleOrDid)
      -> userCommentsProvider(handleOrDid)
      -> userProjectsProvider(handleOrDid)    // new
      -> createPostProvider
      -> deletePostProvider
      -> toggleLikePostProvider
      -> toggleRepostPostProvider
```

### `userProjectsProvider`

Create `app/lib/projects/providers/user_projects_provider.dart` using the existing `@riverpod class UserPosts` pattern:

```text
const userProjectsPageLimit = 10;

@riverpod
class UserProjects extends _$UserProjects {
  static String formatLogValue(Object? value) => value.toString();

  Future<UserProjectsState> build(String handleOrDid) async {
    final repo = ref.watch(postRepositoryProvider);
    final page = await repo.listProjectsByAuthor(handleOrDid, limit: userProjectsPageLimit);
    return UserProjectsState(items: page.items, cursor: page.cursor);
  }

  Future<void> loadMore();       // mirror userPostsProvider copyWithPrevious behavior
  void prepend(Post post);        // no-op without data, dedupe by uri
  void removeByRkey(String rkey); // no-op without data or missing item
  void replace(Post post);        // match by uri or rkey
}
```

Provider requirements split per `DR-002`:

- Must: distinct `UserProjectsState`, endpoint preservation, `userProjectsPageLimit = 10`, append/cursor/hasMore state, no client-side filtering of `project == null` AppView rows (`FR-008`, `RULE-002`).
- Should but expected: pagination parity with existing profile providers, including preserving prior data on load-more failure and avoiding concurrent calls (`NFR-003`).

### Project cache helpers

Define helpers in `user_projects_provider.dart` and import them into feed mutation providers:

```text
void prependLiveUserProjectCaches(Ref ref, Post post) {
  if (post.project == null) return;
  for (final id in {post.author.did, post.author.handle}) {
    if (ref.exists(userProjectsProvider(id))) {
      ref.read(userProjectsProvider(id).notifier).prepend(post);
    }
  }
}

void updateLiveUserProjectCaches(Ref ref, Post post) { ... replace ... }
void removeFromLiveUserProjectCaches(Ref ref, Post post) { ... removeByRkey ... }
```

Use `ref.exists` before `ref.read` to avoid instantiating non-live family entries, matching existing cache guardrails.

## 7. UI, Widgets, Routes, Or User-Facing Surfaces

None identified for this slice.

- Do not implement project cards, profile Projects tab contents, composer UI, project detail pages, route changes, or localization strings.
- Existing profile tab placeholders/counts may remain unchanged.
- Public provider/model contracts should be ready for a later UI slice to consume.

## 8. Error, Loading, Empty, And Edge States

| State / Case | Planned Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| General post lacks `project` | `PostMapper.fromMap` succeeds with `project == null`; `toMap()` omits `project`. | FR-004, NFR-002 | UT-009, REG-001 |
| Minimal project has only `common.craftType` | Parse and serialize without defaults or details requirement. | FR-001, FR-012, RULE-003 | UT-001, UT-009, UT-011, IT-001 |
| Known details `$type` | Decode to matching sealed class; encoder emits the same `$type`. | FR-002 | UT-004-UT-006 |
| Missing/unknown details `$type` | Decode to `UnknownProjectDetails`, preserve raw fields and discriminator when present, no craft-type inference. | FR-003, NFR-002 | AT-006, UT-007, UT-010 |
| `common.craftType`/`details.$type` mismatch | Parse without rejection; expose mismatch naturally through fields. | RULE-003 | UT-020, REG-005 |
| Project create with reply | Provider transitions to `AsyncError` without repository call; API/repository direct calls throw before HTTP. | FR-006, RULE-001 | AT-004, UT-012, IT-002 |
| AppView create response omits input project | `CreatePost` patches returned `Post` for state/cache only; API client remains honest parser. | FR-011 | AT-009, UT-019, IT-010 |
| Profile projects first page empty | `UserProjectsState(items: [], cursor: null)`, `hasMore == false`. | FR-008 | UT-014 |
| Profile projects load-more failure | `AsyncError` preserves previous state/cursor using `copyWithPrevious`; retry reuses cursor. | NFR-003 | AT-007, UT-014 |
| Profile projects already loading or exhausted | `loadMore` no-ops. | NFR-003 | UT-014 |
| Project endpoint returns item with `project == null` | Provider preserves item; no client-side filtering. | RULE-002 | AT-007, UT-015, IT-004 |
| Project create cache fan-out | Timeline and live project caches receive post; profile Posts caches do not. | FR-009, RULE-002 | AT-005, UT-018, IT-007, REG-004 |
| Project delete/like/repost mutations | Live project caches update/remove/rollback alongside timeline; profile Posts caches not polluted by project-specific paths. | FR-009 | AT-008, UT-017, UT-018, IT-008, IT-009 |

## 9. Test Implementation Plan

| Order | Test ID | Target | Setup / Fixture | Initial Expected Failure |
|---|---|---|---|---|
| 1 | UT-001 | `app/test/projects/models/project_test.dart` | TD-001 Project/common map matching AppView camelCase fields. | Import/model missing. |
| 2 | UT-002, UT-003 | `project_test.dart` / `project_details_test.dart` | Pattern and gauge fixtures. | `ProjectPattern`/`ProjectGauge` missing or not mapped. |
| 3 | UT-004-UT-006 | `app/test/projects/models/project_details_test.dart` | Four known details maps with exact `$type` constants. | Details sealed mapper missing/wrong discriminator. |
| 4 | UT-007, UT-010 | `project_details_test.dart` | Unknown future and missing `$type` raw maps. | Unknown fallback throws or drops raw fields. |
| 5 | UT-008, UT-009, REG-001 | `app/test/feed/models/post_test.dart` | Project-bearing and general/sparse post maps. | `Post.project` missing or mapper bootstrap incomplete. |
| 6 | REG-006 | Existing mapper/bootstrap tests plus model tests | `setUpAll(initializeMappers)`. | Project mappers not initialized. |
| 7 | UT-011, IT-001, REG-002 | `app/test/feed/data/post_api_client_test.dart`, `project_test.dart` | Common-only embroidery project and no-project create fixtures. | `createPost` lacks `project` arg/body. |
| 8 | AT-004, UT-012, IT-002 | `create_post_provider_test.dart`, `post_api_client_test.dart`, `post_repository_test.dart` | Project + `PostReply`; Dio mock with no POST expected. | Calls repo/API or sends invalid HTTP. |
| 9 | IT-003 | `app/test/feed/data/post_repository_test.dart`, fake consumers | Fake/API repository capture project/facets/images/reply. | Interface/fake signatures mismatch. |
| 10 | IT-004, IT-005, REG-003 | `post_api_client_test.dart` | Dio mock for `/v1/profiles/@alice.craftsky.social/projects` with cursor/limit. | Method/route missing. |
| 11 | UT-013 | `app/test/projects/models/user_projects_state_test.dart` | Empty/cursor/terminal states. | State class missing. |
| 12 | AT-007, UT-014, UT-015, IT-006 | `app/test/projects/providers/user_projects_provider_test.dart` | Fake repository paged results, failures, unexpected null-project item. | Provider missing or pagination behavior diverges. |
| 13 | UT-016, UT-017 | `user_projects_provider_test.dart` | Live provider state with duplicate/update/remove fixtures. | Cache helpers missing. |
| 14 | AT-005, AT-009, UT-018, UT-019, IT-007, IT-010, REG-004 | `create_post_provider_test.dart`, `user_projects_provider_test.dart` | Live timeline/userProjects/userPosts caches; project create; omitted response project. | Wrong cache fan-out or no response patch. |
| 15 | AT-008, IT-008, IT-009 | `delete_post_provider_test.dart`, `toggle_post_interactions_provider_test.dart` | Project post visible in timeline/userProjects/userPosts; success/failure mutation branches. | Project caches not updated/rolled back. |
| 16 | UT-020, REG-005 | `project_test.dart` | Validation-hint-violating but structurally parseable maps. | Constructors/parsers enforce lexicon hints. |
| 17 | MAN-001, MAN-002, REG-007 | Manual review plus commands | Run build_runner/analyze/test; inspect dependency diffs. | Generated files stale or dependency changes appear. |

Focused commands for implementation loops from repo root:

```text
cd app && flutter test test/projects/models/project_test.dart
cd app && flutter test test/projects/models/project_details_test.dart
cd app && flutter test test/feed/data/post_api_client_test.dart
cd app && flutter test test/projects/providers/user_projects_provider_test.dart
cd app && dart run build_runner build --delete-conflicting-outputs
cd app && flutter analyze
cd app && flutter test
```

## 10. Sequencing And Guardrails

- First TDD step: add `UT-001` in `app/test/projects/models/project_test.dart` for parsing/serializing `Project` + `ProjectCommon` camelCase JSON with `dart_mappable` value semantics.
- Dependencies between work items:
  1. Project models and mapper bootstrap before `Post.project` tests.
  2. `Post.project` before API/repository project-bearing response tests.
  3. Repository `listProjectsByAuthor` before `userProjectsProvider`.
  4. `userProjectsProvider` cache helpers before create/delete/like/repost cache fan-out.
  5. Build runner after annotated model/provider changes and before final analyze/test.
- Guardrails:
  - Keep all new project concepts under `app/lib/projects/...` except post-shaped AppView API/repository and existing feed mutation providers.
  - Do not add dependencies; use existing `dart_mappable`, Riverpod, Dio, and test packages.
  - Do not edit `lexicon/`, AppView Go code, migrations, routes, UI widgets, routes, or localization.
  - Do not enforce lexicon max lengths, known values, URI formats, positive integers, or array limits in constructors/parsers.
  - Do not client-side filter profile project endpoint rows where `post.project == null`.
  - Do not infer details variant from `common.craftType`; `$type` is authoritative.
  - Do not advertise arbitrary future `UnknownProjectDetails` construction as a composer/create API; unknown details support is for parsing/re-encoding existing data.
  - Use `ref.exists` before touching provider-family cache entries so mutations do not instantiate non-live caches.
  - For project creates, never prepend to `userPostsProvider`; for general top-level creates, preserve current userPosts behavior.
- Out of scope:
  - Project composer UI, profile Projects tab rendering, feed cards, details pages, search/discovery providers, routes, localization.
  - AppView/backend/lexicon/database changes.
  - Local persistence/drafts/private project metadata/analytics/metrics.

## 11. Risks And Open Questions

| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Non-blocking risk | `dart_mappable` default subclass plus raw-map preservation needs careful hook usage for `UnknownProjectDetails`. | Unknown future details could parse but lose raw fields. | Drive with `UT-007`/`UT-010`; use `UnmappedPropertiesHook('raw')` or a minimal local hook/helper if generated code falls short. |
| CPQ-002 | Non-blocking risk | Empty optional arrays must be omitted from create JSON while returned empty arrays should parse. | A single model `toMap()` may include empty arrays unless adjusted. | Add focused serialization tests; implement a narrow encode hook/helper without changing parse behavior. |
| CPQ-003 | Non-blocking risk | Cache fan-out crosses feed and projects packages. | Import cycles or accidental profile Posts pollution. | Keep project models dependency-free from feed; project provider may import feed models/repository; feed mutation providers import project cache helpers only. |
| CPQ-004 | Non-blocking risk | Provider pagination Must/Should distinction from `DR-002` can blur. | TDD may over-focus on parity details and miss mandatory AppView preservation. | Keep `FR-008`/`RULE-002` assertions explicit; parity remains expected via `NFR-003` tests. |
| CPQ-005 | Non-blocking clarification | Acceptance test fixture text mentions “tools”, but current AppView/lexicon `ProjectCommon` does not expose a `tools` field. | Adding `tools` in Flutter would diverge from the approved AppView contract. | Model only current AppView/lexicon fields unless requirements are revised. |

No blocking open questions are identified.

## 12. Handoff To TDD Builder

- Coding plan: `docs/changes/2026-06-09-flutter-project-post-plumbing/04-coding-plan.md`
- TDD execution plan: `05-implementation-plan.md`
- Start with test: `UT-001` in `app/test/projects/models/project_test.dart`.
- Focused command: `cd app && flutter test test/projects/models/project_test.dart`
- Source of truth: `01-requirements.md`, `02-acceptance-tests.md`, `03-document-review.md`, and this coding plan.
- Notes:
  - Follow the test order from §9 unless a smaller red-green step is needed within the same dependency layer.
  - Run `cd app && dart run build_runner build --delete-conflicting-outputs` after annotated models/providers are changed.
  - Final implementation handoff should include `cd app && flutter analyze` and `cd app && flutter test` results.
