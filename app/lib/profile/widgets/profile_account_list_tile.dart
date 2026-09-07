import 'dart:async';

import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile_account_summary.dart';
import 'package:craftsky_app/profile/widgets/profile_card_modal.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter/material.dart';

class ProfileAccountListTile extends StatelessWidget {
  const ProfileAccountListTile({required this.account, super.key});

  final ProfileAccountSummary account;

  @override
  Widget build(BuildContext context) {
    final title = account.displayName?.isNotEmpty ?? false
        ? account.displayName!
        : account.handle.toString();
    final handle = '@${account.handle}';
    void openProfile() => unawaited(
      showUserProfileCard(context, handleOrDid: account.handle.toString()),
    );
    return Semantics(
      label: '$title, $handle',
      hint: AppLocalizations.of(context).profileVisitAction,
      button: true,
      onTap: openProfile,
      excludeSemantics: true,
      child: ListTile(
        title: Text(title),
        subtitle: Text(handle),
        trailing: const Icon(CraftskyIconsBold.next),
        onTap: openProfile,
      ),
    );
  }
}
