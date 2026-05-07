# Flutter Feed Post Data Layer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add the Flutter client data layer for the four AppView post CRUD endpoints (`POST /v1/posts`, `GET /v1/posts/{did}/{rkey}`, `DELETE /v1/posts/{did}/{rkey}`, `GET /v1/profiles/@{handleOrDid}/posts`) — model, API client, repository, and Riverpod providers (single-post read, paginated list-by-author, plus standalone `CreatePost` and `DeletePost` mutation notifiers). No widget or route changes.

**Architecture:** Three-layer split mirroring `app/lib/profile/` verbatim — `PostApiClient` (Dio wrapper) → `PostRepository` (interface + `ApiPostRepository` impl) → Riverpod providers. List pagination uses an `AsyncNotifier<UserPostsState>` family with `loadMore()`, leveraging `AsyncValue.copyWithPrevious` to keep the list visible across loading/error transitions (no parallel `isLoadingMore` flag). Mutation notifiers (`CreatePost`, `DeletePost`) are standalone idle-by-default singletons that, on success, push directly into live `userPostsProvider` family entries via `prepend(Post)` and `removeByRkey(String)` cache helpers — sidestepping the AppView's known read-after-write and read-after-delete inconsistency windows.

**Tech Stack:** Dart 3.11+, Flutter, `dart_mappable` 4.6 for models, `flutter_riverpod` 3.x with `riverpod_annotation`/`riverpod_generator` codegen, `dio` 5.7, `http_mock_adapter` 0.6 for HTTP tests. All new files live under `app/lib/feed/` and `app/test/feed/`.

**Spec:** [docs/superpowers/specs/2026-05-07-flutter-feed-post-data-layer-design.md](../specs/2026-05-07-flutter-feed-post-data-layer-design.md)

---

## Conventions used in this plan

- **Working directory** for all `dart` and `flutter` commands is `app/`. The `flutter` CLI must be run from there because `pubspec.yaml` lives there.
- **Codegen command:** `cd app && dart run build_runner build --delete-conflicting-outputs`. Runs after every change to a `dart_mappable` model or a `@riverpod`-annotated provider. Generated files (`*.mapper.dart`, `*.g.dart`) are committed.
- **Test command:** `cd app && flutter test <path>`. Examples below pass the path so we get fast per-file runs.
- **Imports:** package-style (`package:craftsky_app/...`), matching the rest of `app/lib/`. Test files use relative imports for sibling `fakes/` only when the path is short.
- **Commit style:** conventional-commits-ish ("feat(feed): ...", "test(feed): ...") matching recent history. Frequent commits — one per task.
- **Don't bypass hooks.** No `--no-verify` on commits.

---

## Task 0: Verify environment

**Files:** none (verification only)

- [ ] **Step 1: Confirm `app/` builds clean before starting**

Run from repo root:

```bash
cd app && flutter analyze && flutter test
```

Expected: 0 analyzer issues, all tests pass. Bail and report if not — anything failing here is unrelated to this plan and will mask our own breakage later.

- [ ] **Step 2: Confirm `dart_mappable_builder`, `riverpod_generator`, and `build_runner` are wired**

```bash
cd app && dart run build_runner --help
```

Expected: prints help (build / watch / serve subcommands). If the command is missing, `pub get` first: `cd app && dart pub get`.

---

## Task 1: `Post`, `PostAuthor`, `PostRef`, `PostReply` models

**Files:**
- Create: `app/lib/feed/models/post.dart`
- Create: `app/lib/feed/models/post.mapper.dart` *(generated)*
- Create: `app/test/feed/models/post_test.dart`

- [ ] **Step 1: Write the failing round-trip test**

Create `app/test/feed/models/post_test.dart`:

```dart
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  group('Post', () {
    test('round-trips a fully-populated wire payload', () {
      final json = {
        'uri': 'at://did:plc:alice/social.craftsky.feed.post/3lf2abc',
        'cid': 'bafy123',
        'rkey': '3lf2abc',
        'text': 'Cast on for the Hitchhiker shawl tonight.',
        'facets': [
          {
            'index': {'byteStart': 0, 'byteEnd': 7},
            'features': [
              {r'$type': 'app.bsky.richtext.facet#tag', 'tag': 'knitting'},
            ],
          },
        ],
        'tags': ['knitting'],
        'reply': {
          'root': {'uri': 'at://x/y/1', 'cid': 'bafyR'},
          'parent': {'uri': 'at://x/y/2', 'cid': 'bafyP'},
        },
        'quote': {'uri': 'at://x/y/q', 'cid': 'bafyQ'},
        'createdAt': '2026-05-04T18:23:45.000Z',
        'indexedAt': '2026-05-04T18:23:47.000Z',
        'author': {
          'did': 'did:plc:alice',
          'handle': 'alice.craftsky.social',
          'displayName': 'Alice',
          'avatarCid': 'bafyA',
        },
      };

      final post = PostMapper.fromMap(json);

      expect(post.uri, json['uri']);
      expect(post.text, json['text']);
      expect(post.tags, ['knitting']);
      expect(post.reply!.root.uri, 'at://x/y/1');
      expect(post.reply!.parent.cid, 'bafyP');
      expect(post.quote!.uri, 'at://x/y/q');
      expect(post.author.handle, 'alice.craftsky.social');
      expect(post.author.avatarCid, 'bafyA');

      expect(post.toMap(), json);
    });

    test('round-trips a minimal payload (optionals absent)', () {
      final json = {
        'uri': 'at://did:plc:alice/social.craftsky.feed.post/3lf2abc',
        'cid': 'bafy123',
        'rkey': '3lf2abc',
        'text': 'hello',
        'tags': <String>[],
        'createdAt': '2026-05-04T18:23:45.000Z',
        'indexedAt': '2026-05-04T18:23:47.000Z',
        'author': {
          'did': 'did:plc:alice',
          'handle': 'alice.craftsky.social',
        },
      };

      final post = PostMapper.fromMap(json);

      expect(post.facets, isNull);
      expect(post.reply, isNull);
      expect(post.quote, isNull);
      expect(post.author.displayName, isNull);
      expect(post.author.avatarCid, isNull);
      expect(post.tags, isEmpty);
      expect(post.toMap(), json);
    });
  });
}
```

- [ ] **Step 2: Run the test — expected: fail (model not defined)**

```bash
cd app && flutter test test/feed/models/post_test.dart
```

Expected: compilation failure ("Target of URI doesn't exist: 'package:craftsky_app/feed/models/post.dart'.")

- [ ] **Step 3: Create `app/lib/feed/models/post.dart`**

```dart
import 'package:dart_mappable/dart_mappable.dart';

part 'post.mapper.dart';

/// Wire shape for `social.craftsky.feed.post` records as returned by
/// the AppView's post endpoints. Author hydration (`{did, handle,
/// displayName, avatarCid}`) is embedded so the client can render a
/// feed without N+1 lookups.
///
/// `facets` is preserved as raw JSON (`List<Map<String, dynamic>>?`) —
/// no client code renders rich text yet, and the AppView treats facets
/// as a pass-through (lexicon-validated by the receiving PDS). A typed
/// `Facet` model lands when the richtext renderer does.
///
/// `images` is omitted from this model entirely — the v1 AppView
/// response shape does not include it.
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
  final List<Map<String, dynamic>>? facets;
  final List<String> tags;
  final PostReply? reply;
  final PostRef? quote;
  final DateTime createdAt;
  final DateTime indexedAt;
  final PostAuthor author;
}

