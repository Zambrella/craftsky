import 'package:craftsky_app/search/models/search_sort.dart';
import 'package:craftsky_app/search/models/search_suggestions.dart';
import 'package:dart_mappable/dart_mappable.dart';

part 'search_queries.mapper.dart';

@MappableClass()
class SearchSuggestionQuery with SearchSuggestionQueryMappable {
  const new({
    required this.q,
    this.types = const [],
    this.profileLimit,
    this.hashtagLimit,
  });

  final String q;
  final List<SearchSuggestionType> types;
  final int? profileLimit;
  final int? hashtagLimit;
}

@MappableClass()
class HashtagSearchQuery with HashtagSearchQueryMappable {
  const new({
    required this.tag,
    this.sort = SearchSort.chronological,
  });

  final String tag;
  final SearchSort sort;
}

@MappableClass()
class HashtagResultSearchQuery with HashtagResultSearchQueryMappable {
  const new({required this.q});

  final String q;
}

@MappableClass()
class ProfileSearchQuery with ProfileSearchQueryMappable {
  const new({required this.q});

  final String q;
}

@MappableClass()
class PostSearchQuery with PostSearchQueryMappable {
  const new({required this.q});

  final String q;
}

@MappableClass()
class ProjectSearchQuery with ProjectSearchQueryMappable {
  const new({required this.q});

  final String q;
}

@MappableClass()
class TopHashtagsQuery with TopHashtagsQueryMappable {
  const new({this.craftTypes = const [], this.limit});

  final List<String> craftTypes;
  final int? limit;
}
