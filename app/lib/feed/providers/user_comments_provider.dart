import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/user_posts_state.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'user_comments_provider.g.dart';

const userCommentsPageLimit = 10;

/// Cursor-accumulating authored comments/replies list, keyed by `handleOrDid`.
@riverpod
class UserComments extends _$UserComments {
  static String formatLogValue(Object? value) => value.toString();

  @override
  Future<UserPostsState> build(String handleOrDid) async {
    final repo = ref.watch(postRepositoryProvider);
    final page = await repo.listCommentsByAuthor(
      handleOrDid,
      limit: userCommentsPageLimit,
    );
    return UserPostsState(items: page.items, cursor: page.cursor);
  }

  Future<void> loadMore() async {
    if (!state.hasValue || state.isLoading) return;
    final current = state.requireValue;
    if (!current.hasMore) return;
    final ownership = captureActiveAccountOperation(ref);

    state = const AsyncLoading<UserPostsState>();

    final next = await AsyncValue.guard(() async {
      final repo = ref.read(postRepositoryProvider);
      final page = await repo.listCommentsByAuthor(
        handleOrDid,
        cursor: current.cursor,
        limit: userCommentsPageLimit,
      );
      return UserPostsState(
        items: [...current.items, ...page.items],
        cursor: page.cursor,
      );
    });

    if (!isActiveAccountOperationCurrent(ref, ownership)) return;
    state = next;
  }

  void prependOrReplace(Post post) {
    final current = state.value;
    if (current == null) return;
    if (!current.items.any((item) => item.uri == post.uri)) {
      state = AsyncData(current.copyWith(items: [post, ...current.items]));
      return;
    }
    state = AsyncData(
      current.copyWith(
        items: [
          for (final item in current.items)
            if (item.uri == post.uri || item.rkey == post.rkey) post else item,
        ],
      ),
    );
  }

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

void updateLiveUserCommentCaches(Ref ref, Post post) {
  if (post.reply == null) return;
  for (final id in <String>{post.author.did, post.author.handle}) {
    if (ref.exists(userCommentsProvider(id))) {
      ref.read(userCommentsProvider(id).notifier).prependOrReplace(post);
    }
  }
}
