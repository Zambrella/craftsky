import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/notifications/data/notification_repository.dart';
import 'package:craftsky_app/notifications/models/notification_category.dart';
import 'package:craftsky_app/notifications/models/notification_preferences.dart';
import 'package:craftsky_app/notifications/providers/notification_preferences_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('IT-006 preferences and mutations remain account-scoped', () async {
    final alice = AccountKey('did:plc:alice');
    final bob = AccountKey('did:plc:bob');
    final aliceRepository = _PreferencesRepository();
    final bobRepository = _PreferencesRepository(pushEnabled: false);
    final repositories = {alice: aliceRepository, bob: bobRepository};
    final container = ProviderContainer.test(
      overrides: [
        accountNotificationPreferencesRepositoryProvider.overrideWith(
          (ref, account) async => repositories[account]!,
        ),
      ],
    );

    final alicePreferences = await container.read(
      accountNotificationPreferencesProvider(alice).future,
    );
    final bobPreferences = await container.read(
      accountNotificationPreferencesProvider(bob).future,
    );
    expect(
      alicePreferences.known[NotificationCategory.like]!.pushEnabled,
      isTrue,
    );
    expect(
      bobPreferences.known[NotificationCategory.like]!.pushEnabled,
      isFalse,
    );

    final edit = container
        .read(accountNotificationPreferencesProvider(bob).notifier)
        .setPushEnabled(NotificationCategory.like, value: true);
    await Future<void>.delayed(Duration.zero);
    bobRepository.completions.single.complete(bobRepository.initial);
    await edit;
    expect(aliceRepository.completions, isEmpty);
    expect(bobRepository.completions, hasLength(1));
  });

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
  _PreferencesRepository({bool pushEnabled = true})
    : initial = NotificationPreferences(
        known: {
          for (final category in NotificationCategory.preferenceValues)
            category: NotificationPreference(
              scope: NotificationPreferenceScope.everyone,
              pushEnabled: pushEnabled,
            ),
        },
        unknown: const {'future': <String, Object?>{}},
      );

  final NotificationPreferences initial;
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
