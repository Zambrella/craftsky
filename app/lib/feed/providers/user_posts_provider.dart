import 'package:craftsky_app/feed/models/user_posts_state.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'user_posts_provider.g.dart';

/// Cursor-accumulating list-by-author provider, keyed by `handleOrDid`.
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
      final page = await repo.listByAuthor(handleOrDid, cursor: current.cursor);
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
}
