import 'package:craftsky_app/feed/models/post.dart';
import 'package:dart_mappable/dart_mappable.dart';

part 'user_projects_state.mapper.dart';

@MappableClass()
class UserProjectsState with UserProjectsStateMappable {
  const UserProjectsState({
    required this.items,
    this.cursor,
    this.pinnedPostUri,
  });

  final List<Post> items;
  final String? cursor;
  final String? pinnedPostUri;

  bool get hasMore => cursor != null;

  @override
  String toString() {
    return 'UserProjectsState(items: ${items.length}, hasMore: $hasMore)';
  }
}
