import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

import '../test_support/source_scan.dart';

void main() {
  test('IT-020 preview production code uses only the AppView API boundary', () {
    final previewSources = scanDartSources(
      'lib/feed',
    ).where((file) => file.path.contains('link_preview'));
    final violations = forbiddenSourceMatches(previewSources, [
      "import 'dart:io'",
      'package:http/',
      'Image.network(',
      'NetworkImage(',
      '.getUri(',
      '.postUri(',
    ]);
    expect(violations, isEmpty, reason: violations.join('\n'));

    final controller = File(
      'lib/feed/composer/link_preview_controller.dart',
    ).readAsStringSync();
    expect(controller, contains('postApiClientProvider'));
    expect(controller, contains('fetchLinkPreview'));
  });
}
