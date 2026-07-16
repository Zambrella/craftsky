import 'package:craftsky_app/notifications/models/account_subscription_id.dart';
import 'package:craftsky_app/notifications/models/notification_category.dart';
import 'package:craftsky_app/notifications/models/notification_id.dart';
import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/notifications/services/pending_notification_open.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-014 / AT-011 retains latest only through transient readiness', () {
    final pending = PendingNotificationOpen();
    final first = _event('00000000-0000-0000-0000-000000000001');
    final second = _event('00000000-0000-0000-0000-000000000002');

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

NotificationOpenEvent _event(String id) => NotificationOpenEvent(
  notificationId: NotificationId.parse(id),
  category: NotificationCategory.like,
  accountSubscriptionId: AccountSubscriptionId.parse('binding'),
  source: NotificationOpenSource.backgroundOpen,
);
