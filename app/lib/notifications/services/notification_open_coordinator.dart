// Public callback parameter names intentionally initialize private fields.
// ignore_for_file: prefer_initializing_formals

import 'dart:async';

import 'package:craftsky_app/notifications/models/account_subscription_id.dart';
import 'package:craftsky_app/notifications/models/notification_destination.dart';
import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/notifications/services/notification_destination_inference.dart';

typedef NotificationBindingLoader =
    Future<AccountSubscriptionId?> Function(String did);
typedef NotificationOutcomeCallback =
    FutureOr<void> Function(NotificationOpenOutcome outcome);
typedef NotificationUnavailableCallback = FutureOr<void> Function();

final class NotificationOpenCoordinator {
  const NotificationOpenCoordinator({
    required this.currentDid,
    required NotificationBindingLoader loadBinding,
    required NotificationOutcomeCallback onOutcome,
    required NotificationUnavailableCallback onUnavailable,
  }) : _loadBinding = loadBinding,
       _onOutcome = onOutcome,
       _onUnavailable = onUnavailable;

  final String currentDid;
  final NotificationBindingLoader _loadBinding;
  final NotificationOutcomeCallback _onOutcome;
  final NotificationUnavailableCallback _onUnavailable;

  Future<void> open(NotificationOpenAttempt attempt) async {
    final payloadBinding = attempt.accountSubscriptionId;
    final storedBinding = await _loadBinding(currentDid);
    if (payloadBinding == null ||
        storedBinding == null ||
        storedBinding != payloadBinding) {
      await _onUnavailable();
      return;
    }

    await _onOutcome(NotificationDestinationInference.forFacts(attempt.facts));
  }
}
