# Flutter Feed Post Data Layer Design

- **Status:** Draft
- **Date:** 2026-05-07
- **Related:**
  - [feed-post-crud-endpoints](2026-05-04-feed-post-crud-endpoints-design.md) — the AppView surface this client consumes. Wire shapes, error codes, and pagination semantics are defined there; this spec is the Flutter-side implementation.
  - [appview-api-architecture](2026-04-21-appview-api-architecture-design.md) — the `/v1/*` URL convention, auth headers, error envelope, and opaque-cursor pagination this layer relies on.
  - [api-wire-alignment](2026-04-22-api-wire-alignment-design.md) — camelCase across the entire `/v1/*` surface; lines up with `dart_mappable`'s default JSON serialization.
  - [profile-onboarding](2026-04-23-profile-onboarding-design.md) — the existing `ProfileApiClient` / `ProfileRepository` / `UserProfile` / `SaveProfile` pattern this layer mirrors verbatim.
  - [flutter-auth](2026-04-21-flutter-auth-design.md) — owns the Dio session-token interceptor stack; new endpoints inherit auth + 401 handling for free.

## Summary

Add the client-side data layer for Craftsky's text-post CRUD: a `Post` model, a Dio-backed `PostApiClient`, a `PostRepository` interface and production impl, and the Riverpod providers (single-post read, paginated list-by-author, plus standalone `CreatePost` and `DeletePost` mutation notifiers). The slice is data-layer-only — no widgets, pages, or routes change, and `PlaceholderPost` continues to back `post_card.dart` and `profile_posts_tab.dart` until a follow-up spec wires the UI.

## Goals

1. Land the four AppView post endpoints in the Flutter client with the same three-layer shape the profile feature uses (`ApiClient` → `Repository` → providers), so a contributor reading `post_repository.dart` next to `profile_repository.dart` sees the same structure.
2. Establish the cursor-accumulating list-pagination pattern in this codebase. There is no prior pagination idiom in `app/`; this spec sets the precedent for every future paginated surface (timeline, search, notifications).
3. Wire the cache-update path so create and delete update live `userPostsProvider` family entries directly — avoiding a refetch that would race the firehose-driven indexer (the AppView spec calls this out as a known v1 risk on both create and delete).
4. Keep the slice tight: text-only writes, no facets, no reply, no quote on the write surface; full read surface preserved.

## Non-goals

