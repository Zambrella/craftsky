import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/widgets/profile_bio.dart';
import 'package:craftsky_app/profile/widgets/profile_craft_chips.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
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
    final spacing = theme.extension<SpacingTheme>()!;
    final l10n = AppLocalizations.of(context);
    final hasBio = profile.description?.isNotEmpty ?? false;

    // 20px sits between sp4(16) and sp5(24) — used as the section
    // gap. Kept as a literal because it doesn't map to a SpacingTheme
    // token.
    const sectionGap = SizedBox(height: 20);

    final children = <Widget>[
      if (hasBio)
        ProfileBio(description: profile.description)
      else
        Text(
          l10n.profileAboutEmpty,
          // `outline` carries the brand's ink3 (tertiary text) per
          // the ColorScheme override in app_theme.dart.
          style: theme.textTheme.bodyMedium?.copyWith(
            color: theme.colorScheme.outline,
          ),
        ),
      sectionGap,
      if (profile.crafts.isNotEmpty) ...[
        Text(l10n.profileAboutCraftsHeading, style: theme.textTheme.labelSmall),
        SizedBox(height: spacing.sp2),
        ProfileCraftChips(crafts: profile.crafts),
        sectionGap,
      ],
      if (profile.createdAt != null) ...[
        Text(l10n.profileAboutJoinedHeading, style: theme.textTheme.labelSmall),
        SizedBox(height: spacing.sp1),
        Text(
          // Pass the active locale so e.g. "April 2026" / "avril 2026"
          // follow the user's language rather than the intl default.
          DateFormat.yMMMM(l10n.localeName).format(profile.createdAt!),
          style: theme.textTheme.bodyMedium,
        ),
      ],
    ];

    return SliverPadding(
      padding: EdgeInsets.all(spacing.sp4),
      sliver: SliverList.builder(
        itemCount: children.length,
        itemBuilder: (context, index) => children[index],
      ),
    );
  }
}
