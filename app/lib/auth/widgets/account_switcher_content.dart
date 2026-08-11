import 'package:craftsky_app/auth/models/account_deletion.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/account_switcher_state.dart';
import 'package:craftsky_app/auth/widgets/account_avatar.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:flutter/material.dart';

class AccountSwitcherContent extends StatelessWidget {
  const AccountSwitcherContent({
    required this.state,
    required this.onSelect,
    required this.onAddAccount,
    this.onOpenDeletionStatus,
    this.activating,
    this.showAddAccount = true,
    super.key,
  });

  final AccountSwitcherState state;
  final ValueChanged<AccountSessionLease> onSelect;
  final VoidCallback onAddAccount;
  final ValueChanged<String>? onOpenDeletionStatus;
  final AccountSessionLease? activating;
  final bool showAddAccount;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final busy = activating != null;
    return SafeArea(
      child: ListView(
        shrinkWrap: true,
        padding: const EdgeInsets.symmetric(vertical: 8),
        children: [
          for (final row in state.rows)
            Semantics(
              selected: row.isCurrent,
              child: ListTile(
                selected: row.isCurrent,
                enabled: !busy && !row.isCurrent,
                leading: AccountAvatar(
                  avatarUrl: row.avatarUrl,
                  seed: row.displayLabel,
                  customisation: row.customisation,
                  selected: row.isCurrent,
                ),
                title: Text(row.displayLabel),
                subtitle: row.displayLabel == row.handle
                    ? null
                    : Text('@${row.handle}'),
                trailing: row.lease == activating
                    ? const SizedBox.square(
                        dimension: 24,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                    : row.isCurrent
                    ? const Icon(Icons.check)
                    : null,
                onTap: busy || row.isCurrent ? null : () => onSelect(row.lease),
              ),
            ),
          for (final row in state.deletingRows)
            Semantics(
              enabled: false,
              child: ListTile(
                leading: AccountAvatar(
                  avatarUrl: row.avatarUrl,
                  seed: row.displayLabel,
                ),
                title: Text(row.displayLabel),
                subtitle: Text(
                  '${row.displayLabel == row.handle ? '' : '@${row.handle} · '}'
                  '${_deletionLabel(l10n, row)}',
                ),
                trailing: const Icon(Icons.hourglass_top),
                onTap: busy || onOpenDeletionStatus == null
                    ? null
                    : () => onOpenDeletionStatus!(row.jobId),
              ),
            ),
          if (showAddAccount) ...[
            const Divider(),
            ListTile(
              enabled: !busy && state.canAddAccount,
              leading: const Icon(Icons.person_add_alt_1),
              title: Text(l10n.accountSwitcherAdd),
              subtitle: state.canAddAccount
                  ? null
                  : Text(l10n.accountSwitcherMaximum),
              onTap: !busy && state.canAddAccount ? onAddAccount : null,
            ),
          ],
        ],
      ),
    );
  }
}

String _deletionLabel(
  AppLocalizations l10n,
  DeletingAccountSwitcherRow row,
) {
  if (row.status == AccountDeletionStatus.needsAttention) {
    return l10n.accountDeletionNeedsAttention;
  }
  if (row.status == AccountDeletionStatus.retrying) {
    return l10n.accountDeletionRetrying;
  }
  return switch (row.phase) {
    AccountDeletionPhase.preparing => l10n.accountDeletionPreparing,
    AccountDeletionPhase.removingPrivateData =>
      l10n.accountDeletionRemovingPrivateData,
    AccountDeletionPhase.removingCraftskyRecords =>
      l10n.accountDeletionRemovingRecords,
    AccountDeletionPhase.waitingForCraftsky =>
      l10n.accountDeletionWaitingForCraftsky,
    AccountDeletionPhase.finalizing => l10n.accountDeletionFinalizing,
    AccountDeletionPhase.deleted => l10n.accountDeletionDeleted,
  };
}
