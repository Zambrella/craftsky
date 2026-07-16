import 'dart:async';

import 'package:craftsky_app/notifications/models/foreground_notification_event.dart';

typedef ForegroundBannerCallback =
    FutureOr<void> Function(ForegroundNotificationEvent event);
typedef NotificationRefreshCallback = FutureOr<void> Function();

final class ForegroundNotificationHandler {
  const ForegroundNotificationHandler({
    required this._showBanner,
    required this._invalidateList,
    required this._refreshCount,
  });

  final ForegroundBannerCallback _showBanner;
  final NotificationRefreshCallback _invalidateList;
  final NotificationRefreshCallback _refreshCount;

  Future<void> handle(ForegroundNotificationEvent event) async {
    await _showBanner(event);
    await _invalidateList();
    await _refreshCount();
  }
}
