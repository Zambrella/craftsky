import 'package:craftsky_app/search/models/search_sort.dart';
import 'package:craftsky_app/search/models/search_suggestions.dart';
import 'package:dart_mappable/dart_mappable.dart';

part 'search_queries.mapper.dart';

@MappableClass()
class SearchSuggestionQuery with SearchSuggestionQueryMappable {
  const SearchSuggestionQuery({
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
  const HashtagSearchQuery({
    required this.tag,
    this.sort = SearchSort.chronological,
  });

  final String tag;
  final SearchSort sort;
}

@MappableClass()
class HashtagResultSearchQuery with HashtagResultSearchQueryMappable {
  const HashtagResultSearchQuery({required this.q});

  final String q;
}

@MappableClass()
class ProfileSearchQuery with ProfileSearchQueryMappable {
  const ProfileSearchQuery({required this.q});

  final String q;
}

@MappableClass()
class PostSearchQuery with PostSearchQueryMappable {
  const PostSearchQuery({required this.q});

  final String q;
}

@MappableClass()
class ProjectSearchQuery with ProjectSearchQueryMappable {
  const ProjectSearchQuery({required this.q});

  final String q;
}

@MappableClass()
class TopHashtagsQuery with TopHashtagsQueryMappable {
  const TopHashtagsQuery({this.craftTypes = const [], this.limit});

  final List<String> craftTypes;
  final int? limit;
}
