import 'package:craftsky_app/feed/models/post.dart';
import 'package:dart_mappable/dart_mappable.dart';

part 'timeline_state.mapper.dart';

/// State held by the authenticated home timeline provider.
///
/// `cursor` is the opaque next cursor returned by AppView; `null` means the
/// server has no more pages. Loading and error state live on the surrounding
/// `AsyncValue`, matching the existing profile-post provider pattern.
@MappableClass()
class TimelineState with TimelineStateMappable {
  const TimelineState({required this.items, this.cursor});

  final List<Post> items;
  final String? cursor;

  bool get hasMore => cursor != null;

  @override
  String toString() {
    return 'TimelineState(items: ${items.length}, hasMore: $hasMore)';
  }
}
