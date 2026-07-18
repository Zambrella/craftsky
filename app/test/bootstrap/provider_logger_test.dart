import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:logging/logging.dart';

void main() {
  group('ProviderLogger', () {
    test('reports provider failures classified as reportable', () async {
      final reporter = _RecordingReporter();
      final provider = FutureProvider<int>(
        name: 'failingProvider',
        (ref) => throw StateError('boom'),
      );
      final container = ProviderContainer(
        retry: appProviderRetry,
        observers: [ProviderLogger(reporter: reporter)],
      );
      addTearDown(container.dispose);

      await expectLater(container.read(provider.future), throwsStateError);

      expect(reporter.errors, hasLength(1));
      expect(reporter.contexts.single.feature, 'failingProvider');
      expect(reporter.contexts.single.classification, 'provider.failed');
    });

    test('does not report expected provider failures', () async {
      final reporter = _RecordingReporter();
      final provider = FutureProvider<int>(
        name: 'offlineProvider',
        (ref) => throw const ApiNetworkError('offline'),
      );
      final container = ProviderContainer(
        retry: appProviderRetry,
        observers: [ProviderLogger(reporter: reporter)],
      );
      addTearDown(container.dispose);

      await expectLater(
        container.read(provider.future),
        throwsA(isA<ApiNetworkError>()),
      );

      expect(reporter.errors, isEmpty);
    });

    test(
      'includes safe API diagnostics for reportable provider failures',
      () async {
        final reporter = _RecordingReporter();
        final provider = FutureProvider<int>(
          name: 'timelineProvider',
          (ref) => throw const ApiServerError(
            'http_500',
            details: ApiFailureDetails(
              statusCode: 500,
              appViewError: 'internal_error',
              requestId: 'req_123',
              endpointCategory: 'appview.feed.timeline',
            ),
          ),
        );
        final container = ProviderContainer(
          retry: appProviderRetry,
          observers: [ProviderLogger(reporter: reporter)],
        );
        addTearDown(container.dispose);

        await expectLater(
          container.read(provider.future),
          throwsA(isA<ApiServerError>()),
        );

        expect(reporter.contexts.single.feature, 'timelineProvider');
        expect(
          reporter.contexts.single.safeDiagnostics,
          containsPair('httpStatus', 500),
        );
        expect(
          reporter.contexts.single.safeDiagnostics,
          containsPair('appViewError', 'internal_error'),
        );
        expect(
          reporter.contexts.single.safeDiagnostics,
          containsPair('appViewRequestId', 'req_123'),
        );
        expect(
          reporter.contexts.single.safeDiagnostics,
          containsPair('endpointCategory', 'appview.feed.timeline'),
        );
      },
    );

    test('uses a generic feature for unnamed family providers', () async {
      final reporter = _RecordingReporter();
      final provider = FutureProvider.family<int, String>(
        (ref, handleOrDid) => throw StateError('provider failed'),
      );
      final container = ProviderContainer(
        retry: appProviderRetry,
        observers: [ProviderLogger(reporter: reporter)],
      );
      addTearDown(container.dispose);

      await expectLater(
        container.read(provider('did:plc:secret').future),
        throwsStateError,
      );

      expect(reporter.contexts.single.feature, 'riverpod.provider');
      expect(
        reporter.contexts.single.safeDiagnostics.values.join(' '),
        isNot(contains('did:plc:secret')),
      );
    });

    test('UT-014 redacts arbitrary provider failure text', () async {
      const sentinels = [
        'private-token-sentinel',
        'routing-id-sentinel',
        'did:plc:private-sentinel',
        'private.handle.test',
      ];
      final reporter = _RecordingReporter();
      final records = <LogRecord>[];
      final previousLevel = Logger.root.level;
      Logger.root.level = Level.ALL;
      final subscription = Logger.root.onRecord.listen(records.add);
      addTearDown(() async {
        await subscription.cancel();
        Logger.root.level = previousLevel;
      });
      final provider = FutureProvider<int>(
        name: 'safeProvider',
        (ref) => throw StateError(sentinels.join(' ')),
      );
      final container = ProviderContainer(
        retry: appProviderRetry,
        observers: [ProviderLogger(reporter: reporter)],
      );
      addTearDown(container.dispose);

      await expectLater(container.read(provider.future), throwsStateError);
      await Future<void>.delayed(Duration.zero);

      final diagnostic = '${records.join(' ')} ${reporter.errors.join(' ')}';
      for (final sentinel in sentinels) {
        expect(diagnostic, isNot(contains(sentinel)));
      }
    });
  });
}

final class _RecordingReporter implements ErrorReporter {
  final errors = <Object>[];
  final contexts = <ReportContext>[];

  @override
  bool get enabled => true;

  @override
  void addBreadcrumb(SafeBreadcrumb breadcrumb) {}

  @override
  Future<String?> captureException(
    Object error, {
    required ReportContext context,
    StackTrace? stackTrace,
  }) async {
    errors.add(error);
    contexts.add(context);
    return '0123456789abcdef0123456789abcdef';
  }

  @override
  Future<void> captureMessage(
    String message, {
    required ReportContext context,
  }) async {}
}
