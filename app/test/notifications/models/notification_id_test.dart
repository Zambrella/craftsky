import 'package:craftsky_app/notifications/models/notification_id.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('parses UUID wire values and redacts string output', () {
    const wireValue = '018f47a2-4b0e-7f39-a621-9f6f6c75e312';
    final id = NotificationId.parse(wireValue);

    expect(id.wireValue, wireValue);
    expect(id, NotificationId.parse(wireValue));
    expect(id.toString(), '<redacted-notification-id>');
  });

  test('rejects malformed notification IDs', () {
    for (final value in ['', 'not-a-uuid', '018f47a2-4b0e-7f39-a621']) {
      expect(() => NotificationId.parse(value), throwsFormatException);
    }
  });
}
