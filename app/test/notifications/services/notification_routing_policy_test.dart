import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/notifications/services/notification_routing_policy.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('UT-003 routing binding policy', () {
    final current = AccountSubscriptionId.parse('current_binding');
    final other = AccountSubscriptionId.parse('other_binding');

    test('permits only an exact current-account binding match', () {
      expect(
        NotificationRoutingPolicy.canResolve(
          storedBinding: current,
          eventBinding: current,
        ),
        isTrue,
      );
      expect(
        NotificationRoutingPolicy.canResolve(
          storedBinding: current,
          eventBinding: other,
        ),
        isFalse,
      );
      expect(
        NotificationRoutingPolicy.canResolve(
          storedBinding: null,
          eventBinding: current,
        ),
        isFalse,
      );
    });
  });
}
