import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:flutter/foundation.dart';

@immutable
final class AccountDeletionLeaseFence {
  const AccountDeletionLeaseFence._(this.lease, this.requiredHandle);

  factory AccountDeletionLeaseFence.capture(SessionRegistry registry) {
    final active = registry.activeLease;
    if (active == null) throw StateError('No active account');
    final stored = registry.sessions[active.session.account.did];
    if (stored == null) throw StateError('Active account unavailable');
    return AccountDeletionLeaseFence._(active, '@${stored.handle.value}');
  }

  final ActiveAccountLease lease;
  final String requiredHandle;

  AccountKey get account => lease.session.account;

  bool isCurrent(SessionRegistry registry) => registry.isCurrent(lease);

  @override
  String toString() => 'AccountDeletionLeaseFence(<redacted>)';
}
