import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/providers/error_mapping_interceptor.dart';
import 'package:craftsky_app/shared/errors/app_error_mapper.dart';
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

    test(
      'SIM-T01 client diagnostics retain codes and correlation '
      'without server prose or paths',
      () {
        const ErrorMappingInterceptor().onError(
          _ex(
            status: 422,
            path: '/v1/posts/did:plc:alice/post1?cursor=private-cursor',
            data: {
              'error': 'validation_failed',
              'message': 'validation failed',
              'requestId': 'req_validation',
              'fields': {
                'text': 'must not be empty',
                r'embed.video.blob.ref.$link': 'must be a canonical CID',
                'privateTarget': 'opaque private target',
                'textValue': 'opaque private draft',
              },
            },
          ),
          handler,
        );
        final mapped = AppErrorMapper.map(handler.error!);
        expect(mapped.safeDiagnostics, containsPair('httpMethod', 'GET'));
        expect(
          mapped.safeDiagnostics,
          containsPair('appViewError', 'validation_failed'),
        );
        expect(
          mapped.safeDiagnostics,
          containsPair('appViewRequestId', 'req_validation'),
        );
        for (final key in [
          'appViewMessage',
          'validationFields',
          'routePattern',
          'incomingPath',
        ]) {
          expect(mapped.safeDiagnostics, isNot(contains(key)));
        }
        expect(
          mapped.safeDiagnostics.toString(),
          isNot(contains('opaque private')),
        );
        expect(
          mapped.safeDiagnostics.toString(),
          isNot(contains('private-cursor')),
        );
        for (final status in [400, 401, 404, 422, 500, 503]) {
          handler = _CapturingHandler();
          const ErrorMappingInterceptor().onError(
            _ex(
              status: status,
              data: {
                'error': 'internal_error',
                'message': 'opaque private body',
                'requestId': 'req_matrix',
                'fields': {'text': 'opaque private'},
              },
            ),
            handler,
          );
          final mapped = AppErrorMapper.map(handler.error!);
          expect(
            mapped.safeDiagnostics,
            containsPair('appViewRequestId', 'req_matrix'),
          );
          expect(
            mapped.safeDiagnostics.toString(),
            isNot(contains('opaque private')),
          );
        }
      },
    );
    test(
      'UT-009 typed underlying parse cause and supplied stack survive mapping',
      () {
        const cause = FormatException('opaque private response');
        final stack = StackTrace.fromString(
          '#0 decodePublicRecord (package:craftsky_app/post.dart:10:2)',
        );
        final dioError = DioException(
          requestOptions: RequestOptions(path: '/v1/posts/did:plc:alice/post1'),
          error: cause,
          stackTrace: stack,
        );
        const ErrorMappingInterceptor().onError(dioError, handler);
        final mapped = AppErrorMapper.map(handler.error!);
        expect(mapped.reportable, isTrue);
        expect(mapped.sentryClassification, 'parse.failed');
        expect(mapped.diagnosticCause, same(cause));
        expect(mapped.diagnosticStack, same(stack));
      },
    );
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

    test('retains string field validation errors for 4xx responses only', () {
      const ErrorMappingInterceptor().onError(
        _ex(
          status: 422,
          data: {
            'error': 'validation_failed',
            'fields': {'tagline': 'is invalid', 'bad': 42},
          },
        ),
        handler,
      );
      expect((handler.error! as ApiBadRequest).details.fields, {
        'tagline': 'is invalid',
      });
      final server = _CapturingHandler();
      const ErrorMappingInterceptor().onError(
        _ex(
          status: 500,
          data: {
            'fields': {'tagline': 'private internals'},
          },
        ),
        server,
      );
      expect((server.error! as ApiServerError).details.fields, isEmpty);
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
          path: '/v1/feed/timeline?cursor=secret',
        ),
        handler,
      );

      final error = handler.error as ApiServerError?;
      expect(error?.details.statusCode, 500);
      expect(error?.details.appViewError, 'internal_error');
      expect(error?.details.requestId, 'req_123');
      expect(error?.details.method, 'GET');
      expect(error?.message, isNot(contains('database failed')));
      expect(error?.message, isNot(contains('did:plc:alice')));
    });

    test(
      'registration 502 errors retain bounded codes and safe diagnostics',
      () {
        const providerText =
            'provider-error auth-code par-uri access-token '
            'refresh-token dpop-key';
        for (final code in <String>[
          'registration_provider_unavailable',
          'registration_incomplete',
        ]) {
          handler = _CapturingHandler();
          const ErrorMappingInterceptor().onError(
            _ex(
              status: 502,
              data: <String, dynamic>{
                'error': code,
                'message': providerText,
                'requestId': 'req_registration',
              },
              path: '/v1/auth/registrations?code=auth-code',
            ),
            handler,
          );

          final error = handler.error as ApiServerError?;
          expect(error?.details.appViewError, code);
          expect(error?.details.requestId, 'req_registration');
          expect(
            error?.details.method,
            'GET',
          );
          expect(
            AppErrorMapper.map(error!).safeDiagnostics['appViewMessage'],
            isNull,
          );
          expect(error.toString(), isNot(contains(providerText)));
          expect(error.toString(), isNot(contains('auth-code')));
        }
      },
    );

    test('UT-014 redacts complete language preference values', () {
      const ErrorMappingInterceptor().onError(
        _ex(
          status: 500,
          data: <String, dynamic>{
            'error': 'internal_error',
            'message': 'failed primaryLanguage=fr contentLanguages=[fr,en,cy]',
            'requestId': 'req_languages',
          },
          path: '/v1/languages/preferences',
        ),
        handler,
      );

      final error = handler.error as ApiServerError?;
      expect(
        error?.details.method,
        'GET',
      );
      expect(error.toString(), isNot(contains('primaryLanguage')));
      expect(error.toString(), isNot(contains('[fr,en,cy]')));
    });

    test(
      'SIM-T01 resource paths never become client diagnostic categories',
      () {
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
          (
            path: '/v1/posts/did:plc:alice/rkey-secret/saves',
            category: 'appview.posts.saves',
          ),
          (
            path: '/v1/saved-posts?folderId=folder-secret&cursor=cursor-secret',
            category: 'appview.saved_posts',
          ),
          (
            path: '/v1/saved-post-folders',
            category: 'appview.saved_post_folders',
          ),
          (
            path: '/v1/saved-post-folders/folder-secret',
            category: 'appview.saved_post_folders.detail',
          ),
          (
            path: '/v1/languages/preferences/initialize',
            category: 'appview.languages.preferences.initialize',
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
            error?.details.method,
            'GET',
            reason: testCase.path,
          );
          expect(
            error?.details.method,
            isNot(anyOf(contains('alice'), contains('secret'))),
          );
        }
      },
    );

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
