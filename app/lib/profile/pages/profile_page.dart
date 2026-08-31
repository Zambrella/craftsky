import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/business/providers/profile_business_events_provider.dart';
import 'package:craftsky_app/feed/widgets/post_image_gallery.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/moderation/widgets/report_flow.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/models/profile_relationship.dart';
import 'package:craftsky_app/profile/pages/edit_profile_dialog.dart';
import 'package:craftsky_app/profile/providers/profile_relationship_provider.dart';
import 'package:craftsky_app/profile/providers/toggle_follow_profile_provider.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:craftsky_app/profile/widgets/profile_actions.dart';
import 'package:craftsky_app/profile/widgets/profile_customisation_theme.dart';
import 'package:craftsky_app/profile/widgets/profile_meta_section.dart';
import 'package:craftsky_app/profile/widgets/profile_sliver_app_bar.dart';
import 'package:craftsky_app/profile/widgets/profile_tab_bar.dart';
import 'package:craftsky_app/profile/widgets/profile_tabs/profile_about_tab.dart';
import 'package:craftsky_app/profile/widgets/profile_tabs/profile_comments_tab.dart';
import 'package:craftsky_app/profile/widgets/profile_tabs/profile_empty_tab.dart';
import 'package:craftsky_app/profile/widgets/profile_tabs/profile_events_tab.dart';
import 'package:craftsky_app/profile/widgets/profile_tabs/profile_posts_tab.dart';
import 'package:craftsky_app/profile/widgets/profile_tabs/profile_products_tab.dart';
import 'package:craftsky_app/profile/widgets/profile_tabs/profile_projects_tab.dart';
import 'package:craftsky_app/router/app_shell_drawer.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/errors/notification_destination_error.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/shared/widgets/notification_destination_error_state.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Unified profile screen. Used by the bottom-nav `Profile` branch
/// (no [handle], resolves to the signed-in user) and by
/// `/profile/:handle` deep links. Self-vs-visitor differs only in the
/// action row — the rest of the chrome is shared.
class ProfilePage extends ConsumerWidget {
  const ProfilePage({
    this.handle,
    super.key,
  });

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
    final viewerAccount = switch (auth) {
      SignedIn(:final did) => AccountKey(did.toString()),
      _ => null,
    };

    final targetHandle = handle ?? myHandle;
    if (targetHandle == null) {
      // Either auth is still loading or a visitor route somehow
      // landed here without a handle. Both are transient — show a
      // neutral progress state and let the router redirect resolve.
      return const Scaffold(
        body: Center(child: StitchProgressIndicator()),
      );
    }

    return _ProfileScaffold(
      handle: targetHandle,
      isOwnProfile: targetHandle == myHandle,
      viewerAccount: viewerAccount,
    );
  }
}

class _ProfileScaffold extends ConsumerWidget {
  const _ProfileScaffold({
    required this.handle,
    required this.isOwnProfile,
    required this.viewerAccount,
  });

  final String handle;
  final bool isOwnProfile;
  final AccountKey? viewerAccount;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final profileAsync = ref.watch(userProfileProvider(handle));
    if (isOwnProfile) {
      ref.listen(userProfileProvider(handle), (previous, next) {
        final profile = next.value;
        final lease = ref
            .read(sessionRegistryProvider)
            .value
            ?.activeLease
            ?.session;
        if (profile == null || lease == null) return;
        unawaited(
          ref
              .read(sessionRegistryProvider.notifier)
              .updateCachedIdentity(
                lease,
                displayName: profile.displayName,
                avatarUrl: profile.avatar,
                customisation: profile.customisation,
              ),
        );
      });
    }
    final destinationError = profileAsync.error;
    if (destinationError != null &&
        classifyNotificationDestinationError(destinationError) ==
            NotificationDestinationErrorKind.permanentUnavailable) {
      return Scaffold(
        appBar: _profileNavigationAppBar(context),
        body: _destinationErrorState(context, ref, destinationError),
      );
    }

