import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/pending_account_deletion.dart';
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

  factory AccountDeletionLeaseFence.fromPending(
    PendingAccountDeletion pending,
  ) => AccountDeletionLeaseFence._(pending.lease, pending.requiredHandle);

  final ActiveAccountLease lease;
  final String requiredHandle;

  AccountKey get account => lease.session.account;

  bool isCurrent(SessionRegistry registry) => registry.isCurrent(lease);

  PendingAccountDeletion pending({
    required String jobId,
    required DateTime expiresAt,
  }) => PendingAccountDeletion.capture(
    jobId: jobId,
    lease: lease,
    handle: requiredHandle,
    expiresAt: expiresAt,
  );

  @override
  String toString() => 'AccountDeletionLeaseFence(<redacted>)';
}
