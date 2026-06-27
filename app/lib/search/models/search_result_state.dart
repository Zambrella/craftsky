import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/search/models/hashtag_search_page.dart';
import 'package:craftsky_app/search/models/profile_search_page.dart';
import 'package:dart_mappable/dart_mappable.dart';

part 'search_result_state.mapper.dart';

@MappableClass()
class SearchPostResultsState with SearchPostResultsStateMappable {
  const SearchPostResultsState({
    required this.items,
    this.cursor,
    this.hashtag,
  });

  final List<Post> items;
  final String? cursor;
  final String? hashtag;

  bool get hasMore => cursor != null;
  @override
  String toString() =>
      'SearchPostResultsState(items: ${items.length}, hasMore: $hasMore)';
}

@MappableClass()
class ProfileSearchResultsState with ProfileSearchResultsStateMappable {
  const ProfileSearchResultsState({required this.items, this.cursor});

  final List<ProfileSearchResult> items;
  final String? cursor;

  bool get hasMore => cursor != null;

  @override
  String toString() =>
      'ProfileSearchResultsState(items: ${items.length}, hasMore: $hasMore)';
}

@MappableClass()
class HashtagSearchResultsState with HashtagSearchResultsStateMappable {
  const HashtagSearchResultsState({required this.items, this.cursor});

  final List<HashtagSearchResult> items;
  final String? cursor;

  bool get hasMore => cursor != null;

  @override
  String toString() =>
      'HashtagSearchResultsState(items: ${items.length}, hasMore: $hasMore)';
}
