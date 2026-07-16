import 'package:craftsky_app/notifications/models/notification_permission.dart';
import 'package:craftsky_app/notifications/providers/notification_service_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

final notificationPermissionProvider = FutureProvider<NotificationPermission>(
  (ref) => ref.watch(notificationServiceProvider).getPermission(),
);