- **No widget changes.** `feed/widgets/post_card.dart`, `feed/pages/feed_page.dart`, and `profile/widgets/profile_tabs/profile_posts_tab.dart` stay bound to `PlaceholderPost`. A follow-up spec replaces them with the real `Post` model and adds a compose entry point.
- **No new routes.** No deep-link page for `/posts/{did}/{rkey}`, no compose route, no thread page. The single-post `postProvider` exists for future routes to consume; nothing in the UI calls it in v1.
- **No write parameters beyond `text`.** Reply, quote, and facets are all read-side only in v1. Adding them is purely additive (extra optional named args on `createPost` / `PostRepository.create`).
- **No `atproto.dart` adoption.** Identifiers (`did`, `handle`, `uri`, `cid`, `rkey`) stay plain `String` to match the existing `Profile` model. AGENTS.md flags `atproto.dart`'s typed wrappers as preferred; introducing them via this slice is scope creep, and a future "atproto.dart adoption" spec switches the whole client at once.
- **No `DummyPostRepository`.** `DummyProfileRepository` exists in `lib/` but is not wired into anything (it's an opt-in dev convenience for design previews); skipping the equivalent here keeps the slice focused. Adding one later is mechanical.
- **No image proxying.** `author.avatarCid` is surfaced as a bare CID, exactly as the AppView returns it. Rendering is a UI-layer concern coupled to a future image-proxy spec.
- **No facet richtext model.** The `facets` field is preserved as raw JSON (`List<Map<String, dynamic>>?`). A typed `Facet` model lands when the richtext renderer does.

## Context

### What's already in place

- **API plumbing.** `dioProvider` ([dio_provider.dart](../../../app/lib/shared/api/providers/dio_provider.dart)) attaches the auth interceptor, error-mapping interceptor, and 401 sign-out interceptor; every `/v1/*` request inherits them.
- **Sealed exceptions.** `ApiException` and its variants ([api_exception.dart](../../../app/lib/shared/api/api_exception.dart)) are the single error surface; `unwrapApi` ([api_unwrap.dart](../../../app/lib/shared/api/api_unwrap.dart)) translates `DioException` into them.
- **Profile-shaped precedent.** `ProfileApiClient` / `ApiProfileRepository` / `ProfileRepository` / `UserProfile` family / `SaveProfile` standalone notifier — every pattern this spec reaches for is already proven in `app/lib/profile/`.
- **Mappable convention.** `dart_mappable` is the only data-modeling library; codegen part files (`*.mapper.dart`) supply equality, `copyWith`, and JSON round-trip.
- **Riverpod 3.x with codegen.** `@Riverpod(keepAlive: true)` for repository singletons, `@riverpod` for families and notifiers, `AsyncValue.guard` + `ref.mounted` checks for mutations.
- **Placeholder UI.** `PlaceholderPost` is what `PostCard` and `ProfilePostsTab` consume today. It stays untouched.

### Scope decisions

Resolved during brainstorming and locked here:

- **Slice:** data layer only. No widget, page, or route changes.
- **Endpoints:** all four AppView endpoints (`createPost`, `getPost`, `deletePost`, `listPostsByAuthor`).
- **Write surface:** `text` only. Reply, quote, facets are read-side only.
- **Pagination shape:** cursor-accumulating `AsyncNotifier` with `loadMore()`. Sets the precedent for future paginated surfaces.
- **Mutation shape:** standalone `CreatePost` and `DeletePost` notifiers (mirroring `SaveProfile`), each pushing into live `userPostsProvider` family entries on success via cache helpers (`prepend`, `removeByRkey`) on the list notifier.
- **Loading-more UX signal:** `AsyncValue.isLoading` only — no parallel `isLoadingMore` boolean. `copyWithPrevious` preserves visible items across loading and error transitions.

## Design

### 1. File layout

```
app/lib/feed/
├── data/
│   ├── post_api_client.dart          # Dio wrapper, four methods
│   ├── post_repository.dart          # interface
│   └── api_post_repository.dart      # production impl
├── models/
│   ├── post.dart                     # wire model + nested PostAuthor / PostRef / PostReply
│   ├── post.mapper.dart              # generated
│   ├── post_page.dart                # { items, cursor } envelope
│   ├── post_page.mapper.dart         # generated
│   ├── user_posts_state.dart         # { items, cursor } — UserPosts notifier state
│   ├── user_posts_state.mapper.dart  # generated
│   └── placeholder_post.dart         # unchanged — still backs post_card.dart and profile_posts_tab.dart
└── providers/
    ├── post_api_client_provider.dart       # + .g.dart  — keepAlive: true
    ├── post_repository_provider.dart       # + .g.dart  — keepAlive: true
    ├── post_provider.dart                  # + .g.dart  — single-post read family
    ├── user_posts_provider.dart            # + .g.dart  — list-by-author with loadMore + helpers
    ├── create_post_provider.dart           # + .g.dart  — CreatePost mutation notifier
    └── delete_post_provider.dart           # + .g.dart  — DeletePost mutation notifier
```

Test layout mirrors the existing `app/test/profile/`:

```
app/test/feed/
├── data/
│   └── post_api_client_test.dart
├── fakes/
│   └── fake_post_repository.dart
├── models/
│   ├── post_test.dart
│   ├── post_page_test.dart
│   └── user_posts_state_test.dart
└── providers/
    ├── post_provider_test.dart
    ├── user_posts_provider_test.dart
    ├── create_post_provider_test.dart
    └── delete_post_provider_test.dart
```

### 2. Models

Three nested mappable types in `post.dart`, plus `PostPage` and `UserPostsState` as sibling files.

```dart
@MappableClass()
class Post with PostMappable {
  const Post({
    required this.uri,
    required this.cid,
    required this.rkey,
    required this.text,
    required this.tags,
    required this.createdAt,
    required this.indexedAt,
    required this.author,
    this.facets,
    this.reply,
    this.quote,
  });

  final String uri;
  final String cid;
  final String rkey;
  final String text;
  final List<Map<String, dynamic>>? facets;  // raw JSON pass-through
  final List<String> tags;                    // always present per AppView spec
  final PostReply? reply;
  final PostRef? quote;
  final DateTime createdAt;
  final DateTime indexedAt;
  final PostAuthor author;
}

@MappableClass()
class PostAuthor with PostAuthorMappable {
  const PostAuthor({
    required this.did,
    required this.handle,
    this.displayName,
    this.avatarCid,
  });
  final String did;
  final String handle;
  final String? displayName;
  final String? avatarCid;        // bare CID, not a URL
}

@MappableClass()
class PostRef with PostRefMappable {
  const PostRef({required this.uri, required this.cid});
  final String uri;
  final String cid;
}

@MappableClass()
class PostReply with PostReplyMappable {
  const PostReply({required this.root, required this.parent});
  final PostRef root;
  final PostRef parent;
}
```

```dart
@MappableClass()
class PostPage with PostPageMappable {
  const PostPage({required this.items, this.cursor});
  final List<Post> items;
  final String? cursor;   // absent in JSON when no more pages — dart_mappable maps absence to null
}
```

```dart
@MappableClass()
class UserPostsState with UserPostsStateMappable {
  const UserPostsState({required this.items, this.cursor});
  final List<Post> items;
  final String? cursor;
  bool get hasMore => cursor != null;
}
```

Modeling notes:

- **`facets` typed as `List<Map<String, dynamic>>?`.** The AppView treats facets as a pass-through; lexicon validation lives on the receiving PDS. No v1 client code renders rich text, so committing to a typed `Facet` shape now would churn when the renderer lands.
- **Plain `String` for atproto identifiers.** Matches `Profile`. A future spec migrates the entire client to `atproto.dart`'s typed wrappers.
- **No `images` field.** AppView omits it from the response shape entirely in v1.

### 3. `PostApiClient`

Direct mirror of `ProfileApiClient`. Every call wraps in `unwrapApi`; JSON shapes round-trip through generated `*Mapper.fromMap` / `toMap`.

```dart
class PostApiClient {
  const PostApiClient(this._dio);
  final Dio _dio;

  /// POST /v1/posts — text-only create. AppView returns a synthetic
  /// [Post] populated from the PDS write response; the firehose
  /// indexer hasn't necessarily caught up yet.
  Future<Post> createPost({required String text}) => unwrapApi(() async {
    final res = await _dio.post<Map<String, dynamic>>(
      '/v1/posts',
      data: {'text': text},
    );
    return PostMapper.fromMap(res.data!);
  });

  /// GET /v1/posts/{did}/{rkey}
  Future<Post> getPost(String did, String rkey) => unwrapApi(() async {
    final res = await _dio.get<Map<String, dynamic>>('/v1/posts/$did/$rkey');
    return PostMapper.fromMap(res.data!);
  });

  /// DELETE /v1/posts/{did}/{rkey} — idempotent per AppView spec.
  Future<void> deletePost(String did, String rkey) => unwrapApi(() async {
    await _dio.delete<void>('/v1/posts/$did/$rkey');
  });

  /// GET /v1/profiles/@{handleOrDid}/posts — newest-first.
  /// `@`-prefix matches the existing convention in [ProfileApiClient.getProfile];
  /// the AppView strips it before resolving. [limit] caps server-side at 100.
  Future<PostPage> listPostsByAuthor(
    String handleOrDid, {
    String? cursor,
    int? limit,
  }) => unwrapApi(() async {
    final res = await _dio.get<Map<String, dynamic>>(
      '/v1/profiles/@$handleOrDid/posts',
      queryParameters: {
        if (cursor != null) 'cursor': cursor,
        if (limit != null) 'limit': limit,
      },
    );
    return PostPageMapper.fromMap(res.data!);
  });
}
```

- Method names follow `ProfileApiClient` style (`createPost`, `getPost`, `deletePost`, `listPostsByAuthor`).
- `text` is named on `createPost` to leave room for additive `reply: ...`, `quote: ...`, `facets: ...` parameters later without breaking callers.
- Cursor is opaque — passed through verbatim from `PostPage.cursor` to the next call.

### 4. `PostRepository` and `ApiPostRepository`

```dart
abstract interface class PostRepository {
  Future<Post> create({required String text});
  Future<Post> fetch(String did, String rkey);
  Future<void> delete(String did, String rkey);
  Future<PostPage> listByAuthor(
    String handleOrDid, {
    String? cursor,
    int? limit,
  });
}
```

```dart
class ApiPostRepository implements PostRepository {
  const ApiPostRepository(this._api);
  final PostApiClient _api;

  @override
  Future<Post> create({required String text}) => _api.createPost(text: text);

  @override
  Future<Post> fetch(String did, String rkey) => _api.getPost(did, rkey);

  @override
  Future<void> delete(String did, String rkey) => _api.deletePost(did, rkey);

  @override
  Future<PostPage> listByAuthor(
    String handleOrDid, {
    String? cursor,
    int? limit,
  }) => _api.listPostsByAuthor(handleOrDid, cursor: cursor, limit: limit);
}
```

Repository method names drop the `Post` suffix (already namespaced by the type), matching `ProfileRepository.fetch` / `fetchMe` / `updateMe`.

### 5. Providers

#### Wrappers

```dart
@Riverpod(keepAlive: true)
PostApiClient postApiClient(Ref ref) =>
    PostApiClient(ref.watch(dioProvider));

@Riverpod(keepAlive: true)
PostRepository postRepository(Ref ref) =>
    ApiPostRepository(ref.watch(postApiClientProvider));
```

#### `postProvider(did, rkey)` — single-post read

```dart
@riverpod
Future<Post> post(Ref ref, String did, String rkey) =>
    ref.watch(postRepositoryProvider).fetch(did, rkey);
```

No UI consumes this in v1. It exists for future routes (deep-link share, thread page).

#### `userPostsProvider(handleOrDid)` — paginated list-by-author

```dart
@riverpod
class UserPosts extends _$UserPosts {
  @override
  Future<UserPostsState> build(String handleOrDid) async {
    final repo = ref.watch(postRepositoryProvider);
    final page = await repo.listByAuthor(handleOrDid);
    return UserPostsState(items: page.items, cursor: page.cursor);
  }

  /// Append-next-page. No-op if data not loaded, no more pages, or
  /// already loading. On success appends and advances cursor; on failure
  /// `state` becomes AsyncError but `state.value` still returns the
  /// existing list (via copyWithPrevious), so the UI keeps showing items
  /// and a retry uses the same cursor.
  Future<void> loadMore() async {
    final current = state.value;
    if (current == null || !current.hasMore || state.isLoading) return;

    state = const AsyncLoading<UserPostsState>().copyWithPrevious(state);

    final next = await AsyncValue.guard(() async {
      final repo = ref.read(postRepositoryProvider);
      final page = await repo.listByAuthor(handleOrDid, cursor: current.cursor);
      return UserPostsState(
        items: [...current.items, ...page.items],
        cursor: page.cursor,
      );
    });

    if (!ref.mounted) return;
    state = next.copyWithPrevious(state);
  }

  /// Cache helper called by [CreatePost] on success. Dedupes by uri so a
  /// firehose-driven refresh that races the synthetic response doesn't
  /// double-insert.
  void prepend(Post post) {
    final current = state.value;
    if (current == null) return;
    if (current.items.any((p) => p.uri == post.uri)) return;
    state = AsyncData(
      current.copyWith(items: [post, ...current.items]),
    );
  }

  /// Cache helper called by [DeletePost] on success.
  void removeByRkey(String rkey) {
    final current = state.value;
    if (current == null) return;
    state = AsyncData(
      current.copyWith(
        items: current.items.where((p) => p.rkey != rkey).toList(),
      ),
    );
  }
}
```

Both `copyWithPrevious` calls in `loadMore` are load-bearing:

- The first preserves `current.items` through `AsyncLoading` so `state.value` keeps returning the existing list during the fetch, and the UI's data-first switch keeps rendering the list.
- The second preserves them through `AsyncError` on failure. The cursor is unchanged on failure (we don't advance until the response arrives), so a retry just calls `loadMore()` again with the same cursor.

Re-entrancy is guarded by `state.isLoading`, which subsumes the role a separate `isLoadingMore` flag would otherwise play.

#### `CreatePost` — text-only mutation

```dart
@riverpod
class CreatePost extends _$CreatePost {
  @override
  FutureOr<Post?> build() => null;

  Future<void> create({required String text}) async {
    state = const AsyncLoading();
    state = await AsyncValue.guard(() async {
      final repo = ref.read(postRepositoryProvider);
      final post = await repo.create(text: text);
      if (!ref.mounted) return null;

      // Push into any live userPostsProvider entries keyed by either
      // form of the author's identity. ref.exists guards against
      // accidentally instantiating a non-live family entry, which would
      // race a fresh build against our prepend.
      for (final id in <String>{post.author.handle, post.author.did}) {
        final entry = userPostsProvider(id);
        if (ref.exists(entry)) {
          ref.read(entry.notifier).prepend(post);
        }
      }

      return post;
    });
  }

  void reset() => state = const AsyncData(null);
}
```

Mirrors `SaveProfile` exactly. UI binds `ref.listen(createPostProvider, ...)` for navigation/snackbar, calls `reset()` after consuming a transition.

#### `DeletePost` — takes `Post` for cache-update bookkeeping

```dart
@riverpod
class DeletePost extends _$DeletePost {
  @override
  FutureOr<Post?> build() => null;

  Future<void> delete({required Post post}) async {
    state = const AsyncLoading();
    state = await AsyncValue.guard(() async {
      final repo = ref.read(postRepositoryProvider);
      await repo.delete(post.author.did, post.rkey);
      if (!ref.mounted) return null;

      for (final id in <String>{post.author.did, post.author.handle}) {
        final entry = userPostsProvider(id);
        if (ref.exists(entry)) {
          ref.read(entry.notifier).removeByRkey(post.rkey);
        }
      }

      return post;
    });
  }

  void reset() => state = const AsyncData(null);
}
```

`delete` takes the whole `Post` because the cache update needs `did`, `handle`, and `rkey` to splice the post out of any live family entries (lists may be keyed by either form). The caller — UI deleting a post it's already rendering — has the `Post` in hand.

`build()` returns `Post?` so the `AsyncData(post)` transition carries the deleted post for `ref.listen` consumers (e.g. an "undo delete" snackbar).

### 6. Cache update strategy

The AppView spec calls out two known v1 inconsistency windows:

- **Read-after-write:** between `POST /v1/posts` returning and the indexer populating the row, a refetch of `listPostsByAuthor` may not include the new post.
- **Read-after-delete:** between `DELETE /v1/posts/...` returning and the firehose tombstone arriving, a refetch may still include the deleted post.

Blind invalidation of `userPostsProvider` after a mutation would expose both windows directly to the user ("I created a post and it's missing"; "I deleted a post and it came back"). The cache-helper pattern (`prepend`, `removeByRkey`) sidesteps both by mutating the in-memory list directly, using the synthetic response from the AppView as the source of truth for the local cache. The next firehose-driven refresh — whenever it happens — converges naturally; `prepend` dedupes by uri.

No optimistic-before-success: cache mutations only run after the AppView confirms. If the network fails, the in-memory list is unchanged and the mutation notifier transitions to `AsyncError`.

### 7. Error handling

The existing infrastructure carries the weight. `ErrorMappingInterceptor` on `dioProvider` maps HTTP status to sealed `ApiException` subtypes; `unwrapApi` translates them into direct throws.

| AppView error code | HTTP | Maps to | Surfaced as |
|---|---|---|---|
| `validation_failed` | 422 | `ApiBadRequest('validation_failed')` | `AsyncError` on `createPostProvider` |
| `post_not_found` | 404 | `ApiBadRequest('post_not_found')` | `AsyncError` on `postProvider(did, rkey)` |
| `forbidden` | 403 | `ApiBadRequest('forbidden')` | `AsyncError` on `deletePostProvider` |
| `pds_unavailable` / `pds_write_failed` | 502 | `ApiServerError(message)` | `AsyncError` |
| connection / timeout | — | `ApiNetworkError(message)` | `AsyncError` |
| any | 401 | `ApiUnauthorized` | global signout (existing 401 handler) |

No new exception types. UI-layer copy keyed on `ApiBadRequest.code` is deferred to the future UI-wiring spec.

### 8. Tests

`ProviderContainer.test()` per test, override `postRepositoryProvider` with `FakePostRepository`, `await container.read(provider.future)` for async reads, `ref.listen` transition assertions for mutations.

**Coverage targets:**

- **`PostApiClient`** (`http_mock_adapter`):
  - Each method's URL, body, query params.
  - `unwrapApi` translates `DioException` (401/4xx/5xx/network) into the right `ApiException` subtypes.
  - JSON round-trip for response shapes.
- **`Post` / `PostPage` / `UserPostsState` mappers:**
  - `fromMap(json).toMap() == json` for representative payloads (with and without optional fields).
  - `PostPage` cursor-present and cursor-absent.
- **`UserPosts` notifier:**
  - Initial fetch — no-cursor and with-cursor branches.
  - `loadMore` appends and advances cursor.
  - `loadMore` no-op when `!hasMore`.
  - `loadMore` no-op when `state.isLoading`.
  - `loadMore` failure: `state` becomes `AsyncError`, `state.value` still returns previous list, cursor unchanged, retry uses same cursor.
  - `prepend` dedupes by uri; no-op when state has no data.
  - `removeByRkey` filters; no-op when state has no data.
- **`CreatePost`:**
  - Idle → loading → data transition.
  - Success pushes into both live family entries (`userPostsProvider(did)` and `userPostsProvider(handle)`).
  - Success does NOT instantiate a non-live family entry.
  - `reset()` returns to `AsyncData(null)`.
  - Failure transitions to `AsyncError`, no cache mutation.
- **`DeletePost`:**
  - Success removes from both family entries by `post.author.did` and `post.author.handle`.
  - Failure: cache untouched, transitions to `AsyncError`.
  - `reset()` works.
- **`postProvider(did, rkey)`:**
  - Round-trip; AsyncError propagation.

### 9. Wiring & dependencies

- **No `pubspec.yaml` changes.** Everything uses already-declared dependencies (`dio`, `dart_mappable`, `riverpod_annotation`, `flutter_riverpod`).
- **No router changes.** Data layer only.
- **No widget changes.** `feed/widgets/post_card.dart`, `feed/pages/feed_page.dart`, `profile/widgets/profile_tabs/profile_posts_tab.dart` stay on `PlaceholderPost`.
- **Codegen.** Run `dart run build_runner build --delete-conflicting-outputs` after creating mappable models and `@riverpod` provider files.

## Alternatives considered

### Single-shot list provider (no `loadMore`) instead of an accumulating notifier

A simpler `userPostsProvider(handleOrDid)` returning `Future<PostPage>` from the first page only.

**Rejected** because the eventual UI consumer (profile posts tab) needs scroll-driven pagination. A single-shot provider has nowhere natural to grow `loadMore` without becoming the accumulating notifier anyway, so picking the simpler shape now would force a refactor of every consumer when the UI lands.

### In-place mutations on the list notifier instead of standalone notifiers

`userPostsProvider(did).notifier.createPost(text)` and `.deletePost(rkey)` — same notifier owns both the list and its mutations, like `UserProfile.updateDisplayName`.

**Rejected** because compose and delete are standalone-flow shapes — a UI eventually wants `ref.listen(createPostProvider, ...)` for snackbar/navigation, and a delete-confirmation dialog needs its own loading/error state separate from the list it's deleting from. The codebase already splits these patterns (`UserProfile` for in-place edits where the resource *is* the cache; `SaveProfile` for "submit form, listen for success/error"). Following the precedent.

### Optimistic-before-success cache updates with rollback on error

Mutate the in-memory list immediately on user action, call the API, restore on failure (mirroring `UserProfile._patch`).

**Rejected** for v1. Optimistic UX is appealing but the rollback paths are subtle: where does a deleted post reappear in the list on failure? At the same index? At the top? Does a failed create produce a "ghost" post that flickers? The success-then-mutate path is simpler and the latency cost (waiting for the AppView round-trip before the post visibly disappears or appears) is small. The synthetic response on create makes the mutation appear within hundreds of milliseconds, and a delete confirmation already has a natural "loading…" affordance. Revisit if real users complain.

### Typed atproto identifiers via `atproto.dart`

Use `atproto.dart`'s `DID`, `Handle`, `AtUri`, `Cid` typed wrappers instead of plain `String` throughout the model.

**Rejected** for this slice. AGENTS.md flags them as the preferred approach, but the current Flutter side has zero `atproto.dart` usage — adopting it via this spec would mean either making `Profile` divergent (plain strings) or migrating it too, both of which expand scope past the agreed slice. A future "atproto.dart adoption" spec converts the whole client at once.

### `DummyPostRepository` for design-preview parity

Match `DummyProfileRepository` so the app can run against canned posts without an AppView.

**Rejected** for now. `DummyProfileRepository` exists in `lib/` but isn't wired into anything — it's an opt-in dev convenience that adds ~80 lines. We can add the equivalent for posts when (or if) someone needs it.

### Typed `Facet` model

Mirror the lexicon's `app.bsky.richtext.facet` shape — `index: ByteSlice`, `features: List<FeatureUnion>` — instead of the raw-JSON pass-through.

**Rejected** for v1. No client code renders rich text; committing to a typed shape now would churn when the renderer lands. The pass-through preserves round-trip data (a post created elsewhere with facets read back unchanged) without locking in the model.

## Consequences

### Code changes required

- New: `app/lib/feed/data/post_api_client.dart`, `post_repository.dart`, `api_post_repository.dart`.
- New: `app/lib/feed/models/post.dart`, `post_page.dart`, `user_posts_state.dart` (each with generated `*.mapper.dart`).
- New: `app/lib/feed/providers/post_api_client_provider.dart`, `post_repository_provider.dart`, `post_provider.dart`, `user_posts_provider.dart`, `create_post_provider.dart`, `delete_post_provider.dart` (each with generated `*.g.dart`).
- New: `app/test/feed/` mirroring the file layout above, plus `app/test/feed/fakes/fake_post_repository.dart`.
- Modified: nothing in `lib/`. No widgets, pages, routes, or wiring code is touched.

### Migration path

None. No existing call sites depend on a `Post` model that doesn't exist yet, and `PlaceholderPost` is left in place.

### Performance and storage

- **Initial profile-posts load:** one HTTP round-trip; AppView returns a hydrated page (author embedded), so no follow-up identity calls.
- **`loadMore`:** one HTTP round-trip per page; in-memory list grows linearly. For a worst-case profile with 1000 posts at default page size 50, ~20 pages, ~25 KB worst-case per post — payload-bound rather than CPU-bound.
- **Cache update on create/delete:** O(n) over the in-memory list (filter or prepend). Fine for any realistic list size.
- **`copyWithPrevious` in `loadMore`:** allocation-free in the common case (`AsyncLoading` and `AsyncData` instances; no list copies until success).

### Risks

- **Pagination shape becomes the precedent.** This is the first paginated surface in the client; whatever ergonomic warts ship here will be replicated for timeline, search, and notifications. Mitigation: the `UserPostsState` + `loadMore` + `state.isLoading` shape is the standard Flutter/Riverpod 3.x idiom; deviation would be the warning sign, not adoption.
- **`AsyncValue.copyWithPrevious` semantics on error.** Subtle for someone unfamiliar with Riverpod 3.x — `AsyncError(...).copyWithPrevious(state)` returns an AsyncError whose `.value` getter returns the previous data. The test suite explicitly covers this (failure-then-state.value-still-returns-previous-list).
- **`prepend` dedupe relies on uri equality.** If a synthetic-response post and a firehose-driven version of the same post end up with different `cid` (would only happen on a bug in the indexer or AppView), only the uri match prevents double-insertion. The dedupe is defensive; the AppView spec already guarantees uri stability.
- **`DeletePost.delete(post:)` couples the mutation API to the full `Post` model.** A future caller that only has `(did, rkey)` (e.g. a "delete by URI" admin tool) would need either a second method or a way to look up the `Post` first. We don't anticipate this in v1; the UI consumer always has the `Post` in hand.

## Open questions

None at time of writing. Resolved during brainstorming:

- **Slice scope:** data layer only; widgets stay on `PlaceholderPost`.
- **Endpoint coverage:** all four AppView endpoints, including single-post read (`postProvider`) even though no UI calls it in v1.
- **Write surface:** `text` only on `createPost`; reply, quote, facets are read-side only.
- **Pagination shape:** cursor-accumulating `AsyncNotifier` with `loadMore()`.
- **Mutation shape:** standalone `CreatePost` and `DeletePost` notifiers, using `prepend` / `removeByRkey` cache helpers on the list notifier.
- **Loading-more UX signal:** `AsyncValue.isLoading` only — no parallel `isLoadingMore` boolean. `copyWithPrevious` preserves visible items.
- **`atproto.dart` adoption:** deferred. Plain `String` for all atproto identifiers.
- **`DummyPostRepository`:** deferred. Add when a consumer needs it.
