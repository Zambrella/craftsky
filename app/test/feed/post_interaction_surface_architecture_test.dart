import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

void main() {
  test('REG-002 interaction navigation stays scoped to thread details', () {
    const threadPath = 'lib/feed/pages/post_thread_page.dart';
    const cardDefinitionPath = 'lib/feed/widgets/post_card.dart';
    const summaryDefinitionPath =
        'lib/feed/widgets/post_interaction_summary.dart';
    const expectedCardConsumers = {
      'lib/feed/pages/feed_page.dart',
      'lib/feed/pages/post_quotes_page.dart',
      threadPath,
      'lib/profile/widgets/profile_tabs/profile_comments_tab.dart',
      'lib/profile/widgets/profile_tabs/profile_post_feed_slivers.dart',
      'lib/projects/pages/projects_page.dart',
      'lib/search/pages/search_page.dart',
      'lib/search/pages/tag_search_page.dart',
    };

    final sources = <String, String>{};
    for (final entity in Directory('lib').listSync(recursive: true)) {
      if (entity is! File || !entity.path.endsWith('.dart')) continue;
      final path = entity.path.replaceAll(Platform.pathSeparator, '/');
      if (path.endsWith('.g.dart') || path.endsWith('.mapper.dart')) continue;
      sources[path] = entity.readAsStringSync();
    }

    final cardConsumers = {
      for (final MapEntry(:key, :value) in sources.entries)
        if (key != cardDefinitionPath &&
            RegExp(r'\bPostCard\s*\(').hasMatch(value))
          key,
    };
    expect(cardConsumers, expectedCardConsumers);

    final summaryOwners = {
      for (final MapEntry(:key, :value) in sources.entries)
        if (key != summaryDefinitionPath &&
            RegExp(r'\bPostInteractionSummary\s*\(').hasMatch(value))
          key,
    };
    final responseListOwners = {
      for (final MapEntry(:key, :value) in sources.entries)
        if (key != cardDefinitionPath &&
            RegExp(r'\bonViewLikes\s*:').hasMatch(value))
          key,
    };
    expect(summaryOwners, {threadPath});
    expect(responseListOwners, {threadPath});

    final threadSource = sources[threadPath]!;
    expect(_occurrences(threadSource, 'PostInteractionSummary('), 1);
    expect(_occurrences(threadSource, 'onViewLikes:'), 2);

    final notificationSource =
        sources['lib/notifications/widgets/notification_row.dart']!;
    expect(notificationSource, contains('PostSummary('));
    expect(notificationSource, isNot(contains('PostInteractionSummary(')));
    expect(notificationSource, isNot(contains('onViewLikes:')));
  });
}

int _occurrences(String source, String token) =>
    token.allMatches(source).length;
