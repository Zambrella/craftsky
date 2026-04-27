import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/widgets/profile_bio.dart';
import 'package:craftsky_app/profile/widgets/profile_craft_chips.dart';
import 'package:craftsky_app/theme/brand_colors.dart';
import 'package:flutter/material.dart';
import 'package:intl/intl.dart';

/// About-tab body: bio, crafts, member-since. Returns a
/// [SliverPadding] wrapping a [SliverList] so it composes inside the
/// page's outer [CustomScrollView]. Folds gracefully when description /
/// createdAt are absent.
class ProfileAboutTab extends StatelessWidget {
  const ProfileAboutTab({required this.profile, super.key});

  final Profile profile;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final hasBio = profile.description?.isNotEmpty ?? false;

    final children = <Widget>[
      if (hasBio)
        ProfileBio(description: profile.description)
      else
        Text(
          'Nothing here yet.',
          style: theme.textTheme.bodyMedium?.copyWith(color: BrandColors.ink3),
        ),
      const SizedBox(height: 20),
      if (profile.crafts.isNotEmpty) ...[
        Text('Crafts', style: theme.textTheme.labelSmall),
        const SizedBox(height: 8),
        ProfileCraftChips(crafts: profile.crafts),
        const SizedBox(height: 20),
      ],
      if (profile.createdAt != null) ...[
        Text('Joined', style: theme.textTheme.labelSmall),
        const SizedBox(height: 4),
        Text(
          DateFormat.yMMMM().format(profile.createdAt!),
          style: theme.textTheme.bodyMedium,
        ),
      ],
    ];

    return SliverPadding(
      padding: const EdgeInsets.all(16),
      sliver: SliverList.builder(
        itemCount: children.length,
        itemBuilder: (context, index) => children[index],
      ),
    );
  }
}
