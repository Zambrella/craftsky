import 'package:craftsky_app/search/models/project_search_filters.dart';
import 'package:craftsky_app/search/models/search_sort.dart';
import 'package:dart_mappable/dart_mappable.dart';

part 'search_queries.mapper.dart';

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
class ProfileSearchQuery with ProfileSearchQueryMappable {
  const ProfileSearchQuery({required this.q});

  final String q;
}

@MappableClass()
class PostSearchQuery with PostSearchQueryMappable {
  const PostSearchQuery({
    required this.q,
    this.sort = SearchSort.chronological,
  });

  final String q;
  final SearchSort sort;
}

@MappableClass()
class ProjectSearchQuery with ProjectSearchQueryMappable {
  const ProjectSearchQuery({
    this.q,
    this.sort = SearchSort.chronological,
    this.filters = const ProjectSearchFilters(),
  });

  final String? q;
  final SearchSort sort;
  final ProjectSearchFilters filters;
}

@MappableClass()
class TopHashtagsQuery with TopHashtagsQueryMappable {
  const TopHashtagsQuery({this.craftTypes = const [], this.limit});

  final List<String> craftTypes;
  final int? limit;
}
