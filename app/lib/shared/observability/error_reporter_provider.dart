import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

final errorReporterProvider = Provider<ErrorReporter>(
  (ref) => const NoopErrorReporter(),
);
