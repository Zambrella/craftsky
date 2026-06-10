import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/user_posts_state.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'user_posts_provider.g.dart';

const userPostsPageLimit = 10;

/// Cursor-accumulating list-by-author provider, keyed by `handleOrDid`.
@riverpod
class UserPosts extends _$UserPosts {
  static String formatLogValue(Object? value) => value.toString();

  @override
  Future<UserPostsState> build(String handleOrDid) async {
    final repo = ref.watch(postRepositoryProvider);
    final page = await repo.listByAuthor(
      handleOrDid,
      limit: userPostsPageLimit,
    );
    return UserPostsState(items: page.items, cursor: page.cursor);
  }

  /// Append-next-page. No-op when:
  ///   - data hasn't loaded yet (`state.value == null`),
  ///   - we've reached the end (`!hasMore`),
  ///   - or a `loadMore` is already in flight (`state.isLoading`).
  ///
  /// On success, appends items and advances cursor. On failure, the
  /// state becomes `AsyncError` but `state.value` still returns the
  /// previous list (via `copyWithPrevious`) so the UI keeps showing
  /// items; the cursor is unchanged so a retry uses the same cursor.
  Future<void> loadMore() async {
    final current = state.value;
    if (current == null || !current.hasMore || state.isLoading) return;

    // copyWithPrevious is the only public-facing mechanism in Riverpod 3.x for
    // preserving previous data during a loading/error transition. The @internal
    // annotation is a package-boundary guard, not a stability concern; Riverpod
    // uses this pattern in its own generated code.
    // ignore: invalid_use_of_internal_member
    state = const AsyncLoading<UserPostsState>().copyWithPrevious(state);

    final next = await AsyncValue.guard(() async {
      final repo = ref.read(postRepositoryProvider);
      final page = await repo.listByAuthor(
        handleOrDid,
        cursor: current.cursor,
        limit: userPostsPageLimit,
      );
      return UserPostsState(
        items: [...current.items, ...page.items],
        cursor: page.cursor,
      );
    });

    if (!ref.mounted) return;
    // Same rationale as above: copyWithPrevious keeps previous items visible
    // when guard returns an AsyncError, enabling retry with the same cursor.
    // ignore: invalid_use_of_internal_member
    state = next.copyWithPrevious(state);
  }

  /// Cache helper. Inserts [post] at the head of the items list. No-op
  /// when the state has no data yet, or when a post with the same
  /// `uri` is already present (dedupe — protects against a synthetic
  /// create response and a later firehose-driven refresh both inserting
  /// the same row).
  void prepend(Post post) {
    final current = state.value;
    if (current == null) return;
    if (current.items.any((p) => p.uri == post.uri)) return;
    state = AsyncData(current.copyWith(items: [post, ...current.items]));
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

  /// Cache helper. Replaces a rendered post with [post] by stable URI or rkey.
  void replace(Post post) {
    final current = state.value;
    if (current == null) return;
    state = AsyncData(
      current.copyWith(
        items: [
          for (final item in current.items)
            if (item.uri == post.uri || item.rkey == post.rkey) post else item,
        ],
      ),
    );
  }
}

void updateLiveUserPostCaches(Ref ref, Post post) {
  if (post.project != null) return;
  for (final id in <String>{post.author.did, post.author.handle}) {
    if (ref.exists(userPostsProvider(id))) {
      ref.read(userPostsProvider(id).notifier).replace(post);
    }
  }
}
