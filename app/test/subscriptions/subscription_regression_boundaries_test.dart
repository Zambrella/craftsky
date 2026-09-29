import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

void main() {
  test('REG-002 feed and search ranking remain independent of paid tier', () {
    for (final fileName in const [
      'lib/feed/providers/timeline_provider.dart',
      'lib/search/providers/post_search_provider.dart',
      'lib/search/providers/project_search_provider.dart',
      'lib/search/providers/profile_search_provider.dart',
    ]) {
      final source = File(fileName).readAsStringSync();
      expect(
        source,
        isNot(contains('SubscriptionTier.')),
        reason: '$fileName must not sort by paid tier',
      );
      expect(
        source,
        isNot(contains('/subscriptions/')),
        reason: '$fileName must not depend on billing',
      );
    }
  });

  test('REG-003 native billing remains isolated to subscription adapters', () {
    final nativeImports = Directory('lib')
        .listSync(recursive: true)
        .whereType<File>()
        .where((file) => file.path.endsWith('.dart'))
        .where((file) {
          final source = file.readAsStringSync();
          return source.contains('package:purchases_flutter/') ||
              source.contains('package:purchases_ui_flutter/');
        })
        .map((file) => file.path)
        .toList();

    expect(nativeImports, [
      'lib/subscriptions/services/revenuecat_service_native.dart',
    ]);
  });
}
