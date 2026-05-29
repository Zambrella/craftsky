# Coding Plan: Timeline Feed Flutter

## 1. Inputs

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md`
- Document-review verdict: Approved with notes
- Blocking issues: None

## 2. Implementation Strategy

Implement the Flutter home timeline by extending the existing post stack rather than creating a new feed framework. The AppView endpoint already returns the existing `{items, cursor}` post-shaped page, so the client should add one timeline read method at the API/repository layer, introduce a focused Riverpod timeline provider/state, and replace the placeholder `FeedPage` with a paginated `PostCard` timeline.

The plan deliberately mirrors the existing profile posts pattern (`UserPosts`, `ProfilePostsTab`) but keeps the new timeline provider distinct because the home timeline is not keyed by handle/DID and must support global timeline cache updates. Shared mutation providers (`CreatePost`, `ToggleLikePost`, `ToggleRepostPost`, `DeletePost`) will update the live timeline provider through small helper functions, preserving existing profile/comment cache behavior and avoiding a generic normalized cache in this slice.

Key design choices:

- Use existing `Post` and `PostPage` wire models; do not introduce a feed-item response model. (`NFR-002`, `BR-002`)
- Create an in-memory `TimelineState` for provider state, similar to `UserPostsState`. (`FR-003`)
- Use `timelinePageLimit = 20` to match the AppView default documented for this endpoint. (`FR-004`)
- Treat cursor as an opaque string and pass it through untouched. (`FR-005`, `RULE-002`)
- Dedupe timeline rows by `Post.uri`, especially across optimistic create and later server pages. (`FR-015`, `RULE-004`)
- For timeline mutation cache helpers, prefer URI-level matching over rkey-only matching because a home timeline can contain posts from multiple authors whose record keys may collide. (`FR-013`, `RULE-004`)

## 3. Affected Areas

| Area | Existing Pattern | Planned Change | Requirement IDs | Test IDs |
|---|---|---|---|---|
| API client | `PostApiClient` methods call `/v1/*` through shared `Dio`, wrap in `unwrapApi`, parse mappable models. | Add `listTimeline({String? cursor, int? limit})` calling `GET /v1/feed/timeline` and parsing `PostPage`. | FR-001, NFR-001, NFR-002, RULE-001, RULE-002 | IT-001, IT-002, IT-003 |
| Repository | `PostRepository` abstracts AppView post methods; `ApiPostRepository` delegates to `PostApiClient`; `FakePostRepository` supports tests. | Add `listTimeline({String? cursor, int? limit})` to interface, production implementation, and fake callback. Avoid extra seam solely for `IT-004`. | FR-002, RULE-001 | IT-004, UT-001 |
| Timeline state/provider | Profile lists use `UserPostsState` and `UserPosts` `@riverpod` async notifier with cursor accumulation. | Create `TimelineState` and `Timeline` provider with `build`, `loadMore`, `prepend`, `replace`, `removeByUri`, and deduped page merge. | FR-003, FR-004, FR-005, FR-007, FR-015, RULE-002, RULE-004 | UT-002, UT-003, UT-004, UT-005, UT-007 |
| Mutation cache updates | Create/delete/like/repost update live profile/comment caches directly via helper functions. | Add timeline helper functions and call them from create, delete, like, repost, and timeline reply flow while preserving existing caches. | FR-011, FR-012, FR-013, FR-015, RULE-003, RULE-004 | UT-006, UT-008, UT-009, UT-010, AT-008, AT-009, AT-010, AT-012 |
| Feed UI | `FeedPage` is a placeholder; profile tabs render paginated `PostCard` slivers. | Replace Feed body with timeline loading/error/empty/loaded UI, top compose entry, post rows, load-more retry, thread navigation, interactions, and own-post delete. | BR-001, FR-006, FR-008, FR-009, FR-010, FR-012, FR-013, FR-014, NFR-003, NFR-004 | AT-001 through AT-012, MAN-001 |
| Localization/generated code | ARB strings and generated localizations; Riverpod/dart_mappable generated outputs. | Add feed-specific empty/error strings and regenerate localizations/providers/mappers as needed. | NFR-004, NFR-005 | AC-020, AC-021, REG-006 |
| Regression suites | Existing feed/profile tests protect post cards, composer, interactions, routes, auth/device interceptors. | Run focused regression suites after shared provider/UI changes. | BR-002, FR-009, FR-011, FR-012, FR-014, NFR-001, NFR-002 | REG-001 through REG-008 |

## 4. Files And Modules

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `app/lib/feed/data/post_api_client.dart` | Change | Add `listTimeline` method for `GET /v1/feed/timeline`. | FR-001, NFR-001, NFR-002, RULE-002 | IT-001, IT-002, IT-003 |
| `app/lib/feed/data/post_repository.dart` | Change | Add repository method `listTimeline({String? cursor, int? limit})`. | FR-002, RULE-001 | IT-004, UT-001 |
| `app/lib/feed/data/api_post_repository.dart` | Change | Delegate `listTimeline` to `PostApiClient.listTimeline`. | FR-002 | IT-004 |
| `app/lib/feed/models/timeline_state.dart` | Create | In-memory cursor-accumulating timeline state with `items`, `cursor`, `hasMore`, and generated/implemented `copyWith`. | FR-003, FR-007, NFR-005 | UT-002, UT-003, UT-004, UT-005 |
| `app/lib/feed/models/timeline_state.mapper.dart` | Create generated | `dart_mappable` generated file if `TimelineState` follows existing `UserPostsState` pattern. | NFR-005 | REG-006 |
| `app/lib/feed/providers/timeline_provider.dart` | Create | Riverpod `Timeline` async notifier and live timeline cache helper functions. | FR-003, FR-004, FR-005, FR-007, FR-011, FR-012, FR-013, FR-015, RULE-002, RULE-003, RULE-004 | UT-002 through UT-010 |
| `app/lib/feed/providers/timeline_provider.g.dart` | Create generated | Riverpod generated provider output. | NFR-005 | REG-006 |
| `app/lib/feed/providers/create_post_provider.dart` | Change | On top-level create, also prepend into live timeline if it exists; keep reply path comment-only. | FR-015, RULE-003, RULE-004 | UT-006, AT-012 |
| `app/lib/feed/providers/toggle_like_post_provider.dart` | Change | Update and rollback live timeline entries alongside profile/comment caches. | FR-011 | UT-008, AT-008, REG-004 |
| `app/lib/feed/providers/toggle_repost_post_provider.dart` | Change | Update and rollback live timeline entries alongside profile/comment caches. | FR-011 | UT-008, AT-008, REG-004 |
| `app/lib/feed/providers/delete_post_provider.dart` | Change | Remove deleted post from live timeline state by URI after successful delete. | FR-013 | UT-010, AT-010 |
| `app/lib/feed/pages/feed_page.dart` | Change | Replace placeholder with timeline scaffold/body/slivers and user interactions. | BR-001, FR-006, FR-008, FR-009, FR-010, FR-012, FR-013, FR-014, NFR-003, NFR-004 | AT-001 through AT-012 |
| `app/lib/l10n/app_en.arb` | Change | Add feed-specific empty/error strings; reuse `retryButton`, `postComposeAction`, and post delete labels. | FR-006, FR-008, NFR-004 | AT-003, AT-004, REG-006 |
| `app/lib/l10n/generated/app_localizations*.dart` | Change generated | Generated localization accessors for new feed strings. | NFR-004, NFR-005 | REG-006 |
| `app/test/feed/data/post_api_client_test.dart` | Change | Add timeline API tests. | FR-001, FR-005, NFR-001, NFR-002, RULE-002 | IT-001, IT-002, IT-003 |
| `app/test/feed/fakes/fake_post_repository.dart` | Change | Add `onListTimeline` callback and implementation. | FR-002 | UT-001, AT tests |
| `app/test/feed/providers/timeline_provider_test.dart` | Create | Provider/state tests for first load, pagination, errors, guards, dedupe, mutation helpers. | FR-003, FR-004, FR-005, FR-007, FR-011, FR-012, FR-013, FR-015, RULE-002, RULE-003, RULE-004 | UT-002 through UT-010 |
| `app/test/feed/providers/create_post_provider_test.dart` | Change | Add assertions that top-level create updates live timeline and reply create does not. | FR-015, RULE-003 | UT-006 |
| `app/test/feed/providers/toggle_post_interactions_provider_test.dart` | Change | Add assertions that like/repost update and rollback live timeline entries. | FR-011 | UT-008, REG-004 |
| `app/test/feed/feed_page_test.dart` | Change | Replace placeholder-title test with FeedPage acceptance/widget tests. | BR-001, FR-006, FR-008, FR-009, FR-010, FR-012, FR-013, FR-014, FR-015, NFR-003, NFR-004 | AT-001 through AT-012 |

## 5. Services, Interfaces, And Data Flow

### API and repository surface

Add a timeline method to the existing post client/repository stack. Keep the contract post-shaped; no new DTOs beyond the provider state model.

```text
// app/lib/feed/data/post_api_client.dart
Future<PostPage> listTimeline({
  String? cursor,
  int? limit,
}) => unwrapApi(() async {
  GET /v1/feed/timeline
  queryParameters: {
    'cursor': ?cursor,
    'limit': ?limit?.toString(),
  }
  return PostPageMapper.fromMap(response.data!)
})

// app/lib/feed/data/post_repository.dart
Future<PostPage> listTimeline({String? cursor, int? limit});

// app/lib/feed/data/api_post_repository.dart
Future<PostPage> listTimeline({String? cursor, int? limit}) =>
  _api.listTimeline(cursor: cursor, limit: limit);
```

Traceability: `FR-001`, `FR-002`, `NFR-001`, `NFR-002`, `RULE-001`, `RULE-002`; `IT-001` through `IT-004`, `UT-001`.

### Data flow

```text
FeedPage
  watches timelineProvider
    Timeline.build()
      reads postRepositoryProvider
        ApiPostRepository.listTimeline()
          PostApiClient.listTimeline()
            GET /v1/feed/timeline via shared Dio
            PostPageMapper.fromMap(response)
      TimelineState(items: page.items, cursor: page.cursor)

FeedPage near-end scroll / bottom retry
  Timeline.loadMore()
    uses current.cursor exactly as returned by AppView
    appends deduped page.items
    replaces cursor with page.cursor

CreatePost(top-level success)
  existing userPostsProvider cache prepend
  new prependLiveTimelineCache(ref, post)

ToggleLikePost / ToggleRepostPost
  existing profile/comment cache update
  new updateLiveTimelineCache(ref, next)
  rollback timeline on repository failure

DeletePost(success)
  existing profile cache removal
  new removeFromLiveTimelineCache(ref, post.uri)
```

### DR-001 handling

Do not add a `PostApiClient` abstraction solely to unit-test `ApiPostRepository`. `IT-004` can be satisfied by adding the repository method to the interface, production class, and fake, then exercising it through `timelineProvider`/fake tests and compile-time implementation checks. If the builder finds a very small existing-friendly seam, a direct `ApiPostRepository` delegation test is acceptable, but it is not required to introduce new architecture.

## 6. State, Providers, Controllers, Or DI

### New state model

Create a timeline-specific in-memory state class. Prefer matching `UserPostsState` style with `dart_mappable` unless the builder chooses a simple hand-written `copyWith`; if using mappable, include generated output.

```text
// app/lib/feed/models/timeline_state.dart
@MappableClass()
class TimelineState with TimelineStateMappable {
  const TimelineState({required this.items, this.cursor});

  final List<Post> items;
  final String? cursor;

  bool get hasMore => cursor != null;
}
```

### New timeline provider

```text
// app/lib/feed/providers/timeline_provider.dart
const timelinePageLimit = 20;

@riverpod
class Timeline extends _$Timeline {
  @override
  Future<TimelineState> build() async {
    final repo = ref.watch(postRepositoryProvider);
    final page = await repo.listTimeline(limit: timelinePageLimit);
    return TimelineState(items: _dedupe(page.items), cursor: page.cursor);
  }

  Future<void> loadMore() async { ... } // same copyWithPrevious pattern as UserPosts

  void prepend(Post post) { ... }       // no-op if no data or same uri exists
  void replace(Post post) { ... }       // match by uri only
  void removeByUri(AtUri uri) { ... }   // match by uri only
}

void updateLiveTimelineCache(Ref ref, Post post) {
  if (ref.exists(timelineProvider)) {
    ref.read(timelineProvider.notifier).replace(post);
  }
}

void prependLiveTimelineCache(Ref ref, Post post) { ... }
void removeFromLiveTimelineCache(Ref ref, AtUri uri) { ... }
```

Implementation guardrails:

- Use `state.value` guard and `state.isLoading` guard exactly like `UserPosts.loadMore`. (`FR-007`, `UT-004`, `UT-005`)
- For load-more failure, use `AsyncLoading<TimelineState>().copyWithPrevious(state)` and then `next.copyWithPrevious(state)` to preserve visible data and cursor. (`FR-007`, `AC-007`)
- Deduplicate appended page items against existing items by `post.uri`; keep existing item position when duplicate arrives from a later server page. (`RULE-004`, `UT-007`)
- Use `ref.exists(timelineProvider)` before mutation helpers read the notifier so mutation providers do not instantiate a non-live timeline. (`EC-009`, `UT-006`)
- Do not make `timelineProvider` a family; the endpoint is auth-scoped with no handle/DID argument. (`RULE-001`, `FR-002`)

### Provider graph

```text
dioProvider (existing)
  -> postApiClientProvider (existing)
    -> postRepositoryProvider (existing)
      -> timelineProvider (new)

authSessionProvider (existing)
  -> FeedPage own-post delete visibility only

createPostProvider / toggleLikePostProvider / toggleRepostPostProvider / deletePostProvider (existing)
  -> timelineProvider helper functions (new)
  -> userPostsProvider/userCommentsProvider helpers (existing)
```

## 7. UI, Widgets, Routes, Or User-Facing Surfaces

### FeedPage composition

Change `FeedPage` from a placeholder `ConsumerWidget` into the timeline surface. It can remain a `ConsumerWidget` unless implementation needs local state.

```text
Scaffold
  AppBar(title: l10n.feedTitle)
  body: switch (timelineAsync, timelineAsync.value)
    loading initial -> Center(StitchProgressIndicator)
    initial error -> _FeedError(onRetry: ref.invalidate(timelineProvider))
    loaded -> CustomScrollView(
      slivers: [
        _FeedComposeEntry(showPostComposerSheet(context)),
        if items.empty -> SliverFillRemaining(Text(l10n.feedEmpty))
        else -> SliverList.builder(PostCard rows)
        if loadingMore/errorMore -> bottom spinner or retry button
      ]
    )
```

Recommended private widgets/helpers in `feed_page.dart`:

- `_FeedTimelineLoadedSlivers` or `_FeedTimelineBody`
- `_FeedErrorSliver` / `_FeedInitialError`
- `_FeedComposeEntry`
- `_confirmDelete(context, ref, post)` mirroring profile tab
- `_replyAndOpenThread(context, ref, post)` mirroring profile tab, but updating `timelineProvider.notifier.replace(...)`

Keep these private to avoid premature generic feed components. If repeated code becomes unwieldy, extract only small helpers that are clearly shared with profile tabs; do not build a generic feed framework. (`BR-002`, `DR-002`)

### Timeline row behavior

Each `PostCard` row should wire:

- `onTap`: `PostThreadRoute(did: post.author.did, rkey: post.rkey).push<void>(context)` (`FR-010`, `AT-007`)
- `onReply`: show composer with `replyTarget: post`; on created reply, update root row (`replyCount + 1`, `viewerHasReplied: true`) and push thread with `focus: created.uri`, `$extra: created` (`FR-012`, `AT-009`)
- `onLike`: `ref.read(toggleLikePostProvider.notifier).toggle(post: post)` (`FR-011`, `AT-008`)
- `onRepost`: `ref.read(toggleRepostPostProvider.notifier).toggle(post: post)` (`FR-011`, `AT-008`)
- `onDelete`: only when `authSessionProvider.value` is `SignedIn` and `post.author.did == signedIn.did`; confirm then call `deletePostProvider.notifier.delete(post: post)` (`FR-013`, `AT-010`)

### Top-level compose entry

Use existing `showPostComposerSheet(context)` with no `replyTarget` from a top-of-feed `ChunkyButton` or equivalent existing design-system control. Reuse `l10n.postComposeAction` unless product copy requires a feed-specific label. `CreatePost` will handle optimistic timeline insertion. (`FR-014`, `FR-015`, `AT-011`, `AT-012`)

### Delete feedback

Mirror `ProfilePostsTab` listener behavior in `FeedPage`:

- On `deletePostProvider` loading -> data: show `l10n.postDeleteSuccess`, reset provider.
- On loading -> error: show `l10n.postDeleteError`, reset provider.

Do not add new delete copy for this slice.

### Localization

Add minimal feed copy to `app_en.arb`:

```text
feedEmpty: e.g. "Your feed is quiet."
feedLoadError: e.g. "Feed didn't load."
```

Descriptions should mention main chronological Feed tab. Reuse `loading`, `retryButton`, and existing post compose/delete strings. Regenerate localizations.

## 8. Error, Loading, Empty, And Edge States

| State / Case | Planned Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| Initial loading | Show centered `StitchProgressIndicator`; keep AppBar title. | BR-001 | AT-001 |
| Initial API/provider error | Show feed-specific localized error with retry; retry invalidates/rebuilds `timelineProvider`. | FR-006, NFR-004 | AT-004 |
| Empty success | Show localized empty-feed state; no onboarding/discovery/recommendations. Compose entry may still be visible. | FR-008, NFR-004 | AT-003 |
| Loaded first page without cursor | Render posts; do not show bottom spinner/retry or call loadMore. | FR-003, FR-005 | UT-005, AT-002 |
| Loaded page with cursor | Near-end row scheduling triggers `timelineProvider.notifier.loadMore()`. | FR-003, FR-005, NFR-003 | UT-003, AT-005 |
| Load-more in progress | Preserve list and show bottom `StitchProgressIndicator`. | FR-007 | UT-004, AT-005 |
| Load-more failure | Preserve list and cursor; show bottom retry; retry calls `loadMore` with same cursor. | FR-007 | UT-004, AT-006 |
| Duplicate URI from optimistic create and server page | Keep a single row by URI. | FR-015, RULE-004 | UT-007, AT-012 |
| Top-level create while timeline not live | Do not instantiate `timelineProvider`; future timeline load gets AppView data normally. | FR-015 | UT-006 |
| Reply create from timeline | Update root row reply state and navigate to focused thread; do not insert reply row into timeline. | FR-012, RULE-003 | UT-009, AT-009 |
| Like/repost mutation failure | Roll back timeline row alongside existing profile/comment cache rollback. | FR-011 | UT-008, AT-008 |
| Delete own post | Expose delete only for viewer-authored posts; remove by URI after repository success. | FR-013 | UT-010, AT-010 |
| Auth state unavailable/transient | Do not expose delete until signed-in DID is available; normal router should keep Feed signed-in. | RULE-001 | AT-010, REG-008 |

## 9. Test Implementation Plan

| Order | Test ID | Target | Setup / Fixture | Initial Expected Failure |
|---|---|---|---|---|
| 1 | IT-001 | `app/test/feed/data/post_api_client_test.dart` | Mock `/v1/feed/timeline` returning `{items: [samplePost], cursor}`. | `PostApiClient.listTimeline` is undefined. |
| 2 | IT-002 | `app/test/feed/data/post_api_client_test.dart` | Mock query parameters `cursor=c1`, `limit=20`. | Method missing or does not send expected query params. |
| 3 | IT-003 | `app/test/feed/data/post_api_client_test.dart` | Empty response and error-envelope response. | Method missing or error/empty parsing not covered. |
| 4 | IT-004 / UT-001 | Repository interface/fake plus provider tests | Add `onListTimeline` callback to `FakePostRepository`; call `repo.listTimeline(limit: 20)` through fake/provider. | `PostRepository` and fake lack `listTimeline`. |
| 5 | UT-002 | `app/test/feed/providers/timeline_provider_test.dart` | Fake repo returns first page with cursor; initialize mappers. | `timelineProvider`/`TimelineState` missing. |
| 6 | UT-003 | `app/test/feed/providers/timeline_provider_test.dart` | Fake repo records cursor and returns second page. | `loadMore` missing or not passing cursor/appending. |
| 7 | UT-004 | `app/test/feed/providers/timeline_provider_test.dart` | Second call throws; third succeeds. | Provider loses previous data/cursor or cannot retry. |
| 8 | UT-005 | `app/test/feed/providers/timeline_provider_test.dart` | Terminal cursor null and gated completer for in-flight loadMore. | Provider calls repo when it should no-op. |
| 9 | UT-006 | `app/test/feed/providers/create_post_provider_test.dart` + timeline tests | Live timeline; top-level create; reply create. | `CreatePost` updates only profile caches. |
| 10 | UT-007 | `app/test/feed/providers/timeline_provider_test.dart` | Existing URI duplicated in prepend or fetched page. | Timeline duplicates rows by URI. |
| 11 | AT-001 | `app/test/feed/feed_page_test.dart` | Fake repo returns a pending `Completer<PostPage>`. | FeedPage still shows placeholder body. |
| 12 | AT-002 | `app/test/feed/feed_page_test.dart` | Fake repo returns posts with rich fields/images. | FeedPage does not render timeline `PostCard`s. |
| 13 | AT-003 | `app/test/feed/feed_page_test.dart` | Fake repo returns `PostPage(items: [])`. | Empty feed copy missing. |
| 14 | AT-004 | `app/test/feed/feed_page_test.dart` | Fake repo throws then succeeds after retry. | Initial error UI/retry missing. |
| 15 | AT-005 | `app/test/feed/feed_page_test.dart` | First page 10+ rows with cursor; second page one row. | Scroll near end does not load/append. |
| 16 | AT-006 | `app/test/feed/feed_page_test.dart` | Next-page failure then retry success. | Load-more error blanks list or cannot retry same cursor. |
| 17 | AT-007 | `app/test/feed/feed_page_test.dart` | GoRouter test route captures post thread parameters. | Row tap not wired to thread route. |
| 18 | AT-008 / UT-008 | `app/test/feed/feed_page_test.dart`, `toggle_post_interactions_provider_test.dart` | Fake like/repost success and failure; live timeline. | Interaction providers update profile caches only. |
| 19 | AT-009 / UT-009 | `app/test/feed/feed_page_test.dart`, timeline provider tests | Reply composer success; route captures focus. | Timeline row reply state not updated or reply inserted as row. |
| 20 | AT-010 / UT-010 | `app/test/feed/feed_page_test.dart`, timeline provider tests | Override `authSessionProvider` with `SignedInAuthSession`; viewer and other posts. | Delete visibility/removal not implemented for timeline. |
| 21 | AT-011 | `app/test/feed/feed_page_test.dart` | Loaded feed; fake `onCreate` captures `reply == null`. | Feed has no top-level compose entry. |
| 22 | AT-012 | `app/test/feed/feed_page_test.dart` | Top-level create then duplicate server page/refresh. | Created post not prepended or duplicates. |
| 23 | REG-001 through REG-008 | Existing focused suites | Run listed regression commands and generated-code command. | Any shared-helper, l10n, provider, route, or post-card regressions surface. |

## 10. Sequencing And Guardrails

- First TDD step: Add failing `IT-001` for `PostApiClient.listTimeline()` in `app/test/feed/data/post_api_client_test.dart`.
- Dependencies between work items:
  - API/repository methods must land before `timelineProvider` can compile.
  - `TimelineState`/`timelineProvider` must land before FeedPage widget tests can use provider overrides/fakes.
  - Timeline mutation helpers must land before create/like/repost/delete provider tests can pass.
  - FeedPage top-level UI can be built after provider first-load/pagination behavior is covered.
  - Generated code must be refreshed after adding Riverpod providers, mappable state, and ARB strings.
- Guardrails:
  - Do not modify AppView/Go code, lexicons, migrations, dependencies, or endpoint contracts.
  - Do not add a generic feed framework, feed-item envelope, ranking, recommendations, discovery cards, project/search/list feed filters, durable cache, or PDS read-through.
  - Keep timeline cursor opaque; do not parse or construct cursor payloads.
  - Use URI matching for timeline dedupe/replacement/removal.
  - Preserve existing profile/comment cache behavior; add timeline updates alongside existing helpers rather than replacing them.
  - Keep `IT-004` pragmatic per DR-001; avoid adding an API-client abstraction solely for a delegation test.
  - If extracting shared UI/provider helpers from profile tabs, keep extraction small and covered by profile regression tests.
- Out of scope:
  - Full-device E2E with live AppView auth.
  - Performance benchmark beyond bounded calls/lazy rendering and manual smoke check.
  - Quote-card expansion and repost feed reasons.

## 11. Risks And Open Questions

| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Non-blocking | `TimelineState` can use `dart_mappable` like `UserPostsState` or a hand-written immutable class. | Generated-code scope varies slightly. | Recommended: use `dart_mappable` for consistency; include generated output and `REG-006`. |
| CPQ-002 | Non-blocking | Direct `ApiPostRepository` delegation test may be awkward without a `PostApiClient` interface. | Over-testing could lead to unnecessary abstraction. | Follow DR-001: verify through interface/fake/provider tests unless a trivial seam already exists. |
| CPQ-003 | Non-blocking | Feed UI compose entry placement could be top sliver or floating action button. | Widget tests need stable target text/semantics. | Recommended: use a top sliver `ChunkyButton` with existing `postComposeAction`, matching profile tab conventions. |
| CPQ-004 | Non-blocking | Shared cache-update helpers may stay in `timeline_provider.dart` or be extracted later. | Poor placement could increase coupling. | Recommended: keep timeline helpers in `timeline_provider.dart` for this slice; revisit normalized cache later if repeated patterns grow. |
| CPQ-005 | Non-blocking | Existing analyzer may have info-level findings unrelated to this change. | `flutter analyze` may not be a clean gate. | Use focused tests and generated-code verification as required; run analyze opportunistically and do not expand scope to unrelated cleanup. |

## 12. Handoff To TDD Builder

- Coding plan: `docs/changes/2026-05-29-timeline-feed-flutter/04-coding-plan.md`
- TDD execution plan: `docs/changes/2026-05-29-timeline-feed-flutter/05-implementation-plan.md`
- Start with test: `IT-001` in `app/test/feed/data/post_api_client_test.dart` for `PostApiClient.listTimeline()` no-cursor parsing.
- Focused command: `cd app && flutter test test/feed/data/post_api_client_test.dart --plain-name "PostApiClient.listTimeline"`
- Follow-up focused commands:
  - `cd app && flutter test test/feed/providers/timeline_provider_test.dart`
  - `cd app && flutter test test/feed/feed_page_test.dart`
  - `cd app && flutter test test/feed/providers/create_post_provider_test.dart test/feed/providers/toggle_post_interactions_provider_test.dart`
  - `cd app && flutter test test/profile/widgets/profile_posts_tab_test.dart test/profile/widgets/profile_comments_tab_test.dart`
  - `cd app && dart run build_runner build --delete-conflicting-outputs`
  - `cd app && flutter test`
- Notes:
  - Read `01-requirements.md`, `02-acceptance-tests.md`, and `03-document-review.md` first.
  - Keep commits focused in the implementation stage; include generated files when provider/model/l10n changes require them.
  - Record actual test execution and any deviations from this plan in `05-implementation-plan.md`.
