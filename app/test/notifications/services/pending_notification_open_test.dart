import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/notifications/services/notification_routing_storage.dart';
import 'package:craftsky_app/notifications/services/pending_notification_open.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-007 retains only latest work and its resolved generation', () {
    final pending = PendingNotificationOpen();
    final first = _work(NotificationOpenSource.foregroundBanner, generation: 1);
    final second = _work(NotificationOpenSource.initialOpen, generation: 2);

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

PendingNotificationOpenWork _work(
  NotificationOpenSource source, {
  required int generation,
}) => PendingNotificationOpenWork(
  attempt: NotificationOpenAttempt.fromProviderData(
    {
      'payloadVersion': '1',
      'type': 'everythingElse',
      'accountSubscriptionId': 'binding',
    },
    source: source,
  ),
  resolution: ExactNotificationRecipient(
    AccountSessionLease(
      account: AccountKey('did:plc:recipient'),
      sessionGeneration: generation,
    ),
  ),
);
