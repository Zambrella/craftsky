import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:flutter/foundation.dart';

/// Immutable authority for async work started by one stored session.
@immutable
final class AccountSessionLease {
  const AccountSessionLease({
    required this.account,
    required this.sessionGeneration,
  });

  final AccountKey account;
  final int sessionGeneration;

  @override
  bool operator ==(Object other) =>
      other is AccountSessionLease &&
      other.account == account &&
      other.sessionGeneration == sessionGeneration;

  @override
  int get hashCode => Object.hash(account, sessionGeneration);

  @override
  String toString() => 'AccountSessionLease(<redacted>)';
}

@immutable
final class ActiveAccountLease {
  const ActiveAccountLease({
    required this.session,
    required this.activationGeneration,
  });

  final AccountSessionLease session;
  final int activationGeneration;

  @override
  bool operator ==(Object other) =>
      other is ActiveAccountLease &&
      other.session == session &&
      other.activationGeneration == activationGeneration;

  @override
  int get hashCode => Object.hash(session, activationGeneration);

  @override
  String toString() => 'ActiveAccountLease(<redacted>)';
}
