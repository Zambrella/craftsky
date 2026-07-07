import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/error_reporter_provider.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

Widget appHarness({
  required Widget child,
  ErrorReporter reporter = const NoopErrorReporter(),
}) {
  return ProviderScope(
    retry: appProviderRetry,
    overrides: [
      errorReporterProvider.overrideWithValue(reporter),
    ],
    child: child,
  );
}
