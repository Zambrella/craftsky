import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

/// The five top-level profile tabs. Sealed in an enum so the page,
/// the tab bar delegate, and the tab content list all reference the
/// same source of truth and can't drift. Display labels are looked up
/// via [ProfileTabLabel.label] on the current [AppLocalizations] —
/// keeping the enum locale-agnostic.
enum ProfileTab { posts, projects, saved, reposts, about }

extension ProfileTabLabel on ProfileTab {
  /// Localised tab label for [AppLocalizations].
  String label(AppLocalizations l10n) => switch (this) {
    ProfileTab.posts => l10n.profileTabPosts,
    ProfileTab.projects => l10n.profileTabProjects,
    ProfileTab.saved => l10n.profileTabSaved,
    ProfileTab.reposts => l10n.profileTabReposts,
    ProfileTab.about => l10n.profileTabAbout,
  };
}

/// Sticky tab bar for the profile screen. Pinned via
/// [SliverPersistentHeader] above the [TabBarView] body so tabs stay
/// reachable while the post list scrolls under them.
class ProfileTabBarDelegate extends SliverPersistentHeaderDelegate {
  const ProfileTabBarDelegate({
    this.projectsCountLabel,
    this.savedCountLabel,
  });

  /// Optional inline counts ("Projects · 15"). Mockup hints at this; real
  /// counts plug in once feed data is wired.
  final String? projectsCountLabel;
  final String? savedCountLabel;

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
    final onSurface = theme.colorScheme.onSurface;
    // `outline` carries the brand's ink3 (tertiary text) per the
    // ColorScheme override in app_theme.dart.
    final muted = theme.colorScheme.outline;
    return ColoredBox(
      color: swatches.paper,
      child: Column(
        children: [
          Expanded(
            child: TabBar(
              isScrollable: true,
              tabAlignment: TabAlignment.start,
              padding: EdgeInsets.symmetric(horizontal: spacing.sp2),
              labelStyle: theme.textTheme.labelMedium,
              unselectedLabelStyle: theme.textTheme.labelMedium?.copyWith(
                color: muted,
              ),
              labelColor: onSurface,
              unselectedLabelColor: muted,
              indicatorColor: onSurface,
              dividerColor: Colors.transparent,
              tabs: [
                for (final tab in ProfileTab.values)
                  Tab(text: _labelFor(tab, l10n)),
              ],
            ),
          ),
          Container(height: 1, color: swatches.borderHair),
        ],
      ),
    );
  }

  String _labelFor(ProfileTab tab, AppLocalizations l10n) {
    final base = tab.label(l10n);
    return switch (tab) {
      ProfileTab.projects when projectsCountLabel != null =>
        '$base · $projectsCountLabel',
      ProfileTab.saved when savedCountLabel != null =>
        '$base · $savedCountLabel',
      _ => base,
    };
  }

  @override
  bool shouldRebuild(covariant ProfileTabBarDelegate oldDelegate) {
    return projectsCountLabel != oldDelegate.projectsCountLabel ||
        savedCountLabel != oldDelegate.savedCountLabel;
  }
}
