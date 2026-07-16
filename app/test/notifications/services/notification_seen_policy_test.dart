import 'package:craftsky_app/notifications/services/notification_seen_policy.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-008 / AT-007 acknowledges each successful render token once', () {
    final gate = NotificationSeenGate();

    expect(gate.consume(token: null, rendered: false), isFalse);
    expect(gate.consume(token: 1, rendered: false), isFalse);
    expect(gate.consume(token: 1, rendered: true), isTrue);
    expect(gate.consume(token: 1, rendered: true), isFalse);
    expect(gate.consume(token: 2, rendered: true), isTrue);
  });
}
