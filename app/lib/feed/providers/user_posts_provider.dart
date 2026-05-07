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
