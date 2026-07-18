import 'dart:async';

import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';

enum AccountActivationResult { activated, alreadyActive, stale, cancelled }

/// Enforces the local account boundary independently of network availability.
class AccountActivationCoordinator {
  AccountActivationCoordinator({
    required this.readRegistry,
    required this.commitActivation,
    required this.invalidateAccountState,
    required this.resetToHome,
    Future<bool> Function(AccountSessionLease owner)? confirmLeave,
  }) : confirmLeave = confirmLeave ?? _allowLeave;

  final SessionRegistry Function() readRegistry;
  final Future<void> Function(AccountSessionLease lease) commitActivation;
  final Future<void> Function() invalidateAccountState;
  final Future<void> Function() resetToHome;
  final Future<bool> Function(AccountSessionLease owner) confirmLeave;

  AccountSessionLease? _inFlightTarget;
  Future<AccountActivationResult>? _inFlight;

  Future<AccountActivationResult> activate(AccountSessionLease target) {
    if (_inFlightTarget == target) return _inFlight!;
    final operation = _activate(target);
    _inFlightTarget = target;
    _inFlight = operation;
    unawaited(
      operation.then<void>(
        (_) => _clearInFlight(operation),
        onError: (Object _, StackTrace _) => _clearInFlight(operation),
      ),
    );
    return operation;
  }

  Future<AccountActivationResult> _activate(AccountSessionLease target) async {
    final registry = readRegistry();
    if (registry.leaseFor(target.account) != target) {
      return AccountActivationResult.stale;
    }
    if (registry.activeDid == target.account.did) {
      return AccountActivationResult.alreadyActive;
    }

    final owner = registry.activeLease?.session;
    if (owner != null && !await confirmLeave(owner)) {
      return AccountActivationResult.cancelled;
    }
    final current = readRegistry();
    if (current.leaseFor(target.account) != target) {
      return AccountActivationResult.stale;
    }
    if (current.activeDid == target.account.did) {
      return AccountActivationResult.alreadyActive;
    }

    await invalidateAccountState();
    await commitActivation(target);
    await resetToHome();
    return AccountActivationResult.activated;
  }

  static Future<bool> _allowLeave(AccountSessionLease _) async => true;

  void _clearInFlight(Future<AccountActivationResult> operation) {
    if (!identical(_inFlight, operation)) return;
    _inFlight = null;
    _inFlightTarget = null;
  }
}
