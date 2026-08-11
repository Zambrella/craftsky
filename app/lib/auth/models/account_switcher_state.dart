import 'package:craftsky_app/auth/models/account_deletion.dart';
import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/profile/models/profile_customisation.dart';
import 'package:flutter/foundation.dart';

@immutable
final class AccountSwitcherRow {
  const AccountSwitcherRow({
    required this.lease,
    required this.handle,
    required this.displayName,
    required this.avatarUrl,
    required this.customisation,
    required this.isCurrent,
  });

  final AccountSessionLease lease;
  final String handle;
  final String? displayName;
  final String? avatarUrl;
  final ProfileCustomisation customisation;
  final bool isCurrent;

  String get displayLabel {
    final candidate = displayName?.trim();
    return candidate == null || candidate.isEmpty ? handle : candidate;
  }

  @override
  String toString() => 'AccountSwitcherRow(<redacted>)';
}

@immutable
final class AccountSwitcherState {
  const AccountSwitcherState._({
    required this.rows,
    required this.deletingRows,
  });

  factory AccountSwitcherState.fromRegistry(SessionRegistry registry) =>
      AccountSwitcherState._(
        rows: List.unmodifiable([
          for (final session in registry.orderedSessions)
            AccountSwitcherRow(
              lease: registry.leaseFor(AccountKey(session.did.value))!,
              handle: session.handle.value,
              displayName: session.cachedDisplayName,
              avatarUrl: session.cachedAvatarUrl,
              customisation: session.cachedCustomisation,
              isCurrent: session.did == registry.activeDid,
            ),
        ]),
        deletingRows: const [],
      );

  factory AccountSwitcherState.fromRegistries({
    required SessionRegistry sessions,
    required DeletionStatusRegistry deletions,
  }) => AccountSwitcherState._(
    rows: AccountSwitcherState.fromRegistry(sessions).rows,
    deletingRows: List.unmodifiable([
      for (final entry in deletions.entries)
        if (!entry.isTerminal && entry.status != AccountDeletionStatus.intent)
          DeletingAccountSwitcherRow(entry),
    ]),
  );

  final List<AccountSwitcherRow> rows;
  final List<DeletingAccountSwitcherRow> deletingRows;
  bool get canAddAccount =>
      rows.length + deletingRows.length < SessionRegistry.maxRetainedAccounts;

  @override
  String toString() => 'AccountSwitcherState(<redacted>)';
}

@immutable
final class DeletingAccountSwitcherRow {
  const DeletingAccountSwitcherRow(this._entry);

  final DeletionStatusEntry _entry;

  String get jobId => _entry.jobId;
  String get handle => _entry.handle.value;
  String get displayLabel => _entry.displayLabel;
  String? get avatarUrl => _entry.avatarUrl;
  AccountDeletionStatus get status => _entry.status;
  AccountDeletionPhase get phase => _entry.phase;
  bool get canRetry => _entry.canRetry;
  bool get needsReauthentication => _entry.needsReauthentication;
  bool get canActivate => false;

  @override
  String toString() => 'DeletingAccountSwitcherRow(<redacted>)';
}
