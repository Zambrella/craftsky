import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

/// A floating action button with CraftSky's hard-offset paper-cutout shadow.
class CraftskyFloatingActionButton extends StatelessWidget {
  const CraftskyFloatingActionButton.extended({
    required this.onPressed,
    required this.icon,
    required this.label,
    this.tooltip,
    this.minimumHeight = 56,
    super.key,
  });

  final VoidCallback? onPressed;
  final Widget icon;
  final Widget label;
  final String? tooltip;
  final double minimumHeight;

  @override
  Widget build(BuildContext context) {
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    final button = ChunkyButton(
      onPressed: onPressed,
      style: ButtonStyle(
        minimumSize: WidgetStatePropertyAll(Size(0, minimumHeight)),
        padding: WidgetStatePropertyAll(
          EdgeInsets.symmetric(horizontal: spacing.sp5),
        ),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          icon,
          SizedBox(width: spacing.sp2),
          Flexible(child: label),
        ],
      ),
    );
    return tooltip == null ? button : Tooltip(message: tooltip, child: button);
  }
}
