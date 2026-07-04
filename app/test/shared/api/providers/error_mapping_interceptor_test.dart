import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/providers/error_mapping_interceptor.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';

DioException _ex({
  int? status,
  DioExceptionType type = DioExceptionType.badResponse,
  dynamic data,
  String path = '/v1/whoami',
}) {
  final req = RequestOptions(path: path);
  return DioException(
    requestOptions: req,
    type: type,
    response: status == null
        ? null
        : Response(requestOptions: req, statusCode: status, data: data),
  );
}

void main() {
  group('ErrorMappingInterceptor', () {
    late _CapturingHandler handler;

    setUp(() => handler = _CapturingHandler());

    test('401 → ApiUnauthorized', () {
      const ErrorMappingInterceptor().onError(_ex(status: 401), handler);
      expect(handler.error, isA<ApiUnauthorized>());
    });

    test('400 with {"error": "handle_required"} → ApiBadRequest(code)', () {
      const ErrorMappingInterceptor().onError(
        _ex(status: 400, data: <String, dynamic>{'error': 'handle_required'}),
        handler,
      );
      expect(handler.error, isA<ApiBadRequest>());
      expect((handler.error as ApiBadRequest?)?.code, 'handle_required');
      expect((handler.error as ApiBadRequest?)?.details.statusCode, 400);
      expect(
        (handler.error as ApiBadRequest?)?.details.appViewError,
        'handle_required',
      );
    });

    test('400 with no error field → ApiBadRequest(null)', () {
      const ErrorMappingInterceptor().onError(
        _ex(status: 400, data: <String, dynamic>{}),
        handler,
      );
      expect(handler.error, isA<ApiBadRequest>());
      expect((handler.error as ApiBadRequest?)?.code, isNull);
    });

    test('500 → ApiServerError', () {
      const ErrorMappingInterceptor().onError(_ex(status: 500), handler);
      expect(handler.error, isA<ApiServerError>());
      expect((handler.error as ApiServerError?)?.message, 'http_500');
      expect((handler.error as ApiServerError?)?.details.statusCode, 500);
    });

    test('extracts safe AppView diagnostics without backend message', () {
      const ErrorMappingInterceptor().onError(
        _ex(
          status: 500,
          data: <String, dynamic>{
            'error': 'internal_error',
            'message': 'database failed for did:plc:alice',
            'requestId': 'req_123',
          },
          path: '/v1/feed?cursor=secret',
        ),
        handler,
      );

      final error = handler.error as ApiServerError?;
      expect(error?.details.statusCode, 500);
      expect(error?.details.appViewError, 'internal_error');
      expect(error?.details.requestId, 'req_123');
      expect(error?.details.endpointCategory, 'appview.feed');
      expect(error?.message, isNot(contains('database failed')));
      expect(error?.message, isNot(contains('did:plc:alice')));
    });

    test('normalizes dynamic endpoint paths to allowlisted categories', () {
      final cases = <({String path, String category})>[
        (
          path: '/v1/posts/did:plc:alice/rkey-secret',
          category: 'appview.posts.detail',
        ),
        (
          path: '/v1/posts/did:plc:alice/rkey-secret/replies',
          category: 'appview.posts.replies',
        ),
        (
          path: '/v1/profiles/@alice.example',
          category: 'appview.profiles.detail',
        ),
        (
          path: '/v1/profiles/@alice.example/posts',
          category: 'appview.profiles.posts',
        ),
        (
          path: '/v1/search/hashtags/secret-tag/posts?cursor=hidden',
          category: 'appview.search.hashtag_posts',
        ),
        (
          path: '/v1/search/recent/recent-secret-id',
          category: 'appview.search.recent.detail',
        ),
      ];

      for (final testCase in cases) {
        handler = _CapturingHandler();
        const ErrorMappingInterceptor().onError(
          _ex(status: 500, path: testCase.path),
          handler,
        );

        final error = handler.error as ApiServerError?;
        expect(
          error?.details.endpointCategory,
          testCase.category,
          reason: testCase.path,
        );
        expect(
          error?.details.endpointCategory,
          isNot(anyOf(contains('alice'), contains('secret'))),
        );
      }
    });

    test('timeout → ApiNetworkError', () {
      const ErrorMappingInterceptor().onError(
        _ex(type: DioExceptionType.connectionTimeout),
        handler,
      );
      expect(handler.error, isA<ApiNetworkError>());
    });

    test('connection error → ApiNetworkError', () {
      const ErrorMappingInterceptor().onError(
        _ex(type: DioExceptionType.connectionError),
        handler,
      );
      expect(handler.error, isA<ApiNetworkError>());
    });
  });
}

class _CapturingHandler extends ErrorInterceptorHandler {
  Object? error;

  @override
  void next(DioException err) {
    error = err.error;
  }
}
