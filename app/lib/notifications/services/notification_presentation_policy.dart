final class NotificationPresentationOptions {
  const NotificationPresentationOptions({
    required this.alert,
    required this.sound,
    required this.badge,
    this.vibration = false,
    this.localNotification = false,
  });

  final bool alert;
  final bool sound;
  final bool badge;
  final bool vibration;
  final bool localNotification;
}

abstract final class NotificationPresentationPolicy {
  static const permissionRequest = NotificationPresentationOptions(
    alert: true,
    sound: true,
    badge: false,
  );

  static const foreground = NotificationPresentationOptions(
    alert: false,
    sound: false,
    badge: false,
  );
}
