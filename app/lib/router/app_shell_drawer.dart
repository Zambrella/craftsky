import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:flutter/material.dart';

/// Gives top-level branch app bars access to the compact shell's drawer.
class AppShellDrawerScope extends InheritedWidget {
  const AppShellDrawerScope({
    required this.openDrawer,
    required this.isDrawerOpen,
    required super.child,
    this.menuButtonFocusNode,
    super.key,
  });

  final VoidCallback openDrawer;
  final bool isDrawerOpen;
  final FocusNode? menuButtonFocusNode;

  static AppShellDrawerScope? maybeOf(BuildContext context) =>
      context.dependOnInheritedWidgetOfExactType<AppShellDrawerScope>();

  @override
  bool updateShouldNotify(AppShellDrawerScope oldWidget) =>
      openDrawer != oldWidget.openDrawer ||
      isDrawerOpen != oldWidget.isDrawerOpen;
}

/// Leading action used by each compact top-level branch app bar.
class AppShellDrawerButton extends StatelessWidget {
  const AppShellDrawerButton({super.key});

  @override
  Widget build(BuildContext context) {
    final scope = AppShellDrawerScope.maybeOf(context);
    if (scope == null) return const SizedBox.shrink();
    final tooltip = AppLocalizations.of(context).navigationMenuTooltip;
    return Semantics(
      button: true,
      label: tooltip,
      expanded: scope.isDrawerOpen,
      onTap: scope.openDrawer,
      excludeSemantics: true,
      child: IconButton(
        tooltip: tooltip,
        focusNode: scope.menuButtonFocusNode,
        onPressed: scope.openDrawer,
        icon: const Icon(Icons.menu),
      ),
    );
  }
}
