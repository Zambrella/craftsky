import 'dart:async';

import 'package:craftsky_app/app_dependencies.dart';
import 'package:craftsky_app/auth/models/account_switcher_state.dart';
import 'package:craftsky_app/auth/providers/account_activation_coordinator.dart';
import 'package:craftsky_app/auth/providers/account_boundary_provider.dart';
import 'package:craftsky_app/auth/providers/active_account_identity_provider.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/auth/providers/unsaved_work_guard_provider.dart';
import 'package:craftsky_app/auth/widgets/account_avatar.dart';
import 'package:craftsky_app/auth/widgets/account_switcher_launcher.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/notifications/models/notification_badge.dart';
import 'package:craftsky_app/notifications/providers/notification_new_count_provider.dart';
import 'package:craftsky_app/profile/models/profile_customisation.dart';
import 'package:craftsky_app/router/app_shell_drawer.dart';
import 'package:craftsky_app/router/route_locations.dart';
import 'package:craftsky_app/shared/link/external_link.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/theme/craftsky_card.dart';
import 'package:craftsky_app/theme/craftsky_context_menu.dart';
import 'package:craftsky_app/theme/form_factor.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

export 'package:craftsky_app/router/app_shell_drawer.dart';

/// Paired icon + label spec for a shell branch destination.
class _DestinationSpec {
  const _DestinationSpec({
    required this.icon,
    required this.selectedIcon,
  });

  final IconData icon;
  final IconData selectedIcon;
}

const _primaryDestinations = <_DestinationSpec>[
  _DestinationSpec(
    icon: Icons.home_outlined,
    selectedIcon: Icons.home,
  ),
  _DestinationSpec(
    icon: Icons.grid_view_outlined,
    selectedIcon: Icons.grid_view,
  ),
  _DestinationSpec(
    icon: Icons.search_outlined,
    selectedIcon: Icons.search,
  ),
  _DestinationSpec(
    icon: Icons.notifications_outlined,
    selectedIcon: Icons.notifications,
  ),
  _DestinationSpec(
    icon: Icons.person_outline,
    selectedIcon: Icons.person,
  ),
];

const _secondaryDestinations = <_DestinationSpec>[
  _DestinationSpec(
    icon: Icons.bookmarks_outlined,
    selectedIcon: Icons.bookmarks,
  ),
  _DestinationSpec(
    icon: Icons.schedule_outlined,
    selectedIcon: Icons.schedule,
  ),
  _DestinationSpec(
    icon: Icons.edit_note_outlined,
    selectedIcon: Icons.edit_note,
  ),
  _DestinationSpec(
    icon: Icons.settings_outlined,
    selectedIcon: Icons.settings,
  ),
];

const List<_DestinationSpec> _menuDestinations = [
  ..._primaryDestinations,
  ..._secondaryDestinations,
];

const _compactDrawerEdgeDragWidth = 96.0;
const _largeScreenContentMaxWidth = 800.0;
const _desktopSidebarSize = 300.0;
const _utilityLinkHeight = 40.0;
const _profileRailLabelWidth = 168.0;

RoundedRectangleBorder _navigationDrawerShape(ThemeData theme) {
  final radii = theme.extension<RadiusTheme>()!;
  return RoundedRectangleBorder(
    borderRadius: BorderRadiusDirectional.horizontal(
      end: Radius.circular(radii.r4),
    ),
    side: BorderSide(color: theme.colorScheme.onSurface, width: 1.5),
  );
}

BorderDirectional _navigationRailBorder(ThemeData theme) => BorderDirectional(
  end: BorderSide(color: theme.colorScheme.onSurface, width: 1.5),
);

/// Keeps signed-in detail routes beside the navigation rail on large screens.
/// Compact routes remain full-screen because their navigator is returned
/// unchanged.
class AuthenticatedShell extends StatelessWidget {
  const AuthenticatedShell({required this.child, super.key});

  final Widget child;

  @override
  Widget build(BuildContext context) {
    if (!FormFactorWidget.of(context).isLarge) return child;
    final router = GoRouter.of(context);
    return ListenableBuilder(
      listenable: router.routerDelegate,
      builder: (context, _) => _AuthenticatedShellNavigationScope(
        child: _LargeShellNavigationFrame(
          selectedIndex: _destinationIndexForLocation(
            router.state.matchedLocation,
          ),
          onDestinationSelected: (index) => _goDestination(context, index),
          child: child,
        ),
      ),
    );
  }

  void _goDestination(BuildContext context, int index) {
    final location = switch (index) {
      0 => RouteLocations.feed,
      1 => RouteLocations.projects,
      2 => RouteLocations.search,
      3 => RouteLocations.notifications,
      4 => RouteLocations.profile,
      5 => RouteLocations.savedPosts,
      6 => RouteLocations.scheduledPosts,
      7 => RouteLocations.drafts,
      8 => RouteLocations.settings,
      _ => throw RangeError.index(index, _menuDestinations),
    };
    context.go(location);
  }
}

