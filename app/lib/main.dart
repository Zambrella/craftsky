// The local logging sink emits protected INFO+ records in every build.
// CRAFTSKY_DEBUG_LOGS enables FINE output only in debug builds.
import 'dart:async';

import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/shared/observability/diagnostic_emitter.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/error_reporting_controller.dart';
import 'package:craftsky_app/shared/observability/platform_log.dart';
import 'package:craftsky_app/shared/observability/sentry_config.dart';
import 'package:craftsky_app/shared/observability/sentry_error_reporter.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_native_splash/flutter_native_splash.dart';
import 'package:logging/logging.dart';
import 'package:material_ui/material_ui.dart';
import 'package:media_kit/media_kit.dart';
import 'package:shared_preferences/shared_preferences.dart';

final _log = Logger('main');

Future<void> main() async {
  Logger.root.level = Level.FINE;
  configureRootLogForwarding();
  // Install local output before Sentry so its integrations can chain it.
  registerErrorHandlers();
  ErrorReporter reporter = const NoopErrorReporter();
  // Keep bindings, SDK initialization and runApp in one zone on every platform.
  // The zone reads the stable reporter so runtime consent changes take effect.
  await runZonedGuarded(
    () async {
      final binding = WidgetsFlutterBinding.ensureInitialized();
      FlutterNativeSplash.preserve(widgetsBinding: binding);
      try {
        SharedPreferences.setPrefix('craftsky.');
        final reporting = ErrorReportingController(
          preferences: await SharedPreferences.getInstance(),
          config: SentryConfig.fromEnvironment(),
          adapter: const SentryFlutterBootstrapAdapter(),
          closeSdk: closeSentrySdk,
        );
        reporter = reporting;
        await reporting.initialize();
        MediaKit.ensureInitialized();
        await bootstrap(
          binding,
          reporter: reporting,
          errorReporting: reporting,
        );
      } on Object catch (error, stack) {
        FlutterNativeSplash.remove();
        _log.severe('Application startup failed', error, stack);
        await reporter.captureException(
          error,
          stackTrace: stack,
          context: const ReportContext(
            feature: 'main',
            operation: 'bootstrap',
            classification: 'app.startup',
          ),
        );
      }
    },
    (error, stack) {
      _log.severe('Unhandled application error', error, stack);
      unawaited(
        reporter.captureException(
          error,
          stackTrace: stack,
          context: const ReportContext(
            feature: 'main',
            operation: 'unhandled_error',
            classification: 'app.unhandled',
          ),
        ),
      );
    },
  );
}

StreamSubscription<LogRecord> configureRootLogForwarding({
  PlatformLogSink? platformSink,
  bool debugLogs = const bool.fromEnvironment('CRAFTSKY_DEBUG_LOGS'),
}) {
  final emitter = DiagnosticEmitter(
    platformSink: platformSink,
    debugLogs: debugLogs,
  );
  return Logger.root.onRecord.listen(emitter.emitLocal);
}

void registerErrorHandlers() {
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
