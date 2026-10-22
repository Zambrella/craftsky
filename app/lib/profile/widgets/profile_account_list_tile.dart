import 'dart:async';

import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile_account_summary.dart';
import 'package:craftsky_app/profile/models/profile_handle.dart';
import 'package:craftsky_app/profile/widgets/compact_profile_identity.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/profile/widgets/profile_card_modal.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:material_ui/material_ui.dart';

class ProfileAccountListTile extends StatelessWidget {
  const new({
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
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    final handle = ProfileHandle(account.handle);
    final title = handle.currentLabel(
      unavailableLabel: l10n.handleUnavailable,
    );
    final name = account.displayName?.trim();
    void openProfile() => onTap != null
        ? onTap!()
        : unawaited(showUserProfileCard(context, did: account.did));
    return Semantics(
      label: [
        title,
        if (name != null && name.isNotEmpty) name,
        for (final craft in account.crafts) craft.split('#').last,
      ].join(', '),
      hint: l10n.profileVisitAction,
      button: true,
      onTap: openProfile,
      excludeSemantics: trailing == null,
      child: ListTile(
        horizontalTitleGap: 12,
        minLeadingWidth: 0,
        contentPadding: EdgeInsets.symmetric(horizontal: spacing.sp4),
        leading: ProfileAvatar(
          seed: handle.displayLabel(
            displayName: account.displayName,
            unavailableLabel: l10n.handleUnavailable,
          ),
          avatarUrl: account.avatar,
          size: ProfileAvatarSize.small,
          customisation: account.customisation,
        ),
        title: CompactProfileIdentity(
          handle: account.handle.toString(),
          displayName: account.displayName,
          crafts: account.crafts,
        ),
        trailing: trailing,
        onTap: openProfile,
      ),
    );
  }
}
