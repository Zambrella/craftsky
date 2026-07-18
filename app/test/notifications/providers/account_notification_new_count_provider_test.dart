import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/notifications/data/notification_repository.dart';
import 'package:craftsky_app/notifications/models/notification_badge.dart';
import 'package:craftsky_app/notifications/providers/notification_new_count_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'UT-009 caches and refreshes authoritative counts per account',
    () async {
      final alice = AccountKey('did:plc:alice');
      final bob = AccountKey('did:plc:bob');
      final aliceRepository = _CountRepository([3]);
      final bobRepository = _CountRepository([120, 120]);
      final repositories = {alice: aliceRepository, bob: bobRepository};
      final container = ProviderContainer.test(
        retry: (_, _) => null,
        overrides: [
          accountNotificationNewnessRepositoryProvider.overrideWith(
            (ref, account) async => repositories[account]!,
          ),
        ],
      );

      expect(
        await container.read(accountNotificationNewCountProvider(alice).future),
        3,
      );
      expect(
        await container.read(accountNotificationNewCountProvider(bob).future),
        120,
      );
      await container
          .read(accountNotificationNewCountProvider(bob).notifier)
          .refresh();

      expect(
        container.read(accountNotificationNewCountProvider(alice)).value,
        3,
      );
      expect(
        container.read(accountNotificationNewCountProvider(bob)).value,
        120,
      );
      expect(aliceRepository.countCalls, 1);
      expect(bobRepository.countCalls, 2);
      expect(NotificationBadge.fromCount(0).visible, isFalse);
      expect(NotificationBadge.fromCount(99).label, '99');
      expect(NotificationBadge.fromCount(100).label, '99+');
      expect(NotificationBadge.fromCount(120).label, '99+');
    },
  );

  test('UT-009 one account count failure does not disturb another', () async {
    final alice = AccountKey('did:plc:alice');
    final bob = AccountKey('did:plc:bob');
    final repositories = <AccountKey, NotificationNewnessRepository>{
      alice: _CountRepository([3]),
      bob: _CountRepository([Exception('offline')]),
    };
    final container = ProviderContainer.test(
      retry: (_, _) => null,
      overrides: [
        accountNotificationNewnessRepositoryProvider.overrideWith(
          (ref, account) async => repositories[account]!,
        ),
      ],
    );

    expect(
      await container.read(accountNotificationNewCountProvider(alice).future),
      3,
    );
    await expectLater(
      container.read(accountNotificationNewCountProvider(bob).future),
      throwsA(isA<Exception>()),
    );
    expect(container.read(accountNotificationNewCountProvider(alice)).value, 3);
  });
}

final class _CountRepository implements NotificationNewnessRepository {
  _CountRepository(this.responses);

  final List<Object> responses;
  int countCalls = 0;

  @override
  Future<int> count() async {
    final response = responses[countCalls++];
    if (response case final Exception error) throw error;
    return response as int;
  }

  @override
  Future<void> markSeen() async {}
}