class _AuthenticatedShellNavigationScope extends InheritedWidget {
  const _AuthenticatedShellNavigationScope({required super.child});

  static bool ownsLargeRail(BuildContext context) =>
      context
          .dependOnInheritedWidgetOfExactType<
            _AuthenticatedShellNavigationScope
          >() !=
      null;

  @override
  bool updateShouldNotify(_AuthenticatedShellNavigationScope oldWidget) =>
      false;
}

class _LargeShellNavigationFrame extends ConsumerStatefulWidget {
  const _LargeShellNavigationFrame({
    required this.selectedIndex,
    required this.onDestinationSelected,
    required this.child,
    this.linkLauncher = launchExternalLink,
    this.buildVersionLabel,
  });

  final int selectedIndex;
  final ValueChanged<int> onDestinationSelected;
  final Widget child;
  final ExternalLinkLauncher linkLauncher;
  final String? buildVersionLabel;

  @override
  ConsumerState<_LargeShellNavigationFrame> createState() =>
      _LargeShellNavigationFrameState();
}

class _LargeShellNavigationFrameState
    extends ConsumerState<_LargeShellNavigationFrame> {
  late final AccountActivationCoordinator _activation;
  final GlobalKey _profileAnchorKey = GlobalKey();
  final FocusNode _profileSwitcherFocusNode = FocusNode(
    debugLabel: 'Large navigation Profile account switcher',
  );

  @override
  void initState() {
    super.initState();
    _activation = AccountActivationCoordinator(
      readRegistry: () => ref.read(sessionRegistryProvider).requireValue,
      commitActivation: ref.read(sessionRegistryProvider.notifier).activate,
      invalidateAccountState: ref.read(accountStateInvalidatorProvider),
      resetToHome: () async => context.go(RouteLocations.home),
      confirmLeave: ref.read(unsavedWorkGuardProvider).confirmLeave,
    );
  }

  @override
  void dispose() {
    _profileSwitcherFocusNode.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final formFactor = FormFactorWidget.of(context);
    final registry = ref.watch(sessionRegistryProvider).value;
    final activeAccount = registry?.activeLease?.session.account;
    final switcherState = registry == null
        ? null
        : AccountSwitcherState.fromRegistry(registry);
    final packageInfo =
        widget.buildVersionLabel == null && ref.exists(appDependenciesProvider)
        ? ref.watch(packageInfoProvider)
        : null;
    final buildVersionLabel =
        widget.buildVersionLabel ??
        (packageInfo == null
            ? null
            : AppLocalizations.of(context).navigationBuildVersion(
                packageInfo.version,
                packageInfo.buildNumber,
              ));
    final notificationBadge = NotificationBadge.fromCount(
      activeAccount == null
          ? ref.watch(notificationNewCountProvider).value ?? 0
          : ref
                    .watch(accountNotificationNewCountProvider(activeAccount))
                    .value ??
                0,
    );
    final textDirection = Directionality.of(context);
    return Scaffold(
      body: Row(
        textDirection: switch (textDirection) {
          TextDirection.ltr => TextDirection.rtl,
          TextDirection.rtl => TextDirection.ltr,
        },
        children: [
          Expanded(
            child: Directionality(
              textDirection: textDirection,
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Expanded(
                    child: Align(
                      alignment: Alignment.topCenter,
                      child: ConstrainedBox(
                        key: const Key('large-shell-content'),
                        constraints: const BoxConstraints(
                          maxWidth: _largeScreenContentMaxWidth,
                        ),
                        child: SizedBox.expand(child: widget.child),
                      ),
                    ),
                  ),
                  if (formFactor == FormFactor.desktop) ...[
                    SizedBox(
                      key: const Key('desktop-sidebar-divider'),
                      width: 1.5,
                      child: ColoredBox(
                        color: Theme.of(context).colorScheme.onSurface,
                      ),
                    ),
                    const _DesktopSidebarPlaceholder(),
                  ],
                ],
              ),
            ),
          ),
          Directionality(
            textDirection: textDirection,
            child: _ShellNavigationRail(
              selectedIndex: widget.selectedIndex,
              onDestinationSelected: widget.onDestinationSelected,
              notificationBadge: notificationBadge,
              buildVersionLabel: buildVersionLabel,
              profileAnchorKey: _profileAnchorKey,
              profileFocusNode: _profileSwitcherFocusNode,
              onOpenAccountSwitcher: switcherState == null
                  ? null
                  : () => _showLargeSwitcher(switcherState),
              onOpenTerms: () => _openExternalLink(
                Uri.parse('https://craftsky.social/terms'),
              ),
              onOpenPrivacy: () => _openExternalLink(
                Uri.parse('https://craftsky.social/privacy'),
              ),
            ),
          ),
        ],
      ),
    );
  }

  Future<void> _openExternalLink(Uri uri) async {
    var opened = false;
    try {
      opened = await widget.linkLauncher(uri);
    } on Object {
      // Platform link handlers can fail by returning false or by throwing.
    }
    if (!mounted || opened) return;
    context.showError(AppLocalizations.of(context).navigationLinkOpenError);
  }

  Future<void> _showLargeSwitcher(AccountSwitcherState state) async {
    final box =
        _profileAnchorKey.currentContext?.findRenderObject() as RenderBox?;
    final overlay =
        Overlay.of(context).context.findRenderObject() as RenderBox?;
    if (box == null || overlay == null) return;
    final origin = box.localToGlobal(Offset.zero, ancestor: overlay);
    final position = RelativeRect.fromLTRB(
      origin.dx + box.size.width,
      origin.dy,
      overlay.size.width - origin.dx - box.size.width,
      overlay.size.height - origin.dy - box.size.height,
    );
    await showCraftskyContextPopover(
      context,
      position: position,
      estimatedHeight: _estimateAccountSwitcherHeight(state),
      child: SizedBox(
        width: 320,
        child: LiveAccountSwitcherContent(
          fallbackState: state,
          onSelect: _activation.activate,
          onAddAccount: () {
            Navigator.pop(context);
            unawaited(context.push(RouteLocations.addAccount));
          },
        ),
      ),
    );
  }

  double _estimateAccountSwitcherHeight(AccountSwitcherState state) {
    const verticalPadding = 16.0;
    const singleLineTileHeight = 56.0;
    const twoLineTileHeight = 72.0;
    const dividerHeight = 16.0;

    return verticalPadding +
        state.rows.fold<double>(
          0,
          (height, row) =>
              height +
              (row.displayLabel == row.handle
                  ? singleLineTileHeight
                  : twoLineTileHeight),
        ) +
        dividerHeight +
        (state.canAddAccount ? singleLineTileHeight : twoLineTileHeight);
  }
}

