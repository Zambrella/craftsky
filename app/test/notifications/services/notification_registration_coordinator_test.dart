import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/notifications/models/account_subscription_id.dart';
import 'package:craftsky_app/notifications/models/foreground_notification_event.dart';
import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/notifications/models/notification_permission.dart';
import 'package:craftsky_app/notifications/services/notification_registration_coordinator.dart';
import 'package:craftsky_app/notifications/services/notification_service.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  final alice = _lease('did:plc:alice', 1);
  final bob = _lease('did:plc:bob', 2);

  test('UT-011 registers active and inactive retained accounts', () async {
    final service = _FakeService(
      permission: NotificationPermission.authorized,
      token: 'provider-token',
    );
    final registrations = <AccountSessionLease>[];
    final saved = <AccountSessionLease>[];
    final coordinator = _coordinator(
      service,
      onRegister: (lease, token) {
        registrations.add(lease);
        expect(token, 'provider-token');
      },
      onSave: saved.add,
    );

    await coordinator.updateAccounts([alice, bob]);

    expect(registrations, unorderedEquals([alice, bob]));
    expect(saved, unorderedEquals([alice, bob]));
  });

  test('UT-011 gates all accounts on installation permission', () async {
    final denied = _FakeService(
      permission: NotificationPermission.denied,
      token: 'provider-token',
    );
    var registrations = 0;
    await _coordinator(
      denied,
      onRegister: (_, _) => registrations++,
    ).updateAccounts([alice, bob]);
    expect(registrations, 0);

    final prompt = _FakeService(
      permission: NotificationPermission.notDetermined,
      token: 'provider-token',
    );
    await _coordinator(
      prompt,
      onRegister: (_, _) => registrations++,
    ).updateAccounts([alice, bob]);
    expect(prompt.permissionRequests, 1);
    expect(registrations, 2);
  });

  test(
    'IT-004 retries one account failure and refreshes every account',
    () async {
      final service = _FakeService(
        permission: NotificationPermission.authorized,
        token: 'token-a',
      );
      final attempts = <(AccountSessionLease, String)>[];
      var bobFailures = 1;
      final coordinator = _coordinator(
        service,
        onRegister: (lease, token) {
          attempts.add((lease, token));
          if (lease == bob && bobFailures-- > 0) throw Exception('transient');
        },
      );

      await coordinator.updateAccounts([alice, bob]);
      await coordinator.retryRegistration();
      await coordinator.onTokenRefresh('token-b');

      expect(attempts.where((entry) => entry.$1 == alice), [
        (alice, 'token-a'),
        (alice, 'token-b'),
      ]);
      expect(attempts.where((entry) => entry.$1 == bob), [
        (bob, 'token-a'),
        (bob, 'token-a'),
        (bob, 'token-b'),
      ]);
    },
  );

  test('UT-011 removed or reauthenticated account cannot save late', () async {
    final service = _FakeService(
      permission: NotificationPermission.authorized,
      token: 'provider-token',
    );
    final started = Completer<void>();
    final release = Completer<void>();
    final saved = <AccountSessionLease>[];
    final coordinator = _coordinator(
      service,
      onRegister: (lease, _) async {
        if (lease == bob) {
          started.complete();
          await release.future;
        }
      },
      onSave: saved.add,
    );

    final initial = coordinator.updateAccounts([alice, bob]);
    await started.future;
    final reauthenticatedBob = _lease('did:plc:bob', 3);
    final changed = coordinator.updateAccounts([alice, reauthenticatedBob]);
    release.complete();
    await Future.wait([initial, changed]);

    expect(saved, containsAll([alice, reauthenticatedBob]));
    expect(saved, isNot(contains(bob)));
  });

  test('UT-011 serializes token revisions and settles latest', () async {
    final service = _FakeService(
      permission: NotificationPermission.authorized,
      token: 'token-a',
    );
    final started = Completer<void>();
    final release = Completer<void>();
    final tokens = <String>[];
    var activeRuns = 0;
    var maxActiveRuns = 0;
    final coordinator = _coordinator(
      service,
      onRegister: (_, token) async {
        tokens.add(token);
        activeRuns++;
        maxActiveRuns = activeRuns > maxActiveRuns ? activeRuns : maxActiveRuns;
        if (token == 'token-a') {
          started.complete();
          await release.future;
        }
        activeRuns--;
      },
    );

    final readiness = coordinator.updateAccounts([alice]);
    await started.future;
    final refresh = coordinator.onTokenRefresh('token-b');
    release.complete();
    await Future.wait([readiness, refresh]);

    expect(tokens, ['token-a', 'token-b']);
    expect(maxActiveRuns, 1);
  });
}

AccountSessionLease _lease(String did, int generation) => AccountSessionLease(
  account: AccountKey(did),
  sessionGeneration: generation,
);

NotificationRegistrationCoordinator _coordinator(
  _FakeService service, {
  required FutureOr<void> Function(AccountSessionLease lease, String token)
  onRegister,
  FutureOr<void> Function(AccountSessionLease lease)? onSave,
}) => NotificationRegistrationCoordinator(
  service: service,
  platform: NotificationPlatform.ios,
  registerAccount: ({required lease, required platform, required token}) async {
    await onRegister(lease, token);
    return AccountSubscriptionId.parse('binding');
  },
  saveBindingForLease: ({required lease, required binding}) async {
    await onSave?.call(lease);
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
