import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/account_switcher_state.dart';
import 'package:craftsky_app/auth/providers/account_activation_coordinator.dart';
import 'package:craftsky_app/auth/providers/account_boundary_provider.dart';
import 'package:craftsky_app/auth/providers/account_transition_provider.dart';
import 'package:craftsky_app/auth/providers/active_account_identity_provider.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/auth/providers/unsaved_work_guard_provider.dart';
import 'package:craftsky_app/auth/widgets/account_avatar.dart';
import 'package:craftsky_app/auth/widgets/account_switcher_content.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/notifications/models/notification_badge.dart';
import 'package:craftsky_app/notifications/providers/notification_new_count_provider.dart';
import 'package:craftsky_app/router/route_locations.dart';
import 'package:craftsky_app/theme/craftsky_divider.dart';
import 'package:craftsky_app/theme/form_factor.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

/// Paired icon + label spec for a shell branch destination.
class _DestinationSpec {
  const _DestinationSpec({
    required this.icon,
    required this.selectedIcon,
    required this.label,
  });

  final IconData icon;
  final IconData selectedIcon;
  final String label;
}

const _destinations = <_DestinationSpec>[
  _DestinationSpec(
    icon: Icons.home_outlined,
    selectedIcon: Icons.home,
    label: 'Feed',
  ),
  _DestinationSpec(
    icon: Icons.grid_view_outlined,
    selectedIcon: Icons.grid_view,
    label: 'Projects',
  ),
  _DestinationSpec(
    icon: Icons.search_outlined,
    selectedIcon: Icons.search,
    label: 'Search',
  ),
  _DestinationSpec(
    icon: Icons.notifications_outlined,
    selectedIcon: Icons.notifications,
    label: 'Notifications',
  ),
  _DestinationSpec(
    icon: Icons.person_outline,
    selectedIcon: Icons.person,
    label: 'Profile',
  ),
];

class AppShell extends ConsumerStatefulWidget {
  const AppShell({required this.navigationShell, super.key});

  final StatefulNavigationShell navigationShell;

  @override
  ConsumerState<AppShell> createState() => _AppShellState();
}

class _AppShellState extends ConsumerState<AppShell> {
  late final AccountActivationCoordinator _activation;
  final GlobalKey _profileAnchorKey = GlobalKey();
  final FocusNode _profileSwitcherFocusNode = FocusNode(
    debugLabel: 'Profile account switcher',
  );

  @override
  void initState() {
    super.initState();
    _activation = AccountActivationCoordinator(
      readRegistry: () => ref.read(sessionRegistryProvider).requireValue,
      commitActivation: ref.read(sessionRegistryProvider.notifier).activate,
      publishTransition: (transition) =>
          ref.read(accountTransitionStateProvider.notifier).transition =
              transition,
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
                selectedIndex: widget.navigationShell.currentIndex,
                onDestinationSelected: _goBranch,
                notificationBadge: notificationBadge,
                profileAvatarUrl: activeAvatarUrl,
                profileAnchorKey: _profileAnchorKey,
                profileFocusNode: _profileSwitcherFocusNode,
                onOpenAccountSwitcher: switcherState == null
                    ? null
                    : () => _showLargeSwitcher(switcherState),
              ),
            ),
          ],
        ),
      );
    }

    return Scaffold(
      body: widget.navigationShell,
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

  Future<void> _showCompactSwitcher(AccountSwitcherState state) =>
      showModalBottomSheet<void>(
        context: context,
        showDragHandle: true,
        builder: (sheetContext) => _LiveAccountSwitcherContent(
          fallbackState: state,
          onSelect: (lease) {
            Navigator.pop(sheetContext);
            unawaited(
              _activation.activate(
                lease,
                source: AccountActivationSource.manual,
              ),
            );
          },
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
              onSelect: (lease) {
                Navigator.pop(context);
                unawaited(
                  _activation.activate(
                    lease,
                    source: AccountActivationSource.manual,
                  ),
                );
              },
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

class _LiveAccountSwitcherContent extends ConsumerWidget {
  const _LiveAccountSwitcherContent({
    required this.fallbackState,
    required this.onSelect,
    required this.onAddAccount,
  });

  final AccountSwitcherState fallbackState;
  final ValueChanged<AccountSessionLease> onSelect;
  final VoidCallback onAddAccount;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final registry = ref.watch(sessionRegistryProvider).value;
    final state = registry == null
        ? fallbackState
        : AccountSwitcherState.fromRegistry(
            registry,
            notificationCounts: {
              for (final did in registry.sessions.keys)
                AccountKey(did.value):
                    ref
                        .watch(
                          accountNotificationNewCountProvider(
                            AccountKey(did.value),
                          ),
                        )
                        .value ??
                    0,
            },
          );
    return AccountSwitcherContent(
      state: state,
      onSelect: onSelect,
      onAddAccount: onAddAccount,
    );
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
            for (final (index, d) in _destinations.indexed)
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
                  destination: d,
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
  });

  final int selectedIndex;
  final ValueChanged<int> onDestinationSelected;
  final NotificationBadge notificationBadge;
  final String? profileAvatarUrl;
  final GlobalKey profileAnchorKey;
  final FocusNode profileFocusNode;
  final VoidCallback? onOpenAccountSwitcher;

  @override
  Widget build(BuildContext context) {
    return NavigationRail(
      selectedIndex: selectedIndex,
      onDestinationSelected: onDestinationSelected,
      labelType: NavigationRailLabelType.all,
      destinations: [
        for (final (index, d) in _destinations.indexed)
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
                destination: d,
                notificationBadge: notificationBadge,
              ),
              excludeSemantics: true,
              child: Text(d.label),
            ),
          ),
      ],
    );
  }
}

String _destinationSemanticsLabel(
  BuildContext context, {
  required int index,
  required _DestinationSpec destination,
  required NotificationBadge notificationBadge,
}) {
  if (index != 3 || !notificationBadge.visible) return destination.label;
  final countLabel = AppLocalizations.of(
    context,
  ).notificationNewActivityCount(notificationBadge.count);
  return '${destination.label}, $countLabel';
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
