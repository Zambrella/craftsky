import 'package:craftsky_app/notifications/services/notification_service.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'notification_service_provider.g.dart';

@Riverpod(keepAlive: true)
NotificationService notificationService(Ref ref) =>
    const UnavailableNotificationService();
