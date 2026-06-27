import 'package:craftsky_app/search/data/search_repository.dart';
import 'package:craftsky_app/search/models/hashtag_search_page.dart';
import 'package:craftsky_app/search/models/profile_search_page.dart';
import 'package:craftsky_app/search/models/recent_search.dart';
import 'package:craftsky_app/search/models/search_post_page.dart';
import 'package:craftsky_app/search/models/search_sort.dart';
import 'package:craftsky_app/search/models/search_suggestions.dart';
import 'package:craftsky_app/search/models/top_hashtags.dart';

class FakeSearchRepository implements SearchRepository {
  FakeSearchRepository({
    this.onSearchSuggestions,
    this.onSearchHashtags,
    this.onSearchHashtagPosts,
    this.onSearchProfiles,
    this.onSearchPosts,
    this.onSearchProjects,
    this.onTopHashtags,
    this.onListRecentSearches,
    this.onSaveRecentSearch,
    this.onDeleteRecentSearch,
  });

  final Future<SearchSuggestions> Function({
    required String q,
    List<SearchSuggestionType>? types,
    int? profileLimit,
    int? hashtagLimit,
  })?
  onSearchSuggestions;
  final Future<HashtagSearchPage> Function({
    required String q,
    int? limit,
    String? cursor,
  })?
  onSearchHashtags;

  final Future<SearchPostPage> Function(
    String tag, {
    SearchSort? sort,
    int? limit,
    String? cursor,
  })?
  onSearchHashtagPosts;
  final Future<ProfileSearchPage> Function({
    required String q,
    int? limit,
    String? cursor,
  })?
  onSearchProfiles;
  final Future<SearchPostPage> Function({
    required String q,
    int? limit,
    String? cursor,
  })?
  onSearchPosts;
  final Future<SearchPostPage> Function({
    required String q,
    int? limit,
    String? cursor,
  })?
  onSearchProjects;

  @override
  Future<SearchSuggestions> searchSuggestions({
    required String q,
    List<SearchSuggestionType>? types,
    int? profileLimit,
    int? hashtagLimit,
  }) =>
      onSearchSuggestions?.call(
        q: q,
        types: types,
        profileLimit: profileLimit,
        hashtagLimit: hashtagLimit,
      ) ??
      Future<SearchSuggestions>.error(
        UnimplementedError('searchSuggestions not stubbed'),
      );

  @override
  Future<HashtagSearchPage> searchHashtags({
    required String q,
    int? limit,
    String? cursor,
  }) =>
      onSearchHashtags?.call(q: q, limit: limit, cursor: cursor) ??
      Future<HashtagSearchPage>.error(
        UnimplementedError('searchHashtags not stubbed'),
      );
  final Future<TopHashtagsResponse> Function({
    List<String>? craftTypes,
    int? limit,
  })?
  onTopHashtags;
  final Future<RecentSearchPage> Function()? onListRecentSearches;
  final Future<RecentSearchItem> Function(SaveRecentSearchRequest request)?
  onSaveRecentSearch;
  final Future<void> Function(String id)? onDeleteRecentSearch;

  @override
  Future<SearchPostPage> searchHashtagPosts(
    String tag, {
    SearchSort? sort,
    int? limit,
    String? cursor,
  }) =>
      onSearchHashtagPosts?.call(
        tag,
        sort: sort,
        limit: limit,
        cursor: cursor,
      ) ??
      Future<SearchPostPage>.error(
        UnimplementedError('searchHashtagPosts not stubbed'),
      );

  @override
  Future<ProfileSearchPage> searchProfiles({
    required String q,
    int? limit,
    String? cursor,
  }) =>
      onSearchProfiles?.call(q: q, limit: limit, cursor: cursor) ??
      Future<ProfileSearchPage>.error(
        UnimplementedError('searchProfiles not stubbed'),
      );

  @override
  Future<SearchPostPage> searchPosts({
    required String q,
    int? limit,
    String? cursor,
  }) =>
      onSearchPosts?.call(q: q, limit: limit, cursor: cursor) ??
      Future<SearchPostPage>.error(
        UnimplementedError('searchPosts not stubbed'),
      );

  @override
  Future<SearchPostPage> searchProjects({
    required String q,
    int? limit,
    String? cursor,
  }) =>
      onSearchProjects?.call(
        q: q,
        limit: limit,
        cursor: cursor,
      ) ??
      Future<SearchPostPage>.error(
        UnimplementedError('searchProjects not stubbed'),
      );

  @override
  Future<TopHashtagsResponse> topHashtags({
    List<String>? craftTypes,
    int? limit,
  }) =>
      onTopHashtags?.call(craftTypes: craftTypes, limit: limit) ??
      Future<TopHashtagsResponse>.error(
        UnimplementedError('topHashtags not stubbed'),
      );

  @override
  Future<RecentSearchPage> listRecentSearches() =>
      onListRecentSearches?.call() ??
      Future<RecentSearchPage>.error(
        UnimplementedError('listRecentSearches not stubbed'),
      );

  @override
  Future<RecentSearchItem> saveRecentSearch(SaveRecentSearchRequest request) =>
      onSaveRecentSearch?.call(request) ??
      Future<RecentSearchItem>.error(
        UnimplementedError('saveRecentSearch not stubbed'),
      );

  @override
  Future<void> deleteRecentSearch(String id) =>
      onDeleteRecentSearch?.call(id) ??
      Future<void>.error(UnimplementedError('deleteRecentSearch not stubbed'));
}
