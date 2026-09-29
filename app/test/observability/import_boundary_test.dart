import 'package:flutter_test/flutter_test.dart';

import '../test_support/source_scan.dart';

void main() {
  test('only central observability implementation imports Sentry packages', () {
    final offenders = <String>[];

    for (final file in scanDartSources('lib')) {
      final text = file.source;
      final importsSentry =
          text.contains('package:sentry_flutter/') ||
          text.contains('package:sentry_logging/') ||
          text.contains('package:sentry_dio/');
      if (!importsSentry) continue;

      if (file.path != 'lib/shared/observability/sentry_error_reporter.dart') {
        offenders.add(file.path);
      }
    }

    expect(offenders, isEmpty);
  });
}
