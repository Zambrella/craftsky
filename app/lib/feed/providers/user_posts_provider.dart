import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/user_posts_state.dart';
import 'package:craftsky_app/feed/providers/author_post_cache.dart';
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
  /// On success, appends items and advances cursor. Riverpod preserves
  /// previous data across loading/error transitions so retry can use the
  /// same cursor after a load-more failure.
  Future<void> loadMore() async {
    if (!state.hasValue || state.isLoading) return;
    final current = state.requireValue;
    if (!current.hasMore) return;

    state = const AsyncLoading<UserPostsState>();

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
    state = next;
  }

  /// Cache helper. Inserts [post] at the head of the items list. No-op
  /// when the state has no data yet, or when a post with the same
  /// `uri` is already present (dedupe — protects against a synthetic
  /// create response and a later firehose-driven refresh both inserting
  /// the same row).
  void prepend(Post post) {
    final current = state.value;
    if (current == null) return;
    final items = prependPostIfAbsent(current.items, post);
    if (identical(items, current.items)) return;
    state = AsyncData(current.copyWith(items: items));
  }

  /// Cache helper. Removes the post with [rkey] from items if present.
  /// No-op when the state has no data, and quietly succeeds when no
  /// post matches.
  void removeByRkey(String rkey) {
    final current = state.value;
    if (current == null) return;
    state = AsyncData(
      current.copyWith(items: removePostByRkey(current.items, rkey)),
    );
  }

  /// Cache helper. Replaces a rendered post with [post] by stable URI or rkey.
  void replace(Post post) {
    final current = state.value;
    if (current == null) return;
    state = AsyncData(
      current.copyWith(items: replacePostByIdentity(current.items, post)),
    );
  }
}

void updateLiveUserPostCaches(Ref ref, Post post) {
  if (post.project != null) return;
  for (final id in authorPostCacheIds(post)) {
    if (ref.exists(userPostsProvider(id))) {
      ref.read(userPostsProvider(id).notifier).replace(post);
    }
  }
}
