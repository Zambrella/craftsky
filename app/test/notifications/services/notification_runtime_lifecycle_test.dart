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
    final routing = NotificationRoutingStorage(_MemoryRoutingBackend());
    await routing.replace(did, binding);
    final runtime = NotificationRuntime(
      service: service,
      registration: NotificationRegistrationCoordinator(
        service: service,
        platform: NotificationPlatform.ios,
        register: ({required platform, required token}) async => binding,
        saveBinding: ({required did, required binding}) async {},
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
  _RecordingService({required this.initialOpen});

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

final class _MemoryRoutingBackend implements NotificationRoutingStorageBackend {
  String? value;

  @override
  Future<void> delete() async => value = null;

  @override
  Future<String?> read() async => value;

  @override
  Future<void> write(String value) async => this.value = value;
}
