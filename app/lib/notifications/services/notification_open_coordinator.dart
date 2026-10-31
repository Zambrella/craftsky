// Public callback parameter names intentionally initialize private fields.
// ignore_for_file: prefer_initializing_formals

import 'dart:async';

import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_activation_coordinator.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/notifications/models/account_subscription_id.dart';
import 'package:craftsky_app/notifications/models/notification_destination.dart';
import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/notifications/services/notification_destination_inference.dart';
import 'package:craftsky_app/notifications/services/notification_routing_storage.dart';
import 'package:craftsky_app/notifications/services/pending_notification_open.dart';

typedef ChatNotificationResolver = Future<ChatConversationId?> Function(
  AccountSessionLease lease,
  ChatOpenReference reference,
);

typedef NotificationRecipientResolver =
    NotificationRecipientResolution Function(AccountSubscriptionId? binding);
typedef NotificationLeaseGuard = bool Function(AccountSessionLease lease);
typedef NotificationAccountActivator = Future<AccountActivationResult> Function(
  AccountSessionLease lease,
);
typedef NotificationOutcomeCallback = FutureOr<void> Function(
  NotificationOpenOutcome outcome,
);
typedef NotificationUnavailableCallback = FutureOr<void> Function();

final class NotificationOpenCoordinator {
  const new({
    required NotificationRecipientResolver resolveRecipient,
    required NotificationLeaseGuard isCurrentLease,
    required NotificationAccountActivator activate,
    required NotificationOutcomeCallback onOutcome,
    required NotificationUnavailableCallback onUnavailable,
    required NotificationUnavailableCallback onRemovedAccount,
    NotificationLeaseGuard? isActiveLease,
    ChatNotificationResolver? resolveChat,
  }) : _resolveRecipient = resolveRecipient,
       _isCurrentLease = isCurrentLease,
       _activate = activate,
       _onOutcome = onOutcome,
       _onUnavailable = onUnavailable,
       _onRemovedAccount = onRemovedAccount,
       _isActiveLease = isActiveLease,
       _resolveChat = resolveChat;

  final NotificationRecipientResolver _resolveRecipient;
  final NotificationLeaseGuard _isCurrentLease;
  final NotificationAccountActivator _activate;
  final NotificationOutcomeCallback _onOutcome;
  final NotificationUnavailableCallback _onUnavailable;
  final NotificationUnavailableCallback _onRemovedAccount;
  final NotificationLeaseGuard? _isActiveLease;
  final ChatNotificationResolver? _resolveChat;

  Future<void> open(NotificationOpenAttempt attempt) async {
    await openResolved(
      PendingNotificationOpenWork(
        attempt: attempt,
        resolution: _resolveRecipient(attempt.accountSubscriptionId),
      ),
    );
  }

  Future<void> openResolved(PendingNotificationOpenWork work) async {
    final attempt = work.attempt;
    final resolution = work.resolution;
    switch (resolution) {
      case InvalidNotificationRecipient():
        await _onUnavailable();
        return;
      case RemovedNotificationRecipient():
        await _onRemovedAccount();
        return;
      case ExactNotificationRecipient(:final lease):
        if (!_isCurrentLease(lease)) {
          await _onRemovedAccount();
          return;
        }
        final activation = await _activate(lease);
        if (activation == AccountActivationResult.cancelled) return;
        if (activation == AccountActivationResult.stale ||
            !_isCurrentLease(lease)) {
          await _onRemovedAccount();
          return;
        }
        if (attempt.facts case ChatNotificationFacts(:final openReference)) {
          final resolve = _resolveChat;
          if (resolve == null || _isActiveLease?.call(lease) != true) {
            await _onUnavailable();
            return;
          }
          final id = await resolve(lease, openReference);
          if (!_isCurrentLease(lease) || _isActiveLease?.call(lease) != true) {
            return;
          }
          if (id == null) {
            await _onUnavailable();
            return;
          }
          await _onOutcome(
            NotificationOpenOutcome(
              destination: ChatConversationDestination(id),
              isCurrent: () =>
                  _isCurrentLease(lease) && _isActiveLease?.call(lease) == true,
            ),
          );
          return;
        }
        await _onOutcome(
          NotificationDestinationInference.forFacts(attempt.facts),
        );
    }
  }
}