class _DesktopSidebarPlaceholder extends StatelessWidget {
  const _DesktopSidebarPlaceholder();

  @override
  Widget build(BuildContext context) {
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    return SizedBox(
      width: _desktopSidebarSize + (spacing.sp5 * 2),
      child: Align(
        alignment: AlignmentDirectional.topCenter,
        child: Padding(
          padding: EdgeInsets.only(top: spacing.sp5),
          child: const SizedBox.square(
            key: Key('desktop-sidebar-placeholder'),
            dimension: _desktopSidebarSize,
            child: CraftskyCard(child: SizedBox.expand()),
          ),
        ),
      ),
    );
  }
}

class AppShell extends ConsumerStatefulWidget {
  const AppShell({
    required this.navigationShell,
    this.linkLauncher = launchExternalLink,
    this.buildVersionLabel,
    super.key,
  });

  final StatefulNavigationShell navigationShell;
  final ExternalLinkLauncher linkLauncher;
  final String? buildVersionLabel;

  @override
  ConsumerState<AppShell> createState() => _AppShellState();
}

class _AppShellState extends ConsumerState<AppShell> {
  late final AccountActivationCoordinator _activation;
  final GlobalKey<ScaffoldState> _compactScaffoldKey = GlobalKey();
  final FocusNode _drawerProfileSwitcherFocusNode = FocusNode(
    debugLabel: 'Drawer Profile account switcher',
  );
  final FocusNode _drawerButtonFocusNode = FocusNode(
    debugLabel: 'Navigation drawer button',
  );
  final FocusNode _profileSwitcherFocusNode = FocusNode(
    debugLabel: 'Profile account switcher',
  );
  bool _isDrawerOpen = false;

  @override
  void initState() {
    super.initState();
    _activation = AccountActivationCoordinator(
      readRegistry: () => ref.read(sessionRegistryProvider).requireValue,
      commitActivation: ref.read(sessionRegistryProvider.notifier).activate,
      invalidateAccountState: ref.read(accountStateInvalidatorProvider),
      resetToHome: () async => context.go(RouteLocations.home),
      confirmLeave: ref.read(unsavedWorkGuardProvider).confirmLeave,
    );
  }

