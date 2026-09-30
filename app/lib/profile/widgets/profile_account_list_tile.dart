import 'dart:async';

import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile_account_summary.dart';
import 'package:craftsky_app/profile/models/profile_handle.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/profile/widgets/profile_card_modal.dart';
import 'package:craftsky_app/shared/widgets/craft_icon.dart';
import 'package:flutter/material.dart';

class ProfileAccountListTile extends StatelessWidget {
  const ProfileAccountListTile({
    required this.account,
    this.onTap,
    this.trailing,
    super.key,
  });

  final ProfileAccountSummary account;
  final VoidCallback? onTap;
  final Widget? trailing;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final handle = ProfileHandle(account.handle);
    final title = handle.currentLabel(
      unavailableLabel: l10n.handleUnavailable,
    );
    final name = account.displayName?.trim();
    final iconCrafts = account.crafts
        .where((craft) => CraftIcon.assetPathFor(craft) != null)
        .toList(growable: false);
    void openProfile() => onTap != null
        ? onTap!()
        : unawaited(showUserProfileCard(context, did: account.did));
    return Semantics(
      label: [
        title,
        if (name != null && name.isNotEmpty) name,
        for (final craft in iconCrafts) craft.split('#').last,
      ].join(', '),
      hint: l10n.profileVisitAction,
      button: true,
      onTap: openProfile,
      excludeSemantics: trailing == null,
      child: ListTile(
        contentPadding: const EdgeInsets.symmetric(horizontal: 16),
        leading: ProfileAvatar(
          seed: handle.displayLabel(
            displayName: account.displayName,
            unavailableLabel: l10n.handleUnavailable,
          ),
          avatarUrl: account.avatar,
          size: ProfileAvatarSize.small,
          customisation: account.customisation,
        ),
        title: Text(title),
        subtitle: (name == null || name.isEmpty) && iconCrafts.isEmpty
            ? null
            : Row(
                children: [
                  if (name != null && name.isNotEmpty) ...[
                    Flexible(
                      child: Text(
                        name,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                      ),
                    ),
                    if (iconCrafts.isNotEmpty) const SizedBox(width: 8),
                  ],
                  for (final craft in iconCrafts) ...[
                    CraftIcon(craft: craft, size: 16),
                    const SizedBox(width: 4),
                  ],
                ],
              ),
        trailing: trailing,
        onTap: openProfile,
      ),
    );
  }
}
