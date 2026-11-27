import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/account_activation_coordinator.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/notifications/models/account_subscription_id.dart';
import 'package:craftsky_app/notifications/models/foreground_notification_event.dart';
import 'package:craftsky_app/notifications/models/notification_destination.dart';
import 'package:craftsky_app/notifications/models/notification_effect.dart';
import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/notifications/models/notification_permission.dart';
import 'package:craftsky_app/notifications/services/notification_registration_coordinator.dart';
import 'package:craftsky_app/notifications/services/notification_routing_storage.dart';
import 'package:craftsky_app/notifications/services/notification_runtime.dart';
import 'package:craftsky_app/notifications/services/notification_service.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'IT-013 chat callbacks resolve privately and refresh no social state',
    () async {
      final did = Did.parse('did:plc:viewer');
      final binding = AccountSubscriptionId.parse('binding');
      final base = SessionRegistry.empty().upsertAndActivate(
        token: 'synthetic',
        did: did.value,
        handle: 'viewer.test',
      );
      final registry = SessionRegistry(
        nextSessionGeneration: base.nextSessionGeneration,
        nextUseOrdinal: base.nextUseOrdinal,
        activationGeneration: base.activationGeneration,
        activeDid: base.activeDid?.value,
        sessions: {
          for (final entry in base.sessions.entries)
            entry.key.value: entry.value,
        },
        routingBindings: {did.value: binding.wireValue},
      );
      final attempt = NotificationOpenAttempt.fromProviderData({
        'payloadVersion': '1',
        'type': 'chat',
        'accountSubscriptionId': 'binding',
        'openReference': '00000000-0000-4000-8000-000000000123',
      });
      final service = _RecordingService(initialOpen: attempt);
      final effects = StreamController<NotificationEffect>.broadcast();
      final received = <NotificationEffect>[];
      final subscription = effects.stream.listen(received.add);
      var social = 0;
      var chat = 0;
      var resolves = 0;
      final runtime = NotificationRuntime(
        service: service,
        registration: NotificationRegistrationCoordinator(
          service: service,
          platform: NotificationPlatform.ios,
          registerAccount: ({
            required lease,
            required platform,
            required token,
          }) async => binding,
          saveBindingForLease: ({required lease, required binding}) async {},
        ),
        routingStorage: NotificationRoutingStorage(() => registry),
        invalidateList: () => social++,
        refreshCount: () => social++,
        effects: effects,
        refreshChatCount: (account) {
          expect(account.did, did);
          chat++;
        },
        resolveChat: (lease, reference) async {
          resolves++;
          return ChatConversationId.parse('conversation');
        },
      );
      addTearDown(subscription.cancel);
      addTearDown(effects.close);
      addTearDown(service.close);
      addTearDown(runtime.dispose);
      await runtime.start();
      expect(resolves, 0);
      await runtime.updateReadiness(did: did, onboarded: true);
      await Future<void>.delayed(Duration.zero);
      expect(resolves, 1);
      expect(
        received
            .whereType<NotificationNavigationEffect>()
            .single
            .outcome
            .destination,
        isA<ChatConversationDestination>(),
      );
      await runtime.receiveForegroundEvent(
        ForegroundNotificationEvent(
          title: 'Peer',
          body: '',
          openAttempt: attempt,
        ),
      );
      expect(social, 0);
      expect(chat, 1);
      expect(resolves, 1);
    },
  );

  test('cold-start open waits for account initialization', () async {
    final did = Did.parse('did:plc:viewer');
    final binding = AccountSubscriptionId.parse('binding');
    final base = SessionRegistry.empty().upsertAndActivate(
      token: 'token',
      did: did.value,
      handle: 'viewer.test',
    );
    final registry = SessionRegistry(
      nextSessionGeneration: base.nextSessionGeneration,
      nextUseOrdinal: base.nextUseOrdinal,
      activationGeneration: base.activationGeneration,
      activeDid: base.activeDid?.value,
      sessions: {
        for (final entry in base.sessions.entries) entry.key.value: entry.value,
      },
      routingBindings: {did.value: binding.wireValue},
    );
    final service = _RecordingService(
      initialOpen: _attempt(NotificationOpenSource.initialOpen),
    );
    final effects = StreamController<NotificationEffect>.broadcast();
    final received = <NotificationNavigationEffect>[];
    final subscription = effects.stream
        .where((effect) => effect is NotificationNavigationEffect)
        .cast<NotificationNavigationEffect>()
        .listen(received.add);
    SessionRegistry? hydratedRegistry;
    final runtime = NotificationRuntime(
      service: service,
      registration: NotificationRegistrationCoordinator(
        service: service,
        platform: NotificationPlatform.ios,
        registerAccount: ({
          required lease,
          required platform,
          required token,
        }) async => binding,
        saveBindingForLease: ({required lease, required binding}) async {},
      ),
      routingStorage: NotificationRoutingStorage(() => hydratedRegistry),
      invalidateList: () {},
      refreshCount: () {},
      effects: effects,
    );
    addTearDown(subscription.cancel);
    addTearDown(effects.close);
    addTearDown(service.close);
    addTearDown(runtime.dispose);

    await runtime.updateReadiness(
      did: null,
      onboarded: false,
      initialized: false,
    );
    await runtime.start();
    expect(received, isEmpty);
    hydratedRegistry = registry;
    await runtime.updateReadiness(did: did, onboarded: true);
    expect(received, hasLength(1));
    expect(
      received.single.outcome.destination,
      ProfileDestination(Did.parse('did:plc:actor')),
    );
  });

  for (final switchAway in [false, true]) {
    test('deferred navigation is bound to the initialized recipient '
        '(switch away=$switchAway)', () async {
      const service = UnavailableNotificationService();
      final binding = AccountSubscriptionId.parse('bob_binding');
      final alice = AccountKey('did:plc:alice');
      final bob = AccountKey('did:plc:bob');
      var registry = SessionRegistry.empty()
          .upsertAndActivate(
            token: 'bob-token',
            did: bob.did.value,
            handle: 'bob.test',
          )
          .upsertAndActivate(
            token: 'alice-token',
            did: alice.did.value,
            handle: 'alice.test',
          );
      final recipient = registry.leaseFor(bob)!;
      registry = registry.saveRoutingBinding(recipient, binding.wireValue);
      final effects = StreamController<NotificationEffect>.broadcast();
      final received = <NotificationEffect>[];
      final subscription = effects.stream.listen(received.add);
      var resolutions = 0;
      late final NotificationRuntime runtime;
      runtime = NotificationRuntime(
        service: service,
        registration: NotificationRegistrationCoordinator(
          service: service,
          platform: NotificationPlatform.ios,
          registerAccount: ({
            required lease,
            required platform,
            required token,
          }) async => binding,
          saveBindingForLease: ({required lease, required binding}) async {},
        ),
        routingStorage: NotificationRoutingStorage(() => registry),
        invalidateList: () {},
        refreshCount: () {},
        effects: effects,
        activateRecipient: (lease) async {
          registry = registry.activate(lease);
          await runtime.updateReadiness(
            did: null,
            onboarded: false,
            initialized: false,
          );
          return AccountActivationResult.activated;
        },
        resolveChat: (lease, _) async {
          expect(lease, recipient);
          resolutions++;
          return ChatConversationId.parse('conversation');
        },
      );
      addTearDown(effects.close);
      addTearDown(subscription.cancel);
      addTearDown(runtime.dispose);
      await runtime.updateReadiness(did: alice.did, onboarded: true);

      await runtime.receiveOpen(
        NotificationOpenAttempt.fromProviderData({
          'payloadVersion': '1',
          'type': 'chat',
          'accountSubscriptionId': binding.wireValue,
          'openReference': '00000000-0000-4000-8000-000000000123',
        }),
      );
      expect(resolutions, 1);
      expect(received, isEmpty);
      if (switchAway) {
        // Re-selecting Bob has a new activation even though his saved session
        // and subscription are unchanged.
        registry = registry.activate(registry.leaseFor(alice)!);
        registry = registry.activate(recipient);
      }
      await runtime.updateReadiness(did: bob.did, onboarded: true);

      expect(resolutions, 1, reason: 'the resolved destination is retained');
      if (switchAway) {
        expect(received, isEmpty);
      } else {
        final outcome =
            (received.single as NotificationNavigationEffect).outcome;
        expect(outcome.destination, isA<ChatConversationDestination>());
        expect(outcome.isCurrent?.call(), isTrue);
      }
    });
  }

  test('IT-004 AT-004 REG-010 unify callback sources at least once', () async {
    final binding = AccountSubscriptionId.parse('binding');
    final did = Did.parse('did:plc:viewer');
    final foreground = ForegroundNotificationEvent(
      title: 'New activity',
      body: 'Someone followed you',
      openAttempt: _attempt(NotificationOpenSource.foregroundBanner),
    );
    final background = _attempt(NotificationOpenSource.backgroundOpen);
    final initial = _attempt(NotificationOpenSource.initialOpen);
    final service = _RecordingService(initialOpen: initial);
    final effects = StreamController<NotificationEffect>.broadcast();
    final baseRegistry = SessionRegistry.empty().upsertAndActivate(
      token: 'viewer-token',
      did: did.value,
      handle: 'viewer.test',
    );
    final registry = SessionRegistry(
      nextSessionGeneration: baseRegistry.nextSessionGeneration,
      nextUseOrdinal: baseRegistry.nextUseOrdinal,
      activationGeneration: baseRegistry.activationGeneration,
      activeDid: baseRegistry.activeDid?.value,
      sessions: {
        for (final entry in baseRegistry.sessions.entries)
          entry.key.value: entry.value,
      },
      routingBindings: {did.value: binding.wireValue},
    );
    final routing = NotificationRoutingStorage(() => registry);
    final runtime = NotificationRuntime(
      service: service,
      registration: NotificationRegistrationCoordinator(
        service: service,
        platform: NotificationPlatform.ios,
        registerAccount: ({
          required lease,
          required platform,
          required token,
        }) async => binding,
        saveBindingForLease: ({required lease, required binding}) async {},
      ),
      routingStorage: routing,
      invalidateList: () {},
      refreshCount: () {},
      effects: effects,
    );
    addTearDown(effects.close);
    addTearDown(service.close);
    addTearDown(runtime.dispose);
    await runtime.updateReadiness(did: did, onboarded: true);

    final outcomes = effects.stream
        .where((effect) => effect is NotificationNavigationEffect)
        .cast<NotificationNavigationEffect>()
        .take(4)
        .toList();
    await runtime.start();
    service.openController
      ..add(background)
      ..add(background);
    await runtime.receiveOpen(foreground.openAttempt);

    final received = await outcomes;
    expect(service.initializeCalls, 1);
    expect(service.initialOpenCalls, 1);
    expect(received, hasLength(4));
    expect(
      received.map((effect) => effect.outcome.destination),
      everyElement(ProfileDestination(Did.parse('did:plc:actor'))),
    );
  });
}

