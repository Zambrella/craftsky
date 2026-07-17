// Public constructor labels stay descriptive while dependencies remain private.
// ignore_for_file: prefer_initializing_formals

import 'dart:async';

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
  }) : _service = service,
       _registration = registration,
       _routingStorage = routingStorage,
       _invalidateList = invalidateList,
       _refreshCount = refreshCount,
       _effects = effects;

  final NotificationService _service;
  final NotificationRegistrationCoordinator _registration;
  final NotificationRoutingStorage _routingStorage;
  final FutureOr<void> Function() _invalidateList;
  final FutureOr<void> Function() _refreshCount;
  final StreamController<NotificationEffect> _effects;
  final PendingNotificationOpen _pending = PendingNotificationOpen();
  final _subscriptions = <StreamSubscription<Object?>>[];

  Future<void>? _startFuture;
  bool _disposed = false;
  Did? _did;
  NotificationOpenReadiness _readiness =
      NotificationOpenReadiness.requiresSignIn;
  int _readinessRevision = 0;
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
    if (_lastReadinessDid == did && _lastOnboarded == onboarded) return;
    _lastReadinessDid = did;
    _lastOnboarded = onboarded;
    _did = did;
    _readiness = did == null
        ? NotificationOpenReadiness.requiresSignIn
        : onboarded
        ? NotificationOpenReadiness.ready
        : NotificationOpenReadiness.transient;
    _readinessRevision++;
    await _registration.updateReadiness(did: did, onboarded: onboarded);
    final pending = _pending.updateReadiness(_readiness);
    if (pending != null) await _processOpen(pending);
  }

  Future<void> receiveOpen(NotificationOpenAttempt event) async {
    final ready = _pending.receive(event, readiness: _readiness);
    if (ready != null) await _processOpen(ready);
  }

  Future<void> receiveForegroundEvent(
    ForegroundNotificationEvent event,
  ) async {
    _effects.add(NotificationBannerEffect(event));
    await _invalidateList();
    await _refreshCount();
  }

  Future<void> resume() => _registration.retryRegistration();

  Future<void> _processOpen(NotificationOpenAttempt event) async {
    final did = _did;
    if (did == null) return;
    final readinessRevision = _readinessRevision;
    final opener = NotificationOpenCoordinator(
      currentDid: did.toString(),
      loadBinding: (_) => _routingStorage.read(did),
      onOutcome: (outcome) {
        if (!_isCurrentOpen(did, readinessRevision)) return;
        _effects.add(NotificationNavigationEffect(outcome));
      },
      onUnavailable: () {
        if (!_isCurrentOpen(did, readinessRevision)) return;
        _effects.add(const NotificationUnavailableEffect());
      },
    );
    await opener.open(event);
  }

  bool _isCurrentOpen(Did did, int readinessRevision) =>
      !_disposed &&
      _readinessRevision == readinessRevision &&
      _readiness == NotificationOpenReadiness.ready &&
      _did == did;

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
