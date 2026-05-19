import 'package:craftsky_app/feed/models/post.dart';
import 'package:dart_mappable/dart_mappable.dart';

part 'user_posts_state.mapper.dart';

/// State held by the `UserPosts` notifier (the cursor-accumulating
/// list-by-author provider). `cursor` is the *next* cursor to fetch
/// with — `null` once we've reached the end. `hasMore` is derived.
///
/// Loading state lives on the surrounding `AsyncValue`, not on this
/// class — the notifier sets
/// `state = AsyncLoading<UserPostsState>().copyWithPrevious(state)`
/// while a `loadMore` is in flight, which means `state.value` keeps
/// returning the previous (non-null) instance and `state.isLoading`
/// is `true`. UI consumers do `(value != null, isLoading)` pattern
/// matching to decide between "full-page spinner" and "list + bottom
/// spinner".
@MappableClass()
class UserPostsState with UserPostsStateMappable {
  const UserPostsState({required this.items, this.cursor});

  final List<Post> items;
  final String? cursor;

  bool get hasMore => cursor != null;

  @override
  String toString() {
    return 'UserPostsState(items: ${items.length}, hasMore: $hasMore)';
  }
}
