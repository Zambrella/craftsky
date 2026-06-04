import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/rich_text/data/facet_suggestion_repository.dart';
import 'package:dio/dio.dart';

/// Dio-backed account suggestions from authenticated AppView facet endpoints.
class AppViewAccountSuggestionRepository
    implements AccountSuggestionRepository {
  /// Creates a repository using the app's authenticated Dio instance.
  const AppViewAccountSuggestionRepository(this._dio);

  final Dio _dio;

  @override
  Future<List<AccountSuggestion>> searchAccounts(String query) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/v1/facets/mentions',
        queryParameters: {'q': query, 'limit': 10},
      );
      final items = res.data?['items'];
      if (items is! List) return const [];
      return items
          .whereType<Map>()
          .map((item) => item.cast<String, dynamic>())
          .map(_accountFromMap)
          .toList();
    } on ApiException {
      return const [];
    } on DioException {
      return const [];
    }
  }

  @override
  Future<String?> didForHandle(String handle) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/v1/facets/mentions/resolve',
        queryParameters: {'handle': handle},
      );
      final did = res.data?['did'];
      return did is String && did.isNotEmpty ? did : null;
    } on ApiBadRequest catch (error) {
      if (error.code == 'mention_not_found') return null;
      return null;
    } on ApiException {
      return null;
    } on DioException {
      return null;
    }
  }

  AccountSuggestion _accountFromMap(Map<String, dynamic> item) {
    return AccountSuggestion(
      did: item['did'] as String,
      handle: item['handle'] as String,
      displayName: item['displayName'] as String?,
      avatar: item['avatar'] as String?,
      isCraftskyProfile: item['isCraftskyProfile'] as bool? ?? false,
      viewerIsFollowing: item['viewerIsFollowing'] as bool? ?? false,
    );
  }
}

/// Dio-backed hashtag suggestions from authenticated AppView facet endpoints.
class AppViewHashtagSuggestionRepository
    implements HashtagSuggestionRepository {
  /// Creates a repository using the app's authenticated Dio instance.
  const AppViewHashtagSuggestionRepository(this._dio);

  final Dio _dio;

  @override
  Future<List<HashtagSuggestion>> searchHashtags(String query) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/v1/facets/hashtags',
        queryParameters: {'q': query, 'limit': 10},
      );
      final items = res.data?['items'];
      if (items is! List) return const [];
      return items
          .whereType<Map>()
          .map((item) => item.cast<String, dynamic>())
          .map(
            (item) => HashtagSuggestion(
              tag: item['tag'] as String,
              postsLast28Days: item['postsLast28Days'] as int? ?? 0,
            ),
          )
          .toList();
    } on ApiException {
      return const [];
    } on DioException {
      return const [];
    }
  }
}
