import 'dart:async';

import 'package:craftsky_app/notifications/models/account_subscription_id.dart';
import 'package:craftsky_app/notifications/models/foreground_notification_event.dart';
import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/notifications/models/notification_permission.dart';
import 'package:craftsky_app/notifications/services/notification_registration_coordinator.dart';
import 'package:craftsky_app/notifications/services/notification_service.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  final alice = Did.parse('did:plc:alice');
  final bob = Did.parse('did:plc:bob');

  group('UT-001 / UT-013 / IT-002 registration lifecycle', () {
    test('gates permission and registration on account readiness', () async {
      final service = _FakeService(
        permission: NotificationPermission.notDetermined,
        token: 'token',
      );
      var registrations = 0;
      final coordinator = _coordinator(
        service,
        onRegister: (_) => registrations++,
      );

      await coordinator.updateReadiness(did: null, onboarded: false);
      await coordinator.updateReadiness(did: alice, onboarded: false);
      expect(service.permissionRequests, 0);
      expect(registrations, 0);

      await coordinator.updateReadiness(did: alice, onboarded: true);

      expect(service.permissionRequests, 1);
      expect(registrations, 1);
    });

    test(
      'defers refreshes and registers the latest token when ready',
      () async {
        final service = _FakeService(
          permission: NotificationPermission.authorized,
        );
        final tokens = <String>[];
        final coordinator = _coordinator(
          service,
          onRegister: tokens.add,
        );

        await coordinator.onTokenRefresh('tokenA');
        await coordinator.onTokenRefresh('tokenB');
        expect(tokens, isEmpty);

        await coordinator.updateReadiness(did: alice, onboarded: true);

        expect(tokens, ['tokenB']);
      },
    );

    test('skips empty tokens and retries failure on a later trigger', () async {
      final service = _FakeService(
        permission: NotificationPermission.authorized,
        token: '',
      );
      var attempts = 0;
      final coordinator = _coordinator(
        service,
        onRegister: (_) {
          attempts++;
          if (attempts == 1) throw Exception('transient');
        },
      );

      await coordinator.updateReadiness(did: alice, onboarded: true);
      expect(attempts, 0);

      service.token = 'tokenC';
      await coordinator.retryRegistration();
      expect(attempts, 1);

      await Future<void>.delayed(Duration.zero);
      expect(attempts, 1, reason: 'failure must not create a retry loop');

      await coordinator.retryRegistration();
      expect(attempts, 2);
    });

    test('never registers after readiness becomes ineligible', () async {
      final service = _FakeService(
        permission: NotificationPermission.authorized,
        token: 'token',
      );
      var registrations = 0;
      final coordinator = _coordinator(
        service,
        onRegister: (_) => registrations++,
      );

      await coordinator.updateReadiness(did: null, onboarded: false);
      await coordinator.onTokenRefresh('newToken');

      expect(registrations, 0);
    });

    test('keeps the app usable when permission lookup fails', () async {
      final service = _FakeService(
        permission: NotificationPermission.authorized,
        token: 'token',
      )..permissionError = Exception('provider unavailable');

      await _coordinator(
        service,
        onRegister: (_) => fail('must not register'),
      ).updateReadiness(did: alice, onboarded: true);
    });

    test('rechecks denied permission on resume without prompting', () async {
      final service = _FakeService(
        permission: NotificationPermission.denied,
        token: 'token',
      );
      var registrations = 0;
      final coordinator = _coordinator(
        service,
        onRegister: (_) => registrations++,
      );

      await coordinator.updateReadiness(did: alice, onboarded: true);
      expect(registrations, 0);

      service.permission = NotificationPermission.authorized;
      await coordinator.retryRegistration();

      expect(service.permissionChecks, 2);
      expect(service.permissionRequests, 0);
      expect(registrations, 1);
    });

    test('serializes overlap and finishes with the latest token', () async {
      final firstRegistrationStarted = Completer<void>();
      final releaseFirstRegistration = Completer<void>();
      final registrations = <String>[];
      var activeRegistrations = 0;
      var maxActiveRegistrations = 0;
      final service = _FakeService(
        permission: NotificationPermission.authorized,
        token: 'tokenA',
      );
      final coordinator = _coordinator(
        service,
        onRegister: (token) async {
          registrations.add(token);
          activeRegistrations++;
          if (activeRegistrations > maxActiveRegistrations) {
            maxActiveRegistrations = activeRegistrations;
          }
          if (token == 'tokenA') {
            firstRegistrationStarted.complete();
            await releaseFirstRegistration.future;
          }
          activeRegistrations--;
        },
      );

      final readiness = coordinator.updateReadiness(
        did: alice,
        onboarded: true,
      );
      await firstRegistrationStarted.future;
      final refresh = coordinator.onTokenRefresh('tokenB');
      releaseFirstRegistration.complete();
      await Future.wait([readiness, refresh]);

      expect(registrations, ['tokenA', 'tokenB']);
      expect(maxActiveRegistrations, 1);
    });

    test('drops an old-DID result and registers the current DID', () async {
      final firstRegistrationStarted = Completer<void>();
      final releaseFirstRegistration = Completer<void>();
      final registrations = <String>[];
      final savedDids = <Did>[];
      final service = _FakeService(
        permission: NotificationPermission.authorized,
        token: 'token',
      );
      final coordinator = _coordinator(
        service,
        onRegister: (token) async {
          registrations.add(token);
          if (registrations.length == 1) {
            firstRegistrationStarted.complete();
            await releaseFirstRegistration.future;
          }
        },
        onSave: savedDids.add,
      );

      final aliceReadiness = coordinator.updateReadiness(
        did: alice,
        onboarded: true,
      );
      await firstRegistrationStarted.future;
      final bobReadiness = coordinator.updateReadiness(
        did: bob,
        onboarded: true,
      );
      await Future<void>.delayed(Duration.zero);
      releaseFirstRegistration.complete();
      await Future.wait([aliceReadiness, bobReadiness]);

      expect(registrations, ['token', 'token']);
      expect(savedDids, [bob]);
    });
  });
}

NotificationRegistrationCoordinator _coordinator(
  _FakeService service, {
  required FutureOr<void> Function(String token) onRegister,
  FutureOr<void> Function(Did did)? onSave,
}) => NotificationRegistrationCoordinator(
  service: service,
  platform: NotificationPlatform.ios,
  register: ({required platform, required token}) async {
    await onRegister(token);
    return AccountSubscriptionId.parse('binding');
  },
  saveBinding: ({required did, required binding}) async {
    await onSave?.call(did);
  },
);

final class _FakeService implements NotificationService {
  _FakeService({required this.permission, this.token});

  NotificationPermission permission;
  String? token;
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
  Future<String?> getToken() async => token;

  @override
  Future<void> initialize() async {}

  @override
  Stream<NotificationOpenAttempt> get openedNotifications =>
      const Stream.empty();

  @override
  Future<void> openSystemNotificationSettings() async {}

  @override
  Future<NotificationPermission> requestPermission() async {
    permissionRequests++;
    return permission = NotificationPermission.authorized;
  }

  @override
  Future<NotificationOpenAttempt?> takeInitialOpen() async => null;

  @override
  Stream<String> get tokenRefreshes => const Stream.empty();
}
