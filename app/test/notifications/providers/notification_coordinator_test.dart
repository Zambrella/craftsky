import 'dart:async';

import 'package:craftsky_app/notifications/models/foreground_notification_event.dart';
import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/notifications/models/notification_permission.dart';
import 'package:craftsky_app/notifications/services/notification_coordinator.dart';
import 'package:craftsky_app/notifications/services/notification_registration_coordinator.dart';
import 'package:craftsky_app/notifications/services/notification_service.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  final alice = Did.parse('did:plc:alice');

  test(
    'IT-002 / AT-002 gates permission and registration on readiness',
    () async {
      final service = _FakeService(NotificationPermission.notDetermined);
      var registrations = 0;
      final registration = NotificationRegistrationCoordinator(
        platform: NotificationPlatform.android,
        getToken: service.getToken,
        register: ({required platform, required token}) async {
          registrations++;
          return AccountSubscriptionId.parse('binding');
        },
        saveBinding: ({required did, required binding}) async {},
      );
      final coordinator = NotificationCoordinator(
        service: service,
        registration: registration,
      );

      await coordinator.updateReadiness(did: null, onboarded: false);
      await coordinator.updateReadiness(did: alice, onboarded: false);
      expect(service.permissionRequests, 0);
      expect(registrations, 0);

      await coordinator.updateReadiness(did: alice, onboarded: true);
      expect(service.permissionRequests, 1);
      expect(registrations, 1);
    },
  );

  test('IT-002 keeps readiness usable when permission lookup fails', () async {
    final service = _FakeService(NotificationPermission.authorized)
      ..permissionError = Exception('provider unavailable');
    final registration = NotificationRegistrationCoordinator(
      platform: NotificationPlatform.ios,
      getToken: service.getToken,
      register: ({required platform, required token}) async =>
          AccountSubscriptionId.parse('binding'),
      saveBinding: ({required did, required binding}) async {},
    );

    await NotificationCoordinator(
      service: service,
      registration: registration,
    ).updateReadiness(did: alice, onboarded: true);
  });

  test('IT-002 retries eligible registration when the app resumes', () async {
    final service = _FakeService(NotificationPermission.authorized);
    var registrations = 0;
    final coordinator = NotificationCoordinator(
      service: service,
      registration: NotificationRegistrationCoordinator(
        platform: NotificationPlatform.ios,
        getToken: service.getToken,
        register: ({required platform, required token}) async {
          registrations++;
          return AccountSubscriptionId.parse('binding-$registrations');
        },
        saveBinding: ({required did, required binding}) async {},
      ),
    );

    await coordinator.updateReadiness(did: alice, onboarded: true);
    await coordinator.retryRegistration();

    expect(registrations, 2);
  });

  test(
    'IT-002 rechecks denied permission and registers when resume is authorized',
    () async {
      final service = _FakeService(NotificationPermission.denied);
      var registrations = 0;
      final coordinator = NotificationCoordinator(
        service: service,
        registration: NotificationRegistrationCoordinator(
          platform: NotificationPlatform.ios,
          getToken: service.getToken,
          register: ({required platform, required token}) async {
            registrations++;
            return AccountSubscriptionId.parse('binding-$registrations');
          },
          saveBinding: ({required did, required binding}) async {},
        ),
      );

      await coordinator.updateReadiness(did: alice, onboarded: true);
      expect(service.permissionChecks, 1);
      expect(registrations, 0);

      service.permission = NotificationPermission.authorized;
      await coordinator.retryRegistration();

      expect(service.permissionChecks, 2);
      expect(service.permissionRequests, 0);
      expect(registrations, 1);
    },
  );
}

final class _FakeService implements NotificationService {
  _FakeService(this.permission);

  NotificationPermission permission;
  Object? permissionError;
  int permissionChecks = 0;
  int permissionRequests = 0;

  @override
  Future<void> deleteToken() async {}

  @override
  Future<void> dispose() async {}

  @override
  Stream<ForegroundNotificationEvent> get foregroundEvents =>
      const Stream.empty();

  @override
  Future<NotificationPermission> getPermission() async {
    permissionChecks++;
    if (permissionError case final error?) throw Exception(error);
    return permission;
  }

  @override
  Future<String?> getToken() async => 'token';

  @override
  Future<void> initialize() async {}

  @override
  Stream<NotificationOpenEvent> get openedNotifications => const Stream.empty();

  @override
  Future<void> openSystemNotificationSettings() async {}

  @override
  Future<NotificationPermission> requestPermission() async {
    permissionRequests++;
    return permission = NotificationPermission.authorized;
  }

  @override
  Future<NotificationOpenEvent?> takeInitialOpen() async => null;

  @override
  Stream<String> get tokenRefreshes => const Stream.empty();
}
