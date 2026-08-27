import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

void main() {
  test('IT-020 preview production code uses only the AppView API boundary', () {
    final paths = [
      'lib/feed/composer/link_preview_candidate.dart',
      'lib/feed/composer/link_preview_candidates.dart',
      'lib/feed/composer/link_preview_controller.dart',
      'lib/feed/models/link_preview.dart',
      'lib/feed/widgets/composer_link_preview_carousel.dart',
    ];
    for (final path in paths) {
      final source = File(path).readAsStringSync();
      for (final forbidden in [
        "import 'dart:io'",
        'package:http/',
        'Image.network(',
        'NetworkImage(',
        '.getUri(',
        '.postUri(',
      ]) {
        expect(source, isNot(contains(forbidden)), reason: '$path: $forbidden');
      }
    }

    final controller = File(
      'lib/feed/composer/link_preview_controller.dart',
    ).readAsStringSync();
    expect(controller, contains('postApiClientProvider'));
    expect(controller, contains('fetchLinkPreview'));
  });
}
