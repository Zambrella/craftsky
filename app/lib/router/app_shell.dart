import 'dart:async';

import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/account_switcher_state.dart';
import 'package:craftsky_app/auth/providers/account_activation_coordinator.dart';
import 'package:craftsky_app/auth/providers/account_boundary_provider.dart';
import 'package:craftsky_app/auth/providers/active_account_identity_provider.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/auth/providers/unsaved_work_guard_provider.dart';
import 'package:craftsky_app/auth/widgets/account_avatar.dart';
import 'package:craftsky_app/auth/widgets/account_switcher_content.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/notifications/models/notification_badge.dart';
import 'package:craftsky_app/notifications/providers/notification_new_count_provider.dart';
import 'package:craftsky_app/router/app_shell_drawer.dart';
import 'package:craftsky_app/router/route_locations.dart';
import 'package:craftsky_app/shared/link/external_link.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/theme/craftsky_divider.dart';
import 'package:craftsky_app/theme/form_factor.dart';
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

class AppShell extends ConsumerStatefulWidget {
  const AppShell({
    required this.navigationShell,
    this.linkLauncher = launchExternalLink,
    super.key,
  });

  final StatefulNavigationShell navigationShell;
  final ExternalLinkLauncher linkLauncher;

  @override
  ConsumerState<AppShell> createState() => _AppShellState();
}

class _AppShellState extends ConsumerState<AppShell> {
  late final AccountActivationCoordinator _activation;
  final GlobalKey<ScaffoldState> _compactScaffoldKey = GlobalKey();
  final GlobalKey _profileAnchorKey = GlobalKey();
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
    final activeAvatarUrl =
        (activeIdentity?.lease == activeLease
            ? activeIdentity?.profile.avatar
            : null) ??
        (activeAccount == null
            ? null
            : registry?.sessions[activeAccount.did]?.cachedAvatarUrl);
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