/// Author identity embedded in every [Post] response.
///
/// `avatarCid` is a bare CID, not a URL — image proxying is its own
/// future spec.
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
  final String? avatarCid;
}

/// `(uri, cid)` reference to another atproto record. Used for reply
/// roots/parents and embedded quotes.
@MappableClass()
class PostRef with PostRefMappable {
  const PostRef({required this.uri, required this.cid});
  final String uri;
  final String cid;
}

/// Reply target, lexicon-shaped.
@MappableClass()
class PostReply with PostReplyMappable {
  const PostReply({required this.root, required this.parent});
  final PostRef root;
  final PostRef parent;
}
```

- [ ] **Step 4: Run codegen**

```bash
cd app && dart run build_runner build --delete-conflicting-outputs
```

Expected: generates `app/lib/feed/models/post.mapper.dart`. Output ends with "Succeeded after ... with X outputs (...)" and no warnings about `post.dart`.

- [ ] **Step 5: Run the test — expected: pass**

```bash
cd app && flutter test test/feed/models/post_test.dart
```

Expected: 2 tests, all pass.

- [ ] **Step 6: Commit**

```bash
git add app/lib/feed/models/post.dart app/lib/feed/models/post.mapper.dart app/test/feed/models/post_test.dart
git commit -m "feat(feed): add Post wire model with author/reply/quote nested types"
```

---

## Task 2: `PostPage` envelope

**Files:**
- Create: `app/lib/feed/models/post_page.dart`
- Create: `app/lib/feed/models/post_page.mapper.dart` *(generated)*
- Create: `app/test/feed/models/post_page_test.dart`

- [ ] **Step 1: Write the failing test**

Create `app/test/feed/models/post_page_test.dart`:

```dart
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  group('PostPage', () {
    test('round-trips with cursor present', () {
      final json = {
        'items': <Map<String, dynamic>>[],
        'cursor':
            'eyJpbmRleGVkQXQiOiIyMDI2LTA1LTA0VDE4OjIzOjQ3WiIsInVyaSI6ImF0Oi8vIn0',
      };

      final page = PostPageMapper.fromMap(json);
      expect(page.items, isEmpty);
      expect(page.cursor, json['cursor']);
      expect(page.toMap(), json);
    });

    test('absent cursor decodes as null and re-encodes without the key', () {
      final json = {'items': <Map<String, dynamic>>[]};

      final page = PostPageMapper.fromMap(json);
      expect(page.cursor, isNull);

      // Re-encoding omits the null cursor entirely (matches AppView's
      // pagination contract: `cursor` is omitted, not `null`, when no
      // more pages exist).
      expect(page.toMap(), {'items': <Map<String, dynamic>>[]});
    });
  });
}
```

- [ ] **Step 2: Run the test — expected: fail (file not found)**

```bash
cd app && flutter test test/feed/models/post_page_test.dart
```

Expected: compilation failure on `package:craftsky_app/feed/models/post_page.dart`.

- [ ] **Step 3: Create `app/lib/feed/models/post_page.dart`**

```dart
import 'package:craftsky_app/feed/models/post.dart';
import 'package:dart_mappable/dart_mappable.dart';

part 'post_page.mapper.dart';

/// Envelope for paginated post responses from the AppView. `cursor` is
/// opaque to clients; pass it back to advance pagination. `cursor` is
/// absent (not `null`, not `""`) on the wire when there are no more
/// pages — `dart_mappable` maps absence and `null` to the same Dart
/// `null`, and re-encoding drops the key when null.
@MappableClass()
class PostPage with PostPageMappable {
  const PostPage({required this.items, this.cursor});

  final List<Post> items;
  final String? cursor;
}
```

- [ ] **Step 4: Run codegen**

```bash
cd app && dart run build_runner build --delete-conflicting-outputs
```

Expected: generates `post_page.mapper.dart`.

- [ ] **Step 5: Run the test — expected: pass**

```bash
cd app && flutter test test/feed/models/post_page_test.dart
```

Expected: 2 tests pass.

- [ ] **Step 6: Commit**

```bash
git add app/lib/feed/models/post_page.dart app/lib/feed/models/post_page.mapper.dart app/test/feed/models/post_page_test.dart
git commit -m "feat(feed): add PostPage envelope for paginated post responses"
```

---

## Task 3: `UserPostsState` — provider state model

**Files:**
- Create: `app/lib/feed/models/user_posts_state.dart`
- Create: `app/lib/feed/models/user_posts_state.mapper.dart` *(generated)*
- Create: `app/test/feed/models/user_posts_state_test.dart`

- [ ] **Step 1: Write the failing test**

Create `app/test/feed/models/user_posts_state_test.dart`:

```dart
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/user_posts_state.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  group('UserPostsState', () {
    test('hasMore is true when cursor is non-null', () {
      const state = UserPostsState(items: [], cursor: 'abc');
      expect(state.hasMore, isTrue);
    });

    test('hasMore is false when cursor is null', () {
      const state = UserPostsState(items: []);
      expect(state.hasMore, isFalse);
    });

    test('copyWith preserves untouched fields', () {
      const state = UserPostsState(items: [], cursor: 'abc');
      final next = state.copyWith(cursor: null);
      expect(next.items, state.items);
      expect(next.cursor, isNull);
    });
  });
}
```

- [ ] **Step 2: Run the test — expected: fail (file not found)**

```bash
cd app && flutter test test/feed/models/user_posts_state_test.dart
```

- [ ] **Step 3: Create `app/lib/feed/models/user_posts_state.dart`**

```dart
import 'package:craftsky_app/feed/models/post.dart';
import 'package:dart_mappable/dart_mappable.dart';

part 'user_posts_state.mapper.dart';

/// State held by the `UserPosts` notifier (the cursor-accumulating
/// list-by-author provider). `cursor` is the *next* cursor to fetch
/// with — `null` once we've reached the end. `hasMore` is derived.
///
/// Loading state lives on the surrounding [AsyncValue], not on this
/// class — the notifier sets `state = AsyncLoading<UserPostsState>()
/// .copyWithPrevious(state)` while a `loadMore` is in flight, which
/// means `state.value` keeps returning the previous (non-null)
/// instance and `state.isLoading` is `true`. UI consumers do
/// `(value != null, isLoading)` pattern matching to decide between
/// "full-page spinner" and "list + bottom spinner".
@MappableClass()
class UserPostsState with UserPostsStateMappable {
  const UserPostsState({required this.items, this.cursor});

  final List<Post> items;
  final String? cursor;

  bool get hasMore => cursor != null;
}
```

- [ ] **Step 4: Run codegen**

```bash
cd app && dart run build_runner build --delete-conflicting-outputs
```

- [ ] **Step 5: Run the test — expected: pass**

```bash
cd app && flutter test test/feed/models/user_posts_state_test.dart
```

- [ ] **Step 6: Commit**

```bash
git add app/lib/feed/models/user_posts_state.dart app/lib/feed/models/user_posts_state.mapper.dart app/test/feed/models/user_posts_state_test.dart
git commit -m "feat(feed): add UserPostsState provider state model"
```

---

## Task 4: `PostApiClient`

**Files:**
- Create: `app/lib/feed/data/post_api_client.dart`
- Create: `app/test/feed/data/post_api_client_test.dart`

- [ ] **Step 1: Write the failing tests**

Create `app/test/feed/data/post_api_client_test.dart`:

```dart
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/data/post_api_client.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/providers/error_mapping_interceptor.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

