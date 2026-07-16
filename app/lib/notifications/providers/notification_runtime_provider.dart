import 'dart:async';

import 'package:craftsky_app/notifications/models/notification_effect.dart';
import 'package:craftsky_app/notifications/providers/notification_lifecycle_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_new_count_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_service_provider.dart';
import 'package:craftsky_app/notifications/providers/notifications_provider.dart';
import 'package:craftsky_app/notifications/services/foreground_notification_handler.dart';
import 'package:craftsky_app/notifications/services/notification_coordinator.dart';
import 'package:craftsky_app/notifications/services/notification_registration_coordinator.dart';
import 'package:craftsky_app/notifications/services/notification_runtime.dart';
import 'package:craftsky_app/notifications/services/notification_service_owner.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

final notificationEffectStreamProvider = Provider<Stream<NotificationEffect>>(
  (ref) => ref.watch(_notificationEffectControllerProvider).stream,
);

final _notificationEffectControllerProvider =
    Provider<StreamController<NotificationEffect>>((ref) {
      final controller = StreamController<NotificationEffect>.broadcast();
      ref.onDispose(controller.close);
      return controller;
    });

final notificationRuntimeProvider = Provider<NotificationRuntime>((ref) {
  final service = ref.watch(notificationServiceProvider);
  final registration = NotificationRegistrationCoordinator(
    platform: defaultTargetPlatform == TargetPlatform.android
        ? NotificationPlatform.android
        : NotificationPlatform.ios,
    getToken: service.getToken,
    register: ref.watch(notificationDeviceRepositoryProvider).register,
    saveBinding: ({required did, required binding}) =>
        ref.read(notificationRoutingStorageProvider).replace(did, binding),
  );
  final effects = ref.watch(_notificationEffectControllerProvider);
  late final NotificationRuntime runtime;
  final foreground = ForegroundNotificationHandler(
    showBanner: (event) => effects.add(NotificationBannerEffect(event)),
    invalidateList: () => ref.invalidate(notificationsProvider),
    refreshCount: () => ref
        .read(notificationNewCountProvider.notifier)
        .refreshFor(NotificationNewCountTrigger.foregroundEvent),
  );
  final owner = NotificationServiceOwner(
    service: service,
    onTokenRefresh: registration.onTokenRefresh,
    onForegroundEvent: (event) => runtime.receiveForegroundEvent(event),
    onOpen: (event) => runtime.receiveOpen(event),
  );
  runtime = NotificationRuntime(
    coordinator: NotificationCoordinator(
      service: service,
      registration: registration,
    ),
    owner: owner,
    routingStorage: ref.watch(notificationRoutingStorageProvider),
    resolutionRepository: ref.watch(notificationResolutionRepositoryProvider),
    foregroundHandler: foreground,
    effects: effects,
  );
  unawaited(runtime.start());
  ref.onDispose(() => unawaited(runtime.dispose()));
  return runtime;
});
