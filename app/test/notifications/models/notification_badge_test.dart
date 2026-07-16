import 'package:craftsky_app/notifications/models/notification_badge.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-006 formats hidden, literal, and capped badges accessibly', () {
    expect(NotificationBadge.fromCount(0).visible, isFalse);
    expect(NotificationBadge.fromCount(1).label, '1');
    expect(NotificationBadge.fromCount(99).label, '99');
    expect(NotificationBadge.fromCount(100).label, '99+');
  });
}
