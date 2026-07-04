import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

void main() {
  test('only central observability implementation imports Sentry packages', () {
    final offenders = <String>[];

    for (final file in Directory('lib').listSync(recursive: true)) {
      if (file is! File || !file.path.endsWith('.dart')) continue;
      final text = file.readAsStringSync();
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
