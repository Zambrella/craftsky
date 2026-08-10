import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:flutter/material.dart';

/// Compact circular [ChunkyButton] for icon-only actions.
class ChunkyIconButton extends StatelessWidget {
  const ChunkyIconButton({
    required this.onPressed,
    required this.icon,
    this.tooltip,
    super.key,
  });

  final VoidCallback? onPressed;
  final IconData icon;
  final String? tooltip;

  static const double _size = 44;

  @override
  Widget build(BuildContext context) {
    final button = SizedBox(
      width: _size,
      height: _size,
      child: ChunkyButton(
        onPressed: onPressed,
        variant: ChunkyButtonVariant.secondary,
        style: const ButtonStyle(
          padding: WidgetStatePropertyAll(EdgeInsets.zero),
          minimumSize: WidgetStatePropertyAll(Size(_size, _size)),
          fixedSize: WidgetStatePropertyAll(Size(_size, _size)),
        ),
        child: Icon(icon),
      ),
    );
    final label = tooltip;
    if (label == null) return button;
    return Tooltip(message: label, child: button);
  }
}