    return switch (profileAsync) {
      AsyncValue(:final value?) => Scaffold(
        body: switch (destinationError) {
          final error? => Column(
            children: [
              _destinationErrorState(context, ref, error),
              Expanded(
                child: _ProfileBody(
                  profile: value,
                  isOwnProfile: isOwnProfile,
                  viewerAccount: viewerAccount,
                ),
              ),
            ],
          ),
          null => _ProfileBody(
            profile: value,
            isOwnProfile: isOwnProfile,
            viewerAccount: viewerAccount,
          ),
        },
      ),
      AsyncError(:final error) => Scaffold(
        appBar: _profileNavigationAppBar(context),
        body: _destinationErrorState(context, ref, error),
      ),
      _ => Scaffold(
        appBar: _profileNavigationAppBar(context),
        body: const Center(child: StitchProgressIndicator()),
      ),
    };
  }

  Widget _destinationErrorState(
    BuildContext context,
    WidgetRef ref,
    Object error,
  ) => NotificationDestinationErrorState(
    error: error,
    onRetry: () => ref.invalidate(userProfileProvider(handle)),
    onBack: () {
      final navigator = Navigator.of(context);
      if (navigator.canPop()) {
        navigator.pop();
      } else {
        const FeedRoute().go(context);
      }
    },
    onViewNotifications: () => const NotificationsRoute().go(context),
  );
}

PreferredSizeWidget? _profileNavigationAppBar(BuildContext context) {
  if (AppShellDrawerScope.maybeOf(context) != null) {
    return AppBar(leading: const AppShellDrawerButton());
  }
  return Navigator.of(context).canPop() ? AppBar() : null;
}

class _ProfileBody extends ConsumerWidget {
  const _ProfileBody({
    required this.profile,
    required this.isOwnProfile,
    required this.viewerAccount,
  });

  final Profile profile;
  final bool isOwnProfile;
  final AccountKey? viewerAccount;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    ref.listen(toggleFollowProfileProvider, (previous, next) {
      switch ((previous, next)) {
        case (AsyncLoading(), AsyncError()):
          context.showError(l10n.profileFollowToggleError);
          ref.read(toggleFollowProfileProvider.notifier).reset();
        case _:
          break;
      }
    });

    final serverRelationship = ProfileRelationship.fromProfileFlags(
      muted: profile.muted,
      blocking: profile.blocking,
      blockedBy: profile.blockedBy,
    );
    final account = viewerAccount;
    final provider = account == null || isOwnProfile
        ? null
        : profileRelationshipProvider(account, profile.did.toString());
    final cached = provider == null ? null : ref.watch(provider);
    if (provider != null && !(cached?.initialized ?? false)) {
      unawaited(
        Future<void>.microtask(
          () => ref.read(provider.notifier).seed(serverRelationship),
        ),
      );
    }
    final relationship = cached?.initialized ?? false
        ? cached!
        : serverRelationship;
    final actions = _actionsFor(context, ref, relationship);

