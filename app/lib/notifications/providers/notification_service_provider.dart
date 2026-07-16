import 'package:craftsky_app/notifications/services/notification_service.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

final notificationServiceProvider = Provider<NotificationService>(
  (ref) => const UnavailableNotificationService(),
);
