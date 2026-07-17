import 'package:craftsky_app/notifications/models/foreground_notification_event.dart';
import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/notifications/models/notification_permission.dart';

abstract interface class NotificationService {
  Future<void> initialize();
  Future<void> dispose();
  Future<NotificationPermission> getPermission();
  Future<NotificationPermission> requestPermission();
  Future<String?> getToken();
  Stream<String> get tokenRefreshes;
  Stream<ForegroundNotificationEvent> get foregroundEvents;
  Stream<NotificationOpenAttempt> get openedNotifications;
  Future<NotificationOpenAttempt?> takeInitialOpen();
  Future<void> deleteToken();
  Future<void> openSystemNotificationSettings();
}

final class UnavailableNotificationService implements NotificationService {
  const UnavailableNotificationService();

  @override
  Future<void> deleteToken() async {}

  @override
  Future<void> dispose() async {}

  @override
  Stream<ForegroundNotificationEvent> get foregroundEvents =>
      const Stream.empty();

  @override
  Future<NotificationPermission> getPermission() async =>
      NotificationPermission.denied;

  @override
  Future<String?> getToken() async => null;

  @override
  Future<void> initialize() async {}

  @override
  Stream<NotificationOpenAttempt> get openedNotifications =>
      const Stream.empty();

  @override
  Future<void> openSystemNotificationSettings() async {}

  @override
  Future<NotificationPermission> requestPermission() async =>
      NotificationPermission.denied;

  @override
  Future<NotificationOpenAttempt?> takeInitialOpen() async => null;

  @override
  Stream<String> get tokenRefreshes => const Stream.empty();
}
