import 'package:craftsky_app/notifications/services/firebase_notification_background_handler.dart';
import 'package:firebase_messaging/firebase_messaging.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('IT-013 background entry point only initializes Firebase', () async {
    var initializeCalls = 0;
    const sentinel = 'must-not-be-recorded';

    await firebaseMessagingBackgroundHandler(
      const RemoteMessage(data: {'payload': sentinel}),
      initializeFirebase: () async {
        initializeCalls++;
      },
    );

    expect(initializeCalls, 1);
  });
}