  @override
  void dispose() {
    _drawerButtonFocusNode.dispose();
    _drawerProfileSwitcherFocusNode.dispose();
    _profileSwitcherFocusNode.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final formFactor = FormFactorWidget.of(context);
    final registry = ref.watch(sessionRegistryProvider).value;
    final activeIdentity = ref.watch(activeAccountIdentityProvider).value;
    final activeLease = registry?.activeLease?.session;
    final activeAccount = activeLease?.account;
    final switcherState = registry == null
        ? null
        : AccountSwitcherState.fromRegistry(registry);
    final packageInfo =
        widget.buildVersionLabel == null && ref.exists(appDependenciesProvider)
        ? ref.watch(packageInfoProvider)
        : null;
    final buildVersionLabel =
        widget.buildVersionLabel ??
        (packageInfo == null
            ? null
            : AppLocalizations.of(context).navigationBuildVersion(
                packageInfo.version,
                packageInfo.buildNumber,
              ));
    final activeAvatarUrl =
        (activeIdentity?.lease == activeLease
            ? activeIdentity?.profile.avatar
            : null) ??
        (activeAccount == null
            ? null
            : registry?.sessions[activeAccount.did]?.cachedAvatarUrl);
    final activeProfile = activeIdentity?.lease == activeLease
        ? activeIdentity?.profile
        : null;
    final activeCustomisation =
        activeProfile?.customisation ??
        (activeAccount == null
            ? ProfileCustomisation.defaults
            : registry?.sessions[activeAccount.did]?.cachedCustomisation ??
                  ProfileCustomisation.defaults);
    final notificationBadge = NotificationBadge.fromCount(
      activeAccount == null
          ? ref.watch(notificationNewCountProvider).value ?? 0
          : ref
                    .watch(accountNotificationNewCountProvider(activeAccount))
                    .value ??
                0,
    );
    final selectedDestinationIndex = _selectedDestinationIndex(
      GoRouterState.of(context).matchedLocation,
      widget.navigationShell.currentIndex,
    );
    final router = GoRouter.of(context);

    if (formFactor.isLarge) {
      if (_AuthenticatedShellNavigationScope.ownsLargeRail(context)) {
        return widget.navigationShell;
      }
      return _LargeShellNavigationFrame(
        selectedIndex: selectedDestinationIndex,
        onDestinationSelected: _goDestination,
        linkLauncher: widget.linkLauncher,
        buildVersionLabel: widget.buildVersionLabel,
        child: widget.navigationShell,
      );
    }

    return Scaffold(
      key: _compactScaffoldKey,
      drawerEdgeDragWidth: _compactDrawerEdgeDragWidth,
      body: AppShellDrawerScope(
        openDrawer: () => _compactScaffoldKey.currentState?.openDrawer(),
        isDrawerOpen: _isDrawerOpen,
        menuButtonFocusNode: _drawerButtonFocusNode,
        child: widget.navigationShell,
      ),
      onDrawerChanged: (isOpen) {
        if (_isDrawerOpen != isOpen) {
          final shouldRestoreMenuFocus = _isDrawerOpen && !isOpen;
          setState(() => _isDrawerOpen = isOpen);
          if (shouldRestoreMenuFocus) {
            WidgetsBinding.instance.addPostFrameCallback((_) {
              if (mounted) _drawerButtonFocusNode.requestFocus();
            });
          }
        }
      },
      drawer: ListenableBuilder(
        listenable: router.routerDelegate,
        builder: (context, _) => _ShellDrawer(
          selectedIndex: _selectedDestinationIndex(
            router.state.matchedLocation,
            widget.navigationShell.currentIndex,
          ),
          onDestinationSelected: _goDestination,
          notificationBadge: notificationBadge,
          buildVersionLabel: buildVersionLabel,
          profileFocusNode: _drawerProfileSwitcherFocusNode,
          onOpenAccountSwitcher: switcherState == null
              ? null
              : () => _showCompactSwitcher(switcherState),
          onOpenTerms: () => _openExternalLink(
            Uri.parse('https://craftsky.social/terms'),
          ),
          onOpenPrivacy: () => _openExternalLink(
            Uri.parse('https://craftsky.social/privacy'),
          ),
        ),
      ),
      bottomNavigationBar: _ShellNavigationBar(
        selectedIndex: widget.navigationShell.currentIndex,
        onDestinationSelected: _goBranch,
        notificationBadge: notificationBadge,
        profileAvatarUrl: activeAvatarUrl,
        profileSeed: activeProfile?.displayName ?? activeProfile?.handle ?? '',
        profileCustomisation: activeCustomisation,
        profileFocusNode: _profileSwitcherFocusNode,
        onOpenAccountSwitcher: switcherState == null
            ? null
            : () => _showCompactSwitcher(switcherState),
      ),
    );
  }

  void _goBranch(int index) {
    widget.navigationShell.goBranch(
      index,
      initialLocation: index == widget.navigationShell.currentIndex,
    );
  }

