import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/notifications/data/notification_repository.dart';
import 'package:craftsky_app/notifications/providers/notification_new_count_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'account refresh does not dispatch after its provider rebuilds',
    () async {
      final account = AccountKey('did:plc:alice');
      final repository = _QueuedNewnessRepository([
        Future.value(3),
        Future.value(9),
        Future.value(11),
      ]);
      final provider = accountNotificationNewCountProvider(account);
      final container = ProviderContainer.test(
        overrides: [
          accountNotificationNewnessRepositoryProvider.overrideWith(
            (ref, account) async => repository,
          ),
        ],
      );
      await container.read(provider.future);

      final operation = container.read(provider.notifier).refresh();
      container.invalidate(provider);
      expect(await container.read(provider.future), 9);
      await operation;

      expect(repository.countCalls, 2);
    },
  );

  for (final accountScoped in [false, true]) {
    for (final fails in [false, true]) {
      test(
        'refresh cannot replace a rebuilt count '
        '(account=$accountScoped, failure=$fails)',
        () async {
          final account = AccountKey('did:plc:alice');
          final pending = Completer<int>();
          final repository = _QueuedNewnessRepository([
            Future.value(3),
            pending.future,
            Future.value(9),
          ]);
          final container = ProviderContainer.test(
            overrides: [
              notificationNewnessRepositoryProvider.overrideWithValue(
                repository,
              ),
              accountNotificationNewnessRepositoryProvider.overrideWith(
                (ref, account) async => repository,
              ),
            ],
          );
          final provider = accountScoped
              ? accountNotificationNewCountProvider(account)
              : notificationNewCountProvider;
          await container.read(provider.future);
          final operation = accountScoped
              ? container
                    .read(accountNotificationNewCountProvider(account).notifier)
                    .refresh()
              : container.read(notificationNewCountProvider.notifier).refresh();
          await container.pump();
          expect(repository.countCalls, 2);

          container.invalidate(provider);
          expect(await container.read(provider.future), 9);
          if (fails) {
            pending.completeError(Exception('obsolete refresh failure'));
          } else {
            pending.complete(2);
          }
          await operation;

          expect(container.read(provider), isA<AsyncData<int>>());
          expect(container.read(provider).requireValue, 9);
        },
      );
    }
  }

  test('UT-007 refreshes once for each explicit lifecycle request', () async {
    final repository = _RecordingNewnessRepository();
    final container = ProviderContainer.test(
      overrides: [
        notificationNewnessRepositoryProvider.overrideWithValue(repository),
      ],
    );

    await container.read(notificationNewCountProvider.future);
    expect(repository.countCalls, 1);

    await container.read(notificationNewCountProvider.notifier).refresh();
    await container.read(notificationNewCountProvider.notifier).refresh();

    expect(repository.countCalls, 3);
  });

  test(
    'AT-006 suppress decrements the loaded badge without going below zero',
    () async {
      final repository = _RecordingNewnessRepository(initialCount: 3);
      final container = ProviderContainer.test(
        overrides: [
          notificationNewnessRepositoryProvider.overrideWithValue(repository),
        ],
      );

      await container.read(notificationNewCountProvider.future);
      container.read(notificationNewCountProvider.notifier)
        ..suppress(2)
        ..suppress(4);

      expect(container.read(notificationNewCountProvider).requireValue, 0);
    },
  );
}

final class _RecordingNewnessRepository
    implements NotificationNewnessRepository {
  new({this.initialCount});

  final int? initialCount;
  int countCalls = 0;

  @override
  Future<int> count() async {
    countCalls++;
    return initialCount ?? countCalls;
  }

  @override
  Future<void> markSeen() async {}
}

final class _QueuedNewnessRepository implements NotificationNewnessRepository {
  new(this.responses);

  final List<Future<int>> responses;
  int countCalls = 0;

  @override
  Future<int> count() => responses[countCalls++];

  @override
  Future<void> markSeen() async {}
}
