import 'package:craftsky_app/search/models/recent_search.dart';
import 'package:craftsky_app/search/models/top_hashtags.dart';
import 'package:dart_mappable/dart_mappable.dart';

part 'blank_search_data.mapper.dart';

@MappableClass()
class BlankSearchData with BlankSearchDataMappable {
  const BlankSearchData({
    required this.recentSearches,
    required this.topHashtags,
  });

  final RecentSearchPage recentSearches;
  final TopHashtagsResponse topHashtags;
}
