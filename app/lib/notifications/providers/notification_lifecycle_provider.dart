import 'package:craftsky_app/notifications/providers/notification_service_provider.dart';
import 'package:craftsky_app/notifications/services/notification_routing_storage.dart';
import 'package:craftsky_app/notifications/services/notification_sign_out_cleanup.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

final notificationRoutingStorageProvider = Provider<NotificationRoutingStorage>(
  (ref) => const NotificationRoutingStorage(
    FlutterSecureNotificationRoutingStorageBackend(FlutterSecureStorage()),
  ),
);

final notificationSignOutCleanupProvider = Provider<NotificationSignOutCleanup>(
  (ref) => NotificationSignOutCleanup(
    deleteProviderToken: ref.watch(notificationServiceProvider).deleteToken,
    removeRoutingBinding: (did) =>
        ref.read(notificationRoutingStorageProvider).remove(Did.parse(did)),
  ),
);
