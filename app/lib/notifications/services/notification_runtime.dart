// Public constructor labels stay descriptive while dependencies remain private.
// ignore_for_file: prefer_initializing_formals

import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_activation_coordinator.dart';
import 'package:craftsky_app/notifications/models/foreground_notification_event.dart';
import 'package:craftsky_app/notifications/models/notification_destination.dart';
import 'package:craftsky_app/notifications/models/notification_effect.dart';
import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/notifications/services/notification_open_coordinator.dart';
import 'package:craftsky_app/notifications/services/notification_registration_coordinator.dart';
import 'package:craftsky_app/notifications/services/notification_routing_storage.dart';
import 'package:craftsky_app/notifications/services/notification_service.dart';
import 'package:craftsky_app/notifications/services/pending_notification_open.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';

final class NotificationRuntime {
  new({
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
    ChatNotificationResolver? resolveChat,
    FutureOr<void> Function(AccountKey account)? refreshChatCount,
  }) : _service = service,
       _registration = registration,
       _routingStorage = routingStorage,
       _invalidateList = invalidateList,
       _refreshCount = refreshCount,
       _invalidateAccountList = invalidateAccountList,
       _refreshAccountCount = refreshAccountCount,
       _effects = effects,
       _activateRecipient = activateRecipient,
       _eligibleAccounts = eligibleAccounts,
       _resolveChat = resolveChat,
       _refreshChatCount = refreshChatCount;

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
  final ChatNotificationResolver? _resolveChat;
  final FutureOr<void> Function(AccountKey account)? _refreshChatCount;
  final PendingNotificationOpen _pending = PendingNotificationOpen();
  final _subscriptions = <StreamSubscription<Object?>>[];

  ({
    ActiveAccountLease activation,
    int sequence,
    NotificationNavigationEffect effect,
  })?
  _deferredNavigation;
  Future<void>? _startFuture;
  bool _disposed = false;
  Did? _did;
  // Until the host has resolved the initial account, an OS open must wait.
  NotificationOpenReadiness _readiness = NotificationOpenReadiness.transient;
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
    bool initialized = true,
  }) async {
    if (_disposed) return;
    final readiness = !initialized
        ? NotificationOpenReadiness.transient
        : did == null
        ? NotificationOpenReadiness.requiresSignIn
        : onboarded
        ? NotificationOpenReadiness.ready
        : NotificationOpenReadiness.transient;
    if (_lastReadinessDid == did &&
        _lastOnboarded == onboarded &&
        _readiness == readiness) {
      await refreshAccounts();
      return;
    }
    _lastReadinessDid = did;
    _lastOnboarded = onboarded;
    _did = did;
    _readiness = readiness;
    if (_readiness == NotificationOpenReadiness.requiresSignIn) {
      _latestOpenSequence++;
      _deferredNavigation = null;
    }
    _publishDeferredNavigation();
    final pending = _pending.updateReadiness(_readiness);
    if (pending != null) await _processOpen(pending);
    await refreshAccounts();
  }

  Future<void> receiveOpen(NotificationOpenAttempt event) async {
    if (_disposed) return;
    await receiveResolvedOpen(
      event,
      _routingStorage.resolveIfReady(event.accountSubscriptionId),
    );
  }

  Future<void> receiveResolvedOpen(
    NotificationOpenAttempt event,
    NotificationRecipientResolution? resolution,
  ) async {
    if (_disposed) return;
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
    if (_disposed) return;
    final resolution = _routingStorage.resolve(
      event.openAttempt.accountSubscriptionId,
    );
    if (resolution is! ExactNotificationRecipient) return;
    final lease = resolution.lease;
    final activation = _routingStorage.activeLease;
    bool bindingIsCurrent() =>
        !_disposed &&
        _routingStorage.read(lease) == event.openAttempt.accountSubscriptionId;
    NotificationRecipientIdentity? recipient;
    final session = _routingStorage.sessionFor(lease);
    if (session == null) return;
    if (activation?.session != lease) {
      recipient = NotificationRecipientIdentity(
        lease: lease,
        handle: session.handle.value,
        avatarUrl: session.cachedAvatarUrl,
      );
    }
    _effects.add(
      NotificationBannerEffect(
        event,
        resolution: resolution,
        recipient: recipient,
        isCurrent: () =>
            bindingIsCurrent() && _routingStorage.activeLease == activation,
      ),
    );
    if (event.openAttempt.facts is ChatNotificationFacts ||
        event.openAttempt.facts is InvalidChatNotificationFacts) {
      await _refreshChatCount?.call(lease.account);
      return;
    }
    final invalidateAccountList = _invalidateAccountList;
    if (invalidateAccountList != null) {
      await invalidateAccountList(lease.account);
    } else {
      await _invalidateList();
    }
    if (!bindingIsCurrent()) return;
    final refreshAccountCount = _refreshAccountCount;
    if (refreshAccountCount != null) {
      await refreshAccountCount(lease.account);
    } else {
      await _refreshCount();
    }
  }

  Future<void> resume() async {
    await refreshAccounts(retry: true);
  }

  Future<void> refreshAccounts({bool retry = false}) async {
    if (_disposed) return;
    await _registration.updateAccounts(
      _eligibleAccounts?.call() ?? const [],
      retry: retry,
    );
  }

  Future<void> _processOpen(PendingNotificationOpenWork work) async {
    final opener = NotificationOpenCoordinator(
      resolveRecipient: _routingStorage.resolve,
      isCurrentLease: _routingStorage.isCurrentLease,
      isActiveLease: _routingStorage.isActiveLease,
      resolveChat: _resolveChat,
      activate: (lease) async => _activateRecipient == null
          ? AccountActivationResult.alreadyActive
          : await _activateRecipient(lease),
      onOutcome: (outcome) {
        final activation = _routingStorage.activeLease;
        if (activation == null || !_isCurrentWork(work)) return;
        final effect = NotificationNavigationEffect(
          NotificationOpenOutcome(
            destination: outcome.destination,
            feedback: outcome.feedback,
            isCurrent: () =>
                _isCurrentWork(work) &&
                _routingStorage.activeLease == activation &&
                (outcome.isCurrent?.call() ?? true),
          ),
        );
        if (_isReadyFor(activation)) {
          _effects.add(effect);
        } else if (work.sequence == _latestOpenSequence) {
          // Activation can finish before the new account's UI is ready.
          // Keep this resolved destination without repeating its lookup.
          _deferredNavigation = (
            activation: activation,
            sequence: work.sequence,
            effect: effect,
          );
        }
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

  void _publishDeferredNavigation() {
    final deferred = _deferredNavigation;
    if (deferred == null || _readiness != NotificationOpenReadiness.ready) {
      return;
    }
    _deferredNavigation = null;
    if (deferred.sequence == _latestOpenSequence &&
        _isReadyFor(deferred.activation) &&
        deferred.effect.outcome.isCurrent?.call() == true) {
      _effects.add(deferred.effect);
    }
  }

  bool _isReadyFor(ActiveAccountLease activation) =>
      _readiness == NotificationOpenReadiness.ready &&
      _did == activation.session.account.did;

  bool _isCurrentOpen(PendingNotificationOpenWork work) =>
      _isCurrentWork(work) && _readiness == NotificationOpenReadiness.ready;

  bool _isCurrentWork(PendingNotificationOpenWork work) =>
      !_disposed &&
      (!work.latestOnly || _latestOpenSequence == work.sequence) &&
      _readiness != NotificationOpenReadiness.requiresSignIn;

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
