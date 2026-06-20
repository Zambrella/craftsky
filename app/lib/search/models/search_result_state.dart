import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/search/models/profile_search_page.dart';

class SearchPostResultsState {
  const SearchPostResultsState({
    required this.items,
    this.cursor,
    this.hashtag,
  });

  final List<Post> items;
  final String? cursor;
  final String? hashtag;

  bool get hasMore => cursor != null;

  SearchPostResultsState copyWith({
    List<Post>? items,
    String? cursor,
    String? hashtag,
  }) => SearchPostResultsState(
    items: items ?? this.items,
    cursor: cursor ?? this.cursor,
    hashtag: hashtag ?? this.hashtag,
  );

  @override
  String toString() =>
      'SearchPostResultsState(items: ${items.length}, hasMore: $hasMore)';
}

class ProfileSearchResultsState {
  const ProfileSearchResultsState({required this.items, this.cursor});

  final List<ProfileSearchResult> items;
  final String? cursor;

  bool get hasMore => cursor != null;

  @override
  String toString() =>
      'ProfileSearchResultsState(items: ${items.length}, hasMore: $hasMore)';
}
