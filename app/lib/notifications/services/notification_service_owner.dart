import 'dart:async';

import 'package:craftsky_app/notifications/models/foreground_notification_event.dart';
import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/notifications/services/notification_service.dart';

typedef NotificationTokenCallback = FutureOr<void> Function(String token);
typedef ForegroundEventCallback =
    FutureOr<void> Function(ForegroundNotificationEvent event);
typedef NotificationOpenCallback =
    FutureOr<void> Function(NotificationOpenEvent event);

final class NotificationServiceOwner {
  NotificationServiceOwner({
    required this._service,
    required this._onTokenRefresh,
    required this._onForegroundEvent,
    required this._onOpen,
  });

  final NotificationService _service;
  final NotificationTokenCallback _onTokenRefresh;
  final ForegroundEventCallback _onForegroundEvent;
  final NotificationOpenCallback _onOpen;

  Future<void>? _startFuture;
  final _subscriptions = <StreamSubscription<Object?>>[];
  bool _disposed = false;

  Future<void> start() => _startFuture ??= _start();

  Future<void> _start() async {
    if (_disposed) return;
    await _service.initialize();
    if (_disposed) return;

    _subscriptions
      ..add(
        _service.tokenRefreshes.listen(
          (token) => unawaited(Future.sync(() => _onTokenRefresh(token))),
        ),
      )
      ..add(
        _service.foregroundEvents.listen(
          (event) => unawaited(Future.sync(() => _onForegroundEvent(event))),
        ),
      )
      ..add(
        _service.openedNotifications.listen(
          (event) => unawaited(Future.sync(() => _onOpen(event))),
        ),
      );

    final initialOpen = await _service.takeInitialOpen();
    if (!_disposed && initialOpen != null) await _onOpen(initialOpen);
  }

  Future<void> dispose() async {
    if (_disposed) return;
    _disposed = true;
    for (final subscription in _subscriptions) {
      await subscription.cancel();
    }
    _subscriptions.clear();
    await _service.dispose();
  }
}
