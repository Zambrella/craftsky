import 'dart:async';

import 'package:craftsky_app/notifications/models/notification_effect.dart';
import 'package:craftsky_app/notifications/providers/notification_lifecycle_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_new_count_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_service_provider.dart';
import 'package:craftsky_app/notifications/providers/notifications_provider.dart';
import 'package:craftsky_app/notifications/services/notification_registration_coordinator.dart';
import 'package:craftsky_app/notifications/services/notification_runtime.dart';
import 'package:flutter/foundation.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'notification_runtime_provider.g.dart';

@Riverpod(keepAlive: true)
Raw<Stream<NotificationEffect>> notificationEffectStream(Ref ref) =>
    ref.watch(_notificationEffectControllerProvider).stream;

@Riverpod(keepAlive: true)
StreamController<NotificationEffect> _notificationEffectController(Ref ref) {
  final controller = StreamController<NotificationEffect>.broadcast();
  ref.onDispose(controller.close);
  return controller;
}

@Riverpod(keepAlive: true)
NotificationRuntime notificationRuntime(Ref ref) {
  final service = ref.watch(notificationServiceProvider);
  final registration = NotificationRegistrationCoordinator(
    service: service,
    platform: defaultTargetPlatform == TargetPlatform.android
        ? NotificationPlatform.android
        : NotificationPlatform.ios,
    register: ref.watch(notificationDeviceRepositoryProvider).register,
    saveBinding: ({required did, required binding}) =>
        ref.read(notificationRoutingStorageProvider).replace(did, binding),
  );
  final effects = ref.watch(_notificationEffectControllerProvider);
  final runtime = NotificationRuntime(
    service: service,
    registration: registration,
    routingStorage: ref.watch(notificationRoutingStorageProvider),
    invalidateList: () => ref.invalidate(notificationsProvider),
    refreshCount: () =>
        ref.read(notificationNewCountProvider.notifier).refresh(),
    effects: effects,
  );
  unawaited(runtime.start());
  ref.onDispose(() => unawaited(runtime.dispose()));
  return runtime;
}
