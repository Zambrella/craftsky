import 'dart:async';

import 'package:craftsky_app/notifications/data/api_notification_repository.dart';
import 'package:craftsky_app/notifications/data/notification_api_client.dart';
import 'package:craftsky_app/notifications/models/foreground_notification_event.dart';
import 'package:craftsky_app/notifications/models/notification_category.dart';
import 'package:craftsky_app/notifications/models/notification_effect.dart';
import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/notifications/models/notification_permission.dart';
import 'package:craftsky_app/notifications/services/foreground_notification_handler.dart';
import 'package:craftsky_app/notifications/services/notification_coordinator.dart';
import 'package:craftsky_app/notifications/services/notification_registration_coordinator.dart';
import 'package:craftsky_app/notifications/services/notification_resolution_policy.dart';
import 'package:craftsky_app/notifications/services/notification_routing_storage.dart';
import 'package:craftsky_app/notifications/services/notification_runtime.dart';
import 'package:craftsky_app/notifications/services/notification_service.dart';
import 'package:craftsky_app/notifications/services/notification_service_owner.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

void main() {
  test(
    'IT-004 / IT-012 resolves only matching current-DID opens',
    () async {
      const idValue = '00000000-0000-0000-0000-000000000001';
      final alice = Did.parse('did:plc:alice');
      final binding = AccountSubscriptionId.parse('alice_binding');
      final dio = Dio(BaseOptions(baseUrl: 'https://appview.example.com'));
      var resolveCalls = 0;
      DioAdapter(dio: dio).onGet(
        '/v1/notifications/$idValue',
        (server) {
          resolveCalls++;
          server.reply(200, {
            'id': idValue,
            'type': 'futureCategory',
            'state': 'active',
            'target': {
              'kind': 'post',
              'uri': 'at://did:plc:actor/social.craftsky.feed.post/server',
            },
          });
        },
      );
      final repository = ApiNotificationRepository(NotificationApiClient(dio));
      final routing = NotificationRoutingStorage(_MemoryRoutingBackend());
      await routing.replace(alice, binding);
      final effects = StreamController<NotificationEffect>.broadcast();
      final runtime = _runtime(
        routing: routing,
        resolutionRepository: repository,
        effects: effects,
      );
      addTearDown(effects.close);
      addTearDown(runtime.dispose);
      await runtime.updateReadiness(did: alice, onboarded: true);

      final resolvedEffect = effects.stream.first;
      await runtime.receiveOpen(_event(idValue, binding));
      final resolved = await resolvedEffect;

      expect(resolveCalls, 1);
      expect(resolved, isA<NotificationNavigationEffect>());
      expect(
        (resolved as NotificationNavigationEffect).outcome.destination,
        NotificationDestination.post(
          AtUri.parse(
            'at://did:plc:actor/social.craftsky.feed.post/server',
          ),
        ),
      );

      final unavailableEffect = effects.stream.first;
      await runtime.receiveOpen(
        _event(idValue, AccountSubscriptionId.parse('stale_binding')),
      );
      expect(await unavailableEffect, isA<NotificationUnavailableEffect>());
      expect(resolveCalls, 1);

      await runtime.updateReadiness(did: null, onboarded: false);
      await runtime.receiveOpen(_event(idValue, binding));
      await Future<void>.delayed(Duration.zero);
      expect(resolveCalls, 1);
    },
  );
}

NotificationOpenEvent _event(
  String id,
  AccountSubscriptionId binding,
) => NotificationOpenEvent(
  notificationId: NotificationId.parse(id),
  category: NotificationCategory.unknown,
  accountSubscriptionId: binding,
  source: NotificationOpenSource.backgroundOpen,
);

NotificationRuntime _runtime({
  required NotificationRoutingStorage routing,
  required ApiNotificationRepository resolutionRepository,
  required StreamController<NotificationEffect> effects,
}) {
  final service = _FakeNotificationService();
  final registration = NotificationRegistrationCoordinator(
    platform: NotificationPlatform.ios,
    getToken: service.getToken,
    register: ({required platform, required token}) async =>
        AccountSubscriptionId.parse('unused'),
    saveBinding: ({required did, required binding}) async {},
  );
  return NotificationRuntime(
    coordinator: NotificationCoordinator(
      service: service,
      registration: registration,
    ),
    owner: NotificationServiceOwner(
      service: service,
      onTokenRefresh: registration.onTokenRefresh,
      onForegroundEvent: (_) {},
      onOpen: (_) {},
    ),
    routingStorage: routing,
    resolutionRepository: resolutionRepository,
    foregroundHandler: ForegroundNotificationHandler(
      showBanner: (_) {},
      invalidateList: () {},
      refreshCount: () {},
    ),
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
  Stream<NotificationOpenEvent> get openedNotifications => const Stream.empty();

  @override
  Future<void> openSystemNotificationSettings() async {}

  @override
  Future<NotificationPermission> requestPermission() async =>
      NotificationPermission.authorized;

  @override
  Future<NotificationOpenEvent?> takeInitialOpen() async => null;

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
