// Public constructor labels stay descriptive while dependencies remain private.
// ignore_for_file: prefer_initializing_formals

import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_activation_coordinator.dart';
import 'package:craftsky_app/notifications/models/foreground_notification_event.dart';
import 'package:craftsky_app/notifications/models/notification_effect.dart';
import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/notifications/services/notification_open_coordinator.dart';
import 'package:craftsky_app/notifications/services/notification_registration_coordinator.dart';
import 'package:craftsky_app/notifications/services/notification_routing_storage.dart';
import 'package:craftsky_app/notifications/services/notification_service.dart';
import 'package:craftsky_app/notifications/services/pending_notification_open.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';

final class NotificationRuntime {
  NotificationRuntime({
    required NotificationService service,
    required NotificationRegistrationCoordinator registration,
    required NotificationRoutingStorage routingStorage,
    required FutureOr<void> Function() invalidateList,
    required FutureOr<void> Function() refreshCount,
    required StreamController<NotificationEffect> effects,
    FutureOr<void> Function(AccountKey account)? invalidateAccountList,
    FutureOr<void> Function(AccountKey account)? refreshAccountCount,
    Future<AccountActivationResult> Function(AccountSessionLease lease)?
    activateRecipient,
    List<AccountSessionLease> Function()? eligibleAccounts,
  }) : _service = service,
       _registration = registration,
       _routingStorage = routingStorage,
       _invalidateList = invalidateList,
       _refreshCount = refreshCount,
       _invalidateAccountList = invalidateAccountList,
       _refreshAccountCount = refreshAccountCount,
       _effects = effects,
       _activateRecipient = activateRecipient,
       _eligibleAccounts = eligibleAccounts;

  final NotificationService _service;
  final NotificationRegistrationCoordinator _registration;
  final NotificationRoutingStorage _routingStorage;
  final FutureOr<void> Function() _invalidateList;
  final FutureOr<void> Function() _refreshCount;
  final FutureOr<void> Function(AccountKey account)? _invalidateAccountList;
  final FutureOr<void> Function(AccountKey account)? _refreshAccountCount;
  final StreamController<NotificationEffect> _effects;
  final Future<AccountActivationResult> Function(AccountSessionLease lease)?
  _activateRecipient;
  final List<AccountSessionLease> Function()? _eligibleAccounts;
  final PendingNotificationOpen _pending = PendingNotificationOpen();
  final _subscriptions = <StreamSubscription<Object?>>[];

  Future<void>? _startFuture;
  bool _disposed = false;
  Did? _did;
  NotificationOpenReadiness _readiness =
      NotificationOpenReadiness.requiresSignIn;
  int _latestOpenSequence = 0;
  Did? _lastReadinessDid;
  bool? _lastOnboarded;

  Future<void> start() => _startFuture ??= _start();

  Future<void> _start() async {
    if (_disposed) return;
    await _service.initialize();
    if (_disposed) return;

    _subscriptions
      ..add(
        _service.tokenRefreshes.listen(
          (token) => unawaited(_registration.onTokenRefresh(token)),
        ),
      )
      ..add(
        _service.foregroundEvents.listen(
          (event) => unawaited(receiveForegroundEvent(event)),
        ),
      )
      ..add(
        _service.openedNotifications.listen(
          (event) => unawaited(receiveOpen(event)),
        ),
      );

    final initialOpen = await _service.takeInitialOpen();
    if (!_disposed && initialOpen != null) await receiveOpen(initialOpen);
  }

  Future<void> updateReadiness({
    required Did? did,
    required bool onboarded,
  }) async {
    await _registration.updateAccounts(_eligibleAccounts?.call() ?? const []);
    if (_lastReadinessDid == did && _lastOnboarded == onboarded) return;
    _lastReadinessDid = did;
    _lastOnboarded = onboarded;
    _did = did;
    _readiness = did == null
        ? NotificationOpenReadiness.requiresSignIn
        : onboarded
        ? NotificationOpenReadiness.ready
        : NotificationOpenReadiness.transient;
    if (_readiness == NotificationOpenReadiness.requiresSignIn) {
      _latestOpenSequence++;
    }
    final pending = _pending.updateReadiness(_readiness);
    if (pending != null) await _processOpen(pending);
  }

  Future<void> receiveOpen(NotificationOpenAttempt event) async {
    await receiveResolvedOpen(
      event,
      _routingStorage.resolve(event.accountSubscriptionId),
    );
  }

  Future<void> receiveResolvedOpen(
    NotificationOpenAttempt event,
    NotificationRecipientResolution resolution,
  ) async {
    final requiresActivation = switch (resolution) {
      ExactNotificationRecipient(:final lease) =>
        !_routingStorage.isActiveLease(lease),
      _ => false,
    };
    final work = PendingNotificationOpenWork(
      attempt: event,
      resolution: resolution,
      sequence: ++_latestOpenSequence,
      latestOnly:
          _readiness != NotificationOpenReadiness.ready || requiresActivation,
    );
    final ready = _pending.receive(work, readiness: _readiness);
    if (ready != null) await _processOpen(ready);
  }

  Future<void> receiveForegroundEvent(
    ForegroundNotificationEvent event,
  ) async {
    final resolution = _routingStorage.resolve(
      event.openAttempt.accountSubscriptionId,
    );
    NotificationRecipientIdentity? recipient;
    AccountKey? recipientAccount;
    if (resolution case ExactNotificationRecipient(:final lease)) {
      recipientAccount = lease.account;
      final session = _routingStorage.sessionFor(lease);
      if (session != null && !_routingStorage.isActiveLease(lease)) {
        recipient = NotificationRecipientIdentity(
          lease: lease,
          handle: session.handle.value,
          avatarUrl: session.cachedAvatarUrl,
        );
      }
    }
    _effects.add(
      NotificationBannerEffect(
        event,
        resolution: resolution,
        recipient: recipient,
      ),
    );
    if (recipientAccount case final account?) {
      await (_invalidateAccountList?.call(account) ?? _invalidateList());
      await (_refreshAccountCount?.call(account) ?? _refreshCount());
    } else {
      await _invalidateList();
      await _refreshCount();
    }
  }

  Future<void> resume() async {
    await _registration.updateAccounts(_eligibleAccounts?.call() ?? const []);
    await _registration.retryRegistration();
  }

  Future<void> _processOpen(PendingNotificationOpenWork work) async {
    final opener = NotificationOpenCoordinator(
      resolveRecipient: _routingStorage.resolve,
      isCurrentLease: _routingStorage.isCurrentLease,
      activate: (lease) async => _activateRecipient == null
          ? AccountActivationResult.alreadyActive
          : _activateRecipient(lease),
      onOutcome: (outcome) {
        if (!_isCurrentOpen(work)) return;
        _effects.add(NotificationNavigationEffect(outcome));
      },
      onUnavailable: () {
        if (!_isCurrentOpen(work)) return;
        _effects.add(const NotificationUnavailableEffect());
      },
      onRemovedAccount: () {
        if (!_isCurrentOpen(work)) return;
        _effects.add(const NotificationRemovedAccountEffect());
      },
    );
    await opener.openResolved(work);
  }

  bool _isCurrentOpen(PendingNotificationOpenWork work) =>
      !_disposed &&
      (!work.latestOnly || _latestOpenSequence == work.sequence) &&
      _readiness == NotificationOpenReadiness.ready &&
      _did != null;

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
