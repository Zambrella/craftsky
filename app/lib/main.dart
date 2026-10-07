// This file is the debug sink for the `logging` package: it configures the
// root logger to forward records to stdout via `print` when running in debug
// mode. That is the one legitimate place in the codebase where `print` is
// used; everywhere else, use `Logger`.
import 'dart:async';

import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/shared/observability/diagnostic_emitter.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/log_forwarder.dart';
import 'package:craftsky_app/shared/observability/observability_bootstrap.dart';
import 'package:craftsky_app/shared/observability/platform_log.dart';
import 'package:craftsky_app/shared/observability/sentry_config.dart';
import 'package:craftsky_app/shared/observability/sentry_error_reporter.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_native_splash/flutter_native_splash.dart';
import 'package:logging/logging.dart';
import 'package:media_kit/media_kit.dart';

final _log = Logger('main');

Future<void> main() async {
  ErrorReporter reporter = const NoopErrorReporter();
  await runZonedGuarded(
    () async {
      final binding = WidgetsFlutterBinding.ensureInitialized();
      FlutterNativeSplash.preserve(widgetsBinding: binding);
      MediaKit.ensureInitialized();

      // Configure logging before anything else so error handlers and
      // bootstrap can both log through the root logger.
      Logger.root.level = Level.FINE;
      configureRootLogForwarding(
        reporter: reporter,
        currentReporter: () => reporter,
      );

      reporter = await ObservabilityBootstrap.initialize(
        config: SentryConfig.fromEnvironment(),
        adapter: const SentryFlutterBootstrapAdapter(),
      );

      registerErrorHandlers(reporter: reporter);

      await bootstrap(binding, reporter: reporter);
    },
    (error, stack) {
      FlutterNativeSplash.remove();
      _log.severe(
        const DiagnosticMessage(
          'runZonedGuarded caught error',
          context: ReportContext(
            feature: 'main',
            operation: 'runZonedGuarded',
            classification: 'dart.root_zone',
          ),
        ),
        error,
        stack,
      );
      unawaited(
        reporter.captureException(
          error,
          stackTrace: stack,
          context: const ReportContext(
            feature: 'main',
            operation: 'runZonedGuarded',
            classification: 'dart.root_zone',
          ),
        ),
      );
    },
  );
}

StreamSubscription<LogRecord> configureRootLogForwarding({
  required ErrorReporter reporter,
  PlatformLogSink? platformSink,
  ErrorReporter Function()? currentReporter,
}) {
  final emitter = DiagnosticEmitter(platformSink: platformSink);
  return Logger.root.onRecord.listen((record) {
    emitter.emitLocal(record);
    final forwarder = LogForwarder(
      GuardedErrorReporter(
        currentReporter?.call() ?? reporter,
        fallbackSink: platformSink,
      ),
    );
    unawaited(
      forwarder.handle(record),
    );
  });
}

void registerErrorHandlers({required ErrorReporter reporter}) {
  final log = Logger('ErrorHandlers');

  FlutterError.onError = (details) {
    FlutterError.presentError(
      FlutterErrorDetails(
        exception: FlutterError(
          'Framework failure (${details.exception.runtimeType})',
        ),
      ),
    );
    log.severe(
      const DiagnosticMessage(
        'FlutterError: Framework failure',
        context: ReportContext(
          feature: 'flutter',
          operation: 'framework_error',
          classification: 'flutter.framework',
        ),
      ),
      details.exception,
      details.stack,
    );
    unawaited(
      reporter.captureException(
        details.exception,
        stackTrace: details.stack,
        context: const ReportContext(
          feature: 'flutter',
          operation: 'framework_error',
          classification: 'flutter.framework',
        ),
      ),
    );
  };

  PlatformDispatcher.instance.onError = (error, stack) {
    log.severe(
      const DiagnosticMessage(
        'Platform error',
        context: ReportContext(
          feature: 'flutter',
          operation: 'platform_error',
          classification: 'flutter.platform',
        ),
      ),
      error,
      stack,
    );
    unawaited(
      reporter.captureException(
        error,
        stackTrace: stack,
        context: const ReportContext(
          feature: 'flutter',
          operation: 'platform_error',
          classification: 'flutter.platform',
        ),
      ),
    );
    return true;
  };

  ErrorWidget.builder = (details) {
    log.warning(
      'FlutterError: Framework failure',
      details.exception,
      details.stack,
    );
    if (kDebugMode) {
      return ErrorWidget(
        'Rendering failure (${details.exception.runtimeType})',
      );
    }
    // Release fallback. `ErrorWidget.builder` is called in situations where
    // there may be no ambient Directionality (e.g. an error above MaterialApp),
    // so we inject one rather than rely on the tree.
    return const Directionality(
      textDirection: TextDirection.ltr,
      child: ColoredBox(
        color: Colors.red,
        child: Center(
          child: Text(
            'An error occurred rendering this element',
            style: TextStyle(color: Colors.white),
          ),
        ),
      ),
    );
  };
}