NotificationOpenAttempt _attempt(NotificationOpenSource source) =>
    NotificationOpenAttempt.fromProviderData(
      {
        'payloadVersion': '1',
        'type': 'follow',
        'accountSubscriptionId': 'binding',
        'actorDid': 'did:plc:actor',
      },
      source: source,
    );

final class _RecordingService implements NotificationService {
  new({required this.initialOpen});

  final NotificationOpenAttempt initialOpen;
  final openController = StreamController<NotificationOpenAttempt>.broadcast();
  int initializeCalls = 0;
  int initialOpenCalls = 0;

  Future<void> close() => openController.close();

  @override
  Future<void> deleteToken() async {}

  @override
  Future<void> dispose() async {}

  @override
  Stream<ForegroundNotificationEvent> get foregroundEvents =>
      const Stream.empty();

  @override
  Future<NotificationPermission> getPermission() async =>
      NotificationPermission.denied;

  @override
  Future<String?> getToken() async => null;

  @override
  Future<void> initialize() async => initializeCalls++;

  @override
  Stream<NotificationOpenAttempt> get openedNotifications =>
      openController.stream;

  @override
  Future<void> openSystemNotificationSettings() async {}

  @override
  Future<NotificationPermission> requestPermission() async =>
      NotificationPermission.denied;

  @override
  Future<NotificationOpenAttempt?> takeInitialOpen() async {
    initialOpenCalls++;
    return initialOpen;
  }

  @override
  Stream<String> get tokenRefreshes => const Stream.empty();
}
