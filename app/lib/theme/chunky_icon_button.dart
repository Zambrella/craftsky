import 'package:craftsky_app/shared/widgets/action_progress.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:material_ui/material_ui.dart';

/// Compact circular [ChunkyButton] for icon-only actions.
class ChunkyIconButton extends StatelessWidget {
  const new({
    required this.onPressed,
    required this.icon,
    this.tooltip,
    this.isLoading = false,
    super.key,
  });

  final VoidCallback? onPressed;
  final IconData icon;
  final String? tooltip;
  final bool isLoading;

  static const double _size = 44;

  @override
  Widget build(BuildContext context) {
    final button = SizedBox(
      width: _size,
      height: _size,
      child: ChunkyButton(
        onPressed: isLoading ? null : onPressed,
        variant: ChunkyButtonVariant.secondary,
        style: const ButtonStyle(
          padding: WidgetStatePropertyAll(EdgeInsets.zero),
          minimumSize: WidgetStatePropertyAll(Size(_size, _size)),
          fixedSize: WidgetStatePropertyAll(Size(_size, _size)),
        ),
        child: ActionProgress(isLoading: isLoading, child: Icon(icon)),
      ),
    );
    final label = tooltip;
    if (label == null) return button;
    return Tooltip(message: label, child: button);
  }
}