  void _goDestination(int index) {
    if (index < _primaryDestinations.length) {
      _goBranch(index);
      return;
    }
    final location = switch (index - _primaryDestinations.length) {
      0 => RouteLocations.savedPosts,
      1 => RouteLocations.scheduledPosts,
      2 => RouteLocations.drafts,
      3 => RouteLocations.settings,
      _ => throw RangeError.index(index, _menuDestinations),
    };
    context.go(location);
  }

  Future<void> _openExternalLink(Uri uri) async {
    var opened = false;
    try {
      opened = await widget.linkLauncher(uri);
    } on Object {
      // Platform link handlers can fail by returning false or by throwing.
    }
    if (!mounted || opened) return;
    context.showError(AppLocalizations.of(context).navigationLinkOpenError);
  }

  Future<void> _showCompactSwitcher(AccountSwitcherState state) =>
      showAccountSwitcherSheet(
        context: context,
        fallbackState: state,
        onSelect: _activation.activate,
        onAddAccount: () {
          Navigator.pop(context);
          unawaited(context.push(RouteLocations.addAccount));
        },
      );
}

class _ShellDrawer extends StatefulWidget {
  const _ShellDrawer({
    required this.selectedIndex,
    required this.onDestinationSelected,
    required this.notificationBadge,
    required this.buildVersionLabel,
    required this.profileFocusNode,
    required this.onOpenAccountSwitcher,
    required this.onOpenTerms,
    required this.onOpenPrivacy,
  });

  final int selectedIndex;
  final ValueChanged<int> onDestinationSelected;
  final NotificationBadge notificationBadge;
  final String? buildVersionLabel;
  final FocusNode profileFocusNode;
  final VoidCallback? onOpenAccountSwitcher;
  final Future<void> Function() onOpenTerms;
  final Future<void> Function() onOpenPrivacy;

  @override
  State<_ShellDrawer> createState() => _ShellDrawerState();
}

class _ShellDrawerState extends State<_ShellDrawer> {
  var _actionClaimed = false;

