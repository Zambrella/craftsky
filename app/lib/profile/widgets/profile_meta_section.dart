import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/widgets/profile_bio.dart';
import 'package:craftsky_app/profile/widgets/profile_craft_chips.dart';
import 'package:craftsky_app/profile/widgets/profile_mutual_followers_link.dart';
import 'package:craftsky_app/profile/widgets/profile_stats.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

/// Below-the-bar header content: bio, craft chips, and the
/// follow / followers / projects counts row. The avatar, action row,
/// and the large display-name + `@handle` identity block all live
/// inside `ProfileSliverAppBar` so they fade with the banner on
/// collapse — this section is purely the column of textual metadata
/// that flows below the bar and scrolls normally.
///
class ProfileMetaSection extends StatelessWidget {
  const ProfileMetaSection({
    required this.profile,
    required this.isOwnProfile,
    super.key,
  });

  final Profile profile;
  final bool isOwnProfile;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    final hasBio = profile.description?.isNotEmpty ?? false;
    final hasCrafts = profile.crafts.isNotEmpty;

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
          if (hasBio) ...[
            ProfileBio(description: profile.description),
            SizedBox(height: spacing.sp3),
          ],
          if (hasCrafts) ...[
            ProfileCraftChips(crafts: profile.crafts),
            SizedBox(height: spacing.sp3),
          ],
          if (!isOwnProfile && (profile.mutualFollowerCount ?? 0) > 0) ...[
            ProfileMutualFollowersLink(
              count: profile.mutualFollowerCount!,
              targetHandleOrDid: profile.handle.toString(),
            ),
            SizedBox(height: spacing.sp2),
          ],
          ProfileStats(profile: profile),
        ],
      ),
    );
  }
}
