import 'dart:async';

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
    'UT-007 IT-003 AT-001 emit direct outcomes without a resolver',
    () async {
      final alice = Did.parse('did:plc:alice');
      final binding = AccountSubscriptionId.parse('alice_binding');
      final routing = NotificationRoutingStorage(_MemoryRoutingBackend());
      await routing.replace(alice, binding);
      final effects = StreamController<NotificationEffect>.broadcast();
      final runtime = _runtime(routing: routing, effects: effects);
      addTearDown(effects.close);
      addTearDown(runtime.dispose);
      await runtime.updateReadiness(did: alice, onboarded: true);

      final directEffect = effects.stream.first;
      await runtime.receiveOpen(
        _attempt(
          binding: binding.wireValue,
          type: 'like',
          subjectUri: 'at://did:plc:actor/social.craftsky.feed.post/subject',
        ),
      );
      final direct = await directEffect as NotificationNavigationEffect;

      expect(
        direct.outcome.destination,
        PostDestination(
          AtUri.parse(
            'at://did:plc:actor/social.craftsky.feed.post/subject',
          ),
        ),
      );
      expect(direct.outcome.feedback, isNull);

      final legacyEffect = effects.stream.first;
      await runtime.receiveOpen(
        NotificationOpenAttempt.fromProviderData({
          'type': 'like',
          'accountSubscriptionId': binding.wireValue,
          'notificationId': '00000000-0000-0000-0000-000000000001',
        }),
      );
      final legacy = await legacyEffect as NotificationNavigationEffect;
      expect(legacy.outcome.destination, const NotificationsDestination());
      expect(legacy.outcome.feedback, NotificationOpenFeedback.unableToOpen);

      final unavailableEffect = effects.stream.first;
      await runtime.receiveOpen(
        _attempt(binding: 'stale_binding', type: 'everythingElse'),
      );
      expect(await unavailableEffect, isA<NotificationUnavailableEffect>());
    },
  );

  test(
    'IR-003 discards an in-flight open across account readiness changes',
    () async {
      final alice = Did.parse('did:plc:alice');
      final bob = Did.parse('did:plc:bob');
      final binding = AccountSubscriptionId.parse('alice_binding');

      for (final nextDid in <Did?>[null, bob]) {
        final backend = _BlockingRoutingBackend();
        final routing = NotificationRoutingStorage(backend);
        await routing.replace(alice, binding);
        final effects = StreamController<NotificationEffect>.broadcast();
        final emitted = <NotificationEffect>[];
        final subscription = effects.stream.listen(emitted.add);
        final runtime = _runtime(routing: routing, effects: effects);
        await runtime.updateReadiness(did: alice, onboarded: true);
        backend.blockNextRead();

        final open = runtime.receiveOpen(
          _attempt(binding: binding.wireValue, type: 'everythingElse'),
        );
        await backend.readStarted;
        await runtime.updateReadiness(
          did: nextDid,
          onboarded: nextDid != null,
        );
        backend.releaseRead();
        await open;
        await Future<void>.delayed(Duration.zero);

        expect(
          emitted,
          isEmpty,
          reason: nextDid == null
              ? 'sign-in-required must discard the old in-flight open'
              : 'an account switch must discard the old in-flight open',
        );

        await subscription.cancel();
        await runtime.dispose();
        await effects.close();
      }
    },
  );
}

NotificationOpenAttempt _attempt({
  required String binding,
  required String type,
  String? subjectUri,
}) => NotificationOpenAttempt.fromProviderData({
  'payloadVersion': '1',
  'type': type,
  'accountSubscriptionId': binding,
  'subjectUri': ?subjectUri,
});

NotificationRuntime _runtime({
  required NotificationRoutingStorage routing,
  required StreamController<NotificationEffect> effects,
}) {
  final service = _FakeNotificationService();
  final registration = NotificationRegistrationCoordinator(
    service: service,
    platform: NotificationPlatform.ios,
    register: ({required platform, required token}) async =>
        AccountSubscriptionId.parse('unused'),
    saveBinding: ({required did, required binding}) async {},
  );
  return NotificationRuntime(
    service: service,
    registration: registration,
    routingStorage: routing,
    invalidateList: () {},
    refreshCount: () {},
    effects: effects,
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

final class _MemoryRoutingBackend implements NotificationRoutingStorageBackend {
  String? value;

  @override
  Future<void> delete() async => value = null;

  @override
  Future<String?> read() async => value;

  @override
  Future<void> write(String value) async => this.value = value;
}

final class _BlockingRoutingBackend
    implements NotificationRoutingStorageBackend {
  String? value;
  Completer<String?>? _blockedRead;
  late Completer<void> _readStarted;

  Future<void> get readStarted => _readStarted.future;

  void blockNextRead() {
    _blockedRead = Completer<String?>();
    _readStarted = Completer<void>();
  }

  void releaseRead() {
    final blockedRead = _blockedRead!;
    _blockedRead = null;
    blockedRead.complete(value);
  }

  @override
  Future<void> delete() async => value = null;

  @override
  Future<String?> read() {
    final blockedRead = _blockedRead;
    if (blockedRead == null) return Future.value(value);
    _readStarted.complete();
    return blockedRead.future;
  }

  @override
  Future<void> write(String value) async => this.value = value;
}
