import 'package:craftsky_app/notifications/models/notification_permission.dart';
import 'package:craftsky_app/notifications/services/notification_registration_coordinator.dart';
import 'package:craftsky_app/notifications/services/notification_service.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';

final class NotificationCoordinator {
  NotificationCoordinator({
    required this._service,
    required this._registration,
  });

  final NotificationService _service;
  final NotificationRegistrationCoordinator _registration;
  Did? _did;
  bool _onboarded = false;

  Future<void> updateReadiness({
    required Did? did,
    required bool onboarded,
  }) async {
    _did = did;
    _onboarded = onboarded;
    if (did == null || !onboarded) {
      await _registration.onReadinessChanged(did: did, eligible: false);
      return;
    }

    try {
      var permission = await _service.getPermission();
      final action = NotificationPermissionPolicy.actionFor(
        signedIn: true,
        onboarded: true,
        permission: permission,
      );
      if (action == NotificationPermissionAction.request) {
        permission = await _service.requestPermission();
      }
      await _registration.onReadinessChanged(
        did: did,
        eligible: permission == NotificationPermission.authorized,
      );
    } on Object {
      await _registration.onReadinessChanged(did: did, eligible: false);
    }
  }

  Future<void> retryRegistration() async {
    final did = _did;
    if (did == null || !_onboarded) return;
    try {
      final permission = await _service.getPermission();
      await _registration.onReadinessChanged(
        did: did,
        eligible: permission == NotificationPermission.authorized,
      );
    } on Object {
      await _registration.onReadinessChanged(did: did, eligible: false);
    }
  }
}
