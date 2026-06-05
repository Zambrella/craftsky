import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/rich_text/data/facet_suggestion_repository.dart';
import 'package:dio/dio.dart';
import 'package:logging/logging.dart';

final _log = Logger('AppViewFacetSuggestionRepository');

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
      return _decodeSuggestionItems(
        data: res.data,
        endpoint: '/v1/facets/mentions',
        decode: AccountSuggestionMapper.fromMap,
      );
    } on ApiException catch (error, stackTrace) {
      _log.warning('mention suggestions API error', error, stackTrace);
      return const [];
    } on DioException catch (error, stackTrace) {
      _log.warning('mention suggestions network error', error, stackTrace);
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
      if (did is String && did.isNotEmpty) return did;
      _log.warning(
        'unexpected /v1/facets/mentions/resolve response shape: '
        'did=${did.runtimeType}',
      );
      return null;
    } on ApiBadRequest catch (error, stackTrace) {
      if (error.code == 'mention_not_found') return null;
      _log.warning('mention resolve API error', error, stackTrace);
      return null;
    } on ApiException catch (error, stackTrace) {
      _log.warning('mention resolve API error', error, stackTrace);
      return null;
    } on DioException catch (error, stackTrace) {
      _log.warning('mention resolve network error', error, stackTrace);
      return null;
    }
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
      return _decodeSuggestionItems(
        data: res.data,
        endpoint: '/v1/facets/hashtags',
        decode: HashtagSuggestionMapper.fromMap,
      );
    } on ApiException catch (error, stackTrace) {
      _log.warning('hashtag suggestions API error', error, stackTrace);
      return const [];
    } on DioException catch (error, stackTrace) {
      _log.warning('hashtag suggestions network error', error, stackTrace);
      return const [];
    }
  }
}

List<T> _decodeSuggestionItems<T>({
  required Map<String, dynamic>? data,
  required String endpoint,
  required T Function(Map<String, dynamic> item) decode,
}) {
  final items = data?['items'];
  if (items is! List) {
    _log.warning(
      'unexpected $endpoint response shape: items=${items.runtimeType}',
    );
    return const [];
  }

  final decoded = <T>[];
  for (final (index, item) in items.indexed) {
    if (item is! Map<String, dynamic>) {
      _log.warning(
        'unexpected $endpoint item shape at index $index: ${item.runtimeType}',
      );
      continue;
    }

    try {
      decoded.add(decode(item));
    } on Object catch (error, stackTrace) {
      _log.warning(
        'failed to decode $endpoint item at index $index',
        error,
        stackTrace,
      );
    }
  }
  return decoded;
}
