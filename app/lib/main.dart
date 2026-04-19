import 'dart:async';
import 'dart:developer' as developer;

import 'package:craftsky_app/bootstrap.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:logging/logging.dart';

final _log = Logger('main');

Future<void> main() async {
  await runZonedGuarded(
    () async {
      final binding = WidgetsFlutterBinding.ensureInitialized();

      // Configure logging before anything else so error handlers and
      // bootstrap can both log through the root logger.
      Logger.root.level = Level.FINE;
      Logger.root.onRecord.listen((record) {
        if (kDebugMode) {
          // ignore: avoid_print
          print('${record.level.name} | ${record.loggerName}: ${record.message}');
          if (record.error != null) {
            // ignore: avoid_print
            print('  error: ${record.error}');
          }
          if (record.stackTrace != null) {
            // ignore: avoid_print
            print('  stack: ${record.stackTrace}');
          }
        }
      });

      registerErrorHandlers();

      await bootstrap(binding);
    },
    (Object error, StackTrace stack) {
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
    },
  );
}

void registerErrorHandlers() {
  final log = Logger('ErrorHandlers');

  FlutterError.onError = (FlutterErrorDetails details) {
    FlutterError.presentError(details);
    log.severe('FlutterError: ${details.exception}', details.exception, details.stack);
  };

  PlatformDispatcher.instance.onError = (Object error, StackTrace stack) {
    log.severe('Platform error', error, stack);
    return true;
  };

  ErrorWidget.builder = (FlutterErrorDetails details) {
    log.warning('Error building widget: ${details.exception}', details.exception, details.stack);
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
