import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/account_switcher_state.dart';
import 'package:craftsky_app/auth/widgets/account_avatar.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter/material.dart';

class AccountSwitcherContent extends StatelessWidget {
  const AccountSwitcherContent({
    required this.state,
    required this.onSelect,
    required this.onAddAccount,
    this.activating,
    this.tierBuilder,
    this.showAddAccount = true,
    super.key,
  });

  final AccountSwitcherState state;
  final ValueChanged<AccountSessionLease> onSelect;
  final VoidCallback onAddAccount;
  final AccountSessionLease? activating;
  final Widget Function(AccountSessionLease lease)? tierBuilder;
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
            AccountSwitcherRowTile(
              row: row,
              selected: row.isCurrent,
              enabled: !busy && !row.isCurrent,
              badge: tierBuilder?.call(row.lease),
              trailing: row.lease == activating
                  ? const SizedBox.square(
                      dimension: 24,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : row.isCurrent
                  ? const Icon(CraftskyIcons.check)
                  : null,
              onTap: () => onSelect(row.lease),
            ),
          if (showAddAccount) ...[
            const Divider(),
            ListTile(
              enabled: !busy && state.canAddAccount,
              leading: const Icon(CraftskyIconsBold.addAccount),
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

class AccountSwitcherRowTile extends StatelessWidget {
  const AccountSwitcherRowTile({
    required this.row,
    required this.onTap,
    this.selected = false,
    this.enabled = true,
    this.badge,
    this.trailing,
    super.key,
  });

  final AccountSwitcherRow row;
  final VoidCallback onTap;
  final bool selected;
  final bool enabled;
  final Widget? badge;
  final Widget? trailing;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Semantics(
      selected: selected,
      child: Material(
        type: MaterialType.transparency,
        child: ListTile(
          selected: selected,
          enabled: enabled,
          leading: AccountAvatar(
            avatarUrl: row.avatarUrl,
            seed: row.displayLabel(l10n.handleUnavailable),
            customisation: row.customisation,
            selected: selected,
          ),
          title: Row(
            children: [
              Expanded(
                child: Text(
                  row.displayLabel(l10n.handleUnavailable),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
              ),
              if (badge case final badge?) ...[
                const SizedBox(width: 8),
                Flexible(child: badge),
              ],
            ],
          ),
          subtitle: row.displayName?.trim().isEmpty ?? true
              ? null
              : Text(row.currentHandleLabel(l10n.handleUnavailable)),
          trailing: trailing,
          onTap: enabled ? onTap : null,
        ),
      ),
    );
  }
}
