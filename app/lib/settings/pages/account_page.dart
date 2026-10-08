import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/providers/active_account_identity_provider.dart';
import 'package:craftsky_app/auth/providers/active_account_initialization_provider.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/settings/models/delete_account_confirmation.dart';
import 'package:craftsky_app/settings/models/settings_row.dart';
import 'package:craftsky_app/settings/providers/account_deletion_controller.dart';
import 'package:craftsky_app/settings/widgets/settings_row_tile.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_access_provider.dart';
import 'package:craftsky_app/theme/brand_colors.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class AccountPage extends ConsumerWidget {
  const AccountPage({this.onDeleteConfirmed, super.key});

  final Future<void> Function(String did)? onDeleteConfirmed;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final auth = ref.watch(authSessionProvider).value;
    final did = auth is SignedIn ? auth.did.value : null;
    final profileType = ref
        .watch(activeAccountIdentityProvider)
        .value
        ?.profile
        .accountType;
    final lease = ref
        .watch(sessionRegistryProvider)
        .value
        ?.activeLease
        ?.session;
    final access = lease == null
        ? null
        : ref.watch(subscriptionAccessProvider(lease));
    final accountType = profileType == null
        ? null
        : switch (access) {
            AsyncData(:final value) when value.allowsBusiness =>
              AccountType.business,
            _ => AccountType.regular,
          };
    final isRestricted =
        ref
            .watch(activeAccountInitializationProvider)
            .value
            ?.accountEligibility
            .restricted ??
        false;
    return Scaffold(
      appBar: AppBar(title: Text(l10n.accountTitle)),
      body: ListView(
        children: [
          if (!isRestricted && accountType != null) ...[
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
              child: Text(
                accountType == AccountType.business
                    ? l10n.accountTypeBusiness
                    : l10n.accountTypeRegular,
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
            leading: CraftskyIconsBold.deleteForever,
            onTap: did == null ? null : () => _begin(context, ref, did),
          ),
        ],
      ),
    );
  }

  Future<void> _begin(
    BuildContext context,
    WidgetRef ref,
    String did,
  ) async {
    final l10n = AppLocalizations.of(context);
    final proceed = await showCraftskyDestructiveConfirmDialog(
      context,
      title: l10n.deleteAccountTitle,
      message: l10n.deleteAccountDidBoundary(did),
      confirmLabel: l10n.deleteAccountContinue,
      cancelLabel: l10n.actionCancel,
    );
    if (!proceed || !context.mounted) return;
    if (onDeleteConfirmed case final callback?) {
      await _confirmDid(context, did, callback);
      return;
    }
    final jobId = await ref
        .read(accountDeletionControllerProvider.notifier)
        .startReauthentication();
    if (jobId == null && context.mounted) {
      context.showError(AppLocalizations.of(context).errorActionFailed);
    }
  }

  Future<void> _confirmDid(
    BuildContext context,
    String did,
    Future<void> Function(String did) callback,
  ) async {
    final confirmed = await showCraftskyModal<bool>(
      context,
      builder: (_) => _DidConfirmationDialog(confirmationDid: did),
    );
    if (confirmed == true) await callback(did);
  }
}

class _DidConfirmationDialog extends StatefulWidget {
  const _DidConfirmationDialog({required this.confirmationDid});

  final String confirmationDid;

  @override
  State<_DidConfirmationDialog> createState() => _DidConfirmationDialogState();
}

class _DidConfirmationDialogState extends State<_DidConfirmationDialog> {
  final _controller = TextEditingController();

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return CraftskyDialog(
      title: l10n.deleteAccountConfirmTitle,
      body: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(
            l10n.deleteAccountDidConfirmationPrompt(widget.confirmationDid),
          ),
          const SizedBox(height: 16),
          TextField(
            controller: _controller,
            autocorrect: false,
            enableSuggestions: false,
            decoration: InputDecoration(
              labelText: l10n.deleteAccountTypeDidLabel,
            ),
            onChanged: (_) => setState(() {}),
          ),
        ],
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(context, false),
          child: Text(l10n.actionCancel),
        ),
        ChunkyButton(
          backgroundColor: BrandColors.red,
          onPressed:
              matchesDeletionConfirmationDid(
                confirmationDid: widget.confirmationDid,
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
