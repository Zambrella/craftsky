import 'package:craftsky_app/theme/brand_colors.dart';
import 'package:flutter/material.dart';

/// The five top-level profile tabs. Sealed in an enum so the page,
/// the tab bar delegate, and the tab content list all reference the
/// same source of truth and can't drift.
enum ProfileTab {
  posts(label: 'Posts'),
  projects(label: 'Projects'),
  saved(label: 'Saved'),
  reposts(label: 'Reposts'),
  about(label: 'About')
  ;

  const ProfileTab({required this.label});

  final String label;
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
    return ColoredBox(
      color: BrandColors.paper,
      child: Column(
        children: [
          Expanded(
            child: TabBar(
              isScrollable: true,
              tabAlignment: TabAlignment.start,
              padding: const EdgeInsets.symmetric(horizontal: 8),
              labelStyle: theme.textTheme.labelMedium,
              unselectedLabelStyle: theme.textTheme.labelMedium?.copyWith(
                color: BrandColors.ink3,
              ),
              labelColor: BrandColors.ink,
              unselectedLabelColor: BrandColors.ink3,
              indicatorColor: BrandColors.ink,
              dividerColor: Colors.transparent,
              tabs: [
                for (final tab in ProfileTab.values) Tab(text: _labelFor(tab)),
              ],
            ),
          ),
          Container(height: 1, color: BrandColors.borderHair),
        ],
      ),
    );
  }

  String _labelFor(ProfileTab tab) {
    return switch (tab) {
      ProfileTab.projects when projectsCountLabel != null =>
        '${tab.label} · $projectsCountLabel',
      ProfileTab.saved when savedCountLabel != null =>
        '${tab.label} · $savedCountLabel',
      _ => tab.label,
    };
  }

  @override
  bool shouldRebuild(covariant ProfileTabBarDelegate oldDelegate) {
    return projectsCountLabel != oldDelegate.projectsCountLabel ||
        savedCountLabel != oldDelegate.savedCountLabel;
  }
}
