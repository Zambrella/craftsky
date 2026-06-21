import 'package:craftsky_app/search/data/search_api_client.dart';
import 'package:craftsky_app/search/data/search_repository.dart';
import 'package:craftsky_app/search/models/hashtag_search_page.dart';
import 'package:craftsky_app/search/models/profile_search_page.dart';
import 'package:craftsky_app/search/models/recent_search.dart';
import 'package:craftsky_app/search/models/search_post_page.dart';
import 'package:craftsky_app/search/models/search_sort.dart';
import 'package:craftsky_app/search/models/search_suggestions.dart';
import 'package:craftsky_app/search/models/top_hashtags.dart';

class ApiSearchRepository implements SearchRepository {
  const ApiSearchRepository(this._api);

  final SearchApiClient _api;

  @override
  Future<SearchSuggestions> searchSuggestions({
    required String q,
    List<SearchSuggestionType>? types,
    int? profileLimit,
    int? hashtagLimit,
  }) => _api.searchSuggestions(
    q: q,
    types: types,
    profileLimit: profileLimit,
    hashtagLimit: hashtagLimit,
  );

  @override
  Future<HashtagSearchPage> searchHashtags({
    required String q,
    int? limit,
    String? cursor,
  }) => _api.searchHashtags(q: q, limit: limit, cursor: cursor);

  @override
  Future<SearchPostPage> searchHashtagPosts(
    String tag, {
    SearchSort? sort,
    int? limit,
    String? cursor,
  }) => _api.searchHashtagPosts(tag, sort: sort, limit: limit, cursor: cursor);

  @override
  Future<ProfileSearchPage> searchProfiles({
    required String q,
    int? limit,
    String? cursor,
  }) => _api.searchProfiles(q: q, limit: limit, cursor: cursor);

  @override
  Future<SearchPostPage> searchPosts({
    required String q,
    int? limit,
    String? cursor,
  }) => _api.searchPosts(q: q, limit: limit, cursor: cursor);

  @override
  Future<SearchPostPage> searchProjects({
    required String q,
    int? limit,
    String? cursor,
  }) => _api.searchProjects(q: q, limit: limit, cursor: cursor);

  @override
  Future<TopHashtagsResponse> topHashtags({
    List<String>? craftTypes,
    int? limit,
  }) => _api.topHashtags(craftTypes: craftTypes, limit: limit);

  @override
  Future<RecentSearchPage> listRecentSearches() => _api.listRecentSearches();

  @override
  Future<RecentSearchItem> saveRecentSearch(SaveRecentSearchRequest request) =>
      _api.saveRecentSearch(request);

  @override
  Future<void> deleteRecentSearch(String id) => _api.deleteRecentSearch(id);
}