  int get selectedIndex => widget.selectedIndex;
  ValueChanged<int> get onDestinationSelected => widget.onDestinationSelected;
  NotificationBadge get notificationBadge => widget.notificationBadge;
  String? get buildVersionLabel => widget.buildVersionLabel;
  FocusNode get profileFocusNode => widget.profileFocusNode;
  VoidCallback? get onOpenAccountSwitcher => widget.onOpenAccountSwitcher;
  Future<void> Function() get onOpenTerms => widget.onOpenTerms;
  Future<void> Function() get onOpenPrivacy => widget.onOpenPrivacy;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context);
    final swatches = theme.extension<BrandSwatchTheme>()!;
    return Drawer(
      backgroundColor: swatches.paper3,
      surfaceTintColor: Colors.transparent,
      clipBehavior: Clip.antiAlias,
      shape: _navigationDrawerShape(theme),
      child: SafeArea(
        child: Column(
          children: [
            Expanded(
              child: ListView(
                padding: const EdgeInsets.symmetric(vertical: 8),
                children: [
                  for (final (index, destination)
                      in _primaryDestinations.indexed)
                    ListTile(
                      selected: selectedIndex == index,
                      leading: index == 4
                          ? _DestinationIcon(
                              icon: selectedIndex == index
                                  ? destination.selectedIcon
                                  : destination.icon,
                              profileIconOnly: true,
                              onTapDestination: () =>
                                  _selectAndClose(context, index),
                              onOpenAccountSwitcher:
                                  onOpenAccountSwitcher == null
                                  ? null
                                  : () => _openAccountSwitcher(context),
                            )
                          : ExcludeSemantics(
                              child: _DestinationIcon(
                                icon: selectedIndex == index
                                    ? destination.selectedIcon
                                    : destination.icon,
                                badge: index == 3 ? notificationBadge : null,
                              ),
                            ),
                      title: Semantics(
                        label: _destinationSemanticsLabel(
                          context,
                          index: index,
                          notificationBadge: notificationBadge,
                        ),
                        excludeSemantics: true,
                        child: Text(_destinationLabel(l10n, index)),
                      ),
                      trailing: index == 4 && onOpenAccountSwitcher != null
                          ? _AccountSwitcherButton(
                              focusNode: profileFocusNode,
                              onPressed: () => _openAccountSwitcher(context),
                            )
                          : null,
                      onTap: () => _selectAndClose(context, index),
                    ),
                  for (final (offset, destination)
                      in _secondaryDestinations.indexed)
                    ListTile(
                      selected:
                          selectedIndex == _primaryDestinations.length + offset,
                      leading: Icon(
                        selectedIndex == _primaryDestinations.length + offset
                            ? destination.selectedIcon
                            : destination.icon,
                      ),
                      title: Text(
                        _destinationLabel(
                          l10n,
                          _primaryDestinations.length + offset,
                        ),
                      ),
                      onTap: () => _selectAndClose(
                        context,
                        _primaryDestinations.length + offset,
                      ),
                    ),
                ],
              ),
            ),
            ListTile(
              dense: true,
              minTileHeight: _utilityLinkHeight,
              title: Text(l10n.navigationTerms),
              onTap: () => _openExternalLink(context, onOpenTerms),
            ),
            ListTile(
              dense: true,
              minTileHeight: _utilityLinkHeight,
              title: Text(l10n.navigationPrivacy),
              onTap: () => _openExternalLink(context, onOpenPrivacy),
            ),
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 4, 16, 12),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  OutlinedButton.icon(
                    onPressed: () {},
                    icon: const Icon(Icons.chat_bubble_outline),
                    label: Text(l10n.navigationFeedback),
                  ),
                  if (buildVersionLabel case final label?) ...[
                    const SizedBox(height: 4),
                    _BuildVersionText(label),
                  ],
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  void _selectAndClose(BuildContext context, int index) {
    if (!_claimAction()) return;
    Navigator.pop(context);
    onDestinationSelected(index);
  }

  void _openAccountSwitcher(BuildContext context) {
    if (!_claimAction()) return;
    Navigator.pop(context);
    onOpenAccountSwitcher?.call();
  }

  void _openExternalLink(
    BuildContext context,
    Future<void> Function() open,
  ) {
    if (!_claimAction()) return;
    Navigator.pop(context);
    unawaited(open());
  }

  bool _claimAction() {
    if (_actionClaimed) return false;
    _actionClaimed = true;
    return true;
  }
}

class _ShellNavigationBar extends StatelessWidget {
  const _ShellNavigationBar({
    required this.selectedIndex,
    required this.onDestinationSelected,
    required this.notificationBadge,
    required this.profileAvatarUrl,
    required this.profileSeed,
    required this.profileCustomisation,
    required this.profileFocusNode,
    required this.onOpenAccountSwitcher,
  });

  final int selectedIndex;
  final ValueChanged<int> onDestinationSelected;
  final NotificationBadge notificationBadge;
  final String? profileAvatarUrl;
  final String profileSeed;
  final ProfileCustomisation profileCustomisation;
  final FocusNode profileFocusNode;
  final VoidCallback? onOpenAccountSwitcher;

  @override
  Widget build(BuildContext context) {
    final onSurface = Theme.of(context).colorScheme.onSurface;
    // Column sits above the safe-area inset as the Scaffold's
    // bottomNavigationBar, so the ink rule paints on top of the
    // NavigationBar's fill rather than being clipped behind it.
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        Container(height: 1.5, color: onSurface),
        NavigationBar(
          selectedIndex: selectedIndex,
          labelBehavior: NavigationDestinationLabelBehavior.alwaysHide,
          onDestinationSelected: onDestinationSelected,
          destinations: [
            for (final (index, d) in _primaryDestinations.indexed)
              NavigationDestination(
                icon: _DestinationIcon(
                  icon: d.icon,
                  badge: index == 3 ? notificationBadge : null,
                  profileAvatarUrl: index == 4 ? profileAvatarUrl : null,
                  profileSeed: index == 4 ? profileSeed : '',
                  profileCustomisation: index == 4
                      ? profileCustomisation
                      : ProfileCustomisation.defaults,
                  onTapDestination: index == 4
                      ? () => onDestinationSelected(index)
                      : null,
                  profileFocusNode: index == 4 ? profileFocusNode : null,
                  onOpenAccountSwitcher: index == 4
                      ? onOpenAccountSwitcher
                      : null,
                ),
                selectedIcon: _DestinationIcon(
                  icon: d.selectedIcon,
                  badge: index == 3 ? notificationBadge : null,
                  profileAvatarUrl: index == 4 ? profileAvatarUrl : null,
                  profileSeed: index == 4 ? profileSeed : '',
                  profileCustomisation: index == 4
                      ? profileCustomisation
                      : ProfileCustomisation.defaults,
                  profileSelected: index == 4,
                  onTapDestination: index == 4
                      ? () => onDestinationSelected(index)
                      : null,
                  profileFocusNode: index == 4 ? profileFocusNode : null,
                  onOpenAccountSwitcher: index == 4
                      ? onOpenAccountSwitcher
                      : null,
                ),
                label: _destinationSemanticsLabel(
                  context,
                  index: index,
                  notificationBadge: notificationBadge,
                ),
              ),
          ],
        ),
      ],
    );
  }
}

