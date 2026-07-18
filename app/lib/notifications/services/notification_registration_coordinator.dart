// Public constructor labels stay descriptive while dependencies remain private.
// ignore_for_file: prefer_initializing_formals

import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/notifications/models/account_subscription_id.dart';
import 'package:craftsky_app/notifications/models/notification_permission.dart';
import 'package:craftsky_app/notifications/services/notification_service.dart';

enum NotificationPlatform { android, ios }

typedef NotificationAccountRegistrar =
    Future<AccountSubscriptionId> Function({
      required AccountSessionLease lease,
      required NotificationPlatform platform,
      required String token,
    });
typedef NotificationLeaseBindingSaver =
    Future<void> Function({
      required AccountSessionLease lease,
      required AccountSubscriptionId binding,
    });

/// Coordinates installation-scoped permission/token state with independent
/// account-scoped registration work.
final class NotificationRegistrationCoordinator {
  NotificationRegistrationCoordinator({
    required NotificationService service,
    required this.platform,
    required NotificationAccountRegistrar registerAccount,
    required NotificationLeaseBindingSaver saveBindingForLease,
  }) : _service = service,
       _registerAccount = registerAccount,
       _saveBindingForLease = saveBindingForLease;

  final NotificationService _service;
  final NotificationPlatform platform;
  final NotificationAccountRegistrar _registerAccount;
  final NotificationLeaseBindingSaver _saveBindingForLease;

  Map<AccountKey, AccountSessionLease> _eligible = const {};
  final Map<AccountSessionLease, String> _settledToken = {};
  String? _latestToken;
  bool _authorized = false;
  int _revision = 0;
  Future<void>? _inFlight;

  Future<void> updateAccounts(List<AccountSessionLease> accounts) async {
    final next = {for (final lease in accounts) lease.account: lease};
    if (!_sameAccounts(_eligible, next)) {
      _eligible = Map.unmodifiable(next);
      _settledToken.removeWhere(
        (lease, _) => _eligible[lease.account] != lease,
      );
      _revision++;
    }
    if (_eligible.isEmpty) {
      _setAuthorized(false);
      return;
    }

    try {
      var permission = await _service.getPermission();
      if (_eligible.isEmpty) return;
      if (permission == NotificationPermission.notDetermined) {
        permission = await _service.requestPermission();
      }
      _setAuthorized(permission == NotificationPermission.authorized);
      if (!_authorized) return;
      await _refreshToken();
      await _attemptRegistration();
    } on Object {
      _setAuthorized(false);
    }
  }

  Future<void> retryRegistration() async {
    if (_eligible.isEmpty) return;
    try {
      final permission = await _service.getPermission();
      _setAuthorized(permission == NotificationPermission.authorized);
      if (!_authorized) return;
      await _refreshToken();
      await _attemptRegistration();
    } on Object {
      _setAuthorized(false);
    }
  }

  Future<void> onTokenRefresh(String token) async {
    if (token.isEmpty) return;
    _setLatestToken(token);
    await _attemptRegistration();
  }

  void _setAuthorized(bool value) {
    if (_authorized == value) return;
    _authorized = value;
    _revision++;
  }

  void _setLatestToken(String token) {
    if (_latestToken == token) return;
    _latestToken = token;
    _revision++;
  }

  Future<void> _refreshToken() async {
    try {
      final token = await _service.getToken();
      if (token != null && token.isNotEmpty) _setLatestToken(token);
    } on Object {
      // Registration is opportunistic. A later lifecycle trigger retries.
    }
  }

  Future<void> _attemptRegistration() async {
    final existing = _inFlight;
    if (existing != null) return existing;
    if (!_authorized || _eligible.isEmpty || _latestToken == null) return;

    final attempt = _drainRegistrationChanges();
    _inFlight = attempt;
    try {
      await attempt;
    } finally {
      _inFlight = null;
    }
  }

  Future<void> _drainRegistrationChanges() async {
    while (_authorized && _eligible.isNotEmpty) {
      final token = _latestToken;
      if (token == null || token.isEmpty) return;
      final revision = _revision;
      final unsettled = _eligible.values
          .where((lease) => _settledToken[lease] != token)
          .toList(growable: false);
      await Future.wait([
        for (final lease in unsettled) _registerAndSave(lease, token),
      ]);
      if (_revision == revision) return;
    }
  }

  Future<void> _registerAndSave(
    AccountSessionLease lease,
    String token,
  ) async {
    try {
      final binding = await _registerAccount(
        lease: lease,
        platform: platform,
        token: token,
      );
      if (!_authorized ||
          _latestToken != token ||
          _eligible[lease.account] != lease) {
        return;
      }
      await _saveBindingForLease(lease: lease, binding: binding);
      if (_latestToken == token && _eligible[lease.account] == lease) {
        _settledToken[lease] = token;
      }
    } on Object {
      // Failure remains scoped to this lease and retryable on the next trigger.
    }
  }

  static bool _sameAccounts(
    Map<AccountKey, AccountSessionLease> left,
    Map<AccountKey, AccountSessionLease> right,
  ) {
    if (left.length != right.length) return false;
    for (final entry in left.entries) {
      if (right[entry.key] != entry.value) return false;
    }
    return true;
  }
}
