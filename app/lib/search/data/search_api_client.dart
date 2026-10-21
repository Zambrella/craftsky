import 'package:craftsky_app/search/models/hashtag_search_page.dart';
import 'package:craftsky_app/search/models/profile_search_page.dart';
import 'package:craftsky_app/search/models/recent_search.dart';
import 'package:craftsky_app/search/models/search_post_page.dart';
import 'package:craftsky_app/search/models/search_sort.dart';
import 'package:craftsky_app/search/models/search_suggestions.dart';
import 'package:craftsky_app/search/models/top_hashtags.dart';
import 'package:craftsky_app/shared/api/api_unwrap.dart';
import 'package:craftsky_app/shared/observability/diagnostic_failure.dart';
import 'package:dio/dio.dart';

/// Search-related AppView endpoints. Assumes the attached [Dio] has the
/// auth + error interceptors installed; each call is wrapped in [unwrapApi].
class SearchApiClient {
  const new(this._dio);

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
    try {
      return SearchSuggestionsMapper.fromMap(res.data!);
    } on Object catch (error, stack) {
      Error.throwWithStackTrace(
        DiagnosticFormatException(
          _hashtagDecodeReason(res.data?['hashtags']),
          diagnosticCause: error,
          diagnosticStack: stack,
        ),
        stack,
      );
    }
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
    try {
      return HashtagSearchPageMapper.fromMap(res.data!);
    } on Object catch (error, stack) {
      Error.throwWithStackTrace(
        DiagnosticFormatException(
          _hashtagDecodeReason(res.data),
          diagnosticCause: error,
          diagnosticStack: stack,
        ),
        stack,
      );
    }
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

// Select schema facts only. Search text and mapper messages stay private.
String _hashtagDecodeReason(Object? section) {
  if (section is! Map) return 'Search hashtag section missing or invalid';
  final items = section['items'];
  if (items is! List) return 'Search hashtag items missing or invalid';
  for (final item in items) {
    if (item is! Map) return 'Search hashtag item is not an object';
    if (item['tag'] is! String) return 'Search hashtag tag missing or invalid';
    if (!item.containsKey('postCount')) {
      return 'Search hashtag postCount missing';
    }
    if (item['postCount'] == null) return 'Search hashtag postCount is null';
    if (item['postCount'] is! int) {
      return 'Search hashtag postCount is not an integer';
    }
  }
  return 'Search response decoding failed with valid hashtag fields';
}
