import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/notifications/services/notification_routing_storage.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'notification_lifecycle_provider.g.dart';

@Riverpod(keepAlive: true)
NotificationRoutingStorage notificationRoutingStorage(Ref ref) =>
    NotificationRoutingStorage(
      () => ref.read(sessionRegistryProvider).requireValue,
    );
