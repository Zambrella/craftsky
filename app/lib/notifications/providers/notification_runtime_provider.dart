import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/providers/account_activation_coordinator.dart';
import 'package:craftsky_app/auth/providers/account_boundary_provider.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/auth/providers/unsaved_work_guard_provider.dart';
import 'package:craftsky_app/notifications/models/notification_effect.dart';
import 'package:craftsky_app/notifications/providers/notification_lifecycle_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_new_count_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_service_provider.dart';
import 'package:craftsky_app/notifications/providers/notifications_provider.dart';
import 'package:craftsky_app/notifications/services/notification_registration_coordinator.dart';
import 'package:craftsky_app/notifications/services/notification_runtime.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_status_provider.dart';
import 'package:craftsky_app/router/router.dart';
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
  final registrySnapshot = ref.watch(sessionRegistryProvider).value;
  final onboardingStatuses = {
    if (registrySnapshot != null)
      for (final account in registrySnapshot.sessions.keys)
        registrySnapshot.leaseFor(AccountKey(account.value))!: ref.watch(
          onboardingStatusProvider(
            registrySnapshot.leaseFor(AccountKey(account.value))!,
          ),
        ),
  };
  final service = ref.watch(notificationServiceProvider);
  final registration = NotificationRegistrationCoordinator(
    service: service,
    platform: defaultTargetPlatform == TargetPlatform.android
        ? NotificationPlatform.android
        : NotificationPlatform.ios,
    registerAccount:
        ({required lease, required platform, required token}) async {
          final repository = await ref.read(
            accountNotificationDeviceRepositoryProvider(lease.account).future,
          );
          return repository.register(platform: platform, token: token);
        },
    saveBindingForLease: ({required lease, required binding}) async {
      await ref
          .read(sessionRegistryProvider.notifier)
          .saveRoutingBinding(lease, binding);
    },
  );
  final effects = ref.watch(_notificationEffectControllerProvider);
  final activation = AccountActivationCoordinator(
    readRegistry: () => ref.read(sessionRegistryProvider).requireValue,
    commitActivation: ref.read(sessionRegistryProvider.notifier).activate,
    invalidateAccountState: ref.read(accountStateInvalidatorProvider),
    resetToHome: () async =>
        ref.read(goRouterProvider).go(const FeedRoute().location),
    confirmLeave: ref.read(unsavedWorkGuardProvider).confirmLeave,
  );
  final runtime = NotificationRuntime(
    service: service,
    registration: registration,
    routingStorage: ref.watch(notificationRoutingStorageProvider),
    invalidateList: () => ref.invalidate(notificationsProvider),
    refreshCount: () =>
        ref.read(notificationNewCountProvider.notifier).refresh(),
    invalidateAccountList: (account) =>
        ref.invalidate(accountNotificationsProvider(account)),
    refreshAccountCount: (account) => ref
        .read(accountNotificationNewCountProvider(account).notifier)
        .refresh(),
    effects: effects,
    activateRecipient: activation.activate,
    eligibleAccounts: () {
      final registry = ref.read(sessionRegistryProvider).value;
      if (registry == null) return const [];
      return [
        for (final lease in onboardingStatuses.keys)
          if (onboardingStatuses[lease]?.value?.completed ?? false) lease,
      ];
    },
  );
  unawaited(runtime.start());
  ref.onDispose(() => unawaited(runtime.dispose()));
  return runtime;
}
