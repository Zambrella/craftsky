import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/pending_account_deletion.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:flutter/foundation.dart';

@immutable
final class AccountDeletionLeaseFence {
  const AccountDeletionLeaseFence._(this.lease);

  factory AccountDeletionLeaseFence.capture(SessionRegistry registry) {
    final active = registry.activeLease;
    if (active == null) throw StateError('No active account');
    return AccountDeletionLeaseFence._(active);
  }

  factory AccountDeletionLeaseFence.fromPending(
    PendingAccountDeletion pending,
  ) => AccountDeletionLeaseFence._(pending.lease);

  final ActiveAccountLease lease;

  AccountKey get account => lease.session.account;

  bool isCurrent(SessionRegistry registry) => registry.isCurrent(lease);

  PendingAccountDeletion pending({
    required String jobId,
    required String confirmationDid,
    required DateTime expiresAt,
  }) => PendingAccountDeletion.capture(
    jobId: jobId,
    lease: lease,
    confirmationDid: confirmationDid,
    expiresAt: expiresAt,
  );

  @override
  String toString() => 'AccountDeletionLeaseFence(<redacted>)';
}
