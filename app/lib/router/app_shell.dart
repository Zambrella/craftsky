import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/notifications/models/notification_badge.dart';
import 'package:craftsky_app/notifications/providers/notification_new_count_provider.dart';
import 'package:craftsky_app/theme/craftsky_divider.dart';
import 'package:craftsky_app/theme/form_factor.dart';
import 'package:flutter/material.dart';
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

class AppShell extends ConsumerWidget {
  const AppShell({required this.navigationShell, super.key});

  final StatefulNavigationShell navigationShell;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final formFactor = FormFactorWidget.of(context);
    final notificationBadge = NotificationBadge.fromCount(
      ref.watch(notificationNewCountProvider).value ?? 0,
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
                child: navigationShell,
              ),
            ),
            const CraftskyDivider(axis: Axis.vertical),
            Directionality(
              textDirection: textDirection,
              child: _ShellNavigationRail(
                selectedIndex: navigationShell.currentIndex,
                onDestinationSelected: _goBranch,
                notificationBadge: notificationBadge,
              ),
            ),
          ],
        ),
      );
    }

    return Scaffold(
      body: navigationShell,
      bottomNavigationBar: _ShellNavigationBar(
        selectedIndex: navigationShell.currentIndex,
        onDestinationSelected: _goBranch,
        notificationBadge: notificationBadge,
      ),
    );
  }

  void _goBranch(int index) {
    navigationShell.goBranch(
      index,
      initialLocation: index == navigationShell.currentIndex,
    );
  }
}

class _ShellNavigationBar extends StatelessWidget {
  const _ShellNavigationBar({
    required this.selectedIndex,
    required this.onDestinationSelected,
    required this.notificationBadge,
  });

  final int selectedIndex;
  final ValueChanged<int> onDestinationSelected;
  final NotificationBadge notificationBadge;

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
                ),
                selectedIcon: _DestinationIcon(
                  icon: d.selectedIcon,
                  badge: index == 3 ? notificationBadge : null,
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
  });

  final int selectedIndex;
  final ValueChanged<int> onDestinationSelected;
  final NotificationBadge notificationBadge;

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
            ),
            selectedIcon: _DestinationIcon(
              icon: d.selectedIcon,
              badge: index == 3 ? notificationBadge : null,
            ),
            label: Semantics(
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
  const _DestinationIcon({required this.icon, this.badge});

  final IconData icon;
  final NotificationBadge? badge;

  @override
  Widget build(BuildContext context) {
    final value = badge;
    final child = Icon(icon);
    if (value == null || !value.visible) return child;
    return Badge(label: Text(value.label), child: child);
  }
}
