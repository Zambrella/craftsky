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
}

final class _RecordingNewnessRepository
    implements NotificationNewnessRepository {
  int countCalls = 0;

  @override
  Future<int> count() async => ++countCalls;

  @override
  Future<void> markSeen() async {}
}