    if (relationship.hasBlock) {
      return _BlockedProfileView(
        profile: profile,
        actions: actions,
        relationship: relationship,
      );
    }
    return _ProfileTabbedBody(
      key: ValueKey(profile.did),
      profile: profile,
      actions: actions,
      isOwnProfile: isOwnProfile,
      relationship: relationship,
      viewerAccount: viewerAccount,
    );
  }

  ProfileActionSet _actionsFor(
    BuildContext context,
    WidgetRef ref,
    ProfileRelationship relationship,
  ) {
    final l10n = AppLocalizations.of(context);
    if (isOwnProfile) {
      return SelfProfileActionSet(
        onEdit: () => showEditProfileDialog(context),
        onSettings: () => const SettingsRoute().go(context),
      );
    }

    final toggleState = ref.watch(toggleFollowProfileProvider);
    return VisitorProfileActionSet(
      isFollowing: profile.viewerIsFollowing,
      isBusy: toggleState.isLoading || relationship.pendingAction != null,
      isMuted: relationship.muted,
      isBlocking: relationship.blocking,
      canFollow: !relationship.hasBlock,
      canToggleMute: !relationship.hasBlock || relationship.muted,
      onFollowToggle: () {
        unawaited(
          ref
              .read(toggleFollowProfileProvider.notifier)
              .toggle(
                cacheKey: profile.handle.toString(),
                profile: profile,
              ),
        );
      },
      onShare: () => context.showInfo(l10n.profileShareComingSoon),
      onReport: () => showProfileReportSheet(context, ref, profile.handle),
      onMuteToggle: () => unawaited(
        _mutateRelationship(
          context,
          ref,
          relationship.muted
              ? ProfileRelationshipAction.unmute
              : ProfileRelationshipAction.mute,
        ),
      ),
      onBlockToggle: () => unawaited(
        _confirmAndMutateBlock(context, ref, relationship.blocking),
      ),
    );
  }

  Future<void> _confirmAndMutateBlock(
    BuildContext context,
    WidgetRef ref,
    bool isBlocking,
  ) async {
    final l10n = AppLocalizations.of(context);
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: Text(
          isBlocking
              ? l10n.profileUnblockConfirmTitle
              : l10n.profileBlockConfirmTitle,
        ),
        content: Text(
          isBlocking
              ? l10n.profileUnblockConfirmBody
              : l10n.profileBlockConfirmBody,
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(false),
            child: Text(l10n.actionCancel),
          ),
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(true),
            child: Text(
              isBlocking ? l10n.profileUnblockAction : l10n.profileBlockAction,
            ),
          ),
        ],
      ),
    );
    if (confirmed != true || !context.mounted) return;
    await _mutateRelationship(
      context,
      ref,
      isBlocking
          ? ProfileRelationshipAction.unblock
          : ProfileRelationshipAction.block,
    );
  }

  Future<void> _mutateRelationship(
    BuildContext context,
    WidgetRef ref,
    ProfileRelationshipAction action,
  ) async {
    final account = viewerAccount;
    if (account == null) return;
    final provider = profileRelationshipProvider(
      account,
      profile.did.toString(),
    );
    await ref.read(provider.notifier).mutate(action);
    if (!context.mounted) return;
    final l10n = AppLocalizations.of(context);
    final result = ref.read(provider);
    if (result.lastError != null) {
      context.showError(l10n.profileRelationshipError);
      return;
    }
    context.showInfo(switch (action) {
      ProfileRelationshipAction.mute => l10n.profileMuteSuccess,
      ProfileRelationshipAction.unmute => l10n.profileUnmuteSuccess,
      ProfileRelationshipAction.block => l10n.profileBlockSuccess,
      ProfileRelationshipAction.unblock => l10n.profileUnblockSuccess,
    });
  }
}

class _ProfileTabbedBody extends StatefulWidget {
  const _ProfileTabbedBody({
    required this.profile,
    required this.actions,
    required this.isOwnProfile,
    required this.relationship,
    required this.viewerAccount,
    super.key,
  });

  final Profile profile;
  final ProfileActionSet actions;
  final bool isOwnProfile;
  final ProfileRelationship relationship;
  final AccountKey? viewerAccount;

  @override
  State<_ProfileTabbedBody> createState() => _ProfileTabbedBodyState();
}

class _ProfileTabbedBodyState extends State<_ProfileTabbedBody> {
  ProfileTab _selected = ProfileTab.projects;

  @override
  void didUpdateWidget(covariant _ProfileTabbedBody oldWidget) {
    super.didUpdateWidget(oldWidget);
    _selected = ProfileTabPolicy.selectionAfterChange(
      selected: _selected,
      tabs: _tabsFor(widget.profile),
    );
  }

  @override
  Widget build(BuildContext context) {
    final tabs = _tabsFor(widget.profile);
    return _ProfileTabControllerBody(
      key: ValueKey(widget.profile.accountType),
      profile: widget.profile,
      actions: widget.actions,
      isOwnProfile: widget.isOwnProfile,
      relationship: widget.relationship,
      viewerAccount: widget.viewerAccount,
      tabs: tabs,
      initialSelection: _selected,
      onSelectionChanged: (tab) => _selected = tab,
    );
  }

  List<ProfileTab> _tabsFor(Profile profile) => ProfileTabPolicy.forProfile(
    accountType: profile.accountType,
    isBlocked: false,
  );
}

class _ProfileTabControllerBody extends StatefulWidget {
  const _ProfileTabControllerBody({
    required this.profile,
    required this.actions,
    required this.isOwnProfile,
    required this.relationship,
    required this.viewerAccount,
    required this.tabs,
    required this.initialSelection,
    required this.onSelectionChanged,
    super.key,
  });

