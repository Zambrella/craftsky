import 'dart:async';
import 'dart:ui';

import 'package:craftsky_app/main.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:logging/logging.dart';

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

    test('captures Flutter framework errors and keeps local severe logs', () {
      final reporter = _RecordingReporter();
      final stack = StackTrace.current;

      registerErrorHandlers(reporter: reporter);
      FlutterError.onError!(
        FlutterErrorDetails(
          exception: StateError('framework failed'),
          stack: stack,
        ),
      );

      expect(reporter.errors.single, isA<StateError>());
      expect(reporter.contexts.single.classification, 'flutter.framework');
      expect(
        records.any(
          (record) =>
              record.level == Level.SEVERE &&
              record.message.startsWith('FlutterError:'),
        ),
        isTrue,
      );
    });

    test('captures platform errors and reports them handled', () {
      final reporter = _RecordingReporter();
      final stack = StackTrace.current;

      registerErrorHandlers(reporter: reporter);
      final handled = PlatformDispatcher.instance.onError!(
        StateError('platform failed'),
        stack,
      );

      expect(handled, isTrue);
      expect(reporter.errors.single, isA<StateError>());
      expect(reporter.contexts.single.classification, 'flutter.platform');
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
  Future<ReportResult> captureException(
    Object error, {
    required ReportContext context,
    StackTrace? stackTrace,
  }) async {
    errors.add(error);
    contexts.add(context);
    return const ReportResult.captured();
  }

  @override
  Future<void> captureLog(
    LogRecord record, {
    required ReportContext context,
  }) async {}
}
