import 'dart:async';

import 'package:craftsky_app/notifications/data/notification_repository.dart';
import 'package:craftsky_app/notifications/models/notification_category.dart';
import 'package:craftsky_app/notifications/models/notification_preferences.dart';
import 'package:craftsky_app/notifications/providers/notification_preferences_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-010 stale failure cannot roll back a newer edit', () async {
    final repository = _PreferencesRepository();
    final container = ProviderContainer.test(
      overrides: [
        notificationPreferencesRepositoryProvider.overrideWithValue(repository),
      ],
    );
    await container.read(notificationPreferencesProvider.future);

    final first = container
        .read(notificationPreferencesProvider.notifier)
        .setPushEnabled(
          NotificationCategory.like,
          value: false,
        );
    final second = container
        .read(notificationPreferencesProvider.notifier)
        .setPushEnabled(
          NotificationCategory.like,
          value: true,
        );
    expect(
      container
          .read(notificationPreferencesProvider)
          .value!
          .known[NotificationCategory.like]!
          .pushEnabled,
      isTrue,
    );

    repository.completions[0].completeError(Exception('old failure'));
    repository.completions[1].complete(repository.initial);
    expect(await first, isFalse);
    expect(await second, isTrue);
    expect(
      container
          .read(notificationPreferencesProvider)
          .value!
          .known[NotificationCategory.like]!
          .pushEnabled,
      isTrue,
    );
  });
}

final class _PreferencesRepository
    implements NotificationPreferencesRepository {
  final initial = NotificationPreferences(
    known: {
      for (final category in NotificationCategory.preferenceValues)
        category: const NotificationPreference(
          scope: NotificationPreferenceScope.everyone,
          pushEnabled: true,
        ),
    },
    unknown: const {'future': <String, Object?>{}},
  );
  final completions = <Completer<NotificationPreferences>>[];

  @override
  Future<NotificationPreferences> load() async => initial;

  @override
  Future<NotificationPreferences> patch(NotificationPreferencePatch patch) {
    final completion = Completer<NotificationPreferences>();
    completions.add(completion);
    return completion.future;
  }
}
