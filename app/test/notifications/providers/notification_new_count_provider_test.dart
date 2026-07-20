import 'package:craftsky_app/notifications/data/notification_repository.dart';
import 'package:craftsky_app/notifications/providers/notification_new_count_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
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
  _RecordingNewnessRepository({this.initialCount});

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
