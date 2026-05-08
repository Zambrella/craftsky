import 'package:craftsky_app/feed/models/post.dart';
import 'package:dart_mappable/dart_mappable.dart';

part 'post_page.mapper.dart';

/// Envelope for paginated post responses from the AppView. `cursor` is
/// opaque to clients; pass it back to advance pagination. `cursor` is
/// absent (not `null`, not `""`) on the wire when there are no more
/// pages — `dart_mappable` maps absence and `null` to the same Dart
/// `null`, and re-encoding drops the key when null.
@MappableClass(ignoreNull: true)
class PostPage with PostPageMappable {
  const PostPage({required this.items, this.cursor});

  final List<Post> items;
  final String? cursor;
}
