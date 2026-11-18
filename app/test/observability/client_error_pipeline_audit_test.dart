import 'dart:async';

import 'package:craftsky_app/auth/models/auth_error.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/chat/data/chat_draft_store.dart';
import 'package:craftsky_app/feed/composer/composer_submission_coordinator.dart';
import 'package:craftsky_app/feed/data/video_service_client.dart';
import 'package:craftsky_app/service_status/data/service_status_repository.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/providers/error_mapping_interceptor.dart';
import 'package:craftsky_app/shared/errors/app_error.dart';
import 'package:craftsky_app/shared/errors/app_error_mapper.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/sentry_error_reporter.dart';
import 'package:craftsky_app/shared/observability/sentry_sanitizer.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_service_native.dart';
import 'package:dio/dio.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:sentry_flutter/sentry_flutter.dart';

import '../test_support/serialized_sentry_transport.dart';

const _requestId = '10000000-0000-4000-8000-000000000001';
final _causeStack = StackTrace.fromString(
  '#0 originalStorageOperation (package:craftsky_app/storage.dart:12:3)',
);

void main() {
  test('a reporting failure cannot escape the provider error owner', () async {
    final reporter = _ThrowingReporter();
    final escaped = <Object>[];
    final cause = _DependencyFailure();
    await runZonedGuarded(() async {
      final container = ProviderContainer(
        observers: [ProviderLogger(reporter: reporter)],
        retry: appProviderRetry,
      );
      addTearDown(container.dispose);
      final provider = FutureProvider<int>((ref) => throw cause);
      await expectLater(container.read(provider.future), throwsA(same(cause)));
      await Future<void>.delayed(Duration.zero);
    }, (error, stack) => escaped.add(error));
    expect(reporter.errors, [cause]);
    expect(escaped, isEmpty);
  });

  test('provider-wrapped API failures keep their useful UI classification', () {
    for (final sample in [
      (
        error: const ApiNetworkError('PRIVATE_NETWORK'),
        kind: AppErrorKind.networkUnavailable,
      ),
      (error: const ApiUnauthorized(), kind: AppErrorKind.sessionExpired),
    ]) {
      final container = ProviderContainer.test();
      final provider = Provider<Object>((ref) => throw sample.error);
      Object? wrapped;
      try {
        container.read(provider);
      } on Object catch (error) {
        wrapped = error;
      }
      expect(wrapped, isNotNull);
      final mapped = AppErrorMapper.map(wrapped!);
      expect(mapped.kind, sample.kind);
      expect(mapped.reportable, isFalse);
    }
  });

  test(
    'HTTP permission and missing-content responses have specific UI kinds',
    () {
      for (final sample in [
        (status: 403, kind: AppErrorKind.permissionDenied),
        (status: 404, kind: AppErrorKind.contentUnavailable),
        (status: 429, kind: AppErrorKind.rateLimited),
      ]) {
        final mapped = AppErrorMapper.map(
          ApiBadRequest(
            null,
            details: ApiFailureDetails(statusCode: sample.status),
          ),
        );
        expect(mapped.kind, sample.kind);
        expect(mapped.reportable, isFalse);
      }
    },
  );

  test('fallback mappings retain the original typed diagnostic cause', () {
    final cause = _DependencyFailure();
    final mapped = AppErrorMapper.map(cause, source: 'storage');
    expect(mapped.diagnosticCause, same(cause));
  });

  test(
    'mapped parse fallback keeps native cause and protected stack',
    () async {
      final transport = SerializedTransport();
      await Sentry.init((options) {
        configureDiagnosticOptions(options);
        options
          ..dsn = 'https://public@example.invalid/1'
          ..transport = transport;
      });
      addTearDown(Sentry.close);
      await const SentryErrorReporter().captureException(
        AppErrorMapper.map(const FormatException('PRIVATE_PARSE_PAYLOAD')),
        stackTrace: _causeStack,
        context: const ReportContext(
          feature: 'Audit',
          operation: 'load',
          classification: 'parse.failed',
        ),
      );
      await transport.issueReceived.future.timeout(const Duration(seconds: 5));
      final serialized = transport.payloads.join();
      expect(serialized, contains('FormatException'));
      expect(serialized, contains('originalStorageOperation'));
      expect(serialized, isNot(contains('PRIVATE_')));
    },
  );

  test(
    'unknown adapter failures are actionable instead of reported as offline',
    () {
      for (final type in [
        DioExceptionType.unknown,
        DioExceptionType.badCertificate,
      ]) {
        final original = DioException(
          requestOptions: RequestOptions(path: '/PRIVATE_PATH'),
          type: type,
          error: _DependencyFailure(),
          stackTrace: _causeStack,
        );
        final handler = _Handler();
        const ErrorMappingInterceptor().onError(original, handler);
        expect(handler.error, isA<ApiServerError>());
        final mapped = AppErrorMapper.map(handler.error!);
        expect(mapped.kind, AppErrorKind.serviceUnavailable);
        expect(mapped.reportable, isTrue);
        expect(handler.error!.details.cause, same(original.error));
        expect(handler.error!.details.stackTrace, same(_causeStack));
      }
    },
  );

  test(
    'invalid envelope correlation falls back to the valid server header',
    () {
      final request = RequestOptions(path: '/PRIVATE_PATH');
      final details = ErrorMappingInterceptor.detailsFor(
        DioException(
          requestOptions: request,
          response: Response<Object?>(
            requestOptions: request,
            statusCode: 500,
            headers: Headers.fromMap({
              'x-request-id': [_requestId],
            }),
            data: const {
              'error': 'internal_error',
              'requestId': 'PRIVATE_REQUEST_ID_CANARY',
            },
          ),
        ),
      );
      expect(details.requestId, _requestId);
      expect(
        SentrySanitizer.sanitizeContext({
          'appViewRequestId': 'PRIVATE_REQUEST_ID_CANARY',
        }),
        isEmpty,
      );
    },
  );

  final wrappers = <String, Object Function(Object, StackTrace)>{
    'auth server': (cause, stack) =>
        ServerUnavailable(cause: cause, stackTrace: stack),
    'auth storage': (cause, stack) => StorageFailure(cause, stackTrace: stack),
    'session storage': (cause, stack) => SessionRegistryStorageException(
      'write_failed',
      diagnosticCause: cause,
      diagnosticStack: stack,
    ),
    'chat draft': ChatDraftStorageException.new,
    'video': (cause, stack) => VideoTransportException(
      VideoTransportFailure.unavailable,
      diagnosticCause: cause,
      diagnosticStack: stack,
    ),
    'service status': StatusNetworkFailure.new,
    'composer': (cause, stack) => ComposerSubmissionFailure(
      diagnosticCause: cause,
      diagnosticStack: stack,
    ),
    'native subscriptions': (cause, stack) => RevenueCatServiceException(
      'restore_failed',
      diagnosticCause: cause,
      diagnosticStack: stack,
    ),
    'mapped API': (cause, stack) =>
        AppErrorMapper.map(cause, source: 'storage'),
  };
  for (final entry in wrappers.entries) {
    test(
      '${entry.key} preserves native causes and stacks in serialized issues',
      () async {
        final transport = SerializedTransport();
        await Sentry.init((options) {
          configureDiagnosticOptions(options);
          options
            ..dsn = 'https://public@example.invalid/1'
            ..transport = transport;
        });
        addTearDown(Sentry.close);
        final platform = PlatformException(
          code: 'write_error',
          message: 'PRIVATE_PLATFORM_MESSAGE',
          details: const {'draft': 'PRIVATE_DRAFT_CONTENT'},
        );
        final cause = ApiServerError(
          'PRIVATE_SERVER_MESSAGE',
          details: ApiFailureDetails(
            statusCode: 502,
            requestId: _requestId,
            method: 'POST',
            cause: platform,
            stackTrace: _causeStack,
          ),
        );
        await const SentryErrorReporter().captureException(
          entry.value(cause, _causeStack),
          stackTrace: StackTrace.current,
          context: const ReportContext(
            feature: 'Audit',
            operation: 'save',
            classification: 'storage.failed',
            safeDiagnostics: {'appViewRequestId': 'PRIVATE_REQUEST_ID_CANARY'},
          ),
        );
        await transport.issueReceived.future.timeout(
          const Duration(seconds: 5),
        );
        final serialized = transport.payloads.join();
        expect(serialized, contains('ApiServerError'));
        expect(serialized, contains('PlatformException'));
        expect(serialized, contains('originalStorageOperation'));
        expect(serialized, contains(_requestId));
        expect(serialized, contains('502'));
        expect(serialized, contains('POST'));
        expect(serialized, isNot(contains('PRIVATE_')));
      },
    );
  }
}

final class _ThrowingReporter implements ErrorReporter {
  final errors = <Object>[];

  @override
  bool get enabled => true;

  @override
  Future<String?> captureException(
    Object error, {
    required ReportContext context,
    StackTrace? stackTrace,
  }) async {
    errors.add(error);
    throw _DependencyFailure();
  }
}

final class _DependencyFailure implements Exception {
  @override
  String toString() => 'PRIVATE_DEPENDENCY_MESSAGE';
}

final class _Handler extends ErrorInterceptorHandler {
  ApiException? error;
  @override
  void next(DioException err) {
    error = err.error! as ApiException;
  }
}
