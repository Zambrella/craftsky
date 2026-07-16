import 'dart:async';

import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';

enum NotificationPlatform { android, ios }

typedef NotificationTokenReader = Future<String?> Function();
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
    required this.platform,
    required this._getToken,
    required this._register,
    required this._saveBinding,
  });

  final NotificationPlatform platform;
  final NotificationTokenReader _getToken;
  final NotificationDeviceRegistrar _register;
  final NotificationBindingSaver _saveBinding;

  Did? _did;
  bool _eligible = false;
  String? _latestToken;
  Future<void>? _inFlight;

  Future<void> onReadinessChanged({
    required Did? did,
    required bool eligible,
  }) async {
    _did = did;
    _eligible = eligible;
    if (!eligible || did == null) return;
    await _refreshToken();
    await _attemptRegistration();
  }

  Future<void> onTokenRefresh(String token) async {
    if (token.isEmpty) return;
    _latestToken = token;
    await _attemptRegistration();
  }

  Future<void> retry() async {
    if (!_eligible || _did == null) return;
    await _refreshToken();
    await _attemptRegistration();
  }

  Future<void> _refreshToken() async {
    try {
      final token = await _getToken();
      if (token != null && token.isNotEmpty) _latestToken = token;
    } on Object {
      // Registration is opportunistic. A later eligible trigger retries.
    }
  }

  Future<void> _attemptRegistration() async {
    final existing = _inFlight;
    if (existing != null) return existing;
    final did = _did;
    final token = _latestToken;
    if (!_eligible || did == null || token == null || token.isEmpty) return;

    final attempt = _registerAndSave(did: did, token: token);
    _inFlight = attempt;
    try {
      await attempt;
    } finally {
      _inFlight = null;
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
