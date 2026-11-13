import 'dart:async';
import 'dart:ui';

import 'package:craftsky_app/application_startup.dart';
import 'package:craftsky_app/initialization_error_screen.dart';
import 'package:craftsky_app/initialization_loading_screen.dart';
import 'package:craftsky_app/main.dart' as application;
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';
import 'package:sentry_flutter/sentry_flutter.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  testWidgets('preference failure before runApp shows localized recovery', (
    tester,
  ) async {
    SharedPreferences.resetStatic();
    addTearDown(SharedPreferences.resetStatic);
    final oldFlutter = FlutterError.onError;
    final oldPlatform = PlatformDispatcher.instance.onError;
    final oldBuilder = ErrorWidget.builder;
    final messenger =
        TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger;
    const channel = MethodChannel('plugins.flutter.io/shared_preferences');
    messenger.setMockMethodCallHandler(channel, (call) async {
      throw PlatformException(
        code: 'read_error',
        message: 'PRIVATE_STARTUP_PAYLOAD',
      );
    });
    addTearDown(() {
      messenger.setMockMethodCallHandler(channel, null);
      FlutterError.onError = oldFlutter;
      PlatformDispatcher.instance.onError = oldPlatform;
      ErrorWidget.builder = oldBuilder;
    });

    await tester.runAsync(application.main);
    FlutterError.onError = oldFlutter;
    PlatformDispatcher.instance.onError = oldPlatform;
    ErrorWidget.builder = oldBuilder;
    await tester.pump();

    expect(find.byType(InitializationErrorScreen), findsOneWidget);
    expect(
      find.text("CraftSky couldn't finish starting. Please try again."),
      findsOneWidget,
    );
    expect(find.text('Retry'), findsOneWidget);
    expect(find.textContaining('PRIVATE_'), findsNothing);
    expect(Sentry.isEnabled, isFalse);
  });

  testWidgets('startup reports its original cause and user retry is usable', (
    tester,
  ) async {
    final retry = Completer<void>();
    final export = Completer<void>();
    final reporter = _PendingReporter(export.future);
    final cause = StateError('PRIVATE_STARTUP_FAILURE');
    final stack = StackTrace.fromString(
      '#0 initializeMedia (package:craftsky_app/main.dart:45:3)',
    );
    var attempts = 0;
    var splashRemoved = 0;
    await runApplicationStartup(
      initialize: () async {
        attempts++;
        if (attempts == 1) Error.throwWithStackTrace(cause, stack);
        await retry.future;
        // This terminal fixture has no provider dependencies.
        // ignore: riverpod_lint/missing_provider_scope
        runApp(
          const Directionality(
            textDirection: TextDirection.ltr,
            child: Text('App ready'),
          ),
        );
      },
      reporter: () => reporter,
      removeSplash: () => splashRemoved++,
    );
    await tester.pump();
    expect(reporter.errors, [cause]);
    expect(reporter.stacks.single, same(stack));
    expect(reporter.contexts.single.classification, 'app.startup');
    expect(splashRemoved, 1);
    expect(find.byType(InitializationErrorScreen), findsOneWidget);
    expect(find.textContaining('PRIVATE_'), findsNothing);

    tester.widget<InitializationErrorScreen>(
        find.byType(InitializationErrorScreen),
      )
      ..onRetry()
      ..onRetry();
    await tester.pump();
    expect(attempts, 2);
    expect(find.byType(InitializationLoadingScreen), findsOneWidget);
    retry.complete();
    await tester.pump();
    expect(find.text('App ready'), findsOneWidget);
    expect(reporter.errors, hasLength(1));
    export.complete();
    await tester.pump();
  });
}

final class _PendingReporter implements ErrorReporter {
  new(this.pending);
  final Future<void> pending;
  final errors = <Object>[];
  final stacks = <StackTrace?>[];
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
    stacks.add(stackTrace);
    contexts.add(context);
    await pending;
    return null;
  }
}