void main() {
  setUpAll(initializeMappers);

  Dio buildDio() {
    return Dio(BaseOptions(baseUrl: 'https://appview.example.com'))
      ..interceptors.add(const ErrorMappingInterceptor());
  }

  Map<String, dynamic> samplePost({String text = 'hello'}) {
    return {
      'uri': 'at://did:plc:alice/social.craftsky.feed.post/3lf2abc',
      'cid': 'bafy123',
      'rkey': '3lf2abc',
      'text': text,
      'tags': <String>[],
      'createdAt': '2026-05-04T18:23:45.000Z',
      'indexedAt': '2026-05-04T18:23:47.000Z',
      'author': {
        'did': 'did:plc:alice',
        'handle': 'alice.craftsky.social',
      },
    };
  }

  group('PostApiClient.createPost', () {
    test('POSTs /v1/posts with text body and parses response', () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onPost(
        '/v1/posts',
        (server) => server.reply(201, samplePost(text: 'hi')),
        data: {'text': 'hi'},
      );

      final post = await PostApiClient(dio).createPost(text: 'hi');
      expect(post.text, 'hi');
      expect(post.rkey, '3lf2abc');
    });

    test('422 validation_failed surfaces as ApiBadRequest', () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onPost(
        '/v1/posts',
        (server) => server.reply(422, {'error': 'validation_failed'}),
        data: {'text': ''},
      );

      await expectLater(
        () => PostApiClient(dio).createPost(text: ''),
        throwsA(
          isA<ApiBadRequest>().having(
            (e) => e.code,
            'code',
            'validation_failed',
          ),
        ),
      );
    });
  });

  group('PostApiClient.getPost', () {
    test('GETs /v1/posts/{did}/{rkey} and parses', () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onGet(
        '/v1/posts/did:plc:alice/3lf2abc',
        (server) => server.reply(200, samplePost()),
      );

      final post = await PostApiClient(dio).getPost('did:plc:alice', '3lf2abc');
      expect(post.rkey, '3lf2abc');
    });

    test('404 surfaces as ApiBadRequest(post_not_found)', () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onGet(
        '/v1/posts/did:plc:alice/missing',
        (server) => server.reply(404, {'error': 'post_not_found'}),
      );

      await expectLater(
        () => PostApiClient(dio).getPost('did:plc:alice', 'missing'),
        throwsA(
          isA<ApiBadRequest>().having((e) => e.code, 'code', 'post_not_found'),
        ),
      );
    });
  });

  group('PostApiClient.deletePost', () {
    test('DELETEs /v1/posts/{did}/{rkey} and returns on 204', () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onDelete(
        '/v1/posts/did:plc:alice/3lf2abc',
        (server) => server.reply(204, null),
      );

      await PostApiClient(dio).deletePost('did:plc:alice', '3lf2abc');
    });

    test('403 forbidden surfaces as ApiBadRequest', () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onDelete(
        '/v1/posts/did:plc:bob/3lf2abc',
        (server) => server.reply(403, {'error': 'forbidden'}),
      );

      await expectLater(
        () => PostApiClient(dio).deletePost('did:plc:bob', '3lf2abc'),
        throwsA(
          isA<ApiBadRequest>().having((e) => e.code, 'code', 'forbidden'),
        ),
      );
    });
  });

  group('PostApiClient.listPostsByAuthor', () {
    test('GETs /v1/profiles/@{handleOrDid}/posts (no cursor)', () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onGet(
        '/v1/profiles/@alice.craftsky.social/posts',
        (server) => server.reply(200, {
          'items': [samplePost()],
          'cursor': 'next-cursor',
        }),
      );

      final page = await PostApiClient(dio).listPostsByAuthor(
        'alice.craftsky.social',
      );
      expect(page.items, hasLength(1));
      expect(page.cursor, 'next-cursor');
    });

    test('passes cursor and limit as query params', () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onGet(
        '/v1/profiles/@alice.craftsky.social/posts',
        (server) => server.reply(200, {'items': <Map<String, dynamic>>[]}),
        queryParameters: {'cursor': 'c1', 'limit': '50'},
      );

      final page = await PostApiClient(dio).listPostsByAuthor(
        'alice.craftsky.social',
        cursor: 'c1',
        limit: 50,
      );
      expect(page.items, isEmpty);
      expect(page.cursor, isNull);
    });
  });
}
```

- [ ] **Step 2: Run the tests — expected: fail (file not found)**

```bash
cd app && flutter test test/feed/data/post_api_client_test.dart
```

- [ ] **Step 3: Create `app/lib/feed/data/post_api_client.dart`**

```dart
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/shared/api/api_unwrap.dart';
import 'package:dio/dio.dart';

/// Post-related AppView endpoints. Assumes the attached [Dio] has the
/// auth + error interceptors installed (see `dioProvider`); each call
/// is wrapped in `unwrapApi` so consumers see sealed `ApiException`
/// subtypes instead of raw `DioException`s.
class PostApiClient {
  const PostApiClient(this._dio);

  final Dio _dio;

  /// POST /v1/posts — text-only create.
  ///
  /// AppView returns a synthetic [Post] populated from the PDS write
  /// response; the firehose-driven indexer hasn't necessarily caught
  /// up yet by the time this returns.
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
  ///
  /// `@`-prefix matches the existing convention in
  /// [ProfileApiClient.getProfile]; the AppView strips it before
  /// resolving. [limit] caps server-side at 100.
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

- [ ] **Step 4: Run the tests — expected: pass**

```bash
cd app && flutter test test/feed/data/post_api_client_test.dart
```

Expected: 8 tests pass.

- [ ] **Step 5: Commit**

```bash
git add app/lib/feed/data/post_api_client.dart app/test/feed/data/post_api_client_test.dart
git commit -m "feat(feed): add PostApiClient covering all four post endpoints"
```

---

## Task 5: `PostRepository` interface and `ApiPostRepository` impl

**Files:**
- Create: `app/lib/feed/data/post_repository.dart`
- Create: `app/lib/feed/data/api_post_repository.dart`

No dedicated tests — this layer is a thin pass-through over `PostApiClient`. Behavior is covered transitively by `PostApiClient` tests (above) and provider tests (below).

- [ ] **Step 1: Create `app/lib/feed/data/post_repository.dart`**

```dart
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';

/// Read/write surface the post providers depend on. The production
/// binding is `ApiPostRepository`; the test suite swaps in
/// `FakePostRepository` (under `test/feed/fakes/`) for unit tests.
abstract interface class PostRepository {
  /// POST /v1/posts. AppView returns a synthetic [Post] populated from
  /// the PDS write response.
  Future<Post> create({required String text});

  /// GET /v1/posts/{did}/{rkey}
  Future<Post> fetch(String did, String rkey);

  /// DELETE /v1/posts/{did}/{rkey}. Idempotent.
  Future<void> delete(String did, String rkey);

  /// GET /v1/profiles/@{handleOrDid}/posts — newest-first, paginated.
  Future<PostPage> listByAuthor(
    String handleOrDid, {
    String? cursor,
    int? limit,
  });
}
```

- [ ] **Step 2: Create `app/lib/feed/data/api_post_repository.dart`**

```dart
import 'package:craftsky_app/feed/data/post_api_client.dart';
import 'package:craftsky_app/feed/data/post_repository.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';

/// Production [PostRepository] backed by the AppView HTTP API.
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

- [ ] **Step 3: Run analyzer to confirm shape**

```bash
cd app && flutter analyze lib/feed/data/
```

Expected: 0 issues.

- [ ] **Step 4: Commit**

```bash
git add app/lib/feed/data/post_repository.dart app/lib/feed/data/api_post_repository.dart
git commit -m "feat(feed): add PostRepository interface and ApiPostRepository impl"
```

---

## Task 6: `FakePostRepository` for tests

**Files:**
- Create: `app/test/feed/fakes/fake_post_repository.dart`

No dedicated test — exercised via the provider tests in later tasks.

- [ ] **Step 1: Create `app/test/feed/fakes/fake_post_repository.dart`**

```dart
import 'package:craftsky_app/feed/data/post_repository.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';

/// Programmable [PostRepository] for unit tests. Each method delegates
/// to an optional callback the test sets up; unstubbed methods complete
/// with `UnimplementedError` so a test that misses a dependency fails
/// loudly instead of silently no-op'ing.
///
/// Mirrors [FakeProfileRepository].
///
/// Usage:
///
/// ```dart
/// final repo = FakePostRepository(
///   onListByAuthor: (id, {cursor, limit}) async => PostPage(items: [...]),
/// );
/// final container = ProviderContainer.test(
///   overrides: [postRepositoryProvider.overrideWithValue(repo)],
/// );
/// ```
class FakePostRepository implements PostRepository {
  FakePostRepository({
    this.onCreate,
    this.onFetch,
    this.onDelete,
    this.onListByAuthor,
  });