class _ShellNavigationRail extends StatelessWidget {
  const _ShellNavigationRail({
    required this.selectedIndex,
    required this.onDestinationSelected,
    required this.notificationBadge,
    required this.buildVersionLabel,
    required this.profileAnchorKey,
    required this.profileFocusNode,
    required this.onOpenAccountSwitcher,
    required this.onOpenTerms,
    required this.onOpenPrivacy,
  });

  final int selectedIndex;
  final ValueChanged<int> onDestinationSelected;
  final NotificationBadge notificationBadge;
  final String? buildVersionLabel;
  final GlobalKey profileAnchorKey;
  final FocusNode profileFocusNode;
  final VoidCallback? onOpenAccountSwitcher;
  final Future<void> Function() onOpenTerms;
  final Future<void> Function() onOpenPrivacy;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context);
    return Material(
      color: theme.scaffoldBackgroundColor,
      surfaceTintColor: Colors.transparent,
      clipBehavior: Clip.antiAlias,
      shape: _navigationRailBorder(theme),
      child: NavigationRail(
        backgroundColor: Colors.transparent,
        selectedIndex: selectedIndex,
        onDestinationSelected: onDestinationSelected,
        extended: true,
        scrollable: true,
        trailingAtBottom: true,
        trailing: Padding(
          padding: const EdgeInsets.fromLTRB(12, 8, 12, 12),
          child: SizedBox(
            width: 200,
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                TextButton(
                  style: _utilityLinkButtonStyle(),
                  onPressed: () => unawaited(onOpenTerms()),
                  child: Text(l10n.navigationTerms),
                ),
                TextButton(
                  style: _utilityLinkButtonStyle(),
                  onPressed: () => unawaited(onOpenPrivacy()),
                  child: Text(l10n.navigationPrivacy),
                ),
                OutlinedButton.icon(
                  onPressed: () {},
                  icon: const Icon(Icons.chat_bubble_outline),
                  label: Text(l10n.navigationFeedback),
                ),
                if (buildVersionLabel case final label?) ...[
                  const SizedBox(height: 4),
                  _BuildVersionText(label),
                ],
              ],
            ),
          ),
        ),
        destinations: [
          for (final (index, d) in _menuDestinations.indexed)
            NavigationRailDestination(
              icon: _DestinationIcon(
                icon: d.icon,
                badge: index == 3 ? notificationBadge : null,
                profileIconOnly: index == 4,
                onTapDestination: index == 4
                    ? () => onDestinationSelected(index)
                    : null,
                onOpenAccountSwitcher: index == 4
                    ? onOpenAccountSwitcher
                    : null,
              ),
              selectedIcon: _DestinationIcon(
                icon: d.selectedIcon,
                badge: index == 3 ? notificationBadge : null,
                profileIconOnly: index == 4,
                onTapDestination: index == 4
                    ? () => onDestinationSelected(index)
                    : null,
                onOpenAccountSwitcher: index == 4
                    ? onOpenAccountSwitcher
                    : null,
              ),
              label: index == 4
                  ? _ProfileRailLabel(
                      anchorKey: profileAnchorKey,
                      label: _destinationSemanticsLabel(
                        context,
                        index: index,
                        notificationBadge: notificationBadge,
                      ),
                      focusNode: profileFocusNode,
                      onOpenAccountSwitcher: onOpenAccountSwitcher,
                    )
                  : Semantics(
                      label: _destinationSemanticsLabel(
                        context,
                        index: index,
                        notificationBadge: notificationBadge,
                      ),
                      excludeSemantics: true,
                      child: Text(_destinationLabel(l10n, index)),
                    ),
            ),
        ],
      ),
    );
  }
}

class _BuildVersionText extends StatelessWidget {
  const _BuildVersionText(this.label);

  final String label;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Text(
      label,
      textAlign: TextAlign.center,
      style: theme.textTheme.labelSmall?.copyWith(
        color: theme.colorScheme.onSurfaceVariant,
      ),
    );
  }
}

class _ProfileRailLabel extends StatelessWidget {
  const _ProfileRailLabel({
    required this.anchorKey,
    required this.label,
    required this.focusNode,
    required this.onOpenAccountSwitcher,
  });

  final GlobalKey anchorKey;
  final String label;
  final FocusNode focusNode;
  final VoidCallback? onOpenAccountSwitcher;

  @override
  Widget build(BuildContext context) {
    final open = onOpenAccountSwitcher;
    return SizedBox(
      width: _profileRailLabelWidth,
      child: Row(
        children: [
          Expanded(
            child: Semantics(
              label: label,
              excludeSemantics: true,
              child: Text(label),
            ),
          ),
          if (open != null)
            _AccountSwitcherButton(
              key: anchorKey,
              focusNode: focusNode,
              onPressed: open,
            ),
        ],
      ),
    );
  }
}

