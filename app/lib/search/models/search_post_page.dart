import 'package:craftsky_app/feed/models/post.dart';
import 'package:dart_mappable/dart_mappable.dart';

part 'search_post_page.mapper.dart';

@MappableClass(ignoreNull: true)
class SearchPostPage with SearchPostPageMappable {
  const SearchPostPage({required this.items, this.cursor, this.hashtag});

  final List<Post> items;
  final String? cursor;
  final String? hashtag;
}