  final Future<Post> Function({required String text})? onCreate;
  final Future<Post> Function(String did, String rkey)? onFetch;
  final Future<void> Function(String did, String rkey)? onDelete;
  final Future<PostPage> Function(
    String handleOrDid, {
    String? cursor,
    int? limit,
  })?
  onListByAuthor;

  @override
  Future<Post> create({required String text}) =>
      onCreate?.call(text: text) ??
      Future<Post>.error(UnimplementedError('create not stubbed'));

  @override
  Future<Post> fetch(String did, String rkey) =>
      onFetch?.call(did, rkey) ??
      Future<Post>.error(UnimplementedError('fetch not stubbed'));

  @override
  Future<void> delete(String did, String rkey) =>
      onDelete?.call(did, rkey) ??
      Future<void>.error(UnimplementedError('delete not stubbed'));

  @override
  Future<PostPage> listByAuthor(
    String handleOrDid, {
    String? cursor,
    int? limit,
  }) =>
      onListByAuthor?.call(handleOrDid, cursor: cursor, limit: limit) ??
      Future<PostPage>.error(UnimplementedError('listByAuthor not stubbed'));
}
```

- [ ] **Step 2: Run analyzer to confirm shape**

```bash
cd app && flutter analyze test/feed/fakes/
```

Expected: 0 issues.

- [ ] **Step 3: Commit**

```bash
git add app/test/feed/fakes/fake_post_repository.dart
git commit -m "test(feed): add FakePostRepository programmable test fake"
```

---

## Task 7: Wrapper providers (`postApiClientProvider`, `postRepositoryProvider`)

**Files:**
- Create: `app/lib/feed/providers/post_api_client_provider.dart`
- Create: `app/lib/feed/providers/post_api_client_provider.g.dart` *(generated)*
- Create: `app/lib/feed/providers/post_repository_provider.dart`
- Create: `app/lib/feed/providers/post_repository_provider.g.dart` *(generated)*

No dedicated tests — these are mechanical wrappers (one line each). Exercised via the consumer providers in later tasks.

- [ ] **Step 1: Create `app/lib/feed/providers/post_api_client_provider.dart`**

```dart
import 'package:craftsky_app/feed/data/post_api_client.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'post_api_client_provider.g.dart';

@Riverpod(keepAlive: true)
PostApiClient postApiClient(Ref ref) =>
    PostApiClient(ref.watch(dioProvider));
```

- [ ] **Step 2: Create `app/lib/feed/providers/post_repository_provider.dart`**

```dart
import 'package:craftsky_app/feed/data/api_post_repository.dart';
import 'package:craftsky_app/feed/data/post_repository.dart';
import 'package:craftsky_app/feed/providers/post_api_client_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'post_repository_provider.g.dart';

@Riverpod(keepAlive: true)
PostRepository postRepository(Ref ref) =>
    ApiPostRepository(ref.watch(postApiClientProvider));
```

- [ ] **Step 3: Run codegen**

```bash
cd app && dart run build_runner build --delete-conflicting-outputs
```

Expected: generates `post_api_client_provider.g.dart` and `post_repository_provider.g.dart`.

- [ ] **Step 4: Run analyzer**

```bash
cd app && flutter analyze lib/feed/providers/
```

Expected: 0 issues.

- [ ] **Step 5: Commit**

```bash
git add app/lib/feed/providers/post_api_client_provider.dart \
        app/lib/feed/providers/post_api_client_provider.g.dart \
        app/lib/feed/providers/post_repository_provider.dart \
        app/lib/feed/providers/post_repository_provider.g.dart
git commit -m "feat(feed): add postApiClient and postRepository providers"
```

---

## Task 8: `postProvider(did, rkey)` — single-post read family

**Files:**
- Create: `app/lib/feed/providers/post_provider.dart`
- Create: `app/lib/feed/providers/post_provider.g.dart` *(generated)*
- Create: `app/test/feed/providers/post_provider_test.dart`

- [ ] **Step 1: Write the failing test**

Create `app/test/feed/providers/post_provider_test.dart`:

```dart
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/post_provider.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/fake_post_repository.dart';

void main() {
  setUpAll(initializeMappers);

  group('postProvider', () {
    test('returns the post fetched from the repository', () async {
      final fake = FakePostRepository(
        onFetch: (did, rkey) async => PostMapper.fromMap({
          'uri': 'at://did:plc:alice/social.craftsky.feed.post/$rkey',
          'cid': 'bafy123',
          'rkey': rkey,
          'text': 'hello',
          'tags': <String>[],
          'createdAt': '2026-05-04T18:23:45.000Z',
          'indexedAt': '2026-05-04T18:23:47.000Z',
          'author': {'did': did, 'handle': 'alice.craftsky.social'},
        }),
      );

      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      final post = await container.read(
        postProvider('did:plc:alice', '3lf2abc').future,
      );
      expect(post.rkey, '3lf2abc');
      expect(post.author.did, 'did:plc:alice');
    });

    test('propagates repository errors as AsyncError', () async {
      final fake = FakePostRepository(
        onFetch: (_, _) async => throw Exception('boom'),
      );

      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      await expectLater(
        container.read(postProvider('did:plc:alice', 'missing').future),
        throwsA(isA<Exception>()),
      );
    });
  });
}
```

- [ ] **Step 2: Run the test — expected: fail (file not found)**

```bash
cd app && flutter test test/feed/providers/post_provider_test.dart
```

- [ ] **Step 3: Create `app/lib/feed/providers/post_provider.dart`**

```dart
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'post_provider.g.dart';

/// Single-post read by `(did, rkey)`. No UI consumer in v1; exists for
/// future routes (deep-link share, thread page).
@riverpod
Future<Post> post(Ref ref, String did, String rkey) =>
    ref.watch(postRepositoryProvider).fetch(did, rkey);
```

- [ ] **Step 4: Run codegen**

```bash
cd app && dart run build_runner build --delete-conflicting-outputs
```

- [ ] **Step 5: Run the test — expected: pass**

```bash
cd app && flutter test test/feed/providers/post_provider_test.dart
```

Expected: 2 tests pass.

- [ ] **Step 6: Commit**

```bash
git add app/lib/feed/providers/post_provider.dart \
        app/lib/feed/providers/post_provider.g.dart \
        app/test/feed/providers/post_provider_test.dart
git commit -m "feat(feed): add postProvider for single-post read by (did, rkey)"
```

---

## Task 9: `userPostsProvider` — initial fetch (`build` only)

**Files:**
- Create: `app/lib/feed/providers/user_posts_provider.dart`
- Create: `app/lib/feed/providers/user_posts_provider.g.dart` *(generated)*
- Create: `app/test/feed/providers/user_posts_provider_test.dart`

This task adds the bare `UserPosts` notifier with only the `build` method. `loadMore`, `prepend`, and `removeByRkey` come in tasks 10 and 11 — separate commits keep the diffs reviewable.

- [ ] **Step 1: Write the failing test**

Create `app/test/feed/providers/user_posts_provider_test.dart`:

```dart
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/providers/user_posts_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/fake_post_repository.dart';

Map<String, dynamic> _samplePostMap({required String rkey, String? did}) => {
  'uri':
      'at://${did ?? 'did:plc:alice'}/social.craftsky.feed.post/$rkey',
  'cid': 'bafy_$rkey',
  'rkey': rkey,
  'text': 'post $rkey',
  'tags': <String>[],
  'createdAt': '2026-05-04T18:23:45.000Z',
  'indexedAt': '2026-05-04T18:23:47.000Z',
  'author': {
    'did': did ?? 'did:plc:alice',
    'handle': 'alice.craftsky.social',
  },
};

Post _samplePost({required String rkey, String? did}) =>
    PostMapper.fromMap(_samplePostMap(rkey: rkey, did: did));

void main() {
  setUpAll(initializeMappers);

  group('userPostsProvider build', () {
    test('first build fetches page 1 and surfaces items + cursor', () async {
      final fake = FakePostRepository(
        onListByAuthor: (id, {cursor, limit}) async => PostPage(
          items: [_samplePost(rkey: 'a'), _samplePost(rkey: 'b')],
          cursor: 'next',
        ),
      );

      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      final state = await container.read(
        userPostsProvider('alice.craftsky.social').future,
      );
      expect(state.items.map((p) => p.rkey), ['a', 'b']);
      expect(state.cursor, 'next');
      expect(state.hasMore, isTrue);
    });

    test('first build with empty page yields hasMore == false', () async {
      final fake = FakePostRepository(
        onListByAuthor: (id, {cursor, limit}) async =>
            const PostPage(items: []),
      );

      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      final state = await container.read(
        userPostsProvider('alice.craftsky.social').future,
      );
      expect(state.items, isEmpty);
      expect(state.cursor, isNull);
      expect(state.hasMore, isFalse);
    });
  });
}
```

- [ ] **Step 2: Run the test — expected: fail (file not found)**

```bash
cd app && flutter test test/feed/providers/user_posts_provider_test.dart
```

- [ ] **Step 3: Create `app/lib/feed/providers/user_posts_provider.dart`**

```dart
import 'package:craftsky_app/feed/models/user_posts_state.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'user_posts_provider.g.dart';

/// Cursor-accumulating list-by-author provider, keyed by `handleOrDid`.
///
/// `loadMore`, `prepend`, and `removeByRkey` are added in subsequent
/// commits. `build` fetches the first page only.
@riverpod
class UserPosts extends _$UserPosts {
  @override
  Future<UserPostsState> build(String handleOrDid) async {
    final repo = ref.watch(postRepositoryProvider);
    final page = await repo.listByAuthor(handleOrDid);
    return UserPostsState(items: page.items, cursor: page.cursor);
  }
}
```

- [ ] **Step 4: Run codegen**

```bash
cd app && dart run build_runner build --delete-conflicting-outputs
```

- [ ] **Step 5: Run the test — expected: pass**

```bash
cd app && flutter test test/feed/providers/user_posts_provider_test.dart
```

Expected: 2 tests pass.

- [ ] **Step 6: Commit**

```bash
git add app/lib/feed/providers/user_posts_provider.dart \
        app/lib/feed/providers/user_posts_provider.g.dart \
        app/test/feed/providers/user_posts_provider_test.dart
git commit -m "feat(feed): add userPostsProvider initial-fetch (build only)"
```

---

## Task 10: `userPostsProvider.loadMore`

**Files:**
- Modify: `app/lib/feed/providers/user_posts_provider.dart`
- Modify: `app/test/feed/providers/user_posts_provider_test.dart`

- [ ] **Step 1: Add the failing tests**

Append to `app/test/feed/providers/user_posts_provider_test.dart` (inside the existing `void main()` body, after the existing `group`):

```dart
  group('userPostsProvider loadMore', () {
    test('appends next page and advances cursor', () async {
      var call = 0;
      final fake = FakePostRepository(
        onListByAuthor: (id, {cursor, limit}) async {
          call++;
          if (call == 1) {
            return PostPage(items: [_samplePost(rkey: 'a')], cursor: 'c1');
          }
          expect(cursor, 'c1');
          return PostPage(items: [_samplePost(rkey: 'b')], cursor: null);
        },
      );

      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      // First build to populate the state.
      await container.read(
        userPostsProvider('alice.craftsky.social').future,
      );

      await container
          .read(userPostsProvider('alice.craftsky.social').notifier)
          .loadMore();

      final state = container
          .read(userPostsProvider('alice.craftsky.social'))
          .value!;
      expect(state.items.map((p) => p.rkey), ['a', 'b']);
      expect(state.cursor, isNull);
      expect(state.hasMore, isFalse);
    });

    test('no-op when hasMore is false', () async {
      var calls = 0;
      final fake = FakePostRepository(
        onListByAuthor: (id, {cursor, limit}) async {
          calls++;
          return const PostPage(items: []);
        },
      );

      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      await container.read(
        userPostsProvider('alice.craftsky.social').future,
      );
      expect(calls, 1);

      await container
          .read(userPostsProvider('alice.craftsky.social').notifier)
          .loadMore();
      expect(calls, 1, reason: 'loadMore must not call repo when !hasMore');
    });

    test('failure preserves visible items and cursor for retry', () async {
      var call = 0;
      final fake = FakePostRepository(
        onListByAuthor: (id, {cursor, limit}) async {
          call++;
          if (call == 1) {
            return PostPage(items: [_samplePost(rkey: 'a')], cursor: 'c1');
          }
          if (call == 2) {
            throw Exception('network down');
          }
          // Retry succeeds with the same cursor.
          expect(cursor, 'c1');
          return PostPage(items: [_samplePost(rkey: 'b')], cursor: null);
        },
      );

      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      await container.read(
        userPostsProvider('alice.craftsky.social').future,
      );

      // First loadMore fails.
      await container
          .read(userPostsProvider('alice.craftsky.social').notifier)
          .loadMore();

      final mid = container.read(userPostsProvider('alice.craftsky.social'));
      expect(mid.hasError, isTrue, reason: 'state is AsyncError after failure');
      expect(
        mid.value?.items.map((p) => p.rkey),
        ['a'],
        reason: 'previous data preserved via copyWithPrevious',
      );
      expect(mid.value?.cursor, 'c1', reason: 'cursor unchanged on failure');

      // Retry uses the same cursor and succeeds.
      await container
          .read(userPostsProvider('alice.craftsky.social').notifier)
          .loadMore();

      final after = container
          .read(userPostsProvider('alice.craftsky.social'))
          .value!;
      expect(after.items.map((p) => p.rkey), ['a', 'b']);
    });
  });
```

- [ ] **Step 2: Run the tests — expected: fail (loadMore method does not exist)**

```bash
cd app && flutter test test/feed/providers/user_posts_provider_test.dart
```

Expected: compilation error on `.notifier).loadMore()`.

- [ ] **Step 3: Add `loadMore` to `UserPosts`**

Replace the `UserPosts` class body in `app/lib/feed/providers/user_posts_provider.dart` with:

```dart
@riverpod
class UserPosts extends _$UserPosts {
  @override
  Future<UserPostsState> build(String handleOrDid) async {
    final repo = ref.watch(postRepositoryProvider);
    final page = await repo.listByAuthor(handleOrDid);
    return UserPostsState(items: page.items, cursor: page.cursor);
  }

