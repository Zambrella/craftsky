import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/notifications/models/notification_badge.dart';
import 'package:flutter/foundation.dart';

enum AccountSwitcherAction { select, add }

@immutable
final class AccountSwitcherRow {
  const AccountSwitcherRow({
    required this.lease,
    required this.handle,
    required this.displayName,
    required this.avatarUrl,
    required this.isCurrent,
    required this.badge,
  });

  final AccountSessionLease lease;
  final String handle;
  final String? displayName;
  final String? avatarUrl;
  final bool isCurrent;
  final NotificationBadge badge;

  String get displayLabel {
    final candidate = displayName?.trim();
    return candidate == null || candidate.isEmpty ? handle : candidate;
  }

  @override
  String toString() => 'AccountSwitcherRow(<redacted>)';
}

@immutable
final class AccountSwitcherState {
  AccountSwitcherState._({required this.rows})
    : canAddAccount = rows.length < SessionRegistry.maxRetainedAccounts,
      addAccountHelper = rows.length < SessionRegistry.maxRetainedAccounts
          ? null
          : 'Maximum of 5 accounts',
      actions = Set.unmodifiable({
        AccountSwitcherAction.select,
        if (rows.length < SessionRegistry.maxRetainedAccounts)
          AccountSwitcherAction.add,
      });

  factory AccountSwitcherState.fromRegistry(
    SessionRegistry registry, {
    Map<AccountKey, int> notificationCounts = const {},
  }) => AccountSwitcherState._(
    rows: List.unmodifiable([
      for (final session in registry.orderedSessions)
        AccountSwitcherRow(
          lease: registry.leaseFor(AccountKey(session.did.value))!,
          handle: session.handle.value,
          displayName: session.cachedDisplayName,
          avatarUrl: session.cachedAvatarUrl,
          isCurrent: session.did == registry.activeDid,
          badge: NotificationBadge.fromCount(
            notificationCounts[AccountKey(session.did.value)] ?? 0,
          ),
        ),
    ]),
  );

  final List<AccountSwitcherRow> rows;
  final bool canAddAccount;
  final String? addAccountHelper;
  final Set<AccountSwitcherAction> actions;

  @override
  String toString() => 'AccountSwitcherState(<redacted>)';
}
