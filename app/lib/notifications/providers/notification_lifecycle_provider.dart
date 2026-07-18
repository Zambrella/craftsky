import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_service_provider.dart';
import 'package:craftsky_app/notifications/services/notification_routing_storage.dart';
import 'package:craftsky_app/notifications/services/notification_sign_out_cleanup.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'notification_lifecycle_provider.g.dart';

@Riverpod(keepAlive: true)
NotificationRoutingStorage notificationRoutingStorage(Ref ref) =>
    NotificationRoutingStorage(
      () => ref.read(sessionRegistryProvider).requireValue,
    );

@Riverpod(keepAlive: true)
NotificationSignOutCleanup notificationSignOutCleanup(Ref ref) =>
    NotificationSignOutCleanup(
      deleteProviderToken: ref.watch(notificationServiceProvider).deleteToken,
      // Registry removal atomically deletes the usable session and binding.
      removeRoutingBinding: (_) async {},
    );
