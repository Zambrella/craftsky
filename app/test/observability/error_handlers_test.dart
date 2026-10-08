import 'dart:async';
import 'dart:convert';
import 'dart:ui';

import 'package:craftsky_app/main.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/errors/app_error.dart';
import 'package:craftsky_app/shared/observability/diagnostic_emitter.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/sentry_error_reporter.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:logging/logging.dart';
import 'package:sentry_flutter/sentry_flutter.dart' hide ErrorCallback;

import '../test_support/serialized_sentry_transport.dart';

void main() {
  group('registerErrorHandlers', () {
    late FlutterExceptionHandler? oldFlutterHandler;
    late ErrorCallback? oldPlatformHandler;
    late Widget Function(FlutterErrorDetails) oldErrorWidgetBuilder;
    late List<LogRecord> records;
    late StreamSubscription<LogRecord> logSub;

    setUp(() {
      oldFlutterHandler = FlutterError.onError;
      oldPlatformHandler = PlatformDispatcher.instance.onError;
      oldErrorWidgetBuilder = ErrorWidget.builder;
      records = <LogRecord>[];
      logSub = Logger.root.onRecord.listen(records.add);
    });

    tearDown(() async {
      FlutterError.onError = oldFlutterHandler;
      PlatformDispatcher.instance.onError = oldPlatformHandler;
      ErrorWidget.builder = oldErrorWidgetBuilder;
      await logSub.cancel();
    });

    test(
      'SDK-T04 local fallback leaves capture to installed SDK integrations',
      () async {
        final reporter = _RecordingReporter();
        registerErrorHandlers();
        final transport = _AutomaticErrorTransport();
        await SentryFlutter.init((options) {
          options
            ..dsn = 'https://public@example.invalid/1'
            ..autoInitializeNativeSdk = false
            ..enableAutoPerformanceTracing = false
            ..transport = transport;
          configureDiagnosticOptions(options);
        });
        addTearDown(Sentry.close);
        final stack = StackTrace.fromString(
          '#0 renderPost (package:craftsky_app/post.dart:12:3)',
        );
        FlutterError.onError!(
          FlutterErrorDetails(
            exception: StateError('private-framework-data'),
            stack: stack,
          ),
        );
        PlatformDispatcher.instance.onError!(
          StateError('private-platform-data'),
          stack,
        );
        await transport.bothIssues.future.timeout(const Duration(seconds: 5));
        await Sentry.close();
        expect(reporter.errors, isEmpty);
        final events = transport.payloads
            .map(jsonDecode)
            .where((e) => (e as Map).containsKey('exception'))
            .toList();
        expect(events, hasLength(2));
        expect(events.toString(), contains('FlutterError'));
        expect(events.toString(), contains('PlatformDispatcher.onError'));
        expect(events.toString(), contains('renderPost'));
        expect(events.toString(), isNot(contains('private-framework-data')));
        expect(events.toString(), isNot(contains('private-platform-data')));
      },
    );

    for (final boundary in ['framework', 'platform', 'zone']) {
      test(
        'SDK-T04 $boundary filters expected failures at the SDK boundary',
        () async {
          registerErrorHandlers();
          final transport = SerializedTransport();
          final local = <String>[];
          final subscription = configureRootLogForwarding(
            platformSink: local.add,
          );
          addTearDown(subscription.cancel);
          final processed = Completer<void>();
          var count = 0;
          await SentryFlutter.init((options) {
            options
              ..dsn = 'https://public@example.invalid/1'
              ..autoInitializeNativeSdk = false
              ..enableAutoPerformanceTracing = false
              ..transport = transport;
            configureDiagnosticOptions(options);
            final beforeSend = options.beforeSend!;
            options.beforeSend = (event, hint) async {
              final result = await beforeSend(event, hint);
              if (++count == 8) processed.complete();
              return result;
            };
          });
          addTearDown(Sentry.close);
          final stack = StackTrace.fromString(
            '#0 upload (package:craftsky_app/upload.dart:12:3)',
          );
          final failures = <Object>[
            const ApiCanceled(),
            const AppError(AppErrorKind.networkUnavailable),
            const ApiUnauthorized(),
            const AppError(AppErrorKind.contentUnavailable),
            const ApiBadRequest('validation_failed'),
            const AppError(AppErrorKind.unexpected, reportableOverride: false),
            const AppError(
              AppErrorKind.networkUnavailable,
              reportableOverride: true,
            ),
            AppError(
              AppErrorKind.unexpected,
              diagnosticCause: StateError('private-automatic-canary'),
              diagnosticStack: stack,
            ),
          ];
          for (final error in failures) {
            switch (boundary) {
              case 'framework':
                FlutterError.onError!(
                  FlutterErrorDetails(exception: error, stack: stack),
                );
              case 'platform':
                expect(
                  PlatformDispatcher.instance.onError!(error, stack),
                  isTrue,
                );
              case 'zone':
                Sentry.runZonedGuarded(
                  () => Error.throwWithStackTrace(error, stack),
                  (error, stack) =>
                      Logger('Zone').severe('Unhandled failure', error, stack),
                );
            }
          }
          await processed.future.timeout(const Duration(seconds: 5));
          await Sentry.close();
          final events = transport.payloads
              .map(jsonDecode)
              .where((event) => (event as Map).containsKey('exception'))
              .toList();
          expect(events, hasLength(2));
          expect(
            events.toString(),
            contains(switch (boundary) {
              'framework' => 'FlutterError',
              'platform' => 'PlatformDispatcher.onError',
              _ => 'runZonedGuarded',
            }),
          );
          expect(events.toString(), contains('StateError'));
          expect(events.toString(), contains('upload'));
          expect(local, hasLength(8));
          expect(local.join(), contains('ApiCanceled'));
          expect(local.join(), contains('upload'));
          expect(
            local.join() + transport.payloads.join(),
            isNot(contains('private-automatic-canary')),
          );
        },
      );
    }

    test(
      'UT-010 supporting root logs do not recapture callback occurrences',
      () async {
        final reporter = _RecordingReporter();
        final subscription = configureRootLogForwarding(
          platformSink: (_) {},
        );
        addTearDown(subscription.cancel);
        registerErrorHandlers();
        final error = StateError('same opaque failure');
        final stack = StackTrace.current;
        PlatformDispatcher.instance.onError!(error, stack);
        PlatformDispatcher.instance.onError!(error, stack);
        await Future<void>.delayed(Duration.zero);
        expect(reporter.errors, isEmpty);
        expect(
          records.where((record) => record.object is DiagnosticMessage),
          hasLength(2),
        );
      },
    );

    test('keeps protected local framework output without owning capture', () {
      final reporter = _RecordingReporter();
      final stack = StackTrace.current;

      registerErrorHandlers();
      FlutterError.onError!(
        FlutterErrorDetails(
          exception: StateError('framework failed'),
          stack: stack,
        ),
      );

      expect(reporter.errors, isEmpty);
      expect(
        records.any(
          (record) =>
              record.level == Level.SEVERE &&
              record.message.startsWith('FlutterError:'),
        ),
        isTrue,
      );
    });

    test(
      'keeps protected local platform output and suppresses raw printing',
      () {
        final reporter = _RecordingReporter();
        final stack = StackTrace.current;

        registerErrorHandlers();
        final handled = PlatformDispatcher.instance.onError!(
          StateError('platform failed'),
          stack,
        );

        expect(handled, isTrue);
        expect(reporter.errors, isEmpty);
      },
    );
  });
}

final class _RecordingReporter implements ErrorReporter {
  final errors = <Object>[];
  final contexts = <ReportContext>[];

  @override
  bool get enabled => true;

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
}

final class _AutomaticErrorTransport extends SerializedTransport {
  final bothIssues = Completer<void>();
  @override
  Future<SentryId?> send(SentryEnvelope envelope) async {
    final id = await super.send(envelope);
    if (payloads
                .where((p) => (jsonDecode(p) as Map).containsKey('exception'))
                .length ==
            2 &&
        !bothIssues.isCompleted) {
      bothIssues.complete();
    }
    return id;
  }
}
