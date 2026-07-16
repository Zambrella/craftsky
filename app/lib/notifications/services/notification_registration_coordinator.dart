// Public constructor labels stay descriptive while dependencies remain private.
// ignore_for_file: prefer_initializing_formals

import 'dart:async';

import 'package:craftsky_app/notifications/models/account_subscription_id.dart';
import 'package:craftsky_app/notifications/models/notification_permission.dart';
import 'package:craftsky_app/notifications/services/notification_service.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';

enum NotificationPlatform { android, ios }

typedef NotificationDeviceRegistrar =
    Future<AccountSubscriptionId> Function({
      required NotificationPlatform platform,
      required String token,
    });
typedef NotificationBindingSaver =
    Future<void> Function({
      required Did did,
      required AccountSubscriptionId binding,
    });

final class NotificationRegistrationCoordinator {
  NotificationRegistrationCoordinator({
    required NotificationService service,
    required this.platform,
    required NotificationDeviceRegistrar register,
    required NotificationBindingSaver saveBinding,
  }) : _service = service,
       _register = register,
       _saveBinding = saveBinding;

  final NotificationService _service;
  final NotificationPlatform platform;
  final NotificationDeviceRegistrar _register;
  final NotificationBindingSaver _saveBinding;

  Did? _did;
  bool _onboarded = false;
  bool _eligible = false;
  String? _latestToken;
  Future<void>? _inFlight;
  int _registrationRevision = 0;

  Future<void> updateReadiness({
    required Did? did,
    required bool onboarded,
  }) async {
    _setReadiness(did: did, onboarded: onboarded);
    if (did == null || !onboarded) {
      _setEligible(false);
      return;
    }

    try {
      var permission = await _service.getPermission();
      if (!_isCurrent(did)) return;
      if (permission == NotificationPermission.notDetermined) {
        permission = await _service.requestPermission();
      }
      if (!_isCurrent(did)) return;
      await _setEligibility(
        did: did,
        eligible: permission == NotificationPermission.authorized,
      );
    } on Object {
      if (_did == did) _setEligible(false);
    }
  }

  Future<void> retryRegistration() async {
    final did = _did;
    if (did == null || !_onboarded) return;
    try {
      final permission = await _service.getPermission();
      if (!_isCurrent(did)) return;
      await _setEligibility(
        did: did,
        eligible: permission == NotificationPermission.authorized,
      );
    } on Object {
      if (_did == did) _setEligible(false);
    }
  }

  Future<void> onTokenRefresh(String token) async {
    if (token.isEmpty) return;
    _setLatestToken(token);
    await _attemptRegistration();
  }

  bool _isCurrent(Did did) => _did == did && _onboarded;

  void _setReadiness({required Did? did, required bool onboarded}) {
    if (_did == did && _onboarded == onboarded) return;
    _did = did;
    _onboarded = onboarded;
    _registrationRevision++;
  }

  void _setEligible(bool value) {
    if (_eligible == value) return;
    _eligible = value;
    _registrationRevision++;
  }

  void _setLatestToken(String token) {
    if (_latestToken == token) return;
    _latestToken = token;
    _registrationRevision++;
  }

  Future<void> _setEligibility({
    required Did did,
    required bool eligible,
  }) async {
    if (!_isCurrent(did)) return;
    _setEligible(eligible);
    if (!eligible) return;
    await _refreshToken();
    await _attemptRegistration();
  }

  Future<void> _refreshToken() async {
    try {
      final token = await _service.getToken();
      if (token != null && token.isNotEmpty) _setLatestToken(token);
    } on Object {
      // Registration is opportunistic. A later eligible trigger retries.
    }
  }

  Future<void> _attemptRegistration() async {
    final existing = _inFlight;
    if (existing != null) return existing;
    if (!_eligible || _did == null || _latestToken == null) return;

    final attempt = _drainRegistrationChanges();
    _inFlight = attempt;
    try {
      await attempt;
    } finally {
      _inFlight = null;
    }
  }

  Future<void> _drainRegistrationChanges() async {
    while (_eligible) {
      final did = _did;
      final token = _latestToken;
      if (did == null || token == null || token.isEmpty) return;
      final revision = _registrationRevision;
      await _registerAndSave(did: did, token: token);
      if (_registrationRevision == revision) return;
    }
  }

  Future<void> _registerAndSave({
    required Did did,
    required String token,
  }) async {
    try {
      final binding = await _register(platform: platform, token: token);
      if (!_eligible || _did != did) return;
      await _saveBinding(did: did, binding: binding);
    } on Object {
      // Keep the latest token in memory for a later readiness/resume trigger.
    }
  }
}
