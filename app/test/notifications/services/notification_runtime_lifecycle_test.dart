import 'dart:async';

import 'package:craftsky_app/notifications/data/notification_repository.dart';
import 'package:craftsky_app/notifications/models/account_subscription_id.dart';
import 'package:craftsky_app/notifications/models/foreground_notification_event.dart';
import 'package:craftsky_app/notifications/models/notification_category.dart';
import 'package:craftsky_app/notifications/models/notification_effect.dart';
import 'package:craftsky_app/notifications/models/notification_id.dart';
import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/notifications/models/notification_permission.dart';
import 'package:craftsky_app/notifications/models/notification_resolution.dart';
import 'package:craftsky_app/notifications/services/notification_registration_coordinator.dart';
import 'package:craftsky_app/notifications/services/notification_routing_storage.dart';
import 'package:craftsky_app/notifications/services/notification_runtime.dart';
import 'package:craftsky_app/notifications/services/notification_service.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'UT-012 / UT-018 / IT-010 runtime owns streams and foreground effects',
    () async {
      final service = _RecordingService();
      final effects = StreamController<NotificationEffect>.broadcast();
      var listInvalidations = 0;
      var countRefreshes = 0;
      final registration = NotificationRegistrationCoordinator(
        service: service,
        platform: NotificationPlatform.ios,
        register: ({required platform, required token}) async =>
            AccountSubscriptionId.parse('binding'),
        saveBinding: ({required did, required binding}) async {},
      );
      final runtime = NotificationRuntime(
        service: service,
        registration: registration,
        routingStorage: NotificationRoutingStorage(_MemoryRoutingBackend()),
        resolutionRepository: _UnusedResolutionRepository(),
        invalidateList: () => listInvalidations++,
        refreshCount: () => countRefreshes++,
        effects: effects,
      );
      addTearDown(effects.close);
      addTearDown(service.close);

      await runtime.start();
      await runtime.start();

      expect(service.initializeCalls, 1);
      expect(service.initialOpenCalls, 1);
      expect(service.tokenController.hasListener, isTrue);
      expect(service.eventController.hasListener, isTrue);
      expect(service.openController.hasListener, isTrue);

      final event = _foregroundEvent();
      final receivedEffects = effects.stream.take(2).toList();
      service.eventController
        ..add(event)
        ..add(event);

      expect(
        await receivedEffects,
        everyElement(isA<NotificationBannerEffect>()),
      );
      await Future<void>.delayed(Duration.zero);
      expect(listInvalidations, 2);
      expect(countRefreshes, 2);

      await runtime.dispose();

      expect(service.disposeCalls, 1);
      expect(service.tokenController.hasListener, isFalse);
      expect(service.eventController.hasListener, isFalse);
      expect(service.openController.hasListener, isFalse);
    },
  );
}

ForegroundNotificationEvent _foregroundEvent() => ForegroundNotificationEvent(
  title: 'New activity',
  body: 'Someone interacted with your work',
  openEvent: NotificationOpenEvent(
    notificationId: NotificationId.parse(
      '00000000-0000-0000-0000-000000000001',
    ),
    category: NotificationCategory.like,
    accountSubscriptionId: AccountSubscriptionId.parse('binding'),
    source: NotificationOpenSource.foregroundBanner,
  ),
);

final class _RecordingService implements NotificationService {
  final tokenController = StreamController<String>.broadcast();
  final eventController =
      StreamController<ForegroundNotificationEvent>.broadcast();
  final openController = StreamController<NotificationOpenEvent>.broadcast();
  int initializeCalls = 0;
  int initialOpenCalls = 0;
  int disposeCalls = 0;

  Future<void> close() async {
    await tokenController.close();
    await eventController.close();
    await openController.close();
  }

  @override
  Future<void> deleteToken() async {}

  @override
  Future<void> dispose() async => disposeCalls++;

  @override
  Stream<ForegroundNotificationEvent> get foregroundEvents =>
      eventController.stream;

  @override
  Future<NotificationPermission> getPermission() async =>
      NotificationPermission.denied;

  @override
  Future<String?> getToken() async => null;

  @override
  Future<void> initialize() async => initializeCalls++;

  @override
  Stream<NotificationOpenEvent> get openedNotifications =>
      openController.stream;

  @override
  Future<void> openSystemNotificationSettings() async {}

  @override
  Future<NotificationPermission> requestPermission() async =>
      NotificationPermission.denied;

  @override
  Future<NotificationOpenEvent?> takeInitialOpen() async {
    initialOpenCalls++;
    return null;
  }

  @override
  Stream<String> get tokenRefreshes => tokenController.stream;
}

final class _UnusedResolutionRepository
    implements NotificationResolutionRepository {
  @override
  Future<NotificationResolution> resolve(NotificationId id) =>
      throw UnimplementedError();
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
