import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/error_reporter_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'app_harness.dart';

void main() {
  testWidgets('app harness provides no-op reporter and disables retry', (
    tester,
  ) async {
    var attempts = 0;
    final failingProvider = FutureProvider<int>((ref) async {
      attempts++;
      throw StateError('boom');
    });

    await tester.pumpWidget(
      appHarness(
        child: Consumer(
          builder: (context, ref, child) {
            final reporter = ref.watch(errorReporterProvider);
            final value = ref.watch(failingProvider);
            return MaterialApp(
              home: Text('${reporter.enabled}:${value.hasError}'),
            );
          },
        ),
      ),
    );

    await tester.pumpAndSettle();
    await tester.pump(const Duration(seconds: 1));

    expect(find.text('false:true'), findsOneWidget);
    expect(attempts, 1);
  });

  testWidgets('app harness can override the reporter', (tester) async {
    final reporter = _EnabledReporter();

    await tester.pumpWidget(
      appHarness(
        reporter: reporter,
        child: Directionality(
          textDirection: TextDirection.ltr,
          child: Consumer(
            builder: (context, ref, child) {
              return Text('${ref.watch(errorReporterProvider).enabled}');
            },
          ),
        ),
      ),
    );

    expect(find.text('true'), findsOneWidget);
  });
}

final class _EnabledReporter implements ErrorReporter {
  @override
  bool get enabled => true;

  @override
  void addBreadcrumb(SafeBreadcrumb breadcrumb) {}

  @override
  Future<String?> captureException(
    Object error, {
    required ReportContext context,
    StackTrace? stackTrace,
  }) async {
    return '0123456789abcdef0123456789abcdef';
  }

  @override
  Future<void> captureMessage(
    String message, {
    required ReportContext context,
  }) async {}
}
