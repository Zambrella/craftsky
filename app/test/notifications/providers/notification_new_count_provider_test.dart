import 'package:craftsky_app/notifications/providers/notification_new_count_provider.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-007 accepts only the five event-driven refresh triggers', () {
    expect(
      [
        NotificationNewCountTrigger.ready,
        NotificationNewCountTrigger.resume,
        NotificationNewCountTrigger.foregroundEvent,
        NotificationNewCountTrigger.pageRefresh,
        NotificationNewCountTrigger.markSeen,
      ].map(NotificationNewCountPolicy.shouldRefresh),
      everyElement(isTrue),
    );
    expect(
      NotificationNewCountPolicy.shouldRefresh(
        NotificationNewCountTrigger.elapsedTimer,
      ),
      isFalse,
    );
    expect(
      NotificationNewCountPolicy.shouldRefresh(
        NotificationNewCountTrigger.unrelatedRebuild,
      ),
      isFalse,
    );
  });
}
