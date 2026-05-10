import 'package:craftsky_app/feed/models/post.dart';
import 'package:dart_mappable/dart_mappable.dart';

part 'post_thread.mapper.dart';

@MappableClass()
class PostThread with PostThreadMappable {
  const PostThread({
    required this.post,
    required this.replies,
    required this.truncated,
  });

  final Post post;
  final List<PostThread> replies;
  final bool truncated;
}
