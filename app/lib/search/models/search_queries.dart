import 'package:craftsky_app/search/models/project_search_filters.dart';
import 'package:craftsky_app/search/models/search_sort.dart';
import 'package:flutter/foundation.dart';

@immutable
class HashtagSearchQuery {
  const HashtagSearchQuery({
    required this.tag,
    this.sort = SearchSort.chronological,
  });

  final String tag;
  final SearchSort sort;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is HashtagSearchQuery && tag == other.tag && sort == other.sort;

  @override
  int get hashCode => Object.hash(tag, sort);
}

@immutable
class ProfileSearchQuery {
  const ProfileSearchQuery({required this.q});

  final String q;

  @override
  String toString() => 'ProfileSearchQuery(q: $q)';

  @override
  bool operator ==(Object other) =>
      identical(this, other) || other is ProfileSearchQuery && q == other.q;

  @override
  int get hashCode => q.hashCode;
}

@immutable
class PostSearchQuery {
  const PostSearchQuery({
    required this.q,
    this.sort = SearchSort.chronological,
  });

  final String q;
  final SearchSort sort;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is PostSearchQuery && q == other.q && sort == other.sort;

  @override
  int get hashCode => Object.hash(q, sort);
}

@immutable
class ProjectSearchQuery {
  const ProjectSearchQuery({
    this.q,
    this.sort = SearchSort.chronological,
    this.filters = const ProjectSearchFilters(),
  });

  final String? q;
  final SearchSort sort;
  final ProjectSearchFilters filters;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ProjectSearchQuery &&
          q == other.q &&
          sort == other.sort &&
          filters == other.filters;

  @override
  int get hashCode => Object.hash(q, sort, filters);
}

@immutable
class TopHashtagsQuery {
  const TopHashtagsQuery({this.craftTypes = const [], this.limit});

  final List<String> craftTypes;
  final int? limit;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is TopHashtagsQuery &&
          _listEquals(craftTypes, other.craftTypes) &&
          limit == other.limit;

  @override
  int get hashCode => Object.hash(Object.hashAll(craftTypes), limit);
}

bool _listEquals(List<String> a, List<String> b) {
  if (identical(a, b)) return true;
  if (a.length != b.length) return false;
  for (var i = 0; i < a.length; i++) {
    if (a[i] != b[i]) return false;
  }
  return true;
}
