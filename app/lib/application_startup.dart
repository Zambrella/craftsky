import 'dart:async';

import 'package:craftsky_app/initialization_error_screen.dart';
import 'package:craftsky_app/initialization_loading_screen.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:logging/logging.dart';
import 'package:material_ui/material_ui.dart';

final _log = Logger('ApplicationStartup');

/// Owns failures before the normal app and its dependency error UI can mount.
/// Retry stays in the application's original startup zone and reloads consent.
Future<void> runApplicationStartup({
  required Future<void> Function() initialize,
  required ErrorReporter Function() reporter,
  required VoidCallback removeSplash,
}) async {
  var running = false;
  var retrying = false;

  Future<void> start() async {
    if (running) return;
    running = true;
    if (retrying) {
      // Recovery works before the normal provider graph can initialize.
      // ignore: riverpod_lint/missing_provider_scope
      runApp(_recoveryApp(const InitializationLoadingScreen()));
    }
    try {
      await initialize();
    } on Object catch (error, stack) {
      retrying = true;
      removeSplash();
      _log.severe('Application startup failed', error, stack);
      // These recovery screens have no provider dependencies.
      // ignore: riverpod_lint/missing_provider_scope
      runApp(
        _recoveryApp(
          InitializationErrorScreen(
            error: error,
            onRetry: () => unawaited(start()),
          ),
        ),
      );
      unawaited(
        GuardedErrorReporter(reporter()).captureException(
          error,
          stackTrace: stack,
          context: const ReportContext(
            feature: 'main',
            operation: 'bootstrap',
            classification: 'app.startup',
          ),
        ),
      );
    } finally {
      running = false;
    }
  }

  await start();
}

Widget _recoveryApp(Widget screen) => MaterialApp(
  debugShowCheckedModeBanner: false,
  localizationsDelegates: appLocalizationsDelegates,
  supportedLocales: AppLocalizations.supportedLocales,
  theme: AppTheme.lightThemeData,
  darkTheme: AppTheme.darkThemeData,
  home: screen,
);