  /// Append-next-page. No-op when:
  ///   - data hasn't loaded yet (`state.value == null`),
  ///   - we've reached the end (`!hasMore`),
  ///   - or a `loadMore` is already in flight (`state.isLoading`).
  ///
  /// On success, appends items and advances cursor. On failure, the
  /// state becomes [AsyncError] but `state.value` still returns the
  /// previous list (via `copyWithPrevious`) so the UI keeps showing
  /// items; the cursor is unchanged so a retry uses the same cursor.
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
}
```

- [ ] **Step 4: Run codegen**

The notifier surface didn't change (`@riverpod` annotation handles methods automatically), but it's safe to re-run:

```bash
cd app && dart run build_runner build --delete-conflicting-outputs
```

- [ ] **Step 5: Run the tests — expected: pass**

```bash
cd app && flutter test test/feed/providers/user_posts_provider_test.dart
```

Expected: 5 tests pass (2 from build + 3 from loadMore).

- [ ] **Step 6: Commit**

```bash
git add app/lib/feed/providers/user_posts_provider.dart \
        app/lib/feed/providers/user_posts_provider.g.dart \
        app/test/feed/providers/user_posts_provider_test.dart
git commit -m "feat(feed): add UserPosts.loadMore with copyWithPrevious-preserved retries"
```

---

## Task 11: `userPostsProvider.prepend` and `removeByRkey` cache helpers

**Files:**
- Modify: `app/lib/feed/providers/user_posts_provider.dart`
- Modify: `app/test/feed/providers/user_posts_provider_test.dart`

- [ ] **Step 1: Add the failing tests**

Append to `app/test/feed/providers/user_posts_provider_test.dart` (inside `void main()`):

```dart
  group('userPostsProvider prepend', () {
    test('inserts a new post at the head', () async {
      final fake = FakePostRepository(
        onListByAuthor: (id, {cursor, limit}) async =>
            PostPage(items: [_samplePost(rkey: 'a')]),
      );

      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      await container.read(
        userPostsProvider('alice.craftsky.social').future,
      );

      container
          .read(userPostsProvider('alice.craftsky.social').notifier)
          .prepend(_samplePost(rkey: 'new'));

      final state = container
          .read(userPostsProvider('alice.craftsky.social'))
          .value!;
      expect(state.items.map((p) => p.rkey), ['new', 'a']);
    });

    test('dedupes by uri', () async {
      final fake = FakePostRepository(
        onListByAuthor: (id, {cursor, limit}) async =>
            PostPage(items: [_samplePost(rkey: 'a')]),
      );

      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      await container.read(
        userPostsProvider('alice.craftsky.social').future,
      );

      // Same uri as 'a' — must not double-insert.
      container
          .read(userPostsProvider('alice.craftsky.social').notifier)
          .prepend(_samplePost(rkey: 'a'));

      final state = container
          .read(userPostsProvider('alice.craftsky.social'))
          .value!;
      expect(state.items.map((p) => p.rkey), ['a']);
    });
  });

  group('userPostsProvider removeByRkey', () {
    test('filters the matching post out of the list', () async {
      final fake = FakePostRepository(
        onListByAuthor: (id, {cursor, limit}) async => PostPage(
          items: [
            _samplePost(rkey: 'a'),
            _samplePost(rkey: 'b'),
            _samplePost(rkey: 'c'),
          ],
        ),
      );

      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      await container.read(
        userPostsProvider('alice.craftsky.social').future,
      );

      container
          .read(userPostsProvider('alice.craftsky.social').notifier)
          .removeByRkey('b');

      final state = container
          .read(userPostsProvider('alice.craftsky.social'))
          .value!;
      expect(state.items.map((p) => p.rkey), ['a', 'c']);
    });

    test('no-op when rkey not present', () async {
      final fake = FakePostRepository(
        onListByAuthor: (id, {cursor, limit}) async =>
            PostPage(items: [_samplePost(rkey: 'a')]),
      );

      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      await container.read(
        userPostsProvider('alice.craftsky.social').future,
      );

      container
          .read(userPostsProvider('alice.craftsky.social').notifier)
          .removeByRkey('not-here');

      final state = container
          .read(userPostsProvider('alice.craftsky.social'))
          .value!;
      expect(state.items.map((p) => p.rkey), ['a']);
    });
  });
