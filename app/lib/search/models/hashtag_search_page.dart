import 'package:dart_mappable/dart_mappable.dart';

part 'hashtag_search_page.mapper.dart';

@MappableClass(ignoreNull: true)
class HashtagSearchPage with HashtagSearchPageMappable {
  const new({required this.items, this.cursor});

  final List<HashtagSearchResult> items;
  final String? cursor;
}

@MappableClass()
class HashtagSearchResult with HashtagSearchResultMappable {
  const new({
    required this.tag,
    required this.postsLast28Days,
  });

  final String tag;
  final int postsLast28Days;
}
