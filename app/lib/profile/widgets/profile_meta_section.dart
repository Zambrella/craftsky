import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/moderation/widgets/moderation_warning_banner.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/widgets/profile_actions.dart';
import 'package:craftsky_app/profile/widgets/profile_bio.dart';
import 'package:craftsky_app/profile/widgets/profile_mutual_followers_link.dart';
import 'package:craftsky_app/profile/widgets/profile_stats.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

/// The full profile details below the collapsing identity header.
///
/// Keeps the card-style summary stats and richer full-page actions together,
/// then places the long-form bio at the bottom before the profile tabs.
class ProfileMetaSection extends StatelessWidget {
  const ProfileMetaSection({
    required this.profile,
    required this.isOwnProfile,
    required this.actions,
    super.key,
  });

  final Profile profile;
  final bool isOwnProfile;
  final ProfileActionSet actions;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    final hasBio = profile.description?.isNotEmpty ?? false;

    return Padding(
      padding: EdgeInsets.fromLTRB(
        spacing.sp4,
        spacing.sp1,
        spacing.sp4,
        spacing.sp4,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (!profile.isCraftskyProfile) ...[
            Text(
              l10n.profileNonCraftskyMarker,
              style: Theme.of(context).textTheme.bodySmall?.copyWith(
                color: Theme.of(context).colorScheme.onSurfaceVariant,
              ),
            ),
            SizedBox(height: spacing.sp3),
          ],
          if (profile.moderation?.warningKind case final kind?) ...[
            ModerationWarningBanner(warningKind: kind),
            SizedBox(height: spacing.sp3),
          ],
          ProfileStats(profile: profile),
          SizedBox(height: spacing.sp4),
          ProfileActionSection(actions: actions),
          if (!isOwnProfile && (profile.mutualFollowerCount ?? 0) > 0) ...[
            SizedBox(height: spacing.sp3),
            ProfileMutualFollowersLink(
              count: profile.mutualFollowerCount!,
              targetHandleOrDid: profile.handle.toString(),
            ),
          ],
          if (hasBio) ...[
            SizedBox(height: spacing.sp3),
            ProfileBio(description: profile.description),
          ],
        ],
      ),
    );
  }
}
