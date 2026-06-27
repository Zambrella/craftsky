import 'package:craftsky_app/search/models/hashtag_search_page.dart';
import 'package:craftsky_app/search/models/profile_search_page.dart';
import 'package:craftsky_app/search/models/recent_search.dart';
import 'package:craftsky_app/search/models/search_post_page.dart';
import 'package:craftsky_app/search/models/search_sort.dart';
import 'package:craftsky_app/search/models/search_suggestions.dart';
import 'package:craftsky_app/search/models/top_hashtags.dart';
import 'package:craftsky_app/shared/api/api_unwrap.dart';
import 'package:dio/dio.dart';

/// Search-related AppView endpoints. Assumes the attached [Dio] has the
/// auth + error interceptors installed; each call is wrapped in [unwrapApi].
class SearchApiClient {
  const SearchApiClient(this._dio);

  final Dio _dio;

  /// GET /v1/search/suggestions
  Future<SearchSuggestions> searchSuggestions({
    required String q,
    List<SearchSuggestionType>? types,
    int? profileLimit,
    int? hashtagLimit,
  }) => unwrapApi(() async {
    final res = await _dio.get<Map<String, dynamic>>(
      '/v1/search/suggestions',
      queryParameters: {
        'q': q,
        if (types != null && types.isNotEmpty)
          'types': types.map((type) => type.wireValue).join(','),
        'profileLimit': ?profileLimit?.toString(),
        'hashtagLimit': ?hashtagLimit?.toString(),
      },
    );
    return SearchSuggestionsMapper.fromMap(res.data!);
  });

  /// GET /v1/search/hashtags
  Future<HashtagSearchPage> searchHashtags({
    required String q,
    int? limit,
    String? cursor,
  }) => unwrapApi(() async {
    final res = await _dio.get<Map<String, dynamic>>(
      '/v1/search/hashtags',
      queryParameters: {'q': q, 'limit': ?limit?.toString(), 'cursor': ?cursor},
    );
    return HashtagSearchPageMapper.fromMap(res.data!);
  });

  /// GET /v1/search/hashtags/{tag}/posts
  Future<SearchPostPage> searchHashtagPosts(
    String tag, {
    SearchSort? sort,
    int? limit,
    String? cursor,
  }) => unwrapApi(() async {
    final encodedTag = Uri.encodeComponent(tag);
    final res = await _dio.get<Map<String, dynamic>>(
      '/v1/search/hashtags/$encodedTag/posts',
      queryParameters: {
        'sort': ?sort?.wireValue,
        'limit': ?limit?.toString(),
        'cursor': ?cursor,
      },
    );
    return SearchPostPageMapper.fromMap(res.data!);
  });

  /// GET /v1/search/profiles
  Future<ProfileSearchPage> searchProfiles({
    required String q,
    int? limit,
    String? cursor,
  }) => unwrapApi(() async {
    final res = await _dio.get<Map<String, dynamic>>(
      '/v1/search/profiles',
      queryParameters: {'q': q, 'limit': ?limit?.toString(), 'cursor': ?cursor},
    );
    return ProfileSearchPageMapper.fromMap(res.data!);
  });

  /// GET /v1/search/posts
  Future<SearchPostPage> searchPosts({
    required String q,
    int? limit,
    String? cursor,
  }) => unwrapApi(() async {
    final res = await _dio.get<Map<String, dynamic>>(
      '/v1/search/posts',
      queryParameters: {
        'q': q,
        'limit': ?limit?.toString(),
        'cursor': ?cursor,
      },
    );
    return SearchPostPageMapper.fromMap(res.data!);
  });

  /// GET /v1/search/projects
  Future<SearchPostPage> searchProjects({
    required String q,
    int? limit,
    String? cursor,
  }) => unwrapApi(() async {
    final res = await _dio.get<Map<String, dynamic>>(
      '/v1/search/projects',
      queryParameters: {
        'q': q,
        'limit': ?limit?.toString(),
        'cursor': ?cursor,
      },
    );
    return SearchPostPageMapper.fromMap(res.data!);
  });

  /// GET /v1/search/hashtags/top
  Future<TopHashtagsResponse> topHashtags({
    List<String>? craftTypes,
    int? limit,
  }) => unwrapApi(() async {
    final res = await _dio.get<Map<String, dynamic>>(
      '/v1/search/hashtags/top',
      queryParameters: {
        if (craftTypes != null && craftTypes.isNotEmpty)
          'craftTypes': craftTypes,
        'limit': ?limit?.toString(),
      },
      options: Options(listFormat: ListFormat.multi),
    );
    return TopHashtagsResponseMapper.fromMap(res.data!);
  });

  /// GET /v1/search/recent
  Future<RecentSearchPage> listRecentSearches() => unwrapApi(() async {
    final res = await _dio.get<Map<String, dynamic>>('/v1/search/recent');
    return RecentSearchPage.fromMap(res.data!);
  });

  /// POST /v1/search/recent
  Future<RecentSearchItem> saveRecentSearch(SaveRecentSearchRequest request) =>
      unwrapApi(() async {
        final res = await _dio.post<Map<String, dynamic>>(
          '/v1/search/recent',
          data: request.toMap(),
        );
        return RecentSearchItem.fromMap(res.data!);
      });

  /// DELETE /v1/search/recent/{id}
  Future<void> deleteRecentSearch(String id) => unwrapApi(() async {
    await _dio.delete<void>('/v1/search/recent/${Uri.encodeComponent(id)}');
  });
}
