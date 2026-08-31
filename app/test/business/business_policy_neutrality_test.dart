import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

void main() {
  test('AT-014 business presentation makes no prohibited claim', () {
    final source =
        [
              ...Directory('lib/business').listSync(recursive: true),
              File('lib/profile/widgets/profile_meta_section.dart'),
              File('lib/profile/widgets/profile_tabs/profile_about_tab.dart'),
              File(
                'lib/profile/widgets/profile_tabs/profile_products_tab.dart',
              ),
              File('lib/profile/widgets/profile_tabs/profile_events_tab.dart'),
              File('lib/settings/pages/account_page.dart'),
            ]
            .whereType<File>()
            .where((file) => file.path.endsWith('.dart'))
            .map(
              (file) => file.readAsStringSync().toLowerCase(),
            )
            .join('\n');

    for (final claim in [
      'verified business',
      'verification badge',
      'business subscription',
      'subscriber benefit',
      'ranking boost',
      'rank higher',
      'increase reach',
      'moderation priority',
      'priority moderation',
      'inventory',
      'in stock',
      'availability guaranteed',
      'tax included',
      'shipping included',
      'checkout',
      'buy now',
    ]) {
      expect(source, isNot(contains(claim)), reason: claim);
    }
  });
}
