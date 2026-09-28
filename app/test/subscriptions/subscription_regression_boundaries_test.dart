import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

void main() {
  test('REG-001 existing feature areas contain no paid-tier gates', () {
    for (final area in const [
      'feed',
      'profile',
      'drafts',
      'saved_posts',
      'scheduled_posts',
      'business',
    ]) {
      final files = Directory('lib/$area')
          .listSync(recursive: true)
          .whereType<File>()
          .where((file) => file.path.endsWith('.dart'));
      for (final file in files) {
        final source = file.readAsStringSync();
        expect(
          source,
          isNot(contains('SubscriptionTier.')),
          reason: '${file.path} must not gate existing behavior by paid tier',
        );
        expect(
          source,
          isNot(contains('/subscriptions/')),
          reason: '${file.path} must not depend on subscription implementation',
        );
      }
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
