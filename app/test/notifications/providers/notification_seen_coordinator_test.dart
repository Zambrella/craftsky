import 'package:craftsky_app/notifications/data/notification_repository.dart';
import 'package:craftsky_app/notifications/providers/notification_seen_provider.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
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
