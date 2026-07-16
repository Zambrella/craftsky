import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

void main() {
  final notificationFiles = Directory('lib/notifications')
      .listSync(recursive: true)
      .whereType<File>()
      .where((file) => file.path.endsWith('.dart'))
      .toList();

  test('REG-009 Firebase listener APIs stay in the single adapter', () {
    const listenerTokens = [
      'FirebaseMessaging.onMessage',
      'FirebaseMessaging.onMessageOpenedApp',
      '_messaging.onTokenRefresh',
      '_messaging.getInitialMessage()',
    ];
    final offenders = <String>[];
    for (final file in notificationFiles) {
      final source = file.readAsStringSync();
      if (listenerTokens.any(source.contains) &&
          file.path !=
              'lib/notifications/services/firebase_notification_service.dart') {
        offenders.add(file.path);
      }
    }

    expect(offenders, isEmpty);
  });

  test('REG-009 one root mounts notification effects', () {
    final appSource = File('lib/app.dart').readAsStringSync();
    expect(_occurrences(appSource, 'NotificationEffectHost('), 1);
    for (final file
        in Directory('lib')
            .listSync(recursive: true)
            .whereType<File>()
            .where((file) => file.path.endsWith('.dart'))) {
      if (file.path == 'lib/app.dart' ||
          file.path ==
              'lib/notifications/widgets/notification_effect_host.dart') {
        continue;
      }
      expect(
        file.readAsStringSync(),
        isNot(contains('NotificationEffectHost(')),
        reason: 'Second notification effect host in ${file.path}',
      );
    }
  });

  test('REG-004 has no polling, icon badge, event store, or open queue', () {
    final allSource = notificationFiles
        .map((file) => file.readAsStringSync())
        .join('\n');
    expect(allSource, isNot(matches(RegExp(r'\bTimer(?:\.periodic)?\s*\('))));
    expect(
      allSource.toLowerCase(),
      isNot(anyOf(contains('flutter_app_badger'), contains('updateappbadge'))),
    );

    final pendingOpen = File(
      'lib/notifications/services/pending_notification_open.dart',
    ).readAsStringSync();
    expect(pendingOpen, isNot(contains('dart:io')));
    expect(pendingOpen, isNot(contains('shared_preferences')));
    expect(pendingOpen, isNot(contains('flutter_secure_storage')));
    expect(pendingOpen, isNot(matches(RegExp(r'\b(read|write|delete)\s*\('))));

    final routingStorage = File(
      'lib/notifications/services/notification_routing_storage.dart',
    ).readAsStringSync();
    expect(routingStorage, isNot(contains('notificationId')));
    expect(routingStorage, isNot(contains('ForegroundNotificationEvent')));
  });

  test('notification providers use Riverpod code generation', () {
    final providerFiles = Directory('lib/notifications/providers')
        .listSync()
        .whereType<File>()
        .where(
          (file) =>
              file.path.endsWith('.dart') && !file.path.endsWith('.g.dart'),
        );

    for (final file in providerFiles) {
      final source = file.readAsStringSync();
      final basename = file.uri.pathSegments.last.replaceFirst('.dart', '');
      expect(
        source,
        contains("part '$basename.g.dart';"),
        reason: '${file.path} must include its generated Riverpod part',
      );
      expect(
        source,
        anyOf(contains('@riverpod'), contains('@Riverpod(')),
        reason: '${file.path} must declare generated providers',
      );
      expect(
        source,
        isNot(
          matches(
            RegExp(
              r'\b(?:Provider|FutureProvider|StreamProvider|'
              r'NotifierProvider|AsyncNotifierProvider)\s*<',
            ),
          ),
        ),
        reason: '${file.path} must not construct providers manually',
      );
    }
  });
}

int _occurrences(String source, String token) =>
    token.allMatches(source).length;