  final Profile profile;
  final ProfileActionSet actions;
  final bool isOwnProfile;
  final ProfileRelationship relationship;
  final AccountKey? viewerAccount;
  final List<ProfileTab> tabs;
  final ProfileTab initialSelection;
  final ValueChanged<ProfileTab> onSelectionChanged;

  @override
  State<_ProfileTabControllerBody> createState() =>
      _ProfileTabControllerBodyState();
}

class _ProfileTabControllerBodyState extends State<_ProfileTabControllerBody>
    with SingleTickerProviderStateMixin {
  late final TabController _controller;

  @override
  void initState() {
    super.initState();
    _controller = TabController(
      length: widget.tabs.length,
      initialIndex: widget.tabs.indexOf(widget.initialSelection),
      vsync: this,
    )..addListener(_trackSelection);
  }

  @override
  void dispose() {
    _controller
      ..removeListener(_trackSelection)
      ..dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => _ProfileScrollView(
    profile: widget.profile,
    actions: widget.actions,
    isOwnProfile: widget.isOwnProfile,
    relationship: widget.relationship,
    viewerAccount: widget.viewerAccount,
    tabs: widget.tabs,
    controller: _controller,
  );

  void _trackSelection() {
    if (_controller.indexIsChanging) return;
    widget.onSelectionChanged(widget.tabs[_controller.index]);
  }
}

/// The profile screen's scroll structure. A [NestedScrollView]
/// coordinates the outer collapsing header (illustration / avatar / meta /
/// pinned tab bar) with each tab's own inner [CustomScrollView]. Each
/// tab gets its own [PageStorageKey] so its scroll position is
/// preserved across tab switches — switching tabs no longer jumps to
/// the top, and the outer [ProfileSliverAppBar] keeps whatever
/// collapse state the user established.
class _ProfileScrollView extends StatelessWidget {
  const _ProfileScrollView({
    required this.profile,
    required this.actions,
    required this.isOwnProfile,
    required this.relationship,
    required this.tabs,
    required this.controller,
    required this.viewerAccount,
  });

  final Profile profile;
  final ProfileActionSet actions;
  final bool isOwnProfile;
  final ProfileRelationship relationship;
  final List<ProfileTab> tabs;
  final TabController controller;
  final AccountKey? viewerAccount;

  @override
  Widget build(BuildContext context) {
    return NestedScrollView(
      headerSliverBuilder: (context, innerBoxIsScrolled) => [
        ProfileCustomisationTheme(
          customisation: profile.customisation,
          child: ProfileSliverAppBar(
            handle: profile.handle,
            crafts: profile.crafts,
            displayName: profile.displayName,
            avatarUrl: profile.avatar,
            customisation: profile.customisation,
            actions: actions,
            onAvatarTap: profile.avatar == null
                ? null
                : () => _openProfileImage(
                    context,
                    url: profile.avatar!,
                    alt: _profileImageAlt(profile, 'profile picture'),
                  ),
          ),
        ),
        SliverToBoxAdapter(
          child: ProfileCustomisationTheme(
            customisation: profile.customisation,
            child: ProfileMetaSection(
              profile: profile,
              isOwnProfile: isOwnProfile,
              actions: actions,
            ),
          ),
        ),
        if (relationship.kind != ProfileRelationshipKind.none)
          SliverToBoxAdapter(
            child: ProfileCustomisationTheme(
              customisation: profile.customisation,
              child: _RelationshipAnnotation(relationship: relationship),
            ),
          ),
        SliverPersistentHeader(
          pinned: true,
          delegate: ProfileTabBarDelegate(tabs: tabs, controller: controller),
        ),
      ],
      body: TabBarView(
        controller: controller,
        children: [
          for (final tab in tabs)
            _ProfileTabScrollView(
              tab: tab,
              profile: profile,
              isOwnProfile: isOwnProfile,
              viewerAccount: viewerAccount,
            ),
        ],
      ),
    );
  }

  void _openProfileImage(
    BuildContext context, {
    required String url,
    required String alt,
  }) {
    unawaited(
      showImageGallery(
        context,
        images: [GalleryImage(alt: alt, thumb: url, fullsize: url)],
      ),
    );
  }

  String _profileImageAlt(Profile profile, String imageLabel) {
    final name = (profile.displayName?.isNotEmpty ?? false)
        ? profile.displayName!
        : '@${profile.handle}';
    return '$name $imageLabel';
  }
}

class _BlockedProfileView extends StatelessWidget {
  const _BlockedProfileView({
    required this.profile,
    required this.actions,
    required this.relationship,
  });

  final Profile profile;
  final ProfileActionSet actions;
  final ProfileRelationship relationship;

  @override
  Widget build(BuildContext context) => CustomScrollView(
    slivers: [
      ProfileCustomisationTheme(
        customisation: profile.customisation,
        child: ProfileSliverAppBar(
          handle: profile.handle,
          displayName: profile.displayName,
          avatarUrl: profile.avatar,
          customisation: profile.customisation,
          actions: actions,
        ),
      ),
      SliverToBoxAdapter(
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: ProfileActionSection(actions: actions),
        ),
      ),
      SliverToBoxAdapter(
        child: _RelationshipAnnotation(relationship: relationship),
      ),
    ],
  );
}

