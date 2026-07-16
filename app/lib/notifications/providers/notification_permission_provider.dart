import 'package:craftsky_app/notifications/models/notification_permission.dart';
import 'package:craftsky_app/notifications/providers/notification_service_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'notification_permission_provider.g.dart';

@Riverpod(keepAlive: true)
Future<NotificationPermission> notificationPermission(Ref ref) =>
    ref.watch(notificationServiceProvider).getPermission();
