import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:craftsky_app/profile/widgets/profile_actions.dart';
import 'package:craftsky_app/profile/widgets/profile_meta_section.dart';
import 'package:craftsky_app/profile/widgets/profile_page_error.dart';
import 'package:craftsky_app/profile/widgets/profile_sliver_app_bar.dart';
import 'package:craftsky_app/profile/widgets/profile_tab_bar.dart';
import 'package:craftsky_app/profile/widgets/profile_tabs/profile_about_tab.dart';
import 'package:craftsky_app/profile/widgets/profile_tabs/profile_empty_tab.dart';
import 'package:craftsky_app/profile/widgets/profile_tabs/profile_posts_tab.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Unified profile screen. Used by the bottom-nav `Profile` branch
/// (no [handle], resolves to the signed-in user) and by
/// `/profile/:handle` deep links. Self-vs-visitor differs only in the
/// action row — the rest of the chrome is shared.
class ProfilePage extends ConsumerWidget {
  const ProfilePage({this.handle, super.key});

  /// Handle of the profile to render. `null` resolves to the signed-in
  /// user from `authSessionProvider`.
  final String? handle;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final auth = ref.watch(authSessionProvider).value;
    final myHandle = switch (auth) {
      SignedIn(:final handle) => handle,
      _ => null,
    };

    final targetHandle = handle ?? myHandle;
    if (targetHandle == null) {
      // Either auth is still loading or a visitor route somehow
      // landed here without a handle. Both are transient — show a
      // neutral progress state and let the router redirect resolve.
      return const Scaffold(body: Center(child: CircularProgressIndicator()));
    }

    return _ProfileScaffold(
      handle: targetHandle,
      isOwnProfile: targetHandle == myHandle,
    );
  }
}

class _ProfileScaffold extends ConsumerWidget {
  const _ProfileScaffold({required this.handle, required this.isOwnProfile});

  final String handle;
  final bool isOwnProfile;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final profileAsync = ref.watch(userProfileProvider(handle));
    final swatches = Theme.of(context).extension<BrandSwatchTheme>()!;
    // Per-user banner colour will eventually come from the profile
    // record. For now, every banner is clay so the layout is stable
    // across users.
    final bannerColor = swatches.clay;

    return Scaffold(
      body: switch (profileAsync) {
        AsyncValue(:final value?) => _ProfileBody(
          profile: value,
          handle: handle,
          isOwnProfile: isOwnProfile,
          bannerColor: bannerColor,
        ),
        AsyncError(:final error) => ProfilePageError(
          error: error,
          onRetry: () => ref.invalidate(userProfileProvider(handle)),
        ),
        _ => const Center(child: CircularProgressIndicator()),
      },
    );
  }
}

class _ProfileBody extends ConsumerWidget {
  const _ProfileBody({
    required this.profile,
    required this.handle,
    required this.isOwnProfile,
    required this.bannerColor,
  });

  final Profile profile;
  final String handle;
  final bool isOwnProfile;
  final Color bannerColor;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return DefaultTabController(
      length: ProfileTab.values.length,
      child: _ProfileScrollView(
        profile: profile,
        bannerColor: bannerColor,
        actions: _actionsFor(context, ref),
      ),
    );
  }

  ProfileActionSet _actionsFor(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    if (isOwnProfile) {
      return SelfProfileActionSet(
        onEdit: () => const EditProfileRoute().go(context),
        onSettings: () => const SettingsRoute().go(context),
      );
    }

    // Visitor follow state isn't wired through yet — placeholder
    // false and a no-op so the button still demos correctly.
    return VisitorProfileActionSet(
      isFollowing: false,
      onFollowToggle: () {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(l10n.profileFollowComingSoon)),
        );
      },
      onShare: () {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(l10n.profileShareComingSoon)),
        );
      },
    );
  }
}

/// The profile screen's scroll structure. A [NestedScrollView]
/// coordinates the outer collapsing header (banner / avatar / meta /
/// pinned tab bar) with each tab's own inner [CustomScrollView]. Each
/// tab gets its own [PageStorageKey] so its scroll position is
/// preserved across tab switches — switching tabs no longer jumps to
/// the top, and the outer [ProfileSliverAppBar] keeps whatever
/// collapse state the user established.
class _ProfileScrollView extends StatelessWidget {
  const _ProfileScrollView({
    required this.profile,
    required this.bannerColor,
    required this.actions,
  });

  final Profile profile;
  final Color bannerColor;
  final ProfileActionSet actions;

  @override
  Widget build(BuildContext context) {
    return NestedScrollView(
      headerSliverBuilder: (context, innerBoxIsScrolled) => [
        ProfileSliverAppBar(
          handle: profile.handle,
          displayName: profile.displayName,
          bannerColor: bannerColor,
          avatarUrl: profile.avatar,
          bannerUrl: profile.banner,
          bannerChipLabel: 'Jacket weather',
          actions: actions,
        ),
        SliverToBoxAdapter(child: ProfileMetaSection(profile: profile)),
        const SliverPersistentHeader(
          pinned: true,
          delegate: ProfileTabBarDelegate(),
        ),
      ],
      body: TabBarView(
        children: [
          for (final tab in ProfileTab.values)
            _ProfileTabScrollView(tab: tab, profile: profile),
        ],
      ),
    );
  }
}

/// Inner scrollable for one tab. Wraps the tab's slivers in a
/// [CustomScrollView] keyed by tab name so [PageStorage] retains the
/// scroll offset when the user swipes back to it.
class _ProfileTabScrollView extends StatelessWidget {
  const _ProfileTabScrollView({required this.tab, required this.profile});

  final ProfileTab tab;
  final Profile profile;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return CustomScrollView(
      key: PageStorageKey<String>('profile_tab_${tab.name}'),
      slivers: [_slivertForTab(tab, profile, l10n)],
    );
  }

  Widget _slivertForTab(
    ProfileTab tab,
    Profile profile,
    AppLocalizations l10n,
  ) {
    return switch (tab) {
      ProfileTab.posts => ProfilePostsTab(handle: profile.handle),
      ProfileTab.projects =>
        ProfileEmptyTab(message: l10n.profileEmptyProjects),
      ProfileTab.saved => ProfileEmptyTab(message: l10n.profileEmptySaved),
      ProfileTab.reposts => ProfileEmptyTab(message: l10n.profileEmptyReposts),
      ProfileTab.about => ProfileAboutTab(profile: profile),
    };
  }
}
