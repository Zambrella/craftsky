import 'dart:async';

import 'package:craftsky_app/notifications/data/notification_repository.dart';
import 'package:craftsky_app/notifications/models/foreground_notification_event.dart';
import 'package:craftsky_app/notifications/models/notification_effect.dart';
import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/notifications/services/foreground_notification_handler.dart';
import 'package:craftsky_app/notifications/services/notification_coordinator.dart';
import 'package:craftsky_app/notifications/services/notification_open_coordinator.dart';
import 'package:craftsky_app/notifications/services/notification_resolution_policy.dart';
import 'package:craftsky_app/notifications/services/notification_routing_storage.dart';
import 'package:craftsky_app/notifications/services/notification_service_owner.dart';
import 'package:craftsky_app/notifications/services/pending_notification_open.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';

final class NotificationRuntime {
  NotificationRuntime({
    required this._coordinator,
    required this._owner,
    required this._routingStorage,
    required this._resolutionRepository,
    required this._foregroundHandler,
    required this._effects,
  });

  final NotificationCoordinator _coordinator;
  final NotificationServiceOwner _owner;
  final NotificationRoutingStorage _routingStorage;
  final NotificationResolutionRepository _resolutionRepository;
  final ForegroundNotificationHandler _foregroundHandler;
  final StreamController<NotificationEffect> _effects;
  final PendingNotificationOpen _pending = PendingNotificationOpen();

  Did? _did;
  NotificationOpenReadiness _readiness =
      NotificationOpenReadiness.requiresSignIn;
  Did? _lastReadinessDid;
  bool? _lastOnboarded;

  Future<void> start() => _owner.start();

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
    await _coordinator.updateReadiness(did: did, onboarded: onboarded);
    final pending = _pending.updateReadiness(_readiness);
    if (pending != null) await _processOpen(pending);
  }

  Future<void> receiveOpen(NotificationOpenEvent event) async {
    final ready = _pending.receive(event, readiness: _readiness);
    if (ready != null) await _processOpen(ready);
  }

  Future<void> receiveForegroundEvent(
    ForegroundNotificationEvent event,
  ) async => _foregroundHandler.handle(event);

  Future<void> resume() => _coordinator.retryRegistration();

  Future<void> _processOpen(NotificationOpenEvent event) async {
    final did = _did;
    if (did == null) return;
    final opener = NotificationOpenCoordinator(
      currentDid: did.toString(),
      loadBinding: (_) => _routingStorage.read(did),
      resolve: _resolutionRepository.resolve,
      onOutcome: (outcome) =>
          _effects.add(NotificationNavigationEffect(outcome)),
      onUnavailable: () => _effects.add(const NotificationUnavailableEffect()),
    );
    try {
      await opener.open(event);
    } on Object catch (error) {
      _effects.add(
        NotificationNavigationEffect(
          NotificationResolutionPolicy.forException(error),
        ),
      );
    }
  }

  Future<void> dispose() => _owner.dispose();
}
