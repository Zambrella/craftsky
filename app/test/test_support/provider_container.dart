import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/error_reporter_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

ProviderContainer createTestContainer({
  List<dynamic> overrides = const [],
  List<ProviderObserver> observers = const [],
  ErrorReporter reporter = const NoopErrorReporter(),
}) => ProviderContainer.test(
  retry: appProviderRetry,
  observers: observers,
  overrides: List.from([
    errorReporterProvider.overrideWithValue(reporter),
    ...overrides,
  ]),
);
