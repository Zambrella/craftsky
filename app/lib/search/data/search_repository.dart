import 'package:craftsky_app/search/models/profile_search_page.dart';
import 'package:craftsky_app/search/models/project_search_filters.dart';
import 'package:craftsky_app/search/models/recent_search.dart';
import 'package:craftsky_app/search/models/search_post_page.dart';
import 'package:craftsky_app/search/models/search_sort.dart';
import 'package:craftsky_app/search/models/top_hashtags.dart';

abstract interface class SearchRepository {
  Future<SearchPostPage> searchHashtagPosts(
    String tag, {
    SearchSort? sort,
    int? limit,
    String? cursor,
  });

  Future<ProfileSearchPage> searchProfiles({
    required String q,
    int? limit,
    String? cursor,
  });

  Future<SearchPostPage> searchPosts({
    required String q,
    SearchSort? sort,
    int? limit,
    String? cursor,
  });

  Future<SearchPostPage> searchProjects({
    String? q,
    SearchSort? sort,
    ProjectSearchFilters? filters,
    int? limit,
    String? cursor,
  });

  Future<TopHashtagsResponse> topHashtags({
    List<String>? craftTypes,
    int? limit,
  });

  Future<RecentSearchPage> listRecentSearches();
  Future<RecentSearchItem> saveRecentSearch(SaveRecentSearchRequest request);
  Future<void> deleteRecentSearch(String id);
}
