import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/notifications/data/notification_repository.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_seen_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('IT-006 seen state remains account-scoped', () async {
    final alice = AccountKey('did:plc:alice');
    final bob = AccountKey('did:plc:bob');
    final aliceRepository = _RecordingNewnessRepository();
    final bobRepository = _RecordingNewnessRepository();
    final repositories = {alice: aliceRepository, bob: bobRepository};
    final container = ProviderContainer.test(
      overrides: [
        accountNotificationNewnessRepositoryProvider.overrideWith(
          (ref, account) async => repositories[account]!,
        ),
      ],
    );
    final aliceSeen = await container.read(
      accountNotificationSeenProvider(alice).future,
    );
    final bobSeen = await container.read(
      accountNotificationSeenProvider(bob).future,
    );

    await aliceSeen.afterSuccessfulRender(1);
    await bobSeen.afterSuccessfulRender(1);

    expect(aliceRepository.markSeenCalls, 1);
    expect(bobRepository.markSeenCalls, 1);
  });

  test('UT-008 / AT-007 acknowledges each render token once', () async {
    final repository = _RecordingNewnessRepository();
    var countRefreshes = 0;
    final coordinator = NotificationSeenCoordinator(
      repository: repository,
      refreshCount: () async => countRefreshes++,
    );

    await coordinator.afterSuccessfulRender(1);
    await coordinator.afterSuccessfulRender(1);
    await coordinator.afterSuccessfulRender(2);

    expect(repository.markSeenCalls, 2);
    expect(countRefreshes, 2);
  });
}

final class _RecordingNewnessRepository
    implements NotificationNewnessRepository {
  int markSeenCalls = 0;

  @override
  Future<int> count() async => 0;

  @override
  Future<void> markSeen() async => markSeenCalls++;
}
