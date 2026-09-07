import 'dart:async';

import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile_account_summary.dart';
import 'package:craftsky_app/profile/models/profile_handle.dart';
import 'package:craftsky_app/profile/widgets/profile_card_modal.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter/material.dart';

class ProfileAccountListTile extends StatelessWidget {
  const ProfileAccountListTile({required this.account, super.key});

  final ProfileAccountSummary account;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final handle = ProfileHandle(account.handle);
    final title = handle.displayLabel(
      displayName: account.displayName,
      unavailableLabel: l10n.handleUnavailable,
    );
    final handleLabel = handle.currentLabel(
      unavailableLabel: l10n.handleUnavailable,
    );
    void openProfile() => unawaited(
      showUserProfileCard(context, did: account.did),
    );
    return Semantics(
      label: '$title, $handleLabel',
      hint: l10n.profileVisitAction,
      button: true,
      onTap: openProfile,
      excludeSemantics: true,
      child: ListTile(
        title: Text(title),
        subtitle:
            handle.isAvailable ||
                (account.displayName?.trim().isNotEmpty ?? false)
            ? Text(handleLabel)
            : null,
        trailing: const Icon(CraftskyIconsBold.next),
        onTap: openProfile,
      ),
    );
  }
}
