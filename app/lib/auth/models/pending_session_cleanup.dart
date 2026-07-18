import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:flutter/foundation.dart';

@immutable
final class PendingSessionCleanup {
  const PendingSessionCleanup({
    required this.account,
    required this.sessionGeneration,
    required this.token,
  });

  final AccountKey account;
  final int sessionGeneration;
  final String token;

  @override
  bool operator ==(Object other) =>
      other is PendingSessionCleanup &&
      other.account == account &&
      other.sessionGeneration == sessionGeneration;

  @override
  int get hashCode => Object.hash(account, sessionGeneration);

  @override
  String toString() => 'PendingSessionCleanup(<redacted>)';
}
