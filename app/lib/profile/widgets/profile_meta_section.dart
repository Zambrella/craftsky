import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/widgets/profile_bio.dart';
import 'package:craftsky_app/profile/widgets/profile_craft_chips.dart';
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
/// Stats are hard-coded placeholders for now; counts plug in from the
/// AppView once the relevant indexers land.
class ProfileMetaSection extends StatelessWidget {
  const ProfileMetaSection({required this.profile, super.key});

  final Profile profile;

  @override
  Widget build(BuildContext context) {
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
          if (hasBio) ...[
            ProfileBio(description: profile.description),
            SizedBox(height: spacing.sp3),
          ],
          if (hasCrafts) ...[
            ProfileCraftChips(crafts: profile.crafts),
            SizedBox(height: spacing.sp3),
          ],
          // TODO(craftsky): wire real counts when follow / project
          // indexers land. Numbers chosen to mirror the design mockup.
          const ProfileStats(
            followingCount: 342,
            followerCount: 1200,
            projectCount: 15,
          ),
        ],
      ),
    );
  }
}
