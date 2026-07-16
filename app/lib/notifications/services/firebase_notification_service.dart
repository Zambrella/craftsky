import 'package:app_settings/app_settings.dart';
import 'package:craftsky_app/notifications/models/foreground_notification_event.dart';
import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/notifications/models/notification_permission.dart';
import 'package:craftsky_app/notifications/services/notification_presentation_policy.dart';
import 'package:craftsky_app/notifications/services/notification_service.dart';
import 'package:firebase_messaging/firebase_messaging.dart';

final class FirebaseNotificationService implements NotificationService {
  const FirebaseNotificationService(this._messaging);

  final FirebaseMessaging _messaging;

  @override
  Future<void> initialize() =>
      _messaging.setForegroundNotificationPresentationOptions(
        alert: NotificationPresentationPolicy.foreground.alert,
        badge: NotificationPresentationPolicy.foreground.badge,
        sound: NotificationPresentationPolicy.foreground.sound,
      );

  @override
  Future<void> dispose() async {}

  @override
  Future<NotificationPermission> getPermission() async => _mapPermission(
    (await _messaging.getNotificationSettings()).authorizationStatus,
  );

  @override
  Future<NotificationPermission> requestPermission() async {
    final settings = await _messaging.requestPermission(
      alert: NotificationPresentationPolicy.permissionRequest.alert,
      badge: NotificationPresentationPolicy.permissionRequest.badge,
      sound: NotificationPresentationPolicy.permissionRequest.sound,
    );
    return _mapPermission(settings.authorizationStatus);
  }

  @override
  Future<String?> getToken() => _messaging.getToken();

  @override
  Stream<String> get tokenRefreshes => _messaging.onTokenRefresh;

  @override
  Stream<ForegroundNotificationEvent> get foregroundEvents => FirebaseMessaging
      .onMessage
      .map(_foregroundEventFromMessage)
      .where((event) => event != null)
      .cast<ForegroundNotificationEvent>();

  @override
  Stream<NotificationOpenEvent> get openedNotifications => FirebaseMessaging
      .onMessageOpenedApp
      .map(
        (message) => NotificationOpenEvent.tryParseProviderData(
          message.data,
        ),
      )
      .where((event) => event != null)
      .cast<NotificationOpenEvent>();

  @override
  Future<NotificationOpenEvent?> takeInitialOpen() async {
    final message = await _messaging.getInitialMessage();
    if (message == null) return null;
    return NotificationOpenEvent.tryParseProviderData(
      message.data,
      source: NotificationOpenSource.initialOpen,
    );
  }

  @override
  Future<void> deleteToken() => _messaging.deleteToken();

  @override
  Future<void> openSystemNotificationSettings() => AppSettings.openAppSettings(
    type: AppSettingsType.notification,
  );

  static NotificationPermission _mapPermission(AuthorizationStatus status) =>
      switch (status) {
        AuthorizationStatus.authorized ||
        AuthorizationStatus.provisional => NotificationPermission.authorized,
        AuthorizationStatus.denied => NotificationPermission.denied,
        AuthorizationStatus.notDetermined =>
          NotificationPermission.notDetermined,
      };

  static ForegroundNotificationEvent? _foregroundEventFromMessage(
    RemoteMessage message,
  ) {
    final notification = message.notification;
    final openEvent = NotificationOpenEvent.tryParseProviderData(
      message.data,
      source: NotificationOpenSource.foregroundBanner,
    );
    if (notification == null || openEvent == null) return null;
    return ForegroundNotificationEvent(
      title: notification.title ?? '',
      body: notification.body ?? '',
      openEvent: openEvent,
    );
  }
}
