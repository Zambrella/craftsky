import 'dart:async';

import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/providers/active_account_identity_provider.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/providers/account_type_controller.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/settings/models/delete_account_confirmation.dart';
import 'package:craftsky_app/settings/models/settings_row.dart';
import 'package:craftsky_app/settings/providers/account_deletion_controller.dart';
import 'package:craftsky_app/settings/widgets/settings_row_tile.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class AccountPage extends ConsumerWidget {
  const AccountPage({this.onDeleteConfirmed, super.key});

  final Future<void> Function(String handle)? onDeleteConfirmed;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final auth = ref.watch(authSessionProvider).value;
    final handle = auth is SignedIn ? '@${auth.handle.value}' : null;
    final profileType = ref
        .watch(activeAccountIdentityProvider)
        .value
        ?.profile
        .accountType;
    final accountTypeState = ref.watch(accountTypeControllerProvider);
    final accountType = accountTypeState.value ?? profileType;
    return Scaffold(
      appBar: AppBar(title: Text(l10n.accountTitle)),
      body: ListView(
        children: [
          if (accountType != null) ...[
            Padding(
              padding: const EdgeInsetsDirectional.fromSTEB(16, 20, 16, 8),
              child: Text(
                l10n.accountTypeTitle,
                style: Theme.of(context).textTheme.titleSmall?.copyWith(
                  color: Theme.of(context).colorScheme.primary,
                ),
              ),
            ),
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 16),
              child: SegmentedButton<AccountType>(
                expandedInsets: EdgeInsets.zero,
                segments: [
                  ButtonSegment(
                    value: AccountType.regular,
                    label: Text(l10n.accountTypeRegular),
                  ),
                  ButtonSegment(
                    value: AccountType.business,
                    label: Text(l10n.accountTypeBusiness),
                  ),
                ],
                selected: {accountType},
                onSelectionChanged: accountTypeState.isLoading
                    ? null
                    : (selected) => unawaited(
                        _setAccountType(context, ref, selected.single),
                      ),
              ),
            ),
            const SizedBox(height: 12),
          ],
          SettingsRowTile(
            descriptor: const SettingsRowDescriptor(
              id: SettingsRowId.deleteAccount,
              kind: SettingsRowKind.destructiveAction,
            ),
            label: l10n.deleteAccountAction,
            leading: Icons.delete_forever_outlined,
            onTap: handle == null ? null : () => _begin(context, ref, handle),
          ),
        ],
      ),
    );
  }

  Future<void> _setAccountType(
    BuildContext context,
    WidgetRef ref,
    AccountType accountType,
  ) async {
    final succeeded = await ref
        .read(accountTypeControllerProvider.notifier)
        .setAccountType(accountType);
    if (!succeeded && context.mounted) {
      context.showError(AppLocalizations.of(context).errorActionFailed);
    }
  }

  Future<void> _begin(
    BuildContext context,
    WidgetRef ref,
    String handle,
  ) async {
    final l10n = AppLocalizations.of(context);
    final proceed = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        scrollable: true,
        title: Text(l10n.deleteAccountTitle),
        content: Text(l10n.deleteAccountBoundary(handle)),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(dialogContext, false),
            child: Text(l10n.actionCancel),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(dialogContext, true),
            child: Text(l10n.deleteAccountContinue),
          ),
        ],
      ),
    );
    if (proceed != true || !context.mounted) return;
    if (onDeleteConfirmed case final callback?) {
      await _confirmHandle(context, handle, callback);
      return;
    }
    final jobId = await ref
        .read(accountDeletionControllerProvider.notifier)
        .startReauthentication();
    if (jobId == null && context.mounted) {
      context.showError(AppLocalizations.of(context).errorActionFailed);
    }
  }

  Future<void> _confirmHandle(
    BuildContext context,
    String handle,
    Future<void> Function(String handle) callback,
  ) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (_) => _HandleConfirmationDialog(requiredHandle: handle),
    );
    if (confirmed == true) await callback(handle);
  }
}

class _HandleConfirmationDialog extends StatefulWidget {
  const _HandleConfirmationDialog({required this.requiredHandle});

  final String requiredHandle;

  @override
  State<_HandleConfirmationDialog> createState() =>
      _HandleConfirmationDialogState();
}

class _HandleConfirmationDialogState extends State<_HandleConfirmationDialog> {
  final _controller = TextEditingController();

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return AlertDialog(
      title: Text(l10n.deleteAccountConfirmTitle),
      content: SingleChildScrollView(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(
              l10n.deleteAccountConfirmationPrompt(widget.requiredHandle),
            ),
            const SizedBox(height: 16),
            TextField(
              controller: _controller,
              autocorrect: false,
              enableSuggestions: false,
              decoration: InputDecoration(
                labelText: l10n.deleteAccountTypeHandleLabel,
              ),
              onChanged: (_) => setState(() {}),
            ),
          ],
        ),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(context, false),
          child: Text(l10n.actionCancel),
        ),
        FilledButton(
          onPressed:
              matchesDeletionConfirmationHandle(
                requiredHandle: widget.requiredHandle,
                input: _controller.text,
              )
              ? () => Navigator.pop(context, true)
              : null,
          child: Text(l10n.deleteAccountAction),
        ),
      ],
    );
  }
}