```

- [ ] **Step 2: Run the tests — expected: fail (methods not defined)**

```bash
cd app && flutter test test/feed/providers/user_posts_provider_test.dart
```

- [ ] **Step 3: Add `prepend` and `removeByRkey` to `UserPosts`**

Append the following methods inside the `UserPosts` class in `app/lib/feed/providers/user_posts_provider.dart` (after `loadMore`):

```dart
  /// Cache helper. Inserts [post] at the head of the items list. No-op
  /// when the state has no data yet, or when a post with the same
  /// `uri` is already present (dedupe — protects against a synthetic
  /// create response and a later firehose-driven refresh both inserting
  /// the same row).
  void prepend(Post post) {
    final current = state.value;
    if (current == null) return;
    if (current.items.any((p) => p.uri == post.uri)) return;
    state = AsyncData(
      current.copyWith(items: [post, ...current.items]),
    );
  }

  /// Cache helper. Removes the post with [rkey] from items if present.
  /// No-op when the state has no data, and quietly succeeds when no
  /// post matches.
  void removeByRkey(String rkey) {
    final current = state.value;
    if (current == null) return;
    state = AsyncData(
      current.copyWith(
        items: current.items.where((p) => p.rkey != rkey).toList(),
      ),
    );
  }
```

You'll also need to add the missing import at the top of the file:

```dart
import 'package:craftsky_app/feed/models/post.dart';
```

- [ ] **Step 4: Run the tests — expected: pass**

```bash
cd app && flutter test test/feed/providers/user_posts_provider_test.dart
```

Expected: 9 tests pass (build × 2 + loadMore × 3 + prepend × 2 + removeByRkey × 2).

- [ ] **Step 5: Commit**

```bash
git add app/lib/feed/providers/user_posts_provider.dart \
        app/test/feed/providers/user_posts_provider_test.dart
git commit -m "feat(feed): add UserPosts prepend and removeByRkey cache helpers"
```

---

## Task 12: `CreatePost` notifier

**Files:**
- Create: `app/lib/feed/providers/create_post_provider.dart`
- Create: `app/lib/feed/providers/create_post_provider.g.dart` *(generated)*
- Create: `app/test/feed/providers/create_post_provider_test.dart`

- [ ] **Step 1: Write the failing test**

Create `app/test/feed/providers/create_post_provider_test.dart`:

```dart
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/providers/create_post_provider.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/providers/user_posts_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/fake_post_repository.dart';

Map<String, dynamic> _postMap({
  required String rkey,
  String did = 'did:plc:alice',
  String handle = 'alice.craftsky.social',
}) => {
  'uri': 'at://$did/social.craftsky.feed.post/$rkey',
  'cid': 'bafy_$rkey',
  'rkey': rkey,
  'text': 'post $rkey',
  'tags': <String>[],
  'createdAt': '2026-05-04T18:23:45.000Z',
  'indexedAt': '2026-05-04T18:23:47.000Z',
  'author': {'did': did, 'handle': handle},
};

