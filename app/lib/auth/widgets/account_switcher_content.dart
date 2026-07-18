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
    this.activating,
    super.key,
  });

  final AccountSwitcherState state;
  final ValueChanged<AccountSessionLease> onSelect;
  final VoidCallback onAddAccount;
  final AccountSessionLease? activating;

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
      ),
    );
  }
}
