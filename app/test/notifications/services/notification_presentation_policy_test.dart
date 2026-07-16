import 'package:craftsky_app/notifications/services/notification_presentation_policy.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-016 / AT-012 requests background effects but silent foreground', () {
    const permission = NotificationPresentationPolicy.permissionRequest;
    const foreground = NotificationPresentationPolicy.foreground;

    expect(permission.alert, isTrue);
    expect(permission.sound, isTrue);
    expect(permission.badge, isFalse);
    expect(foreground.alert, isFalse);
    expect(foreground.sound, isFalse);
    expect(foreground.badge, isFalse);
    expect(foreground.vibration, isFalse);
    expect(foreground.localNotification, isFalse);
  });
}
