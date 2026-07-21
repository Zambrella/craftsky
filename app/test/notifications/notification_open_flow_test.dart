import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/account_activation_coordinator.dart';
import 'package:craftsky_app/notifications/models/account_subscription_id.dart';
import 'package:craftsky_app/notifications/models/foreground_notification_event.dart';
import 'package:craftsky_app/notifications/models/notification_effect.dart';
import 'package:craftsky_app/notifications/models/notification_destination.dart';
import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/notifications/models/notification_permission.dart';
import 'package:craftsky_app/notifications/services/notification_registration_coordinator.dart';
import 'package:craftsky_app/notifications/services/notification_routing_storage.dart';
import 'package:craftsky_app/notifications/services/notification_runtime.dart';
import 'package:craftsky_app/notifications/services/notification_service.dart';
import 'package:craftsky_app/notifications/widgets/notification_row.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-008 rejects a row outside its producing account lease', () {
    var registry = _registry();
    final alice = registry.activeLease!.session;

    expect(canOpenNotificationRow(alice, registry), isTrue);
    registry = registry.activate(
      registry.leaseFor(AccountKey('did:plc:bob'))!,
    );
    expect(canOpenNotificationRow(alice, registry), isFalse);
  });

  test('IT-005 activates exact recipient before runtime navigation', () async {
    var registry = _registry();
    final operations = <String>[];
    final effects = StreamController<NotificationEffect>.broadcast();
    final runtime = _runtime(
      routing: NotificationRoutingStorage(() => registry),
      effects: effects,
      activate: (lease) async {
        operations.add('activate');
        registry = registry.activate(lease);
        return AccountActivationResult.activated;
      },
    );
    addTearDown(runtime.dispose);
    addTearDown(effects.close);
    await runtime.updateReadiness(
      did: Did.parse('did:plc:alice'),
      onboarded: true,
    );

    final effectFuture = effects.stream.first;
    await runtime.receiveOpen(_attempt('bob_binding'));
    final effect = await effectFuture;
    operations.add('navigate');

    expect(effect, isA<NotificationNavigationEffect>());
    expect(operations, ['activate', 'navigate']);
    expect(registry.activeDid?.value, 'did:plc:bob');
  });

  test('IT-005 separates invalid and removed-account outcomes', () async {
    final registry = _registry();
    final effects = StreamController<NotificationEffect>.broadcast();
    final runtime = _runtime(
      routing: NotificationRoutingStorage(() => registry),
      effects: effects,
    );
    addTearDown(runtime.dispose);
    addTearDown(effects.close);
    await runtime.updateReadiness(
      did: Did.parse('did:plc:alice'),
      onboarded: true,
    );

    final invalid = effects.stream.first;
    await runtime.receiveOpen(_attempt(null));
    expect(await invalid, isA<NotificationUnavailableEffect>());

    final removed = effects.stream.first;
    await runtime.receiveOpen(_attempt('removed_binding'));
    expect(await removed, isA<NotificationRemovedAccountEffect>());
  });

  test('IT-005 identifies an inactive foreground recipient', () async {
    final registry = _registry();
    final refreshedAccounts = <AccountKey>[];
    final effects = StreamController<NotificationEffect>.broadcast();
    final runtime = _runtime(
      routing: NotificationRoutingStorage(() => registry),
      effects: effects,
      refreshAccountCount: refreshedAccounts.add,
    );
    addTearDown(runtime.dispose);
    addTearDown(effects.close);

    final banner = effects.stream.first;
    await runtime.receiveForegroundEvent(
      ForegroundNotificationEvent(
        title: 'A liked your post',
        body: 'Open the notification',
        openAttempt: _attempt('bob_binding'),
      ),
    );
    final effect = await banner as NotificationBannerEffect;

    expect(effect.event.title, 'A liked your post');
    expect(effect.recipient?.handle, 'bob.test');
    expect(effect.recipient?.avatarUrl, isNull);
    expect(effect.resolution, isA<ExactNotificationRecipient>());
    expect(effect.toString(), isNot(contains('bob.test')));

    await runtime.receiveForegroundEvent(
      ForegroundNotificationEvent(
        title: 'Duplicate delivery',
        body: 'Same server state',
        openAttempt: _attempt('bob_binding'),
      ),
    );
    expect(refreshedAccounts, [
      AccountKey('did:plc:bob'),
      AccountKey('did:plc:bob'),
    ]);
  });

  test(
    'IT-017 activates the exact retained account before Instagram navigation',
    () async {
      var registry = _registry();
      final operations = <String>[];
      final effects = StreamController<NotificationEffect>.broadcast();
      final runtime = _runtime(
        routing: NotificationRoutingStorage(() => registry),
        effects: effects,
        activate: (lease) async {
          operations.add('activate:${lease.account.did.value}');
          registry = registry.activate(lease);
          return AccountActivationResult.activated;
        },
      );
      addTearDown(runtime.dispose);
      addTearDown(effects.close);
      await runtime.updateReadiness(
        did: Did.parse('did:plc:alice'),
        onboarded: true,
      );

      final effectFuture = effects.stream.first;
      await runtime.receiveOpen(_instagramAttempt('bob_binding'));
      final effect = await effectFuture as NotificationNavigationEffect;

      expect(operations, ['activate:did:plc:bob']);
      expect(registry.activeDid?.value, 'did:plc:bob');
      expect(effect.outcome.destination, const InstagramMigrationDestination());
    },
  );
}

