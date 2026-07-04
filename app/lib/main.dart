// This file is the debug sink for the `logging` package: it configures the
// root logger to forward records to stdout via `print` when running in debug
// mode. That is the one legitimate place in the codebase where `print` is
// used; everywhere else, use `Logger`.
import 'dart:async';
import 'dart:developer' as developer;

import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/log_forwarder.dart';
import 'package:craftsky_app/shared/observability/observability_bootstrap.dart';
import 'package:craftsky_app/shared/observability/sentry_config.dart';
import 'package:craftsky_app/shared/observability/sentry_error_reporter.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:logging/logging.dart';

final _log = Logger('main');

Future<void> main() async {
  ErrorReporter reporter = const NoopErrorReporter();
  await runZonedGuarded(
    () async {
      final binding = WidgetsFlutterBinding.ensureInitialized();

      // Configure logging before anything else so error handlers and
      // bootstrap can both log through the root logger.
      Logger.root.level = Level.FINE;
      Logger.root.onRecord.listen((record) {
        if (kDebugMode) {
          print(
            '${record.level.name} | ${record.loggerName}: ${record.message}',
          );
          if (record.error != null) {
            print('  error: ${record.error}');
          }
          if (record.stackTrace != null) {
            print('  stack: ${record.stackTrace}');
          }
        }
      });

      reporter = await ObservabilityBootstrap.initialize(
        config: SentryConfig.fromEnvironment(),
        adapter: const SentryFlutterBootstrapAdapter(),
      );
      configureRootLogForwarding(reporter: reporter);

      registerErrorHandlers(reporter: reporter);

      await bootstrap(binding, reporter: reporter);
    },
    (error, stack) {
      // Last-resort sink: use dart:developer log because logging may not be
      // fully wired yet depending on where the crash originates.
      developer.log(
        'runZonedGuarded: $error',
        name: 'main',
        error: error,
        stackTrace: stack,
        level: 1000,
      );
      _log.severe('runZonedGuarded caught error', error, stack);
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
}) {
  final forwarder = LogForwarder(reporter);
  return Logger.root.onRecord.listen((record) {
    unawaited(forwarder.handle(record));
  });
}

void registerErrorHandlers({required ErrorReporter reporter}) {
  final log = Logger('ErrorHandlers');

  FlutterError.onError = (details) {
    FlutterError.presentError(details);
    log.severe(
      'FlutterError: ${details.exception}',
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
    log.severe('Platform error', error, stack);
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
      'Error building widget: ${details.exception}',
      details.exception,
      details.stack,
    );
    if (kDebugMode) {
      return ErrorWidget(details.exception);
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
