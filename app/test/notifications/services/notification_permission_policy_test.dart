import 'package:craftsky_app/notifications/models/notification_permission.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('UT-001 permission readiness policy', () {
    test('requests only for signed-in onboarded undetermined state', () {
      for (final testCase
          in <
            ({
              bool signedIn,
              bool onboarded,
              NotificationPermission permission,
              NotificationPermissionAction expected,
            })
          >[
            (
              signedIn: false,
              onboarded: false,
              permission: NotificationPermission.notDetermined,
              expected: NotificationPermissionAction.none,
            ),
            (
              signedIn: true,
              onboarded: false,
              permission: NotificationPermission.notDetermined,
              expected: NotificationPermissionAction.none,
            ),
            (
              signedIn: true,
              onboarded: true,
              permission: NotificationPermission.notDetermined,
              expected: NotificationPermissionAction.request,
            ),
            (
              signedIn: true,
              onboarded: true,
              permission: NotificationPermission.authorized,
              expected: NotificationPermissionAction.register,
            ),
            (
              signedIn: true,
              onboarded: true,
              permission: NotificationPermission.denied,
              expected: NotificationPermissionAction.none,
            ),
          ]) {
        expect(
          NotificationPermissionPolicy.actionFor(
            signedIn: testCase.signedIn,
            onboarded: testCase.onboarded,
            permission: testCase.permission,
          ),
          testCase.expected,
        );
      }
    });
  });
}