Post _post({
  required String rkey,
  String did = 'did:plc:alice',
  String handle = 'alice.craftsky.social',
}) => PostMapper.fromMap(_postMap(rkey: rkey, did: did, handle: handle));

void main() {
  setUpAll(initializeMappers);

  group('CreatePost', () {
    test('idle build returns null', () async {
      final container = ProviderContainer.test(
        overrides: [
          postRepositoryProvider.overrideWithValue(FakePostRepository()),
        ],
      );

      final state = container.read(createPostProvider);
      expect(state.value, isNull);
      expect(state.isLoading, isFalse);
    });

    test('successful create transitions loading -> data(post)', () async {
      final fake = FakePostRepository(
        onCreate: ({required text}) async => _post(rkey: 'new'),
      );
      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      final transitions = <AsyncValue<Post?>>[];
      container.listen(createPostProvider, (_, next) => transitions.add(next));

      await container.read(createPostProvider.notifier).create(text: 'hi');

      expect(transitions.first, isA<AsyncLoading<Post?>>());
      expect(transitions.last.value?.rkey, 'new');
    });

    test('success prepends into live userPostsProvider entries '
        '(both did and handle keys)', () async {
      final fake = FakePostRepository(
        onListByAuthor: (id, {cursor, limit}) async =>
            PostPage(items: [_post(rkey: 'old')]),
        onCreate: ({required text}) async => _post(rkey: 'new'),
      );
      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      // Pre-instantiate both family entries so they are "live".
      await container.read(userPostsProvider('did:plc:alice').future);
      await container.read(
        userPostsProvider('alice.craftsky.social').future,
      );

      await container.read(createPostProvider.notifier).create(text: 'hi');

      final didEntry = container
          .read(userPostsProvider('did:plc:alice'))
          .value!;
      final handleEntry = container
          .read(userPostsProvider('alice.craftsky.social'))
          .value!;
      expect(didEntry.items.map((p) => p.rkey), ['new', 'old']);
      expect(handleEntry.items.map((p) => p.rkey), ['new', 'old']);
    });

    test('does not instantiate a non-live family entry', () async {
      final calls = <String>[];
      final fake = FakePostRepository(
        onListByAuthor: (id, {cursor, limit}) async {
          calls.add(id);
          return PostPage(items: [_post(rkey: 'x')]);
        },
        onCreate: ({required text}) async => _post(rkey: 'new'),
      );
      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      await container.read(createPostProvider.notifier).create(text: 'hi');

      expect(
        calls,
        isEmpty,
        reason:
            'CreatePost must not call ref.exists() in a way that '
            'auto-instantiates the family entry',
      );
    });

    test('reset() returns to AsyncData(null)', () async {
      final fake = FakePostRepository(
        onCreate: ({required text}) async => _post(rkey: 'new'),
      );
      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      await container.read(createPostProvider.notifier).create(text: 'hi');
      expect(container.read(createPostProvider).value?.rkey, 'new');

      container.read(createPostProvider.notifier).reset();
      expect(container.read(createPostProvider).value, isNull);
    });

    test('failure surfaces as AsyncError, no cache mutation', () async {
      final fake = FakePostRepository(
        onListByAuthor: (id, {cursor, limit}) async =>
            PostPage(items: [_post(rkey: 'old')]),
        onCreate: ({required text}) async => throw Exception('boom'),
      );
      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      await container.read(userPostsProvider('did:plc:alice').future);

      await container.read(createPostProvider.notifier).create(text: 'hi');

      expect(container.read(createPostProvider).hasError, isTrue);
      final list = container.read(userPostsProvider('did:plc:alice')).value!;
      expect(list.items.map((p) => p.rkey), ['old']);
    });
  });
}
```

- [ ] **Step 2: Run the tests — expected: fail (file not found)**

```bash
cd app && flutter test test/feed/providers/create_post_provider_test.dart
```

- [ ] **Step 3: Create `app/lib/feed/providers/create_post_provider.dart`**

```dart
import 'dart:async';

import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/providers/user_posts_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'create_post_provider.g.dart';

/// Standalone create-a-post mutation notifier. Idle until [create] runs,
/// then transitions `AsyncLoading` -> `AsyncData(post)` on success, or
/// `AsyncError` on failure.
///
/// On success, prepends the synthetic post into any live
/// `userPostsProvider` family entries keyed by either the author's
/// handle or DID — sidestepping the AppView's read-after-write window
/// (where a refetch could miss the just-created row until the firehose
/// indexer catches up). `ref.exists` guards against accidentally
/// instantiating a non-live family entry, which would race a fresh
/// `build` against our prepend.
///
/// Callers should bind via `ref.listen(createPostProvider, ...)` and
/// call [reset] after consuming a transition so a re-entry to the
/// compose page doesn't see the previous result.
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

      for (final id in <String>{post.author.handle, post.author.did}) {
        final entry = userPostsProvider(id);
        if (ref.exists(entry)) {
          ref.read(entry.notifier).prepend(post);
        }
      }

      return post;
    });
  }

  /// Resets the notifier to its idle state. Call after consuming a
  /// success/failure transition so a re-entry doesn't see prior result.
  void reset() => state = const AsyncData(null);
}
```

- [ ] **Step 4: Run codegen**

```bash
cd app && dart run build_runner build --delete-conflicting-outputs
```

- [ ] **Step 5: Run the tests — expected: pass**

```bash
cd app && flutter test test/feed/providers/create_post_provider_test.dart
```

Expected: 6 tests pass.

- [ ] **Step 6: Commit**

```bash
git add app/lib/feed/providers/create_post_provider.dart \
        app/lib/feed/providers/create_post_provider.g.dart \
        app/test/feed/providers/create_post_provider_test.dart
git commit -m "feat(feed): add CreatePost mutation notifier with cache prepend"
```

---

## Task 13: `DeletePost` notifier

**Files:**
- Create: `app/lib/feed/providers/delete_post_provider.dart`
- Create: `app/lib/feed/providers/delete_post_provider.g.dart` *(generated)*
- Create: `app/test/feed/providers/delete_post_provider_test.dart`

- [ ] **Step 1: Write the failing test**

Create `app/test/feed/providers/delete_post_provider_test.dart`:

```dart
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/providers/delete_post_provider.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/providers/user_posts_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/fake_post_repository.dart';

Map<String, dynamic> _postMap({
  required String rkey,
  String did = 'did:plc:alice',
  String handle = 'alice.craftsky.social',
}) => {
  'uri': 'at://$did/social.craftsky.feed.post/$rkey',
  'cid': 'bafy_$rkey',
  'rkey': rkey,
  'text': 'post $rkey',
  'tags': <String>[],
  'createdAt': '2026-05-04T18:23:45.000Z',
  'indexedAt': '2026-05-04T18:23:47.000Z',
  'author': {'did': did, 'handle': handle},
};

Post _post({
  required String rkey,
  String did = 'did:plc:alice',
  String handle = 'alice.craftsky.social',
}) => PostMapper.fromMap(_postMap(rkey: rkey, did: did, handle: handle));