SessionRegistry _registry() {
  final base = SessionRegistry.empty()
      .upsertAndActivate(
        token: 'bob-token',
        did: 'did:plc:bob',
        handle: 'bob.test',
      )
      .upsertAndActivate(
        token: 'alice-token',
        did: 'did:plc:alice',
        handle: 'alice.test',
      );
  return SessionRegistry(
    nextSessionGeneration: base.nextSessionGeneration,
    nextUseOrdinal: base.nextUseOrdinal,
    activationGeneration: base.activationGeneration,
    activeDid: base.activeDid?.value,
    sessions: {
      for (final entry in base.sessions.entries) entry.key.value: entry.value,
    },
    routingBindings: const {
      'did:plc:alice': 'alice_binding',
      'did:plc:bob': 'bob_binding',
    },
  );
}

NotificationOpenAttempt _attempt(String? binding) =>
    NotificationOpenAttempt.fromProviderData({
      'payloadVersion': '1',
      'type': 'everythingElse',
      'accountSubscriptionId': ?binding,
    });

NotificationOpenAttempt _instagramAttempt(String? binding) =>
    NotificationOpenAttempt.fromProviderData({
      'payloadVersion': '1',
      'type': 'instagramMatch',
      'accountSubscriptionId': ?binding,
      'notificationId': '00000000-0000-0000-0000-000000000321',
      'count': '3',
      'countCapped': 'false',
      'destination': 'instagramMigration',
    });

NotificationRuntime _runtime({
  required NotificationRoutingStorage routing,
  required StreamController<NotificationEffect> effects,
  Future<AccountActivationResult> Function(
    AccountSessionLease lease,
  )?
  activate,
  void Function(AccountKey account)? refreshAccountCount,
}) {
  final service = _FakeNotificationService();
  final registration = NotificationRegistrationCoordinator(
    service: service,
    platform: NotificationPlatform.ios,
    registerAccount:
        ({required lease, required platform, required token}) async =>
            AccountSubscriptionId.parse('unused'),
    saveBindingForLease: ({required lease, required binding}) async {},
  );
  return NotificationRuntime(
    service: service,
    registration: registration,
    routingStorage: routing,
    invalidateList: () {},
    refreshCount: () {},
    effects: effects,
    activateRecipient: activate,
    refreshAccountCount: refreshAccountCount,
  );
}

final class _FakeNotificationService implements NotificationService {
  @override
  Future<void> deleteToken() async {}
  @override
  Future<void> dispose() async {}
  @override
  Stream<ForegroundNotificationEvent> get foregroundEvents =>
      const Stream.empty();
  @override
  Future<NotificationPermission> getPermission() async =>
      NotificationPermission.authorized;
  @override
  Future<String?> getToken() async => null;
  @override
  Future<void> initialize() async {}
  @override
  Stream<NotificationOpenAttempt> get openedNotifications =>
      const Stream.empty();
  @override
  Future<void> openSystemNotificationSettings() async {}
  @override
  Future<NotificationPermission> requestPermission() async =>
      NotificationPermission.authorized;
  @override
  Future<NotificationOpenAttempt?> takeInitialOpen() async => null;
  @override
  Stream<String> get tokenRefreshes => const Stream.empty();
}
