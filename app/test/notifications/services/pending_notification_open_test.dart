import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/notifications/services/pending_notification_open.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-008 and AT-008 retain latest and clear at sign-in', () {
    final pending = PendingNotificationOpen();
    final first = _attempt(NotificationOpenSource.foregroundBanner);
    final second = _attempt(NotificationOpenSource.initialOpen);

    expect(
      pending.receive(first, readiness: NotificationOpenReadiness.transient),
      isNull,
    );
    expect(
      pending.receive(second, readiness: NotificationOpenReadiness.transient),
      isNull,
    );
    expect(
      pending.updateReadiness(NotificationOpenReadiness.ready),
      same(second),
    );
    expect(pending.updateReadiness(NotificationOpenReadiness.ready), isNull);

    pending.receive(first, readiness: NotificationOpenReadiness.transient);
    expect(
      pending.updateReadiness(NotificationOpenReadiness.requiresSignIn),
      isNull,
    );
    expect(pending.updateReadiness(NotificationOpenReadiness.ready), isNull);
  });
}

NotificationOpenAttempt _attempt(NotificationOpenSource source) =>
    NotificationOpenAttempt.fromProviderData(
      {
        'payloadVersion': '1',
        'type': 'everythingElse',
        'accountSubscriptionId': 'binding',
      },
      source: source,
    );
