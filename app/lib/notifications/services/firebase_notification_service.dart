// The explicit values below document the notification presentation contract.
// ignore_for_file: avoid_redundant_argument_values

import 'dart:async';

import 'package:app_settings/app_settings.dart';
import 'package:craftsky_app/notifications/models/foreground_notification_event.dart';
import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/notifications/models/notification_permission.dart';
import 'package:craftsky_app/notifications/services/notification_delivery_envelope.dart';
import 'package:craftsky_app/notifications/services/notification_service.dart';
import 'package:firebase_messaging/firebase_messaging.dart';

final class FirebaseNotificationService implements NotificationService {
  FirebaseNotificationService(this._messaging);

  final FirebaseMessaging _messaging;

  @override
  Future<void> initialize() async {
    await _messaging.setForegroundNotificationPresentationOptions(
      alert: false,
      badge: false,
      sound: false,
    );
  }

  @override
  Future<void> dispose() async {}

  @override
  Future<NotificationPermission> getPermission() async => _mapPermission(
    (await _messaging.getNotificationSettings()).authorizationStatus,
  );

  @override
  Future<NotificationPermission> requestPermission() async {
    final settings = await _messaging.requestPermission(
      alert: true,
      badge: false,
      sound: true,
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
      .map(foregroundEventFromMessage)
      .where((event) => event != null)
      .cast<ForegroundNotificationEvent>();

  @override
  Stream<NotificationOpenAttempt> get openedNotifications => FirebaseMessaging
      .onMessageOpenedApp
      .map(
        (message) => _providerOpenFromMessage(
          message,
          NotificationOpenSource.backgroundOpen,
        ),
      )
      .where((attempt) => attempt != null)
      .cast<NotificationOpenAttempt>();

  @override
  Future<NotificationOpenAttempt?> takeInitialOpen() async {
    final message = await _messaging.getInitialMessage();
    if (message == null) return null;
    return _providerOpenFromMessage(
      message,
      NotificationOpenSource.initialOpen,
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

  static ForegroundNotificationEvent? foregroundEventFromMessage(
    RemoteMessage message,
  ) {
    final envelope = _envelopeFromMessage(
      message,
      source: NotificationOpenSource.foregroundBanner,
    );
    if (envelope == null) return null;
    return ForegroundNotificationEvent(
      title: envelope.title,
      body: envelope.body,
      openAttempt: envelope.openAttempt,
    );
  }

  static NotificationOpenAttempt? _providerOpenFromMessage(
    RemoteMessage message,
    NotificationOpenSource source,
  ) {
    final envelope = _envelopeFromMessage(message, source: source);
    return envelope?.openAttempt;
  }

  static NotificationDeliveryEnvelope? _envelopeFromMessage(
    RemoteMessage message, {
    required NotificationOpenSource source,
  }) {
    final data = <String, Object?>{...message.data};
    final notification = message.notification;
    if (notification != null) {
      data['displayTitle'] ??= notification.title;
      data['displayBody'] ??= notification.body;
    }
    return NotificationDeliveryEnvelope.tryParse(data, source: source);
  }
}