class _AccountSwitcherButton extends StatelessWidget {
  const _AccountSwitcherButton({
    required this.focusNode,
    required this.onPressed,
    super.key,
  });

  final FocusNode focusNode;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) => IconButton(
    focusNode: focusNode,
    tooltip: AppLocalizations.of(context).accountSwitcherTooltip,
    onPressed: onPressed,
    icon: const Icon(Icons.switch_account),
  );
}

ButtonStyle _utilityLinkButtonStyle() => TextButton.styleFrom(
  minimumSize: const Size(64, _utilityLinkHeight),
  tapTargetSize: MaterialTapTargetSize.shrinkWrap,
);

String _destinationLabel(AppLocalizations l10n, int index) => switch (index) {
  0 => l10n.feedTitle,
  1 => l10n.projectsTitle,
  2 => l10n.searchTitle,
  3 => l10n.notificationsTitle,
  4 => l10n.navigationProfile,
  5 => l10n.navigationSaved,
  6 => l10n.navigationScheduled,
  7 => l10n.draftsTitle,
  8 => l10n.profileSettingsAction,
  _ => throw RangeError.index(index, _menuDestinations),
};

int _selectedDestinationIndex(String matchedLocation, int branchIndex) {
  if (matchedLocation.startsWith(RouteLocations.savedPosts)) return 5;
  if (matchedLocation.startsWith(RouteLocations.scheduledPosts)) return 6;
  if (matchedLocation.startsWith(RouteLocations.drafts)) return 7;
  if (matchedLocation.startsWith(RouteLocations.settings)) return 8;
  return branchIndex;
}

int _destinationIndexForLocation(String matchedLocation) {
  final personalDestination = _selectedDestinationIndex(matchedLocation, 4);
  if (personalDestination != 4) return personalDestination;
  if (matchedLocation.startsWith(RouteLocations.projects)) return 1;
  if (matchedLocation.startsWith(RouteLocations.search)) return 2;
  if (matchedLocation.startsWith(RouteLocations.notifications)) return 3;
  if (matchedLocation.startsWith(RouteLocations.profile)) return 4;
  return 0;
}

String _destinationSemanticsLabel(
  BuildContext context, {
  required int index,
  required NotificationBadge notificationBadge,
}) {
  final l10n = AppLocalizations.of(context);
  final label = _destinationLabel(l10n, index);
  if (index != 3 || !notificationBadge.visible) return label;
  final countLabel = l10n.notificationNewActivityCount(notificationBadge.count);
  return '$label, $countLabel';
}

class _DestinationIcon extends StatelessWidget {
  const _DestinationIcon({
    required this.icon,
    this.badge,
    this.profileAvatarUrl,
    this.profileSeed = '',
    this.profileCustomisation = ProfileCustomisation.defaults,
    this.profileSelected = false,
    this.profileIconOnly = false,
    this.onTapDestination,
    this.profileFocusNode,
    this.onOpenAccountSwitcher,
  });

  final IconData icon;
  final NotificationBadge? badge;
  final String? profileAvatarUrl;
  final String profileSeed;
  final ProfileCustomisation profileCustomisation;
  final bool profileSelected;
  final bool profileIconOnly;
  final VoidCallback? onTapDestination;
  final FocusNode? profileFocusNode;
  final VoidCallback? onOpenAccountSwitcher;

  @override
  Widget build(BuildContext context) {
    final value = badge;
    Widget child = onOpenAccountSwitcher == null || profileIconOnly
        ? Icon(icon)
        : AccountAvatar(
            avatarUrl: profileAvatarUrl,
            seed: profileSeed,
            customisation: profileCustomisation,
            selected: profileSelected,
          );
    if (value != null && value.visible) {
      child = Badge(label: Text(value.label), child: child);
    }
    final open = onOpenAccountSwitcher;
    if (open == null) return child;
    final detector = FocusableActionDetector(
      focusNode: profileFocusNode,
      shortcuts: {
        LogicalKeySet(LogicalKeyboardKey.alt, LogicalKeyboardKey.arrowDown):
            const ActivateIntent(),
      },
      actions: {
        ActivateIntent: CallbackAction<ActivateIntent>(
          onInvoke: (_) {
            open();
            return null;
          },
        ),
      },
      child: Semantics(
        onLongPress: open,
        hint: AppLocalizations.of(context).accountSwitcherLongPressHint,
        child: InkWell(
          customBorder: const CircleBorder(),
          onTap: () {
            profileFocusNode?.requestFocus();
            onTapDestination?.call();
          },
          onLongPress: open,
          child: child,
        ),
      ),
    );
    if (profileIconOnly) return detector;
    return Tooltip(
      message: AppLocalizations.of(context).accountSwitcherTooltip,
      child: detector,
    );
  }
}
