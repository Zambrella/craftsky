import 'package:craftsky_app/notifications/providers/notification_service_provider.dart';
import 'package:craftsky_app/notifications/services/notification_routing_storage.dart';
import 'package:craftsky_app/notifications/services/notification_sign_out_cleanup.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'notification_lifecycle_provider.g.dart';

@Riverpod(keepAlive: true)
NotificationRoutingStorage notificationRoutingStorage(Ref ref) =>
    const NotificationRoutingStorage(
      FlutterSecureNotificationRoutingStorageBackend(FlutterSecureStorage()),
    );

@Riverpod(keepAlive: true)
NotificationSignOutCleanup notificationSignOutCleanup(Ref ref) =>
    NotificationSignOutCleanup(
      deleteProviderToken: ref.watch(notificationServiceProvider).deleteToken,
      removeRoutingBinding: (did) =>
          ref.read(notificationRoutingStorageProvider).remove(Did.parse(did)),
    );
