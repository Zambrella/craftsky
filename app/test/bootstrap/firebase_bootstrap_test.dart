import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

void main() {
  test('AT-001 initializes Firebase before constructing messaging', () {
    final source = File(
      'lib/notifications/services/firebase_notification_bootstrap.dart',
    ).readAsStringSync();
    final initialize = source.indexOf('Firebase.initializeApp');
    final construct = source.indexOf('return FirebaseNotificationService(');

    expect(initialize, greaterThanOrEqualTo(0));
    expect(construct, greaterThan(initialize));
  });
}