void main() {
  setUpAll(initializeMappers);

  group('DeletePost', () {
    test('idle build returns null', () async {
      final container = ProviderContainer.test(
        overrides: [
          postRepositoryProvider.overrideWithValue(FakePostRepository()),
        ],
      );

      expect(container.read(deletePostProvider).value, isNull);
    });

    test('successful delete removes from live family entries '
        '(both did and handle keys)', () async {
      final deleted = <(String, String)>[];
      final fake = FakePostRepository(
        onListByAuthor: (id, {cursor, limit}) async => PostPage(
          items: [_post(rkey: 'a'), _post(rkey: 'b')],
        ),
        onDelete: (did, rkey) async {
          deleted.add((did, rkey));
        },
      );
      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      await container.read(userPostsProvider('did:plc:alice').future);
      await container.read(
        userPostsProvider('alice.craftsky.social').future,
      );

      await container
          .read(deletePostProvider.notifier)
          .delete(post: _post(rkey: 'a'));

      expect(deleted, [('did:plc:alice', 'a')]);

      final didList = container.read(userPostsProvider('did:plc:alice')).value!;
      final handleList = container
          .read(userPostsProvider('alice.craftsky.social'))
          .value!;
      expect(didList.items.map((p) => p.rkey), ['b']);
      expect(handleList.items.map((p) => p.rkey), ['b']);
    });

    test('failure surfaces as AsyncError, cache untouched', () async {
      final fake = FakePostRepository(
        onListByAuthor: (id, {cursor, limit}) async =>
            PostPage(items: [_post(rkey: 'a')]),
        onDelete: (did, rkey) async => throw Exception('boom'),
      );
      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      await container.read(userPostsProvider('did:plc:alice').future);

      await container
          .read(deletePostProvider.notifier)
          .delete(post: _post(rkey: 'a'));

      expect(container.read(deletePostProvider).hasError, isTrue);
      final list = container.read(userPostsProvider('did:plc:alice')).value!;
      expect(list.items.map((p) => p.rkey), ['a']);
    });

    test('reset() returns to AsyncData(null)', () async {
      final fake = FakePostRepository(
        onDelete: (did, rkey) async {},
      );
      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      await container
          .read(deletePostProvider.notifier)
          .delete(post: _post(rkey: 'a'));
      expect(container.read(deletePostProvider).value?.rkey, 'a');

      container.read(deletePostProvider.notifier).reset();
      expect(container.read(deletePostProvider).value, isNull);
    });
  });
}
```

- [ ] **Step 2: Run the tests — expected: fail (file not found)**

```bash
cd app && flutter test test/feed/providers/delete_post_provider_test.dart
```

- [ ] **Step 3: Create `app/lib/feed/providers/delete_post_provider.dart`**

```dart
import 'dart:async';

import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/providers/user_posts_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'delete_post_provider.g.dart';

/// Standalone delete-a-post mutation notifier. Takes the full [Post]
/// because the cache update needs `did`, `handle`, and `rkey` to splice
/// the post out of any live family entries (lists may be keyed by
/// either form). The caller — UI deleting a post it's already
/// rendering — has the [Post] in hand.
///
/// `build()` returns `Post?` so the `AsyncData(post)` transition
/// carries the deleted post for `ref.listen` consumers (e.g. an
/// "undo delete" snackbar).
///
/// On success, removes the post from any live `userPostsProvider`
/// family entries keyed by either the author's handle or DID,
/// sidestepping the AppView's read-after-delete window (where a
/// refetch could still include the just-deleted row until the firehose
/// tombstone arrives).
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

- [ ] **Step 4: Run codegen**

```bash
cd app && dart run build_runner build --delete-conflicting-outputs
```

- [ ] **Step 5: Run the tests — expected: pass**

```bash
cd app && flutter test test/feed/providers/delete_post_provider_test.dart
```

Expected: 4 tests pass.

- [ ] **Step 6: Commit**

```bash
git add app/lib/feed/providers/delete_post_provider.dart \
        app/lib/feed/providers/delete_post_provider.g.dart \
        app/test/feed/providers/delete_post_provider_test.dart
git commit -m "feat(feed): add DeletePost mutation notifier with cache splice"
```

---

## Task 14: Final verification

**Files:** none (verification only)

- [ ] **Step 1: Run analyzer over the whole project**

```bash
cd app && flutter analyze
```

Expected: 0 issues.

- [ ] **Step 2: Run the full feed test directory**

```bash
cd app && flutter test test/feed/
```

Expected: all tests pass. Approximate count: ~26 tests across 8 files (round-trip × 7, API client × 8, post provider × 2, user posts × 9, create × 6, delete × 4 — adjust as needed).

- [ ] **Step 3: Run the full app test suite to confirm no regressions**

```bash
cd app && flutter test
```

Expected: all tests pass, no new failures elsewhere.

- [ ] **Step 4: Verify formatting**

```bash
cd app && dart format --set-exit-if-changed lib/feed/ test/feed/
```

Expected: no changes needed (exit 0). If anything is reformatted, re-run without `--set-exit-if-changed`, stage, and amend the most recent task's commit:

```bash
cd app && dart format lib/feed/ test/feed/
git add -u
git commit --amend --no-edit
```

- [ ] **Step 5: Confirm the new files match the planned layout**

```bash
ls app/lib/feed/data/ app/lib/feed/models/ app/lib/feed/providers/ app/test/feed/
```

Expected output (subset — generated `.mapper.dart` and `.g.dart` files also present):

```
app/lib/feed/data/:
api_post_repository.dart
post_api_client.dart
post_repository.dart

app/lib/feed/models/:
placeholder_post.dart
post.dart
post_page.dart
user_posts_state.dart

app/lib/feed/providers/:
create_post_provider.dart
delete_post_provider.dart
post_api_client_provider.dart
post_provider.dart
post_repository_provider.dart
user_posts_provider.dart

app/test/feed/:
data
fakes
feed_page_test.dart
models
providers
```

- [ ] **Step 6: Confirm widgets are untouched**

```bash
git log --oneline -- app/lib/feed/widgets/ app/lib/feed/pages/ app/lib/profile/widgets/profile_tabs/ | head -5
```

Expected: no new commits in those paths from this branch (only pre-existing history). If the most recent commit touched any of those, something diverged from the spec — investigate before declaring done.

- [ ] **Step 7: Final commit (if anything was reformatted in step 4 and amended, this step is already done)**

If you reach this step with no further changes, the plan is complete. The branch is ready for code review.

---

## Self-review notes

- **Spec coverage:** every section of the spec is covered.
  - §1 file layout → tasks 1–13 (all 13 source files + 7 test files created).
  - §2 models → tasks 1, 2, 3.
  - §3 PostApiClient → task 4.
  - §4 PostRepository / ApiPostRepository → task 5.
  - §5 providers (wrappers + post + user posts + create + delete) → tasks 7, 8, 9, 10, 11, 12, 13.
  - §6 cache update strategy → tasks 11 (helpers), 12 (CreatePost calls them), 13 (DeletePost calls them).
  - §7 error handling → task 4 (PostApiClient tests verify ApiBadRequest mapping for 422/404/403); the rest is covered by the existing `ErrorMappingInterceptor` and tested in `app/test/shared/api/providers/error_mapping_interceptor_test.dart`.
  - §8 tests → coverage targets in tasks 1–13 cover every bullet from spec §8.
  - §9 wiring & dependencies → no `pubspec.yaml`, no router, no widgets touched (verified in task 14).
- **Placeholder scan:** none. Every step has concrete code or a concrete shell command.
- **Type consistency:** `prepend(Post)` and `removeByRkey(String)` are referenced consistently from tasks 11 (defined), 12 (called by CreatePost), and 13 (called by DeletePost). `delete({required Post post})` is consistent between task 13 (defined) and the spec. `userPostsProvider` family argument is `handleOrDid` everywhere. Notifier method names match the spec: `loadMore`, `prepend`, `removeByRkey`, `create`, `delete`, `reset`.