    if (formFactor.isLarge) {
      final textDirection = Directionality.of(context);
      return Scaffold(
        body: Row(
          // Put the nested branch navigator first in semantics order so its
          // route boundary does not suppress the rail. Reverse only the Row's
          // layout direction to keep the rail on the leading edge.
          textDirection: switch (textDirection) {
            TextDirection.ltr => TextDirection.rtl,
            TextDirection.rtl => TextDirection.ltr,
          },
          children: [
            Expanded(
              child: Directionality(
                textDirection: textDirection,
                child: widget.navigationShell,
              ),
            ),
            const CraftskyDivider(axis: Axis.vertical),
            Directionality(
              textDirection: textDirection,
              child: _ShellNavigationRail(
                selectedIndex: selectedDestinationIndex,
                onDestinationSelected: _goDestination,
                notificationBadge: notificationBadge,
                profileAvatarUrl: activeAvatarUrl,
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

    return Scaffold(
      key: _compactScaffoldKey,
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
      drawer: _ShellDrawer(
        selectedIndex: selectedDestinationIndex,
        onDestinationSelected: _goDestination,
        notificationBadge: notificationBadge,
        profileAvatarUrl: activeAvatarUrl,
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
      bottomNavigationBar: _ShellNavigationBar(
        selectedIndex: widget.navigationShell.currentIndex,
        onDestinationSelected: _goBranch,
        notificationBadge: notificationBadge,
        profileAvatarUrl: activeAvatarUrl,
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
    widget.navigationShell.goBranch(4, initialLocation: true);
    unawaited(context.push(location));
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
      showModalBottomSheet<void>(
        context: context,
        showDragHandle: true,
        builder: (sheetContext) => _LiveAccountSwitcherContent(
          fallbackState: state,
          onSelect: _activation.activate,
          onAddAccount: () {
            Navigator.pop(sheetContext);
            unawaited(context.push(RouteLocations.addAccount));
          },
        ),
      );

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
      overlay.size.width - origin.dx,
      overlay.size.height - origin.dy - box.size.height,
    );
    await showMenu<void>(
      context: context,
      position: position,
      items: [
        PopupMenuItem<void>(
          enabled: false,
          padding: EdgeInsets.zero,
          child: SizedBox(
            width: 320,
            child: _LiveAccountSwitcherContent(
              fallbackState: state,
              onSelect: _activation.activate,
              onAddAccount: () {
                Navigator.pop(context);
                unawaited(context.push(RouteLocations.addAccount));
              },
            ),
          ),
        ),
      ],
    );
  }
}

class _ShellDrawer extends StatefulWidget {
  const _ShellDrawer({
    required this.selectedIndex,
    required this.onDestinationSelected,
    required this.notificationBadge,
    required this.profileAvatarUrl,
    required this.profileFocusNode,
    required this.onOpenAccountSwitcher,
    required this.onOpenTerms,
    required this.onOpenPrivacy,
  });

  final int selectedIndex;
  final ValueChanged<int> onDestinationSelected;
  final NotificationBadge notificationBadge;
  final String? profileAvatarUrl;
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
  String? get profileAvatarUrl => widget.profileAvatarUrl;
  FocusNode get profileFocusNode => widget.profileFocusNode;
  VoidCallback? get onOpenAccountSwitcher => widget.onOpenAccountSwitcher;
  Future<void> Function() get onOpenTerms => widget.onOpenTerms;
  Future<void> Function() get onOpenPrivacy => widget.onOpenPrivacy;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Drawer(
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
                              icon: destination.icon,
                              profileAvatarUrl: profileAvatarUrl,
                              profileSelected: selectedIndex == index,
                              onTapDestination: () =>
                                  _selectAndClose(context, index),
                              profileFocusNode: profileFocusNode,
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
                      onTap: () => _selectAndClose(context, index),
                    ),
                  const CraftskyDivider(indent: 16, endIndent: 16),
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
            const CraftskyDivider(indent: 16, endIndent: 16),
            ListTile(
              dense: true,
              title: Text(l10n.navigationTerms),
              onTap: () => _openExternalLink(context, onOpenTerms),
            ),
            ListTile(
              dense: true,
              title: Text(l10n.navigationPrivacy),
              onTap: () => _openExternalLink(context, onOpenPrivacy),
            ),
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 4, 16, 12),
              child: SizedBox(
                width: double.infinity,
                child: OutlinedButton.icon(
                  onPressed: () {},
                  icon: const Icon(Icons.chat_bubble_outline),
                  label: Text(l10n.navigationFeedback),
                ),
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

class _LiveAccountSwitcherContent extends ConsumerStatefulWidget {
  const _LiveAccountSwitcherContent({
    required this.fallbackState,
    required this.onSelect,
    required this.onAddAccount,
  });

  final AccountSwitcherState fallbackState;
  final Future<AccountActivationResult> Function(AccountSessionLease) onSelect;
  final VoidCallback onAddAccount;

  @override
  ConsumerState<_LiveAccountSwitcherContent> createState() =>
      _LiveAccountSwitcherContentState();
}

class _LiveAccountSwitcherContentState
    extends ConsumerState<_LiveAccountSwitcherContent> {
  AccountSessionLease? _activating;

  @override
  Widget build(BuildContext context) {
    final registry = ref.watch(sessionRegistryProvider).value;
    final state = registry == null
        ? widget.fallbackState
        : AccountSwitcherState.fromRegistry(registry);
    return AccountSwitcherContent(
      state: state,
      activating: _activating,
      onSelect: (lease) => unawaited(_activate(lease)),
      onAddAccount: widget.onAddAccount,
    );
  }

  Future<void> _activate(AccountSessionLease lease) async {
    if (_activating != null) return;
    setState(() => _activating = lease);
    try {
      final result = await widget.onSelect(lease);
      if (!mounted) return;
      if (result == AccountActivationResult.activated ||
          result == AccountActivationResult.alreadyActive) {
        await Navigator.maybePop(context);
      }
    } finally {
      if (mounted) setState(() => _activating = null);
    }
  }
}

class _ShellNavigationBar extends StatelessWidget {
  const _ShellNavigationBar({
    required this.selectedIndex,
    required this.onDestinationSelected,
    required this.notificationBadge,
    required this.profileAvatarUrl,
    required this.profileFocusNode,
    required this.onOpenAccountSwitcher,
  });

  final int selectedIndex;
  final ValueChanged<int> onDestinationSelected;
  final NotificationBadge notificationBadge;
  final String? profileAvatarUrl;
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
    required this.profileAvatarUrl,
    required this.profileAnchorKey,
    required this.profileFocusNode,
    required this.onOpenAccountSwitcher,
    required this.onOpenTerms,
    required this.onOpenPrivacy,
  });

  final int selectedIndex;
  final ValueChanged<int> onDestinationSelected;
  final NotificationBadge notificationBadge;
  final String? profileAvatarUrl;
  final GlobalKey profileAnchorKey;
  final FocusNode profileFocusNode;
  final VoidCallback? onOpenAccountSwitcher;
  final Future<void> Function() onOpenTerms;
  final Future<void> Function() onOpenPrivacy;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return NavigationRail(
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
              const CraftskyDivider(),
              TextButton(
                onPressed: () => unawaited(onOpenTerms()),
                child: Text(l10n.navigationTerms),
              ),
              TextButton(
                onPressed: () => unawaited(onOpenPrivacy()),
                child: Text(l10n.navigationPrivacy),
              ),
              OutlinedButton.icon(
                onPressed: () {},
                icon: const Icon(Icons.chat_bubble_outline),
                label: Text(l10n.navigationFeedback),
              ),
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
              profileAvatarUrl: index == 4 ? profileAvatarUrl : null,
              onTapDestination: index == 4
                  ? () => onDestinationSelected(index)
                  : null,
              profileFocusNode: index == 4 ? profileFocusNode : null,
              onOpenAccountSwitcher: index == 4 ? onOpenAccountSwitcher : null,
            ),
            selectedIcon: _DestinationIcon(
              icon: d.selectedIcon,
              badge: index == 3 ? notificationBadge : null,
              profileAvatarUrl: index == 4 ? profileAvatarUrl : null,
              profileSelected: index == 4,
              onTapDestination: index == 4
                  ? () => onDestinationSelected(index)
                  : null,
              profileFocusNode: index == 4 ? profileFocusNode : null,
              onOpenAccountSwitcher: index == 4 ? onOpenAccountSwitcher : null,
            ),
            label: Semantics(
              key: index == 4 ? profileAnchorKey : null,
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
    );
  }
}

String _destinationLabel(AppLocalizations l10n, int index) => switch (index) {
  0 => l10n.feedTitle,
  1 => l10n.projectsTitle,
  2 => l10n.searchTitle,
  3 => l10n.notificationsTitle,
  4 => l10n.navigationProfile,
  5 => l10n.savedPostsTitle,
  6 => l10n.scheduledPostsTitle,
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
    this.profileSelected = false,
    this.onTapDestination,
    this.profileFocusNode,
    this.onOpenAccountSwitcher,
  });

  final IconData icon;
  final NotificationBadge? badge;
  final String? profileAvatarUrl;
  final bool profileSelected;
  final VoidCallback? onTapDestination;
  final FocusNode? profileFocusNode;
  final VoidCallback? onOpenAccountSwitcher;

  @override
  Widget build(BuildContext context) {
    final value = badge;
    Widget child = onOpenAccountSwitcher == null
        ? Icon(icon)
        : AccountAvatar(
            avatarUrl: profileAvatarUrl,
            selected: profileSelected,
          );
    if (value != null && value.visible) {
      child = Badge(label: Text(value.label), child: child);
    }
    final open = onOpenAccountSwitcher;
    if (open == null) return child;
    return Tooltip(
      message: AppLocalizations.of(context).accountSwitcherTooltip,
      child: FocusableActionDetector(
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
      ),
    );
  }
}