class _RelationshipAnnotation extends StatelessWidget {
  const _RelationshipAnnotation({required this.relationship});

  final ProfileRelationship relationship;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final label = switch (relationship.kind) {
      ProfileRelationshipKind.none => '',
      ProfileRelationshipKind.muted => l10n.profileMuteAnnotation,
      ProfileRelationshipKind.blocking => l10n.profileBlockingAnnotation,
      ProfileRelationshipKind.blockedBy => l10n.profileBlockedByAnnotation,
      ProfileRelationshipKind.mutualBlock => l10n.profileMutualBlockAnnotation,
    };
    return Semantics(
      liveRegion: true,
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Text(
          label,
          style: Theme.of(context).textTheme.titleMedium,
        ),
      ),
    );
  }
}

/// Inner scrollable for one tab. Wraps the tab's slivers in a
/// [CustomScrollView] keyed by tab name so [PageStorage] retains the
/// scroll offset when the user swipes back to it.
class _ProfileTabScrollView extends StatelessWidget {
  const _ProfileTabScrollView({
    required this.tab,
    required this.profile,
    required this.isOwnProfile,
    required this.viewerAccount,
  });

  final ProfileTab tab;
  final Profile profile;
  final bool isOwnProfile;
  final AccountKey? viewerAccount;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return CustomScrollView(
      key: PageStorageKey<String>(tab.storageKey),
      slivers: [_sliverForTab(context, tab, profile, l10n)],
    );
  }

  Widget _sliverForTab(
    BuildContext context,
    ProfileTab tab,
    Profile profile,
    AppLocalizations l10n,
  ) {
    return switch (tab) {
      ProfileTab.posts => ProfilePostsTab(
        handle: profile.handle,
        isOwnProfile: isOwnProfile,
      ),
      ProfileTab.comments => ProfileCommentsTab(
        handle: profile.handle,
        isOwnProfile: isOwnProfile,
      ),
      ProfileTab.projects => ProfileProjectsTab(
        handle: profile.handle,
        isOwnProfile: isOwnProfile,
      ),
      ProfileTab.reposts => ProfileEmptyTab(message: l10n.profileEmptyReposts),
      ProfileTab.products => ProfileProductsTab(
        products: profile.business?.products ?? const [],
        isOwnProfile: isOwnProfile,
        onManage: isOwnProfile
            ? () => const BusinessProductsRoute().go(context)
            : null,
      ),
      ProfileTab.upcomingEvents => switch (viewerAccount) {
        final account? => ProfileEventsTab(
          target: ProfileBusinessEventsTarget(
            account: account,
            owner: AtIdentifier.parse(profile.did.toString()),
          ),
          isOwnProfile: isOwnProfile,
          onOpen: (did, rkey) => BusinessEventRoute(
            did: did.toString(),
            rkey: rkey.toString(),
          ).push<void>(context),
          onManage: isOwnProfile
              ? () => const BusinessEventsRoute().go(context)
              : null,
        ),
        null => const SliverFillRemaining(
          hasScrollBody: false,
          child: Center(child: StitchProgressIndicator()),
        ),
      },
      ProfileTab.about => ProfileAboutTab(profile: profile),
    };
  }
}
