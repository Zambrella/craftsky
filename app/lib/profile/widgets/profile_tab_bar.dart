import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/craftsky_divider.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

/// The five top-level profile tabs. Sealed in an enum so the page,
/// the tab bar delegate, and the tab content list all reference the
/// same source of truth and can't drift. Display labels are looked up
/// via [ProfileTabLabel.label] on the current [AppLocalizations] —
/// keeping the enum locale-agnostic.
enum ProfileTab { projects, posts, comments, reposts, about }

extension ProfileTabLabel on ProfileTab {
  /// Localised tab label for [AppLocalizations].
  String label(AppLocalizations l10n) => switch (this) {
    ProfileTab.posts => l10n.profileTabPosts,
    ProfileTab.comments => l10n.profileTabComments,
    ProfileTab.projects => l10n.profileTabProjects,
    ProfileTab.reposts => l10n.profileTabReposts,
    ProfileTab.about => l10n.profileTabAbout,
  };
}

/// Sticky tab bar for the profile screen. Pinned via
/// [SliverPersistentHeader] above the [TabBarView] body so tabs stay
/// reachable while the post list scrolls under them.
class ProfileTabBarDelegate extends SliverPersistentHeaderDelegate {
  const ProfileTabBarDelegate({this.projectsCountLabel});

  /// Optional inline counts ("Projects · 15"). Mockup hints at this; real
  /// counts plug in once feed data is wired.
  final String? projectsCountLabel;

  static const double height = 48;

  @override
  double get minExtent => height;

  @override
  double get maxExtent => height;

  @override
  Widget build(
    BuildContext context,
    double shrinkOffset,
    bool overlapsContent,
  ) {
    final theme = Theme.of(context);
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final spacing = theme.extension<SpacingTheme>()!;
    final l10n = AppLocalizations.of(context);
    return ColoredBox(
      color: swatches.paper,
      child: Column(
        children: [
          Expanded(
            child: TabBar(
              isScrollable: true,
              tabAlignment: TabAlignment.start,
              padding: EdgeInsets.symmetric(horizontal: spacing.sp2),
              tabs: [
                for (final tab in ProfileTab.values)
                  Tab(text: _labelFor(tab, l10n)),
              ],
            ),
          ),
          const CraftskyDivider(),
        ],
      ),
    );
  }

  String _labelFor(ProfileTab tab, AppLocalizations l10n) {
    final base = tab.label(l10n);
    return switch (tab) {
      ProfileTab.projects when projectsCountLabel != null =>
        '$base · $projectsCountLabel',
      _ => base,
    };
  }

  @override
  bool shouldRebuild(covariant ProfileTabBarDelegate oldDelegate) {
    return projectsCountLabel != oldDelegate.projectsCountLabel;
  }
}
