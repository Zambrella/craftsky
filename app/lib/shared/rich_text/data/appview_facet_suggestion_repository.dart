import 'dart:async';

import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/errors/app_error.dart';
import 'package:craftsky_app/shared/errors/app_error_mapper.dart';
import 'package:craftsky_app/shared/observability/diagnostic_emitter.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/rich_text/data/facet_suggestion_repository.dart';
import 'package:dio/dio.dart';
import 'package:logging/logging.dart';

final _log = Logger('AppViewFacetSuggestionRepository');

/// Dio-backed account suggestions from authenticated AppView facet endpoints.
class AppViewAccountSuggestionRepository
    implements AccountSuggestionRepository {
  /// Creates a repository using the app's authenticated Dio instance.
  const AppViewAccountSuggestionRepository(
    this._dio, {
    this.reporter = const NoopErrorReporter(),
  });

  final Dio _dio;
  final ErrorReporter reporter;

  @override
  Future<List<AccountSuggestion>> searchAccounts(String query) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/v1/facets/mentions',
        queryParameters: {'q': query, 'limit': 10},
      );
      return _decodeSuggestionItems(
        reporter: reporter,
        data: res.data,
        endpoint: '/v1/facets/mentions',
        decode: AccountSuggestionMapper.fromMap,
      );
    } on ApiException catch (error, stackTrace) {
      _logFacetFailure(
        reporter,
        'mention suggestions API error',
        '/v1/facets/mentions',
        'request',
        error,
        stackTrace,
      );
      return const [];
    } on DioException catch (error, stackTrace) {
      _logFacetFailure(
        reporter,
        'mention suggestions network error',
        '/v1/facets/mentions',
        'request',
        error,
        stackTrace,
      );
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
      _logFacetFailure(
        reporter,
        'facet response decode failed',
        '/v1/facets/mentions/resolve',
        'decode',
        const FormatException('Invalid resolver response'),
        StackTrace.current,
      );
      return null;
    } on ApiBadRequest catch (error, stackTrace) {
      if (error.code == 'mention_not_found') return null;
      _logFacetFailure(
        reporter,
        'mention resolve API error',
        '/v1/facets/mentions/resolve',
        'request',
        error,
        stackTrace,
      );
      return null;
    } on ApiException catch (error, stackTrace) {
      _logFacetFailure(
        reporter,
        'mention resolve API error',
        '/v1/facets/mentions/resolve',
        'request',
        error,
        stackTrace,
      );
      return null;
    } on DioException catch (error, stackTrace) {
      _logFacetFailure(
        reporter,
        'mention resolve network error',
        '/v1/facets/mentions/resolve',
        'request',
        error,
        stackTrace,
      );
      return null;
    }
  }
}

/// Dio-backed hashtag suggestions from authenticated AppView facet endpoints.
class AppViewHashtagSuggestionRepository
    implements HashtagSuggestionRepository {
  /// Creates a repository using the app's authenticated Dio instance.
  const AppViewHashtagSuggestionRepository(
    this._dio, {
    this.reporter = const NoopErrorReporter(),
  });

  final Dio _dio;
  final ErrorReporter reporter;

  @override
  Future<List<HashtagSuggestion>> searchHashtags(String query) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/v1/facets/hashtags',
        queryParameters: {'q': query, 'limit': 10},
      );
      return _decodeSuggestionItems(
        reporter: reporter,
        data: res.data,
        endpoint: '/v1/facets/hashtags',
        decode: HashtagSuggestionMapper.fromMap,
      );
    } on ApiException catch (error, stackTrace) {
      _logFacetFailure(
        reporter,
        'hashtag suggestions API error',
        '/v1/facets/hashtags',
        'request',
        error,
        stackTrace,
      );
      return const [];
    } on DioException catch (error, stackTrace) {
      _logFacetFailure(
        reporter,
        'hashtag suggestions network error',
        '/v1/facets/hashtags',
        'request',
        error,
        stackTrace,
      );
      return const [];
    }
  }
}

List<T> _decodeSuggestionItems<T>({
  required ErrorReporter reporter,
  required Map<String, dynamic>? data,
  required String endpoint,
  required T Function(Map<String, dynamic> item) decode,
}) {
  final items = data?['items'];
  if (items is! List) {
    _logFacetFailure(
      reporter,
      'facet response decode failed',
      endpoint,
      'decode',
      const FormatException('Invalid suggestions response'),
      StackTrace.current,
    );
    return const [];
  }

  final decoded = <T>[];
  Object? failure;
  StackTrace? failureStack;
  var invalidCount = 0;
  for (final item in items) {
    if (item is! Map<String, dynamic>) {
      invalidCount++;
      failure = const FormatException('Invalid suggestion item');
      failureStack = StackTrace.current;
      continue;
    }
    try {
      decoded.add(decode(item));
    } on Object catch (error, stackTrace) {
      invalidCount++;
      failure = error;
      failureStack = stackTrace;
    }
  }
  if (failure != null) {
    _logFacetFailure(
      reporter,
      'facet response decode failed',
      endpoint,
      'decode',
      failure,
      failureStack!,
      itemCount: invalidCount,
    );
  }
  return decoded;
}

String _facetOperation(String endpoint) => switch (endpoint) {
  '/v1/facets/mentions' => 'facet.mentions.search',
  '/v1/facets/mentions/resolve' => 'facet.mentions.resolve',
  '/v1/facets/hashtags' => 'facet.hashtags.search',
  _ => 'facet.decode',
};
void _logFacetFailure(
  ErrorReporter reporter,
  String message,
  String endpoint,
  String stage,
  Object error,
  StackTrace stackTrace, {
  int? itemCount,
}) {
  final mapped = AppErrorMapper.map(
    error,
    fallbackKind: AppErrorKind.backgroundLoadFailed,
    source: 'facet',
    fallbackClassification: 'facet.failed',
  );
  final offline =
      error is DioException &&
      const {
        DioExceptionType.connectionError,
        DioExceptionType.connectionTimeout,
        DioExceptionType.sendTimeout,
        DioExceptionType.receiveTimeout,
        DioExceptionType.cancel,
      }.contains(error.type) &&
      error.error is! FormatException;
  final context = ReportContext(
    feature: 'FacetSuggestions',
    operation: _facetOperation(endpoint),
    classification: stage == 'decode'
        ? 'parse.failed'
        : mapped.sentryClassification,
    outcome: offline ? DiagnosticOutcome.expected : DiagnosticOutcome.automatic,
    safeDiagnostics: {
      ...mapped.safeDiagnostics,
      'failureStage': stage,
      'itemCount': ?itemCount,
      'httpMethod': 'GET',
    },
  );
  final diagnostic = DiagnosticMessage(message, context: context);
  if (mapped.reportable && !offline) {
    _log.severe(diagnostic, error, stackTrace);
    unawaited(
      GuardedErrorReporter(
        reporter,
      ).captureException(error, stackTrace: stackTrace, context: context),
    );
  } else {
    _log.warning(diagnostic, error, stackTrace);
  }
}
