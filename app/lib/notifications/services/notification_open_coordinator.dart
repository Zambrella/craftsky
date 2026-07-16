import 'dart:async';

import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/notifications/models/notification_resolution.dart';
import 'package:craftsky_app/notifications/services/notification_resolution_policy.dart';
import 'package:craftsky_app/notifications/services/notification_routing_policy.dart';

typedef NotificationBindingLoader =
    Future<AccountSubscriptionId?> Function(String did);
typedef NotificationResolver =
    Future<NotificationResolution> Function(NotificationId id);
typedef NotificationOutcomeCallback =
    FutureOr<void> Function(NotificationResolutionOutcome outcome);
typedef NotificationUnavailableCallback = FutureOr<void> Function();

final class NotificationOpenCoordinator {
  const NotificationOpenCoordinator({
    required this.currentDid,
    required this._loadBinding,
    required this._resolve,
    required this._onOutcome,
    required this._onUnavailable,
  });

  final String currentDid;
  final NotificationBindingLoader _loadBinding;
  final NotificationResolver _resolve;
  final NotificationOutcomeCallback _onOutcome;
  final NotificationUnavailableCallback _onUnavailable;

  Future<void> open(NotificationOpenEvent event) async {
    final storedBinding = await _loadBinding(currentDid);
    if (!NotificationRoutingPolicy.canResolve(
      storedBinding: storedBinding,
      eventBinding: event.accountSubscriptionId,
    )) {
      await _onUnavailable();
      return;
    }

    final resolution = await _resolve(event.notificationId);
    await _onOutcome(NotificationResolutionPolicy.forResolution(resolution));
  }
}
