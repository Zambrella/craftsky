import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:flutter/foundation.dart';

@immutable
final class AccountSwitcherRow {
  const AccountSwitcherRow({
    required this.lease,
    required this.handle,
    required this.displayName,
    required this.avatarUrl,
    required this.isCurrent,
  });

  final AccountSessionLease lease;
  final String handle;
  final String? displayName;
  final String? avatarUrl;
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
  const AccountSwitcherState._({required this.rows});

  factory AccountSwitcherState.fromRegistry(SessionRegistry registry) =>
      AccountSwitcherState._(
        rows: List.unmodifiable([
          for (final session in registry.orderedSessions)
            AccountSwitcherRow(
              lease: registry.leaseFor(AccountKey(session.did.value))!,
              handle: session.handle.value,
              displayName: session.cachedDisplayName,
              avatarUrl: session.cachedAvatarUrl,
              isCurrent: session.did == registry.activeDid,
            ),
        ]),
      );

  final List<AccountSwitcherRow> rows;
  bool get canAddAccount => rows.length < SessionRegistry.maxRetainedAccounts;

  @override
  String toString() => 'AccountSwitcherState(<redacted>)';
}
