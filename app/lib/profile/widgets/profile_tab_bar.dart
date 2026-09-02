import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/craftsky_divider.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

/// The top-level profile tabs. Sealed in an enum so the page,
/// the tab bar delegate, and the tab content list all reference the
/// same source of truth and can't drift. Display labels are looked up
/// via [ProfileTabLabel.label] on the current [AppLocalizations] —
/// keeping the enum locale-agnostic.
enum ProfileTab {
  projects,
  posts,
  comments,
  reposts,
  products,
  upcomingEvents,
  about,
}

abstract final class ProfileTabPolicy {
  static const ordinaryTabs = <ProfileTab>[
    ProfileTab.projects,
    ProfileTab.posts,
    ProfileTab.comments,
    ProfileTab.reposts,
  ];

  static const businessTabs = <ProfileTab>[
    ProfileTab.products,
    ProfileTab.upcomingEvents,
    ProfileTab.about,
    ProfileTab.projects,
    ProfileTab.posts,
    ProfileTab.comments,
    ProfileTab.reposts,
  ];

  static List<ProfileTab> forProfile({
    required AccountType? accountType,
    required bool isBlocked,
    bool isOwnProfile = false,
    bool hasProducts = false,
    bool hasUpcomingEvents = false,
  }) {
    if (isBlocked || accountType != AccountType.business) {
      return ordinaryTabs;
    }
    return [
      for (final tab in businessTabs)
        if (tab != ProfileTab.products || isOwnProfile || hasProducts)
          if (tab != ProfileTab.upcomingEvents ||
              isOwnProfile ||
              hasUpcomingEvents)
            tab,
    ];
  }

  static ProfileTab selectionAfterChange({
    required ProfileTab selected,
    required List<ProfileTab> tabs,
  }) => tabs.contains(selected) ? selected : tabs.first;
}

extension ProfileTabLabel on ProfileTab {
  /// Localised tab label for [AppLocalizations].
  String label(AppLocalizations l10n) => switch (this) {
    ProfileTab.posts => l10n.profileTabPosts,
    ProfileTab.comments => l10n.profileTabComments,
    ProfileTab.projects => l10n.profileTabProjects,
    ProfileTab.reposts => l10n.profileTabReposts,
    ProfileTab.products => l10n.profileTabProducts,
    ProfileTab.upcomingEvents => l10n.profileTabUpcomingEvents,
    ProfileTab.about => l10n.profileTabAbout,
  };

  String get storageKey => 'profile_tab_$name';
}

/// Sticky tab bar for the profile screen. Pinned via
/// [SliverPersistentHeader] above the [TabBarView] body so tabs stay
/// reachable while the post list scrolls under them.
class ProfileTabBarDelegate extends SliverPersistentHeaderDelegate {
  const ProfileTabBarDelegate({
    this.tabs = ProfileTabPolicy.ordinaryTabs,
    this.controller,
    this.projectsCountLabel,
  });

  final List<ProfileTab> tabs;
  final TabController? controller;

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
  ) => ProfileTabBar(
    tabs: tabs,
    controller: controller,
    projectsCountLabel: projectsCountLabel,
  );

  @override
  bool shouldRebuild(covariant ProfileTabBarDelegate oldDelegate) {
    return tabs != oldDelegate.tabs ||
        controller != oldDelegate.controller ||
        projectsCountLabel != oldDelegate.projectsCountLabel;
  }
}

/// The reusable visual tab bar used by the full profile and its expanding
/// presentation.
class ProfileTabBar extends StatelessWidget {
  const ProfileTabBar({
    this.tabs = ProfileTabPolicy.ordinaryTabs,
    this.controller,
    this.projectsCountLabel,
    super.key,
  });

  final List<ProfileTab> tabs;
  final TabController? controller;
  final String? projectsCountLabel;

  @override
  Widget build(BuildContext context) {
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
              controller: controller,
              isScrollable: true,
              tabAlignment: TabAlignment.start,
              padding: EdgeInsets.symmetric(horizontal: spacing.sp2),
              tabs: [
                for (final tab in tabs) Tab(text: _labelFor(tab, l10n)),
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
}
